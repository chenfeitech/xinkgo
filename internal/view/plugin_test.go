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

func TestCoreOn_defaultsToEverything(t *testing.T) {
	w := &Workspace{}
	if !w.coreOn(CoreFileExplorer) || !w.coreOn(CoreWordCount) {
		t.Fatal("a workspace with no core map hid a feature")
	}
	w.CoreFeatures = map[string]bool{CoreWordCount: false}
	if w.coreOn(CoreWordCount) || !w.coreOn(CoreOutline) {
		t.Fatal("an explicit core choice was not honoured")
	}
}

func TestWordsIn_countsLatinAndCJK(t *testing.T) {
	cases := []struct {
		text string
		want int
	}{
		{"one two three", 3},
		{"one\n\ntwo", 2},
		{"你好世界", 4},
		{"hello 世界", 3},
		{"", 0},
	}
	for _, one := range cases {
		if got := wordsIn(one.text); got != one.want {
			t.Errorf("wordsIn(%q) = %d, want %d", one.text, got,
				one.want)
		}
	}
}

func TestPluginViewDrawer_pickReportsView(t *testing.T) {
	w := &Workspace{
		PluginViews: []PluginView{
			{ID: "demo:graph", Name: "graph"},
		},
	}
	called := ""
	w.OnPluginView = func(id string) { called = id }
	tt := ui.NewTester(w.pluginViewDrawer, 320, 200)
	if err := tt.Click("graph"); err != nil {
		t.Fatal(err)
	}
	if called != "demo:graph" {
		t.Errorf("chosen view = %q, want demo:graph", called)
	}
}
