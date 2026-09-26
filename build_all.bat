@echo off
rem ---------------------------------------------------------------------------
rem gopipe - build script (Phase 1)
rem
rem Builds both Windows binaries with build metadata injected through -ldflags.
rem   gopipe.exe      CLI entry point
rem   gopipe-gui.exe  native GUI skeleton (window lands in Phase 6)
rem
rem Spec: docs/phase1_unified_binary_architecture.md
rem        8  - build script
rem        7  - version metadata (gopipe/internal/version.*)
rem        11 - CGO_ENABLED=0 for both binaries
rem ---------------------------------------------------------------------------
setlocal

rem Windows/amd64 only, and no CGo dependency in either binary (spec §2, §11).
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64

set VERSION=1.0.0

rem Commit: GIT_COMMIT wins when the caller sets it (CI), otherwise derive it
rem from HEAD. Never pass an empty value to -X: it would blank the Commit line.
if not defined GIT_COMMIT (
    for /f "tokens=* usebackq" %%c in (`git rev-parse --short HEAD 2^>nul`) do set "GIT_COMMIT=%%c"
)
if not defined GIT_COMMIT set "GIT_COMMIT=unknown"
set COMMIT=%GIT_COMMIT%

rem Built: RFC3339 UTC, the format shown in spec §7. %DATE%/%TIME% are locale
rem dependent, so PowerShell is preferred and %DATE%T%TIME% is the fallback.
set BUILT=
for /f "tokens=* usebackq" %%b in (`powershell -NoProfile -Command "[DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')" 2^>nul`) do set "BUILT=%%b"
if not defined BUILT set BUILT=%DATE%T%TIME%

set LDFLAGS=-X gopipe/internal/version.Version=%VERSION% -X gopipe/internal/version.Commit=%COMMIT% -X gopipe/internal/version.Built=%BUILT%

echo gopipe build: version=%VERSION% commit=%COMMIT% built=%BUILT%
echo target:       %GOOS%/%GOARCH% (CGO_ENABLED=%CGO_ENABLED%)
echo.

echo Building gopipe.exe ...
go build -ldflags "%LDFLAGS%" -o gopipe.exe ./cmd/gopipe
if errorlevel 1 goto :fail

echo Building gopipe-gui.exe ...
rem rsrc embeds the comctl32 v6 + DPI manifest (§5). It is a Phase 6 build-time
rem tool (§9), so a missing rsrc skips embedding instead of failing Phase 1.
where rsrc >nul 2>nul
if errorlevel 1 (
    echo   rsrc not found on PATH - skipping manifest embedding.
    echo   Phase 6 setup: go install github.com/akavel/rsrc@latest
) else (
    rsrc -manifest cmd/gopipe-gui/gopipe-gui.manifest -o cmd/gopipe-gui/rsrc.syso
    if errorlevel 1 goto :fail
)

go build -ldflags "-s -w -H windowsgui %LDFLAGS%" -o gopipe-gui.exe ./cmd/gopipe-gui
if errorlevel 1 goto :fail

echo.
echo Build complete:
gopipe.exe version
exit /b 0

:fail
echo.
echo Build FAILED.
exit /b 1
