package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"gopipe/internal/connector"
	"gopipe/internal/dbextract"
	"gopipe/internal/dbtask"
	"gopipe/internal/dispatch"
)

func init() { dispatch.UpdateRun("db", dbExtractHandler) }

// Kept as a function variable to allow a fake connector in command tests.
var openDB = connector.Open

func dbExtractHandler(ctx context.Context, args []string) int {
	_, stderr := dispatch.Output(ctx)
	if err := runDBExtract(ctx, args, stderr); err != nil {
		fmt.Fprintf(stderr, "gopipe db extract: %v\n", err)
		return 1
	}
	return 0
}

func runDBExtract(ctx context.Context, args []string, stderr io.Writer) error {
	flags := flag.NewFlagSet("db extract", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonPath := flags.String("json", "", "JSON task file")
	yamlPath := flags.String("yaml", "", "YAML task file")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}
	if (*jsonPath == "") == (*yamlPath == "") {
		return fmt.Errorf("provide exactly one of --json <task.json> or --yaml <task.yaml>")
	}
	path, format := *jsonPath, "json"
	if *yamlPath != "" {
		path, format = *yamlPath, "yaml"
	}
	tasks, err := dbtask.Load(path, format)
	if err != nil {
		return err
	}
	for i, task := range tasks {
		conn, err := openDB(ctx, task.Driver, task.DSN)
		if err != nil {
			return fmt.Errorf("task %d: %w", i+1, err)
		}
		runErr := dbextract.Execute(ctx, task, conn)
		closeErr := conn.Close()
		if runErr != nil {
			return fmt.Errorf("task %d: %w", i+1, runErr)
		}
		if closeErr != nil {
			return fmt.Errorf("task %d: close connector: %w", i+1, closeErr)
		}
	}
	return nil
}
