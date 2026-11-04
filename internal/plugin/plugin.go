// Package plugin is the pure data of an Obsidian-compatible plugin: its
// manifest, the community catalog, and a conservative scan of the API a
// plugin bundle asks for. It does no I/O and knows nothing of windows.
package plugin

import "strings"

// APIProfile is the Obsidian API profile this build of Xinkgo emulates.
// It is not Xinkgo's own version, and is deliberately lower than the
// version of a recent Obsidian desktop app.
const APIProfile = "1.0.0"

// Level is how much of Obsidian a plugin needs from the host.
type Level int

// The compatibility levels. C0 is only a valid package; the levels a running
// plugin can use are C1 through C4.
const (
	LevelUnsupported Level = iota // C0: package cannot be loaded
	LevelCore                     // C1: headless core API
	LevelNative                   // C2: native UI and Vault/Editor adapters
	LevelDOM                      // C3: plugin-owned DOM surface
	LevelEditor                   // C4: CodeMirror 6 or Electron internals
)

// Code is the short name of a level, for logs and catalog data.
func (l Level) Code() string {
	switch l {
	case LevelCore:
		return "C1"
	case LevelNative:
		return "C2"
	case LevelDOM:
		return "C3"
	case LevelEditor:
		return "C4"
	default:
		return "C0"
	}
}

// Supported reports whether the spike host can run a plugin of this level.
// C3 needs a plugin DOM surface and C4 needs CodeMirror 6 or Electron;
// neither is part of the P0 host.
func (l Level) Supported() bool { return l == LevelCore || l == LevelNative }

// Capability is one permission a plugin may need at runtime.
type Capability string

// The capabilities the host knows. A plugin's scan only names what it
// needs; the user still grants each one in the plugin manager.
const (
	CapabilityData        Capability = "data"
	CapabilityCommands    Capability = "commands"
	CapabilityNotice      Capability = "notice"
	CapabilityVaultRead   Capability = "vault.read"
	CapabilityVaultWrite  Capability = "vault.write"
	CapabilityEditorRead  Capability = "editor.read"
	CapabilityEditorWrite Capability = "editor.write"
	CapabilityMetadata    Capability = "metadata"
	CapabilityNetwork     Capability = "network"
	CapabilityEvents      Capability = "events.app"
	CapabilityUI          Capability = "ui"
)

// AllCapabilities is every capability the manager may show, in order.
var AllCapabilities = []Capability{
	CapabilityData, CapabilityCommands, CapabilityNotice,
	CapabilityVaultRead, CapabilityVaultWrite,
	CapabilityEditorRead, CapabilityEditorWrite,
	CapabilityMetadata, CapabilityNetwork, CapabilityEvents,
	CapabilityUI,
}

// Issue is one reason a plugin is held back. Code is a stable identifier
// the UI maps to a catalog message; Arg is the import or API that caused it.
type Issue struct {
	Code string `json:"code"`
	Arg  string `json:"arg,omitempty"`
}

// Report is what the host knows about one plugin package without running it.
type Report struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	Description  string       `json:"description"`
	Author       string       `json:"author"`
	Dir          string       `json:"dir"`
	Source       string       `json:"source"`
	Level        Level        `json:"level"`
	Supported    bool         `json:"supported"`
	Capabilities []Capability `json:"capabilities,omitempty"`
	Imports      []string     `json:"imports,omitempty"`
	APIs         []string     `json:"apis,omitempty"`
	Issues       []Issue      `json:"issues,omitempty"`
}

// HasCapability reports whether a plugin asks for a capability.
func (r Report) HasCapability(c Capability) bool {
	for _, one := range r.Capabilities {
		if one == c {
			return true
		}
	}
	return false
}

// unique keeps a sorted list of strings with no duplicate values.
func unique(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, one := range in {
		if one == "" || seen[one] {
			continue
		}
		seen[one] = true
		out = append(out, one)
	}
	sortStrings(out)
	return out
}

// sortStrings is separated so tests can replace it with a stable check.
var sortStrings = func(in []string) {
	for i := 1; i < len(in); i++ {
		for j := i; j > 0 && strings.ToLower(in[j]) < strings.ToLower(in[j-1]); j-- {
			in[j], in[j-1] = in[j-1], in[j]
		}
	}
}
