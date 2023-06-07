package main

import (
	"fmt"
	"github.com/armon/go-socks5"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/proxy"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/context"
	"net"
	"sync"
)

var proxies map[string]*proxyConfig

func init() {
	proxies = map[string]*proxyConfig{}
}

func startProxies(config *appConfig) {
	wg := &sync.WaitGroup{}
	for _, serverCfg := range config.Proxies {
		wg.Add(1)
		go func(cfg *proxyConfig) {
			sshConf := ssh.ClientConfig{User: cfg.Username, Auth: []ssh.AuthMethod{
				ssh.PublicKeys(cfg.signer),
			}, HostKeyCallback: ssh.InsecureIgnoreHostKey()}
			conn, errConn := ssh.Dial("tcp", cfg.RemoteAddr, &sshConf)
			if errConn != nil {
				logger.Error("Failed to connect to host", zap.Error(errConn))
				wg.Done()
				return
			}
			logger.Info("Connect to ssh host", zap.String("addr", cfg.RemoteAddr))
			socksConf := &socks5.Config{Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return conn.Dial(network, addr)
			}}
			server, errServer := socks5.New(socksConf)
			if errServer != nil {
				logger.Error("Failed to initialize socks5", zap.Error(errServer))
				wg.Done()
				return
			}
			logger.Info("Starting socks5 service", zap.String("addr", cfg.LocalAddr))
			cfg.conn = conn
			cfg.socks = server
			proxies[cfg.RemoteAddr] = cfg
			wg.Done()
			if errListen := server.ListenAndServe("tcp", cfg.LocalAddr); errListen != nil {
				logger.Error("Socks5 listener returned error", zap.Error(errListen))
			}
		}(serverCfg)
	}
	wg.Wait()

}

func stopProxies() {
	wg := &sync.WaitGroup{}
	for _, curProxy := range proxies {
		wg.Add(1)
		go func(p *proxyConfig) {
			defer wg.Done()
			if errClose := p.conn.Close(); errClose != nil {
				logger.Error("Error closing connection", zap.Error(errClose), zap.String("addr", p.RemoteAddr))
				return
			}
			logger.Info("Closed connection", zap.String("addr", p.RemoteAddr))
		}(curProxy)
	}
	wg.Wait()
}

func setupProxies(c *colly.Collector, config *appConfig) error {
	var proxyAddresses []string
	for _, p := range config.Proxies {
		proxyAddresses = append(proxyAddresses, fmt.Sprintf("socks5://%s", p.LocalAddr))
	}
	proxiesFunc, errProxies := proxy.RoundRobinProxySwitcher(proxyAddresses...)
	if errProxies != nil {
		return errProxies
	}
	c.SetProxyFunc(proxiesFunc)
	return nil
}
