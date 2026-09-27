// Package pipeline — consumer runner with retry and exponential backoff (spec §F-07).
package pipeline

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync/atomic"
	"time"

	"gopipe/internal/logger"
	"gopipe/internal/summary"
)

// RunResult is the outcome of one consumer invocation (possibly with retries).
type RunResult struct {
	ExitCode  int
	Retried   bool
	Recovered bool // retried and ultimately succeeded
}

// RunConsumer executes one consumer command, feeding input via stdin.
// Retries up to cfg.Retry times with exponential backoff on non-zero exit.
// Returns the final RunResult and records stats into stats.
func RunConsumer(
	ctx context.Context,
	cmdArgs []string,
	stdin io.Reader,
	stdout, stderr io.Writer,
	workdir string,
	env []string,
	maxRetry int,
	retryWait time.Duration,
	stats *summary.ConsumerStats,
	log *logger.Logger,
	consumerIdx int,
) RunResult {
	source := fmt.Sprintf("consumer[%d]", consumerIdx+1)
	var retried bool
	var lastCode int

	for attempt := 0; attempt <= maxRetry; attempt++ {
		if attempt > 0 {
			retried = true
			wait := retryWait * (1 << (attempt - 1)) // exponential backoff
			log.Log(logger.Entry{
				Level: logger.LevelRetry, Source: source,
				Message: fmt.Sprintf("retry %d/%d after %s", attempt, maxRetry, wait),
				ConsumerIndex: consumerIdx + 1, Attempt: attempt,
			})
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				stats.RecordRun(true, retried, false)
				return RunResult{ExitCode: 1, Retried: retried}
			}
		}

		code := runOnce(ctx, cmdArgs, stdin, stdout, stderr, workdir, env)
		lastCode = code

		if code == 0 {
			recovered := retried
			stats.RecordRun(false, retried, recovered)
			log.Log(logger.Entry{
				Level: logger.LevelInfo, Source: source,
				Message: "ok", ConsumerIndex: consumerIdx + 1, Attempt: attempt + 1, ExitCode: 0,
			})
			return RunResult{ExitCode: 0, Retried: retried, Recovered: recovered}
		}

		log.Log(logger.Entry{
			Level: logger.LevelError, Source: source,
			Message: fmt.Sprintf("exit %d", code),
			ConsumerIndex: consumerIdx + 1, Attempt: attempt + 1, ExitCode: code,
		})
	}

	stats.RecordRun(true, retried, false)
	return RunResult{ExitCode: lastCode, Retried: retried}
}

func runOnce(ctx context.Context, cmdArgs []string, stdin io.Reader, stdout, stderr io.Writer, workdir string, env []string) int {
	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Dir = workdir
	if len(env) > 0 {
		cmd.Env = append(cmd.Environ(), env...)
	}
	if err := cmd.Run(); err != nil {
		if cmd.ProcessState != nil {
			return cmd.ProcessState.ExitCode()
		}
		return 1
	}
	return 0
}

// consumerFailed is a shared atomic flag set by any goroutine that sees a failure.
type failFlag struct{ v atomic.Bool }

func (f *failFlag) Set()      { f.v.Store(true) }
func (f *failFlag) IsSet() bool { return f.v.Load() }
