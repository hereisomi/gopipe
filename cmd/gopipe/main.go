// Command gopipe is the CLI entry point for the gopipe pipeline wrapper.
//
// main is deliberately thin: it wires a signal-cancellable context, hands
// os.Args[1:] to the dispatch table, and exits with the code the chosen
// subcommand returned. All routing lives in internal/dispatch, so later phases
// add subcommands without touching this file.
//
// Spec: docs/phase1_unified_binary_architecture.md §6.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"gopipe/internal/dispatch"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	os.Exit(dispatch.Run(ctx, os.Args[1:]))
}
