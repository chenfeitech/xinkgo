package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/theme"
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

// TestSettingsView_aThemeCardChoosesThatScheme checks that the window is
// drawn in the scheme the user clicks, and not only in the one the
// drop-down says.
func TestSettingsView_aThemeCardChoosesThatScheme(t *testing.T) {
	s := &appState{}
	tt := settingsShown(t, s)
	if s.config.Basic.Theme != theme.Default {
		t.Fatalf("the scheme is %q, want %q", s.config.Basic.Theme,
			theme.Default)
	}
	if err := tt.Click("Ocean"); err != nil {
		t.Fatal(err)
	}
	if s.config.Basic.Theme != theme.Ocean {
		t.Errorf("the scheme is %q, want %q", s.config.Basic.Theme,
			theme.Ocean)
	}
	// It is a choice that is kept: the app comes up as it was left.
	got, err := config.Load(s.dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Basic.Theme != theme.Ocean {
		t.Errorf("the scheme kept is %q, want %q", got.Basic.Theme,
			theme.Ocean)
	}
}

// TestSettingsView_choosingASchemeShowsItAtOnce checks that the window
// itself is drawn in the scheme chosen, and not the next time it is
// opened: it is what makes a choice of colors one the user sees.
func TestSettingsView_choosingASchemeShowsItAtOnce(t *testing.T) {
	s := &appState{}
	tt := settingsShown(t, s)
	if err := tt.Click("Moss"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Moss") {
		t.Error("the card of the scheme chosen is not on show")
	}
	// The code is painted in the preset that goes with the scheme.
	if s.config.Highlight.Theme != theme.Code(theme.Moss) {
		t.Errorf("the code is painted in %q, want %q",
			s.config.Highlight.Theme, theme.Code(theme.Moss))
	}
}

// TestFollowCodeTheme_theSchemeBringsItsPreset checks that the code
// follows the scheme, and that a preset the user chose themselves is
// theirs, and does not follow.
func TestFollowCodeTheme_theSchemeBringsItsPreset(t *testing.T) {
	s := appWithWindow(t)
	s.config = config.Default()
	next := s.config
	next.Basic.Theme = theme.Ocean
	s.followCodeTheme(&next)
	if next.Highlight.Theme != theme.Code(theme.Ocean) {
		t.Errorf("the code is painted in %q, want %q",
			next.Highlight.Theme, theme.Code(theme.Ocean))
	}
	// A preset of the user's own is a choice kept.
	s.config.Highlight.Theme = config.HighlightSolarized
	kept := s.config
	kept.Basic.Theme = theme.Ocean
	s.followCodeTheme(&kept)
	if kept.Highlight.Theme != config.HighlightSolarized {
		t.Errorf("the code is painted in %q, want the one chosen, %q",
			kept.Highlight.Theme, config.HighlightSolarized)
	}
}
