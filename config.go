package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/armon/go-socks5"
	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
)

type proxyContext struct {
	Username   string `mapstructure:"username"`
	RemoteAddr string `mapstructure:"remote_addr"`
	LocalAddr  string `mapstructure:"local_addr"`
	conn       *ssh.Client
	socks      *socks5.Server
	signer     ssh.Signer
}

type appConfig struct {
	ListenAddr               string          `mapstructure:"listen_addr"`
	SteamAPIKey              string          `mapstructure:"steam_api_key"`
	DSN                      string          `mapstructure:"dsn"`
	RunMode                  string          `mapstructure:"run_mode"`
	LogLevel                 string          `mapstructure:"log_level"`
	LogFileEnabled           bool            `mapstructure:"log_file_enabled"`
	LogFilePath              string          `mapstructure:"log_file_path"`
	SourcebansScraperEnabled bool            `mapstructure:"sourcebans_scraper_enabled"`
	ProxiesEnabled           bool            `mapstructure:"proxies_enabled"`
	Proxies                  []*proxyContext `mapstructure:"proxies"`
	PrivateKeyPath           string          `mapstructure:"private_key_path"`
	EnableCache              bool            `mapstructure:"enable_cache"`
	CacheDir                 string          `mapstructure:"cache_dir"`
}

func makeSigner(keyPath string) (ssh.Signer, error) { //nolint:ireturn
	privateKeyBody, errPKBody := os.ReadFile(keyPath)
	if errPKBody != nil {
		return nil, errors.Join(errPKBody, domain.ErrSSHPrivateKeyRead)
	}

	var signer ssh.Signer
	key, keyFound := os.LookupEnv("PASSWORD")

	if keyFound {
		newSigner, errSigner := ssh.ParsePrivateKeyWithPassphrase(privateKeyBody, []byte(key))
		if errSigner != nil {
			return nil, errors.Join(errSigner, domain.ErrSSPPrivateKeyParse)
		}

		signer = newSigner
	} else {
		newSigner, errSigner := ssh.ParsePrivateKey(privateKeyBody)
		if errSigner != nil {
			return nil, errors.Join(errSigner, domain.ErrSSPPrivateKeyParse)
		}

		signer = newSigner
	}

	return signer, nil
}

func readConfig(config *appConfig) error {
	if home, errHomeDir := homedir.Dir(); errHomeDir == nil {
		viper.AddConfigPath(home)
	}

	viper.AddConfigPath(".")
	viper.SetConfigName("bdapi")
	viper.SetConfigType("yml")
	viper.SetEnvPrefix("bdapi")
	viper.AutomaticEnv()

	if errReadConfig := viper.ReadInConfig(); errReadConfig != nil {
		return errors.Join(errReadConfig, domain.ErrConfigRead)
	}

	if errUnmarshal := viper.Unmarshal(config); errUnmarshal != nil {
		return errors.Join(errUnmarshal, domain.ErrConfigDecode)
	}

	if config.SteamAPIKey == "" {
		return domain.ErrConfigSteamKeyInvalid
	}

	if errSteam := steamid.SetKey(config.SteamAPIKey); errSteam != nil {
		return fmt.Errorf("%w: %w", domain.ErrConfigSteamKey, errSteam)
	}

	if errSteamWeb := steamweb.SetKey(config.SteamAPIKey); errSteamWeb != nil {
		return fmt.Errorf("%w: %w", domain.ErrConfigSteamKey, errSteamWeb)
	}

	if config.ProxiesEnabled {
		signer, errSigner := makeSigner(config.PrivateKeyPath)
		if errSigner != nil {
			return errors.Join(errSigner, domain.ErrSSHSignerCreate)
		}

		for _, cfg := range config.Proxies {
			cfg.signer = signer
		}
	}

	return nil
}
