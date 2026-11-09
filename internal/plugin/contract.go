package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// A Report answers "which Obsidian APIs does this package use". A contract
// answers the question that actually matters for Xinkgo: "does the host
// really implement every one of them". The two are deliberately separate:
// a C2, non-degraded report can still fail the contract when it depends on
// a capability whose implementation is an empty object.
//
// Only plugins in the official library are held to this contract. External
// Obsidian-compatible plugins keep using Report for their degraded listing.

// SupportedCapabilities is the exact set of capabilities Xinkgo fully
// implements. Order matches AllCapabilities.
var SupportedCapabilities = []Capability{
	CapabilityData, CapabilityCommands, CapabilityNotice,
	CapabilityVaultRead, CapabilityVaultWrite,
	CapabilityEditorRead, CapabilityEditorWrite,
	CapabilityNetwork, CapabilityEvents, CapabilityUI, CapabilityDOM,
}

// UnsupportedCapabilities can be scanned but must never appear in an
// official plugin: their host implementation is a placeholder.
var UnsupportedCapabilities = []Capability{
	CapabilityMetadata, CapabilityEditorExt, CapabilityCompatStub,
}

// ContractStubAPIs are Obsidian APIs whose shim implementation is a no-op,
// an empty object or an inert proxy. The demo tests and the library gate
// share this one list so the two cannot drift apart.
var ContractStubAPIs = []string{
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

// MaxContractLevel is the highest compatibility level a fully working
// plugin may reach. LevelDOM is allowed because registerView really is
// implemented; LevelEditor means a placeholder capability was used.
const MaxContractLevel = LevelDOM

// Violation is one reason a package cannot be certified as fully working.
type Violation struct {
	Code string `json:"code"`
	Arg  string `json:"arg,omitempty"`
}

// ContractResult is the verdict for one package.
type ContractResult struct {
	OK           bool         `json:"ok"`
	Level        Level        `json:"level"`
	Capabilities []Capability `json:"capabilities"`
	Violations   []Violation  `json:"violations,omitempty"`
}

// HasViolation reports whether the verdict contains a given reason.
func (r ContractResult) HasViolation(code string) bool {
	for _, one := range r.Violations {
		if one.Code == code {
			return true
		}
	}
	return false
}

// SupportedCapability reports whether a capability has a real Xinkgo
// implementation behind it.
func SupportedCapability(c Capability) bool {
	for _, one := range SupportedCapabilities {
		if one == c {
			return true
		}
	}
	return false
}

// ValidateContract certifies one plugin package. mainJS is needed because
// Report only carries the API names that matched a scan rule; a text check
// also catches stubs that have no rule of their own.
func ValidateContract(r Report, mainJS []byte) ContractResult {
	out := ContractResult{
		OK:           true,
		Level:        r.Level,
		Capabilities: append([]Capability(nil), r.Capabilities...),
	}
	add := func(code, arg string) {
		out.OK = false
		out.Violations = append(out.Violations,
			Violation{Code: code, Arg: arg})
	}
	if !r.Supported {
		add("notSupported", "")
	}
	if r.Level > MaxContractLevel {
		add("levelTooHigh", r.Level.Code())
	}
	for _, one := range r.Issues {
		switch one.Code {
		case "esmSyntax", "missingDefaultExport", "badManifest",
			"apiProfile", "dynamicRequire", "relativeImport",
			"unknownImport", "stubImport":
			add("scanIssue:"+one.Code, one.Arg)
		}
	}
	for _, name := range r.Imports {
		if !HostModules[name] {
			add("externalImport", name)
		}
	}
	for _, one := range r.Capabilities {
		if !SupportedCapability(one) {
			add("unsupportedCapability", string(one))
		}
	}
	src := string(mainJS)
	for _, api := range ContractStubAPIs {
		if strings.Contains(src, api) {
			add("stubAPI", api)
		}
	}
	return out
}

// LibrarySchema is the format version of library.json.
const LibrarySchema = 1

// Library is the official, curated list of plugins Xinkgo guarantees to
// run fully. It ships with the app's resources.
type Library struct {
	Schema  int             `json:"schema"`
	Profile string          `json:"profile"`
	Plugins []LibraryPlugin `json:"plugins"`
}

// LibraryPlugin is one certified plugin. Path is a repository-relative
// location used by the offline gate; a packaged app only needs the metadata.
type LibraryPlugin struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Version      string              `json:"version"`
	Kind         string              `json:"kind"`
	Path         string              `json:"path"`
	Capabilities []string            `json:"capabilities"`
	Verification LibraryVerification `json:"verification"`
}

// LibraryVerification records the three gates a plugin passed.
type LibraryVerification struct {
	Static    string `json:"static"`
	Runtime   string `json:"runtime"`
	CheckedAt string `json:"checkedAt"`
}

// ParseLibrary reads and validates the official plugin list.
func ParseLibrary(data []byte) (Library, error) {
	var l Library
	if len(data) == 0 {
		return l, errors.New("empty library")
	}
	if err := json.Unmarshal(data, &l); err != nil {
		return l, fmt.Errorf("parse library: %w", err)
	}
	if problems := l.Validate(); len(problems) > 0 {
		return l, fmt.Errorf("invalid library: %s",
			strings.Join(problems, "; "))
	}
	return l, nil
}

// Validate checks the list's own shape: an entry may look fine here and
// still fail ValidateContract against its real package, which the gate
// does separately.
func (l Library) Validate() []string {
	var problems []string
	if l.Schema != LibrarySchema {
		problems = append(problems,
			fmt.Sprintf("schema %d, want %d", l.Schema, LibrarySchema))
	}
	if l.Profile != APIProfile {
		problems = append(problems,
			fmt.Sprintf("profile %q, want %q", l.Profile, APIProfile))
	}
	if len(l.Plugins) == 0 {
		problems = append(problems, "library has no plugins")
	}
	seen := map[string]bool{}
	for _, p := range l.Plugins {
		label := "plugin " + p.ID
		if !idPattern.MatchString(p.ID) {
			problems = append(problems, label+" has an invalid id")
		}
		if seen[p.ID] {
			problems = append(problems, label+" is duplicated")
		}
		seen[p.ID] = true
		if strings.TrimSpace(p.Name) == "" ||
			strings.TrimSpace(p.Path) == "" {
			problems = append(problems, label+" needs a name and a path")
		}
		if !versionPattern.MatchString(p.Version) {
			problems = append(problems,
				fmt.Sprintf("%s has an invalid version %q", label, p.Version))
		}
		if p.Kind != "reference" && p.Kind != "plugin" {
			problems = append(problems,
				fmt.Sprintf("%s has an unknown kind %q", label, p.Kind))
		}
		if len(p.Capabilities) == 0 {
			problems = append(problems, label+" declares no capabilities")
		}
		for _, one := range p.Capabilities {
			if !SupportedCapability(Capability(one)) {
				problems = append(problems,
					fmt.Sprintf("%s declares unsupported capability %q",
						label, one))
			}
		}
		if p.Verification.Static != "pass" ||
			p.Verification.Runtime != "pass" ||
			strings.TrimSpace(p.Verification.CheckedAt) == "" {
			problems = append(problems,
				label+" is missing a complete verification record")
		}
	}
	return problems
}
