package plugin

import (
	"encoding/json"
	"fmt"
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
	Compatible bool
	Level      string
	Note       string
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
			Level: known.Level, Note: known.Note}
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
