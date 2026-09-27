//go:build !windows

package windows

import "os"

// Non-Windows fallback: no Win32 console APIs exist here. Keeping the same
// signatures lets the pipeline and CLI build and run on other platforms.
func EnableANSI()             {}
func SetTitle(string)         {}
func RestoreTitle()           {}
func IsConsole(*os.File) bool { return false }
