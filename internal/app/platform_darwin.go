//go:build darwin

package app

// isMacOS reports whether the app runs on macOS, where it stays open with
// no window.
func isMacOS() bool { return true }
