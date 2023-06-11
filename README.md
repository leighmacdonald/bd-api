# bd-api

Simple api that provides a caching proxy for fetching various TF2 related data points from 3rd party apis and scraped
site data.


## SourceBans scraper

The scraper functionality is currently designed to work via establishing a socks5 proxy over ssh (`ssh -D`). The scraper 
cycles through all the configured ssh endpoints when making requests.

## Example config.yml

```yaml
private_key_path: "./private.key"
proxies:
  - username: user
    remote_addr: sea-1.us.kittyland.com:22
    # Must be unique port for each connection
    local_addr: localhost:3000
  - username: user
    remote_addr: sfo-1.us.kittyland.com:22
    local_addr: localhost:3001
```

## Summary 

![apis](https://imgs.xkcd.com/comics/standards.png)
