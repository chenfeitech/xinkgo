package plugin

import (
	"regexp"
	"strings"
)

// The imports esbuild leaves external in a typical Obsidian plugin. A
// plugin needing one of these must run inside Obsidian or a host that
// reimplements Electron/Node/CodeMirror.
var deniedImports = map[string]bool{
	"electron": true, "fs": true, "path": true, "os": true,
	"child_process": true, "crypto": true, "util": true, "url": true,
	"events": true, "stream": true, "buffer": true, "http": true,
	"https": true, "net": true, "tls": true, "zlib": true,
	"assert": true, "querystring": true, "process": true,
	"module": true, "timers": true, "vm": true, "worker_threads": true,
}

var (
	requirePattern = regexp.MustCompile(`require\(\s*["']([^"']+)["']\s*\)`)
	dynamicRequire = regexp.MustCompile(`require\(\s*[^"')]`)
	esmPattern     = regexp.MustCompile(`(?:^|[;{}]\s*)(?:import|export)\s+`)
	exportPattern  = regexp.MustCompile(`(?:module\.exports|exports\.default|__toCommonJS)`)
)

// apiRule is one API pattern and what it means for compatibility.
type apiRule struct {
	Name        string
	Level       Level
	Capability  Capability
	Unsupported bool
}

// apiRules are conservative textual probes. They are not a security
// boundary: the runtime require proxy and Go bridge still enforce every
// capability.
var apiRules = []apiRule{
	{"registerEditorExtension", LevelEditor, "", true},
	{"registerEditorSuggest", LevelEditor, "", true},
	{"EditorSuggest", LevelEditor, "", true},
	{"WorkspaceLeaf", LevelDOM, "", true},
	{"ItemView", LevelDOM, "", true},
	{"containerEl", LevelDOM, "", true},
	{"registerView", LevelDOM, "", true},
	{"addSettingTab", LevelDOM, "", true},
	{"PluginSettingTab", LevelDOM, "", true},
	{"SettingTab", LevelDOM, "", true},
	{"new Setting(", LevelDOM, "", true},
	{"extends Modal", LevelDOM, "", true},
	{"new Modal", LevelDOM, "", true},
	{"Modal", LevelDOM, "", true},
	{"FuzzySuggestModal", LevelDOM, "", true},
	{"SuggestModal", LevelDOM, "", true},
	{"new Menu(", LevelDOM, "", true},
	{"MenuItem", LevelDOM, "", true},
	{"registerMarkdownPostProcessor", LevelDOM, "", true},
	{"registerMarkdownCodeBlockProcessor", LevelDOM, "", true},
	{"MarkdownRenderer", LevelDOM, "", true},
	{"registerHoverLinkSource", LevelDOM, "", true},
	{"registerBasesView", LevelDOM, "", true},
	{"registerExtensions", LevelDOM, "", true},
	{"registerObsidianProtocolHandler", LevelDOM, "", true},
	{"registerCliHandler", LevelDOM, "", true},
	{"secretStorage", LevelDOM, "", true},
	{"activeDocument", LevelDOM, "", true},
	{"activeWindow", LevelDOM, "", true},
	{"addIcon", LevelDOM, "", true},
	{"setIcon", LevelDOM, "", true},
	{"addRibbonIcon", LevelNative, CapabilityUI, false},
	{"addStatusBarItem", LevelNative, CapabilityUI, false},
	{"registerInterval", LevelNative, CapabilityEvents, false},
	{"registerDomEvent", LevelNative, CapabilityEvents, false},
	{"requestUrl", LevelNative, CapabilityNetwork, false},
	{"MetadataCache", LevelNative, CapabilityMetadata, false},
	{"getFileCache", LevelNative, CapabilityMetadata, false},
	{"getFirstLinkpathDest", LevelNative, CapabilityMetadata, false},
	{"vault.read", LevelNative, CapabilityVaultRead, false},
	{"cachedRead", LevelNative, CapabilityVaultRead, false},
	{"adapter.read", LevelNative, CapabilityVaultRead, false},
	{"vault.modify", LevelNative, CapabilityVaultWrite, false},
	{"vault.create", LevelNative, CapabilityVaultWrite, false},
	{"vault.delete", LevelNative, CapabilityVaultWrite, false},
	{"vault.rename", LevelNative, CapabilityVaultWrite, false},
	{"replaceSelection", LevelNative, CapabilityEditorWrite, false},
	{"editor.setValue", LevelNative, CapabilityEditorWrite, false},
	{"getValue", LevelNative, CapabilityEditorRead, false},
	{"getSelection", LevelNative, CapabilityEditorRead, false},
	{"getActiveViewOfType", LevelNative, CapabilityEditorRead, false},
	{"addCommand", LevelCore, CapabilityCommands, false},
	{"removeCommand", LevelCore, CapabilityCommands, false},
	{"new Notice", LevelCore, CapabilityNotice, false},
	{"Notice(", LevelCore, CapabilityNotice, false},
	{"loadData", LevelCore, CapabilityData, false},
	{"saveData", LevelCore, CapabilityData, false},
}

// ScanResult is what a textual scan of main.js found.
type ScanResult struct {
	Level        Level
	Supported    bool
	Imports      []string
	APIs         []string
	Capabilities []Capability
	Issues       []Issue
}

// ScanSource scans a CommonJS bundle without executing it.
func ScanSource(mainJS []byte) ScanResult {
	src := string(mainJS)
	out := ScanResult{Level: LevelCore, Supported: true}
	for _, match := range requirePattern.FindAllStringSubmatch(src, -1) {
		name := match[1]
		out.Imports = append(out.Imports, name)
		if name == "obsidian" || strings.HasPrefix(name, "./") ||
			strings.HasPrefix(name, "../") {
			continue
		}
		if deniedImports[name] || strings.HasPrefix(name, "@codemirror/") ||
			strings.HasPrefix(name, "@lezer/") ||
			strings.HasPrefix(name, "node:") {
			out.Issues = append(out.Issues, Issue{
				Code: "externalImport", Arg: name,
			})
			out.Level = LevelEditor
			out.Supported = false
			continue
		}
		out.Issues = append(out.Issues, Issue{
			Code: "unknownImport", Arg: name,
		})
		out.Level = LevelEditor
		out.Supported = false
	}
	if dynamicRequire.MatchString(src) {
		out.Issues = append(out.Issues, Issue{Code: "dynamicRequire"})
		out.Level = LevelEditor
		out.Supported = false
	}
	if esmPattern.MatchString(src) {
		out.Issues = append(out.Issues, Issue{Code: "esmSyntax"})
		out.Level = LevelUnsupported
		out.Supported = false
	}
	if !exportPattern.MatchString(src) {
		out.Issues = append(out.Issues, Issue{Code: "missingDefaultExport"})
		out.Level = LevelUnsupported
		out.Supported = false
	}
	for _, rule := range apiRules {
		if !strings.Contains(src, rule.Name) {
			continue
		}
		out.APIs = append(out.APIs, rule.Name)
		if rule.Capability != "" {
			out.Capabilities = append(out.Capabilities, rule.Capability)
		}
		if rule.Unsupported {
			out.Issues = append(out.Issues, Issue{
				Code: "unsupportedAPI", Arg: rule.Name,
			})
			if rule.Level > out.Level {
				out.Level = rule.Level
			}
			out.Supported = false
			continue
		}
		if rule.Level > out.Level {
			out.Level = rule.Level
		}
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
	r.Level, r.Supported = scan.Level, scan.Supported
	r.Imports, r.APIs = scan.Imports, scan.APIs
	r.Capabilities = scan.Capabilities
	r.Issues = append(r.Issues, scan.Issues...)
	if err := ValidateManifest(m, ""); err != nil {
		r.Issues = append(r.Issues, Issue{
			Code: "badManifest", Arg: err.Error(),
		})
		r.Level, r.Supported = LevelUnsupported, false
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
