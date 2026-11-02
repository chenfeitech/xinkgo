package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/view"
)

// settingsShown is the settings window, drawn without one.
func settingsShown(t *testing.T, s *appState) *ui.Tester {
	t.Helper()
	s.dir = t.TempDir()
	s.config = config.Default()
	s.ws = view.NewWorkspace(config.Default())
	s.draft = settings{cfg: s.config, tab: tabBasic}
	s.draft.read(&s.config)
	return ui.NewTester(s.settingsView, 820, 680)
}

// TestSettingsView_everyGroupIsInTheList checks the window as Obsidian
// has it: the groups in a list, and the rows of one of them beside it.
func TestSettingsView_everyGroupIsInTheList(t *testing.T) {
	s := &appState{}
	tt := settingsShown(t, s)
	for _, name := range []string{
		"Basic", "Editor", "Preview", "AI", "Vault", "Settings file",
	} {
		if !tt.HasText(name) {
			t.Errorf("the list of groups does not say %q", name)
		}
	}
	// The first group's rows are the ones on show at the start.
	if !tt.HasText("Theme") {
		t.Error("the window shows no rows of the group chosen")
	}
}

// TestSettingsView_choosingAGroupShowsOnlyItsRows checks that the list
// chooses what is drawn: the rows of another group are not on show, so
// that a group is not drawn for nothing.
func TestSettingsView_choosingAGroupShowsOnlyItsRows(t *testing.T) {
	s := &appState{}
	tt := settingsShown(t, s)
	if !tt.HasText("Show every file") {
		t.Fatal("the basic rows are not on show")
	}
	if err := tt.Click("Editor"); err != nil {
		t.Fatal(err)
	}
	if s.draft.tab != tabEditor {
		t.Fatalf("the group chosen is %q, want the editor's", s.draft.tab)
	}
	if !tt.HasText("Line height") {
		t.Error("the editor's rows are not on show")
	}
	if tt.HasText("Show every file") {
		t.Error("the basic rows are still on show")
	}
}
