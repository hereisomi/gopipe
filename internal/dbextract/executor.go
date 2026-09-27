// Package dbextract streams SQL query results into a selected output format.
package dbextract

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"gopipe/internal/connector"
	"gopipe/internal/dbextract/writer"
	"gopipe/internal/dbtask"
)

// Execute runs a query and streams its rows to the output file. The file is
// staged in the destination directory and only replaces an existing output on
// success; partial results are removed on any query, scan, or write error.
// The caller owns and must close the connector.
func Execute(ctx context.Context, task dbtask.ExtractTask, conn connector.Connector) error {
	if err := task.Validate(); err != nil {
		return err
	}
	rows, err := conn.Query(ctx, task.SQL, nil)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("query columns: %w", err)
	}
	if len(columns) == 0 {
		return fmt.Errorf("query returned no columns")
	}
	dir := filepath.Dir(task.Output)
	file, err := os.CreateTemp(dir, "."+filepath.Base(task.Output)+"-*.tmp")
	if err != nil {
		return fmt.Errorf("create output %q: %w", task.Output, err)
	}
	defer func() { file.Close(); os.Remove(file.Name()) }()
	w, err := writer.New(task.Format, file)
	if err != nil {
		return err
	}
	if err = w.WriteHeader(columns); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	if err = stream(ctx, rows, columns, w); err != nil {
		return err
	}
	if err = rows.Err(); err != nil {
		return fmt.Errorf("read rows: %w", err)
	}
	if err = rows.Close(); err != nil {
		return fmt.Errorf("close rows: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("finish output: %w", err)
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("close output: %w", err)
	}
	if err = os.Rename(file.Name(), task.Output); err != nil {
		return fmt.Errorf("publish output %q: %w", task.Output, err)
	}
	return nil
}

func stream(ctx context.Context, rows *sql.Rows, columns []string, w writer.Writer) error {
	values := make([]any, len(columns))
	dest := make([]any, len(columns))
	for i := range dest {
		dest[i] = &values[i]
	}
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := rows.Scan(dest...); err != nil {
			return fmt.Errorf("scan row: %w", err)
		}
		if err := w.WriteRow(values); err != nil {
			return fmt.Errorf("write row: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read rows: %w", err)
	}
	return ctx.Err()
}
