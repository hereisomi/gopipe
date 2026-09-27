package writer

import (
	"encoding/json"
	"io"
)

type jsonlWriter struct {
	enc  *json.Encoder
	cols []string
}

func NewJSONL(out io.Writer) Writer { return &jsonlWriter{enc: json.NewEncoder(out)} }
func (w *jsonlWriter) WriteHeader(cols []string) error {
	if err := checkColumns(cols); err != nil {
		return err
	}
	w.cols = append([]string{}, cols...)
	return nil
}
func (w *jsonlWriter) WriteRow(vals []any) error {
	if err := checkRow(w.cols, vals); err != nil {
		return err
	}
	return w.enc.Encode(rowObject(w.cols, vals))
}
func (w *jsonlWriter) Close() error { return nil }
