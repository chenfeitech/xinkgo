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

// demoPluginsDir resolves the examples folder from this test file's own
// location, so the assertions do not depend on the go command's cwd.
func demoPluginsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller could not locate this test file")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "examples", "plugins")
}

type demoPackage struct {
	manifest Manifest
	mainJS   []byte
	report   Report
}

func loadDemoPackage(t *testing.T, root, id string) demoPackage {
	t.Helper()
	dir := filepath.Join(root, id)
	manifestData, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatalf("%s manifest.json: %v", id, err)
	}
	m, err := ParseManifest(manifestData)
	if err != nil {
		t.Fatalf("%s ParseManifest: %v", id, err)
	}
	if err := ValidateManifest(m, id); err != nil {
		t.Fatalf("%s folder must match the id: %v", id, err)
	}
	mainJS, err := os.ReadFile(filepath.Join(dir, "main.js"))
	if err != nil {
		t.Fatalf("%s main.js: %v", id, err)
	}
	css, err := os.ReadFile(filepath.Join(dir, "styles.css"))
	if err != nil {
		t.Fatalf("%s styles.css: %v", id, err)
	}
	if len(css) == 0 {
		t.Errorf("%s has an empty styles.css", id)
	}
	return demoPackage{
		manifest: m, mainJS: mainJS,
		report: ScanPackage(m, mainJS, dir, "examples", len(css) > 0),
	}
}

func demoHasIssue(r Report, code string) bool {
	for _, one := range r.Issues {
		if one.Code == code {
			return true
		}
	}
	return false
}

// supportedCapabilities is the full set of capabilities Xinkgo actually
// implements. A demo must not ask for anything outside this list.
var supportedCapabilities = []Capability{
	CapabilityData, CapabilityCommands, CapabilityNotice,
	CapabilityVaultRead, CapabilityVaultWrite,
	CapabilityEditorRead, CapabilityEditorWrite,
	CapabilityNetwork, CapabilityEvents, CapabilityUI, CapabilityDOM,
}

// unsupportedAPIs are Obsidian APIs whose Xinkgo implementation is a no-op,
// an empty object or an inert proxy. They must not appear in either demo.
var unsupportedAPIs = []string{
	"addSettingTab", "PluginSettingTab", "new Setting(",
	"new obsidian.Setting(", "SettingTab",
	"new obsidian.Modal", "SuggestModal", "FuzzySuggestModal",
	"new obsidian.Menu(", "MenuItem",
	"MarkdownRenderer", "HoverPopover",
	"registerMarkdownPostProcessor", "registerMarkdownCodeBlockProcessor",
	"registerEditorExtension", "EditorSuggest", "registerEditorSuggest",
	"registerObsidianProtocolHandler", "registerExtensions",
	"registerHoverLinkSource", "registerBasesView", "registerCliHandler",
	"secretStorage", "addIcon", "setIcon",
	"getFileCache", "getFirstLinkpathDest", "MetadataCache", "metadataCache",
	"require(\"path\")",
}

func demoHasIssueArg(r Report, code, arg string) bool {
	for _, one := range r.Issues {
		if one.Code == code && one.Arg == arg {
			return true
		}
	}
	return false
}

// A shimAPI issue is an annotation, not a failure: ScanSource emits one for
// every API pattern it finds. The demos only have to avoid the hard stops.
func TestDemoPlugins_packagesScanWithoutHardErrors(t *testing.T) {
	root := demoPluginsDir(t)
	for _, id := range []string{"xinkgo-demo-basic", "xinkgo-demo-full"} {
		pkg := loadDemoPackage(t, root, id)
		if !RequireAPIVersion(pkg.manifest.MinAppVersion) {
			t.Errorf("%s asks for a newer API profile than %s",
				id, APIProfile)
		}
		for _, code := range []string{
			"esmSyntax", "missingDefaultExport", "badManifest",
			"apiProfile", "dynamicRequire", "relativeImport",
			"unknownImport", "stubImport",
		} {
			if demoHasIssue(pkg.report, code) {
				t.Errorf("%s: unexpected %s issue: %+v",
					id, code, pkg.report.Issues)
			}
		}
		if !pkg.report.Supported {
			t.Errorf("%s is not supported: %+v", id, pkg.report)
		}
	}
}

func TestDemoPlugins_basicStaysOnNativeAdapters(t *testing.T) {
	pkg := loadDemoPackage(t, demoPluginsDir(t), "xinkgo-demo-basic")
	r := pkg.report
	if r.Level != LevelNative || r.Degraded || !r.Supported {
		t.Fatalf("basic report = %+v, want C2 / supported / not degraded", r)
	}
	want := []Capability{
		CapabilityData, CapabilityCommands, CapabilityNotice,
		CapabilityUI, CapabilityVaultRead,
		CapabilityEditorRead, CapabilityEditorWrite,
	}
	for _, one := range want {
		if !r.HasCapability(one) {
			t.Errorf("basic is missing capability %s", one)
		}
	}
	for _, one := range []Capability{
		CapabilityVaultWrite, CapabilityMetadata, CapabilityNetwork,
		CapabilityEvents, CapabilityDOM, CapabilityEditorExt,
		CapabilityCompatStub,
	} {
		if r.HasCapability(one) {
			t.Errorf("basic unexpectedly asks for %s", one)
		}
	}
	if len(r.Capabilities) != len(want) {
		t.Errorf("basic capabilities = %v, want exactly the C1/C2 set",
			r.Capabilities)
	}
	if len(r.Imports) != 1 || r.Imports[0] != "obsidian" {
		t.Errorf("basic imports = %v, want only obsidian", r.Imports)
	}
}

func TestDemoPlugins_fullCoversEverySupportedCapability(t *testing.T) {
	pkg := loadDemoPackage(t, demoPluginsDir(t), "xinkgo-demo-full")
	r := pkg.report
	if r.Level != LevelDOM || !r.Supported {
		t.Fatalf("full report = %+v, want C3 / supported", r)
	}
	for _, one := range supportedCapabilities {
		if !r.HasCapability(one) {
			t.Errorf("full is missing supported capability %s", one)
		}
	}
	if len(r.Capabilities) != len(supportedCapabilities) {
		t.Errorf("full capabilities = %v, want exactly %d supported ones",
			r.Capabilities, len(supportedCapabilities))
	}
	for _, one := range []Capability{
		CapabilityMetadata, CapabilityEditorExt, CapabilityCompatStub,
	} {
		if r.HasCapability(one) {
			t.Errorf("full asks for unsupported capability %s", one)
		}
	}
	if demoHasIssue(r, "stubImport") {
		t.Errorf("full pulls in an external module: %+v", r.Issues)
	}
	if len(r.Imports) != 1 || r.Imports[0] != "obsidian" {
		t.Errorf("full imports = %v, want only obsidian", r.Imports)
	}
}

func TestDemoPlugins_useOnlySupportedAPIs(t *testing.T) {
	root := demoPluginsDir(t)
	for _, id := range []string{"xinkgo-demo-basic", "xinkgo-demo-full"} {
		pkg := loadDemoPackage(t, root, id)
		src := string(pkg.mainJS)
		for _, api := range unsupportedAPIs {
			if strings.Contains(src, api) {
				t.Errorf("%s uses an API Xinkgo does not fully support: %s",
					id, api)
			}
		}
	}
}

func TestDemoPlugins_fullManifestHintsMatchTheScan(t *testing.T) {
	pkg := loadDemoPackage(t, demoPluginsDir(t), "xinkgo-demo-full")
	if pkg.manifest.Xinkgo == nil {
		t.Fatal("full manifest has no xinkgo capabilities hint")
	}
	hints := append([]string(nil), pkg.manifest.Xinkgo.Capabilities...)
	sort.Strings(hints)
	scanned := make([]string, 0, len(pkg.report.Capabilities))
	for _, one := range pkg.report.Capabilities {
		scanned = append(scanned, string(one))
	}
	sort.Strings(scanned)
	if !reflect.DeepEqual(hints, scanned) {
		t.Errorf("manifest hints %v do not match the scan %v",
			hints, scanned)
	}
}

func TestDemoPlugins_fullShipsTheAssetItsViewReferences(t *testing.T) {
	root := demoPluginsDir(t)
	pkg := loadDemoPackage(t, root, "xinkgo-demo-full")
	if !strings.Contains(string(pkg.mainJS), "/plugin/assets/badge.svg") {
		t.Fatal("the full demo view no longer references badge.svg")
	}
	data, err := os.ReadFile(filepath.Join(root, "xinkgo-demo-full",
		"assets", "badge.svg"))
	if err != nil || len(data) == 0 {
		t.Fatalf("badge.svg is missing or empty: %v", err)
	}
}
