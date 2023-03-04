package main

import (
	"github.com/leighmacdonald/steamweb"
	"log"
	"net/http"
	"os"
	"regexp"
)

func main() {

	http.HandleFunc("/bans", limit(getHandler(handleGetBans())))
	http.HandleFunc("/summary", limit(getHandler(handleGetSummary())))
	http.HandleFunc("/competitive", limit(getHandler(handleGetCompetitive())))
	http.HandleFunc("/kick", limit(onPostKick))

	if errServe := http.ListenAndServe(":8090", nil); errServe != nil {
		log.Printf("HTTP Server returned error: %v", errServe)
	}
}

func init() {
	steamApiKey, steamApiKeyFound := os.LookupEnv("STEAM_API_KEY")
	if !steamApiKeyFound || steamApiKey == "" {
		log.Panicf("Must set STEAM_API_KEY")
	}
	if errSetKey := steamweb.SetKey(steamApiKey); errSetKey != nil {
		log.Panicf("Failed to set steam api key: %v\n", errSetKey)
	}
	reLOGSResults = regexp.MustCompile(`<p>(\d+|\d+,\d+)\sresults</p>`)
	//reETF2L = regexp.MustCompile(`.org/forum/user/(\d+)`)
	reUGCRank = regexp.MustCompile(`Season (\d+) (\D+) (\S+)`)
}
