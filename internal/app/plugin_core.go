package app

import (
	"log/slog"

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
	mygo.RunOnMain(func() {
		if h.st.ws != nil {
			fn(h.st.ws)
		}
	})
}

// applyCoreFeatures publishes the built-in core plugin choices to the
// workspace. It is the one place that turns a plugins.json choice into
// what the window draws; a nil map means every feature stands, which is
// what a workspace built without a host (as in tests) expects.
func (h *pluginHost) applyCoreFeatures() {
	h.mu.RLock()
	saved := h.saved
	h.mu.RUnlock()
	features := make(map[string]bool, len(plugin.CorePlugins))
	for _, one := range plugin.CorePlugins {
		features[one.ID] = saved.CoreOf(one.ID)
	}
	h.setWorkspace(func(ws *view.Workspace) {
		ws.CoreFeatures = features
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
	h.applyCoreFeatures()
}

// coreOn is the settings pane's read of one built-in core plugin: it
// takes a lock so a frame never reads a choice being written.
func (h *pluginHost) coreOn(id string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.saved.CoreOf(id)
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
