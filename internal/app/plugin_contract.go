package app

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/helightxu/xinkgo/internal/plugin"
)

// refreshLibrary loads the official plugin list shipped with the app. A
// missing or unusable list must not stop startup: it only means no plugin
// gets the library's guarantee this run.
func (h *pluginHost) refreshLibrary() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.library = plugin.Library{}
	h.libraryIDs = map[string]bool{}
	path := filepath.Join(h.resources, "xinkgo-plugins", "library.json")
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Warn("official plugin library missing", "path", path,
			"err", err)
		return
	}
	lib, err := plugin.ParseLibrary(data)
	if err != nil {
		slog.Warn("official plugin library is not usable", "path", path,
			"err", err)
		return
	}
	h.library = lib
	for _, one := range lib.Plugins {
		h.libraryIDs[one.ID] = true
	}
}

// scanContracts gives every discovered package a contract verdict. refresh
// calls it, not the per-frame view functions, so no view does file I/O.
// External plugins get a verdict too, but only library members are held to
// it: the L2 compatibility layer keeps running as before.
func (h *pluginHost) scanContracts(
	reports map[string]plugin.Report) map[string]plugin.ContractResult {
	out := make(map[string]plugin.ContractResult, len(reports))
	for id, report := range reports {
		mainJS, err := os.ReadFile(filepath.Join(report.Dir, "main.js"))
		if err != nil {
			continue
		}
		out[id] = plugin.ValidateContract(report, mainJS)
	}
	return out
}

// installGrants is the capability set a one-click install writes. Xinkgo
// never grants implicitly; the user ticks each capability in Grants and
// reloads. Keeping the policy in one function makes it explicit and testable.
func installGrants(_ plugin.Report) []string { return nil }

// contractRefused reports whether an official-library plugin is barred from
// running by a failed capability contract. External (L2) plugins and library
// members whose verdict has not been computed are never refused here.
func (h *pluginHost) contractRefused(id string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if !h.libraryIDs[id] {
		return false
	}
	contract, checked := h.contracts[id]
	return checked && !contract.OK
}
