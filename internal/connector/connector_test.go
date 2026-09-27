package connector

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

func TestDriverRegistrations(t *testing.T) {
	registered := make(map[string]bool)
	for _, name := range sql.Drivers() {
		registered[name] = true
	}
	for _, name := range []string{sybaseDriver, oracleDriver, mssqlDriver, postgresDriver, gaussdbDriver} {
		if !registered[name] {
			t.Errorf("driver %q not registered", name)
		}
	}
}

func TestOpenRejectsInvalidConfiguration(t *testing.T) {
	for _, tc := range []struct{ driver, dsn, want string }{
		{"invalid", "fake", "unsupported database driver"},
		{"postgres", "", "DSN is required"},
	} {
		conn, err := Open(context.Background(), tc.driver, tc.dsn)
		if conn != nil || err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("Open(%q, %q) = %v, %v; want %q", tc.driver, tc.dsn, conn, err, tc.want)
		}
	}
}
