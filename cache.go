package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	errCacheExpired = errors.New("cache expired")
	errCacheInit    = errors.New("failed to init cache")
	errCacheRead    = errors.New("failed to read cached file")
	errCacheCreate  = errors.New("failed to create caches file")
	errCacheWrite   = errors.New("failed to write cache data")
)

type CacheKeyType string

const (
	KeySummary CacheKeyType = "summary"
	KeyBans    CacheKeyType = "bans"
	KeyFriends CacheKeyType = "friends"
	KeyRGL                  = "rgl"
)

func makeKey(keyType CacheKeyType, sid64 steamid.SteamID) string {
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

func createCache(enabled bool, cacheDir string) (cache, error) {
	if !enabled {
		return &nopCache{}, nil
	}

	localCache, cacheErr := newFSCache(cacheDir)

	if cacheErr != nil {
		return nil, cacheErr
	}

	return localCache, nil
}

type fsCache struct {
	cacheDir string
}

func newFSCache(cacheDir string) (*fsCache, error) {
	const cachePerms = 0o755

	if !exists(cacheDir) {
		if errMkDir := os.MkdirAll(cacheDir, cachePerms); errMkDir != nil {
			return nil, errors.Join(errMkDir, errCacheInit)
		}
	}

	return &fsCache{cacheDir: cacheDir}, nil
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
		slog.Error("Could not stat file", ErrAttr(errStat), slog.String("file", fullPath))

		return nil, errCacheExpired
	}

	defer logCloser(cachedFile)

	if time.Since(stat.ModTime()) > maxAge {
		return nil, errCacheExpired
	}

	body, errRead := io.ReadAll(cachedFile)
	if errRead != nil {
		return nil, errors.Join(errRead, errCacheRead)
	}

	return body, nil
}

func (c *fsCache) set(key string, reader io.Reader) error {
	dir, fullPath := c.hashKey(key)

	if errDir := os.MkdirAll(dir, os.ModePerm); errDir != nil {
		return errors.Join(errDir, errCacheInit)
	}

	outFile, errOF := os.Create(fullPath)
	if errOF != nil {
		return errors.Join(errOF, errCacheCreate)
	}

	defer logCloser(outFile)

	_, errWrite := io.Copy(outFile, reader)
	if errWrite != nil {
		return errors.Join(errWrite, errCacheWrite)
	}

	return nil
}
