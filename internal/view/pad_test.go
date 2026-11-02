package view

import (
	"testing"

	"github.com/helightxu/xinkgo/internal/config"
)

// TestPad_theNoteStandsAsFarFromOneSideAsTheOther checks what the padding
// of the note is for: its text stands the room the user chose from the
// left edge of its pane, and the very same room from the right one. The
// editor keeps a room of its own beside its lines, wider on one side than
// on the other, and that room is taken out of what the pane is given, so
// that it is the text, and not the editor, that is being measured.
//
// The tree and the preview stand down for the measurement, so that the
// note's pane is the window itself and where it begins and ends is not in
// question.
func TestPad_theNoteStandsAsFarFromOneSideAsTheOther(t *testing.T) {
	const width = 900
	w, tt := zoomTester(t, width)
	w.Layout.Left, w.Layout.Right = 0, 0
	for _, pad := range []uint32{16, 32, config.MaxPadding} {
		w.Config.Editor.PaddingX = pad
		tt.Frame()
		ed, ok := tt.Find(editorLabel)
		if !ok {
			t.Fatalf("with a padding of %d, the note is not drawn", pad)
		}
		// Where the note's text stands: the room the pane was given,
		// and what the editor keeps of its own beside the lines.
		for side, got := range map[string]float32{
			"left":  ed.X + editorInsetLeft,
			"right": width - (ed.X + ed.W) + editorInsetRight,
		} {
			if got != float32(pad) {
				t.Errorf("with a padding of %d, the note stands %.0f "+
					"from the %s edge of its pane", pad, got, side)
			}
		}
	}
}

// TestPad_aPaneGivenLessThanTheEditorKeepsKeepsNone checks the end of the
// range: a padding narrower than the room the editor keeps of its own is
// given to the pane as no room at all, for there is none to give — the
// text stands as near the edges as the editor itself puts it.
func TestPad_aPaneGivenLessThanTheEditorKeepsKeepsNone(t *testing.T) {
	const width = 900
	w, tt := zoomTester(t, width)
	w.Layout.Left, w.Layout.Right = 0, 0
	w.Config.Editor.PaddingX = config.MinPadding
	tt.Frame()
	ed, ok := tt.Find(editorLabel)
	if !ok {
		t.Fatal("the note is not drawn")
	}
	if ed.X != 0 {
		t.Errorf("the note's pane keeps %.0f beside the editor, want "+
			"none: the padding is narrower than what the editor keeps",
			ed.X)
	}
	if got := width - (ed.X + ed.W); got != 0 {
		t.Errorf("the note's pane keeps %.0f on the other side, want none",
			got)
	}
}

// TestPad_theRoomTheNoteIsGivenIsTheRoomItKeeps checks that the padding is
// what decides where the note stands, and that it is given to the two
// sides alike: more of it brings the note in from both edges at once, and
// by as much on one as on the other.
func TestPad_theRoomTheNoteIsGivenIsTheRoomItKeeps(t *testing.T) {
	w, tt := zoomTester(t, 900)
	w.Config.Editor.PaddingX = 16
	tt.Frame()
	near, ok := tt.Find(editorLabel)
	if !ok {
		t.Fatal("the note is not drawn")
	}
	w.Config.Editor.PaddingX = 40
	tt.Frame()
	far, ok := tt.Find(editorLabel)
	if !ok {
		t.Fatal("the note is not drawn")
	}
	if got := far.X - near.X; got != 24 {
		t.Errorf("the note came in %v on the left, want the 24 of the "+
			"padding it was given more of", got)
	}
	if got := (near.X + near.W) - (far.X + far.W); got != 24 {
		t.Errorf("the note came in %v on the right, want 24 alike", got)
	}
}

// TestPad_whatIsReadStandsOffTheEdgesOfItsPane checks the preview's own:
// the room the user chose on either side of what is read, and the room
// over its top, which is a choice of its own. The preview is given the
// whole window for the measurement, as the note's pane is in the other
// test, so that where its pane begins is not in question.
func TestPad_whatIsReadStandsOffTheEdgesOfItsPane(t *testing.T) {
	w, tt := zoomTester(t, 900)
	w.setZoom(zoomPreview)
	for _, pad := range []uint32{config.MinPadding, 16, 40} {
		w.Config.Preview.PaddingX = pad
		tt.Frame()
		text, ok := tt.Find("head")
		if !ok {
			t.Fatalf("with a padding of %d, the note is not read", pad)
		}
		if got := text.X; got != float32(pad) {
			t.Errorf("with a padding of %d, what is read stands %.0f "+
				"from the side of its pane", pad, got)
		}
	}
	// The room over the top of it is a choice of its own: what is read
	// stands that much lower down. A preview given the window keeps a row
	// of its own above it, which is the way out of a pane the note's
	// toolbar is not drawn in, so it is the room that grows that is
	// measured, and not where it begins.
	w.Config.Preview.PaddingY = 12
	tt.Frame()
	near, ok := tt.Find("head")
	if !ok {
		t.Fatal("the note is not read")
	}
	w.Config.Preview.PaddingY = 36
	tt.Frame()
	far, ok := tt.Find("head")
	if !ok {
		t.Fatal("the note is not read")
	}
	if got := far.Y - near.Y; got != 24 {
		t.Errorf("what is read came down %v, want the 24 of the padding "+
			"it was given more of", got)
	}
}
