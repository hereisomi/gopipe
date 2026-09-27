// Package connector provides database/sql adapters for extract tasks.
package connector

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Connector streams query results and releases the underlying connection pool.
type Connector interface {
	Query(ctx context.Context, query string, args []any) (*sql.Rows, error)
	Close() error
}

type sqlConnector struct{ db *sql.DB }

func (c *sqlConnector) Query(ctx context.Context, query string, args []any) (*sql.Rows, error) {
	return c.db.QueryContext(ctx, query, args...)
}
func (c *sqlConnector) Close() error { return c.db.Close() }

// Open selects a registered driver and verifies that the DSN is reachable.
// gaussdb speaks the PostgreSQL wire protocol and shares the pgx driver.
func Open(ctx context.Context, driver, dsn string) (Connector, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("DSN is required")
	}
	var name string
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "sybase":
		name = sybaseDriver
	case "oracle":
		name = oracleDriver
	case "mssql", "sqlserver":
		name = mssqlDriver
	case "postgres", "postgresql":
		name = postgresDriver
	case "gaussdb":
		name = gaussdbDriver
	default:
		return nil, fmt.Errorf("unsupported database driver %q (expected sybase, oracle, mssql, postgres, or gaussdb)", driver)
	}
	db, err := sql.Open(name, dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", driver, err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to %s: %w", driver, err)
	}
	return &sqlConnector{db: db}, nil
}
