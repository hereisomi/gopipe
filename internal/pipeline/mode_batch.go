// Package pipeline — batch mode (spec §F-04).
package pipeline

import (
	"bufio"
	"bytes"
	"context"
	"io"

	"gopipe/internal/logger"
	"gopipe/internal/pipeconfig"
	"gopipe/internal/summary"
)

func runBatch(ctx context.Context, cfg pipeconfig.Config, producerOut io.Reader,
	stdout, stderr io.Writer, stats []*summary.ConsumerStats, log *logger.Logger, fail *failFlag) {

	size := cfg.BatchSize
	if size < 1 {
		size = 1
	}

	flush := func(batch []string) {
		if len(batch) == 0 {
			return
		}
		for i, consumer := range cfg.Consumers {
			args := append(append([]string{}, consumer...), batch...)
			res := RunConsumer(ctx, args, bytes.NewReader(nil), stdout, stderr,
				cfg.Workdir, cfg.Env, cfg.Retry, cfg.RetryWait, stats[i], log, i)
			if res.ExitCode != 0 {
				fail.Set()
			}
		}
	}

	batch := make([]string, 0, size)
	scanner := bufio.NewScanner(producerOut)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return
		}
		batch = append(batch, scanner.Text())
		if len(batch) >= size {
			flush(batch)
			batch = batch[:0]
		}
	}
	flush(batch) // partial tail
}
