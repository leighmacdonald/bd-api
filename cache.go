package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var errCacheExpired = errors.New("cache expired")

type CacheKeyType string

const (
	KeySummary CacheKeyType = "summary"
	KeyBans    CacheKeyType = "bans"
	KeyFriends CacheKeyType = "friends"
	KeyRGL                  = "rgl"
)

func makeKey(keyType CacheKeyType, sid64 steamid.SID64) string {
	return fmt.Sprintf("steam-%s-%d", keyType, sid64.Int64())
}

type cache interface {
	get(url string) ([]byte, error)
	set(key string, reader io.Reader) error
}

type nopCache struct{}

func (c *nopCache) get(_ string) ([]byte, error) {
	return nil, errCacheExpired
}

func (c *nopCache) set(_ string, _ io.Reader) error {
	return nil
}

type fsCache struct {
	cacheDir string
	log      *zap.Logger
}

func newFSCache(logger *zap.Logger, cacheDir string) (*fsCache, error) {
	const cachePerms = 0o755

	if !exists(cacheDir) {
		if errMkDir := os.MkdirAll(cacheDir, cachePerms); errMkDir != nil {
			return nil, errors.Wrap(errMkDir, "Failed to create cache dir")
		}
	}

	return &fsCache{cacheDir: cacheDir, log: logger.Named("fsCache")}, nil
}

func (c *fsCache) hashKey(fullURL string) (string, string) {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(fullURL)))
	dir := c.hashedPath(hash)

	return dir, path.Join(dir, hash)
}

func (c *fsCache) hashedPath(hash string) string {
	return path.Join(c.cacheDir, hash[0:2], hash[2:4])
}

func (c *fsCache) get(url string) ([]byte, error) {
	const maxAge = time.Hour * 24 * 7

	_, fullPath := c.hashKey(url)

	cachedFile, errOpen := os.Open(fullPath)
	if errOpen != nil {
		return nil, errCacheExpired
	}

	stat, errStat := cachedFile.Stat()
	if errStat != nil {
		c.log.Error("Could not stat file",
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

func (c *fsCache) set(key string, reader io.Reader) error {
	dir, fullPath := c.hashKey(key)
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
