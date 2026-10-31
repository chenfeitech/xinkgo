package view

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// railOf is where the rail's buttons stand in a window of the width given,
// with the preview as wide as the preview says.
func railOf(t *testing.T, width, preview float32) ui.Rect {
	t.Helper()
	w, _ := workspaceShown(t)
	w.Layout.Right = preview
	tt := ui.NewTester(w.View, int(width), 600)
	r, ok := tt.Find("大纲")
	if !ok {
		t.Fatalf("no rail button in the window")
	}
	return r
}

// TestRailView_theRailStandsAtTheWindowEdge checks where the buttons hang:
// at the right edge of the window, and not at the right end of the note,
// which is where the preview begins.
func TestRailView_theRailStandsAtTheWindowEdge(t *testing.T) {
	const width, preview = float32(1000), float32(320)
	r := railOf(t, width, preview)
	// The buttons are the window's, so they stand at its edge however
	// wide the panels are: 320 in from it is where the note ends.
	if end := r.X + r.W; end < width-40 || end > width {
		t.Fatalf("the rail ends at %v, want it at the window's own edge %v",
			end, width)
	}
}

// TestRailView_theRailStaysPutWhenThePreviewIsShut checks that shutting the
// preview, which widens the note, does not move the buttons: they are not
// the note's to move.
func TestRailView_theRailStaysPutWhenThePreviewIsShut(t *testing.T) {
	const width = float32(1000)
	open := railOf(t, width, 320)
	shut := railOf(t, width, 0)
	if open.X != shut.X {
		t.Fatalf("the rail moved from %v to %v when the preview was shut",
			shut.X, open.X)
	}
}

// drawerStrip is how many of the pixels of a thin strip inside where the
// drawer stands are not the face of the window: the drawer's own face,
// once it is open, hides the note behind it, and the strip is then all
// one color, while the note behind it is not.
func drawerStrip(t *testing.T, open bool) int {
	t.Helper()
	const width, preview = float32(1000), float32(320)
	w, _ := workspaceShown(t)
	w.Layout.Right = preview
	// A note long enough that the preview fills the panel with lines of
	// text, so that there is something behind the drawer to hide.
	w.Tabs.ActiveNote().Editor.Buffer.Set("# a heading\n\n" +
		strings.Repeat("word ", 3000))
	if open {
		w.drawer = drawerOutline
	}
	tt := ui.NewTester(w.View, int(width), 600)
	tt.Frame()
	tt.Frame()
	img := tt.Image()
	face := ui.LightTheme().Background
	rail, ok := tt.Find("大纲")
	if !ok {
		t.Fatalf("no rail button in the window")
	}
	// The strip stands well inside the drawer: near its right edge, where
	// nothing of what it says reaches, as what it says reads from the
	// left, and over the height it was given beside the rail.
	other := 0
	for y := int(rail.Y) + 30; y <= int(rail.Y)+60; y++ {
		r, g, b, _ := img.At(int(rail.X)-16, y).RGBA()
		if int(r>>8) != int(face.R) || int(g>>8) != int(face.G) ||
			int(b>>8) != int(face.B) {
			other++
		}
	}
	return other
}

// TestDrawerPanel_theDrawerStandsOnItsOwnFace checks that a drawer open
// over the note is read on its own face, and not on the note it covers.
func TestDrawerPanel_theDrawerStandsOnItsOwnFace(t *testing.T) {
	if behind := drawerStrip(t, false); behind == 0 {
		t.Fatalf("the note behind the drawer shows nothing, so the test " +
			"would pass with a drawer of no face at all")
	}
	if over := drawerStrip(t, true); over != 0 {
		t.Errorf("%d of the drawer's pixels are the note behind it, "+
			"want none of them", over)
	}
}
