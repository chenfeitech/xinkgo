package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/plugin"
	"github.com/helightxu/xinkgo/internal/theme"
)

type pluginViewState struct {
	status    string
	installed []plugin.Report
	busy      map[string]bool
}

// openPlugins shows the plugin manager, or brings it back when it is
// already open. The catalog is fetched once, in the background.
func (s *appState) openPlugins() {
	if s.plugins == nil {
		return
	}
	s.plugins.openWindow()
}

func (h *pluginHost) openWindow() {
	if h.win != nil && !h.win.IsDestroyed() {
		h.win.Show()
		h.win.Focus()
		return
	}
	h.refresh()
	h.win = mygo.NewWindow(mygo.WindowOptions{
		Title:     Name + " " + locale.TitlePlugins(),
		Width:     960,
		Height:    680,
		MinWidth:  680,
		MinHeight: 420,
		Content:   ui.View(h.view),
	})
	loaded, _ := h.catalogState()
	if !loaded {
		h.refreshCatalog()
	}
}

func (h *pluginHost) uiSnapshot() pluginViewState {
	h.mu.RLock()
	defer h.mu.RUnlock()
	state := pluginViewState{
		status: h.ui.status,
		busy:   map[string]bool{},
	}
	state.installed = append([]plugin.Report(nil), h.ui.installed...)
	for id, on := range h.ui.busy {
		state.busy[id] = on
	}
	return state
}

func (h *pluginHost) view(c *ui.Context) {
	theme.Apply(c, h.st.config.Basic.Theme)
	th := c.Theme()
	state := h.uiSnapshot()
	installedLabel := locale.PluginTabInstalled()
	browseLabel := locale.PluginTabBrowse()
	selected := installedLabel
	if h.ui.tab == "browse" {
		selected = browseLabel
	}
	ui.Column(c).Fill().Gap(8).Padding(8).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, locale.TitlePlugins()).Bold()
			ui.Select(c.Key("plugin-tab"), &selected,
				[]string{installedLabel, browseLabel})
			if ui.Button(c.Key("plugin-refresh"),
				locale.PluginRefresh()).Clicked() {
				h.refresh()
				h.refreshCatalog()
			}
		})
		if selected == browseLabel {
			h.ui.tab = "browse"
		} else {
			h.ui.tab = "installed"
		}
		ui.Box(c).FillWidth().Height(1).Background(th.Border)
		ui.Text(c, locale.PluginThirdParty()).FontSize(th.FontSize - 2).
			TextColor(th.TextMuted)
		ui.Scroll(c).TrackScroll(&h.ui.scroll).Fill().Children(func() {
			ui.Column(c).FillWidth().Gap(6).Children(func() {
				if h.ui.tab == "browse" {
					h.browseView(c, th, state)
				} else {
					h.installedView(c, th, state)
				}
			})
		})
		if state.status != "" {
			ui.Text(c, state.status).FontSize(th.FontSize - 2).
				TextColor(th.TextMuted)
		}
	})
}

func (h *pluginHost) installedView(c *ui.Context, th *ui.Theme,
	state pluginViewState) {
	if len(state.installed) == 0 {
		ui.Text(c, locale.PluginNoInstalled()).TextColor(th.TextMuted)
		return
	}
	for _, one := range state.installed {
		h.installedRow(c, th, one, state.busy)
		ui.Box(c).FillWidth().Height(1).Background(th.Border)
	}
}

func (h *pluginHost) installedRow(c *ui.Context, th *ui.Theme,
	one plugin.Report, busy map[string]bool) {
	ui.Column(c).FillWidth().Gap(2).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, one.Name).Bold()
			ui.Text(c, locale.PluginLevelLabel(one.Level.Code())).
				FontSize(th.FontSize - 2).TextColor(th.TextMuted)
			if h.running(one.ID) {
				ui.Text(c, locale.PluginEnabled()).
					FontSize(th.FontSize - 2).TextColor(th.TextMuted)
			}
			if busy[one.ID] {
				ui.Text(c, locale.PluginCatalogLoading()).
					FontSize(th.FontSize - 2).TextColor(th.TextMuted)
			}
		})
		ui.Text(c, locale.PluginVersionLine(one.Version, one.Author,
			h.sourceLabel(one.Source))).FontSize(th.FontSize - 2).
			TextColor(th.TextMuted)
		if one.Description != "" {
			ui.Text(c, one.Description).FontSize(th.FontSize - 2)
		}
		for _, issue := range one.Issues {
			ui.Text(c, locale.PluginIssue(issue.Code, issue.Arg)).
				FontSize(th.FontSize - 2).TextColor(th.TextMuted)
		}
		if !one.Supported {
			ui.Text(c, locale.PluginInstallRefused()).
				FontSize(th.FontSize - 2).TextColor(th.TextMuted)
			return
		}
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			if h.enabled(one.ID) {
				if ui.Button(c.Key("plugin-disable-"+one.ID),
					locale.PluginDisable()).Clicked() {
					h.setEnabled(one.ID, false)
				}
			} else if ui.Button(c.Key("plugin-enable-"+one.ID),
				locale.PluginEnable()).Clicked() {
				h.setEnabled(one.ID, true)
			}
			if ui.Button(c.Key("plugin-reload-"+one.ID),
				locale.PluginReload()).Clicked() {
				h.reload(one.ID)
			}
			if ui.Button(c.Key("plugin-folder-"+one.ID),
				locale.PluginOpenFolder()).Clicked() {
				_ = mygo.Shell.OpenPath(one.Dir)
			}
		})
		ui.Text(c, locale.PluginGrants()).Bold().
			FontSize(th.FontSize - 2)
		for _, capability := range plugin.AllCapabilities {
			if !one.HasCapability(capability) {
				continue
			}
			checked := h.grantOn(one.ID, string(capability))
			if ui.Checkbox(c.Key("plugin-grant-"+one.ID+"-"+
				string(capability)), &checked,
				locale.PluginCapabilityLabel(string(capability))).
				Changed() {
				h.setGrant(one.ID, string(capability), checked)
			}
		}
	})
}

func (h *pluginHost) browseView(c *ui.Context, th *ui.Theme,
	state pluginViewState) {
	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		ui.TextInput(c.Key("plugin-search"), &h.ui.query).
			Placeholder(locale.PluginSearchPlaceholder()).Grow(1)
		ui.Checkbox(c.Key("plugin-supported"), &h.ui.supportedOnly,
			locale.PluginSupportedOnly())
		if ui.Button(c.Key("plugin-catalog-refresh"),
			locale.PluginRefresh()).Clicked() {
			h.refreshCatalog()
		}
	})
	loaded, loading := h.catalogState()
	if loading {
		ui.Text(c, locale.PluginCatalogLoading()).
			TextColor(th.TextMuted)
	}
	if !loaded && !loading {
		ui.Text(c, locale.PluginCatalogFailed()).
			TextColor(th.TextMuted)
	}
	items := h.browserItems(80)
	if len(items) == 0 {
		if h.ui.supportedOnly {
			ui.Text(c, locale.PluginNoCompatible()).
				TextColor(th.TextMuted)
		}
		return
	}
	for _, item := range items {
		h.browserRow(c, th, item, state.busy)
		ui.Box(c).FillWidth().Height(1).Background(th.Border)
	}
}

func (h *pluginHost) browserRow(c *ui.Context, th *ui.Theme,
	item plugin.BrowserItem, busy map[string]bool) {
	ui.Column(c).FillWidth().Gap(2).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, item.Name).Bold()
			ui.Text(c, locale.PluginLevelLabel(item.Level)).
				FontSize(th.FontSize - 2).TextColor(th.TextMuted)
			if h.installed(item.ID) {
				ui.Text(c, locale.PluginInstalled()).
					FontSize(th.FontSize - 2).TextColor(th.TextMuted)
			}
			if busy[item.ID] {
				ui.Text(c, locale.PluginCatalogLoading()).
					FontSize(th.FontSize - 2).TextColor(th.TextMuted)
			}
		})
		ui.Text(c, item.Author+" · "+item.Repo).
			FontSize(th.FontSize - 2).TextColor(th.TextMuted)
		if item.Description != "" {
			ui.Text(c, item.Description).FontSize(th.FontSize - 2)
		}
		note := item.Note
		if !item.Compatible {
			note = locale.PluginNotChecked()
		}
		if note != "" {
			ui.Text(c, note).FontSize(th.FontSize - 2).
				TextColor(th.TextMuted)
		}
		if item.Compatible && !h.installed(item.ID) &&
			!busy[item.ID] {
			if ui.Button(c.Key("plugin-install-"+item.ID),
				locale.PluginInstall()).Clicked() {
				go h.install(item)
			}
		}
	})
}

func (h *pluginHost) installed(id string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.reports[id]
	return ok
}

func (h *pluginHost) sourceLabel(source string) string {
	switch source {
	case "vault":
		return locale.PluginSourceVault()
	case "compat":
		return locale.PluginSourceCompat()
	case "user":
		return locale.PluginSourceUser()
	default:
		return locale.PluginSourceUnknown()
	}
}
