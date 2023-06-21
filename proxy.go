package main

import (
	"fmt"
	"net"
	"sync"

	"github.com/armon/go-socks5"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/proxy"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/context"
)

var proxies map[string]*proxyConfig

func init() {
	proxies = map[string]*proxyConfig{}
}

func startProxies(config *appConfig) {
	waitGroup := &sync.WaitGroup{}
	for _, serverCfg := range config.Proxies {
		waitGroup.Add(1)

		go func(cfg *proxyConfig) {
			sshConf := ssh.ClientConfig{User: cfg.Username, Auth: []ssh.AuthMethod{ //nolint:exhaustruct
				ssh.PublicKeys(cfg.signer),
			}, HostKeyCallback: ssh.InsecureIgnoreHostKey()} //nolint:gosec

			conn, errConn := ssh.Dial("tcp", cfg.RemoteAddr, &sshConf)
			if errConn != nil {
				logger.Error("Failed to connect to host", zap.Error(errConn))
				waitGroup.Done()

				return
			}

			logger.Info("Connect to ssh host", zap.String("addr", cfg.RemoteAddr))

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
				logger.Error("Failed to initialize socks5", zap.Error(errServer))
				waitGroup.Done()

				return
			}

			logger.Info("Starting socks5 service", zap.String("addr", cfg.LocalAddr))

			cfg.conn = conn
			cfg.socks = server
			proxies[cfg.RemoteAddr] = cfg

			waitGroup.Done()

			if errListen := server.ListenAndServe("tcp", cfg.LocalAddr); errListen != nil {
				logger.Error("Socks5 listener returned error", zap.Error(errListen))
			}
		}(serverCfg)
	}

	waitGroup.Wait()
}

func stopProxies() {
	waitGroup := &sync.WaitGroup{}
	for _, curProxy := range proxies {
		waitGroup.Add(1)

		go func(proxyConf *proxyConfig) {
			defer waitGroup.Done()

			if errClose := proxyConf.conn.Close(); errClose != nil {
				logger.Error("Error closing connection", zap.Error(errClose), zap.String("addr", proxyConf.RemoteAddr))

				return
			}

			logger.Info("Closed connection", zap.String("addr", proxyConf.RemoteAddr))
		}(curProxy)
	}

	waitGroup.Wait()
}

func setupProxies(collector *colly.Collector, config *appConfig) error {
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
