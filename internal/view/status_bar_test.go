package view

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/index"
	"github.com/helightxu/xinkgo/internal/locale"
)

// Plugin status items are keyed by plugin ID: one plugin's text must not
// overwrite another's, nor the window's own message.
func TestPluginStatuses_areIndependentAndStable(t *testing.T) {
	w := NewWorkspace(config.Default())
	w.Say("a window message")
	w.AddPluginStatus("beta")
	w.SetPluginStatus("alpha", "A text")
	w.SetPluginStatus("beta", "B text")
	if len(w.PluginStatuses) != 2 {
		t.Fatalf("statuses = %+v", w.PluginStatuses)
	}
	if w.PluginStatuses[0].ID != "alpha" ||
		w.PluginStatuses[1].ID != "beta" {
		t.Errorf("statuses are not in a stable order: %+v",
			w.PluginStatuses)
	}
	if w.PluginStatuses[0].Text != "A text" ||
		w.PluginStatuses[1].Text != "B text" {
		t.Errorf("status texts were overwritten: %+v", w.PluginStatuses)
	}
	// Reserving an existing item again must not clear its text: the two
	// fire() calls from addStatusBarItem may arrive in either order.
	w.AddPluginStatus("beta")
	if w.PluginStatuses[1].Text != "B text" {
		t.Error("a late status.add cleared an existing status text")
	}
	w.RemovePluginStatus("alpha")
	if len(w.PluginStatuses) != 1 ||
		w.PluginStatuses[0].ID != "beta" {
		t.Errorf("after removal: %+v", w.PluginStatuses)
	}
}

// With no note open there is no file name and no word count, but the status
// bar still stands and shows whatever a plugin put on it.
func TestStatusBar_emptyWindowStillShowsPluginItems(t *testing.T) {
	w := NewWorkspace(config.Default())
	w.SetPluginStatus("xinkgo-demo-basic", "Demo Basic: 3 runs")
	tt := ui.NewTester(w.statusBar, 600, 40)
	if !tt.HasText("Demo Basic: 3 runs") {
		t.Error("the status bar did not draw with no note open")
	}
}

func TestStatusBar_showsEveryPluginItem(t *testing.T) {
	w := NewWorkspace(config.Default())
	w.SetPluginStatus("xinkgo-demo-basic", "Demo Basic: 3 runs")
	w.SetPluginStatus("xinkgo-demo-full", "Demo Full: 5 runs")
	tt := ui.NewTester(w.statusBar, 600, 40)
	for _, want := range []string{
		"Demo Basic: 3 runs", "Demo Full: 5 runs",
	} {
		if !tt.HasText(want) {
			t.Errorf("the status bar does not say %q", want)
		}
	}
}

// An entirely empty window must still leave the row in place.
func TestStatusBar_noContentDoesNotPanic(t *testing.T) {
	w := NewWorkspace(config.Default())
	tt := ui.NewTester(w.statusBar, 600, 40)
	tt.Frame()
}

// An empty status bar must be visible, not just a blank row: it shows the
// vault name, or an idle label when no vault has been scanned yet.
func TestStatusBar_idleTextIsVisible(t *testing.T) {
	w := NewWorkspace(config.Default())
	tt := ui.NewTester(w.statusBar, 600, 40)
	if !tt.HasText(locale.StatusBarIdle()) {
		t.Error("an empty status bar shows nothing")
	}
}

// The decisive geometry check: in the full window the status bar must sit
// inside the viewport, not be pushed below it by the panels row.
func TestStatusBar_isVisibleInTheFullWindow(t *testing.T) {
	w := NewWorkspace(config.Default())
	w.Index = index.New()
	w.SetPluginStatus("demo", "STATUS-BAR-MARKER")
	tt := ui.NewTester(w.View, 820, 680)
	r, ok := tt.Find("STATUS-BAR-MARKER")
	if !ok {
		t.Fatal("the status bar was not drawn in the full window")
	}
	if r.Y < 0 || r.Y+r.H > 680 {
		t.Errorf("the status bar fell outside the window: %+v", r)
	}
}
