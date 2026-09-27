// Package logger provides a thread-safe structured file logger (spec §F-10).
// Supports plain-text (with optional timestamp) and JSON (one object per line).
package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Level is a log event severity / type label.
type Level string

const (
	LevelInfo    Level = "info"
	LevelError   Level = "error"
	LevelRetry   Level = "retry"
	LevelTimeout Level = "timeout"
	LevelCancel  Level = "cancel"
)

// Entry is one log record (spec §F-10 JSON schema).
type Entry struct {
	Timestamp     string `json:"timestamp"`
	Level         Level  `json:"level"`
	Source        string `json:"source"`
	Message       string `json:"message"`
	ConsumerIndex int    `json:"consumer_index,omitempty"`
	Attempt       int    `json:"attempt,omitempty"`
	ExitCode      int    `json:"exit_code,omitempty"`
}

// Logger writes structured log entries to a file.
type Logger struct {
	mu        sync.Mutex
	w         io.WriteCloser
	json      bool
	timestamp bool
}

// New opens path for appending and returns a Logger.
// If json is true, each entry is written as a JSON object.
// If addTimestamp is true (plain mode), each line is prefixed with a timestamp.
func New(path string, jsonMode, addTimestamp bool) (*Logger, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &Logger{w: f, json: jsonMode, timestamp: addTimestamp}, nil
}

// Log writes one entry. Safe to call from multiple goroutines.
func (l *Logger) Log(e Entry) {
	if e.Timestamp == "" {
		e.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.json {
		b, _ := json.Marshal(e)
		_, _ = fmt.Fprintf(l.w, "%s\n", b)
	} else {
		if l.timestamp {
			_, _ = fmt.Fprintf(l.w, "%s [%s] %s: %s\n", e.Timestamp, e.Level, e.Source, e.Message)
		} else {
			_, _ = fmt.Fprintf(l.w, "[%s] %s: %s\n", e.Level, e.Source, e.Message)
		}
	}
}

// Close flushes and closes the underlying file.
func (l *Logger) Close() error { return l.w.Close() }

// Nop returns a logger that discards all entries (used when --log is not set).
func Nop() *Logger { return &Logger{w: nopCloser{io.Discard}} }

type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }
