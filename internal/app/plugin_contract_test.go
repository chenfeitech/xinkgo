package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/plugin"
)

func TestRefreshLibrary_loadsTheBundledList(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller could not locate this test file")
	}
	h := testPluginHost(t)
	h.resources = filepath.Join(filepath.Dir(file), "..", "..",
		"resources")
	h.refreshLibrary()
	if len(h.library.Plugins) < 2 {
		t.Fatalf("library has %d entries", len(h.library.Plugins))
	}
	if h.library.Profile != plugin.APIProfile {
		t.Errorf("profile = %q, want %q", h.library.Profile,
			plugin.APIProfile)
	}
	for _, id := range []string{
		"xinkgo-demo-basic", "xinkgo-demo-full",
	} {
		if !h.libraryIDs[id] {
			t.Errorf("%s is not marked as a library plugin", id)
		}
	}
}

func TestRefreshLibrary_aMissingListIsNotFatal(t *testing.T) {
	h := testPluginHost(t) // resources points at an empty temp dir
	h.refreshLibrary()
	if len(h.library.Plugins) != 0 || len(h.libraryIDs) != 0 {
		t.Fatal("a missing library should leave an empty registry")
	}
}

func TestScanContracts_marksAStubPlugin(t *testing.T) {
	h := testPluginHost(t)
	write := func(id, main string) {
		dir := filepath.Join(h.userDir, id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		manifest := fmt.Sprintf(
			`{"id":%q,"name":%q,"version":"1.0.0",`+
				`"minAppVersion":"1.0.0","description":"d","author":"a"}`,
			id, id)
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"),
			[]byte(manifest), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "main.js"),
			[]byte(main), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// The native module name must pass the runtime scan too.
	write("good-plugin", `"use strict";var o=require("xinkgo");`+
		`module.exports=class extends o.Plugin{onload(){`+
		`this.addCommand({id:"x",name:"X",callback:()=>new o.Notice("hi")})}}`)
	write("stub-plugin", `"use strict";var o=require("obsidian");`+
		`module.exports=class extends o.Plugin{onload(){`+
		`this.app.metadataCache.getFileCache(null)}}`)
	reports, issues := h.discover()
	if len(issues) != 0 {
		t.Fatalf("discovery issues = %+v", issues)
	}
	contracts := h.scanContracts(reports)
	if !contracts["good-plugin"].OK {
		t.Errorf("a plain command plugin failed: %+v",
			contracts["good-plugin"].Violations)
	}
	if contracts["stub-plugin"].OK {
		t.Error("a plugin using an empty metadata cache passed")
	}
	if !contracts["stub-plugin"].HasViolation("stubAPI") {
		t.Errorf("missing stubAPI: %+v",
			contracts["stub-plugin"].Violations)
	}
}

func TestEnable_refusesALibraryPluginThatFailedTheContract(t *testing.T) {
	h := testPluginHost(t)
	h.reports["broken"] = plugin.Report{
		ID: "broken", Name: "Broken", Supported: true,
		Level: plugin.LevelNative,
	}
	h.libraryIDs["broken"] = true
	h.contracts["broken"] = plugin.ContractResult{
		OK: false,
		Violations: []plugin.Violation{
			{Code: "stubAPI", Arg: "getFileCache"},
		},
	}
	h.enable("broken", false)
	if _, running := h.instances["broken"]; running {
		t.Error("a failed-contract library plugin was given a WebView")
	}
	if !strings.Contains(h.ui.status, locale.PluginContractRefused()) {
		t.Errorf("status = %q, want the contract refusal", h.ui.status)
	}
}

func TestContractRefused_onlyAppliesToCheckedLibraryPlugins(t *testing.T) {
	h := testPluginHost(t)
	h.contracts["external"] = plugin.ContractResult{OK: false}
	if h.contractRefused("external") {
		t.Error("an external plugin was refused by the library contract")
	}
	h.libraryIDs["checked"] = true
	h.contracts["checked"] = plugin.ContractResult{OK: false}
	if !h.contractRefused("checked") {
		t.Error("a failed-contract library plugin was not refused")
	}
	h.libraryIDs["unchecked"] = true
	if h.contractRefused("unchecked") {
		t.Error("a library plugin with no verdict was refused")
	}
}

func TestInstallGrants_neverWritesImplicitGrants(t *testing.T) {
	report := plugin.Report{Capabilities: plugin.AllCapabilities}
	if got := installGrants(report); len(got) != 0 {
		t.Fatalf("install grants = %v, want none", got)
	}
}

func TestUISnapshot_copiesLibraryAndContractState(t *testing.T) {
	h := testPluginHost(t)
	h.library = plugin.Library{Plugins: []plugin.LibraryPlugin{
		{ID: "verified", Name: "Verified"},
	}}
	h.libraryIDs["verified"] = true
	h.contracts["verified"] = plugin.ContractResult{OK: true}
	state := h.uiSnapshot()
	if len(state.library) != 1 || !state.libraryIDs["verified"] ||
		!state.contracts["verified"].OK {
		t.Fatalf("snapshot = %+v", state)
	}
}

func TestLibraryView_showsVerifiedEntries(t *testing.T) {
	h := pluginPaneHost()
	state := pluginViewState{
		library: []plugin.LibraryPlugin{{
			ID: "xinkgo-demo-full", Name: "Xinkgo Demo: Full",
			Version: "1.0.0", Kind: "reference",
			Capabilities: []string{"data", "commands"},
		}},
		libraryIDs:   map[string]bool{"xinkgo-demo-full": true},
		installedIDs: map[string]bool{"xinkgo-demo-full": true},
		contracts: map[string]plugin.ContractResult{
			"xinkgo-demo-full": {OK: true},
		},
		busy: map[string]bool{},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		h.libraryView(c, c.Theme(), state)
	}, 820, 680)
	for _, want := range []string{
		"Xinkgo Demo: Full", locale.PluginLibraryVerified(),
		locale.PluginLibraryInstalled(),
	} {
		if !tt.HasText(want) {
			t.Errorf("library view does not say %q", want)
		}
	}
}

func TestLibraryView_anEmptyLibrarySaysSo(t *testing.T) {
	h := pluginPaneHost()
	tt := ui.NewTester(func(c *ui.Context) {
		h.libraryView(c, c.Theme(), pluginViewState{})
	}, 820, 680)
	if !tt.HasText(locale.PluginLibraryEmpty()) {
		t.Error("an empty library has no explanation")
	}
}
