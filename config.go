package main

import (
	"github.com/armon/go-socks5"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
	"os"
)

type proxyConfig struct {
	Username   string `yaml:"username"`
	RemoteAddr string `yaml:"remote_addr"`
	LocalAddr  string `yaml:"local_addr"`
	conn       *ssh.Client
	socks      *socks5.Server
	signer     ssh.Signer
}

type appConfig struct {
	ListenAddr               string         `yaml:"listen_addr"`
	SteamAPIKey              string         `yaml:"steam_api_key"`
	DSN                      string         `yaml:"dsn"`
	SourcebansScraperEnabled bool           `yaml:"sourcebans_scraper_enabled"`
	Proxies                  []*proxyConfig `yaml:"proxies"`
	PrivateKeyPath           string         `yaml:"private_key_path"`
	EnableCache              bool           `yaml:"enable_cache"`
}

func makeSigner(keyPath string) (ssh.Signer, error) {
	privateKeyBody, errPKBody := os.ReadFile(keyPath)
	if errPKBody != nil {
		logger.Panic("Cannot read private key", zap.String("path", keyPath))
	}
	var signer ssh.Signer
	key, keyFound := os.LookupEnv("PASSWORD")

	if keyFound {
		newSigner, errSigner := ssh.ParsePrivateKeyWithPassphrase(privateKeyBody, []byte(key))
		if errSigner != nil {
			logger.Panic("Failed to parse private key", zap.Error(errSigner))
		}
		signer = newSigner
	} else {
		newSigner, errSigner := ssh.ParsePrivateKey(privateKeyBody)
		if errSigner != nil {
			logger.Panic("Failed to parse private key", zap.Error(errSigner))
		}
		signer = newSigner
	}
	return signer, nil
}

func readConfig(configFile string, config *appConfig) error {
	cf, errCf := os.Open(configFile)
	if errCf != nil {
		return errCf
	}
	defer logCloser(cf)
	if errDecode := yaml.NewDecoder(cf).Decode(&config); errDecode != nil {
		return errDecode
	}
	signer, errSigner := makeSigner(config.PrivateKeyPath)
	if errSigner != nil {
		return errors.Wrap(errSigner, "Failed to setup SSH signer")
	}
	for _, cfg := range config.Proxies {
		cfg.signer = signer
	}
	if key, found := os.LookupEnv("STEAM_API_KEY"); found && key != "" {
		if errKey := steamweb.SetKey(key); errKey != nil {
			return errKey
		}
	}
	return nil
}
