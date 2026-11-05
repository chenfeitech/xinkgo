package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo"
	"github.com/helightxu/xinkgo/internal/plugin"
	"github.com/helightxu/xinkgo/internal/state"
	"github.com/helightxu/xinkgo/internal/view"
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

func TestPluginCatalog_coreAndOtherPagesDoNotOverlap(t *testing.T) {
	h := testPluginHost(t)
	h.catalog = []plugin.CatalogEntry{
		{ID: "core-addon", Name: "Core", Repo: "a/core"},
		{ID: "other-addon", Name: "Other", Repo: "b/other"},
	}
	h.compatible = []plugin.CompatibleEntry{
		{ID: "core-addon", Repo: "a/core", Level: "C4", Core: true},
		{ID: "other-addon", Repo: "b/other", Level: "C2"},
	}
	core := h.coreItems()
	other := h.browserItems(10)
	if len(core) != 1 || core[0].ID != "core-addon" {
		t.Fatalf("core items = %+v", core)
	}
	if len(other) != 1 || other[0].ID != "other-addon" {
		t.Fatalf("other items = %+v", other)
	}
}

func TestPluginViewState_hidesCoreFromOtherInstalled(t *testing.T) {
	h := testPluginHost(t)
	h.compatible = []plugin.CompatibleEntry{
		{ID: "core-addon", Repo: "a/core", Level: "C4", Core: true},
	}
	h.ui.installed = []plugin.Report{
		{ID: "core-addon", Name: "Core"},
		{ID: "other-addon", Name: "Other"},
	}
	state := h.uiSnapshot()
	if len(state.installed) != 1 ||
		state.installed[0].ID != "other-addon" ||
		!state.coreIDs["core-addon"] {
		t.Fatalf("installed = %+v coreIDs = %v", state.installed,
			state.coreIDs)
	}
}

func TestSetCore_andCoreOn(t *testing.T) {
	h := testPluginHost(t)
	id := plugin.CorePlugins[0].ID
	// An absent choice takes the registry default.
	if h.coreOn(id) != plugin.CorePlugins[0].Default {
		t.Fatal("a built-in core plugin did not start at its default")
	}
	h.setCore(id, false)
	if h.coreOn(id) {
		t.Fatal("setCore(false) did not turn the feature off")
	}
	// The choice is in plugins.json, ready for the next run.
	saved, err := state.LoadPlugins(h.dir)
	if err != nil || saved.CoreOf(id) {
		t.Fatalf("core choice not saved: %+v, %v", saved, err)
	}
	h.setCore(id, true)
	if !h.coreOn(id) {
		t.Fatal("setCore(true) did not turn the feature back on")
	}
	// An unknown id is ignored rather than written.
	h.setCore("not-a-core-plugin", false)
	if _, ok := h.saved.Core["not-a-core-plugin"]; ok {
		t.Fatal("an unknown core id was written")
	}
}

func TestViewRegister_recordsASurface(t *testing.T) {
	h := testPluginHost(t)
	inst := &pluginInstance{
		id:     "demo",
		grants: map[string]bool{string(plugin.CapabilityDOM): true},
		views:  map[string]view.PluginView{},
	}
	h.instances[inst.id] = inst
	raw, _ := json.Marshal(map[string]string{
		"viewType": "demo-graph", "name": "Demo graph",
	})
	if _, err := h.handleRPC(inst, "view.register", raw); err != nil {
		t.Fatalf("view.register was refused: %v", err)
	}
	one, ok := inst.views["demo:demo-graph"]
	if !ok || one.Name != "Demo graph" || one.ViewType != "demo-graph" {
		t.Fatalf("registered view = %+v, present=%v", one, ok)
	}
}

func TestViewRegister_refusedWithoutDom(t *testing.T) {
	h := testPluginHost(t)
	inst := &pluginInstance{
		id:     "demo",
		grants: map[string]bool{},
		views:  map[string]view.PluginView{},
	}
	raw, _ := json.Marshal(map[string]string{"viewType": "demo-graph"})
	if _, err := h.handleRPC(inst, "view.register", raw); err == nil {
		t.Fatal("view.register ran without the dom capability")
	}
}
