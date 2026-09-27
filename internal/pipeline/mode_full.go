// Package pipeline — full-collect mode: stdin and tempfile delivery (spec §F-05).
package pipeline

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"gopipe/internal/logger"
	"gopipe/internal/pipeconfig"
	"gopipe/internal/summary"
)

func runFull(ctx context.Context, cfg pipeconfig.Config, producerOut io.Reader,
	stdout, stderr io.Writer, stats []*summary.ConsumerStats, log *logger.Logger, fail *failFlag) {

	data, err := io.ReadAll(producerOut)
	if err != nil {
		log.Log(logger.Entry{Level: logger.LevelError, Source: "producer", Message: fmt.Sprintf("read: %v", err)})
		fail.Set()
		return
	}

	switch cfg.FullVia {
	case pipeconfig.FullViaTempfile:
		runFullTempfile(ctx, cfg, data, stdout, stderr, stats, log, fail)
	default:
		runFullStdin(ctx, cfg, data, stdout, stderr, stats, log, fail)
	}
}

func runFullStdin(ctx context.Context, cfg pipeconfig.Config, data []byte,
	stdout, stderr io.Writer, stats []*summary.ConsumerStats, log *logger.Logger, fail *failFlag) {

	for i, consumer := range cfg.Consumers {
		if ctx.Err() != nil {
			return
		}
		res := RunConsumer(ctx, consumer, bytes.NewReader(data), stdout, stderr,
			cfg.Workdir, cfg.Env, cfg.Retry, cfg.RetryWait, stats[i], log, i)
		if res.ExitCode != 0 {
			fail.Set()
		}
	}
}

func runFullTempfile(ctx context.Context, cfg pipeconfig.Config, data []byte,
	stdout, stderr io.Writer, stats []*summary.ConsumerStats, log *logger.Logger, fail *failFlag) {

	tmp, err := os.CreateTemp("", "gopipe-full-*.tmp")
	if err != nil {
		log.Log(logger.Entry{Level: logger.LevelError, Source: "pipeline", Message: fmt.Sprintf("tempfile: %v", err)})
		fail.Set()
		return
	}
	// Per spec §7.3 contract 10: defer Remove immediately after creation.
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		fail.Set()
		return
	}
	_ = tmp.Close()

	for i, consumer := range cfg.Consumers {
		if ctx.Err() != nil {
			return
		}
		args := append(append([]string{}, consumer...), tmp.Name())
		res := RunConsumer(ctx, args, bytes.NewReader(nil), stdout, stderr,
			cfg.Workdir, cfg.Env, cfg.Retry, cfg.RetryWait, stats[i], log, i)
		if res.ExitCode != 0 {
			fail.Set()
		}
	}
}
