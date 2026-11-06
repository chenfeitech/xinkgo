package view

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/view/geom"
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

// TestWorkspace_theWayOutStandsBesideWechatInTheHead checks that the button
// the note is written out by stands in the head with the WeChat button, in
// each of the shapes the preview takes, and not over what is read.
func TestWorkspace_theWayOutStandsBesideWechatInTheHead(t *testing.T) {
	for _, c := range []struct {
		name string
		zoom geom.Zoom
	}{
		{"in its pane", geom.ZoomNone},
		{"given the whole window", geom.ZoomPreview},
		{"spread over the note", geom.ZoomPreviewSpread},
	} {
		t.Run(c.name, func(t *testing.T) {
			w, te := windowDrawn(t, 600)
			if c.zoom != geom.ZoomNone {
				w.setZoom(c.zoom)
				te.Frame()
			}
			export, ok := te.Find(locale.PreviewExport())
			if !ok {
				t.Fatal("the preview has no way out of it")
			}
			wechat, ok := te.Find(locale.WechatStyle())
			if !ok {
				t.Fatal("the preview has no WeChat button")
			}
			read, ok := te.Find("head")
			if !ok {
				t.Fatal("the note is not read in the preview")
			}
			if export.Y != wechat.Y {
				t.Errorf("the way out is at y=%v, and WeChat at y=%v, "+
					"want one line", export.Y, wechat.Y)
			}
			if export.Y+export.H > headHeight ||
				wechat.Y+wechat.H > headHeight {
				t.Errorf("the buttons take y=%v..%v and %v..%v, "+
					"want them in the head of %v", export.Y,
					export.Y+export.H, wechat.Y, wechat.Y+wechat.H,
					headHeight)
			}
			if export.Y+export.H >= read.Y {
				t.Errorf("the way out ends at y=%v, and what is read "+
					"begins at y=%v", export.Y+export.H, read.Y)
			}
			if wechat.X+wechat.W >= export.X {
				t.Errorf("WeChat ends at x=%v, and the way out begins "+
					"at x=%v, want them side by side",
					wechat.X+wechat.W, export.X)
			}
		})
	}
}

// TestWorkspace_theWayOutStandsOnlyWhereAPreviewReads checks that the
// button goes away with the note it writes out, and with the pane it stands
// over: a file the app only shows has nothing to write out, and no preview
// has no head to hold the button.
func TestWorkspace_theWayOutStandsOnlyWhereAPreviewReads(t *testing.T) {
	t.Run("a file the app only shows", func(t *testing.T) {
		w, te := windowDrawn(t, 600)
		dir := t.TempDir()
		pict := writeFile(t, dir, "a.png", "not really a picture")
		if _, err := w.Tabs.Open(pict); err != nil {
			t.Fatalf("Open: %v", err)
		}
		te.Frame()
		te.Frame()
		if _, ok := te.Find(locale.PreviewExport()); ok {
			t.Error("a picture has a way out of it")
		}
		if _, ok := te.Find(locale.WechatStyle()); !ok {
			t.Error("the WeChat button went away with the way out")
		}
	})
	t.Run("the preview shut", func(t *testing.T) {
		w, te := windowDrawn(t, 600)
		w.SetPreviewShown(false)
		te.Frame()
		te.Frame()
		if _, ok := te.Find(locale.PreviewExport()); ok {
			t.Error("the way out is drawn with the preview shut")
		}
	})
}

// TestWorkspace_theWayOutMenuAsksForAFile checks that the two choices of
// the button still ask the app for the note in the kind that was chosen.
func TestWorkspace_theWayOutMenuAsksForAFile(t *testing.T) {
	for _, c := range []struct {
		name, item, kind string
	}{
		{"a page of HTML", locale.MenuExportPage(), ExportPage},
		{"a PDF", locale.MenuExportPdf(), ExportPDF},
	} {
		t.Run(c.name, func(t *testing.T) {
			w, te := windowDrawn(t, 600)
			asked := ""
			w.OnExport = func(kind string) { asked = kind }
			if err := te.Click(locale.PreviewExport()); err != nil {
				t.Fatalf("Click %q: %v", locale.PreviewExport(), err)
			}
			te.Frame()
			if err := te.ChooseMenuItem(c.item); err != nil {
				t.Fatalf("ChooseMenuItem %q: %v", c.item, err)
			}
			if asked != c.kind {
				t.Errorf("asked for %q, want %q", asked, c.kind)
			}
		})
	}
}

// TestWorkspace_aNarrowPreviewKeepsTheWayOutInTheOverflow checks a pane too
// narrow for the two buttons: they are not drawn over one another, but are
// both still to be found in the toolbar's overflow menu, where the way out
// still asks for a file.
func TestWorkspace_aNarrowPreviewKeepsTheWayOutInTheOverflow(t *testing.T) {
	w, te := windowDrawn(t, 600)
	w.Layout.Right = geom.MinPanelWidth
	asked := ""
	w.OnExport = func(kind string) { asked = kind }
	te.Frame()
	wantMenu := func(what string, want []string) {
		t.Helper()
		got := te.Menu()
		if len(got) != len(want) {
			t.Fatalf("the %s menu holds %q, want %q", what, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("the %s menu holds %q, want %q", what, got, want)
			}
		}
	}
	if _, ok := te.Find(locale.PreviewExport()); ok {
		t.Error("the way out is drawn over a pane too narrow for it")
	}
	if err := te.Click("More"); err != nil {
		t.Fatalf("Click %q: %v", "More", err)
	}
	te.Frame()
	wantMenu("overflow", []string{
		locale.WechatStyle(), locale.PreviewExport(),
	})
	if err := te.ChooseMenuItem(locale.PreviewExport()); err != nil {
		t.Fatalf("ChooseMenuItem %q: %v", locale.PreviewExport(), err)
	}
	te.Frame()
	wantMenu("export", []string{
		locale.MenuExportPage(), locale.MenuExportPdf(),
	})
	if err := te.ChooseMenuItem(locale.MenuExportPdf()); err != nil {
		t.Fatalf("ChooseMenuItem %q: %v", locale.MenuExportPdf(), err)
	}
	if asked != ExportPDF {
		t.Errorf("asked for %q, want %q", asked, ExportPDF)
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
