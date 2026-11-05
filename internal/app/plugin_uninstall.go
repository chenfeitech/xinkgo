package app

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/plugin"
	"github.com/helightxu/xinkgo/internal/state"
)

// askUninstall takes a plugin package away, but only after the user has
// seen its full path and said yes. The question is asked off the main
// thread so the window keeps drawing; the removal itself runs on it.
func (h *pluginHost) askUninstall(one plugin.Report) {
	go func() {
		answer, err := mygo.Dialog.Message(mygo.MessageOptions{
			Type:    mygo.MessageQuestion,
			Title:   locale.PluginUninstallTitle(),
			Message: locale.PluginUninstallMessage(one.Name),
			Detail:  one.Dir,
			Buttons: []string{locale.TakeAway(), locale.Cancel()},
			// Taking a plugin away should never be the answer to a slip
			// of the key: Enter and Escape both choose Cancel.
			DefaultButton:      1,
			CancelButton:       1,
			DestructiveButtons: []int{0},
		})
		if err != nil || answer.Button != 0 {
			return
		}
		mygo.RunOnMain(func() { h.uninstall(one) })
	}()
}

// uninstall stops a plugin, removes its package from disk and forgets
// every choice made about it, so a stale enabled flag cannot bring it
// back on the next run. It is called on the main thread.
func (h *pluginHost) uninstall(one plugin.Report) {
	if !h.pluginPackageDir(one.Dir) {
		slog.Warn("plugin package path refused", "id", one.ID, "dir", one.Dir)
		h.setStatus(locale.PluginStatus(locale.PluginUninstallFailed()))
		return
	}
	// Stop the instance first: a page still loading while its files go
	// would only produce confusing errors.
	h.disable(one.ID, false)
	if err := os.RemoveAll(one.Dir); err != nil {
		slog.Warn("plugin package could not be removed", "id", one.ID,
			"dir", one.Dir, "err", err)
		h.setStatus(locale.PluginStatus(locale.PluginUninstallFailed()))
		return
	}
	h.mu.Lock()
	h.saved = h.saved.Without(one.ID)
	delete(h.ui.open, one.ID)
	saved := h.saved
	h.mu.Unlock()
	if err := state.SavePlugins(h.dir, saved); err != nil {
		slog.Warn("plugin state not written", "err", err)
	}
	h.refresh()
	h.refreshCommands()
	h.setStatus(locale.PluginStatus(locale.PluginUninstalledOK(one.Name)))
}

// pluginPackageDir guards the removal: only a direct child of one of the
// three plugin folders is a package we are willing to delete. A blank,
// relative or unrelated path must never reach os.RemoveAll.
func (h *pluginHost) pluginPackageDir(dir string) bool {
	if dir == "" || !filepath.IsAbs(dir) {
		return false
	}
	clean := filepath.Clean(dir)
	parent := filepath.Dir(clean)
	h.mu.RLock()
	root := h.root
	h.mu.RUnlock()
	if parent == filepath.Clean(h.userDir) {
		return true
	}
	if root == "" {
		return false
	}
	compat := filepath.Join(root, ".obsidian", "plugins")
	vault := filepath.Join(root, ".xink", "plugins")
	return parent == filepath.Clean(compat) || parent == filepath.Clean(vault)
}
