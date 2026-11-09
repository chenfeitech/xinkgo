package plugin

import (
	"regexp"
	"strings"
)

// The imports esbuild leaves external in a typical Obsidian plugin. The
// runtime shim returns inert stubs for these; it never gives a plugin
// Node, Electron or CodeMirror internals.
var deniedImports = map[string]bool{
	"electron": true, "fs": true, "path": true, "os": true,
	"child_process": true, "crypto": true, "util": true, "url": true,
	"events": true, "stream": true, "buffer": true, "http": true,
	"https": true, "net": true, "tls": true, "zlib": true,
	"assert": true, "querystring": true, "process": true,
	"module": true, "timers": true, "vm": true, "worker_threads": true,
}

// HostModules are the module names the shim resolves to its own exports.
// "obsidian" stays as the compatibility entry for external (L2) plugins;
// "xinkgo" is the native entry official plugins should use. Both return
// the same object today; a later step may give "xinkgo" a narrower API.
var HostModules = map[string]bool{
	"obsidian": true,
	"xinkgo":   true,
}

var (
	requirePattern = regexp.MustCompile(`require\(\s*["']([^"']+)["']\s*\)`)
	dynamicRequire = regexp.MustCompile(`require\(\s*[^"')]`)
	esmPattern     = regexp.MustCompile(`(?:^|[;{}]\s*)(?:import|export)\s+`)
	exportPattern  = regexp.MustCompile(`(?:module\.exports|exports\.default|__toCommonJS)`)
)

// apiRule is one API pattern and what the tolerant shim needs for it.
// Unsupported APIs are no longer a hard stop: they raise the level and
// add a capability, so the manager can say the plugin is degraded.
type apiRule struct {
	Name       string
	Level      Level
	Capability Capability
}

var apiRules = []apiRule{
	{"registerEditorExtension", LevelEditor, CapabilityEditorExt},
	{"registerEditorSuggest", LevelEditor, CapabilityEditorExt},
	{"EditorSuggest", LevelEditor, CapabilityEditorExt},
	{"WorkspaceLeaf", LevelDOM, CapabilityDOM},
	{"ItemView", LevelDOM, CapabilityDOM},
	{"containerEl", LevelDOM, CapabilityDOM},
	{"registerView", LevelDOM, CapabilityDOM},
	{"addSettingTab", LevelDOM, CapabilityDOM},
	{"PluginSettingTab", LevelDOM, CapabilityDOM},
	{"SettingTab", LevelDOM, CapabilityDOM},
	{"new Setting(", LevelDOM, CapabilityDOM},
	{"extends Modal", LevelDOM, CapabilityDOM},
	{"new Modal", LevelDOM, CapabilityDOM},
	{"Modal", LevelDOM, CapabilityDOM},
	{"FuzzySuggestModal", LevelDOM, CapabilityDOM},
	{"SuggestModal", LevelDOM, CapabilityDOM},
	{"new Menu(", LevelDOM, CapabilityDOM},
	{"MenuItem", LevelDOM, CapabilityDOM},
	{"registerMarkdownPostProcessor", LevelDOM, CapabilityDOM},
	{"registerMarkdownCodeBlockProcessor", LevelDOM, CapabilityDOM},
	{"MarkdownRenderer", LevelDOM, CapabilityDOM},
	{"registerHoverLinkSource", LevelDOM, CapabilityDOM},
	{"registerBasesView", LevelDOM, CapabilityDOM},
	{"registerExtensions", LevelDOM, CapabilityDOM},
	{"registerObsidianProtocolHandler", LevelDOM, CapabilityDOM},
	{"registerCliHandler", LevelDOM, CapabilityDOM},
	{"secretStorage", LevelDOM, CapabilityDOM},
	{"activeDocument", LevelDOM, CapabilityEvents},
	{"activeWindow", LevelDOM, CapabilityEvents},
	{"addIcon", LevelDOM, CapabilityUI},
	{"setIcon", LevelDOM, CapabilityUI},
	{"addRibbonIcon", LevelNative, CapabilityUI},
	{"addStatusBarItem", LevelNative, CapabilityUI},
	{"registerInterval", LevelNative, CapabilityEvents},
	{"registerDomEvent", LevelNative, CapabilityEvents},
	{"requestUrl", LevelNative, CapabilityNetwork},
	{"MetadataCache", LevelNative, CapabilityMetadata},
	{"getFileCache", LevelNative, CapabilityMetadata},
	{"getFirstLinkpathDest", LevelNative, CapabilityMetadata},
	{"vault.read", LevelNative, CapabilityVaultRead},
	{"cachedRead", LevelNative, CapabilityVaultRead},
	{"adapter.read", LevelNative, CapabilityVaultRead},
	{"vault.modify", LevelNative, CapabilityVaultWrite},
	{"vault.create", LevelNative, CapabilityVaultWrite},
	{"vault.delete", LevelNative, CapabilityVaultWrite},
	{"vault.rename", LevelNative, CapabilityVaultWrite},
	{"replaceSelection", LevelNative, CapabilityEditorWrite},
	{"editor.setValue", LevelNative, CapabilityEditorWrite},
	{"getValue", LevelNative, CapabilityEditorRead},
	{"getSelection", LevelNative, CapabilityEditorRead},
	{"getActiveViewOfType", LevelNative, CapabilityEditorRead},
	{"addCommand", LevelCore, CapabilityCommands},
	{"removeCommand", LevelCore, CapabilityCommands},
	{"new Notice", LevelCore, CapabilityNotice},
	{"Notice(", LevelCore, CapabilityNotice},
	{"loadData", LevelCore, CapabilityData},
	{"saveData", LevelCore, CapabilityData},
}

// ScanResult is what a textual scan of main.js found.
type ScanResult struct {
	Level        Level
	Supported    bool
	Degraded     bool
	Imports      []string
	APIs         []string
	Capabilities []Capability
	Issues       []Issue
}

// ScanSource scans a CommonJS bundle without executing it. ESM is the
// only hard stop; imports and APIs the host cannot fully implement become
// explicit degraded issues and capabilities.
func ScanSource(mainJS []byte) ScanResult {
	src := string(mainJS)
	out := ScanResult{Level: LevelCore, Supported: true}
	hard := false
	raise := func(level Level, capability Capability, issue Issue) {
		out.Issues = append(out.Issues, issue)
		if level > out.Level {
			out.Level = level
		}
		if capability != "" {
			out.Capabilities = append(out.Capabilities, capability)
		}
	}
	for _, match := range requirePattern.FindAllStringSubmatch(src, -1) {
		name := match[1]
		out.Imports = append(out.Imports, name)
		if HostModules[name] {
			continue
		}
		if strings.HasPrefix(name, "./") || strings.HasPrefix(name, "../") {
			raise(LevelEditor, CapabilityCompatStub,
				Issue{Code: "relativeImport", Arg: name})
			continue
		}
		if deniedImports[name] ||
			strings.HasPrefix(name, "@codemirror/") ||
			strings.HasPrefix(name, "@lezer/") ||
			strings.HasPrefix(name, "node:") ||
			strings.HasPrefix(name, "@electron/") {
			raise(LevelEditor, CapabilityCompatStub,
				Issue{Code: "stubImport", Arg: name})
			continue
		}
		raise(LevelEditor, CapabilityCompatStub,
			Issue{Code: "unknownImport", Arg: name})
	}
	if dynamicRequire.MatchString(src) {
		raise(LevelEditor, CapabilityCompatStub,
			Issue{Code: "dynamicRequire"})
	}
	if esmPattern.MatchString(src) {
		hard = true
		out.Issues = append(out.Issues, Issue{Code: "esmSyntax"})
	}
	if !exportPattern.MatchString(src) {
		hard = true
		out.Issues = append(out.Issues,
			Issue{Code: "missingDefaultExport"})
	}
	for _, rule := range apiRules {
		if !strings.Contains(src, rule.Name) {
			continue
		}
		out.APIs = append(out.APIs, rule.Name)
		raise(rule.Level, rule.Capability,
			Issue{Code: "shimAPI", Arg: rule.Name})
	}
	out.Imports = unique(out.Imports)
	out.APIs = unique(out.APIs)
	caps := make([]string, 0, len(out.Capabilities))
	for _, c := range out.Capabilities {
		caps = append(caps, string(c))
	}
	caps = unique(caps)
	out.Capabilities = make([]Capability, 0, len(caps))
	for _, c := range caps {
		out.Capabilities = append(out.Capabilities, Capability(c))
	}
	out.Degraded = out.Level.Degraded()
	out.Supported = out.Level != LevelUnsupported
	if hard {
		out.Level, out.Supported, out.Degraded = LevelUnsupported,
			false, false
	}
	return out
}

// ScanPackage joins a manifest and its bundle into the report the manager
// and browser show. hasCSS records whether styles.css came with the package.
func ScanPackage(m Manifest, mainJS []byte, dir, source string, hasCSS bool) Report {
	r := Report{
		ID: m.ID, Name: m.Name, Version: m.Version,
		Description: m.Description, Author: m.Author,
		Dir: dir, Source: source,
	}
	scan := ScanSource(mainJS)
	r.Level, r.Supported, r.Degraded = scan.Level, scan.Supported,
		scan.Degraded
	r.Imports, r.APIs = scan.Imports, scan.APIs
	r.Capabilities = scan.Capabilities
	r.Issues = append(r.Issues, scan.Issues...)
	if err := ValidateManifest(m, ""); err != nil {
		r.Issues = append(r.Issues, Issue{
			Code: "badManifest", Arg: err.Error(),
		})
		r.Level, r.Supported, r.Degraded = LevelUnsupported, false,
			false
	} else if !RequireAPIVersion(m.MinAppVersion) {
		r.Issues = append(r.Issues, Issue{
			Code: "apiProfile", Arg: m.MinAppVersion,
		})
		r.Supported = false
	}
	if hasCSS {
		r.Capabilities = append(r.Capabilities, CapabilityUI)
		caps := make([]string, 0, len(r.Capabilities))
		for _, c := range r.Capabilities {
			caps = append(caps, string(c))
		}
		caps = unique(caps)
		r.Capabilities = make([]Capability, 0, len(caps))
		for _, c := range caps {
			r.Capabilities = append(r.Capabilities, Capability(c))
		}
	}
	return r
}
