package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo"
	"github.com/helightxu/xinkgo/internal/plugin"
)

func testPluginHost(t *testing.T) *pluginHost {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, "vault")
	return &pluginHost{
		st:        &appState{root: root},
		dir:       dir,
		userDir:   filepath.Join(dir, "plugins"),
		resources: filepath.Join(dir, "resources"),
		root:      root,
		assets:    map[string][]byte{},
		instances: map[string]*pluginInstance{},
		byPage:    map[*mygo.Page]*pluginInstance{},
		byHost:    map[string]*pluginInstance{},
		reports:   map[string]plugin.Report{},
		ui: pluginUI{supportedOnly: true,
			busy: map[string]bool{}},
	}
}

func TestDiscover_findsACompatiblePlugin(t *testing.T) {
	h := testPluginHost(t)
	dir := filepath.Join(h.userDir, "demo-addon")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"id":"demo-addon","name":"Demo","version":"1.0.0",` +
		`"minAppVersion":"0.9.12","description":"A demo","author":"Someone"}`
	mainJS := `"use strict";var o=require("obsidian");` +
		`module.exports=class extends o.Plugin{onload(){this.addCommand(` +
		`{id:"x",name:"X",callback:()=>new o.Notice("hi")})}}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"),
		[]byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.js"),
		[]byte(mainJS), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, issues := h.discover()
	if len(issues) != 0 {
		t.Fatalf("issues = %+v", issues)
	}
	one, ok := reports["demo-addon"]
	if !ok || !one.Supported || one.Level != plugin.LevelCore {
		t.Fatalf("report = %+v, present=%v", one, ok)
	}
}

func TestDiscover_badManifestBecomesAnIssue(t *testing.T) {
	h := testPluginHost(t)
	dir := filepath.Join(h.userDir, "broken-addon")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"),
		[]byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, issues := h.discover()
	if len(reports) != 0 || len(issues) != 1 {
		t.Fatalf("reports=%d issues=%d", len(reports), len(issues))
	}
}

func TestVaultPath_refusesTraversal(t *testing.T) {
	h := testPluginHost(t)
	if err := os.MkdirAll(h.st.root, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"../outside.md", "/etc/hosts",
		"a/../../outside.md", "C:/windows/system32"} {
		if _, err := h.vaultPath(raw); err == nil {
			t.Errorf("vaultPath(%q) was accepted", raw)
		}
	}
	if _, err := h.vaultPath("notes/one.md"); err != nil {
		t.Errorf("a normal vault path was refused: %v", err)
	}
}
