# bd-api

Simple api that provides a caching proxy for fetching various TF2 related data points from 3rd party apis and scraped
site data.

There is also a very basic index page that allows for manual searching using the APIs.

## Scraper Proxies

The scraper functionality is currently designed, but not required, to work via establishing a socks5 proxy over ssh (`ssh -D`). The scraper 
cycles through all the configured ssh endpoints when making requests.

## Data Sources

If you know of a good data source that should be index in addition to these, please dont hesitate to open
a issue. If it seems worthwhile it can be added.

### Source Bans 

Close to 100 sites are currently indexed. Support is included for scraping the most commonly used 3rd party themes.

### Logs.tf

Most logs.tf match data is indexed. Some info such as chat logs and player class specific weapon stats are currently omitted, 
however, they may be included in the future.

### Serveme.tf 

The serveme data is taken directly from their [CSV](https://github.com/Arie/serveme/blob/master/doc/banned_steam_ids.csv) and 
returned in a JSON format.

### Bot Detector

There is support for adding bot detector compatible lists to index and enable searching. These
lists need to be inserted into the database manually, this tool does not come with any predefined.

### Competitive Leagues (RGL, ETF2L, UGC, More?)

There is some preliminary support for scraping this data, however its not functional yet.

## API

For more detailed info on the api requests/responses, please check out the [API docs](docs/API.md)
        
## Configuration

Config can be set using either the config file or environment vars. There are no cli args supported currently. 

```yaml
dsn: "postgresql://bdapi:bdapi@localhost:5445/bdapi"
run_mode: release
private_key_path: "./private.key"
steam_api_key: "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
listen_addr: ":8888"
sourcebans_scraper_enabled: true
enable_cache: true
# One of: debug, info, warn, error, dpanic, panic, fatal
log_level: "info"
log_file_enabled: true
log_file_path: "bdapi.log"
cache_dir: "./.cache/"
proxies_enabled: true
proxies:
  - username: user
    remote_addr: sea-1.us.kittyland.com:22
    # Must be unique port for each connection
    local_addr: localhost:3000
  - username: user
    remote_addr: sfo-1.us.kittyland.com:22
    local_addr: localhost:3001
```

You can override these values using matching environment vars with the `BDAPI` prefix like so:

    $ BDAPI_STEAM_API_KEY=ANOTHERSTEAMAPIKEY ./bd-api

## Development Workflow

First, you will want to ensure you are using the filesystem cache, so you don't hammer the servers unnecessarily. See
[scrape_test.go](scrape_test.go) for examples on using a local saved copy.

```yml
enable_cache: true
```

Bring up a temporary database using docker. It will get recreated on every launch. To connect to it, use the following
dsn: `postgresql://bdapi:bdapi@localhost:5445/bdapi`

    make dev_db

Build and run
        
    go build && ./bd-api

## Summary 

![apis](https://imgs.xkcd.com/comics/standards.png)
