package view

import (
	"os"
	"path/filepath"
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

// TestToolbar_theWayToTheModelStandsAtItsEnd checks where the button that
// opens the conversation is: at the end of the row that writes Markdown,
// outside the row itself, so that a window too narrow for every button
// does not hide it in the menu at the row's end.
func TestToolbar_theWayToTheModelStandsAtItsEnd(t *testing.T) {
	for _, width := range []int{220, 320, 520, 1100} {
		tt := toolbarTester(t, width)
		bar, ok := tt.Find("Markdown")
		if !ok {
			t.Fatalf("no toolbar in a window %d wide", width)
		}
		ai, ok := tt.Find("AI 辅助")
		if !ok {
			t.Fatalf("no way to the model in a window %d wide: %v",
				width, tt.Texts())
		}
		if ai.X < bar.X+bar.W {
			t.Errorf("at %d wide the button stands at %v, want it after "+
				"the row that ends at %v", width, ai.X, bar.X+bar.W)
		}
		if end := ai.X + ai.W; end > float32(width) {
			t.Errorf("at %d wide the button ends at %v, want it in the "+
				"window", width, end)
		}
	}
}

// TestToolbar_theWayToTheModelIsAPicture checks that the button is drawn
// as the others are: a picture, and not the word it stands for.
func TestToolbar_theWayToTheModelIsAPicture(t *testing.T) {
	tt := toolbarTester(t, 1100)
	ai, ok := tt.Find("AI 辅助")
	if !ok {
		t.Fatalf("no way to the model: %v", tt.Texts())
	}
	if ai.W > 3*iconSize {
		t.Errorf("the button is %v wide, want no more than %v", ai.W,
			3*iconSize)
	}
}

// TestToolbar_clickingItOpensTheConversation checks what a click on it
// does: the conversation under the note is opened, and clicking again
// hides it, which is all the button is for.
func TestToolbar_clickingItOpensTheConversation(t *testing.T) {
	_, a, _ := vaultWithNotes(t)
	w := NewWorkspace(config.Default())
	if _, err := w.Tabs.Open(a); err != nil {
		t.Fatalf("Open: %v", err)
	}
	w.Layout.Left, w.Layout.Right = 0, 0
	tt := ui.NewTester(w.View, 1100, 320)
	ai, ok := tt.Find("AI 辅助")
	if !ok {
		t.Fatalf("no way to the model: %v", tt.Texts())
	}
	click := func() {
		tt.ClickAt(ai.X+ai.W/2, ai.Y+ai.H/2)
		tt.Frame()
	}
	click()
	if !w.aiOpen(a) {
		t.Fatalf("the conversation did not open")
	}
	click()
	if w.aiOpen(a) {
		t.Errorf("the conversation is still open, want it hidden")
	}
}

// TestToolbar_aFileOfDataHasNoUseOfIt checks a file the toolbar says the
// kind of, rather than writing Markdown into: there is no way to the
// model beside it, for there is nothing to ask about.
func TestToolbar_aFileOfDataHasNoUseOfIt(t *testing.T) {
	root, _, _ := vaultWithNotes(t)
	data := filepath.Join(root, "d.json")
	if err := os.WriteFile(data, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	w := NewWorkspace(config.Default())
	if _, err := w.Tabs.Open(data); err != nil {
		t.Fatalf("Open: %v", err)
	}
	w.Layout.Left, w.Layout.Right = 0, 0
	tt := ui.NewTester(w.View, 1100, 320)
	if _, ok := tt.Find("AI 辅助"); ok {
		t.Errorf("a file of data is offered the model: %v", tt.Texts())
	}
}

// TestToolbar_theButtonStandsClearOfTheRail checks the case the others
// leave out: the panels open, when the room the note has is the window's
// less the tree, the preview and the rail that hangs over them. The
// button is made to keep clear of the rail, and not to stand under it.
func TestToolbar_theButtonStandsClearOfTheRail(t *testing.T) {
	_, a, _ := vaultWithNotes(t)
	w := NewWorkspace(config.Default())
	if _, err := w.Tabs.Open(a); err != nil {
		t.Fatalf("Open: %v", err)
	}
	tt := ui.NewTester(w.View, 1000, 600)
	ai, ok := tt.Find("AI 辅助")
	if !ok {
		t.Fatalf("no way to the model: %v", tt.Texts())
	}
	rail, ok := tt.Find("大纲")
	if !ok {
		t.Fatalf("no rail in the window: %v", tt.Texts())
	}
	if end := ai.X + ai.W; end > rail.X {
		t.Errorf("the button ends at %v, under the rail that starts at %v",
			end, rail.X)
	}
}
