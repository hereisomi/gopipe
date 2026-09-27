// Package writer streams database rows to delimited or JSON output.
package writer

import (
	"fmt"
	"io"
	"strings"
)

// Writer consumes one result set in order. Close finishes any buffered output
// but does not close the underlying io.Writer, which remains owned by the caller.
type Writer interface {
	WriteHeader(cols []string) error
	WriteRow(vals []any) error
	Close() error
}

// New selects a streaming result writer; it does not buffer the result set.
func New(format string, out io.Writer) (Writer, error) {
	switch strings.ToLower(format) {
	case "csv":
		return NewCSV(out), nil
	case "tsv":
		return NewTSV(out), nil
	case "json":
		return NewJSON(out), nil
	case "jsonl":
		return NewJSONL(out), nil
	default:
		return nil, fmt.Errorf("unsupported output format %q", format)
	}
}

func checkRow(cols []string, vals []any) error {
	if cols == nil {
		return fmt.Errorf("write header before rows")
	}
	if len(cols) != len(vals) {
		return fmt.Errorf("row has %d values; expected %d", len(vals), len(cols))
	}
	return nil
}

// Normalizing []byte to text avoids JSON's base64 encoding for database/sql
// text columns (database/sql often returns VARCHAR as []byte).
func normalize(v any) any {
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return v
}

func text(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(normalize(v))
}
