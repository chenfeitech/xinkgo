package view

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestWorkspace_anExportIsGivenWhatThePreviewReads checks that what is
// written out is what the user sees: the very text the preview is given of
// the note that is open, and where the note stands, which is where the
// pictures it names are.
func TestWorkspace_anExportIsGivenWhatThePreviewReads(t *testing.T) {
	// The note written out is the one on show, which is the one opened
	// last.
	w, _, b := shownWorkspace(t)
	path, text, ok := w.ExportText()
	if !ok {
		t.Fatal("there is nothing to write out")
	}
	if path != b {
		t.Errorf("the note is %q, want %q", path, b)
	}
	if text != "# b" {
		t.Errorf("what is written out is %q, want %q", text, "# b")
	}
}

// TestWorkspace_aFileTheAppOnlyShowsIsNotWrittenOut checks a picture and a
// PDF: the machine's own program shows them, and there is nothing of the
// app's own in them to write out.
func TestWorkspace_aFileTheAppOnlyShowsIsNotWrittenOut(t *testing.T) {
	w, _, _ := shownWorkspace(t)
	dir := t.TempDir()
	pict := writeFile(t, dir, "a.png", "not really a picture")
	if _, err := w.Tabs.Open(pict); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, _, ok := w.ExportText(); ok {
		t.Error("a picture is written out")
	}
}

// TestWorkspace_theWayOutStandsOverThePreview checks the button the note
// is written out by: it stands over the preview it writes out, and not
// over a file the app only shows.
func TestWorkspace_theWayOutStandsOverThePreview(t *testing.T) {
	w, _, _ := shownWorkspace(t)
	w.Layout.Right = 300
	w.SetPreviewShown(true)
	found := func() bool {
		tt := ui.NewTester(w.View, 900, 600)
		tt.Frame()
		tt.Frame()
		_, ok := tt.Find("Export")
		return ok
	}
	if !found() {
		t.Error("the preview has no way out of it")
	}
	// A picture has nothing in it to write out, so it gets no button.
	dir := t.TempDir()
	pict := writeFile(t, dir, "a.png", "not really a picture")
	if _, err := w.Tabs.Open(pict); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if found() {
		t.Error("a picture has a way out of it")
	}
}

// writeFile is a file with that in it, for a test to open.
func writeFile(t *testing.T, dir, name, in string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(in), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
