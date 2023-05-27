package main

import (
	"go.uber.org/zap"
	"io"
)

func logCloser(closer io.Closer) {
	if errClose := closer.Close(); errClose != nil {
		logger.Error("Failed to close", zap.Error(errClose))
	}
}
