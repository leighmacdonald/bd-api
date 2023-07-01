package main

import (
	"io"
	"os"
)

func logCloser(closer io.Closer) {
	_ = closer.Close()
}

func exists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	return true
}
