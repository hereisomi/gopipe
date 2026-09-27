package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"gopipe/internal/connector"
	"gopipe/internal/dispatch"
)

func TestDBExtractCLIValidation(t *testing.T) {
	for _, args := range [][]string{
		{"db", "extract"},
		{"db", "extract", "--json", "absent.json"},
		{"db", "extract", "--yaml", "absent.yaml"},
		{"db", "extract", "--json", "a", "--yaml", "b"},
		{"db", "extract", "--other"},
		{"db", "--json", "absent.json"},
	} {
		if code := dispatch.Run(context.Background(), args); code != 1 {
			t.Errorf("%q: exit = %d", args, code)
		}
	}
	path := filepath.Join(t.TempDir(), "task.json")
	if err := os.WriteFile(path, []byte(`{"driver":"postgres"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if code := dispatch.Run(context.Background(), []string{"db", "extract", "--json", path}); code != 1 {
		t.Fatalf("invalid task exit = %d", code)
	}
	if code := dispatch.Run(context.Background(), []string{"db", "load", "--json", path}); code != 1 {
		t.Fatalf("db load stub exit = %d", code)
	}
}

// A database/sql fixture covers the CLI wiring without requiring a real DB.
type cliDriver struct{}

func (cliDriver) Open(string) (driver.Conn, error) { return cliConn{}, nil }

type cliConnector struct{}

func (cliConnector) Driver() driver.Driver                        { return cliDriver{} }
func (cliConnector) Connect(context.Context) (driver.Conn, error) { return cliConn{}, nil }

type cliConn struct{}

func (cliConn) Prepare(string) (driver.Stmt, error) { return nil, fmt.Errorf("unexpected prepare") }
func (cliConn) Close() error                        { return nil }
func (cliConn) Begin() (driver.Tx, error)           { return nil, fmt.Errorf("unexpected transaction") }
func (cliConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return &cliRows{}, nil
}

type cliRows struct{ n int }

func (cliRows) Columns() []string { return []string{"name", "value"} }
func (cliRows) Close() error      { return nil }
func (r *cliRows) Next(dest []driver.Value) error {
	if r.n > 0 {
		return io.EOF
	}
	r.n++
	dest[0] = []byte("alice")
	dest[1] = int64(7)
	return nil
}

type cliDB struct{ *sql.DB }

func (c cliDB) Query(ctx context.Context, q string, args []any) (*sql.Rows, error) {
	return c.DB.QueryContext(ctx, q, args...)
}

func TestDBExtractCLIOutput(t *testing.T) {
	prev := openDB
	openDB = func(context.Context, string, string) (connector.Connector, error) {
		return cliDB{sql.OpenDB(cliConnector{})}, nil
	}
	t.Cleanup(func() { openDB = prev })
	for _, format := range []string{"json", "yaml"} {
		t.Run(format, func(t *testing.T) {
			dir := t.TempDir()
			out := filepath.Join(dir, "out.csv")
			task := filepath.Join(dir, "task."+format)
			var contents string
			if format == "json" {
				contents = fmt.Sprintf(`{"dsn":"fake","driver":"postgres","sql":"select 1","output":%q,"format":"csv"}`, out)
			} else {
				contents = fmt.Sprintf("dsn: fake\ndriver: postgres\nsql: select 1\noutput: %s\nformat: csv\n", out)
			}
			if err := os.WriteFile(task, []byte(contents), 0600); err != nil {
				t.Fatal(err)
			}
			if code := dispatch.Run(context.Background(), []string{"db", "extract", "--" + format, task}); code != 0 {
				t.Fatalf("exit = %d", code)
			}
			data, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "name,value\nalice,7\n" {
				t.Fatalf("output = %q", data)
			}
		})
	}
}
