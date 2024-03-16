package main

import (
	"io"
	"os"
	"strings"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

func logCloser(closer io.Closer) {
	_ = closer.Close()
}

func exists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	return true
}

func SteamIDStringList(collection steamid.Collection) string {
	ids := make([]string, len(collection))
	for index, steamID := range collection {
		ids[index] = steamID.String()
	}

	return strings.Join(ids, ",")
}
