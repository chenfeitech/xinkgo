package view

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	xmd "github.com/helightxu/xinkgo/internal/markdown"
)

// toolbarTester is a window with one note open, drawn as wide as told,
// with no panel beside the note.
func toolbarTester(t *testing.T, width int) *ui.Tester {
	t.Helper()
	_, a, _ := vaultWithNotes(t)
	w := NewWorkspace(config.Default())
	if _, err := w.Tabs.Open(a); err != nil {
		t.Fatalf("Open: %v", err)
	}
	w.Layout.Left, w.Layout.Right = 0, 0
	return ui.NewTester(w.View, width, 320)
}

// TestToolbar_itStaysAsWideAsTheNote checks that the row of buttons never
// runs past the note it writes in, however narrow the window is made: what
// there is no room for goes into the menu at its end.
func TestToolbar_itStaysAsWideAsTheNote(t *testing.T) {
	for _, width := range []int{220, 320, 520, 1100} {
		tt := toolbarTester(t, width)
		bar, ok := tt.Find("Markdown")
		if !ok {
			t.Fatalf("no toolbar in a window %d wide", width)
		}
		if end := bar.X + bar.W; end > float32(width) {
			t.Errorf("at %d wide the toolbar ends at %v, want it in the window",
				width, end)
		}
	}
}

// TestToolbar_aButtonIsAPicture checks that a button is drawn as a
// picture, and not as the word it stands for.
func TestToolbar_aButtonIsAPicture(t *testing.T) {
	tt := toolbarTester(t, 1100)
	for _, f := range xmd.Formats {
		button, ok := tt.Find(f.Label())
		if !ok {
			t.Fatalf("no button for %q", f.Label())
		}
		// An icon, and the room around it: nothing a word would need.
		if button.W > 3*iconSize {
			t.Errorf("the button %q is %v wide, want no more than %v",
				f.Label(), button.W, 3*iconSize)
		}
	}
}

// TestToolbar_everyFormatHasItsOwnPicture checks that no two buttons are
// drawn by the same picture, which would say the same thing twice.
func TestToolbar_everyFormatHasItsOwnPicture(t *testing.T) {
	seen := map[*ui.SVG]xmd.Format{}
	for _, f := range xmd.Formats {
		picture := formatIcon(f)
		if picture == nil {
			t.Fatalf("format %q has no picture", f.Label())
		}
		if other, ok := seen[picture]; ok {
			t.Errorf("%q and %q are drawn by one picture",
				other.Label(), f.Label())
		}
		seen[picture] = f
	}
}

// TestToolbar_whatAButtonDoesIsSaidWhenThePointerRestsOnIt checks the
// other half of a picture: the words come when asked for, as a tip, and
// not as the face of the button.
func TestToolbar_whatAButtonDoesIsSaidWhenThePointerRestsOnIt(t *testing.T) {
	tt := toolbarTester(t, 800)
	r, ok := tt.Find(xmd.Bold.Label())
	if !ok {
		t.Fatalf("no button for %q", xmd.Bold.Label())
	}
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	// A tip comes when the pointer has rested, and not before.
	tt.Advance(time.Second)
	tt.Frame()
	if !tt.HasText(xmd.Bold.Desc()) {
		t.Errorf("texts = %q, want the tip %q in them",
			tt.Texts(), xmd.Bold.Desc())
	}
}

// TestToolbar_pressingAPictureWritesTheMarks checks that a button drawn
// as a picture does what it did when it was a word.
func TestToolbar_pressingAPictureWritesTheMarks(t *testing.T) {
	_, a, _ := vaultWithNotes(t)
	w := NewWorkspace(config.Default())
	n, err := w.Tabs.Open(a)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	n.Editor.Buffer.Set("hello world")
	n.Editor.Select(6, 11)
	tt := ui.NewTester(w.View, 800, 400)
	if err := tt.Click(xmd.Bold.Label()); err != nil {
		t.Fatalf("Click: %v", err)
	}
	if got := n.Editor.Buffer.String(); got != "hello **world**" {
		t.Errorf("note = %q, want %q", got, "hello **world**")
	}
}
