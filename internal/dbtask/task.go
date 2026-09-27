// Package dbtask decodes and validates database extract task files.
package dbtask

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ExtractTask is a single SQL query and its output destination.
type ExtractTask struct {
	DSN       string `json:"dsn" yaml:"dsn"`
	Driver    string `json:"driver" yaml:"driver"`
	SQL       string `json:"sql" yaml:"sql"`
	Output    string `json:"output" yaml:"output"`
	Format    string `json:"format" yaml:"format"`
	BatchSize int    `json:"batch_size" yaml:"batch_size"`
}

// Load reads one task, or a batch with a "tasks" list, from a JSON/YAML file.
// Strict JSON decoding prevents a misspelled configuration key from silently
// producing an empty query or incorrect destination.
func Load(path, format string) ([]ExtractTask, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read task file %q: %w", path, err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("empty task file %q", path)
	}
	var tasks []ExtractTask
	if format == "json" {
		if bytes.TrimSpace(data)[0] == '[' {
			err = decodeJSON(data, &tasks)
		} else {
			var probe map[string]json.RawMessage
			if err = json.Unmarshal(data, &probe); err == nil {
				if _, ok := probe["tasks"]; ok {
					var batch BatchTask
					err = decodeJSON(data, &batch)
					tasks = batch.Tasks
				} else {
					var t ExtractTask
					err = decodeJSON(data, &t)
					tasks = []ExtractTask{t}
				}
			}
		}
	} else if format == "yaml" {
		if bytes.TrimSpace(data)[0] == '-' {
			err = decodeYAML(data, &tasks)
		} else {
			var probe map[string]any
			if err = yaml.Unmarshal(data, &probe); err == nil {
				if _, ok := probe["tasks"]; ok {
					var batch BatchTask
					err = decodeYAML(data, &batch)
					tasks = batch.Tasks
				} else {
					var t ExtractTask
					err = decodeYAML(data, &t)
					tasks = []ExtractTask{t}
				}
			}
		}
	} else {
		return nil, fmt.Errorf("unsupported task file format %q", format)
	}
	if err != nil {
		return nil, fmt.Errorf("parse task file %q: %w", path, err)
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("task file %q contains no tasks", path)
	}
	for i := range tasks {
		if err = tasks[i].Validate(); err != nil {
			return nil, fmt.Errorf("task %d: %w", i+1, err)
		}
		if tasks[i].SQL, err = Substitute(tasks[i].SQL, os.LookupEnv); err != nil {
			return nil, fmt.Errorf("task %d: %w", i+1, err)
		}
	}
	return tasks, nil
}

func decodeJSON(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		if err != nil {
			return err
		}
		return fmt.Errorf("unexpected content after task")
	}
	return nil
}

func decodeYAML(data []byte, v any) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(v); err != nil {
		return err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		if err != nil {
			return err
		}
		return fmt.Errorf("unexpected content after task")
	}
	return nil
}

// Validate checks required fields and normalizes optional defaults.
func (t *ExtractTask) Validate() error {
	if strings.TrimSpace(t.Driver) == "" {
		return fmt.Errorf("driver is required")
	}
	if strings.TrimSpace(t.DSN) == "" {
		return fmt.Errorf("dsn is required")
	}
	if strings.TrimSpace(t.SQL) == "" {
		return fmt.Errorf("sql is required")
	}
	if strings.TrimSpace(t.Output) == "" {
		return fmt.Errorf("output is required")
	}
	if t.Format == "" {
		t.Format = "csv"
	}
	t.Format = strings.ToLower(t.Format)
	switch t.Format {
	case "csv", "tsv", "json", "jsonl":
	default:
		return fmt.Errorf("unsupported output format %q (expected csv, tsv, json, or jsonl)", t.Format)
	}
	if t.BatchSize < 0 {
		return fmt.Errorf("batch_size must not be negative")
	}
	if t.BatchSize == 0 {
		t.BatchSize = 1000
	}
	return nil
}
