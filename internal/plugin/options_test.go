package plugin

import "testing"

func TestDefaultCoreOptions(t *testing.T) {
	o := DefaultCoreOptions()
	if o.DailyNotes.DateFormat != "YYYY-MM-DD" ||
		o.Templates.TimeFormat != "HH:mm" || !o.WebViewer.AdBlock {
		t.Fatalf("unexpected defaults: %+v", o)
	}
	if o.DailyNotes.OpenOnStartup || o.WebViewer.AllowLocal ||
		o.WebViewer.ClearOnQuit {
		t.Fatal("a default that should be off is on")
	}
}

func TestCoreOptions_sanitizedFillsOnlyEmptyFields(t *testing.T) {
	o := DefaultCoreOptions()
	o.DailyNotes.DateFormat = ""
	o.DailyNotes.Folder = `  Daily notes\2026  `
	o.Templates.DateFormat = ""
	o.DailyNotes.OpenOnStartup = true
	o.WebViewer.AdBlock = false
	got := o.Sanitized()
	if got.DailyNotes.DateFormat != "YYYY-MM-DD" ||
		got.Templates.DateFormat != "YYYY-MM-DD" {
		t.Error("empty formats did not take their defaults")
	}
	if got.DailyNotes.Folder != "Daily notes/2026" {
		t.Errorf("folder = %q", got.DailyNotes.Folder)
	}
	if !got.DailyNotes.OpenOnStartup || got.WebViewer.AdBlock {
		t.Error("an explicit choice was overwritten")
	}
}
