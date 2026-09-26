// Package pipeline — Run() orchestrator (spec §7.2).
package pipeline

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"gopipe/internal/exitcode"
	"gopipe/internal/logger"
	"gopipe/internal/pipeconfig"
	"gopipe/internal/progress"
	"gopipe/internal/summary"
	winconsole "gopipe/internal/windows"
)

// Run executes the full pipeline described by cfg.
// stdout/stderr are the output writers for consumer output and gopipe messages.
// Returns the process exit code (spec §F-09).
func Run(ctx context.Context, cfg pipeconfig.Config, stdout, stderr io.Writer) int {
	// Dry-run: print plan and exit.
	if cfg.DryRun {
		printDryRun(cfg, stdout)
		return exitcode.OK
	}

	// Set up logger.
	var log *logger.Logger
	if cfg.Log != "" {
		l, err := logger.New(cfg.Log, cfg.LogJSON, cfg.LogTimestamp)
		if err != nil {
			fmt.Fprintf(stderr, "gopipe: cannot open log %q: %v\n", cfg.Log, err)
			return exitcode.ConsumerFailed
		}
		defer l.Close()
		log = l
	} else {
		log = logger.Nop()
	}

	// Apply pipeline timeout.
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	// Tee writer wraps stdout.
	out, closeTee, err := NewTee(stdout, cfg.Tee)
	if err != nil {
		fmt.Fprintf(stderr, "gopipe: tee: %v\n", err)
		return exitcode.ConsumerFailed
	}
	defer closeTee()

	// Console title.
	if winconsole.IsConsole(os.Stdout) {
		title := fmt.Sprintf("gopipe — %s → %s", cfg.Producer[0], cfg.Consumers[0][0])
		winconsole.SetTitle(title)
		defer winconsole.RestoreTitle()
	}

	// Build summary report.
	report := &summary.Report{
		Producer:  strings.Join(cfg.Producer, " "),
		Mode:      modeLabel(cfg),
		StartTime: time.Now(),
		Binary:    cfg.Binary,
	}
	stats := make([]*summary.ConsumerStats, len(cfg.Consumers))
	for i, c := range cfg.Consumers {
		stats[i] = &summary.ConsumerStats{Label: strings.Join(c, " ")}
	}
	report.Consumers = stats

	// Progress indicator.
	prog := progress.New(out, winconsole.IsConsole(os.Stdout) && !cfg.NoProgress)
	prog.Start()
	defer prog.Stop()

	// Start producer.
	producerCmd := exec.CommandContext(ctx, cfg.Producer[0], cfg.Producer[1:]...)
	producerCmd.Stderr = stderr
	producerCmd.Dir = cfg.Workdir
	if len(cfg.Env) > 0 {
		producerCmd.Env = append(producerCmd.Environ(), cfg.Env...)
	}
	producerOut, err := producerCmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(stderr, "gopipe: producer stdout pipe: %v\n", err)
		return exitcode.ProducerFailed
	}
	if err := producerCmd.Start(); err != nil {
		fmt.Fprintf(stderr, "gopipe: producer start: %v\n", err)
		return exitcode.ProducerFailed
	}

	log.Log(logger.Entry{Level: logger.LevelInfo, Source: "producer",
		Message: fmt.Sprintf("started: %s", strings.Join(cfg.Producer, " "))})

	fail := &failFlag{}

	// Dispatch to the correct mode.
	if cfg.Binary {
		runBinary(ctx, cfg, producerOut, out, stderr, stats, log, fail)
	} else {
		switch cfg.Mode {
		case pipeconfig.ModeParallel:
			runParallel(ctx, cfg, producerOut, out, stderr, stats, log, fail)
		case pipeconfig.ModeStreaming:
			runStreaming(ctx, cfg, producerOut, out, stderr, stats, log, fail)
		case pipeconfig.ModeBatch:
			runBatch(ctx, cfg, producerOut, out, stderr, stats, log, fail)
		case pipeconfig.ModeFull:
			runFull(ctx, cfg, producerOut, out, stderr, stats, log, fail)
		default:
			runSequential(ctx, cfg, producerOut, out, stderr, stats, log, fail)
		}
	}

	// Wait for producer.
	producerFailed := false
	if err := producerCmd.Wait(); err != nil {
		producerFailed = true
		log.Log(logger.Entry{Level: logger.LevelError, Source: "producer",
			Message: fmt.Sprintf("exit: %v", err)})
	}

	prog.Stop()
	report.EndTime = time.Now()

	// Determine exit code.
	code := exitcode.Combine(producerFailed, fail.IsSet())
	if ctx.Err() == context.DeadlineExceeded {
		code = exitcode.Timeout
		fmt.Fprintln(stderr, "gopipe: pipeline timeout exceeded")
	} else if ctx.Err() == context.Canceled {
		code = exitcode.Cancelled
	}

	report.ExitCode = code
	if !cfg.NoSummary {
		report.Print(stdout)
	}

	return code
}

func modeLabel(cfg pipeconfig.Config) string {
	if cfg.Binary {
		return fmt.Sprintf("binary+%s", cfg.Mode)
	}
	if cfg.Mode == pipeconfig.ModeParallel {
		return fmt.Sprintf("parallel (workers: %d)", cfg.Workers)
	}
	return string(cfg.Mode)
}

func printDryRun(cfg pipeconfig.Config, w io.Writer) {
	fmt.Fprintln(w, "[DRY RUN] Pipeline plan:")
	fmt.Fprintf(w, "  Producer  : %s\n", strings.Join(cfg.Producer, " "))
	for i, c := range cfg.Consumers {
		fmt.Fprintf(w, "  Consumer %d: %s\n", i+1, strings.Join(c, " "))
	}
	fmt.Fprintf(w, "  Mode      : %s\n", modeLabel(cfg))
	if cfg.Timeout > 0 {
		fmt.Fprintf(w, "  Timeout   : %s\n", cfg.Timeout)
	}
	if cfg.Retry > 0 {
		fmt.Fprintf(w, "  Retry     : %d attempts, %s backoff\n", cfg.Retry, cfg.RetryWait)
	}
	if cfg.Log != "" {
		format := "plain"
		if cfg.LogJSON {
			format = "JSON"
		}
		fmt.Fprintf(w, "  Log       : %s (%s)\n", cfg.Log, format)
	}
	fmt.Fprintln(w, "[DRY RUN] No processes started.")
}
