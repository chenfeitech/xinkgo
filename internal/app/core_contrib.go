package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/plugin"
	"github.com/helightxu/xinkgo/internal/view"
)

// The settings groups the built-in core plugins' own pages use. A plugin
// that is off, or needs a WebView the machine does not have, gets no page.
const (
	tabDailyNotes = "daily-notes-options"
	tabTemplates  = "templates-options"
	tabWebView    = "web-viewer-options"
)

// coreCommand is one command a built-in core plugin contributes to the
// native command palette. The view only draws it and reports its ID; the
// app owns every side effect.
type coreCommand struct {
	ID    string
	Owner string
	Name  func() string
	Group func() string
	Run   func(*appState)
}

// coreCommands is the whole contribution table. IDs are namespaced so a
// third-party plugin's "id:command" can never collide with one of these.
var coreCommands = []coreCommand{
	{"daily-notes:open-today", "daily-notes", locale.CmdDailyOpenToday,
		locale.CoreDailyNotesName, (*appState).dailyOpenToday},
	{"daily-notes:open-previous", "daily-notes", locale.CmdDailyOpenPrevious,
		locale.CoreDailyNotesName, func(s *appState) { s.dailyOpenAdjacent(-1) }},
	{"daily-notes:open-next", "daily-notes", locale.CmdDailyOpenNext,
		locale.CoreDailyNotesName, func(s *appState) { s.dailyOpenAdjacent(1) }},
	{"templates:insert", "templates", locale.CmdTemplatesInsert,
		locale.CoreTemplatesName, (*appState).templatesInsert},
	{"templates:create-note", "templates", locale.CmdTemplatesCreateNote,
		locale.CoreTemplatesName, (*appState).templatesCreateNote},
	{"web-viewer:open", "web-viewer", locale.CmdWebOpen,
		locale.CoreWebViewName, (*appState).webOpen},
	{"web-viewer:save-to-vault", "web-viewer", locale.CmdWebSave,
		locale.CoreWebViewName, (*appState).webSaveToVault},
	{"web-viewer:toggle-reader", "web-viewer", locale.CmdWebReader,
		locale.CoreWebViewName, (*appState).webToggleReader},
	{"web-viewer:open-in-browser", "web-viewer", locale.CmdWebOpenExternal,
		locale.CoreWebViewName, (*appState).webOpenInBrowser},
}

// coreRibbonIcons names the picture each built-in plugin puts on the
// ribbon. Every registry entry with Ribbon set must have one here; a test
// checks that they stay in step, so a plugin cannot declare a ribbon and
// silently contribute no button.
var coreRibbonIcons = map[string]string{
	"daily-notes": "calendar-days",
	"templates":   "layout-template",
	"web-viewer":  "globe",
}

// coreRibbonActions builds one ribbon button from the registry: its first
// command is what a click runs, and its icon names the picture. Icon names
// are resolved by view.pluginIcon.
func coreRibbonActions(id string) []view.PluginAction {
	spec, ok := plugin.CoreByID(id)
	if !ok || !spec.Ribbon {
		return nil
	}
	icon := coreRibbonIcons[id]
	if icon == "" {
		return nil
	}
	for _, cmd := range coreCommands {
		if cmd.Owner != id {
			continue
		}
		return []view.PluginAction{{
			ID: cmd.ID, Name: cmd.Name(), Icon: icon,
		}}
	}
	return nil
}

// coreOptionsTab maps a built-in plugin to its settings page ID.
func coreOptionsTab(id string) string {
	switch id {
	case "daily-notes":
		return tabDailyNotes
	case "templates":
		return tabTemplates
	case "web-viewer":
		return tabWebView
	default:
		return ""
	}
}

// coreDisableHooks are the resources a built-in plugin must let go of when
// its switch is turned off. Only the Web viewer holds native objects.
var coreDisableHooks = map[string]func(*appState){
	"web-viewer": (*appState).webDispose,
}

// coreAvailable reports whether a built-in plugin can run on this machine.
func (s *appState) coreAvailable(id string) bool {
	spec, ok := plugin.CoreByID(id)
	if !ok || !spec.NeedsWebView {
		return true
	}
	return mygo.WebViewAvailable()
}

// coreEnabled is the one read of a built-in plugin's state: its switch and
// its availability. The settings pane, the palette and the ribbon all use
// it, so a plugin that cannot run never offers a dead control.
func (s *appState) coreEnabled(id string) bool {
	if s.plugins == nil || !s.plugins.coreOn(id) {
		return false
	}
	return s.coreAvailable(id)
}

// coreOptions is the built-in plugins' setup, read under the host lock.
func (s *appState) coreOptions() plugin.CoreOptions {
	if s.plugins == nil {
		return plugin.DefaultCoreOptions()
	}
	return s.plugins.options()
}

// setCoreOptions writes one change to plugins.json and republishes what the
// enabled plugins contribute. Settings fields commit on Submitted/Changed,
// not once a frame, so this is a small, infrequent write.
func (s *appState) setCoreOptions(o plugin.CoreOptions) {
	if s.plugins == nil {
		return
	}
	s.plugins.setOptions(o)
	s.webOptionsChanged()
}

// runCommand is the one dispatcher for a palette choice: built-in commands
// first, then the Obsidian-compatible plugin host. app.go wires the
// workspace's callback here instead of straight to the host.
func (s *appState) runCommand(id string) {
	for _, one := range coreCommands {
		if one.ID == id {
			if s.coreEnabled(one.Owner) {
				one.Run(s)
			}
			return
		}
	}
	if s.plugins != nil {
		s.plugins.runCommand(id)
	}
}

// runAction delivers a ribbon click, built-in or third-party.
func (s *appState) runAction(id string) {
	for _, one := range coreCommands {
		if one.ID == id {
			if s.coreEnabled(one.Owner) {
				one.Run(s)
			}
			return
		}
	}
	if s.plugins != nil {
		s.plugins.runAction(id)
	}
}

// coreOptionTabs are the settings pages the enabled plugins with a
// Settings declaration contribute. They are built per frame from the
// registry, so turning a plugin off removes its page at once.
func (s *appState) coreOptionTabs() []settingsTab {
	var out []settingsTab
	for _, one := range plugin.CorePlugins {
		if !one.Settings || !s.coreEnabled(one.ID) {
			continue
		}
		id := coreOptionsTab(one.ID)
		if id == "" {
			continue
		}
		out = append(out, s.coreOptionTab(id, one.ID))
	}
	return out
}

func (s *appState) coreOptionTab(tab, id string) settingsTab {
	draw := func(c *ui.Context) {
		switch id {
		case "daily-notes":
			s.dailyNotesSettings(c)
		case "templates":
			s.templatesSettings(c)
		case "web-viewer":
			s.webViewerSettings(c)
		}
	}
	var name, note string
	var icon *ui.SVG
	switch id {
	case "daily-notes":
		name, note, icon = locale.SettingsTabDailyNotes(),
			locale.SettingsTabDailyNotesNote(), view.IconSettingsCore
	case "templates":
		name, note, icon = locale.SettingsTabTemplates(),
			locale.SettingsTabTemplatesNote(), view.IconSettingsCore
	case "web-viewer":
		name, note, icon = locale.SettingsTabWebView(),
			locale.SettingsTabWebViewNote(), view.IconSettingsPlugins
	}
	return settingsTab{ID: tab, Name: name, Note: note, Icon: icon,
		Draw: draw}
}

// isCoreOptionsTab reports whether tab is one of the built-in plugins'
// option pages. They need their own cached lists, not the community
// catalog's network work.
func isCoreOptionsTab(tab string) bool {
	return tab == tabDailyNotes || tab == tabTemplates || tab == tabWebView
}
