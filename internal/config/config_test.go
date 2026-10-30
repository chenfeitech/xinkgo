package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestLoad_missingFileDefaults checks that a first start, which has no
// file, gets the shipped defaults and no error.
func TestLoad_missingFileDefaults(t *testing.T) {
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg != Default() {
		t.Errorf("got %+v, want the defaults %+v", cfg, Default())
	}
}

// TestSaveLoad_roundTrip checks that what the app writes it reads back.
func TestSaveLoad_roundTrip(t *testing.T) {
	dir := t.TempDir()
	want := Config{Theme: ThemeDark, LogLevel: "debug"}
	if err := want.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// TestSave_overwrites checks that saving replaces a file that is already
// there, which is the second and later start of the app.
func TestSave_overwrites(t *testing.T) {
	dir := t.TempDir()
	first := Config{Theme: ThemeLight, LogLevel: "info"}
	if err := first.Save(dir); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	second := Config{Theme: ThemeDark, LogLevel: "warn"}
	if err := second.Save(dir); err != nil {
		t.Fatalf("second Save: %v", err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != second {
		t.Errorf("got %+v, want %+v", got, second)
	}
}

// TestSave_rejectsInvalid checks that a wrong configuration never reaches
// the disk.
func TestSave_rejectsInvalid(t *testing.T) {
	dir := t.TempDir()
	if err := (Config{Theme: "neon"}).Save(dir); err == nil {
		t.Error("Save accepted an unknown theme")
	}
	if _, err := os.Stat(filepath.Join(dir, FileName)); err == nil {
		t.Error("Save wrote a file for an invalid configuration")
	}
}

// TestLoad_brokenFileIsAnError checks that a file the app cannot read is
// reported, not quietly turned into the defaults.
func TestLoad_brokenFileIsAnError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName),
		[]byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Error("Load accepted a file that is not JSON")
	}
}

// TestLoad_unknownValuesAreAnError checks the values a user can put in the
// file by hand.
func TestLoad_unknownValuesAreAnError(t *testing.T) {
	dir := t.TempDir()
	data, err := json.Marshal(Config{Theme: ThemeDark, LogLevel: "loud"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, FileName), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Error("Load accepted an unknown log level")
	}
}

// TestLoad_missingFieldsFallBack checks that a file written by an older
// version, without the fields added since, still loads.
func TestLoad_missingFieldsFallBack(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName),
		[]byte(`{"theme":"dark"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Theme != ThemeDark {
		t.Errorf("theme %q, want %q", cfg.Theme, ThemeDark)
	}
	if cfg.LogLevel != Default().LogLevel {
		t.Errorf("log level %q, want %q", cfg.LogLevel, Default().LogLevel)
	}
}
