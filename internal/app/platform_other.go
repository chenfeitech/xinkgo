//go:build !darwin

package app

// isMacOS reports whether the app runs on macOS, where it stays open with
// no window. Windows and Linux quit with the last window instead.
func isMacOS() bool { return false }
