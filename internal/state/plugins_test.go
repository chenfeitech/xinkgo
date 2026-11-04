package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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
}
