package view

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/state"
)

// workspaceShown is a window with one note open in it, and someone told
// whenever the panels are made wider or shut.
func workspaceShown(t *testing.T) (*Workspace, *int) {
	t.Helper()
	_, a, _ := vaultWithNotes(t)
	w := NewWorkspace(config.Default())
	if _, err := w.Tabs.Open(a); err != nil {
		t.Fatalf("Open: %v", err)
	}
	told := 0
	w.OnLayoutChanged = func() { told++ }
	return w, &told
}

// TestSetTreeShown_shuttingItKeepsItsWidth checks that a panel shut and
// opened again comes back as wide as it was, and not as wide as someone
// thought up on the spot.
func TestSetTreeShown_shuttingItKeepsItsWidth(t *testing.T) {
	w, told := workspaceShown(t)
	w.Layout.Left = 300
	w.SetTreeShown(false)
	if w.Layout.Left != 0 || w.TreeShown() {
		t.Fatalf("left = %v, want it shut", w.Layout.Left)
	}
	if w.Layout.LastLeft != 300 {
		t.Fatalf("last left = %v, want the 300 it was", w.Layout.LastLeft)
	}
	if *told != 1 {
		t.Fatalf("told %d times, want once", *told)
	}
	w.SetTreeShown(true)
	if w.Layout.Left != 300 || !w.TreeShown() {
		t.Fatalf("left = %v, want the 300 it was", w.Layout.Left)
	}
	if *told != 2 {
		t.Fatalf("told %d times, want twice", *told)
	}
}

// TestSetTreeShown_oneThatWasNeverSizedGetsTheUsualWidth checks a panel
// shut before anyone gave it a width: it still has to come back.
func TestSetTreeShown_oneThatWasNeverSizedGetsTheUsualWidth(t *testing.T) {
	w, _ := workspaceShown(t)
	w.Layout.Left = 0
	w.Layout.LastLeft = 0
	w.SetTreeShown(true)
	if w.Layout.Left != state.DefaultLayout().Left {
		t.Fatalf("left = %v, want the usual %v", w.Layout.Left,
			state.DefaultLayout().Left)
	}
}

// TestSetPreviewShown_theSameEitherSide checks the preview, which is the
// other one of the two.
func TestSetPreviewShown_theSameEitherSide(t *testing.T) {
	w, _ := workspaceShown(t)
	w.Layout.Right = 500
	w.SetPreviewShown(false)
	if w.Layout.Right != 0 || w.PreviewShown() {
		t.Fatalf("right = %v, want it shut", w.Layout.Right)
	}
	w.SetPreviewShown(true)
	if w.Layout.Right != 500 {
		t.Fatalf("right = %v, want the 500 it was", w.Layout.Right)
	}
}

// TestSetShown_askingForWhatAlreadyIsDoesNothing checks that saying
// "show it" of a panel already shown is not a change, and is not written
// down over and over.
func TestSetShown_askingForWhatAlreadyIsDoesNothing(t *testing.T) {
	w, told := workspaceShown(t)
	w.SetTreeShown(true)
	w.SetPreviewShown(true)
	if *told != 0 {
		t.Fatalf("told %d times, want nothing said", *told)
	}
	w.SetTreeShown(false)
	w.SetTreeShown(false)
	if *told != 1 {
		t.Fatalf("told %d times, want once", *told)
	}
}

// TestLayout_aPanelOfNoWidthIsOneThatWasShut checks the test the window
// draws by.
func TestLayout_aPanelOfNoWidthIsOneThatWasShut(t *testing.T) {
	one := state.DefaultLayout()
	if !one.TreeShown() || !one.PreviewShown() {
		t.Fatal("a first start shows both panels")
	}
	shut := state.Layout{Version: state.LayoutVersion, Left: 0, Right: 0}
	if shut.TreeShown() || shut.PreviewShown() {
		t.Fatal("a panel of no width was said to stand")
	}
}

// windowDrawn is the window with its two panels open and a note in them,
// drawn in a window of the height given.
func windowDrawn(t *testing.T, height int) (*Workspace, *ui.Tester) {
	t.Helper()
	_, a, _ := vaultWithNotes(t)
	w := NewWorkspace(config.Default())
	w.Tree.SetRoot(treeRoot())
	n, err := w.Tabs.Open(a)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	n.Editor.Buffer.Set("# head\n\nsome words\n")
	w.SetTreeShown(true)
	w.SetPreviewShown(true)
	te := ui.NewTester(w.View, 1100, height)
	te.Frame()
	return w, te
}

// TestPanels_theTreeAndThePreviewStartAtTheTop checks where the two
// panels put what they show: at the top of the window, and not in the
// middle of it, which is where a row leaves a panel of no height.
func TestPanels_theTreeAndThePreviewStartAtTheTop(t *testing.T) {
	// What stands in the tree's head, and no more: a row is under it.
	// The preview's first line stands under the row of its own corner,
	// which is the button that spreads the preview over the note.
	const underHead, underTop = 80, 48
	for _, height := range []int{400, 900} {
		_, te := windowDrawn(t, height)
		for _, want := range []struct {
			text string
			top  float32
		}{
			{"a.md", underHead},       // the tree's first file
			{"head", underTop},        // the preview's first line
			{"some words", underHead}, // the line under it
		} {
			r, ok := te.Find(want.text)
			if !ok {
				t.Fatalf("%q is not drawn", want.text)
			}
			if r.Y > want.top {
				t.Errorf("h=%d: %q is at y=%v, want it at the top",
					height, want.text, r.Y)
			}
		}
	}
}

// TestPanels_theSamePlaceInAWindowOfAnyHeight checks that making the
// window taller does not move what the panels show downwards, as it
// would were what they show put in the middle of it.
func TestPanels_theSamePlaceInAWindowOfAnyHeight(t *testing.T) {
	_, short := windowDrawn(t, 500)
	_, tall := windowDrawn(t, 1000)
	for _, text := range []string{"a.md", "head"} {
		a, ok := short.Find(text)
		if !ok {
			t.Fatalf("%q is not drawn", text)
		}
		b, ok := tall.Find(text)
		if !ok {
			t.Fatalf("%q is not drawn", text)
		}
		if a.Y != b.Y {
			t.Errorf("%q is at y=%v in a short window and %v in a tall one,"+
				" want it in the same place in both", text, a.Y, b.Y)
		}
	}
}
