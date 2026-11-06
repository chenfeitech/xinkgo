package plugin

// CorePlugin is one of Xinkgo's own built-in capabilities, offered in the
// same manager as the Obsidian-compatible plugins. Unlike those, a core
// plugin is native UI: it needs no install and no capability grants. The
// registry is pure data; the app decides what each ID contributes.
type CorePlugin struct {
	ID      string
	Default bool
	// Settings says the plugin has a page under the settings window's
	// Plugin options section, shown only while the plugin is on.
	Settings bool
	// Commands are the stable command IDs the plugin contributes, kept
	// here so the registry is the one place that names them.
	Commands []string
	// Ribbon says the plugin puts an icon in the left plugin ribbon.
	Ribbon bool
	// NeedsWebView says the plugin cannot run on a machine without a
	// system WebView. The app checks mygo.WebViewAvailable() for it.
	NeedsWebView bool
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
	{ID: "daily-notes", Default: false, Settings: true, Ribbon: true,
		Commands: []string{
			"daily-notes:open-today", "daily-notes:open-previous",
			"daily-notes:open-next",
		}},
	{ID: "templates", Default: false, Settings: true, Ribbon: true,
		Commands: []string{
			"templates:insert", "templates:create-note",
		}},
	{ID: "web-viewer", Default: false, Settings: true, Ribbon: true,
		NeedsWebView: true,
		Commands: []string{
			"web-viewer:open", "web-viewer:save-to-vault",
			"web-viewer:toggle-reader", "web-viewer:open-in-browser",
		}},
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
	_, ok := CoreByID(id)
	return ok
}

// CoreByID finds a built-in core plugin by its stable ID.
func CoreByID(id string) (CorePlugin, bool) {
	for _, one := range CorePlugins {
		if one.ID == id {
			return one, true
		}
	}
	return CorePlugin{}, false
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
