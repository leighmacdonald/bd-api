package main

import (
	"github.com/armon/go-socks5"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"os"
)

type proxyConfig struct {
	Username   string `mapstructure:"username"`
	RemoteAddr string `mapstructure:"remote_addr"`
	LocalAddr  string `mapstructure:"local_addr"`
	conn       *ssh.Client
	socks      *socks5.Server
	signer     ssh.Signer
}

type appConfig struct {
	ListenAddr               string         `mapstructure:"listen_addr"`
	SteamAPIKey              string         `mapstructure:"steam_api_key"`
	DSN                      string         `mapstructure:"dsn"`
	RunMode                  string         `mapstructure:"run_mode"`
	SourcebansScraperEnabled bool           `mapstructure:"sourcebans_scraper_enabled"`
	ProxiesEnabled           bool           `mapstructure:"proxies_enabled"`
	Proxies                  []*proxyConfig `mapstructure:"proxies"`
	PrivateKeyPath           string         `mapstructure:"private_key_path"`
	EnableCache              bool           `mapstructure:"enable_cache"`
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

func readConfig(config *appConfig) error {
	if errReadConfig := viper.ReadInConfig(); errReadConfig != nil {
		return errors.Wrapf(errReadConfig, "Failed to read config file")
	}
	if errUnmarshal := viper.Unmarshal(config); errUnmarshal != nil {
		return errors.Wrap(errUnmarshal, "Invalid config file format")
	}
	gin.SetMode(config.RunMode)
	if config.SteamAPIKey == "" {
		return errors.New("Invalid steam api key [empty]")
	}
	if errSteam := steamid.SetKey(config.SteamAPIKey); errSteam != nil {
		return errors.Errorf("Failed to set steamid key: %v", errSteam)
	}
	if errSteamWeb := steamweb.SetKey(config.SteamAPIKey); errSteamWeb != nil {
		return errors.Errorf("Failed to set steamweb key: %v", errSteamWeb)
	}
	if config.ProxiesEnabled {
		signer, errSigner := makeSigner(config.PrivateKeyPath)
		if errSigner != nil {
			return errors.Wrap(errSigner, "Failed to setup SSH signer")
		}
		for _, cfg := range config.Proxies {
			cfg.signer = signer
		}
	}
	return nil
}

func init() {
	if home, errHomeDir := homedir.Dir(); errHomeDir == nil {
		viper.AddConfigPath(home)
	}
	viper.AddConfigPath(".")
	viper.SetConfigName("bdapi")
	viper.SetConfigType("yml")
	viper.SetEnvPrefix("bdapi")
	//viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
}
