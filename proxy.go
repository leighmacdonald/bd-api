package main

import (
	"fmt"
	goproxy "golang.org/x/net/proxy"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/armon/go-socks5"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/proxy"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/context"
)

type proxyManager struct {
	proxies     map[string]*proxyConfig
	log         *zap.Logger
	curProxyIdx uint32
	proxyURLs   []*url.URL
}

func newProxyManager(logger *zap.Logger) *proxyManager {
	return &proxyManager{proxies: map[string]*proxyConfig{}, log: logger.Named("proxy")}
}

func (p *proxyManager) start(config *appConfig) {
	waitGroup := &sync.WaitGroup{}
	for _, serverCfg := range config.Proxies {
		waitGroup.Add(1)

		go func(cfg *proxyConfig) {
			sshConf := ssh.ClientConfig{User: cfg.Username, Auth: []ssh.AuthMethod{ //nolint:exhaustruct
				ssh.PublicKeys(cfg.signer),
			}, HostKeyCallback: ssh.InsecureIgnoreHostKey()} //nolint:gosec

			conn, errConn := ssh.Dial("tcp", cfg.RemoteAddr, &sshConf)
			if errConn != nil {
				p.log.Error("Failed to connect to host", zap.Error(errConn))
				waitGroup.Done()

				return
			}

			p.log.Info("Connect to ssh host", zap.String("addr", cfg.RemoteAddr))

			socksConf := &socks5.Config{ //nolint:exhaustruct
				Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
					dialedConn, errDial := conn.Dial(network, addr)
					if errDial != nil {
						return nil, errors.Wrap(errDial, "Failed to dial network")
					}

					return dialedConn, nil
				},
			}

			server, errServer := socks5.New(socksConf)
			if errServer != nil {
				p.log.Error("Failed to initialize socks5", zap.Error(errServer))
				waitGroup.Done()

				return
			}

			p.log.Info("Starting socks5 service", zap.String("addr", cfg.LocalAddr))

			cfg.conn = conn
			cfg.socks = server
			p.proxies[cfg.RemoteAddr] = cfg

			waitGroup.Done()

			if errListen := server.ListenAndServe("tcp", cfg.LocalAddr); errListen != nil {
				p.log.Error("Socks5 listener returned error", zap.Error(errListen))
			}
		}(serverCfg)
	}

	waitGroup.Wait()
}

func (p *proxyManager) stop() {
	waitGroup := &sync.WaitGroup{}
	for _, curProxy := range p.proxies {
		waitGroup.Add(1)

		go func(proxyConf *proxyConfig) {
			defer waitGroup.Done()

			if errClose := proxyConf.conn.Close(); errClose != nil {
				p.log.Error("Error closing connection", zap.Error(errClose), zap.String("addr", proxyConf.RemoteAddr))

				return
			}

			p.log.Info("Closed connection", zap.String("addr", proxyConf.RemoteAddr))
		}(curProxy)
	}

	waitGroup.Wait()
}

// next creates a new http client with the next proxy transport set
func (p *proxyManager) next() *http.Client {
	u := p.proxyURLs[p.curProxyIdx%uint32(len(p.proxyURLs))]

	p.log.Debug("Using proxy", zap.String("proxy", u.String()), zap.Uint32("idx", p.curProxyIdx%uint32(len(p.proxyURLs))))

	atomic.AddUint32(&p.curProxyIdx, 1)

	var dialer goproxy.Dialer
	var err error

	dialer = goproxy.Direct

	dialer, err = goproxy.FromURL(u, goproxy.Direct)
	if err != nil {
		panic(err)
	}

	// setup a http client
	httpTransport := &http.Transport{}
	httpClient := &http.Client{Transport: httpTransport}
	httpTransport.Dial = dialer.Dial

	return httpClient
}

func (p *proxyManager) setup(conf *appConfig) error {
	p.log.Info("Initializing core proxy config")

	// Make sure they use socks prefix
	proxyAddresses := make([]string, len(conf.Proxies))
	for i, pr := range conf.Proxies {
		proxyAddresses[i] = fmt.Sprintf("socks5://%s", pr.LocalAddr)
	}

	// Create addresses for non-colly use
	urls := make([]*url.URL, len(proxyAddresses))
	for i, u := range proxyAddresses {
		parsedU, err := url.Parse(u)
		if err != nil {
			return err
		}
		urls[i] = parsedU
	}

	p.proxyURLs = urls

	return nil
}

func (p *proxyManager) setupColly(collector *colly.Collector, conf *appConfig) error {
	p.log.Info("Initializing coolly proxy config")

	// Make sure they use socks prefix
	proxyAddresses := make([]string, len(conf.Proxies))
	for i, pr := range conf.Proxies {
		proxyAddresses[i] = fmt.Sprintf("socks5://%s", pr.LocalAddr)
	}
	// Setup colly proxy func
	proxyFunc, err := proxy.RoundRobinProxySwitcher(proxyAddresses...)
	if err != nil {
		return err
	}

	collector.SetProxyFunc(proxyFunc)
	return nil
}
