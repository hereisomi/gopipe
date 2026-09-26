// Package pipeconfig — flag parsing for the pipeline subcommand (spec §6.2).
package pipeconfig

import (
	"fmt"
	"strings"
	"time"

	flag "github.com/spf13/pflag"
)

// ParseArgs parses os.Args-style args for the pipeline subcommand.
// Producer and consumers are separated by -r flags:
//
//	<producer...> -r <consumer1...> [-r <consumer2...>] [-- <flags>]
//
// Returns the resolved Config (defaults → file → flags).
func ParseArgs(args []string) (Config, error) {
	cfg := Default()

	// Split args into pipeline args and flag args at "--".
	pipeArgs, flagArgs := splitAtDashDash(args)

	// Parse -r separators to extract producer + consumers from pipeArgs.
	producer, consumers, err := splitPipelineArgs(pipeArgs)
	if err != nil {
		return cfg, err
	}
	cfg.Producer = producer
	cfg.Consumers = consumers

	fs := buildFlagSet(&cfg)
	if err := fs.Parse(flagArgs); err != nil {
		return cfg, err
	}

	// --config file overrides defaults but is overridden by explicit CLI flags.
	var configPath string
	if f := fs.Lookup("config"); f != nil {
		configPath = f.Value.String()
	}
	if configPath != "" {
		fileCfg, err := LoadFile(configPath)
		if err != nil {
			return cfg, fmt.Errorf("--config: %w", err)
		}
		mergeFile(&cfg, fileCfg, fs)
	}

	return cfg, nil
}

func buildFlagSet(cfg *Config) *flag.FlagSet {
	fs := flag.NewFlagSet("pipeline", flag.ContinueOnError)
	fs.StringVar((*string)(&cfg.Mode), "mode", string(cfg.Mode), "Execution mode: sequential|parallel|streaming|batch|full")
	fs.IntVar(&cfg.Workers, "workers", cfg.Workers, "Worker pool size for parallel mode")
	fs.IntVar(&cfg.BatchSize, "batch-size", cfg.BatchSize, "Lines per batch for batch mode")
	fs.StringVar((*string)(&cfg.FullVia), "full-via", string(cfg.FullVia), "Full mode delivery: stdin|tempfile")
	fs.BoolVar(&cfg.Binary, "binary", cfg.Binary, "Binary stream mode")
	fs.IntVar(&cfg.ChunkSize, "chunk-size", cfg.ChunkSize, "Bytes per chunk in binary+batch mode")
	fs.BoolVar(&cfg.Chain, "chain", cfg.Chain, "Chain consumers sequentially")
	fs.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "Pipeline timeout (0 = none)")
	fs.IntVar(&cfg.Retry, "retry", cfg.Retry, "Consumer retry attempts on failure")
	fs.DurationVar(&cfg.RetryWait, "retry-wait", cfg.RetryWait, "Base backoff between retries")
	fs.DurationVar(&cfg.ShutdownWait, "shutdown-wait", cfg.ShutdownWait, "Graceful shutdown drain window")
	fs.StringVar(&cfg.Log, "log", cfg.Log, "Log file path")
	fs.BoolVar(&cfg.LogTimestamp, "log-timestamp", cfg.LogTimestamp, "Add timestamps to log entries")
	fs.BoolVar(&cfg.LogJSON, "log-json", cfg.LogJSON, "Write logs in JSON format")
	fs.StringVar(&cfg.Tee, "tee", cfg.Tee, "Write copy of producer output to file")
	fs.StringArrayVar(&cfg.Env, "env", cfg.Env, "Inject environment variables (repeatable)")
	fs.StringVar(&cfg.Workdir, "workdir", cfg.Workdir, "Working directory for all processes")
	fs.String("config", "", "Load configuration from YAML/JSON file")
	fs.String("save-config", "", "Save resolved configuration to file")
	fs.BoolVar(&cfg.DryRun, "dry-run", cfg.DryRun, "Print execution plan, do not run")
	fs.BoolVar(&cfg.NoSummary, "no-summary", cfg.NoSummary, "Suppress execution summary")
	fs.BoolVar(&cfg.NoProgress, "no-progress", cfg.NoProgress, "Suppress live progress indicator")
	fs.BoolVar(&cfg.NoColor, "no-color", cfg.NoColor, "Disable colored output")
	return fs
}

// splitAtDashDash splits args into [before --] and [after --].
func splitAtDashDash(args []string) (pipe, flags []string) {
	for i, a := range args {
		if a == "--" {
			return args[:i], args[i+1:]
		}
	}
	// No "--": flags are anything starting with "-", pipeline args are the rest.
	for i, a := range args {
		if strings.HasPrefix(a, "-") {
			return args[:i], args[i:]
		}
	}
	return args, nil
}

// splitPipelineArgs splits on -r to extract producer and consumers.
func splitPipelineArgs(args []string) (producer []string, consumers [][]string, err error) {
	segments := [][]string{}
	cur := []string{}
	for _, a := range args {
		if a == "-r" {
			segments = append(segments, cur)
			cur = []string{}
		} else {
			cur = append(cur, a)
		}
	}
	segments = append(segments, cur)

	if len(segments) == 0 || len(segments[0]) == 0 {
		return nil, nil, fmt.Errorf("no producer specified — usage: gopipe <producer...> -r <consumer...>")
	}
	producer = segments[0]
	for _, seg := range segments[1:] {
		if len(seg) == 0 {
			return nil, nil, fmt.Errorf("empty consumer after -r")
		}
		consumers = append(consumers, seg)
	}
	if len(consumers) == 0 {
		return nil, nil, fmt.Errorf("no consumers specified — usage: gopipe <producer...> -r <consumer...>")
	}
	return producer, consumers, nil
}

// mergeFile applies fileCfg values only for flags not explicitly set on the CLI.
func mergeFile(cfg *Config, fileCfg Config, fs *flag.FlagSet) {
	apply := func(name string, fn func()) {
		if f := fs.Lookup(name); f == nil || !f.Changed {
			fn()
		}
	}
	apply("mode", func() { cfg.Mode = fileCfg.Mode })
	apply("workers", func() { cfg.Workers = fileCfg.Workers })
	apply("batch-size", func() { cfg.BatchSize = fileCfg.BatchSize })
	apply("full-via", func() { cfg.FullVia = fileCfg.FullVia })
	apply("binary", func() { cfg.Binary = fileCfg.Binary })
	apply("chunk-size", func() { cfg.ChunkSize = fileCfg.ChunkSize })
	apply("timeout", func() { cfg.Timeout = fileCfg.Timeout })
	apply("retry", func() { cfg.Retry = fileCfg.Retry })
	apply("retry-wait", func() { cfg.RetryWait = fileCfg.RetryWait })
	apply("log", func() { cfg.Log = fileCfg.Log })
	apply("log-json", func() { cfg.LogJSON = fileCfg.LogJSON })
	apply("tee", func() { cfg.Tee = fileCfg.Tee })
	apply("workdir", func() { cfg.Workdir = fileCfg.Workdir })
}

// durationString formats a time.Duration for YAML serialization.
func durationString(d time.Duration) string { return d.String() }
