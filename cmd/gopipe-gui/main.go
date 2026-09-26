// Command gopipe-gui is the entry point of the native Windows GUI binary
// (gopipe-gui.exe).
//
// Phase 1 delivers the skeleton only: the binary must exist and compile with
// CGO_ENABLED=0 so that build_all.bat emits both artifacts and so the
// `gopipe gui` subcommand has something to launch. The lxn/walk window is
// implemented in Phase 6 (docs/phase6_native_gui.md).
//
// Spec: docs/phase1_unified_binary_architecture.md §2, §5, §8.
package main

func main() {
	// Phase 6 replaces this with the walk MainWindow. Until then the process
	// starts and exits cleanly; it must not print, because the binary is built
	// with -H windowsgui and has no attached console.
}
