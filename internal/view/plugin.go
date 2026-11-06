package view

// The IDs of Xinkgo's built-in core plugins, as the workspace gates them.
// They are written to plugins.json, so they must stay in step with
// plugin.CorePlugins.
const (
	CoreFileExplorer   = "file-explorer"
	CoreCommandPalette = "command-palette"
	CoreGlobalSearch   = "global-search"
	CorePagePreview    = "page-preview"
	CoreOutline        = "outline"
	CoreBacklinks      = "backlinks"
	CoreTagPane        = "tag-pane"
	CoreStatusBar      = "status-bar"
	CoreWordCount      = "word-count"
	CorePluginRibbon   = "plugin-ribbon"
	CoreDailyNotes     = "daily-notes"
	CoreTemplates      = "templates"
	CoreWebView        = "web-viewer"
)

// coreOn reports whether a built-in core plugin stands. A feature the app
// did not name is on, so a workspace drawn on its own shows everything.
func (w *Workspace) coreOn(id string) bool {
	if w.CoreFeatures == nil {
		return true
	}
	on, ok := w.CoreFeatures[id]
	return !ok || on
}

// PluginView is one visible surface an Obsidian-compatible plugin
// registered through registerView. The view draws the list and reports the
// chosen ID; the app owns the page the surface shows. ViewType is the type
// the plugin registered, which the app sends back to build the surface.
type PluginView struct {
	ID       string
	Name     string
	ViewType string
}

// PluginCommand is one command an Obsidian-compatible plugin registered.
// The view only draws it and reports its ID; the app owns the plugin
// runtime and all file I/O.
type PluginCommand struct {
	ID      string
	Name    string
	Visible bool
	// Group is the localized name of the plugin that owns the command.
	// The palette shows it under the command instead of a raw ID.
	Group string
}

// PluginAction is one icon an Obsidian-compatible plugin put on the left
// ribbon. The view only draws it and reports its ID.
type PluginAction struct {
	ID   string
	Name string
	Icon string
}
