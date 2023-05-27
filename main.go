package main

import (
	"context"
	"github.com/leighmacdonald/steamweb"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"regexp"
)

func main() {
	if errServe := http.ListenAndServe(":8090", nil); errServe != nil {
		log.Printf("HTTP Server returned error: %v", errServe)
	}
}

var (
	cache  caches
	ctx    context.Context
	logger *zap.Logger
)

func init() {
	ctx = context.Background()
	visitors = map[string]*visitor{}

	newLogger, errLogger := zap.NewProduction()
	if errLogger != nil {
		panic(errLogger)
	}
	logger = newLogger

	cache = newCaches(ctx, steamCacheTimeout, compCacheTimeout, steamCacheTimeout)
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

	go cleanupVisitors()
}
