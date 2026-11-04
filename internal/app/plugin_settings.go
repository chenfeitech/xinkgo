package app

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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
	coreIDs   map[string]bool
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
		status:  h.ui.status,
		busy:    map[string]bool{},
		coreIDs: map[string]bool{},
	}
	for _, one := range h.compatible {
		if one.Core {
			state.coreIDs[one.ID] = true
		}
	}
	for _, one := range h.ui.installed {
		if !state.coreIDs[one.ID] {
			state.installed = append(state.installed, one)
		}
	}
	sort.SliceStable(state.installed, func(i, j int) bool {
		left := h.stats[state.installed[i].ID].Downloads
		right := h.stats[state.installed[j].ID].Downloads
		if left != right {
			return left > right
		}
		return strings.ToLower(state.installed[i].Name) <
			strings.ToLower(state.installed[j].Name)
	})
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
	coreLabel := locale.PluginTabCore()
	installedLabel := locale.PluginTabInstalled()
	browseLabel := locale.PluginTabBrowse()
	selected := coreLabel
	if h.ui.tab == "installed" {
		selected = installedLabel
	} else if h.ui.tab == "browse" {
		selected = browseLabel
	}
	ui.Column(c).FillWidth().Gap(8).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Select(c.Key("plugin-tab"), &selected,
				[]string{coreLabel, installedLabel, browseLabel})
			if ui.Button(c.Key("plugin-refresh"),
				locale.PluginRefresh()).Clicked() {
				h.refresh()
				h.refreshCatalog()
			}
		})
		switch selected {
		case coreLabel:
			h.ui.tab = "core"
		case browseLabel:
			h.ui.tab = "browse"
		default:
			h.ui.tab = "installed"
		}
		ui.Box(c).FillWidth().Height(1).Background(th.Border)
		if h.ui.tab == "core" {
			ui.Text(c, locale.PluginCoreNote()).
				FontSize(th.FontSize - 2).TextColor(th.TextMuted)
			h.coreView(c, th, state)
		} else {
			ui.Text(c, locale.PluginTabOther()).Bold().
				FontSize(th.FontSize - 2)
			ui.Text(c, locale.PluginThirdParty()).
				FontSize(th.FontSize - 2).TextColor(th.TextMuted)
		}
		if h.ui.tab == "browse" {
			h.browseView(c, th, state)
		} else if h.ui.tab == "installed" {
			h.installedView(c, th, state)
		}
		if state.status != "" {
			ui.Text(c, state.status).FontSize(th.FontSize - 2).
				TextColor(th.TextMuted)
		}
	})
}

func (h *pluginHost) coreView(c *ui.Context, th *ui.Theme,
	state pluginViewState) {
	items := h.coreItems()
	if len(items) == 0 {
		ui.Text(c, locale.PluginCoreEmpty()).TextColor(th.TextMuted)
		return
	}
	for _, item := range items {
		h.coreRow(c, th, item, state.busy)
		ui.Box(c).FillWidth().Height(1).Background(th.Border)
	}
}

func (h *pluginHost) coreRow(c *ui.Context, th *ui.Theme,
	item plugin.BrowserItem, busy map[string]bool) {
	ui.Column(c).FillWidth().Gap(2).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, item.Name).Bold()
			ui.Text(c, locale.PluginCore()).
				FontSize(th.FontSize - 2).TextColor(th.TextMuted)
			ui.Text(c, locale.PluginLevelLabel(item.Level)).
				FontSize(th.FontSize - 2).TextColor(th.TextMuted)
			if item.Degraded {
				ui.Text(c, locale.PluginDegraded()).
					FontSize(th.FontSize - 2).TextColor(th.TextMuted)
			}
			if item.Downloads > 0 {
				ui.Text(c, formatPluginCount(item.Downloads)).
					FontSize(th.FontSize - 2).TextColor(th.TextMuted)
			}
			if busy[item.ID] {
				ui.Text(c, locale.PluginCatalogLoading()).
					FontSize(th.FontSize - 2).TextColor(th.TextMuted)
			}
		})
		if item.Description != "" {
			ui.Text(c, item.Description).FontSize(th.FontSize - 2)
		}
		h.detailView(c, th, item)
		if !h.installed(item.ID) {
			ui.Text(c, locale.PluginCoreInstallFirst()).
				FontSize(th.FontSize - 2).TextColor(th.TextMuted)
			if !busy[item.ID] {
				if ui.Button(c.Key("plugin-core-install-"+item.ID),
					locale.PluginInstall()).Clicked() {
					go h.install(item)
				}
			}
			return
		}
		checked := h.enabled(item.ID)
		if ui.Checkbox(c.Key("plugin-core-toggle-"+item.ID),
			&checked, locale.PluginCoreToggle()).Changed() {
			h.setEnabled(item.ID, checked)
		}
		if h.running(item.ID) {
			ui.Text(c, locale.PluginEnabled()).
				FontSize(th.FontSize - 2).TextColor(th.TextMuted)
		} else {
			ui.Text(c, locale.PluginDisabled()).
				FontSize(th.FontSize - 2).TextColor(th.TextMuted)
		}
		ui.Row(c).Gap(6).Children(func() {
			if ui.Button(c.Key("plugin-core-reload-"+item.ID),
				locale.PluginReload()).Clicked() {
				h.reload(item.ID)
			}
			if ui.Button(c.Key("plugin-core-folder-"+item.ID),
				locale.PluginOpenFolder()).Clicked() {
				_ = mygo.Shell.OpenPath(filepath.Join(h.userDir, item.ID))
			}
		})
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
		field := ui.TextInput(c.Key("plugin-search"), &h.ui.query).
			Placeholder(locale.PluginSearchPlaceholder()).Grow(1)
		if field.Changed() {
			h.ui.browseLimit = 40
			h.ui.lastQuery = h.ui.query
		}
		if ui.Checkbox(c.Key("plugin-supported"), &h.ui.supportedOnly,
			locale.PluginSupportedOnly()).Changed() {
			h.ui.browseLimit = 40
			h.ui.lastSupported = h.ui.supportedOnly
		}
		if ui.Button(c.Key("plugin-catalog-refresh"),
			locale.PluginRefresh()).Clicked() {
			h.refreshCatalog()
		}
	})
	if h.ui.query != h.ui.lastQuery ||
		h.ui.supportedOnly != h.ui.lastSupported {
		h.ui.lastQuery = h.ui.query
		h.ui.lastSupported = h.ui.supportedOnly
		h.ui.browseLimit = 40
	}
	loaded, loading := h.catalogState()
	if loading {
		ui.Text(c, locale.PluginCatalogLoading()).
			TextColor(th.TextMuted)
	}
	if !loaded && !loading {
		ui.Text(c, locale.PluginCatalogFailed()).
			TextColor(th.TextMuted)
	}
	items := h.browserItems(h.ui.browseLimit)
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
	if len(items) >= h.ui.browseLimit && h.ui.browseLimit < 200 {
		if ui.Button(c.Key("plugin-show-more"),
			locale.PluginShowMore()).Clicked() {
			h.ui.browseLimit = min(h.ui.browseLimit+40, 200)
		}
	}
}

func (h *pluginHost) browserRow(c *ui.Context, th *ui.Theme,
	item plugin.BrowserItem, busy map[string]bool) {
	ui.Column(c).FillWidth().Gap(2).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, item.Name).Bold()
			if item.Core {
				ui.Text(c, locale.PluginCore()).
					FontSize(th.FontSize - 2).TextColor(th.TextMuted)
			}
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
