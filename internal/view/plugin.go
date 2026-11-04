package view

// PluginCommand is one command an Obsidian-compatible plugin registered.
// The view only draws it and reports its ID; the app owns the plugin
// runtime and all file I/O.
type PluginCommand struct {
	ID      string
	Name    string
	Visible bool
}
