package main

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
)

var reLOGSResults = regexp.MustCompile(`<p>(\d+|\d+,\d+)\sresults</p>`)

func getLogsTF(ctx context.Context, steamid steamid.SID64) (int64, error) {
	resp, err := get(ctx, fmt.Sprintf("https://logs.tf/profile/%d", steamid.Int64()), nil)
	if err != nil {
		return 0, err
	}
	b, errRead := io.ReadAll(resp.Body)
	if errRead != nil {
		return 0, errRead
	}
	defer logCloser(resp.Body)
	bStr := string(b)
	if strings.Contains(bStr, "No logs found.") {
		return 0, nil
	}
	m := reLOGSResults.FindStringSubmatch(bStr)
	if len(m) != 2 {
		log.Printf("Got unexpected results for logs.tf\n")
		return 0, nil
	}
	value := strings.ReplaceAll(m[1], ",", "")
	count, errParse := strconv.ParseInt(value, 10, 64)
	if errParse != nil || count <= 0 {
		return 0, errors.Wrapf(errParse, "Failed to parse results count: %s", m[1])
	}
	return count, nil
}
