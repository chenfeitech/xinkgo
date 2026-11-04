package app

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
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

// TestSettings_thePaddingOfAPaneIsAKnobOfItsOwn checks that the room a
// pane keeps beside what it shows is a setting like any other: taken in
// as the window opens, moved by the knob, and given back to the
// configuration as a whole number of DIPs.
func TestSettings_thePaddingOfAPaneIsAKnobOfItsOwn(t *testing.T) {
	cfg := config.Default()
	st := &settings{}
	st.read(&cfg)
	if st.editorPad != float64(config.DefaultPaddingX) ||
		st.previewPadX != float64(config.DefaultPaddingX) ||
		st.previewPadY != float64(config.DefaultPaddingY) {
		t.Fatalf("the knobs read %v, %v and %v, want %d, %d and %d",
			st.editorPad, st.previewPadX, st.previewPadY,
			config.DefaultPaddingX, config.DefaultPaddingX,
			config.DefaultPaddingY)
	}
	st.editorPad, st.previewPadX, st.previewPadY = 24.4, 31.6, 8.2
	st.write(&cfg)
	if cfg.Editor.PaddingX != 24 {
		t.Errorf("the note's padding is %d, want 24", cfg.Editor.PaddingX)
	}
	if cfg.Preview.PaddingX != 32 {
		t.Errorf("the preview's padding is %d, want 32",
			cfg.Preview.PaddingX)
	}
	if cfg.Preview.PaddingY != 8 {
		t.Errorf("the preview's padding over its top and foot is %d, "+
			"want 8", cfg.Preview.PaddingY)
	}
}

// TestSettingsView_everyGroupIsInTheList checks the window as Obsidian
// has it: the groups in a list, and the rows of one of them beside it.
func TestSettingsView_everyGroupIsInTheList(t *testing.T) {
	s := &appState{}
	tt := settingsShown(t, s)
	for _, name := range []string{
		"Basic", "Editor", "Preview", "AI", "Vault", "Plugins",
		"Settings file",
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

// TestSettingsView_everyGroupHasAPicture checks the list of the groups
// as it stands now: each of them is drawn by a picture of its own, so
// that the group one is looking for is told by its shape as well as by
// its name.
func TestSettingsView_everyGroupHasAPicture(t *testing.T) {
	s := &appState{}
	s.config = config.Default()
	for _, tab := range s.settingsTabs() {
		if tab.Icon == nil {
			t.Errorf("the group %q has no picture", tab.ID)
		}
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

func TestSettingsView_pluginsGroupIsInTheSettings(t *testing.T) {
	s := &appState{}
	tt := settingsShown(t, s)
	if err := tt.Click("Plugins"); err != nil {
		t.Fatal(err)
	}
	if s.draft.tab != tabPlugins {
		t.Errorf("the group chosen is %q, want the plugins'", s.draft.tab)
	}
	if !tt.HasText(locale.PluginNoWebView()) {
		t.Error("the plugin group has no fallback when the host is absent")
	}
}

// TestSettingsView_theStylesOfTheFolderAreSaidThere checks the folder of
// the styles in the settings: the row that names it, and under it a row
// to a style of the folder, saying its name, which of it this is, and who
// made it.
func TestSettingsView_theStylesOfTheFolderAreSaidThere(t *testing.T) {
	s := &appState{}
	tt := settingsShown(t, s)
	dir := t.TempDir()
	write(t, filepath.Join(dir, "mine.json"), []byte(
		`{"id":"mine","name":"我的样式","version":"2.0",`+
			`"author":"谁做的","description":"绿底红字"}`))
	s.draft.tab = tabPreview
	s.draft.wechatDir = dir
	s.draft.wechatRead = "" // read again, as a folder named anew is
	tt.Frame()
	for _, want := range []string{
		locale.SettingWechatDir(), "我的样式", "2.0 · 谁做的 — 绿底红字",
	} {
		if !tt.HasText(want) {
			t.Errorf("the window says nothing of %q", want)
		}
	}
}

// TestSettingsView_theExportStandsWithThePreview checks where the two
// settings of an export are found: with the settings of the preview,
// which is what an export writes out.
func TestSettingsView_theExportStandsWithThePreview(t *testing.T) {
	s := &appState{}
	tt := settingsShown(t, s)
	if err := tt.Click("Preview"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Export", "Sheet of paper", "A4", "Pictures", "In the page",
	} {
		if !tt.HasText(want) {
			t.Errorf("the preview's rows do not say %q", want)
		}
	}
}

// TestSettingsView_thePaddingOfBothPanesIsAChoice checks that the room a
// pane keeps beside what it shows is to be chosen where the rest of the
// pane's settings are: with the note's, and with the preview's.
func TestSettingsView_thePaddingOfBothPanesIsAChoice(t *testing.T) {
	s := &appState{}
	tt := settingsShown(t, s)
	if err := tt.Click("Editor"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(locale.SettingEditorPadding()) {
		t.Errorf("the note's rows do not say %q",
			locale.SettingEditorPadding())
	}
	if err := tt.Click("Preview"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		locale.SettingPreviewPadding(), locale.SettingPreviewPaddingY(),
	} {
		if !tt.HasText(want) {
			t.Errorf("the preview's rows do not say %q", want)
		}
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
