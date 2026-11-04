package view

import "testing"

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
