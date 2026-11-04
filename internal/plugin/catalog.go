package plugin

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// CatalogEntry is one plugin in Obsidian's community-plugins.json.
type CatalogEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Repo        string `json:"repo"`
}

// CompatibleEntry is one plugin a maintainer has checked against this
// build. The list ships with the app and may be overridden by a file in
// the application's own folder.
type CompatibleEntry struct {
	ID         string `json:"id"`
	Repo       string `json:"repo"`
	Level      string `json:"level"`
	Note       string `json:"note"`
	VerifiedAt string `json:"verifiedAt"`
}

// BrowserItem is one catalog row with its compatibility annotation.
type BrowserItem struct {
	CatalogEntry
	Compatible       bool
	Level            string
	Note             string
	VerifiedAt       string
	Downloads        int64
	Updated          int64
	VersionDownloads int64
	Version          string
	MinAppVersion    string
	AuthorURL        string
	FundingURL       string
	IsDesktopOnly    bool
	Stars            string
	License          string
	LastCommit       string
	DetailLoaded     bool
}

// PluginStats is the official community download count and update time.
// ByVersion holds the per-release counts the stats file carries beside
// the two well-known fields.
type PluginStats struct {
	Downloads int64
	Updated   int64
	ByVersion map[string]int64
}

// PluginDetail is extra metadata fetched on demand for a visible plugin:
// the release manifest and a few public shield values. It never blocks
// the catalog itself from loading.
type PluginDetail struct {
	Version       string
	MinAppVersion string
	AuthorURL     string
	FundingURL    string
	IsDesktopOnly bool
	Stars         string
	License       string
	LastCommit    string
	Loaded        bool
}

// ParseCatalog reads the official community plugin list.
func ParseCatalog(data []byte) ([]CatalogEntry, error) {
	var entries []CatalogEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse community catalog: %w", err)
	}
	out := make([]CatalogEntry, 0, len(entries))
	for _, one := range entries {
		if one.ID == "" || one.Repo == "" ||
			!strings.Contains(one.Repo, "/") {
			continue
		}
		out = append(out, one)
	}
	return out, nil
}

// ParseCompatible reads the checked compatibility list.
func ParseCompatible(data []byte) ([]CompatibleEntry, error) {
	var entries []CompatibleEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse compatible list: %w", err)
	}
	out := make([]CompatibleEntry, 0, len(entries))
	for _, one := range entries {
		if one.ID == "" || one.Repo == "" {
			continue
		}
		if one.Level == "" {
			one.Level = LevelCore.Code()
		}
		out = append(out, one)
	}
	return out, nil
}

// FilterCatalog annotates the official list with the compatible list and
// keeps the rows a user asked for. supportedOnly is the browser's default:
// it hides the thousands of plugins the P0 host cannot yet run.
func FilterCatalog(entries []CatalogEntry, compatible []CompatibleEntry,
	query string, supportedOnly bool, limit int) []BrowserItem {
	checked := make(map[string]CompatibleEntry, len(compatible))
	for _, one := range compatible {
		checked[one.ID] = one
	}
	q := strings.ToLower(strings.TrimSpace(query))
	out := []BrowserItem{}
	for _, one := range entries {
		if q != "" &&
			!strings.Contains(strings.ToLower(one.Name), q) &&
			!strings.Contains(strings.ToLower(one.ID), q) &&
			!strings.Contains(strings.ToLower(one.Author), q) &&
			!strings.Contains(strings.ToLower(one.Description), q) {
			continue
		}
		known, ok := checked[one.ID]
		if supportedOnly && !ok {
			continue
		}
		if ok && known.Repo != "" {
			one.Repo = known.Repo
		}
		item := BrowserItem{CatalogEntry: one, Compatible: ok,
			Level: known.Level, Note: known.Note,
			VerifiedAt: known.VerifiedAt}
		if !ok {
			item.Level = "C0"
			item.Note = "notChecked"
		}
		out = append(out, item)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

// ParseStats reads Obsidian's community-plugin-stats.json. The file is a
// map from plugin id to an object whose keys are "downloads", "updated"
// and release versions, each holding a number.
func ParseStats(data []byte) (map[string]PluginStats, error) {
	var raw map[string]map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse community plugin stats: %w", err)
	}
	out := make(map[string]PluginStats, len(raw))
	for id, fields := range raw {
		stat := PluginStats{ByVersion: map[string]int64{}}
		for key, value := range fields {
			number, ok := statNumber(value)
			if !ok {
				continue
			}
			switch key {
			case "downloads":
				stat.Downloads = number
			case "updated":
				stat.Updated = number
			default:
				stat.ByVersion[key] = number
			}
		}
		out[id] = stat
	}
	return out, nil
}

func statNumber(value any) (int64, bool) {
	switch one := value.(type) {
	case float64:
		return int64(one), true
	case int64:
		return one, true
	case json.Number:
		n, err := one.Int64()
		return n, err == nil
	case string:
		n, err := strconv.ParseInt(one, 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

// AnnotateCatalog joins the filtered rows with download statistics and
// on-demand release metadata. It is pure so the browser cache can be
// tested without windows or HTTP.
func AnnotateCatalog(items []BrowserItem,
	stats map[string]PluginStats, details map[string]PluginDetail) []BrowserItem {
	out := append([]BrowserItem(nil), items...)
	for i := range out {
		stat, ok := stats[out[i].ID]
		if ok {
			out[i].Downloads = stat.Downloads
			out[i].Updated = stat.Updated
		}
		detail, ok := details[out[i].ID]
		if !ok {
			continue
		}
		out[i].Version = detail.Version
		out[i].MinAppVersion = detail.MinAppVersion
		out[i].AuthorURL = detail.AuthorURL
		out[i].FundingURL = detail.FundingURL
		out[i].IsDesktopOnly = detail.IsDesktopOnly
		out[i].Stars = detail.Stars
		out[i].License = detail.License
		out[i].LastCommit = detail.LastCommit
		out[i].DetailLoaded = detail.Loaded
		if detail.Version != "" && stat.ByVersion != nil {
			out[i].VersionDownloads = stat.ByVersion[detail.Version]
		}
	}
	return out
}

// FundingText turns a manifest's fundingUrl into one short line, whether
// the author wrote a string or a map of names to addresses.
func FundingText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}
	var links map[string]string
	if err := json.Unmarshal(raw, &links); err != nil {
		return string(raw)
	}
	keys := make([]string, 0, len(links))
	for key := range links {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+" "+links[key])
	}
	return strings.Join(parts, ", ")
}
