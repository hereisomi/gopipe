// Package pipeline — tee writer (spec §F-25, §7.3 contract 8).
package pipeline

import (
	"io"
	"os"
)

// NewTee wraps w so that all writes are also written to path concurrently.
// Returns w unchanged if path is empty.
// The caller must call Close() on the returned closer when done.
func NewTee(w io.Writer, path string) (io.Writer, func(), error) {
	if path == "" {
		return w, func() {}, nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, err
	}
	return io.MultiWriter(w, f), func() { _ = f.Close() }, nil
}
