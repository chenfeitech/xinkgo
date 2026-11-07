package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// RecentModelsFile is where the last models used in the AI bar are kept.
// It is a run-time convenience, not a setting: config.json stays the
// home of every choice the user made on purpose.
const RecentModelsFile = "recent_models.json"

const maxRecentModels = 8

type recentModelsFile struct {
	Models []string `json:"models"`
}

// LoadRecentModels reads the most recently used models, newest first.
// A missing or broken file simply means none are remembered yet.
func LoadRecentModels(dir string) []string {
	data, err := os.ReadFile(filepath.Join(dir, RecentModelsFile))
	if err != nil {
		return nil
	}
	var f recentModelsFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil
	}
	if len(f.Models) > maxRecentModels {
		f.Models = f.Models[:maxRecentModels]
	}
	return f.Models
}

// SaveRecentModels writes them atomically, as every other state file is.
func SaveRecentModels(dir string, models []string) error {
	clean := make([]string, 0, len(models))
	seen := map[string]bool{}
	for _, id := range models {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		clean = append(clean, id)
		if len(clean) == maxRecentModels {
			break
		}
	}
	return writeJSON(filepath.Join(dir, RecentModelsFile),
		recentModelsFile{Models: clean}, true)
}
