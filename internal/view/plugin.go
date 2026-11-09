package view

import "sort"

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

// PluginStatus is one item an Obsidian-compatible plugin put on the status
// bar. Items are keyed by plugin ID so one plugin's text never overwrites
// another's, nor the window's own message.
type PluginStatus struct {
	ID   string
	Text string
}

// AddPluginStatus reserves an empty item for a plugin. It never clears a
// text that already arrived: fire("status.add") and fire("status.set") are
// asynchronous and may reach the host in either order.
func (w *Workspace) AddPluginStatus(id string) {
	for _, one := range w.PluginStatuses {
		if one.ID == id {
			return
		}
	}
	w.PluginStatuses = append(w.PluginStatuses, PluginStatus{ID: id})
	w.sortPluginStatuses()
}

// SetPluginStatus gives one plugin's item its text, adding it if needed.
func (w *Workspace) SetPluginStatus(id, text string) {
	for i := range w.PluginStatuses {
		if w.PluginStatuses[i].ID == id {
			w.PluginStatuses[i].Text = text
			return
		}
	}
	w.PluginStatuses = append(w.PluginStatuses,
		PluginStatus{ID: id, Text: text})
	w.sortPluginStatuses()
}

// RemovePluginStatus takes a plugin's item away when it is unloaded.
func (w *Workspace) RemovePluginStatus(id string) {
	kept := w.PluginStatuses[:0]
	for _, one := range w.PluginStatuses {
		if one.ID != id {
			kept = append(kept, one)
		}
	}
	w.PluginStatuses = kept
}

// sortPluginStatuses keeps the row in a stable order between frames.
func (w *Workspace) sortPluginStatuses() {
	sort.Slice(w.PluginStatuses, func(i, j int) bool {
		return w.PluginStatuses[i].ID < w.PluginStatuses[j].ID
	})
}
