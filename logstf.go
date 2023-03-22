package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var reLOGSResults *regexp.Regexp

func getLogsTF(ctx context.Context, steamid steamid.SID64) (int64, error) {
	fullURL := fmt.Sprintf("https://logs.tf/profile/%d", steamid.Int64())
	body, errCache := cacheGet(fullURL)
	if errCache != nil {
		if !errors.Is(errCache, errCacheExpired) {
			return 0, errCache
		}
		resp, err := get(ctx, fullURL, nil)
		if err != nil {
			return 0, err
		}
		newBody, errRead := io.ReadAll(resp.Body)
		if errRead != nil {
			return 0, errRead
		}
		defer logClose(resp.Body)
		body = newBody
		if errSet := cacheSet(fullURL, bytes.NewReader(newBody)); errSet != nil {
			logger.Error("Failed to update cache", zap.Error(errSet), zap.String("site", "logs.tf"))
		}
	}
	stringBody := string(body)
	if strings.Contains(stringBody, "No logs found.") {
		return 0, nil
	}
	m := reLOGSResults.FindStringSubmatch(stringBody)
	if len(m) != 2 {
		logger.Error("Got unexpected results for logs.tf")
		return 0, nil
	}
	value := strings.ReplaceAll(m[1], ",", "")
	count, errParse := strconv.ParseInt(value, 10, 64)
	if errParse != nil || count <= 0 {
		return 0, errors.Wrapf(errParse, "Failed to parse results count: %s", m[1])
	}
	return count, nil
}
