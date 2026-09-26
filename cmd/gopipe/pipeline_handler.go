// pipeline_handler.go wires the Phase 2 pipeline handler into the dispatch
// table. It lives in cmd/gopipe so it is compiled into the binary without
// touching dispatch or main (spec §6.3 extension point).
package main

import (
	"context"
	"fmt"
	"os"

	"gopipe/internal/dispatch"
	"gopipe/internal/pipeline"
	"gopipe/internal/pipeconfig"
	winconsole "gopipe/internal/windows"
)

func init() {
	winconsole.EnableANSI()
	// Update the pipeline stub registered by dispatch.init() with the real handler.
	dispatch.UpdateRun("pipeline", pipelineHandler)
}

func pipelineHandler(ctx context.Context, args []string) int {
	cfg, err := pipeconfig.ParseArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gopipe: %v\n", err)
		return 1
	}
	return pipeline.Run(ctx, cfg, os.Stdout, os.Stderr)
}
