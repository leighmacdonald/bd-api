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

## Summary 

![apis](https://imgs.xkcd.com/comics/standards.png)
