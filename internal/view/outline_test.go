package view

import (
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
