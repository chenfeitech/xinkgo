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
	"github.com/helightxu/xinkgo/internal/view"
)

type pluginViewState struct {
	status    string
	installed []plugin.Report
	busy      map[string]bool
	coreIDs   map[string]bool
}

// openPlugins opens the community plugins group of the settings window.
// The plugin browser lives there, not in a second window of its own.
func (s *appState) openPlugins() {
	s.openSettingsTab(tabCommunityPlugins)
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

// coreSettingsView draws the Core Integrated Plugins group: Xinkgo's own
// built-in features, each a switch. No third-party code is listed here, so
// it needs neither the catalog nor the community browser. It is one of the
// two plugin groups in the settings window's navigation.
func (h *pluginHost) coreSettingsView(c *ui.Context) {
	th := c.Theme()
	state := h.uiSnapshot()
	ui.Column(c).FillWidth().Gap(th.Space(2)).Children(func() {
		ui.Text(c, locale.PluginBuiltinNote()).
			FontSize(th.FontSize - 1).TextColor(th.TextMuted)
		h.builtinView(c, th)
		if state.status != "" {
			ui.Text(c, state.status).FontSize(th.FontSize - 1).
				TextColor(th.TextMuted)
		}
	})
}

// communitySettingsView draws the Community Plugins group: third-party
// Obsidian-compatible plugins. A sub-tab switches between what is already
// installed, the curated compatible core list, and the community browser.
// It does no I/O; prepareSettings starts the background work.
func (h *pluginHost) communitySettingsView(c *ui.Context) {
	th := c.Theme()
	state := h.uiSnapshot()
	installedLabel := locale.PluginTabInstalled()
	curatedLabel := locale.PluginTabCurated()
	browseLabel := locale.PluginTabBrowse()
	selected := browseLabel
	switch h.ui.tab {
	case "installed":
		selected = installedLabel
	case "core":
		selected = curatedLabel
	}
	ui.Column(c).FillWidth().Gap(th.Space(2)).Children(func() {
		ui.Row(c).FillWidth().Gap(th.Space(1)).
			AlignItems(ui.Center).
			Padding(th.Space(1), settingsRowPad).Children(func() {
			ui.Select(c.Key("plugin-tab"), &selected,
				[]string{installedLabel, curatedLabel, browseLabel})
			if ui.Button(c.Key("plugin-refresh"),
				locale.PluginRefresh()).Clicked() {
				h.refresh()
				h.refreshCatalog()
			}
		})
		switch selected {
		case installedLabel:
			h.ui.tab = "installed"
		case curatedLabel:
			h.ui.tab = "core"
		default:
			h.ui.tab = "browse"
		}
		ui.Text(c, locale.PluginThirdParty()).
			FontSize(th.FontSize - 1).TextColor(th.TextMuted)
		switch h.ui.tab {
		case "installed":
			h.installedView(c, th, state)
		case "core":
			h.coreView(c, th, state)
		default:
			h.browseView(c, th, state)
		}
		if state.status != "" {
			ui.Text(c, state.status).FontSize(th.FontSize - 1).
				TextColor(th.TextMuted)
		}
	})
}

// builtinView is the real core of Xinkgo: its own features, each a switch
// that hides or shows a part of the window. No third-party code runs here.
// The switches share one rounded card, as Obsidian's core plugin list does.
func (h *pluginHost) builtinView(c *ui.Context, th *ui.Theme) {
	cardBox(c, func() {
		group(c, locale.PluginGroupBuiltin(), func() {
			for _, one := range plugin.CorePlugins {
				name, note := builtinCoreText(one.ID)
				id := one.ID
				spec := one
				settingRow(c, name, note, settingsControlWidth, func() {
					on := h.coreOn(id)
					usable := !spec.NeedsWebView || mygo.WebViewAvailable()
					switchBox := ui.Switch(c.Key("plugin-builtin-"+id), &on).
						Label("").Disabled(!usable)
					if switchBox.Changed() {
						h.setCore(id, on)
					}
					if spec.Settings && on && usable && h.st != nil {
						if ui.Button(c.Key("plugin-options-"+id),
							locale.PluginOptionsButton()).Clicked() {
							h.st.openSettingsTab(coreOptionsTab(id))
						}
					}
				})
				if spec.NeedsWebView && !mygo.WebViewAvailable() {
					ui.Text(c, locale.WebNoWebView()).
						FontSize(th.FontSize - 1).TextColor(th.TextMuted)
				}
			}
		})
	})
}

// builtinCoreText gives the name and note of a built-in core plugin, from
// the catalogs so a language switch redraws them on the next frame.
func builtinCoreText(id string) (name, note string) {
	switch id {
	case view.CoreFileExplorer:
		return locale.CoreFileName(), locale.CoreFileNote()
	case view.CoreCommandPalette:
		return locale.CoreCommandPaletteName(), locale.CoreCommandPaletteNote()
	case view.CoreGlobalSearch:
		return locale.CoreGlobalSearchName(), locale.CoreGlobalSearchNote()
	case view.CorePagePreview:
		return locale.CorePagePreviewName(), locale.CorePagePreviewNote()
	case view.CoreOutline:
		return locale.CoreOutlineName(), locale.CoreOutlineNote()
	case view.CoreBacklinks:
		return locale.CoreBacklinksName(), locale.CoreBacklinksNote()
	case view.CoreTagPane:
		return locale.CoreTagPaneName(), locale.CoreTagPaneNote()
	case view.CoreStatusBar:
		return locale.CoreStatusBarName(), locale.CoreStatusBarNote()
	case view.CoreWordCount:
		return locale.CoreWordCountName(), locale.CoreWordCountNote()
	case view.CorePluginRibbon:
		return locale.CorePluginRibbonName(), locale.CorePluginRibbonNote()
	case view.CoreDailyNotes:
		return locale.CoreDailyNotesName(), locale.CoreDailyNotesNote()
	case view.CoreTemplates:
		return locale.CoreTemplatesName(), locale.CoreTemplatesNote()
	case view.CoreWebView:
		return locale.CoreWebViewName(), locale.CoreWebViewNote()
	default:
		return id, ""
	}
}

// cardBox is one rounded card, the way Obsidian sets a plugin apart from
// its neighbours. The theme supplies the surface, border and radius, so a
// light or dark window draws the same shape in its own colours.
func cardBox(c *ui.Context, fn func()) {
	th := c.Theme()
	ui.Column(c).FillWidth().
		Padding(th.Space(2)).Background(th.Surface).
		Border(1, th.Border).Radius(th.Radius+2).
		Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, 0.18)).
		Children(fn)
}

// cardList stacks cards with an even gap between them.
func cardList(c *ui.Context, fn func()) {
	th := c.Theme()
	ui.Column(c).FillWidth().Gap(th.Space(2)).Children(fn)
}

// pluginBadges draws a row's small labels as badges, not as loose text.
func pluginBadges(c *ui.Context, labels []string) {
	for _, label := range labels {
		if label == "" {
			continue
		}
		ui.Badge(c, label).FontSize(11)
	}
}

// pluginHeader is the part of a plugin card that is always on show: the
// name with its badges, a line of metadata, the description, and on the
// far side the switch and buttons that act on the plugin.
func pluginHeader(c *ui.Context, th *ui.Theme, name string,
	badges []string, meta, desc string, controls func()) {
	ui.Row(c).FillWidth().Gap(th.Space(2)).AlignItems(ui.Center).
		Children(func() {
			ui.Column(c).Grow(1).Gap(th.Space(0.5)).Children(func() {
				ui.Row(c).Gap(th.Space(1)).Wrap().AlignItems(ui.Center).
					Children(func() {
						ui.Text(c, name).Bold()
						pluginBadges(c, badges)
					})
				if meta != "" {
					ui.Text(c, meta).FontSize(th.FontSize - 1).
						TextColor(th.TextMuted)
				}
				if desc != "" {
					ui.Text(c, desc).FontSize(th.FontSize - 1).
						TextColor(th.TextMuted)
				}
			})
			ui.Row(c).Justify(ui.End).Gap(th.Space(1)).
				AlignItems(ui.Center).Children(controls)
		})
}

// pluginDetailsToggle folds a card's details and grants away by default.
// It reports whether they are open this frame; the choice is pure UI state.
func (h *pluginHost) pluginDetailsToggle(c *ui.Context, id string) bool {
	open := h.ui.open[id]
	label := locale.PluginShowDetails()
	if open {
		label = locale.PluginHideDetails()
	}
	if ui.Button(c.Key("plugin-details-"+id), label).Clicked() {
		open = !open
		h.ui.open[id] = open
	}
	return open
}

func (h *pluginHost) coreView(c *ui.Context, th *ui.Theme,
	state pluginViewState) {
	items := h.coreItems()
	if len(items) == 0 {
		ui.Text(c, locale.PluginCoreEmpty()).TextColor(th.TextMuted)
		return
	}
	ui.Text(c, locale.PluginDegradedNote()).
		FontSize(th.FontSize - 1).TextColor(th.TextMuted)
	cardList(c, func() {
		for _, item := range items {
			one := item
			cardBox(c, func() {
				h.coreRow(c, th, one, state.busy)
			})
		}
	})
}

func (h *pluginHost) coreRow(c *ui.Context, th *ui.Theme,
	item plugin.BrowserItem, busy map[string]bool) {
	badges := []string{
		locale.PluginCore(),
		locale.PluginLevelLabel(item.Level),
	}
	if item.Degraded {
		badges = append(badges, locale.PluginDegraded())
	}
	if item.Downloads > 0 {
		badges = append(badges, formatPluginCount(item.Downloads))
	}
	if busy[item.ID] {
		badges = append(badges, locale.PluginCatalogLoading())
	}
	meta := strings.Trim(strings.Join([]string{item.Author, item.Repo},
		" · "), " ·")
	open := false
	pluginHeader(c, th, item.Name, badges, meta, item.Description,
		func() {
			if !h.installed(item.ID) {
				if !busy[item.ID] {
					if ui.Button(c.Key("plugin-core-install-"+item.ID),
						locale.PluginInstall()).Clicked() {
						go h.install(item)
					}
				}
			} else {
				on := h.enabled(item.ID)
				if ui.Switch(c.Key("plugin-core-toggle-"+item.ID), &on).
					Label("").Changed() {
					h.setEnabled(item.ID, on)
				}
				if ui.Button(c.Key("plugin-core-reload-"+item.ID),
					locale.PluginReload()).Clicked() {
					h.reload(item.ID)
				}
				if ui.Button(c.Key("plugin-core-folder-"+item.ID),
					locale.PluginOpenFolder()).Clicked() {
					_ = mygo.Shell.OpenPath(
						filepath.Join(h.userDir, item.ID))
				}
			}
			open = h.pluginDetailsToggle(c, item.ID)
		})
	if open {
		ui.Box(c).FillWidth().Height(1).Background(th.Border)
		ui.Column(c).FillWidth().Gap(th.Space(0.5)).Children(func() {
			if !h.installed(item.ID) {
				ui.Text(c, locale.PluginCoreInstallFirst()).
					FontSize(th.FontSize - 1).TextColor(th.TextMuted)
			} else if h.running(item.ID) {
				ui.Text(c, locale.PluginEnabled()).
					FontSize(th.FontSize - 1).TextColor(th.TextMuted)
			} else {
				ui.Text(c, locale.PluginDisabled()).
					FontSize(th.FontSize - 1).TextColor(th.TextMuted)
			}
			h.detailView(c, th, item)
		})
	}
}

func (h *pluginHost) installedView(c *ui.Context, th *ui.Theme,
	state pluginViewState) {
	if len(state.installed) == 0 {
		ui.Text(c, locale.PluginNoInstalled()).TextColor(th.TextMuted)
		return
	}
	cardList(c, func() {
		for _, one := range state.installed {
			report := one
			cardBox(c, func() {
				h.installedRow(c, th, report, state.busy)
			})
		}
	})
}

func (h *pluginHost) installedRow(c *ui.Context, th *ui.Theme,
	one plugin.Report, busy map[string]bool) {
	badges := []string{locale.PluginLevelLabel(one.Level.Code())}
	if one.Degraded {
		badges = append(badges, locale.PluginDegraded())
	}
	if h.running(one.ID) {
		badges = append(badges, locale.PluginEnabled())
	}
	if busy[one.ID] {
		badges = append(badges, locale.PluginCatalogLoading())
	}
	meta := locale.PluginVersionLine(one.Version, one.Author,
		h.sourceLabel(one.Source))
	open := false
	pluginHeader(c, th, one.Name, badges, meta, one.Description,
		func() {
			if one.Supported {
				on := h.enabled(one.ID)
				if ui.Switch(c.Key("plugin-enable-"+one.ID), &on).
					Label("").Changed() {
					h.setEnabled(one.ID, on)
				}
				if ui.Button(c.Key("plugin-reload-"+one.ID),
					locale.PluginReload()).Clicked() {
					h.reload(one.ID)
				}
				if ui.Button(c.Key("plugin-folder-"+one.ID),
					locale.PluginOpenFolder()).Clicked() {
					_ = mygo.Shell.OpenPath(one.Dir)
				}
			}
			if ui.Button(c.Key("plugin-uninstall-"+one.ID),
				locale.PluginUninstall()).Clicked() {
				h.askUninstall(one)
			}
			open = h.pluginDetailsToggle(c, one.ID)
		})
	if open {
		ui.Box(c).FillWidth().Height(1).Background(th.Border)
		ui.Column(c).FillWidth().Gap(th.Space(0.5)).Children(func() {
			for _, issue := range one.Issues {
				ui.Text(c, locale.PluginIssue(issue.Code, issue.Arg)).
					FontSize(th.FontSize - 1).TextColor(th.TextMuted)
			}
			if !one.Supported {
				ui.Text(c, locale.PluginInstallRefused()).
					FontSize(th.FontSize - 1).TextColor(th.TextMuted)
				return
			}
			ui.Text(c, locale.PluginGrants()).Bold().
				FontSize(th.FontSize - 1)
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
}

func (h *pluginHost) browseView(c *ui.Context, th *ui.Theme,
	state pluginViewState) {
	ui.Row(c).FillWidth().Gap(th.Space(1)).AlignItems(ui.Center).
		Padding(th.Space(1), settingsRowPad).Children(func() {
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
		if ui.Checkbox(c.Key("plugin-show-degraded"),
			&h.ui.showDegraded,
			locale.PluginShowDegraded()).Changed() {
			h.ui.browseLimit = 40
		}
		if ui.Button(c.Key("plugin-catalog-refresh"),
			locale.PluginRefresh()).Clicked() {
			h.refreshCatalog()
		}
	})
	ui.Text(c, locale.PluginDegradedNote()).FontSize(th.FontSize - 1).
		TextColor(th.TextMuted)
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
	cardList(c, func() {
		for _, item := range items {
			one := item
			cardBox(c, func() {
				h.browserRow(c, th, one, state.busy)
			})
		}
	})
	if len(items) >= h.ui.browseLimit && h.ui.browseLimit < 200 {
		if ui.Button(c.Key("plugin-show-more"),
			locale.PluginShowMore()).Clicked() {
			h.ui.browseLimit = min(h.ui.browseLimit+40, 200)
		}
	}
}

func (h *pluginHost) browserRow(c *ui.Context, th *ui.Theme,
	item plugin.BrowserItem, busy map[string]bool) {
	badges := []string{}
	if item.Core {
		badges = append(badges, locale.PluginCore())
	}
	badges = append(badges, locale.PluginLevelLabel(item.Level))
	if item.Degraded {
		badges = append(badges, locale.PluginDegraded())
	}
	if h.installed(item.ID) {
		badges = append(badges, locale.PluginInstalled())
	}
	if busy[item.ID] {
		badges = append(badges, locale.PluginCatalogLoading())
	}
	meta := strings.Trim(strings.Join([]string{item.Author, item.Repo},
		" · "), " ·")
	open := false
	pluginHeader(c, th, item.Name, badges, meta, item.Description,
		func() {
			if item.Compatible && !h.installed(item.ID) &&
				!busy[item.ID] {
				if ui.Button(c.Key("plugin-install-"+item.ID),
					locale.PluginInstall()).Clicked() {
					go h.install(item)
				}
			}
			open = h.pluginDetailsToggle(c, item.ID)
		})
	if open {
		ui.Box(c).FillWidth().Height(1).Background(th.Border)
		ui.Column(c).FillWidth().Gap(th.Space(0.5)).Children(func() {
			h.detailView(c, th, item)
			note := item.Note
			if !item.Compatible {
				note = locale.PluginNotChecked()
			}
			if note != "" {
				ui.Text(c, note).FontSize(th.FontSize - 1).
					TextColor(th.TextMuted)
			}
		})
	}
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
			FontSize(th.FontSize - 1).TextColor(th.TextMuted)
	}
	if item.Compatible && !item.DetailLoaded {
		ui.Text(c, locale.PluginDetailLoading()).
			FontSize(th.FontSize - 1).TextColor(th.TextMuted)
	}
}

func pluginDetailText(c *ui.Context, th *ui.Theme, label, value string) {
	ui.Text(c, locale.PluginDetailLine(label, value)).
		FontSize(th.FontSize - 1).TextColor(th.TextMuted)
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
