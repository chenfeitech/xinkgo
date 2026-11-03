package view

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/view/layout"
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
	if w.zoom != layout.ZoomEditor {
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
	if w.zoom != layout.ZoomPreview {
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
	if w.zoom != layout.ZoomNone {
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
	w.setZoom(layout.ZoomEditor)
	w.setZoom(layout.ZoomPreview)
	if w.zoom != layout.ZoomPreview {
		t.Fatalf("zoom = %v, want the preview, and not both", w.zoom)
	}
}

// TestZoom_theButtonSaysWhatItWillDo checks that the button of the pane
// that has the window says the room will be given back, and not that it
// will be taken again.
func TestZoom_theButtonSaysWhatItWillDo(t *testing.T) {
	w, _ := zoomTester(t, 900)
	label := locale.ZoomEditorTooltip()
	if got := w.zoomTip(layout.ZoomEditor, label); got != label {
		t.Fatalf("tip = %q, want %q", got, label)
	}
	w.setZoom(layout.ZoomEditor)
	if got := w.zoomTip(layout.ZoomEditor, label); got != locale.ZoomBackTooltip() {
		t.Fatalf("tip = %q, want the way back", got)
	}
}

// spreadTester is a window with one note open in it, the tree on its left
// and the preview on its right, drawn as wide as told. The note is written
// so that what the note says and what the preview reads of it are two
// different things to look for.
func spreadTester(t *testing.T, width int) (*Workspace, *ui.Tester) {
	t.Helper()
	_, a, _ := vaultWithNotes(t)
	w := NewWorkspace(config.Default())
	w.Tree.SetRoot(treeRoot())
	n, err := w.Tabs.Open(a)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	n.Editor.Buffer.Set("# head\n\nsome words\n")
	return w, ui.NewTester(w.View, width, 320)
}

// TestZoom_thePreviewSpreadsOverTheNote checks the button in the corner of
// the preview's pane: the preview reaches left over the note, and the tree
// keeps the room it was given, so that what is read is read beside the
// tree and not over it.
func TestZoom_thePreviewSpreadsOverTheNote(t *testing.T) {
	w, tt := spreadTester(t, 1100)
	before := w.Layout
	read, ok := tt.Find("head")
	if !ok {
		t.Fatal("the note is not read in the preview")
	}
	if err := tt.Click(locale.SpreadPreview()); err != nil {
		t.Fatalf("Click: %v", err)
	}
	if w.zoom != layout.ZoomPreviewSpread {
		t.Fatalf("zoom = %v, want the preview spread over the note", w.zoom)
	}
	// What the panels were given is kept here too: the preview is given
	// the note's room over it, and not instead of its own.
	if w.Layout != before {
		t.Fatalf("layout = %+v, want the %+v it was", w.Layout, before)
	}
	// The note is not drawn: what is read stands over the room it had.
	if _, ok := tt.Find(editorLabel); ok {
		t.Error("the note is still drawn, want it standing down")
	}
	// The tree is the point of spreading rather than taking the window:
	// it keeps its place, so that the note to read next is chosen in it
	// while this one is read.
	if _, ok := tt.Find("a.md"); !ok {
		t.Error("the tree is not drawn, want it where it stood")
	}
	// What is read begins beside the tree now, and not at the window's
	// own end.
	spread, ok := tt.Find("head")
	if !ok {
		t.Fatal("the note is not read in the preview")
	}
	if spread.X >= read.X {
		t.Errorf("what is read begins at %v, want it left of the %v it was",
			spread.X, read.X)
	}
	want := float32(w.Layout.Left) + layout.DividerGrip + w.geometry().PreviewPadX()
	if d := spread.X - want; d < -8 || d > 8 {
		t.Errorf("what is read begins at %v, want it beside the tree at %v",
			spread.X, want)
	}
}

// TestZoom_theSpreadButtonGivesTheRoomBack checks that the button of the
// corner is the way out of a spread preview, as the buttons of the rail
// are: pressed again, the note comes back as it was.
func TestZoom_theSpreadButtonGivesTheRoomBack(t *testing.T) {
	w, tt := spreadTester(t, 1100)
	before := w.Layout
	if err := tt.Click(locale.SpreadPreview()); err != nil {
		t.Fatalf("Click: %v", err)
	}
	// What the press will do is what the button says now: the note's room
	// given back.
	if err := tt.Click(locale.SpreadPreviewBack()); err != nil {
		t.Fatalf("Click: %v", err)
	}
	if w.zoom != layout.ZoomNone {
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

// TestZoom_theCornerButtonStandsWithItsPane checks that the button of the
// corner is drawn with the pane it spreads, and not without it: not when
// the preview is shut, and not when the preview has the whole window,
// which keeps a bar of its own.
func TestZoom_theCornerButtonStandsWithItsPane(t *testing.T) {
	w, tt := spreadTester(t, 1100)
	w.SetPreviewShown(false)
	tt.Frame()
	if _, ok := tt.Find(locale.SpreadPreview()); ok {
		t.Error("the button is drawn with the preview shut")
	}
	w.SetPreviewShown(true)
	w.setZoom(layout.ZoomPreview)
	tt.Frame()
	if _, ok := tt.Find(locale.SpreadPreview()); ok {
		t.Error("the button of the corner is drawn over the bar of the" +
			" whole window")
	}
	if _, ok := tt.Find(locale.ZoomBack()); !ok {
		t.Error("the bar of the whole window is not drawn")
	}
}

// TestZoom_theSpreadButtonSaysWhatItWillDo checks that the button of the
// corner is named for the press it will give: the room taken while the
// preview stands in its pane, and the room given back once it has it.
func TestZoom_theSpreadButtonSaysWhatItWillDo(t *testing.T) {
	w, tt := spreadTester(t, 1100)
	if _, ok := tt.Find(locale.SpreadPreview()); !ok {
		t.Fatal("the button does not say what it will do")
	}
	if err := tt.Click(locale.SpreadPreview()); err != nil {
		t.Fatalf("Click: %v", err)
	}
	if _, ok := tt.Find(locale.SpreadPreview()); ok {
		t.Error("the button still says it will spread the preview, want" +
			" the way back")
	}
	if _, ok := tt.Find(locale.SpreadPreviewBack()); !ok {
		t.Fatal("the button does not say the way back")
	}
	if w.zoom != layout.ZoomPreviewSpread {
		t.Fatalf("zoom = %v, want the preview spread", w.zoom)
	}
}

// TestZoom_theSpreadIsOneShapeAtATime checks that spreading the preview
// and giving the note the whole window cannot both stand: the one takes
// the room from the other.
func TestZoom_theSpreadIsOneShapeAtATime(t *testing.T) {
	w, _ := spreadTester(t, 1100)
	w.setZoom(layout.ZoomPreviewSpread)
	w.setZoom(layout.ZoomEditor)
	if w.zoom != layout.ZoomEditor {
		t.Fatalf("zoom = %v, want the note, and not both", w.zoom)
	}
}

// TestZoom_aSpreadPreviewShutGivesTheRoomBack checks that a preview shut
// while it is spread over the note does not stay over it: there is no pane
// left to spread, so the note comes back.
func TestZoom_aSpreadPreviewShutGivesTheRoomBack(t *testing.T) {
	w, tt := spreadTester(t, 1100)
	w.setZoom(layout.ZoomPreviewSpread)
	w.SetPreviewShown(false)
	tt.Frame()
	if w.zoom != layout.ZoomNone {
		t.Fatalf("zoom = %v, want the note back", w.zoom)
	}
	if _, ok := tt.Find(editorLabel); !ok {
		t.Error("the note did not come back")
	}
	if _, ok := tt.Find("head"); ok {
		t.Error("the preview is still drawn, want it shut")
	}
}

// buttonFaces is what stands behind the picture of the button named key,
// and what stands beside it: a button of no look leaves the two the same,
// for its face is the pane's own.
func buttonFaces(t *testing.T, tt *ui.Tester, key string) (color.RGBA, color.RGBA) {
	t.Helper()
	r, ok := tt.Find(key)
	if !ok {
		t.Fatalf("%q is not drawn", key)
	}
	img := tt.Image()
	behind := img.RGBAAt(int(r.X+r.W)-3, int(r.Y+r.H)-3)
	beside := img.RGBAAt(int(r.X+r.W)+5, int(r.Y+r.H)-3)
	return behind, beside
}

// TestZoom_theCornerButtonIsItsPictureAlone checks that the button of the
// corner draws no face of its own, as the one that widens the note does
// not: the pane's own face stands behind the picture, whether the preview
// has the room or not, and the arrow turned round is what says so. The
// rail's buttons, which say the way back in words alone, stand lit, and
// are the other half of the rule.
func TestZoom_theCornerButtonIsItsPictureAlone(t *testing.T) {
	w, tt := spreadTester(t, 1100)
	behind, beside := buttonFaces(t, tt, locale.SpreadPreview())
	if behind != beside {
		t.Errorf("the button stands on the face %v, want the pane's own %v",
			behind, beside)
	}
	if err := tt.Click(locale.SpreadPreview()); err != nil {
		t.Fatalf("Click: %v", err)
	}
	if behind, beside = buttonFaces(t, tt, locale.SpreadPreviewBack()); behind != beside {
		t.Errorf("the button stands on the face %v once the preview has"+
			" the room, want the pane's own %v", behind, beside)
	}
	// The rail's is the one that is lit, and it is lit only when the pane
	// it gives the room has it.
	w.setZoom(layout.ZoomNone)
	tt.Frame()
	trail, _ := buttonFaces(t, tt, locale.ZoomPreview())
	w.setZoom(layout.ZoomPreview)
	tt.Frame()
	if lit, _ := buttonFaces(t, tt, locale.ZoomPreview()); lit == trail {
		t.Errorf("the rail's button keeps the face %v when the preview has"+
			" the window, want it lit", lit)
	}
}
