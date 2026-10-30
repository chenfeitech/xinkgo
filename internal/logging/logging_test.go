package logging

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

// TestLevel_knownNames checks the levels the environment can name.
func TestLevel_knownNames(t *testing.T) {
	cases := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
	for name, want := range cases {
		if got := Level(name); got != want {
			t.Errorf("Level(%q) = %v, want %v", name, got, want)
		}
	}
}

// TestLevel_unknownNameIsInfo checks that a level the app does not know
// never turns logging off.
func TestLevel_unknownNameIsInfo(t *testing.T) {
	if got := Level("loud"); got != slog.LevelInfo {
		t.Errorf("Level(%q) = %v, want %v", "loud", got, slog.LevelInfo)
	}
}

// TestSetup_developmentNeedsNoDirectory checks that a build someone is
// working on logs without a directory to log into.
func TestSetup_developmentNeedsNoDirectory(t *testing.T) {
	if err := Setup("", true); err != nil {
		t.Fatalf("Setup: %v", err)
	}
}

// TestSetup_installedWritesToFile checks that an installed app logs into
// the directory it is given.
func TestSetup_installedWritesToFile(t *testing.T) {
	dir := t.TempDir()
	if err := Setup(dir, false); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, FileName)); err != nil {
		t.Errorf("no log file in %s: %v", dir, err)
	}
}
