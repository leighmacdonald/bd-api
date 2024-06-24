package domain

import "errors"

var (
	ErrProxySwitcher   = errors.New("failed to create proxy round robin switcher")
	ErrGoQueryDocument = errors.New("failed to create document reader")
	ErrParseTemplate   = errors.New("failed to parse html template")
	ErrInvalidID       = errors.New("invalid id")

	ErrDatabaseDSN       = errors.New("failed to parse database dsn")
	ErrDatabaseMigrate   = errors.New("failed to migrate database")
	ErrDatabaseConnect   = errors.New("failed to connect to database")
	ErrDatabaseUnique    = errors.New("unique record violation")
	ErrDatabaseNoResults = errors.New("no results")
	ErrDatabaseQuery     = errors.New("query error")

	ErrSteamBanFetch      = errors.New("failed to fetch steam ban state")
	ErrSteamBanDecode     = errors.New("failed to decode steam ban state")
	ErrSteamSummaryFetch  = errors.New("failed to fetch steam summary")
	ErrSteamSummaryDecode = errors.New("failed to decode steam summary")

	ErrCacheInit   = errors.New("failed to init cache")
	ErrCacheRead   = errors.New("failed to read cached file")
	ErrCacheCreate = errors.New("failed to create caches file")
	ErrCacheWrite  = errors.New("failed to write cache data")

	ErrSSHDial            = errors.New("failed to dial ssh host")
	ErrSSHPrivateKeyRead  = errors.New("cannot read private key")
	ErrSSPPrivateKeyParse = errors.New("failed to parse private key")
	ErrSSHSignerCreate    = errors.New("failed to setup SSH signer")

	ErrConfigRead            = errors.New("failed to read config file")
	ErrConfigDecode          = errors.New("invalid config file format")
	ErrConfigSteamKey        = errors.New("failed to set steamid key")
	ErrConfigSteamKeyInvalid = errors.New("invalid steam api key [empty]")

	ErrRequestCreate    = errors.New("failed to create request")
	ErrRequestPerform   = errors.New("failed to perform request")
	ErrResponseInvalid  = errors.New("got unexpected results")
	ErrResponseRead     = errors.New("failed to read response body")
	ErrResponseDecode   = errors.New("failed to decode response")
	ErrResponseJSON     = errors.New("failed to generate json response")
	ErrResponseTokenize = errors.New("failed to tokenize json body")
	ErrResponseCSS      = errors.New("failed to write css body")
	ErrResponseFormat   = errors.New("failed to format body")

	ErrScrapInit          = errors.New("failed to initialize all scrapers")
	ErrScrapeURL          = errors.New("failed to parse scraper URL")
	ErrScrapeQueueInit    = errors.New("failed to initialize scraper queue")
	ErrScrapeLimit        = errors.New("failed to set scraper limit")
	ErrScrapeLauncherInit = errors.New("failed to setup browser launcher")
	ErrScrapeWait         = errors.New("failed to wait for content load")
	ErrScrapeCFOpen       = errors.New("could not open cloudflare transport")
	ErrScrapeParseTime    = errors.New("failed to parse time value")
)
