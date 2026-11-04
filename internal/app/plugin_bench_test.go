package app

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/helightxu/xinkgo/internal/plugin"
)

func benchPluginHost(b *testing.B) *pluginHost {
	b.Helper()
	entries := make([]plugin.CatalogEntry, 9885)
	compatible := make([]plugin.CompatibleEntry, 128)
	stats := make(map[string]plugin.PluginStats, len(entries))
	for i := range entries {
		id := fmt.Sprintf("plugin-%d", i)
		entries[i] = plugin.CatalogEntry{
			ID: id, Name: "Plugin " + strconv.Itoa(i),
			Author: "Someone", Description: "A community plugin",
			Repo: "someone/" + id,
		}
		stats[id] = plugin.PluginStats{Downloads: int64(len(entries) - i)}
		if i < len(compatible) {
			compatible[i] = plugin.CompatibleEntry{
				ID: id, Repo: entries[i].Repo, Level: "C4",
				Core: i < 10, Degraded: true,
			}
		}
	}
	installed := make([]plugin.Report, 20)
	for i := range installed {
		installed[i] = plugin.Report{
			ID: entries[i].ID, Name: entries[i].Name,
		}
	}
	return &pluginHost{
		catalog:       entries,
		compatible:    compatible,
		stats:         stats,
		details:       map[string]plugin.PluginDetail{},
		detailLoading: map[string]bool{},
		catalogRev:    1,
		reports:       map[string]plugin.Report{},
		ui: pluginUI{
			supportedOnly: true,
			busy:          map[string]bool{},
			installed:     installed,
		},
	}
}

func BenchmarkCoreItems(b *testing.B) {
	h := benchPluginHost(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.coreItems()
	}
}

func BenchmarkBrowserItemsColdQuery(b *testing.B) {
	h := benchPluginHost(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.ui.query = strconv.Itoa(i)
		_ = h.browserItems(200)
	}
}

func BenchmarkBrowserItemsCached(b *testing.B) {
	h := benchPluginHost(b)
	_ = h.browserItems(200)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.browserItems(200)
	}
}

func BenchmarkPluginUISnapshot(b *testing.B) {
	h := benchPluginHost(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.uiSnapshot()
	}
}
