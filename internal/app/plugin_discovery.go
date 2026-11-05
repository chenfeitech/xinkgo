package app

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/helightxu/xinkgo/internal/plugin"
)

// discover reads the three plugin folders without executing any code.
// A later source overrides an earlier one with the same id: application
// folder, then the Obsidian compatibility folder, then Xinkgo's own
// folder in the vault. Bad packages become issues the manager can show.
func (h *pluginHost) discover() (map[string]plugin.Report, []plugin.Report) {
	h.mu.RLock()
	root := h.root
	h.mu.RUnlock()
	reports := map[string]plugin.Report{}
	issues := []plugin.Report{}
	places := []struct {
		dir    string
		source string
	}{{h.userDir, "user"}}
	if root != "" {
		places = append(places,
			struct {
				dir    string
				source string
			}{filepath.Join(root, ".obsidian", "plugins"), "compat"},
			struct {
				dir    string
				source string
			}{filepath.Join(root, ".xink", "plugins"), "vault"})
	}
	for _, place := range places {
		entries, err := os.ReadDir(place.dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			id := entry.Name()
			dir := filepath.Join(place.dir, id)
			one, issue := scanPluginPackage(dir, id, place.source)
			if issue != nil {
				issues = append(issues, *issue)
				slog.Warn("plugin package is not usable", "id", id,
					"dir", dir, "issues", issue.Issues)
				continue
			}
			if _, exists := reports[id]; exists {
				issues = append(issues, duplicatePluginIssue(id, dir,
					place.source))
			}
			reports[id] = one
		}
	}
	return reports, issues
}

func scanPluginPackage(dir, id, source string) (plugin.Report, *plugin.Report) {
	bad := func(code, arg string) *plugin.Report {
		return &plugin.Report{
			ID: id, Name: id, Dir: dir, Source: source,
			Level:  plugin.LevelUnsupported,
			Issues: []plugin.Issue{{Code: code, Arg: arg}},
		}
	}
	manifestData, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return plugin.Report{}, bad("manifestNotRead", err.Error())
	}
	m, err := plugin.ParseManifest(manifestData)
	if err != nil {
		return plugin.Report{}, bad("badManifest", err.Error())
	}
	if err := plugin.ValidateManifest(m, id); err != nil {
		return plugin.Report{}, bad("badManifest", err.Error())
	}
	mainJS, err := os.ReadFile(filepath.Join(dir, "main.js"))
	if err != nil {
		return plugin.Report{}, bad("mainJSNotRead", err.Error())
	}
	_, cssErr := os.Stat(filepath.Join(dir, "styles.css"))
	report := plugin.ScanPackage(m, mainJS, dir, source, cssErr == nil)
	return report, nil
}

func duplicatePluginIssue(id, dir, source string) plugin.Report {
	return plugin.Report{
		ID: id, Name: id, Dir: dir, Source: source,
		Level:  plugin.LevelUnsupported,
		Issues: []plugin.Issue{{Code: "duplicateID", Arg: source}},
	}
}
