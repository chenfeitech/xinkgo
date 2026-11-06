package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/helightxu/xinkgo/internal/plugin"
)

func TestLoadPlugins_missingAndBrokenEnableNothing(t *testing.T) {
	dir := t.TempDir()
	st, err := LoadPlugins(dir)
	if err != nil || len(st.Enabled) != 0 {
		t.Fatalf("missing file = %+v, %v", st, err)
	}
	if err := os.WriteFile(filepath.Join(dir, PluginsFile), []byte("{"),
		0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPlugins(dir); err == nil {
		t.Error("a broken plugins.json was accepted")
	}
}

func TestPluginState_withoutForgetsEnabledAndGrants(t *testing.T) {
	st := PluginState{
		Enabled: map[string]bool{"demo": true, "other": true},
		Grants: map[string][]string{
			"demo": {"commands"}, "other": {"data"},
		},
		Core: map[string]bool{"file-explorer": false},
	}
	got := st.Without("demo")
	if got.EnabledOf("demo") || len(got.GrantsOf("demo")) != 0 {
		t.Errorf("the plugin is still in the state: %+v", got)
	}
	if !got.EnabledOf("other") || len(got.GrantsOf("other")) != 1 {
		t.Error("another plugin's choices were lost")
	}
	if got.CoreOf("file-explorer") {
		t.Error("a core choice was lost")
	}
	// The original is unchanged: Without returns a copy.
	if !st.EnabledOf("demo") || len(st.GrantsOf("demo")) != 1 {
		t.Error("the original state was changed")
	}
}

func TestSaveLoadPlugins_roundTrip(t *testing.T) {
	dir := t.TempDir()
	want := PluginState{
		Enabled: map[string]bool{"demo": true},
		Grants:  map[string][]string{"demo": {"commands", "editor.write"}},
	}
	if err := SavePlugins(dir, want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadPlugins(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !got.EnabledOf("demo") || len(got.GrantsOf("demo")) != 2 {
		t.Fatalf("got %+v", got)
	}
	data, err := os.ReadFile(filepath.Join(dir, PluginsFile))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("saved file is not JSON: %v", err)
	}
}

func TestPluginState_copiesDoNotShareMaps(t *testing.T) {
	one := PluginState{Enabled: map[string]bool{"a": true}}
	two := one.WithEnabled("b", true)
	two.Enabled["a"] = false
	if !one.EnabledOf("a") {
		t.Error("WithEnabled changed the original map")
	}
	three := one.WithGrants("a", []string{"commands"})
	three.Grants["a"][0] = "network"
	if len(one.GrantsOf("a")) != 0 {
		t.Error("WithGrants changed the original map")
	}
	four := one.WithCore(plugin.CorePlugins[0].ID, false)
	four.Core[plugin.CorePlugins[0].ID] = true
	if one.CoreOf(plugin.CorePlugins[0].ID) != plugin.CorePlugins[0].Default {
		t.Error("WithCore changed the original map")
	}
}

func TestPluginState_coreDefaultsAndExplicitOff(t *testing.T) {
	id := plugin.CorePlugins[0].ID
	// A missing choice takes the registry default.
	if (PluginState{}).CoreOf(id) != plugin.CorePlugins[0].Default {
		t.Fatal("an absent core id did not take its default")
	}
	// Turning one off is an explicit choice that survives a save/load.
	dir := t.TempDir()
	if err := SavePlugins(dir, (PluginState{}).WithCore(id, false)); err != nil {
		t.Fatal(err)
	}
	got, err := LoadPlugins(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.CoreOf(id) {
		t.Fatal("an explicit core-off was lost on the round trip")
	}
	if !got.CoreOf(plugin.CorePlugins[1].ID) {
		t.Fatal("another core plugin lost its default")
	}
}

func TestPluginState_normalizedDropsUnknownCore(t *testing.T) {
	st := PluginState{Core: map[string]bool{"not-a-core-plugin": true}}
	if st.normalized().Core["not-a-core-plugin"] {
		t.Fatal("an unknown core id was kept")
	}
}

func TestPluginState_optionsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := plugin.DefaultCoreOptions()
	want.DailyNotes.Folder = "Daily notes"
	want.DailyNotes.OpenOnStartup = true
	want.WebViewer.AdBlock = false
	st := PluginState{Core: map[string]bool{"daily-notes": true}}.
		WithOptions(want)
	if err := SavePlugins(dir, st); err != nil {
		t.Fatal(err)
	}
	got, err := LoadPlugins(dir)
	if err != nil {
		t.Fatal(err)
	}
	o := got.OptionsOf()
	if o.DailyNotes.Folder != "Daily notes" ||
		!o.DailyNotes.OpenOnStartup || o.WebViewer.AdBlock {
		t.Fatalf("options did not survive: %+v", o)
	}
	if o.Templates.TimeFormat != "HH:mm" {
		t.Fatal("a field the file left out did not take its default")
	}
	if (PluginState{}).OptionsOf().DailyNotes.DateFormat != "YYYY-MM-DD" {
		t.Fatal("a missing options field did not take its default")
	}
}
