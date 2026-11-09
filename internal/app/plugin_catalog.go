package app

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/plugin"
	"github.com/helightxu/xinkgo/internal/state"
)

const communityCatalogURL = "https://raw.githubusercontent.com/obsidianmd/obsidian-releases/master/community-plugins.json"
const communityStatsURL = "https://raw.githubusercontent.com/obsidianmd/obsidian-releases/master/community-plugin-stats.json"

var repoPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

var pluginHTTP = &http.Client{Timeout: 20 * time.Second}
var pluginDetailSem = make(chan struct{}, 4)

// refreshCompatible loads the checked list shipped with the app, or the
// user's own copy in the application data directory. A bad file falls back
// to the shipped list instead of emptying the browser.
func (h *pluginHost) refreshCompatible() []plugin.CompatibleEntry {
	paths := []string{
		filepath.Join(h.userDir, "compatible.json"),
		filepath.Join(h.resources, "obsidian-plugins", "compatible.json"),
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		entries, err := plugin.ParseCompatible(data)
		if err != nil {
			slog.Warn("compatible plugin list is not usable", "path", path,
				"err", err)
			continue
		}
		h.mu.Lock()
		h.compatible = entries
		h.catalogRev++
		h.mu.Unlock()
		return entries
	}
	return nil
}

func (h *pluginHost) compatibleList() []plugin.CompatibleEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return append([]plugin.CompatibleEntry(nil), h.compatible...)
}

func (h *pluginHost) refreshCatalog() {
	h.mu.Lock()
	if h.catalogLoading {
		h.mu.Unlock()
		return
	}
	h.catalogLoading = true
	h.mu.Unlock()
	h.setStatus(locale.PluginStatus(locale.PluginCatalogLoading()))
	go func() {
		data, err := fetchPluginURL(communityCatalogURL, 16<<20)
		if err != nil {
			h.finishCatalog(nil, nil, err)
			return
		}
		entries, err := plugin.ParseCatalog(data)
		stats := map[string]plugin.PluginStats{}
		if statsData, statsErr := fetchPluginURL(communityStatsURL,
			16<<20); statsErr == nil {
			if parsed, parseErr := plugin.ParseStats(statsData); parseErr == nil {
				stats = parsed
			} else {
				slog.Warn("community plugin stats", "err", parseErr)
			}
		} else {
			slog.Warn("community plugin stats", "err", statsErr)
		}
		h.finishCatalog(entries, stats, err)
	}()
}

func (h *pluginHost) finishCatalog(entries []plugin.CatalogEntry,
	stats map[string]plugin.PluginStats, err error) {
	compatible := h.refreshCompatible()
	h.mu.Lock()
	if err == nil {
		h.catalog = entries
		if stats != nil {
			h.stats = stats
		}
		h.compatible = compatible
		h.catalogLoaded = true
	}
	h.catalogRev++
	h.catalogLoading = false
	h.mu.Unlock()
	if err != nil {
		slog.Warn("community plugin catalog", "err", err)
		h.setStatus(locale.PluginStatus(locale.PluginCatalogFailed()))
		return
	}
	h.setStatus(locale.PluginCatalogLoaded(strconv.Itoa(len(entries))))
	h.loadDetails()
}

// browserItems is the catalog as the window should draw it: filtered by
// the compatibility list, the search query and the supported-only switch.
func (h *pluginHost) browserItems(limit int) []plugin.BrowserItem {
	view, rev := h.catalogSnapshot()
	h.mu.RLock()
	key := strconv.FormatUint(rev, 10) + "|" +
		strconv.FormatBool(h.ui.supportedOnly) + "|" +
		strconv.FormatBool(h.ui.showDegraded) + "|" +
		strings.ToLower(strings.TrimSpace(h.ui.query)) + "|" +
		strconv.Itoa(limit)
	if key == h.browserKey {
		cached := h.browser
		h.mu.RUnlock()
		return cached
	}
	query := strings.ToLower(strings.TrimSpace(h.ui.query))
	supportedOnly := h.ui.supportedOnly
	showDegraded := h.ui.showDegraded
	h.mu.RUnlock()
	out := make([]plugin.BrowserItem, 0, min(limit, len(view)))
	for _, one := range view {
		if one.Core || supportedOnly && !one.Compatible ||
			!matchPluginQuery(one, query) {
			continue
		}
		// A degraded plugin only loads through the tolerant shim; its
		// own views and editor features do not work. Keep those out of
		// the browse list unless the user asks to see them.
		if one.Degraded && !showDegraded {
			continue
		}
		out = append(out, one)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	h.mu.Lock()
	h.browserKey = key
	h.browser = out
	h.mu.Unlock()
	return out
}

// catalogSnapshot is the whole community catalog, annotated once and
// sorted by downloads. A frame only filters this cached view; it does not
// copy the 10k-entry stats map or sort 10k rows again.
func (h *pluginHost) catalogSnapshot() ([]plugin.BrowserItem, uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.catalogViewRev == h.catalogRev && h.catalogView != nil {
		return h.catalogView, h.catalogViewRev
	}
	entries := append([]plugin.CatalogEntry(nil), h.catalog...)
	compatible := append([]plugin.CompatibleEntry(nil), h.compatible...)
	stats := make(map[string]plugin.PluginStats, len(h.stats))
	for id, one := range h.stats {
		stats[id] = one
	}
	details := make(map[string]plugin.PluginDetail, len(h.details))
	for id, one := range h.details {
		details[id] = one
	}
	items := plugin.FilterCatalog(entries, compatible, "", false, 0)
	items = plugin.AnnotateCatalog(items, stats, details)
	items = plugin.SortByDownloads(items)
	h.catalogView = items
	h.catalogViewRev = h.catalogRev
	return items, h.catalogViewRev
}

func matchPluginQuery(one plugin.BrowserItem, query string) bool {
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(one.Name), query) ||
		strings.Contains(strings.ToLower(one.ID), query) ||
		strings.Contains(strings.ToLower(one.Author), query) ||
		strings.Contains(strings.ToLower(one.Description), query)
}

// coreItems is the curated Core plugin list, cached with the catalog view.
func (h *pluginHost) coreItems() []plugin.BrowserItem {
	view, rev := h.catalogSnapshot()
	h.mu.RLock()
	if h.coreRev == rev && h.coreCache != nil {
		cached := h.coreCache
		h.mu.RUnlock()
		return cached
	}
	h.mu.RUnlock()
	core := make([]plugin.BrowserItem, 0, 16)
	for _, one := range view {
		if one.Core {
			core = append(core, one)
		}
	}
	h.mu.Lock()
	h.coreRev = rev
	h.coreCache = core
	h.mu.Unlock()
	return core
}

func (h *pluginHost) catalogState() (loaded, loading bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.catalogLoaded, h.catalogLoading
}

// prepareSettings is called when the settings pane opens or becomes the
// chosen group. It starts the disk and network work off the view frame.
func (h *pluginHost) prepareSettings() {
	go h.refresh()
	loaded, loading := h.catalogState()
	if !loaded && !loading {
		h.refreshCatalog()
	}
}

// loadDetails fetches release metadata for the visible compatible rows.
// Unsupported rows keep the catalog's own metadata; no plugin code runs.
func (h *pluginHost) loadDetails() {
	for _, item := range h.detailCandidates(20) {
		if !item.Compatible {
			continue
		}
		h.mu.Lock()
		if h.detailLoading[item.ID] || h.details[item.ID].Loaded {
			h.mu.Unlock()
			continue
		}
		h.detailLoading[item.ID] = true
		h.mu.Unlock()
		go func(one plugin.BrowserItem) {
			pluginDetailSem <- struct{}{}
			defer h.clearDetailLoading(one.ID)
			defer func() { <-pluginDetailSem }()
			h.loadDetail(one)
		}(item)
	}
}

// detailCandidates is the background form of browserItems: it does not
// read the view's query or supported-only switch, so a frame can change
// those while release metadata is being fetched.
func (h *pluginHost) detailCandidates(limit int) []plugin.BrowserItem {
	view, _ := h.catalogSnapshot()
	out := make([]plugin.BrowserItem, 0, min(limit, len(view)))
	for _, one := range view {
		if one.Compatible {
			out = append(out, one)
		}
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func (h *pluginHost) clearDetailLoading(id string) {
	h.mu.Lock()
	delete(h.detailLoading, id)
	h.mu.Unlock()
}

func (h *pluginHost) loadDetail(item plugin.BrowserItem) {
	if !repoPattern.MatchString(item.Repo) {
		return
	}
	detail := plugin.PluginDetail{}
	if data, ok, _ := fetchPluginAsset(item.Repo, "manifest.json",
		64<<10); ok {
		if m, err := plugin.ParseManifest(data); err == nil {
			detail.Version = m.Version
			detail.MinAppVersion = m.MinAppVersion
			detail.AuthorURL = m.AuthorURL
			detail.FundingURL = plugin.FundingText(m.FundingURL)
			detail.IsDesktopOnly = m.IsDesktopOnly
		}
	}
	type shieldResult struct{ kind, value string }
	results := make(chan shieldResult, 3)
	for _, kind := range []string{"stars", "license", "last-commit"} {
		go func(kind string) {
			results <- shieldResult{kind: kind,
				value: fetchPluginShield(item.Repo, kind)}
		}(kind)
	}
	for i := 0; i < 3; i++ {
		one := <-results
		switch one.kind {
		case "stars":
			detail.Stars = one.value
		case "license":
			detail.License = one.value
		case "last-commit":
			detail.LastCommit = one.value
		}
	}
	detail.Loaded = detail.Version != "" || detail.Stars != "" ||
		detail.License != "" || detail.LastCommit != ""
	h.setDetail(item.ID, detail)
}

func (h *pluginHost) setDetail(id string, detail plugin.PluginDetail) {
	h.mu.Lock()
	h.details[id] = detail
	h.catalogRev++
	h.mu.Unlock()
}

func fetchPluginShield(repo, kind string) string {
	address := "https://img.shields.io/github/" + kind + "/" + repo + ".json"
	data, err := fetchPluginURL(address, 64<<10)
	if err != nil {
		return ""
	}
	var reply struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(data, &reply); err != nil {
		return ""
	}
	return strings.TrimSpace(reply.Value)
}

// install downloads a plugin's release assets, rescans them, writes them
// into the application's plugin folder and enables the plugin. The second
// scan is the guard against a compatible-list entry going stale.
func (h *pluginHost) install(item plugin.BrowserItem) {
	if h.isBusy(item.ID) {
		return
	}
	h.setBusy(item.ID, true)
	defer h.setBusy(item.ID, false)
	if !repoPattern.MatchString(item.Repo) {
		h.setStatus(locale.PluginStatus(locale.PluginInstallFailed()))
		return
	}
	manifestData, ok, err := fetchPluginAsset(item.Repo,
		"manifest.json", 64<<10)
	if err != nil || !ok {
		h.installFailed(err)
		return
	}
	m, err := plugin.ParseManifest(manifestData)
	if err != nil || m.ID != item.ID ||
		plugin.ValidateManifest(m, item.ID) != nil {
		h.setStatus(locale.PluginStatus(locale.PluginInstallRefused()))
		return
	}
	mainJS, ok, err := fetchPluginAsset(item.Repo, "main.js", 4<<20)
	if err != nil || !ok || len(mainJS) == 0 {
		h.installFailed(err)
		return
	}
	css, hasCSS, _ := fetchPluginAsset(item.Repo, "styles.css", 256<<10)
	dir := filepath.Join(h.userDir, item.ID)
	report := plugin.ScanPackage(m, mainJS, dir, "user", hasCSS)
	if !report.Supported {
		reason := locale.PluginInstallRefused()
		if len(report.Issues) > 0 {
			reason = locale.PluginIssue(report.Issues[0].Code,
				report.Issues[0].Arg)
		}
		h.setStatus(locale.PluginStatus(reason))
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		h.installFailed(err)
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"),
		manifestData, 0o644); err != nil {
		h.installFailed(err)
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "main.js"), mainJS,
		0o644); err != nil {
		h.installFailed(err)
		return
	}
	if hasCSS {
		_ = os.WriteFile(filepath.Join(dir, "styles.css"), css, 0o644)
	}
	h.mu.Lock()
	h.saved = h.saved.WithEnabled(item.ID, true).
		WithGrants(item.ID, installGrants(report))
	saved := h.saved
	h.mu.Unlock()
	if err := state.SavePlugins(h.dir, saved); err != nil {
		slog.Warn("plugin install state not written", "err", err)
	}
	h.refresh()
	h.setStatus(locale.PluginInstalledNeedsGrants(item.Name))
	h.enable(item.ID, false)
}

func (h *pluginHost) installFailed(err error) {
	if err != nil {
		slog.Warn("plugin install", "err", err)
	}
	h.setStatus(locale.PluginStatus(locale.PluginInstallFailed()))
}

func fetchPluginURL(address string, limit int64) ([]byte, error) {
	resp, err := pluginHTTP.Get(address)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d", address, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("%s: empty response", address)
	}
	return data, nil
}

func fetchPluginAsset(repo, asset string, limit int64) ([]byte, bool, error) {
	addresses := []string{
		"https://github.com/" + repo + "/releases/latest/download/" + asset,
		"https://raw.githubusercontent.com/" + repo + "/HEAD/" + asset,
	}
	for _, address := range addresses {
		data, err := fetchPluginURL(address, limit)
		if err == nil {
			return data, true, nil
		}
		if strings.Contains(err.Error(), "HTTP 404") {
			continue
		}
	}
	return nil, false, nil
}
