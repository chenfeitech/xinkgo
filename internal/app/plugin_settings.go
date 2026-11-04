package app

import (
	"strconv"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/plugin"
)

type pluginViewState struct {
	status    string
	installed []plugin.Report
	busy      map[string]bool
}

// openPlugins opens the plugin group of the settings window. The plugin
// browser lives there, not in a second window of its own.
func (s *appState) openPlugins() {
	s.openSettingsTab(tabPlugins)
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

// settingsView draws the plugin group inside the settings window's scroll
// area. It does no I/O; prepareSettings starts the background work.
func (h *pluginHost) settingsView(c *ui.Context) {
	th := c.Theme()
	state := h.uiSnapshot()
	installedLabel := locale.PluginTabInstalled()
	browseLabel := locale.PluginTabBrowse()
	selected := installedLabel
	if h.ui.tab == "browse" {
		selected = browseLabel
	}
	ui.Column(c).FillWidth().Gap(8).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
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
		if h.ui.tab == "browse" {
			h.browseView(c, th, state)
		} else {
			h.installedView(c, th, state)
		}
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
			if one.Degraded {
				ui.Text(c, locale.PluginDegraded()).
					FontSize(th.FontSize - 2).TextColor(th.TextMuted)
			}
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
	items := h.browserItems(200)
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
			if item.Degraded {
				ui.Text(c, locale.PluginDegraded()).
					FontSize(th.FontSize - 2).TextColor(th.TextMuted)
			}
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
		ui.Text(c, locale.PluginDetails()).Bold().
			FontSize(th.FontSize - 2)
		h.detailView(c, th, item)
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

func (h *pluginHost) detailView(c *ui.Context, th *ui.Theme,
	item plugin.BrowserItem) {
	if item.Downloads > 0 {
		pluginDetailText(c, th, locale.PluginDetailDownloads(),
			formatPluginCount(item.Downloads))
	}
	if item.Updated > 0 {
		pluginDetailText(c, th, locale.PluginDetailUpdated(),
			time.UnixMilli(item.Updated).Format("2006-01-02"))
	}
	if item.Version != "" {
		pluginDetailText(c, th, locale.PluginDetailVersion(),
			item.Version)
	}
	if item.VersionDownloads > 0 {
		pluginDetailText(c, th, locale.PluginDetailVersionDownloads(),
			formatPluginCount(item.VersionDownloads))
	}
	if item.MinAppVersion != "" && item.MinAppVersion != "0.0.0" {
		pluginDetailText(c, th, locale.PluginDetailMinAppVersion(),
			item.MinAppVersion)
	}
	if item.DetailLoaded {
		yes := locale.PluginDetailNo()
		if item.IsDesktopOnly {
			yes = locale.PluginDetailYes()
		}
		pluginDetailText(c, th, locale.PluginDetailDesktopOnly(), yes)
	}
	if item.Repo != "" {
		pluginDetailText(c, th, locale.PluginDetailRepository(),
			item.Repo)
	}
	if item.AuthorURL != "" {
		pluginDetailText(c, th, locale.PluginDetailHomepage(),
			item.AuthorURL)
	}
	if item.FundingURL != "" {
		pluginDetailText(c, th, locale.PluginDetailFunding(),
			item.FundingURL)
	}
	if item.Stars != "" {
		pluginDetailText(c, th, locale.PluginDetailStars(), item.Stars)
	}
	if item.License != "" {
		pluginDetailText(c, th, locale.PluginDetailLicense(),
			item.License)
	}
	if item.LastCommit != "" {
		pluginDetailText(c, th, locale.PluginDetailLastCommit(),
			item.LastCommit)
	}
	if item.VerifiedAt != "" {
		ui.Text(c, locale.PluginCheckedBy(item.VerifiedAt)).
			FontSize(th.FontSize - 2).TextColor(th.TextMuted)
	}
	if item.Compatible && !item.DetailLoaded {
		ui.Text(c, locale.PluginDetailLoading()).
			FontSize(th.FontSize - 2).TextColor(th.TextMuted)
	}
}

func pluginDetailText(c *ui.Context, th *ui.Theme, label, value string) {
	ui.Text(c, locale.PluginDetailLine(label, value)).
		FontSize(th.FontSize - 2).TextColor(th.TextMuted)
}

func formatPluginCount(n int64) string {
	text := strconv.FormatInt(n, 10)
	if len(text) <= 3 {
		return text
	}
	out := make([]byte, 0, len(text)+len(text)/3)
	for i := range text {
		if i > 0 && (len(text)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, text[i])
	}
	return string(out)
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
