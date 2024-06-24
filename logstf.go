package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

var reLOGSResults = regexp.MustCompile(`<p>(\d+|\d+,\d+)\sresults</p>`)

func getLogsTF(ctx context.Context, steamid steamid.SteamID) (int64, error) {
	const expectedMatches = 2

	resp, err := get(ctx, fmt.Sprintf("https://logs.tf/profile/%d", steamid.Int64()), nil)
	if err != nil {
		return 0, err
	}

	defer logCloser(resp.Body)

	body, errRead := io.ReadAll(resp.Body)
	if errRead != nil {
		return 0, errors.Join(errRead, errResponseRead)
	}

	bStr := string(body)
	if strings.Contains(bStr, "No logs found.") {
		return 0, nil
	}

	match := reLOGSResults.FindStringSubmatch(bStr)
	if len(match) != expectedMatches {
		return 0, errResponseInvalid
	}

	value := strings.ReplaceAll(match[1], ",", "")

	count, errParse := strconv.ParseInt(value, 10, 64)
	if errParse != nil || count <= 0 {
		return 0, errors.Join(errParse, fmt.Errorf("%w: %s", errResponseDecode, match[1]))
	}

	return count, nil
}
