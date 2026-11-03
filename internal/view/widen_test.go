package view

import (
	"testing"

	"github.com/helightxu/xinkgo/internal/locale"
)

// TestWiden_theNoteStretchesOverThePreview checks what the button at the
// end of the tab bar is for: the note is drawn over the preview, out to
// the window's right edge, and the tree keeps the room it was given.
func TestWiden_theNoteStretchesOverThePreview(t *testing.T) {
	const width = 900
	w, tt := zoomTester(t, width)
	before := w.Layout
	ed, ok := tt.Find(editorLabel)
	if !ok {
		t.Fatal("the note is not drawn")
	}
	if _, ok := tt.Find("head"); !ok {
		t.Fatal("the note is not read in the preview")
	}
	if err := tt.Click(locale.WidenNote()); err != nil {
		t.Fatalf("Click: %v", err)
	}
	tt.Advance(widenAnim.Duration)
	tt.Frame()
	if !w.stretched {
		t.Fatal("the button did not widen the note")
	}
	// What the panels were given is kept: the note is stretched over the
	// preview, and not given its room instead.
	if w.Layout != before {
		t.Fatalf("layout = %+v, want the %+v it was", w.Layout, before)
	}
	if _, ok := tt.Find("head"); ok {
		t.Error("the preview is still drawn, want it under the note")
	}
	wide, ok := tt.Find(editorLabel)
	if !ok {
		t.Fatal("the note went away with the preview")
	}
	// The tree keeps its room: the note begins where it began, and it is
	// its right edge that has gone out over the preview.
	if wide.X != ed.X {
		t.Errorf("the note begins at %v, want the %v it began at",
			wide.X, ed.X)
	}
	if wide.W <= ed.W {
		t.Errorf("the note is %v wide, want it wider than the %v it was",
			wide.W, ed.W)
	}
	// It reaches the rail: the pane ends where the rail's strip begins,
	// and the text the room the user chose inside that, less what the
	// editor keeps beside its lines of its own.
	if end := wide.X + wide.W; end < float32(width)-railStrip-
		w.editorPadRight()-1 {
		t.Errorf("the note ends at %v, want it at the rail's strip, "+
			"but for the padding", end)
	}
}

// TestWiden_theNoteStretchesRatherThanJumps checks that the note is seen
// to go over the preview, and not to be over it between two frames: the
// room it takes grows over the animation's own time, and only once the
// window's clock has gone on.
func TestWiden_theNoteStretchesRatherThanJumps(t *testing.T) {
	w, tt := zoomTester(t, 900)
	ed, ok := tt.Find(editorLabel)
	if !ok {
		t.Fatal("the note is not drawn")
	}
	if err := tt.Click(locale.WidenNote()); err != nil {
		t.Fatalf("Click: %v", err)
	}
	// The click is taken in, but the window's clock has not gone on:
	// nothing has moved yet.
	tt.Frame()
	if now, _ := tt.Find(editorLabel); now.W != ed.W {
		t.Fatalf("the note is %v wide at once, want the %v it was",
			now.W, ed.W)
	}
	tt.Advance(widenAnim.Duration / 2)
	tt.Frame()
	half, ok := tt.Find(editorLabel)
	if !ok {
		t.Fatal("the note is gone from the window")
	}
	if half.W <= ed.W || half.W >= ed.W+w.Layout.Right {
		t.Errorf("half way the note is %v wide, want it between the %v "+
			"it was and the %v of the preview's room",
			half.W, ed.W, w.Layout.Right)
	}
	tt.Advance(widenAnim.Duration)
	tt.Frame()
	over, _ := tt.Find(editorLabel)
	if over.W <= half.W {
		t.Errorf("the note stopped at %v wide, want it wider than the "+
			"%v it was half way", over.W, half.W)
	}
	if _, ok := tt.Find("head"); ok {
		t.Error("the preview is still drawn, want it under the note")
	}
}

// TestWiden_pressedAgainGivesThePreviewBack checks that the button of a
// note standing over the preview is the way out of it: pressed again, the
// preview comes back as wide as it was.
func TestWiden_pressedAgainGivesThePreviewBack(t *testing.T) {
	w, tt := zoomTester(t, 900)
	before := w.Layout
	ed, ok := tt.Find(editorLabel)
	if !ok {
		t.Fatal("the note is not drawn")
	}
	for _, label := range []string{locale.WidenNote(), locale.WidenBack()} {
		if err := tt.Click(label); err != nil {
			t.Fatalf("Click %q: %v", label, err)
		}
		tt.Advance(widenAnim.Duration)
		tt.Frame()
	}
	if w.stretched {
		t.Fatal("the note still stands over the preview")
	}
	if w.Layout != before {
		t.Fatalf("layout = %+v, want the %+v it was", w.Layout, before)
	}
	if _, ok := tt.Find("head"); !ok {
		t.Error("the preview did not come back")
	}
	back, ok := tt.Find(editorLabel)
	if !ok {
		t.Fatal("the note is gone from the window")
	}
	if back.W != ed.W {
		t.Errorf("the note is %v wide, want the %v it was", back.W, ed.W)
	}
}

// TestWiden_theButtonStandsWhereThereIsAPreview checks that the button is
// drawn only where there is a room of the preview's to stretch over: it is
// not drawn with the preview shut, and a preview shut under a note
// standing over it puts the stretch away with it.
func TestWiden_theButtonStandsWhereThereIsAPreview(t *testing.T) {
	w, tt := zoomTester(t, 900)
	w.SetPreviewShown(false)
	tt.Frame()
	if _, ok := tt.Find(locale.WidenNote()); ok {
		t.Error("the button is drawn with the preview shut")
	}
	w.SetPreviewShown(true)
	tt.Frame()
	if err := tt.Click(locale.WidenNote()); err != nil {
		t.Fatalf("Click: %v", err)
	}
	if !w.stretched {
		t.Fatal("the button did not widen the note")
	}
	w.SetPreviewShown(false)
	if w.stretched {
		t.Error("the note still stands over a preview that is shut")
	}
}

// cornerDivider is how the line that parts the note from the preview is
// named to the window: a test looks for it to say where the line stands.
const cornerDivider = "divider-preview"

// TestWiden_theTwoCornerButtonsStandLevel checks that the button at the
// end of the tab bar and the one in the preview's own corner are drawn on
// one line: both stand at the top of a pane, and the two panes begin at
// the top of the window together, so a button standing lower than the
// other is read as a step across the middle of the window.
func TestWiden_theTwoCornerButtonsStandLevel(t *testing.T) {
	_, tt := zoomTester(t, 1100)
	note, ok := tt.Find(locale.WidenNote())
	if !ok {
		t.Fatal("the button that widens the note is not drawn")
	}
	preview, ok := tt.Find(locale.SpreadPreview())
	if !ok {
		t.Fatal("the button that spreads the preview is not drawn")
	}
	noteMid := note.Y + note.H/2
	previewMid := preview.Y + preview.H/2
	if d := noteMid - previewMid; d < -1 || d > 1 {
		t.Errorf("the note's button stands at %v and the preview's at %v,"+
			" want them level", noteMid, previewMid)
	}
}

// TestWiden_theTwoCornerButtonsStandAsFarFromTheLine checks that the two
// buttons keep the same room either side of the line that parts the note
// from the preview: the pane of each ends at the line, whose hair is
// drawn down the middle of its grip, so a button standing nearer the
// line than the other is read as a step in the row the two make.
func TestWiden_theTwoCornerButtonsStandAsFarFromTheLine(t *testing.T) {
	_, tt := zoomTester(t, 1100)
	line, ok := tt.Find(cornerDivider)
	if !ok {
		t.Fatal("the line that parts the note from the preview is not drawn")
	}
	note, ok := tt.Find(locale.WidenNote())
	if !ok {
		t.Fatal("the button that widens the note is not drawn")
	}
	preview, ok := tt.Find(locale.SpreadPreview())
	if !ok {
		t.Fatal("the button that spreads the preview is not drawn")
	}
	middle := line.X + line.W/2
	left := middle - (note.X + note.W)
	right := preview.X - middle
	if d := left - right; d < -1 || d > 1 {
		t.Errorf("the note's button stands %v from the line and the"+
			" preview's %v, want them as far from it as one another",
			left, right)
	}
}
