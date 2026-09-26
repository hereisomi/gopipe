// Package pipeline — sequential mode (spec §F-01).
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

func runSequential(ctx context.Context, cfg pipeconfig.Config, producerOut io.Reader,
	stdout, stderr io.Writer, stats []*summary.ConsumerStats, log *logger.Logger, fail *failFlag) {

	scanner := bufio.NewScanner(producerOut)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return
		}
		line := scanner.Text()
		for i, consumer := range cfg.Consumers {
			args := append(append([]string{}, consumer...), line)
			res := RunConsumer(ctx, args, bytes.NewReader(nil), stdout, stderr,
				cfg.Workdir, cfg.Env, cfg.Retry, cfg.RetryWait, stats[i], log, i)
			if res.ExitCode != 0 {
				fail.Set()
			}
		}
	}
}
