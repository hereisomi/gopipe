# gopipe — Phase 1 Handoff Brief
**From:** Principal Architect / Project Lead
**To:** Remote Go Developer
**Repo:** https://github.com/hereisomi/gopipe
**Scope:** Phase 1 only — binary skeleton, subcommand dispatch, version, build script
**Spec authority:** `phase1_unified_binary_architecture.md` (attached — that document is the contract; this note is the operating procedure around it)

---

## What you are building

The foundation of `gopipe`, a Windows-only CLI pipeline wrapper for telecom RPA/ETL
operations. Phase 1 is deliberately thin: **a binary skeleton with correct subcommand
routing and build plumbing — no features.** Every later phase hangs off the dispatch
and layout defined here, so structural correctness matters more than code volume.

Target: `gopipe` compiles, all subcommands route, `version`/`help` work, everything
else is a stub that exits cleanly.

## Explicitly out of scope for Phase 1

Do **not** implement any of these — they are later phases with their own specs:

- Pipeline execution modes (sequential/parallel/streaming/batch/full/binary)
- `gopipe db extract` / `db load` — no database drivers, no task files
- `gopipe run` / `gopipe registry` / `gopipe cred`
- `gopipe-gui.exe` GUI implementation (Phase 6, `lxn/walk`) — Phase 1 only needs the
  `cmd/gopipe-gui` skeleton to compile
- `gopipe shell` REPL

Phase 2+ subcommands must route correctly and print a "not yet implemented" stub
(exit 1). If you find yourself implementing real logic, stop — that's scope creep.

## Workflow

1. Clone the repo, then check out **your assigned branch** (already created and
   pushed — do not create a new one):

   | Developer | Branch |
   |---|---|
   | `yfarzana750-lgtm` | `feat/phase1-yfarzana750` |
   | `fhimi2986` | `feat/phase1-fhimi2986` |

   Both branches implement the **same** Phase 1 spec independently — do not look at
   or copy each other's branch. We merge one implementation after review.

2. Implement per the spec; commit early and often, conventional-commit style
   (`feat:`, `chore:`, `docs:`).
3. Open a PR to `main` when the acceptance checklist passes locally.
4. Do not force-push `main`, do not commit secrets or local `.env` files
   (`.gitignore` already covers `.env` and `*.exe`).

## Environment

- Go 1.23+ on your dev machine; target is Windows (`GOOS=windows GOARCH=amd64`).
- You may develop on any OS — Phase 1 has no Windows-specific code paths (console
  helpers arrive in Phase 2). `CGO_ENABLED=0` builds must pass.

## Acceptance criteria (all must pass before PR review)

- [ ] `go build ./...` clean, `CGO_ENABLED=0` for both binaries
- [ ] `gopipe version` prints version/commit/built/go/os (ldflags injection works —
      note the module path is `gopipe`, use `gopipe/internal/version.X` in `-X` flags)
- [ ] `gopipe help` prints the top-level usage block exactly as spec §10
- [ ] `gopipe help pipeline` / `help db` → "not yet implemented" stubs
- [ ] `gopipe pipeline ...`, `db`, `run`, `registry`, `shell`, `gui` all route and
      print stubs — dispatch table is data-driven (a slice/map of handlers), not a
      switch-per-subcommand, so later phases register themselves without touching
      `main.go`
- [ ] Unknown subcommand → `unknown subcommand "x" — run 'gopipe help' for usage`, exit 1
- [ ] Implicit pipeline mode: `gopipe python a.py -r python b.py` routes to the
      pipeline stub (per spec §3.2/§3.3 detection logic)
- [ ] Panic in a subcommand handler is recovered → exit 1 + message, not a crash dump
- [ ] `build_all.bat` builds both exes with version metadata
- [ ] Repo layout matches spec §5 (create the directories/files for all phases as
      stubs — the skeleton is part of this deliverable)

## Definition of done

PR opened, checklist ticked, `build_all.bat` output attached to the PR description
(both exes + `gopipe version` output). I'll review against the spec section by
section. Questions or spec ambiguities → comment on the PR, don't guess.

## Corrections already applied to the spec

- `build_all.bat` ldflags now inject into `gopipe/internal/version.*` (an earlier
  draft carried a `healer_v1` module path — if you see that string anywhere, it's a bug).
- Phase numbering: GUI moved from Phase 2 to Phase 6 and is now **native (lxn/walk)**,
  not browser-based. Ignore any older reference to a web UI, SSE, or embedded SPA.

— Project Lead
