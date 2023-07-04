# bd-api

Simple api that provides a caching proxy for fetching various TF2 related data points from 3rd party apis and scraped
site data.


## SourceBans scraper

The scraper functionality is currently designed to work via establishing a socks5 proxy over ssh (`ssh -D`). The scraper 
cycles through all the configured ssh endpoints when making requests.

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

## Pretty JSON

If you make API requests with a browser, or otherwise set the `Accept: text/html` header, the JSON output will be encoded 
as "prettified" HTML with syntax highlighting of the JSON data returned. All other cases will return standard JSON output.

## Development Workflow

First, you will want to ensure you are using the filesystem cache, so you don't hammer the servers unnecessarily. See
`scrape_test.go` for examples on using a local saved copy.

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
