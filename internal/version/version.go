// Package version holds the gopipe build metadata that is injected at link
// time and printed by the version subcommand.
//
// The variables below are package-level so that build_all.bat can overwrite
// them without touching this file:
//
//	go build -ldflags "-X gopipe/internal/version.Version=1.0.0 ..."
//
// Spec: docs/phase1_unified_binary_architecture.md §7, §8.
package version

import (
	"fmt"
	"io"
	"os"
	"runtime"
)

// Build metadata. Defaults are what a plain `go build` (no ldflags) produces.
var (
	Version = "dev"     // -X gopipe/internal/version.Version=1.0.0
	Commit  = "unknown" // -X gopipe/internal/version.Commit=abc1234
	Built   = "unknown" // -X gopipe/internal/version.Built=2025-01-15T08:00:00Z
	GoVer   = runtime.Version()
)

// Print writes the version banner to stdout.
//
//	gopipe 1.0.0
//	Built:  2025-01-15T08:00:00Z
//	Commit: abc1234
//	Go:     go1.23.4
//	OS:     windows/amd64
func Print() { Fprint(os.Stdout) }

// Fprint writes the version banner to w. Print is the stdout convenience
// wrapper required by spec §7; Fprint is what the dispatch table uses so the
// output can be captured in tests.
func Fprint(w io.Writer) {
	fmt.Fprintf(w, "gopipe %s\n", Version)
	fmt.Fprintf(w, "%-8s%s\n", "Built:", Built)
	fmt.Fprintf(w, "%-8s%s\n", "Commit:", Commit)
	fmt.Fprintf(w, "%-8s%s\n", "Go:", GoVer)
	fmt.Fprintf(w, "%-8s%s\n", "OS:", runtime.GOOS+"/"+runtime.GOARCH)
}
