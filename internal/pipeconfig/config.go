// Package pipeconfig defines the pipeline configuration model (spec §6.2, §F-14).
package pipeconfig

import "time"

// Mode is the pipeline execution strategy.
type Mode string

const (
	ModeSequential Mode = "sequential"
	ModeParallel   Mode = "parallel"
	ModeStreaming  Mode = "streaming"
	ModeBatch      Mode = "batch"
	ModeFull       Mode = "full"
)

// FullVia controls how full-collect mode delivers data to consumers.
type FullVia string

const (
	FullViaStdin   FullVia = "stdin"
	FullViaTempfile FullVia = "tempfile"
)

// Config is the resolved pipeline configuration after merging defaults,
// config file, and CLI flags (spec §7.3 contract: CLI flags always win).
type Config struct {
	// Producer is the command + args for the producer process.
	Producer []string
	// Consumers is the list of consumer commands (each is a []string).
	Consumers [][]string

	Mode      Mode
	Workers   int
	BatchSize int
	FullVia   FullVia
	Binary    bool
	ChunkSize int
	Chain     bool

	Timeout      time.Duration
	Retry        int
	RetryWait    time.Duration
	ShutdownWait time.Duration

	Log          string
	LogTimestamp bool
	LogJSON      bool
	Tee          string

	Env     []string
	Workdir string

	DryRun     bool
	NoSummary  bool
	NoProgress bool
	NoColor    bool
}

// Default returns a Config populated with spec-defined defaults.
func Default() Config {
	return Config{
		Mode:         ModeSequential,
		Workers:      4,
		BatchSize:    10,
		FullVia:      FullViaStdin,
		ChunkSize:    4096,
		RetryWait:    500 * time.Millisecond,
		ShutdownWait: 5 * time.Second,
		LogTimestamp: true,
	}
}
