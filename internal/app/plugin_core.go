package app

import (
	"log/slog"
	"sort"
	"strings"

	"github.com/egoist/mygo"
	"github.com/helightxu/xinkgo/internal/plugin"
	"github.com/helightxu/xinkgo/internal/state"
	"github.com/helightxu/xinkgo/internal/view"
)

// setWorkspace mutates the workspace on the main thread, as MyGo requires.
// With no workspace there is nothing to draw into: that is the case before
// the window opens and in unit tests, which must not wait on the MyGo main
// loop. Every change to what the window draws goes through here.
func (h *pluginHost) setWorkspace(fn func(ws *view.Workspace)) {
	if h.st == nil || h.st.ws == nil {
		return
	}
	// In a unit test there is no MyGo main loop to wait on. Apply the
	// change directly so the RPC branches can be exercised without one.
	if !mygo.App.IsReady() {
		fn(h.st.ws)
		return
	}
	mygo.RunOnMain(func() {
		if h.st.ws != nil {
			fn(h.st.ws)
		}
	})
}

// contributionSnapshot is a copy of what the compatible plugins registered,
// taken under the host lock so a frame never reads a map being written.
type contributionSnapshot struct {
	commands []view.PluginCommand
	ribbon   []view.PluginAction
	views    []view.PluginView
}

func snapshotInstances(instances map[string]*pluginInstance) contributionSnapshot {
	var snap contributionSnapshot
	for _, inst := range instances {
		if !inst.ready {
			continue
		}
		for _, one := range inst.commands {
			snap.commands = append(snap.commands, one)
		}
		for _, one := range inst.ribbon {
			snap.ribbon = append(snap.ribbon, one)
		}
		for _, one := range inst.views {
			snap.views = append(snap.views, one)
		}
	}
	return snap
}

// mergeContributions is the pure half of publishing: it turns a saved state
// into the four lists the window draws. Built-in and third-party commands
// are merged here, so either side turning on or off cannot wipe the other.
func mergeContributions(saved state.PluginState, webviewAvailable bool,
	snap contributionSnapshot) (map[string]bool, []view.PluginCommand,
	[]view.PluginAction, []view.PluginView) {
	features := make(map[string]bool, len(plugin.CorePlugins))
	commands := append([]view.PluginCommand{}, snap.commands...)
	ribbon := append([]view.PluginAction{}, snap.ribbon...)
	views := append([]view.PluginView{}, snap.views...)
	for _, one := range plugin.CorePlugins {
		on := saved.CoreOf(one.ID)
		if one.NeedsWebView && !webviewAvailable {
			on = false
		}
		features[one.ID] = on
		if !on {
			continue
		}
		for _, cmd := range coreCommands {
			if cmd.Owner != one.ID {
				continue
			}
			commands = append(commands, view.PluginCommand{
				ID: cmd.ID, Name: cmd.Name(), Group: cmd.Group(),
				Visible: true,
			})
		}
		ribbon = append(ribbon, coreRibbonActions(one.ID)...)
	}
	sort.Slice(commands, func(i, j int) bool {
		return strings.ToLower(commands[i].Name) <
			strings.ToLower(commands[j].Name)
	})
	sort.Slice(ribbon, func(i, j int) bool {
		return strings.ToLower(ribbon[i].Name) <
			strings.ToLower(ribbon[j].Name)
	})
	sort.Slice(views, func(i, j int) bool {
		return strings.ToLower(views[i].Name) <
			strings.ToLower(views[j].Name)
	})
	return features, commands, ribbon, views
}

// publishCoreContributions is the one place that turns a plugins.json
// choice into what the window draws.
func (h *pluginHost) publishCoreContributions() {
	h.mu.RLock()
	saved := h.saved
	snap := snapshotInstances(h.instances)
	h.mu.RUnlock()
	features, commands, ribbon, views := mergeContributions(saved,
		mygo.WebViewAvailable(), snap)
	h.setWorkspace(func(ws *view.Workspace) {
		ws.CoreFeatures = features
		ws.PluginCommands = commands
		ws.PluginRibbon = ribbon
		ws.PluginViews = views
		if ws.PluginViewActive != "" && !viewInList(views,
			ws.PluginViewActive) {
			ws.PluginViewActive = ""
			ws.PluginPage = nil
		}
	})
}

// setCore changes one built-in core plugin's choice, keeps it in
// plugins.json alongside the Obsidian-compatible plugins, and draws the
// change at once. No plugin code runs and no webview is needed.
func (h *pluginHost) setCore(id string, on bool) {
	if !plugin.IsCore(id) {
		return
	}
	h.mu.Lock()
	h.saved = h.saved.WithCore(id, on)
	saved := h.saved
	h.mu.Unlock()
	if err := state.SavePlugins(h.dir, saved); err != nil {
		slog.Warn("core plugin choice not written", "err", err)
	}
	h.publishCoreContributions()
	// A plugin that holds native resources must let go of them when its
	// switch is turned off. Plugins without such resources have no hook.
	if !on && h.st != nil {
		if hook := coreDisableHooks[id]; hook != nil {
			hook(h.st)
		}
	}
}

// coreOn is the settings pane's read of one built-in core plugin: it
// takes a lock so a frame never reads a choice being written.
func (h *pluginHost) coreOn(id string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.saved.CoreOf(id)
}

// options gives back the built-in core plugins' choices under the lock.
func (h *pluginHost) options() plugin.CoreOptions {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.saved.OptionsOf()
}

// setOptions writes one built-in plugin's setup change to plugins.json and
// republishes what the enabled plugins contribute.
func (h *pluginHost) setOptions(o plugin.CoreOptions) {
	h.mu.Lock()
	h.saved = h.saved.WithOptions(o)
	saved := h.saved
	h.mu.Unlock()
	if err := state.SavePlugins(h.dir, saved); err != nil {
		slog.Warn("core plugin options not written", "err", err)
	}
	h.publishCoreContributions()
}

// openPluginView shows one plugin's registered surface in the window.
// Picking the surface already on show closes it. Only one surface is on
// show at a time, as the compatibility report promises.
func (h *pluginHost) openPluginView(id string) {
	if h.st == nil || h.st.ws == nil {
		return
	}
	h.mu.RLock()
	var (
		owner    *pluginInstance
		viewType string
		page     *mygo.WebView
	)
	for _, inst := range h.instances {
		if one, ok := inst.views[id]; ok {
			owner, viewType, page = inst, one.ViewType, inst.page
			break
		}
	}
	h.mu.RUnlock()
	if owner == nil || page == nil {
		return
	}

	active := h.st.ws.PluginViewActive // read on the main thread
	if active == id {
		// Close: tell the old page to clear its surface, then draw
		// none. The page stays alive; only what it shows changes.
		h.emitToPage(page, pluginEvent{Type: "viewClose"})
		h.setWorkspace(func(ws *view.Workspace) {
			ws.PluginViewActive = ""
			ws.PluginPage = nil
		})
		return
	}
	// Switching from another plugin's surface: clear that one first so
	// its DOM is not left built when its page stops being shown.
	if active != "" {
		h.closePluginViewPage(active)
	}
	h.emitToPage(page, pluginEvent{Type: "view", ViewType: viewType})
	h.setWorkspace(func(ws *view.Workspace) {
		ws.PluginViewActive = id
		ws.PluginPage = page
	})
}

// closePluginViewPage tells the page that owns id to clear its surface.
func (h *pluginHost) closePluginViewPage(id string) {
	h.mu.RLock()
	for _, inst := range h.instances {
		if _, ok := inst.views[id]; ok && inst.page != nil {
			page := inst.page
			h.mu.RUnlock()
			h.emitToPage(page, pluginEvent{Type: "viewClose"})
			return
		}
	}
	h.mu.RUnlock()
}

// emitToPage sends a control event to one plugin page. The identity of
// the page, not anything in the message, is what the host trusts.
func (h *pluginHost) emitToPage(page *mygo.WebView, event pluginEvent) {
	if err := pluginEvents.EmitPage(page.Page(), event); err != nil {
		slog.Warn("plugin view event", "err", err)
	}
}
