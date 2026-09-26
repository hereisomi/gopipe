// Package pipeconfig — YAML/JSON config file load and save (spec §F-14).
package pipeconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// fileConfig is the serializable form of Config (uses string durations for YAML).
type fileConfig struct {
	Producer     []string   `yaml:"producer"      json:"producer"`
	Consumers    [][]string `yaml:"consumers"     json:"consumers"`
	Mode         string     `yaml:"mode"          json:"mode"`
	Workers      int        `yaml:"workers"       json:"workers"`
	BatchSize    int        `yaml:"batch_size"    json:"batch_size"`
	FullVia      string     `yaml:"full_via"      json:"full_via"`
	Binary       bool       `yaml:"binary"        json:"binary"`
	ChunkSize    int        `yaml:"chunk_size"    json:"chunk_size"`
	Chain        bool       `yaml:"chain"         json:"chain"`
	Timeout      string     `yaml:"timeout"       json:"timeout"`
	Retry        int        `yaml:"retry"         json:"retry"`
	RetryWait    string     `yaml:"retry_wait"    json:"retry_wait"`
	ShutdownWait string     `yaml:"shutdown_wait" json:"shutdown_wait"`
	Log          string     `yaml:"log"           json:"log"`
	LogTimestamp bool       `yaml:"log_timestamp" json:"log_timestamp"`
	LogJSON      bool       `yaml:"log_json"      json:"log_json"`
	Tee          string     `yaml:"tee"           json:"tee"`
	Env          []string   `yaml:"env"           json:"env"`
	Workdir      string     `yaml:"workdir"       json:"workdir"`
}

// LoadFile reads a YAML or JSON config file and returns a Config.
// The caller is responsible for applying CLI flag overrides afterward.
func LoadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var fc fileConfig
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".json" {
		err = json.Unmarshal(data, &fc)
	} else {
		err = yaml.Unmarshal(data, &fc)
	}
	if err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return fromFileConfig(fc)
}

// SaveFile writes cfg to path as YAML or JSON (determined by extension).
func SaveFile(path string, cfg Config) error {
	fc := toFileConfig(cfg)
	var data []byte
	var err error
	if strings.ToLower(filepath.Ext(path)) == ".json" {
		data, err = json.MarshalIndent(fc, "", "  ")
	} else {
		data, err = yaml.Marshal(fc)
	}
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func fromFileConfig(fc fileConfig) (Config, error) {
	cfg := Default()
	if fc.Mode != "" {
		cfg.Mode = Mode(fc.Mode)
	}
	if fc.Workers > 0 {
		cfg.Workers = fc.Workers
	}
	if fc.BatchSize > 0 {
		cfg.BatchSize = fc.BatchSize
	}
	if fc.FullVia != "" {
		cfg.FullVia = FullVia(fc.FullVia)
	}
	cfg.Binary = fc.Binary
	if fc.ChunkSize > 0 {
		cfg.ChunkSize = fc.ChunkSize
	}
	cfg.Chain = fc.Chain
	cfg.Producer = fc.Producer
	cfg.Consumers = fc.Consumers
	cfg.Log = fc.Log
	cfg.LogTimestamp = fc.LogTimestamp
	cfg.LogJSON = fc.LogJSON
	cfg.Tee = fc.Tee
	cfg.Env = fc.Env
	cfg.Workdir = fc.Workdir
	cfg.Retry = fc.Retry

	if d, err := parseDuration(fc.Timeout); err == nil {
		cfg.Timeout = d
	}
	if d, err := parseDuration(fc.RetryWait); err == nil && d > 0 {
		cfg.RetryWait = d
	}
	if d, err := parseDuration(fc.ShutdownWait); err == nil && d > 0 {
		cfg.ShutdownWait = d
	}
	return cfg, nil
}

func toFileConfig(cfg Config) fileConfig {
	return fileConfig{
		Producer:     cfg.Producer,
		Consumers:    cfg.Consumers,
		Mode:         string(cfg.Mode),
		Workers:      cfg.Workers,
		BatchSize:    cfg.BatchSize,
		FullVia:      string(cfg.FullVia),
		Binary:       cfg.Binary,
		ChunkSize:    cfg.ChunkSize,
		Chain:        cfg.Chain,
		Timeout:      durationString(cfg.Timeout),
		Retry:        cfg.Retry,
		RetryWait:    durationString(cfg.RetryWait),
		ShutdownWait: durationString(cfg.ShutdownWait),
		Log:          cfg.Log,
		LogTimestamp: cfg.LogTimestamp,
		LogJSON:      cfg.LogJSON,
		Tee:          cfg.Tee,
		Env:          cfg.Env,
		Workdir:      cfg.Workdir,
	}
}

func parseDuration(s string) (time.Duration, error) {
	if s == "" || s == "0s" {
		return 0, nil
	}
	return time.ParseDuration(s)
}
