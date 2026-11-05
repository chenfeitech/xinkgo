package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/helightxu/xinkgo/internal/plugin"
	"github.com/helightxu/xinkgo/internal/state"
)

// writePluginPackage lays down the smallest package discover() accepts,
// in one of the three plugin folders, and returns its Report.
func writePluginPackage(t *testing.T, h *pluginHost, source, id string) plugin.Report {
	t.Helper()
	var dir string
	switch source {
	case "user":
		dir = filepath.Join(h.userDir, id)
	case "compat":
		dir = filepath.Join(h.root, ".obsidian", "plugins", id)
	case "vault":
		dir = filepath.Join(h.root, ".xink", "plugins", id)
	default:
		t.Fatalf("unknown source %q", source)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"id":"` + id + `","name":"` + id + `","version":"1.0.0",` +
		`"minAppVersion":"0.9.12","description":"demo","author":"Someone"}`
	for name, data := range map[string]string{
		"manifest.json": manifest,
		"main.js":       `"use strict";module.exports=class extends require("obsidian").Plugin{}`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return plugin.Report{ID: id, Name: id, Dir: dir, Source: source,
		Version: "1.0.0", Author: "Someone", Supported: true}
}

// uninstallHost is a host whose state has one enabled, granted plugin and
// one other plugin whose choices must survive the removal.
func uninstallHost(t *testing.T) *pluginHost {
	t.Helper()
	h := testPluginHost(t)
	h.ui.open = map[string]bool{"demo": true}
	h.saved = state.PluginState{
		Enabled: map[string]bool{"demo": true, "other": true},
		Grants: map[string][]string{
			"demo":  {"data", "commands"},
			"other": {"dom"},
		},
	}
	if err := state.SavePlugins(h.dir, h.saved); err != nil {
		t.Fatal(err)
	}
	return h
}

func TestPluginUninstall_removesPackageAndForgetsState(t *testing.T) {
	h := uninstallHost(t)
	one := writePluginPackage(t, h, "user", "demo")

	h.uninstall(one)

	if _, err := os.Stat(one.Dir); !os.IsNotExist(err) {
		t.Fatalf("the plugin package is still on disk: %v", err)
	}
	if h.saved.EnabledOf("demo") {
		t.Error("the enabled flag survived the removal")
	}
	if len(h.saved.GrantsOf("demo")) != 0 {
		t.Error("the grants survived the removal")
	}
	// The other plugin's choices are untouched.
	if !h.saved.EnabledOf("other") || len(h.saved.GrantsOf("other")) != 1 {
		t.Error("another plugin's choices were lost")
	}
	saved, err := state.LoadPlugins(h.dir)
	if err != nil {
		t.Fatal(err)
	}
	if saved.EnabledOf("demo") || len(saved.GrantsOf("demo")) != 0 {
		t.Error("plugins.json still remembers the removed plugin")
	}
	// refresh() rebuilt the installed list without it, and the card's
	// fold state was dropped too.
	if _, ok := h.reports["demo"]; ok {
		t.Error("the report is still in the host")
	}
	for _, one := range h.ui.installed {
		if one.ID == "demo" {
			t.Error("the removed plugin is still listed as installed")
		}
	}
	if h.ui.open["demo"] {
		t.Error("the removed card's fold state was kept")
	}
}

func TestPluginUninstall_worksForAllThreeSources(t *testing.T) {
	for _, source := range []string{"user", "compat", "vault"} {
		t.Run(source, func(t *testing.T) {
			h := uninstallHost(t)
			one := writePluginPackage(t, h, source, "demo")
			h.uninstall(one)
			if _, err := os.Stat(one.Dir); !os.IsNotExist(err) {
				t.Fatalf("the %s package is still on disk: %v", source, err)
			}
			if h.saved.EnabledOf("demo") {
				t.Error("the enabled flag survived the removal")
			}
		})
	}
}

func TestPluginUninstall_refusesUnrelatedPath(t *testing.T) {
	h := uninstallHost(t)
	one := plugin.Report{ID: "demo", Name: "demo",
		Dir: filepath.Join(h.dir, "not-a-plugin-folder", "demo")}
	if err := os.MkdirAll(one.Dir, 0o755); err != nil {
		t.Fatal(err)
	}

	h.uninstall(one)

	if _, err := os.Stat(one.Dir); err != nil {
		t.Fatal("a path outside the plugin folders was removed")
	}
	// A refused removal must not forget the plugin's choices.
	if !h.saved.EnabledOf("demo") || len(h.saved.GrantsOf("demo")) == 0 {
		t.Error("a refused removal still changed the saved state")
	}
}

func TestPluginPackageDir_guardsTheRemoval(t *testing.T) {
	h := testPluginHost(t)
	cases := []struct {
		name string
		dir  string
		want bool
	}{
		{"user child", filepath.Join(h.userDir, "demo"), true},
		{"compat child", filepath.Join(h.root, ".obsidian", "plugins", "demo"), true},
		{"vault child", filepath.Join(h.root, ".xink", "plugins", "demo"), true},
		{"blank", "", false},
		{"relative", "plugins/demo", false},
		{"the user folder itself", h.userDir, false},
		{"a grandchild", filepath.Join(h.userDir, "demo", "nested"), false},
		{"somewhere else", filepath.Join(h.dir, "elsewhere", "demo"), false},
	}
	for _, tc := range cases {
		if got := h.pluginPackageDir(tc.dir); got != tc.want {
			t.Errorf("%s: pluginPackageDir(%q) = %v, want %v",
				tc.name, tc.dir, got, tc.want)
		}
	}
}
