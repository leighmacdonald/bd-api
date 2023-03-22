package main

import (
	"crypto/sha256"
	"fmt"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"os"
	"path"
	"time"
)

var errCacheExpired = errors.New("cache expired")
var maxAge = time.Hour * 24 * 7

func hashKey(fullURL string) (string, string) {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(fullURL)))
	dir := hashedPath(hash)
	return dir, path.Join(dir, hash)
}

func hashedPath(hash string) string {
	return path.Join(cacheDir, hash[0:2], hash[2:4])
}

func cacheGet(url string) ([]byte, error) {
	_, fullPath := hashKey(url)
	f, errOpen := os.Open(fullPath)
	if errOpen != nil {
		return nil, errCacheExpired
	}
	stat, errStat := f.Stat()
	if errStat != nil {
		logger.Error("Could not stat file",
			zap.Error(errStat), zap.String("file", fullPath))
		return nil, errCacheExpired
	}
	if time.Since(stat.ModTime()) > maxAge {
		return nil, errCacheExpired
	}
	defer logClose(f)
	return io.ReadAll(f)
}

func cacheSet(key string, reader io.Reader) error {
	dir, fullPath := hashKey(key)
	if errDir := os.MkdirAll(dir, os.ModePerm); errDir != nil {
		return errDir
	}
	outFile, errOF := os.Create(fullPath)
	if errOF != nil {
		return errOF
	}
	defer logClose(outFile)
	_, errWrite := io.Copy(outFile, reader)
	return errWrite
}
