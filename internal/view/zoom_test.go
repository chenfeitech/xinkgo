package view

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
)

// editorLabel is how the editor of the note is named to the window: what
// a test looks for to tell whether the note is drawn at all, the editor
// being a thing of the machine's own whose text is not read back.
const editorLabel = "Code editor"

// zoomTester is a window with one note open in it, drawn as wide as told,
// with the tree and the preview standing beside the note. The note is
// written so that what the note says and what the preview reads of it are
// two different things to look for: the mark of the heading is the
// editor's, and the heading without it is the preview's.
func zoomTester(t *testing.T, width int) (*Workspace, *ui.Tester) {
	t.Helper()
	_, a, _ := vaultWithNotes(t)
	w := NewWorkspace(config.Default())
	n, err := w.Tabs.Open(a)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	n.Editor.Buffer.Set("# head\n\nsome words\n")
	return w, ui.NewTester(w.View, width, 320)
}

// TestZoom_theNoteTakesTheWholeWindow checks what the rail's button of
// the note is for: the tree and the preview stand down, and the note is
// drawn as wide as the window.
func TestZoom_theNoteTakesTheWholeWindow(t *testing.T) {
	w, tt := zoomTester(t, 900)
	before := w.Layout
	if err := tt.Click(locale.ZoomEditor()); err != nil {
		t.Fatalf("Click: %v", err)
	}
	if w.zoom != zoomEditor {
		t.Fatalf("zoom = %v, want the note given the window", w.zoom)
	}
	// What the panels were given is kept: the note is given the window
	// over them, and not instead of their room.
	if w.Layout != before {
		t.Fatalf("layout = %+v, want the %+v it was", w.Layout, before)
	}
	// The note begins at the window's own edge now: the tree is standing
	// down, and is not drawn over the note.
	bar, ok := tt.Find(locale.ToolbarMarkdown())
	if !ok {
		t.Fatal("the toolbar is gone, and with it the note's buttons")
	}
	if bar.X > 40 {
		t.Errorf("the toolbar begins at %v, want it at the window's edge",
			bar.X)
	}
	// The preview is not drawn, and neither is the bar that is its way
	// back, which is only ever drawn with it.
	if _, ok := tt.Find("head"); ok {
		t.Error("the preview is still drawn, want it standing down")
	}
	if _, ok := tt.Find(locale.ZoomBack()); ok {
		t.Error("the preview's bar is drawn without the preview")
	}
}

// TestZoom_thePreviewTakesTheWholeWindow checks the rail's button of the
// preview: the note's own pane stands down, and the preview is read as
// wide as the window.
func TestZoom_thePreviewTakesTheWholeWindow(t *testing.T) {
	w, tt := zoomTester(t, 900)
	before := w.Layout
	if err := tt.Click(locale.ZoomPreview()); err != nil {
		t.Fatalf("Click: %v", err)
	}
	if w.zoom != zoomPreview {
		t.Fatalf("zoom = %v, want the preview given the window", w.zoom)
	}
	if w.Layout != before {
		t.Fatalf("layout = %+v, want the %+v it was", w.Layout, before)
	}
	// The note is not drawn: what is read is the preview's, and the
	// editor of the note is standing down behind it.
	if _, ok := tt.Find(editorLabel); ok {
		t.Error("the note is still drawn, want it standing down")
	}
	text, ok := tt.Find("head")
	if !ok {
		t.Fatal("the note is not read in the preview")
	}
	if text.X > 100 {
		t.Errorf("the preview is read from %v, want it from the edge",
			text.X)
	}
	// The preview keeps a button of its own, which is the way out of a
	// pane the note's toolbar is not drawn in.
	if _, ok := tt.Find(locale.ZoomBack()); !ok {
		t.Fatal("the preview's way back is not drawn")
	}
}

// TestZoom_pressedAgainGivesTheRoomBack checks that the button of the
// pane that has the window is the way out of it: pressed again, every
// panel comes back as it was.
func TestZoom_pressedAgainGivesTheRoomBack(t *testing.T) {
	w, tt := zoomTester(t, 900)
	before := w.Layout
	for _, label := range []string{
		locale.ZoomEditor(), locale.ZoomEditor(),
		locale.ZoomPreview(), locale.ZoomPreview(),
	} {
		if err := tt.Click(label); err != nil {
			t.Fatalf("Click %q: %v", label, err)
		}
	}
	if w.zoom != zoomNone {
		t.Fatalf("zoom = %v, want the panels as they were", w.zoom)
	}
	if w.Layout != before {
		t.Fatalf("layout = %+v, want the %+v it was", w.Layout, before)
	}
	if _, ok := tt.Find(editorLabel); !ok {
		t.Error("the note did not come back")
	}
	if _, ok := tt.Find("head"); !ok {
		t.Error("the preview did not come back")
	}
}

// TestZoom_onePaneAtATime checks that giving one pane the window takes it
// from the other: two panes given the whole window is no shape at all.
func TestZoom_onePaneAtATime(t *testing.T) {
	w, _ := zoomTester(t, 900)
	w.setZoom(zoomEditor)
	w.setZoom(zoomPreview)
	if w.zoom != zoomPreview {
		t.Fatalf("zoom = %v, want the preview, and not both", w.zoom)
	}
}

// TestZoom_theButtonSaysWhatItWillDo checks that the button of the pane
// that has the window says the room will be given back, and not that it
// will be taken again.
func TestZoom_theButtonSaysWhatItWillDo(t *testing.T) {
	w, _ := zoomTester(t, 900)
	label := locale.ZoomEditorTooltip()
	if got := w.zoomTip(zoomEditor, label); got != label {
		t.Fatalf("tip = %q, want %q", got, label)
	}
	w.setZoom(zoomEditor)
	if got := w.zoomTip(zoomEditor, label); got != locale.ZoomBackTooltip() {
		t.Fatalf("tip = %q, want the way back", got)
	}
}
