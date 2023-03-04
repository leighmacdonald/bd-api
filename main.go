package main

import (
	"context"
	"github.com/leighmacdonald/steamweb"
	"log"
	"net/http"
	"os"
	"regexp"
)

func main() {
	ctx := context.Background()

	cache := newCaches(ctx, steamCacheTimeout, compCacheTimeout, steamCacheTimeout)

	http.HandleFunc("/bans", limit(getHandler(handleGetBans(cache))))
	http.HandleFunc("/summary", limit(getHandler(handleGetSummary(cache))))
	http.HandleFunc("/profile", limit(getHandler(handleGetProfile(cache))))
	http.HandleFunc("/kick", limit(onPostKick))

	if errServe := http.ListenAndServe(":8090", nil); errServe != nil {
		log.Printf("HTTP Server returned error: %v", errServe)
	}
}

func init() {
	steamAPIKey, steamAPIKeyFound := os.LookupEnv("STEAM_API_KEY")
	if !steamAPIKeyFound || steamAPIKey == "" {
		log.Panicf("Must set STEAM_API_KEY")
	}
	if errSetKey := steamweb.SetKey(steamAPIKey); errSetKey != nil {
		log.Panicf("Failed to set steam api key: %v\n", errSetKey)
	}
	reLOGSResults = regexp.MustCompile(`<p>(\d+|\d+,\d+)\sresults</p>`)
	//reETF2L = regexp.MustCompile(`.org/forum/user/(\d+)`)
	reUGCRank = regexp.MustCompile(`Season (\d+) (\D+) (\S+)`)
}
