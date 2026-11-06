package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/index"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/plugin"
	"github.com/helightxu/xinkgo/internal/state"
	"github.com/helightxu/xinkgo/internal/templating"
	"github.com/helightxu/xinkgo/internal/view"
)

// coreTestState is a small app with a real vault, workspace and plugin
// host, but no windows or event loop. Native Window calls are never made.
func coreTestState(t *testing.T) *appState {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, "vault")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	ws := view.NewWorkspace(cfg)
	idx := index.New()
	ws.Index = idx
	st := &appState{dir: dir, root: root, config: cfg, ws: ws, idx: idx}
	st.plugins = &pluginHost{
		st:        st,
		dir:       dir,
		userDir:   filepath.Join(dir, "plugins"),
		resources: filepath.Join(dir, "resources"),
		assets:    map[string][]byte{},
		instances: map[string]*pluginInstance{},
		byPage:    map[*mygo.Page]*pluginInstance{},
		byHost:    map[string]*pluginInstance{},
		reports:   map[string]plugin.Report{},
		stats:     map[string]plugin.PluginStats{},
		saved:     state.PluginState{},
		ui: pluginUI{
			busy: map[string]bool{}, open: map[string]bool{},
		},
	}
	return st
}

func enableCore(s *appState, ids ...string) {
	for _, id := range ids {
		s.plugins.saved = s.plugins.saved.WithCore(id, true)
	}
}

func commandIDs(list []view.PluginCommand) map[string]bool {
	out := map[string]bool{}
	for _, one := range list {
		out[one.ID] = true
	}
	return out
}

func TestMergeContributions_mergesBothKinds(t *testing.T) {
	saved := state.PluginState{}
	for _, id := range []string{"daily-notes", "templates", "web-viewer"} {
		saved = saved.WithCore(id, true)
	}
	snap := contributionSnapshot{
		commands: []view.PluginCommand{{
			ID: "demo:one", Name: "Demo One", Visible: true,
		}},
		ribbon: []view.PluginAction{{ID: "demo:rib", Name: "Demo Rib"}},
	}
	features, commands, ribbon, _ := mergeContributions(saved, true, snap)
	if !features["daily-notes"] || !features["templates"] ||
		!features["web-viewer"] {
		t.Fatalf("features = %+v", features)
	}
	ids := commandIDs(commands)
	for _, id := range []string{
		"daily-notes:open-today", "templates:insert", "web-viewer:open",
		"demo:one",
	} {
		if !ids[id] {
			t.Errorf("command %s is missing", id)
		}
	}
	if len(ribbon) != 4 {
		t.Errorf("ribbon = %+v, want three built-in and one third-party",
			ribbon)
	}
	for _, one := range commands {
		if one.Group == "" && strings.HasPrefix(one.ID, "daily-notes:") {
			t.Errorf("built-in command %s has no group", one.ID)
		}
	}
}

// Every registry entry that declares a ribbon must actually contribute a
// button, and every contributed button must be a command the dispatcher
// can route. This is the guard against a plugin declaring a ribbon and
// silently getting no icon (the Templates case).
func TestCoreRibbonActions_matchRegistry(t *testing.T) {
	for _, one := range plugin.CorePlugins {
		actions := coreRibbonActions(one.ID)
		if one.Ribbon && len(actions) == 0 {
			t.Errorf("%s declares a ribbon but contributes no button",
				one.ID)
		}
		if !one.Ribbon && len(actions) != 0 {
			t.Errorf("%s has no ribbon but contributes %d buttons",
				one.ID, len(actions))
		}
		for _, action := range actions {
			if action.ID == "" || action.Name == "" ||
				action.Icon == "" {
				t.Errorf("%s has an incomplete ribbon button: %+v",
					one.ID, action)
			}
			routable := false
			for _, cmd := range coreCommands {
				if cmd.ID == action.ID && cmd.Owner == one.ID {
					routable = true
				}
			}
			if !routable {
				t.Errorf("%s ribbon button %s has no command handler",
					one.ID, action.ID)
			}
		}
	}
}

func TestMergeContributions_threeCoreRibbonIcons(t *testing.T) {
	saved := state.PluginState{}
	for _, id := range []string{"daily-notes", "templates", "web-viewer"} {
		saved = saved.WithCore(id, true)
	}
	_, _, ribbon, _ := mergeContributions(saved, true,
		contributionSnapshot{})
	ids := map[string]string{}
	for _, one := range ribbon {
		ids[one.ID] = one.Icon
	}
	for id, icon := range map[string]string{
		"daily-notes:open-today": "calendar-days",
		"templates:insert":       "layout-template",
		"web-viewer:open":        "globe",
	} {
		if ids[id] != icon {
			t.Errorf("ribbon %s = %q, want icon %q", id, ids[id], icon)
		}
	}
}

func TestRunAction_routesBuiltInRibbon(t *testing.T) {
	st := coreTestState(t)
	enableCore(st, "daily-notes")
	st.runAction("daily-notes:open-today")
	name, err := templating.Format(time.Now(), "YYYY-MM-DD")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(st.root, name+".md")); err != nil {
		t.Fatalf("the ribbon button did not open today's note: %v", err)
	}
}

func TestMergeContributions_webViewUnavailableHidesWebViewer(t *testing.T) {
	saved := state.PluginState{}.WithCore("daily-notes", true).
		WithCore("web-viewer", true)
	features, commands, _, _ := mergeContributions(saved, false,
		contributionSnapshot{})
	if features["web-viewer"] {
		t.Error("the web viewer stands on a machine with no webview")
	}
	if !features["daily-notes"] {
		t.Error("another plugin was turned off too")
	}
	for _, one := range commands {
		if strings.HasPrefix(one.ID, "web-viewer:") {
			t.Errorf("web viewer command %s is still offered", one.ID)
		}
	}
}

func TestCoreEnabled_offByDefault(t *testing.T) {
	st := coreTestState(t)
	for _, id := range []string{"daily-notes", "templates", "web-viewer"} {
		if st.coreEnabled(id) {
			t.Errorf("%s is on by default", id)
		}
	}
	enableCore(st, "daily-notes", "templates")
	if !st.coreEnabled("daily-notes") || !st.coreEnabled("templates") {
		t.Fatal("a switch did not turn the plugin on")
	}
	if st.coreEnabled("web-viewer") {
		t.Error("the web viewer is on without being enabled")
	}
}

func TestRunCommand_createsTodaysDailyNote(t *testing.T) {
	st := coreTestState(t)
	enableCore(st, "daily-notes")
	st.runCommand("daily-notes:open-today")
	name, err := templating.Format(time.Now(), "YYYY-MM-DD")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(st.root, name+".md")
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("today's note was not made: %v", err)
	}
	if filepath.Base(st.ws.Tabs.Active) != name+".md" {
		t.Errorf("active note = %q", st.ws.Tabs.Active)
	}
}

func TestEnsureNote_neverOverwrites(t *testing.T) {
	st := coreTestState(t)
	full, err := st.ensureNote("a.md", "first")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.ensureNote("a.md", "second"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(full)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "first" {
		t.Errorf("the note was overwritten: %q", data)
	}
}

func TestVaultPath_refusesEscape(t *testing.T) {
	st := coreTestState(t)
	for _, one := range []string{
		"../outside.md", "/etc/passwd", "a/../../b.md", "..\x00x",
	} {
		if _, ok := st.vaultPath(one); ok {
			t.Errorf("%q escaped the vault", one)
		}
	}
	if _, ok := st.vaultPath("Daily notes/2026.md"); !ok {
		t.Error("a good relative path was refused")
	}
}

func TestDailyOpenAdjacent_findsEarlierNote(t *testing.T) {
	st := coreTestState(t)
	enableCore(st, "daily-notes")
	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)
	todayRel, _ := st.dailyRelPath(today)
	yesterdayRel, _ := st.dailyRelPath(yesterday)
	todayFull, err := st.ensureNote(todayRel, "today")
	if err != nil {
		t.Fatal(err)
	}
	yesterdayFull, err := st.ensureNote(yesterdayRel, "yesterday")
	if err != nil {
		t.Fatal(err)
	}
	st.ws.Tabs.Activate(todayFull)
	st.dailyOpenAdjacent(-1)
	if st.ws.Tabs.Active != yesterdayFull {
		t.Errorf("the previous note was not opened: %q", st.ws.Tabs.Active)
	}
}

func TestInsertTemplateNamed_expandsIntoActiveNote(t *testing.T) {
	st := coreTestState(t)
	enableCore(st, "templates")
	tplDir := filepath.Join(st.root, "Templates")
	if err := os.MkdirAll(tplDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tplDir, "demo.md"),
		[]byte("# {{title}}\n{{date}}"), 0o644); err != nil {
		t.Fatal(err)
	}
	o := plugin.DefaultCoreOptions()
	o.Templates.Folder = "Templates"
	st.plugins.setOptions(o)
	full, err := st.ensureNote("note.md", "start\n")
	if err != nil {
		t.Fatal(err)
	}
	st.ws.Tabs.Activate(full)
	st.insertTemplateNamed("demo")
	n := st.ws.Tabs.ActiveNote()
	if n == nil || !strings.Contains(n.Note.Text, "# note") ||
		strings.Contains(n.Note.Text, "{{date}}") {
		t.Errorf("unexpected note text: %+v", n)
	}
}

func TestSettingsPages_showPluginOptions(t *testing.T) {
	st := coreTestState(t)
	enableCore(st, "daily-notes", "templates")
	st.draft = settings{cfg: st.config, tab: tabDailyNotes,
		shown: tabDailyNotes, coreOpts: st.coreOptions(),
		coreOptsLoaded: true}
	st.draft.read(&st.config)
	tt := ui.NewTester(st.settingsView, 820, 680)
	for _, want := range []string{
		locale.SettingsSectionPluginOptions(),
		locale.DailyDateFormat(),
		locale.SettingsTabTemplates(),
	} {
		if !tt.HasText(want) {
			t.Errorf("the settings window does not say %q", want)
		}
	}
}

func TestWebViewerSettings_explainsNoWebView(t *testing.T) {
	st := coreTestState(t)
	tt := ui.NewTester(st.webViewerSettings, 820, 680)
	// On a Mac with WKWebView present the page shows its controls even
	// without a running event loop; elsewhere it says why it cannot run.
	if mygo.WebViewAvailable() {
		if !tt.HasText(locale.WebHomepage()) {
			t.Error("the web viewer options page did not draw its fields")
		}
	} else if !tt.HasText(locale.WebNoWebView()) {
		t.Error("a machine with no webview was not told why")
	}
}

func TestWebRulesAndFields(t *testing.T) {
	if webRules(nil) != nil {
		t.Error("no rules should be an empty list")
	}
	fields := map[string]any{"title": "A", "text": "B"}
	if webField(fields, "title") != "A" ||
		webField(fields, "missing") != "" {
		t.Error("a reader field was not read")
	}
}

// The Web viewer no longer has a window of its own: its tabs are pushed
// to the workspace's preview pane, and the Preview tab takes the note's
// Markdown preview back.
func TestWebPublish_pushesTabsToPreviewPane(t *testing.T) {
	st := coreTestState(t)
	enableCore(st, "web-viewer")
	st.web.tabs = []*webTab{
		{id: "web-1", url: "https://example.com", title: "Example",
			canBack: true, canForward: true},
		{id: "web-2", url: "https://other.example", title: "Other",
			canBack: true, canForward: true},
	}
	st.web.active = "web-1"
	st.webPublish()
	if len(st.ws.WebTabs) != 2 || st.ws.WebTabActive != "web-1" {
		t.Fatalf("preview tabs = %+v, active %q", st.ws.WebTabs,
			st.ws.WebTabActive)
	}
	if st.ws.WebAddress != "https://example.com" {
		t.Errorf("address = %q", st.ws.WebAddress)
	}
	if !st.ws.WebTabs[0].Active || st.ws.WebTabs[1].Active {
		t.Error("the wrong tab is lit")
	}
	if st.ws.WebTabs[0].Page != nil {
		t.Error("a tab with no native page published one anyway")
	}
	st.webShowPreview()
	if st.ws.WebTabActive != "" {
		t.Errorf("the Preview tab did not return to the note: %q",
			st.ws.WebTabActive)
	}
}

func TestWebDispose_clearsPreviewTabs(t *testing.T) {
	st := coreTestState(t)
	st.web.tabs = []*webTab{{id: "web-1", url: "about:blank"}}
	st.web.active = "web-1"
	st.webPublish()
	st.webDispose()
	if len(st.ws.WebTabs) != 0 || st.ws.WebTabActive != "" ||
		len(st.web.tabs) != 0 || st.web.active != "" {
		t.Error("disposing the plugin left preview tabs behind")
	}
}

// The bundled content rules must be valid WebKit trigger JSON: load-type
// and resource-type are arrays, not strings. A string there made the
// whole list fail to compile at runtime.
func TestWebContentRules_shape(t *testing.T) {
	data, err := webViewerAssets.ReadFile(webRulesAsset)
	if err != nil {
		t.Fatal(err)
	}
	var rules []map[string]json.RawMessage
	if err := json.Unmarshal(data, &rules); err != nil {
		t.Fatalf("the rule list is not JSON: %v", err)
	}
	if len(rules) == 0 {
		t.Fatal("the rule list is empty")
	}
	for i, rule := range rules {
		var trigger struct {
			URLFilter    string          `json:"url-filter"`
			LoadType     json.RawMessage `json:"load-type"`
			ResourceType json.RawMessage `json:"resource-type"`
		}
		if raw, ok := rule["trigger"]; ok {
			if err := json.Unmarshal(raw, &trigger); err != nil {
				t.Fatalf("rule %d has a bad trigger: %v", i, err)
			}
		}
		if trigger.URLFilter == "" {
			t.Errorf("rule %d has no url-filter", i)
		}
		if _, ok := rule["action"]; !ok {
			t.Errorf("rule %d has no action", i)
		}
		for name, raw := range map[string]json.RawMessage{
			"load-type":     trigger.LoadType,
			"resource-type": trigger.ResourceType,
		} {
			if len(raw) == 0 {
				continue
			}
			var list []string
			if err := json.Unmarshal(raw, &list); err != nil {
				t.Errorf("rule %d: %s must be an array: %s", i, name,
					raw)
			}
		}
	}
}

// The Templates options page draws a card row with the file's name,
// Markdown badge and vault-relative path, not a bare line of text.
func TestTemplatesSettings_listRowsShowMetadata(t *testing.T) {
	st := coreTestState(t)
	st.draft = settings{cfg: st.config, coreOpts: st.coreOptions(),
		coreOptsLoaded: true}
	st.draft.tplFiles = []coreTemplateInfo{{
		Name: "Daily", Rel: "Templates/Daily.md", Size: 2048,
		ModTime: time.Date(2026, 11, 6, 10, 0, 0, 0, time.Local),
	}}
	tt := ui.NewTester(st.templatesSettings, 820, 680)
	for _, want := range []string{
		locale.TplListTitle(), "Daily", ".md", locale.TplCount("1"),
		locale.TplMeta("Templates/Daily.md", "2.0 KB",
			"2026-11-06 10:00"),
	} {
		if !tt.HasText(want) {
			t.Errorf("the template list does not say %q", want)
		}
	}
}
