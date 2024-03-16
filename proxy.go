package main

import (
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/armon/go-socks5"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/proxy"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/context"
)

type proxyManager struct {
	proxies map[string]*proxyContext
}

func newProxyManager() *proxyManager {
	return &proxyManager{proxies: map[string]*proxyContext{}}
}

func (p *proxyManager) start(config *appConfig) {
	waitGroup := &sync.WaitGroup{}
	for _, serverCfg := range config.Proxies {
		waitGroup.Add(1)

		go func(cfg *proxyContext) {
			sshConf := ssh.ClientConfig{User: cfg.Username, Auth: []ssh.AuthMethod{ //nolint:exhaustruct
				ssh.PublicKeys(cfg.signer),
			}, HostKeyCallback: ssh.InsecureIgnoreHostKey()} //nolint:gosec

			conn, errConn := ssh.Dial("tcp", cfg.RemoteAddr, &sshConf)
			if errConn != nil {
				slog.Error("Failed to connect to host", ErrAttr(errConn))
				waitGroup.Done()

				return
			}

			slog.Info("Connect to ssh host", slog.String("addr", cfg.RemoteAddr))

			socksConf := &socks5.Config{ //nolint:exhaustruct
				Dial: func(ctx context.Context, network string, addr string) (net.Conn, error) {
					dialedConn, errDial := conn.DialContext(ctx, network, addr)
					if errDial != nil {
						return nil, errors.Wrap(errDial, "Failed to dial network")
					}

					return dialedConn, nil
				},
			}

			socksServer, errServer := socks5.New(socksConf)
			if errServer != nil {
				slog.Error("Failed to initialize socks5", ErrAttr(errServer))
				waitGroup.Done()

				return
			}

			slog.Info("Starting socks5 service", slog.String("addr", cfg.LocalAddr))

			cfg.conn = conn
			cfg.socks = socksServer
			p.proxies[cfg.RemoteAddr] = cfg

			waitGroup.Done()

			if errListen := socksServer.ListenAndServe("tcp", cfg.LocalAddr); errListen != nil {
				slog.Error("Socks5 listener returned error", ErrAttr(errListen))
			}
		}(serverCfg)
	}

	waitGroup.Wait()
}

func (p *proxyManager) stop() {
	waitGroup := &sync.WaitGroup{}
	for _, curProxy := range p.proxies {
		waitGroup.Add(1)

		go func(proxyConf *proxyContext) {
			defer waitGroup.Done()

			if errClose := proxyConf.conn.Close(); errClose != nil {
				slog.Error("Error closing connection", ErrAttr(errClose), slog.String("addr", proxyConf.RemoteAddr))

				return
			}

			slog.Info("Closed connection", slog.String("addr", proxyConf.RemoteAddr))
		}(curProxy)
	}

	waitGroup.Wait()
}

func (p *proxyManager) setup(collector *colly.Collector, config *appConfig) error {
	proxyAddresses := make([]string, len(config.Proxies))
	for i, p := range config.Proxies {
		proxyAddresses[i] = fmt.Sprintf("socks5://%s", p.LocalAddr)
	}

	proxiesFunc, errProxies := proxy.RoundRobinProxySwitcher(proxyAddresses...)
	if errProxies != nil {
		return errors.Wrap(errProxies, "Failed to create proxy round robin proxy switcher")
	}

	collector.SetProxyFunc(proxiesFunc)

	return nil
}
