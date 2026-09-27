package writer

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestDelimitedRoundTrip(t *testing.T) {
	for _, format := range []string{"csv", "tsv"} {
		t.Run(format, func(t *testing.T) {
			var out bytes.Buffer
			w, err := New(format, &out)
			if err != nil {
				t.Fatal(err)
			}
			if err = w.WriteHeader([]string{"name", "note"}); err != nil {
				t.Fatal(err)
			}
			if err = w.WriteRow([]any{[]byte("a,\t\n\"b"), nil}); err != nil {
				t.Fatal(err)
			}
			if err = w.Close(); err != nil {
				t.Fatal(err)
			}
			r := csv.NewReader(&out)
			if format == "tsv" {
				r.Comma = '\t'
			}
			header, err := r.Read()
			if err != nil {
				t.Fatal(err)
			}
			row, err := r.Read()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(header, []string{"name", "note"}) || !reflect.DeepEqual(row, []string{"a,\t\n\"b", ""}) {
				t.Fatalf("header=%q row=%q", header, row)
			}
			if _, err := r.Read(); err != io.EOF {
				t.Fatalf("trailing row: %v", err)
			}
		})
	}
}
func TestJSONAndJSONLEmptyAndNull(t *testing.T) {
	for _, format := range []string{"json", "jsonl"} {
		t.Run(format, func(t *testing.T) {
			var out bytes.Buffer
			w, _ := New(format, &out)
			if err := w.WriteHeader([]string{"text", "missing"}); err != nil {
				t.Fatal(err)
			}
			if err := w.WriteRow([]any{[]byte("hello"), nil}); err != nil {
				t.Fatal(err)
			}
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}
			var row map[string]any
			content := out.Bytes()
			if format == "json" {
				var rows []map[string]any
				if err := json.Unmarshal(content, &rows); err != nil {
					t.Fatal(err)
				}
				row = rows[0]
			} else {
				if err := json.Unmarshal(content, &row); err != nil {
					t.Fatal(err)
				}
			}
			if row["text"] != "hello" || row["missing"] != nil {
				t.Fatalf("row = %#v", row)
			}
			out.Reset()
			w, _ = New(format, &out)
			if err := w.WriteHeader([]string{"id"}); err != nil {
				t.Fatal(err)
			}
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}
			if format == "json" && strings.TrimSpace(out.String()) != "[]" {
				t.Errorf("empty JSON = %q", out.String())
			}
			if format == "jsonl" && out.Len() != 0 {
				t.Errorf("empty JSONL = %q", out.String())
			}
		})
	}
}
func TestInvalidRowsAndDuplicateColumns(t *testing.T) {
	for _, format := range []string{"json", "jsonl"} {
		w, _ := New(format, io.Discard)
		if err := w.WriteHeader([]string{"id", "id"}); err == nil {
			t.Errorf("%s accepted duplicate columns", format)
		}
	}
	w, _ := New("csv", io.Discard)
	if err := w.WriteRow([]any{1}); err == nil {
		t.Error("row accepted before header")
	}
	w.WriteHeader([]string{"a", "b"})
	if err := w.WriteRow([]any{1}); err == nil {
		t.Error("mismatched row accepted")
	}
}
