package writer

import (
	"encoding/json"
	"fmt"
	"io"
)

type jsonWriter struct {
	out     io.Writer
	enc     *json.Encoder
	cols    []string
	first   bool
	started bool
	closed  bool
}

func NewJSON(out io.Writer) Writer {
	return &jsonWriter{out: out, enc: json.NewEncoder(out), first: true}
}
func (w *jsonWriter) WriteHeader(cols []string) error {
	if err := checkColumns(cols); err != nil {
		return err
	}
	w.cols = append([]string{}, cols...)
	_, err := io.WriteString(w.out, "[")
	w.started = err == nil
	return err
}
func (w *jsonWriter) WriteRow(vals []any) error {
	if err := checkRow(w.cols, vals); err != nil {
		return err
	}
	if !w.first {
		if _, err := io.WriteString(w.out, ","); err != nil {
			return err
		}
	}
	w.first = false
	return w.enc.Encode(rowObject(w.cols, vals))
}
func (w *jsonWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	if !w.started {
		return fmt.Errorf("JSON header not written")
	}
	_, err := io.WriteString(w.out, "]\n")
	return err
}
func checkColumns(cols []string) error {
	seen := make(map[string]bool, len(cols))
	for _, col := range cols {
		if seen[col] {
			return fmt.Errorf("duplicate result column %q for JSON output", col)
		}
		seen[col] = true
	}
	return nil
}
func rowObject(cols []string, vals []any) map[string]any {
	obj := make(map[string]any, len(cols))
	for i, c := range cols {
		obj[c] = normalize(vals[i])
	}
	return obj
}
