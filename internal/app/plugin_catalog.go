package app

import (
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

var repoPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

var pluginHTTP = &http.Client{Timeout: 20 * time.Second}

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
			h.finishCatalog(nil, err)
			return
		}
		entries, err := plugin.ParseCatalog(data)
		h.finishCatalog(entries, err)
	}()
}

func (h *pluginHost) finishCatalog(entries []plugin.CatalogEntry, err error) {
	compatible := h.refreshCompatible()
	h.mu.Lock()
	if err == nil {
		h.catalog = entries
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
}

// browserItems is the catalog as the window should draw it: filtered by
// the compatibility list, the search query and the supported-only switch.
func (h *pluginHost) browserItems(limit int) []plugin.BrowserItem {
	h.mu.RLock()
	key := strconv.Itoa(len(h.catalog)) + "|" +
		strconv.Itoa(len(h.compatible)) + "|" +
		strconv.FormatUint(h.catalogRev, 10) + "|" +
		strconv.FormatBool(h.ui.supportedOnly) + "|" +
		h.ui.query + "|" + strconv.Itoa(limit)
	if key == h.browserKey {
		cached := h.browser
		h.mu.RUnlock()
		return cached
	}
	entries := append([]plugin.CatalogEntry(nil), h.catalog...)
	compatible := append([]plugin.CompatibleEntry(nil), h.compatible...)
	query := h.ui.query
	supportedOnly := h.ui.supportedOnly
	h.mu.RUnlock()
	items := plugin.FilterCatalog(entries, compatible, query,
		supportedOnly, limit)
	h.mu.Lock()
	h.browserKey = key
	h.browser = items
	h.mu.Unlock()
	return items
}

func (h *pluginHost) catalogState() (loaded, loading bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.catalogLoaded, h.catalogLoading
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
	grants := make([]string, 0, len(report.Capabilities))
	for _, one := range report.Capabilities {
		grants = append(grants, string(one))
	}
	h.mu.Lock()
	h.saved = h.saved.WithEnabled(item.ID, true).
		WithGrants(item.ID, grants)
	saved := h.saved
	h.mu.Unlock()
	if err := state.SavePlugins(h.dir, saved); err != nil {
		slog.Warn("plugin install state not written", "err", err)
	}
	h.refresh()
	h.setStatus(locale.PluginInstalledOK(item.Name))
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
