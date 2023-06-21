package main

import (
	"io"
	"os"

	"go.uber.org/zap"
)

func logCloser(closer io.Closer) {
	if errClose := closer.Close(); errClose != nil {
		logger.Error("Failed to close", zap.Error(errClose))
	}
}

func exists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	return true
}
