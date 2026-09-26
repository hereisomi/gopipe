# gopipe

gopipe is a Windows-native CLI pipeline wrapper for legacy RPA and ETL tools used in
telecom operations.

## Status — Phase 1

Phase 1 delivers the skeleton: subcommand dispatch, version metadata and the build
script. Every subcommand owned by a later phase routes correctly and reports a
`not yet implemented` stub (exit 1) — no pipeline, db, registry, shell or GUI logic
exists yet.

| Subcommand | Phase | State |
|---|---|---|
| `pipeline` (default) | 2 | stub |
| `db extract` | 3 | stub |
| `db load` | 4 | stub |
| `run` | 5 | stub |
| `registry` | 5 | stub |
| `shell` | 2 | stub |
| `gui` | 6 | stub |
| `version` | 1 | implemented |
| `help` | 1 | implemented |

## Build

```bat
build_all.bat
```

Produces `gopipe.exe` and `gopipe-gui.exe` for `windows/amd64` with `CGO_ENABLED=0`,
injecting version, commit and build time into `gopipe/internal/version.*` via
`-ldflags`.

## Usage

```bat
gopipe help                         :: top-level usage (spec §10)
gopipe version                      :: version and build information
gopipe python a.py -r python b.py   :: implicit pipeline mode (Phase 2)
```

## Layout

`docs/phase1_unified_binary_architecture.md` §5 is the contract: `cmd/gopipe`,
`cmd/gopipe-gui` and the `internal/` packages, all created in Phase 1 as stubs for
the phases that fill them in.

## Tests

```sh
go build ./...
go test ./...
```
