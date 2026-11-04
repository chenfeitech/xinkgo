package view

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestPalette_pluginCommandsAreListedAndChosen(t *testing.T) {
	w := &Workspace{
		PluginCommands: []PluginCommand{
			{ID: "demo:one", Name: "Demo: One", Visible: true},
			{ID: "demo:hidden", Name: "Demo: Hidden", Visible: false},
		},
	}
	called := ""
	w.OnPluginCommand = func(id string) { called = id }
	w.paletteQuery = "one"
	items := w.pluginItems()
	if len(items) != 1 || items[0].Plugin == nil ||
		items[0].Plugin.ID != "demo:one" {
		t.Fatalf("plugin items = %+v", items)
	}
	w.paletteOpen = true
	w.choose(0)
	if called != "demo:one" {
		t.Errorf("chosen command = %q, want demo:one", called)
	}
}

func TestPluginRibbon_clickReportsAction(t *testing.T) {
	w := &Workspace{
		PluginRibbon: []PluginAction{{ID: "demo:one", Name: "Demo One"}},
	}
	called := ""
	w.OnPluginAction = func(id string) { called = id }
	tt := ui.NewTester(w.pluginRibbon, 100, 200)
	if err := tt.Click("Demo One"); err != nil {
		t.Fatal(err)
	}
	if called != "demo:one" {
		t.Errorf("clicked action = %q, want demo:one", called)
	}
}
