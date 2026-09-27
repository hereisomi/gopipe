// Package pipeline — streaming mode: long-lived consumers, fan-out via stdin (spec §F-03).
package pipeline

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"gopipe/internal/logger"
	"gopipe/internal/pipeconfig"
	"gopipe/internal/summary"
)

func runStreaming(ctx context.Context, cfg pipeconfig.Config, producerOut io.Reader,
	stdout, stderr io.Writer, stats []*summary.ConsumerStats, log *logger.Logger, fail *failFlag) {

	type consumer struct {
		cmd   *exec.Cmd
		stdin io.WriteCloser
		once  sync.Once
	}

	consumers := make([]*consumer, len(cfg.Consumers))
	var wg sync.WaitGroup

	for i, cmdArgs := range cfg.Consumers {
		cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		cmd.Dir = cfg.Workdir
		if len(cfg.Env) > 0 {
			cmd.Env = append(cmd.Environ(), cfg.Env...)
		}
		stdin, err := cmd.StdinPipe()
		if err != nil {
			log.Log(logger.Entry{Level: logger.LevelError, Source: fmt.Sprintf("consumer[%d]", i+1),
				Message: fmt.Sprintf("stdin pipe: %v", err), ConsumerIndex: i + 1})
			fail.Set()
			continue
		}
		if err := cmd.Start(); err != nil {
			log.Log(logger.Entry{Level: logger.LevelError, Source: fmt.Sprintf("consumer[%d]", i+1),
				Message: fmt.Sprintf("start: %v", err), ConsumerIndex: i + 1})
			fail.Set()
			continue
		}
		consumers[i] = &consumer{cmd: cmd, stdin: stdin}

		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			err := cmd.Wait()
			code := 0
			if err != nil {
				if cmd.ProcessState != nil {
					code = cmd.ProcessState.ExitCode()
				} else {
					code = 1
				}
			}
			stats[idx].RecordRun(code != 0, false, false)
			if code != 0 {
				fail.Set()
			}
		}()
	}

	// Fan-out each producer line to all consumer stdins.
	scanner := bufio.NewScanner(producerOut)
	for scanner.Scan() {
		if ctx.Err() != nil {
			break
		}
		line := append(scanner.Bytes(), '\n')
		for _, c := range consumers {
			if c == nil {
				continue
			}
			_, _ = c.stdin.Write(line)
		}
	}

	// Close all consumer stdins exactly once (signals EOF).
	for _, c := range consumers {
		if c == nil {
			continue
		}
		c.once.Do(func() { _ = c.stdin.Close() })
	}

	wg.Wait()
}
