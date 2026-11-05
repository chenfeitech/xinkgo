package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/plugin"
	"github.com/helightxu/xinkgo/internal/view"
)

// pluginSettingsShown draws one plugin group without opening a window and
// without prepareSettings starting its background work: shown and tab are
// the same, so chosenSettings does not start a refresh. One built-in
// feature and one installed community plugin stand in for the real data.
func pluginSettingsShown(t *testing.T, s *appState, tab string) *ui.Tester {
	t.Helper()
	s.dir = t.TempDir()
	s.config = config.Default()
	s.ws = view.NewWorkspace(config.Default())
	s.draft = settings{cfg: s.config, tab: tab, shown: tab}
	s.draft.read(&s.config)
	s.plugins = pluginPaneHost()
	return ui.NewTester(s.settingsView, 820, 680)
}

// pluginPaneHost is the smallest host whose two settings panes draw real
// rows. The catalog is marked loaded so prepareSettings, should it run,
// never reaches for the network.
func pluginPaneHost() *pluginHost {
	report := plugin.Report{
		ID: "demo", Name: "Demo Plugin", Version: "1.0",
		Author: "Someone", Supported: true,
	}
	return &pluginHost{
		compatible:    []plugin.CompatibleEntry{},
		catalog:       []plugin.CatalogEntry{},
		stats:         map[string]plugin.PluginStats{},
		details:       map[string]plugin.PluginDetail{},
		detailLoading: map[string]bool{},
		reports:       map[string]plugin.Report{"demo": report},
		catalogLoaded: true,
		ui: pluginUI{
			tab:       "installed",
			busy:      map[string]bool{},
			open:      map[string]bool{},
			installed: []plugin.Report{report},
		},
	}
}

// The core pane is a rounded card of switches: it names Xinkgo's own
// features and groups them under a heading, with no third-party rows.
func TestPluginSettings_corePaneShowsBuiltInFeatures(t *testing.T) {
	s := &appState{}
	tt := pluginSettingsShown(t, s, tabCorePlugins)
	for _, want := range []string{
		locale.PluginGroupBuiltin(),
		locale.CoreFileName(),
		locale.CoreFileNote(),
		locale.PluginBuiltinNote(),
	} {
		if !tt.HasText(want) {
			t.Errorf("the core pane does not say %q", want)
		}
	}
}

// An installed card offers to take the plugin away, not just to turn it
// off: the uninstall control sits beside reload and open-folder.
func TestPluginSettings_installedCardOffersUninstall(t *testing.T) {
	s := &appState{}
	tt := pluginSettingsShown(t, s, tabCommunityPlugins)
	if !tt.HasText("Demo Plugin") {
		t.Fatal("the installed plugin card is not on show")
	}
	for _, want := range []string{
		locale.PluginUninstall(),
		locale.PluginReload(),
		locale.PluginOpenFolder(),
	} {
		if !tt.HasText(want) {
			t.Errorf("the installed card does not offer %q", want)
		}
	}
}

// A community plugin is a card: its name and the fold control are on show,
// while the grants are folded away until the card is opened.
func TestPluginSettings_communityCardFoldsItsDetails(t *testing.T) {
	s := &appState{}
	tt := pluginSettingsShown(t, s, tabCommunityPlugins)
	for _, want := range []string{
		"Demo Plugin",
		locale.PluginShowDetails(),
	} {
		if !tt.HasText(want) {
			t.Errorf("the community card does not say %q", want)
		}
	}
	if tt.HasText(locale.PluginGrants()) {
		t.Error("the grants are on show before the card is opened")
	}
	if err := tt.Click(locale.PluginShowDetails()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText(locale.PluginGrants()) {
		t.Error("the grants did not unfold when the card was opened")
	}
	if !tt.HasText(locale.PluginHideDetails()) {
		t.Error("the fold control does not offer to hide the details")
	}
}
