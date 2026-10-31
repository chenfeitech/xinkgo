package view

import (
	"testing"

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
