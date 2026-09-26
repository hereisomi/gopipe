# gopipe — Phase 1: Unified Binary Architecture & Subcommand Dispatch
**Version:** 1.0.0
**Classification:** Internal — Engineering
**Authors:** System Architecture Team
**Target Audience:** Go Developer (implementer)
**Depends on:** nothing — this is the foundation document
**Followed by:** Phase 2 (Pipeline), Phase 3 (DB Extract), Phase 4 (DB Load), Phase 5 (Registry)

---

## 1. Purpose of This Document

This document defines the top-level structure of the gopipe binary: how subcommands are
dispatched, how flags are namespaced across subcommands, how the two binaries
(`gopipe.exe` and `gopipe-gui.exe`) relate to each other, and the shared internal
packages that all subcommands depend on.

Every subsequent phase document describes one subcommand in detail. This document is
the contract they all build on. Implement this first.

---

## 2. Binary Overview

gopipe ships as two self-contained Windows executables:

| Binary | Purpose |
|---|---|
| `gopipe.exe` | CLI entry point — all subcommands, interactive shell |
| `gopipe-gui.exe` | Native Windows GUI (lxn/walk) — wraps the same internal packages. See `phase6_native_gui.md` |

Both binaries are built from the same repository. They share all `internal/` packages.
Neither requires a Go runtime, JVM, Python, or any external dependency on the target machine.

---

## 3. Subcommand Model

gopipe uses a **verb-first subcommand model**, identical in style to `git`, `docker`, and
`kubectl`. The first positional argument after `gopipe` is always the subcommand name.

### 3.1 Subcommand Table

| Subcommand | Alias | Phase | Description |
|---|---|---|---|
| `gopipe pipeline` | *(default)* | Phase 2 | Run a producer→consumer pipeline |
| `gopipe db extract` | `gopipe db` | Phase 3 | Execute a SQL query, write result to file |
| `gopipe db load` | — | Phase 4 | Bulk-load a file into a target database table |
| `gopipe run` | — | Phase 5 | Run a named pipeline from the registry |
| `gopipe registry` | — | Phase 5 | Manage the named pipeline registry |
| `gopipe shell` | — | Phase 2 | Interactive REPL |
| `gopipe gui` | — | Phase 6 | Launch native GUI (`gopipe-gui.exe`, lxn/walk) |
| `gopipe version` | `-v` | Phase 1 | Print version and build info |
| `gopipe help` | `--help` | Phase 1 | Print help for a subcommand |

### 3.2 Default Subcommand

When no subcommand is given and the first argument looks like a command (not a flag),
gopipe treats it as the `pipeline` subcommand implicitly:

```
gopipe python cdr_export.py -r python billing_ingest.py
```

is equivalent to:

```
gopipe pipeline python cdr_export.py -r python billing_ingest.py
```

This preserves backward compatibility with any existing scripts that call gopipe without
a subcommand.

### 3.3 Subcommand Detection Logic

```
args = os.Args[1:]

if len(args) == 0                    → print help, exit 0
if args[0] in {"--help", "-h"}       → print top-level help, exit 0
if args[0] in {"--version", "-v"}    → print version, exit 0
if args[0] == "pipeline"             → dispatch to pipeline subcommand
if args[0] == "db"                   → dispatch to db subcommand group
if args[0] == "run"                  → dispatch to registry run
if args[0] == "registry"             → dispatch to registry management
if args[0] == "shell"                → dispatch to interactive shell
if args[0] == "gui"                  → launch gopipe-gui.exe detached (Phase 6)
if args[0] == "version"              → print version, exit 0
if args[0] == "help"                 → print help for args[1] if given
if args[0] starts with "-"           → implicit pipeline subcommand (flags-first style)
default                              → implicit pipeline subcommand (producer-first style)
```

### 3.4 `db` Subcommand Group

`db` is a subcommand group with its own second-level verb:

```
gopipe db extract --json task.json
gopipe db load    --json load_task.json
gopipe db         --json task.json        ← shorthand for "db extract"
```

---

## 4. Flag Namespacing

Each subcommand owns its own flag set. Flags are not shared across subcommands.
Global flags (version, help) are handled before subcommand dispatch.

### 4.1 Global Flags (pre-dispatch)

| Flag | Description |
|---|---|
| `--version`, `-v` | Print version and exit |
| `--help`, `-h` | Print help (top-level or subcommand-specific) |

### 4.2 Flag Inheritance Rule

Flags that appear in multiple subcommands (e.g. `--dry-run`, `--log`, `--workers`,
`--timeout`) are **independently defined per subcommand**. They share the same name and
semantics by convention, but are registered separately in each subcommand's flag set.
There is no global flag inheritance mechanism.

This keeps each subcommand's flag set self-contained and avoids flag collision when
subcommands are extended independently in future phases.

---

## 5. Repository Layout

This is the complete repository structure for all phases. Each phase fills in its
section. Implement the skeleton in Phase 1; populate each package in its respective phase.

```
gopipe/
├── cmd/
│   ├── gopipe/
│   │   └── main.go                      # Subcommand dispatch (this document)
│   └── gopipe-gui/
│       ├── main.go                      # Native GUI window (lxn/walk) (Phase 6)
│       └── gopipe-gui.manifest          # comctl32 v6 + DPI awareness manifest (Phase 6)
│
├── internal/
│   ├── dispatch/
│   │   └── dispatch.go                  # Subcommand router (this document)
│   │
│   ├── version/
│   │   └── version.go                   # Version constants + print (this document)
│   │
│   ├── windows/
│   │   └── console.go                   # ANSI enable, title, console detection (Phase 2)
│   │
│   ├── pipeline/                        # Phase 2
│   │   ├── pipeline.go
│   │   ├── mode_sequential.go
│   │   ├── mode_parallel.go
│   │   ├── mode_streaming.go
│   │   ├── mode_batch.go
│   │   ├── mode_full.go
│   │   ├── mode_binary.go
│   │   ├── consumer.go
│   │   └── tee.go
│   │
│   ├── pipeconfig/                      # Phase 2
│   │   ├── config.go
│   │   ├── flags.go
│   │   └── file.go
│   │
│   ├── shell/                           # Phase 2
│   │   └── shell.go
│   │
│   ├── logger/                          # Shared — Phase 2 implements, all phases use
│   │   └── logger.go
│   │
│   ├── summary/                         # Shared — Phase 2 implements, all phases use
│   │   └── summary.go
│   │
│   ├── progress/                        # Phase 2
│   │   └── progress.go
│   │
│   ├── exitcode/                        # Shared — Phase 2 implements, all phases use
│   │   └── exitcode.go
│   │
│   ├── connector/                       # Phase 3 — DB drivers
│   │   ├── connector.go
│   │   ├── sybase.go
│   │   ├── gaussdb.go
│   │   ├── oracle.go
│   │   ├── mssql.go
│   │   └── postgres.go
│   │
│   ├── dbtask/                          # Phase 3 + 4 — task file schema
│   │   ├── task.go
│   │   ├── batch.go
│   │   └── template.go
│   │
│   ├── dbextract/                       # Phase 3 — query executor + writers
│   │   ├── executor.go
│   │   └── writer/
│   │       ├── writer.go
│   │       ├── csv.go
│   │       ├── tsv.go
│   │       ├── json.go
│   │       └── jsonl.go
│   │
│   ├── dbload/                          # Phase 4 — bulk loader
│   │   ├── loader.go
│   │   └── reader/
│   │       ├── reader.go
│   │       ├── csv.go
│   │       └── jsonl.go
│   │
│   └── registry/                        # Phase 5 — named pipeline store
│       ├── registry.go
│       └── store.go
│
├── go.mod
├── go.sum
├── build_all.bat
└── README.md
```

---

## 6. `cmd/gopipe/main.go` — Dispatch Implementation

### 6.1 Responsibilities

- Parse `os.Args[1:]` to identify the subcommand
- Delegate to the appropriate subcommand handler
- Handle `--version` and `--help` before dispatch
- Set up the root `context.Context` with signal cancellation
- Exit with the subcommand's returned exit code

### 6.2 Pseudocode

```go
func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()

    code := dispatch.Run(ctx, os.Args[1:])
    os.Exit(code)
}
```

### 6.3 `internal/dispatch/dispatch.go`

```go
// Run routes os.Args[1:] to the correct subcommand handler.
// Returns the exit code the process should use.
func Run(ctx context.Context, args []string) int
```

The dispatch function must:
- Never call `os.Exit` itself — return the exit code to `main`
- Recover from panics in subcommand handlers and return exit code 1 with an error message
- Print a clear error for unknown subcommands: `"unknown subcommand %q — run 'gopipe help' for usage"`

---

## 7. `internal/version/version.go`

Build metadata is injected at compile time via `-ldflags`. The version package exposes
these as package-level variables and a `Print()` function used by all subcommands.

```go
package version

var (
    Version = "dev"           // set by -ldflags "-X version.Version=1.0.0"
    Commit  = "unknown"       // set by -ldflags "-X version.Commit=abc1234"
    Built   = "unknown"       // set by -ldflags "-X version.Built=2025-01-15T08:00:00Z"
    GoVer   = runtime.Version()
)

func Print() {
    // Output:
    // gopipe 1.0.0
    // Built:  2025-01-15T08:00:00Z
    // Commit: abc1234
    // Go:     go1.23.4
    // OS:     windows/amd64
}
```

---

## 8. `build_all.bat` — Build Script

The build script compiles both binaries with version metadata injected:

```bat
@echo off
set VERSION=1.0.0
set COMMIT=%GIT_COMMIT%
set BUILT=%DATE%T%TIME%

go build -ldflags "-X gopipe/internal/version.Version=%VERSION% -X gopipe/internal/version.Commit=%COMMIT% -X gopipe/internal/version.Built=%BUILT%" -o gopipe.exe ./cmd/gopipe
rsrc -manifest gopipe-gui.manifest -o cmd/gopipe-gui/rsrc.syso
go build -ldflags "-s -w -H windowsgui -X gopipe/internal/version.Version=%VERSION% -X gopipe/internal/version.Commit=%COMMIT% -X gopipe/internal/version.Built=%BUILT%" -o gopipe-gui.exe ./cmd/gopipe-gui
```

Both binaries must be built with `CGO_ENABLED=0` to ensure no CGo dependency.
Exception: if a future driver requires CGo, document it explicitly and provide a build
tag to opt in.

---

## 9. `go.mod` — Module and Dependency Baseline

```
module gopipe

go 1.23

require (
    github.com/spf13/pflag       v1.0.5    // flag parsing for all subcommands
    golang.org/x/sys             latest    // Windows console API (SetConsoleMode)
    gopkg.in/yaml.v3             latest    // config file parsing (Phase 2)
)
```

Additional dependencies are added per phase:
- Phase 3/4: `github.com/thda/tds`, `github.com/sijms/go-ora/v2`, `github.com/microsoft/go-mssqldb`, `github.com/jackc/pgx/v5`, `golang.org/x/text`
- Phase 5: no new external dependencies
- Phase 6: `github.com/lxn/walk`, `github.com/lxn/win`, `github.com/akavel/rsrc` (build-time only)

All dependencies must be pure Go or have no CGo requirement on Windows.

---

## 10. Top-Level Help Output

When `gopipe help` or `gopipe --help` is run with no subcommand:

```
gopipe — Windows CLI pipeline wrapper for RPA and ETL tools

Usage:
  gopipe <subcommand> [flags]
  gopipe <producer...> -r <consumer...> [flags]   (implicit pipeline mode)

Subcommands:
  pipeline    Run a producer→consumer pipeline (default when no subcommand given)
  db          Execute database queries or load data (db extract / db load)
  run         Run a named pipeline from the registry
  registry    Manage named pipeline definitions
  shell       Interactive pipeline REPL
  gui         Launch native GUI (gopipe-gui.exe)
  version     Print version and build information
  help        Print help for a subcommand

Examples:
  gopipe python export.py -r python ingest.py --mode parallel
  gopipe db extract --json task.json
  gopipe db load --json load_task.json
  gopipe run nightly-cdr
  gopipe shell
  gopipe gui

Run 'gopipe help <subcommand>' for subcommand-specific flags.
```

---

## 11. Non-Functional Requirements (Phase 1 scope)

| Requirement | Target |
|---|---|
| Dispatch overhead | < 1ms — subcommand routing adds no measurable latency |
| Unknown subcommand | Clear error message + exit code 1, never a panic |
| Help output | Always exits 0, even when called after an error |
| Signal handling | `context.WithCancel` wired to `os.Interrupt` and `syscall.SIGTERM` before any subcommand runs |
| CGo | `CGO_ENABLED=0` for both binaries unless explicitly documented otherwise |

---

## 12. Deliverable Checklist for Phase 1

The following must be complete before Phase 2 begins:

- [ ] `cmd/gopipe/main.go` — signal context + dispatch call
- [ ] `internal/dispatch/dispatch.go` — subcommand routing table
- [ ] `internal/version/version.go` — version variables + `Print()`
- [ ] `go.mod` with baseline dependencies
- [ ] `build_all.bat` with `-ldflags` version injection
- [ ] `gopipe version` prints correct output
- [ ] `gopipe help` prints top-level usage
- [ ] `gopipe help pipeline` prints "pipeline subcommand — not yet implemented" (stub)
- [ ] `gopipe help db` prints "db subcommand — not yet implemented" (stub)
- [ ] All Phase 2+ subcommands (`pipeline`, `db`, `run`, `registry`, `shell`, `gui`) route correctly but print a "not yet implemented" stub and exit 1 — no implementation in Phase 1
- [ ] Unknown subcommand prints error and exits 1
- [ ] Both binaries compile with `CGO_ENABLED=0`
