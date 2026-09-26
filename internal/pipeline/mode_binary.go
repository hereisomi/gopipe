// Package pipeline — binary stream mode (spec §F-06).
// Bypasses line scanning; treats producer stdout as a raw byte stream.
package pipeline

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"gopipe/internal/logger"
	"gopipe/internal/pipeconfig"
	"gopipe/internal/summary"
)

func runBinary(ctx context.Context, cfg pipeconfig.Config, producerOut io.Reader,
	stdout, stderr io.Writer, stats []*summary.ConsumerStats, log *logger.Logger, fail *failFlag) {

	switch cfg.Mode {
	case pipeconfig.ModeBatch:
		runBinaryBatch(ctx, cfg, producerOut, stdout, stderr, stats, log, fail)
	case pipeconfig.ModeFull:
		runFull(ctx, cfg, producerOut, stdout, stderr, stats, log, fail)
	default:
		// sequential/parallel fall back to binary streaming per spec §F-06.
		if cfg.Mode == pipeconfig.ModeSequential || cfg.Mode == pipeconfig.ModeParallel {
			log.Log(logger.Entry{Level: logger.LevelInfo, Source: "pipeline",
				Message: fmt.Sprintf("binary: mode %q not meaningful, falling back to streaming", cfg.Mode)})
		}
		runBinaryStreaming(ctx, cfg, producerOut, stdout, stderr, stats, log, fail)
	}
}

// runBinaryStreaming fans out raw bytes to all consumer stdins via io.MultiWriter.
func runBinaryStreaming(ctx context.Context, cfg pipeconfig.Config, producerOut io.Reader,
	stdout, stderr io.Writer, stats []*summary.ConsumerStats, log *logger.Logger, fail *failFlag) {

	type entry struct {
		cmd   *exec.Cmd
		stdin io.WriteCloser
		once  sync.Once
	}

	entries := make([]*entry, len(cfg.Consumers))
	writers := make([]io.Writer, 0, len(cfg.Consumers))
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
			fail.Set()
			continue
		}
		if err := cmd.Start(); err != nil {
			fail.Set()
			continue
		}
		entries[i] = &entry{cmd: cmd, stdin: stdin}
		writers = append(writers, stdin)

		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			err := cmd.Wait()
			code := 0
			if err != nil && cmd.ProcessState != nil {
				code = cmd.ProcessState.ExitCode()
			}
			stats[idx].RecordRun(code != 0, false, false)
			if code != 0 {
				fail.Set()
			}
		}()
	}

	mw := io.MultiWriter(writers...)
	_, _ = io.Copy(mw, producerOut)

	for _, e := range entries {
		if e == nil {
			continue
		}
		e.once.Do(func() { _ = e.stdin.Close() })
	}
	wg.Wait()
}

// runBinaryBatch reads fixed-size chunks and delivers each to consumer stdin.
func runBinaryBatch(ctx context.Context, cfg pipeconfig.Config, producerOut io.Reader,
	stdout, stderr io.Writer, stats []*summary.ConsumerStats, log *logger.Logger, fail *failFlag) {

	chunkSize := cfg.ChunkSize
	if chunkSize < 1 {
		chunkSize = 4096
	}
	buf := make([]byte, chunkSize)
	for {
		if ctx.Err() != nil {
			return
		}
		n, err := producerOut.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			for i, consumer := range cfg.Consumers {
				res := RunConsumer(ctx, consumer, byteReader(chunk), stdout, stderr,
					cfg.Workdir, cfg.Env, cfg.Retry, cfg.RetryWait, stats[i], log, i)
				if res.ExitCode != 0 {
					fail.Set()
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Log(logger.Entry{Level: logger.LevelError, Source: "producer", Message: fmt.Sprintf("read: %v", err)})
			fail.Set()
			break
		}
	}
}

type byteReader []byte

func (b byteReader) Read(p []byte) (int, error) {
	if len(b) == 0 {
		return 0, io.EOF
	}
	n := copy(p, b)
	return n, io.EOF
}
