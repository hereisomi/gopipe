package writer

import (
	"encoding/csv"
	"io"
)

type delimitedWriter struct {
	enc    *csv.Writer
	cols   []string
	closed bool
}

func NewCSV(out io.Writer) Writer { return newDelimited(out, ',') }
func newDelimited(out io.Writer, comma rune) Writer {
	enc := csv.NewWriter(out)
	enc.Comma = comma
	return &delimitedWriter{enc: enc}
}
func (w *delimitedWriter) WriteHeader(cols []string) error {
	w.cols = append([]string{}, cols...)
	return w.enc.Write(cols)
}
func (w *delimitedWriter) WriteRow(vals []any) error {
	if err := checkRow(w.cols, vals); err != nil {
		return err
	}
	rec := make([]string, len(vals))
	for i, v := range vals {
		rec[i] = text(v)
	}
	return w.enc.Write(rec)
}
func (w *delimitedWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	w.enc.Flush()
	return w.enc.Error()
}
