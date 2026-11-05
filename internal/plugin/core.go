package plugin

// CorePlugin is one of Xinkgo's own built-in capabilities, offered in the
// same manager as the Obsidian-compatible plugins. Unlike those, a core
// plugin is native UI: it needs no webview, no install, no capability
// grants, and toggling it changes what the window draws. The registry is
// pure data; the app decides what each ID gates in the workspace.
type CorePlugin struct {
	ID      string
	Default bool
}

// CorePlugins is the ordered list of Xinkgo's built-in core plugins. The
// IDs are stable: they are written to plugins.json, so a rename would turn
// a user's choice off on the next run.
var CorePlugins = []CorePlugin{
	{ID: "file-explorer", Default: true},
	{ID: "command-palette", Default: true},
	{ID: "global-search", Default: true},
	{ID: "page-preview", Default: true},
	{ID: "outline", Default: true},
	{ID: "backlinks", Default: true},
	{ID: "tag-pane", Default: true},
	{ID: "status-bar", Default: true},
	{ID: "word-count", Default: true},
	{ID: "plugin-ribbon", Default: true},
}

// CoreIDs returns the IDs of every core plugin, in registry order.
func CoreIDs() []string {
	out := make([]string, 0, len(CorePlugins))
	for _, one := range CorePlugins {
		out = append(out, one.ID)
	}
	return out
}

// IsCore reports whether id names a built-in core plugin.
func IsCore(id string) bool {
	for _, one := range CorePlugins {
		if one.ID == id {
			return true
		}
	}
	return false
}

// DefaultCore returns the core plugins that stand on a first run: every
// one whose Default is set. A choice the file leaves out takes its default,
// so an older plugins.json keeps working when a new core plugin is added.
func DefaultCore() map[string]bool {
	out := make(map[string]bool, len(CorePlugins))
	for _, one := range CorePlugins {
		if one.Default {
			out[one.ID] = true
		}
	}
	return out
}
