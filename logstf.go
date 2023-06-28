package main

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
)

var reLOGSResults = regexp.MustCompile(`<p>(\d+|\d+,\d+)\sresults</p>`)

func getLogsTF(ctx context.Context, steamid steamid.SID64) (int64, error) {
	const expectedMatches = 2

	resp, err := get(ctx, fmt.Sprintf("https://logs.tf/profile/%d", steamid.Int64()), nil)
	if err != nil {
		return 0, err
	}

	body, errRead := io.ReadAll(resp.Body)
	if errRead != nil {
		return 0, errors.Wrap(errRead, "Failed to read response body")
	}

	defer logCloser(resp.Body)

	bStr := string(body)
	if strings.Contains(bStr, "No logs found.") {
		return 0, nil
	}

	match := reLOGSResults.FindStringSubmatch(bStr)
	if len(match) != expectedMatches {
		return 0, errors.New("Got unexpected results for logs.tf")
	}

	value := strings.ReplaceAll(match[1], ",", "")

	count, errParse := strconv.ParseInt(value, 10, 64)
	if errParse != nil || count <= 0 {
		return 0, errors.Wrapf(errParse, "Failed to parse results count: %s", match[1])
	}

	return count, nil
}
