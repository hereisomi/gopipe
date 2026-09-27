package dbextract

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopipe/internal/dbtask"
)

// In-process database/sql driver exercises the complete streaming path with
// no database server or credentials. A second row can fail mid-stream.
type fixtureDriver struct{}

func (fixtureDriver) Open(string) (driver.Conn, error) { return fixtureConn{}, nil }

type fixtureConn struct{}

func (fixtureConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not used") }
func (fixtureConn) Close() error                        { return nil }
func (fixtureConn) Begin() (driver.Tx, error)           { return nil, errors.New("not used") }
func (fixtureConn) QueryContext(_ context.Context, q string, _ []driver.NamedValue) (driver.Rows, error) {
	return &fixtureRows{fail: q == "fail"}, nil
}

type fixtureRows struct {
	index int
	fail  bool
}

func (r *fixtureRows) Columns() []string { return []string{"id", "name", "note"} }
func (r *fixtureRows) Close() error      { return nil }
func (r *fixtureRows) Next(dest []driver.Value) error {
	if r.index == 1 && r.fail {
		return errors.New("injected row error")
	}
	if r.index >= 2 {
		return io.EOF
	}
	dest[0] = int64(r.index + 1)
	dest[1] = []byte("a,b\nline")
	dest[2] = nil
	r.index++
	return nil
}

type sqlTestConn struct{ *sql.DB }

func (c sqlTestConn) Query(ctx context.Context, q string, args []any) (*sql.Rows, error) {
	return c.DB.QueryContext(ctx, q, args...)
}

func TestExecuteAllFormats(t *testing.T) {
	db := sql.OpenDB(fixtureConnector{})
	defer db.Close()
	tests := map[string]string{
		"csv":   "id,name,note\n1,\"a,b\nline\",\n2,\"a,b\nline\",\n",
		"tsv":   "id\tname\tnote\n1\t\"a,b\nline\"\t\n2\t\"a,b\nline\"\t\n",
		"jsonl": "\"id\":1",
		"json":  "[",
	}
	for format, want := range tests {
		t.Run(format, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "output."+format)
			task := dbtask.ExtractTask{DSN: "fixture", Driver: "postgres", SQL: "select", Output: path, Format: format}
			if err := Execute(context.Background(), task, sqlTestConn{db}); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), want) {
				t.Errorf("got %q, want %q", data, want)
			}
		})
	}
}
func TestExecuteFailurePreservesExistingOutput(t *testing.T) {
	db := sql.OpenDB(fixtureConnector{})
	defer db.Close()
	path := filepath.Join(t.TempDir(), "out.csv")
	if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	task := dbtask.ExtractTask{DSN: "fixture", Driver: "postgres", SQL: "fail", Output: path, Format: "csv"}
	err := Execute(context.Background(), task, sqlTestConn{db})
	if err == nil || !strings.Contains(err.Error(), "injected row error") {
		t.Fatalf("error = %v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "old" {
		t.Errorf("partial file replaced old data: %q", data)
	}
	files, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".out.csv-*.tmp"))
	if len(files) != 0 {
		t.Errorf("temporary files: %v", files)
	}
}

// Implements driver.Connector for sql.OpenDB without registering a global name.
type fixtureConnector struct{}

func (fixtureConnector) Connect(context.Context) (driver.Conn, error) { return fixtureConn{}, nil }
func (fixtureConnector) Driver() driver.Driver                        { return fixtureDriver{} }
