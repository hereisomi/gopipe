// Package pipeline — parallel mode with worker pool (spec §F-02).
package pipeline

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"sync"

	"gopipe/internal/logger"
	"gopipe/internal/pipeconfig"
	"gopipe/internal/summary"
)

type parallelJob struct {
	consumerIdx int
	args        []string
}

func runParallel(ctx context.Context, cfg pipeconfig.Config, producerOut io.Reader,
	stdout, stderr io.Writer, stats []*summary.ConsumerStats, log *logger.Logger, fail *failFlag) {

	workers := cfg.Workers
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan parallelJob, workers*2)
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if ctx.Err() != nil {
					continue
				}
				res := RunConsumer(ctx, job.args, bytes.NewReader(nil), stdout, stderr,
					cfg.Workdir, cfg.Env, cfg.Retry, cfg.RetryWait,
					stats[job.consumerIdx], log, job.consumerIdx)
				if res.ExitCode != 0 {
					fail.Set()
				}
			}
		}()
	}

	scanner := bufio.NewScanner(producerOut)
	for scanner.Scan() {
		if ctx.Err() != nil {
			break
		}
		line := scanner.Text()
		for i, consumer := range cfg.Consumers {
			args := append(append([]string{}, consumer...), line)
			jobs <- parallelJob{consumerIdx: i, args: args}
		}
	}
	close(jobs)
	wg.Wait()
}
