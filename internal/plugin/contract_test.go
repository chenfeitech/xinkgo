package plugin

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func contractRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller could not locate this test file")
	}
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func TestValidateContract_acceptsBothDemos(t *testing.T) {
	root := demoPluginsDir(t)
	for _, id := range []string{
		"xinkgo-demo-basic", "xinkgo-demo-full",
	} {
		pkg := loadDemoPackage(t, root, id)
		got := ValidateContract(pkg.report, pkg.mainJS)
		if !got.OK {
			t.Errorf("%s failed the contract: %+v", id, got.Violations)
		}
		if got.Level > MaxContractLevel {
			t.Errorf("%s level = %s, above %s", id,
				got.Level.Code(), MaxContractLevel.Code())
		}
	}
}

// A C2, non-degraded report can still be broken: this mirrors the locally
// installed obsidian-sort-and-permute-lines, whose metadata cache is {}.
func TestValidateContract_rejectsEmptyMetadataAtC2(t *testing.T) {
	r := Report{
		ID:        "sort-lines",
		Supported: true, Level: LevelNative, Degraded: false,
		Capabilities: []Capability{
			CapabilityCommands, CapabilityEditorRead, CapabilityMetadata,
		},
		Imports: []string{"obsidian"},
		Issues:  []Issue{{Code: "shimAPI", Arg: "getFileCache"}},
	}
	src := []byte(`var o=require("obsidian");` +
		`var c=o.app.metadataCache.getFileCache(f);`)
	got := ValidateContract(r, src)
	if got.OK {
		t.Fatal("a plugin depending on an empty metadata cache passed")
	}
	if !got.HasViolation("unsupportedCapability") {
		t.Errorf("missing unsupportedCapability: %+v", got.Violations)
	}
	if !got.HasViolation("stubAPI") {
		t.Errorf("missing stubAPI: %+v", got.Violations)
	}
	found := false
	for _, one := range got.Violations {
		if one.Code == "unsupportedCapability" &&
			one.Arg == string(CapabilityMetadata) {
			found = true
		}
	}
	if !found {
		t.Errorf("metadata was not named: %+v", got.Violations)
	}
}

func TestValidateContract_rejectsLevelEditorAndStubs(t *testing.T) {
	r := Report{
		ID:        "old-full",
		Supported: true, Level: LevelEditor, Degraded: true,
		Capabilities: []Capability{
			CapabilityEditorExt, CapabilityCompatStub, CapabilityDOM,
		},
		Imports: []string{"obsidian", "path"},
		Issues:  []Issue{{Code: "stubImport", Arg: "path"}},
	}
	src := []byte(`var p=require("path");` +
		`this.registerEditorExtension(x);this.addSettingTab(tab);`)
	got := ValidateContract(r, src)
	for _, code := range []string{
		"levelTooHigh", "externalImport", "scanIssue:stubImport",
		"unsupportedCapability", "stubAPI",
	} {
		if !got.HasViolation(code) {
			t.Errorf("missing %s: %+v", code, got.Violations)
		}
	}
}

func TestValidateContract_rejectsHardScanFailures(t *testing.T) {
	r := Report{
		ID: "broken", Supported: false, Level: LevelUnsupported,
		Issues: []Issue{{Code: "esmSyntax"}},
	}
	got := ValidateContract(r,
		[]byte("export default class Broken {}"))
	if !got.HasViolation("notSupported") ||
		!got.HasViolation("scanIssue:esmSyntax") {
		t.Errorf("hard scan failures were ignored: %+v", got.Violations)
	}
}

// A plugin can look perfectly fine on the report: C3, every capability on
// the whitelist, no hard scan issue. It still is not fully working when its
// menu/modal/settings UI is a no-op. This mirrors the two locally installed
// community plugins, highlightr and markdown-formatting-assistant.
func TestValidateContract_rejectsStubUIWithSupportedCapabilities(t *testing.T) {
	r := Report{
		ID:        "highlighter",
		Supported: true, Level: LevelDOM, Degraded: true,
		Capabilities: []Capability{
			CapabilityCommands, CapabilityData, CapabilityDOM,
			CapabilityEditorRead, CapabilityEditorWrite,
			CapabilityNotice, CapabilityUI,
		},
		Imports: []string{"obsidian"},
		Issues:  []Issue{{Code: "shimAPI", Arg: "addSettingTab"}},
	}
	src := []byte(`var o=require("obsidian");` +
		`new o.Menu(app);this.addSettingTab(new o.PluginSettingTab(app,this));`)
	got := ValidateContract(r, src)
	if got.OK {
		t.Fatal("stub-only UI passed the contract")
	}
	if !got.HasViolation("stubAPI") {
		t.Errorf("missing stubAPI: %+v", got.Violations)
	}
	if got.HasViolation("unsupportedCapability") ||
		got.HasViolation("levelTooHigh") {
		t.Errorf("the report was clean; only stub APIs should fail: %+v",
			got.Violations)
	}
}

func TestSupportedCapability_matchesAllCapabilities(t *testing.T) {
	for _, one := range SupportedCapabilities {
		if !SupportedCapability(one) {
			t.Errorf("%s is not on the supported list", one)
		}
	}
	for _, one := range UnsupportedCapabilities {
		if SupportedCapability(one) {
			t.Errorf("%s is incorrectly supported", one)
		}
	}
	if len(SupportedCapabilities)+len(UnsupportedCapabilities) !=
		len(AllCapabilities) {
		t.Fatalf("%d + %d != %d capabilities",
			len(SupportedCapabilities), len(UnsupportedCapabilities),
			len(AllCapabilities))
	}
}

const oneLibraryEntry = `{"schema":1,"profile":"1.0.0","plugins":[` +
	`{"id":"demo","name":"Demo","version":"1.0.0","kind":"reference",` +
	`"path":"examples/plugins/demo","capabilities":["data"],` +
	`"verification":{"static":"pass","runtime":"pass",` +
	`"checkedAt":"2026-11-09"}}]}`

func TestParseLibrary_rejectsBadLists(t *testing.T) {
	if _, err := ParseLibrary([]byte(oneLibraryEntry)); err != nil {
		t.Fatalf("a valid library was refused: %v", err)
	}
	onePlugin := `{"id":"demo","name":"Demo","version":"1.0.0",` +
		`"kind":"reference","path":"examples/plugins/demo",` +
		`"capabilities":["data"],` +
		`"verification":{"static":"pass","runtime":"pass",` +
		`"checkedAt":"2026-11-09"}}`
	duplicate := `{"schema":1,"profile":"1.0.0","plugins":[` +
		onePlugin + `,` + onePlugin + `]}`
	cases := map[string]string{
		"empty":                  "",
		"bad json":               "{",
		"wrong schema":           strings.Replace(oneLibraryEntry, `"schema":1`, `"schema":2`, 1),
		"wrong profile":          strings.Replace(oneLibraryEntry, `"profile":"1.0.0"`, `"profile":"9.9.9"`, 1),
		"duplicate id":           duplicate,
		"unsupported capability": strings.Replace(oneLibraryEntry, `["data"]`, `["metadata"]`, 1),
		"bad id":                 strings.Replace(oneLibraryEntry, `"id":"demo"`, `"id":"Bad_ID"`, 1),
		"missing verification":   strings.Replace(oneLibraryEntry, `"static":"pass"`, `"static":"fail"`, 1),
		"unknown kind":           strings.Replace(oneLibraryEntry, `"kind":"reference"`, `"kind":"other"`, 1),
	}
	for name, data := range cases {
		if _, err := ParseLibrary([]byte(data)); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

// The offline gate: every plugin in the official library must really exist,
// match its manifest, pass the contract and declare exactly what it scans as.
func TestLibrary_everyEntryPassesTheContract(t *testing.T) {
	root := contractRepoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "resources",
		"xinkgo-plugins", "library.json"))
	if err != nil {
		t.Fatal(err)
	}
	lib, err := ParseLibrary(data)
	if err != nil {
		t.Fatalf("library.json: %v", err)
	}
	if len(lib.Plugins) < 2 {
		t.Fatalf("library has only %d entries", len(lib.Plugins))
	}
	for _, entry := range lib.Plugins {
		dir := filepath.Join(root, filepath.FromSlash(entry.Path))
		manifestData, err := os.ReadFile(
			filepath.Join(dir, "manifest.json"))
		if err != nil {
			t.Errorf("%s: %v", entry.ID, err)
			continue
		}
		m, err := ParseManifest(manifestData)
		if err != nil {
			t.Errorf("%s: %v", entry.ID, err)
			continue
		}
		if err := ValidateManifest(m, filepath.Base(dir)); err != nil {
			t.Errorf("%s: %v", entry.ID, err)
		}
		if m.ID != entry.ID || m.Name != entry.Name ||
			m.Version != entry.Version {
			t.Errorf("%s: library metadata does not match manifest", entry.ID)
		}
		js, err := os.ReadFile(filepath.Join(dir, "main.js"))
		if err != nil {
			t.Errorf("%s: %v", entry.ID, err)
			continue
		}
		_, cssErr := os.Stat(filepath.Join(dir, "styles.css"))
		r := ScanPackage(m, js, dir, "library", cssErr == nil)
		if !r.Supported || r.Level > MaxContractLevel {
			t.Errorf("%s: report = %+v", entry.ID, r)
		}
		got := ValidateContract(r, js)
		if !got.OK {
			t.Errorf("%s failed the contract: %+v", entry.ID,
				got.Violations)
		}
		declared := append([]string(nil), entry.Capabilities...)
		sort.Strings(declared)
		scanned := make([]string, 0, len(r.Capabilities))
		for _, one := range r.Capabilities {
			scanned = append(scanned, string(one))
		}
		sort.Strings(scanned)
		if !reflect.DeepEqual(declared, scanned) {
			t.Errorf("%s declares %v but scans %v",
				entry.ID, declared, scanned)
		}
	}
}
