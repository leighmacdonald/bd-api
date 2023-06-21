package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var errCacheExpired = errors.New("cache expired")

func hashKey(fullURL string) (string, string) {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(fullURL)))
	dir := hashedPath(hash)

	return dir, path.Join(dir, hash)
}

func hashedPath(hash string) string {
	return path.Join(cacheDir, hash[0:2], hash[2:4])
}

func cacheGet(url string) ([]byte, error) {
	const maxAge = time.Hour * 24 * 7

	_, fullPath := hashKey(url)

	cachedFile, errOpen := os.Open(fullPath)
	if errOpen != nil {
		return nil, errCacheExpired
	}

	stat, errStat := cachedFile.Stat()
	if errStat != nil {
		logger.Error("Could not stat file",
			zap.Error(errStat), zap.String("file", fullPath))

		return nil, errCacheExpired
	}

	defer logCloser(cachedFile)

	if time.Since(stat.ModTime()) > maxAge {
		return nil, errCacheExpired
	}

	body, errRead := io.ReadAll(cachedFile)
	if errRead != nil {
		return nil, errors.Wrap(errRead, "Failed to reach cached file")
	}

	return body, nil
}

func cacheSet(key string, reader io.Reader) error {
	dir, fullPath := hashKey(key)
	if errDir := os.MkdirAll(dir, os.ModePerm); errDir != nil {
		return errors.Wrap(errDir, "Failed to make cache dir")
	}

	outFile, errOF := os.Create(fullPath)
	if errOF != nil {
		return errors.Wrap(errOF, "Error creating cache file")
	}

	defer logCloser(outFile)

	_, errWrite := io.Copy(outFile, reader)
	if errWrite != nil {
		return errors.Wrap(errWrite, "Failed to write content to file")
	}

	return nil
}
