package dispatch

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// runCapture routes args the way main does and returns the exit code plus
// everything the handler wrote.
func runCapture(args ...string) (code int, stdout, stderr string) {
	var out, errb bytes.Buffer
	code = run(context.Background(), args, &out, &errb)
	return code, out.String(), errb.String()
}

// TestTopLevelHelpMatchesSpec pins the help block to the golden copy of
// spec §10 (docs/phase1_unified_binary_architecture.md).
func TestTopLevelHelpMatchesSpec(t *testing.T) {
	golden, err := os.ReadFile(filepath.Join("testdata", "top_level_help.txt"))
	if err != nil {
		t.Fatalf("reading golden: %v", err)
	}

	for _, args := range [][]string{nil, {"--help"}, {"-h"}, {"help"}} {
		code, stdout, stderr := runCapture(args...)
		if code != 0 {
			t.Errorf("args %q: exit %d, want 0 (stderr: %q)", args, code, stderr)
			continue
		}
		if stdout != string(golden) {
			t.Errorf("args %q: help output does not match spec §10 golden\n--- got ---\n%s\n--- want ---\n%s", args, stdout, golden)
		}
	}
}

func TestRouting(t *testing.T) {
	const unknown = `unknown subcommand "frobnicate" — run 'gopipe help' for usage`

	tests := []struct {
		name   string
		args   []string
		code   int
		stdout string
		stderr string
	}{
		// Global flags (spec §3.3, §4.1)
		{"version subcommand", []string{"version"}, 0, "gopipe dev", ""},
		{"version long flag", []string{"--version"}, 0, "gopipe dev", ""},
		{"version short flag", []string{"-v"}, 0, "gopipe dev", ""},

		// Pipeline: explicit and implicit (spec §3.2, §3.3)
		{"pipeline explicit", []string{"pipeline"}, 1, "", "pipeline subcommand — not yet implemented"},
		{"pipeline with args", []string{"pipeline", "python", "a.py", "-r", "python", "b.py"}, 1, "", "pipeline subcommand — not yet implemented"},
		{"implicit pipeline producer-first", []string{"python", "a.py", "-r", "python", "b.py"}, 1, "", "pipeline subcommand — not yet implemented"},
		{"implicit pipeline flags-first", []string{"--mode", "parallel"}, 1, "", "pipeline subcommand — not yet implemented"},
		{"help pipeline exits zero", []string{"help", "pipeline"}, 0, "pipeline subcommand — not yet implemented", ""},

		// db group (spec §3.4)
		{"db bare", []string{"db"}, 1, "", "db subcommand — not yet implemented"},
		{"db shorthand is extract", []string{"db", "--json", "task.json"}, 1, "", "db extract subcommand — not yet implemented"},
		{"db extract", []string{"db", "extract", "--json", "task.json"}, 1, "", "db extract subcommand — not yet implemented"},
		{"db load", []string{"db", "load", "--json", "load_task.json"}, 1, "", "db load subcommand — not yet implemented"},
		{"db unknown verb", []string{"db", "frobnicate"}, 1, "", unknown},
		{"help db exits zero", []string{"help", "db"}, 0, "db subcommand — not yet implemented", ""},
		{"help db extract", []string{"help", "db", "extract"}, 0, "db extract subcommand — not yet implemented", ""},

		// Remaining Phase 2+ subcommands (spec §12)
		{"run", []string{"run"}, 1, "", "run subcommand — not yet implemented"},
		{"registry", []string{"registry"}, 1, "", "registry subcommand — not yet implemented"},
		{"shell", []string{"shell"}, 1, "", "shell subcommand — not yet implemented"},
		{"gui", []string{"gui"}, 1, "", "gui subcommand — not yet implemented"},

		// Unknown subcommand (spec §6.3)
		{"help unknown topic", []string{"help", "frobnicate"}, 1, "", unknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runCapture(tt.args...)
			if code != tt.code {
				t.Errorf("exit = %d, want %d (stdout: %q, stderr: %q)", code, tt.code, stdout, stderr)
			}
			if !strings.Contains(stdout, tt.stdout) {
				t.Errorf("stdout = %q, want it to contain %q", stdout, tt.stdout)
			}
			if !strings.Contains(stderr, tt.stderr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr, tt.stderr)
			}
		})
	}
}

// TestVersionLines checks the ldflags-driven banner (spec §7). A plain build
// reports the defaults; build_all.bat overwrites Version, Commit and Built.
func TestVersionLines(t *testing.T) {
	code, stdout, _ := runCapture("version")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	for _, want := range []string{
		"gopipe dev\n",
		"Built:  unknown\n",
		"Commit: unknown\n",
		"OS:     " + runtime.GOOS + "/" + runtime.GOARCH + "\n",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("version output %q does not contain %q", stdout, want)
		}
	}
}

// TestPanicIsRecovered covers spec §6.3: a panicking handler becomes exit code
// 1 plus a message, never a crash dump.
func TestPanicIsRecovered(t *testing.T) {
	saved := registry
	t.Cleanup(func() { registry = saved })

	Register(Command{
		Name:  "boom",
		Short: "panics on purpose",
		Run:   func(context.Context, []string) int { panic("kaboom") },
	})

	code, stdout, stderr := runCapture("boom")
	if code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if strings.Contains(stdout, "kaboom") {
		t.Errorf("panic message leaked to stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "boom") || !strings.Contains(stderr, "kaboom") {
		t.Errorf("stderr = %q, want it to name the subcommand and the panic value", stderr)
	}
}
