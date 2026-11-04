package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// PluginState is what the app keeps of its plugins: the enabled ones and
// the capabilities the user granted each one. A missing or broken file
// enables nothing rather than keeping the app from starting.
type PluginState struct {
	Enabled map[string]bool     `json:"enabled,omitempty"`
	Grants  map[string][]string `json:"grants,omitempty"`
}

// EnabledOf reports whether a plugin was enabled on the last run.
func (s PluginState) EnabledOf(id string) bool {
	return id != "" && s.Enabled[id]
}

// GrantsOf gives back a copy of what a plugin may do.
func (s PluginState) GrantsOf(id string) []string {
	if len(s.Grants[id]) == 0 {
		return nil
	}
	return append([]string(nil), s.Grants[id]...)
}

// WithEnabled returns a copy with one plugin's enabled flag changed.
func (s PluginState) WithEnabled(id string, on bool) PluginState {
	out := s.copy()
	if out.Enabled == nil {
		out.Enabled = map[string]bool{}
	}
	if on {
		out.Enabled[id] = true
	} else {
		delete(out.Enabled, id)
	}
	return out
}

// WithGrants returns a copy with one plugin's grants replaced.
func (s PluginState) WithGrants(id string, grants []string) PluginState {
	out := s.copy()
	if out.Grants == nil {
		out.Grants = map[string][]string{}
	}
	if len(grants) == 0 {
		delete(out.Grants, id)
	} else {
		out.Grants[id] = append([]string(nil), grants...)
	}
	return out
}

func (s PluginState) copy() PluginState {
	out := PluginState{
		Enabled: map[string]bool{}, Grants: map[string][]string{},
	}
	for id, on := range s.Enabled {
		out.Enabled[id] = on
	}
	for id, grants := range s.Grants {
		out.Grants[id] = append([]string(nil), grants...)
	}
	return out
}

// LoadPlugins reads which plugins are enabled and what they may do.
func LoadPlugins(dir string) (PluginState, error) {
	data, err := os.ReadFile(filepath.Join(dir, PluginsFile))
	if errors.Is(err, os.ErrNotExist) {
		return PluginState{}, nil
	}
	if err != nil {
		return PluginState{}, fmt.Errorf("read %s: %w", PluginsFile, err)
	}
	var st PluginState
	if err := json.Unmarshal(data, &st); err != nil {
		return PluginState{}, fmt.Errorf("parse %s: %w", PluginsFile, err)
	}
	return st.normalized(), nil
}

// SavePlugins writes the enabled flags and grants atomically.
func SavePlugins(dir string, st PluginState) error {
	return writeJSON(filepath.Join(dir, PluginsFile), st.normalized(), true)
}

func (s PluginState) normalized() PluginState {
	out := PluginState{
		Enabled: map[string]bool{}, Grants: map[string][]string{},
	}
	for id, on := range s.Enabled {
		if id != "" && on {
			out.Enabled[id] = true
		}
	}
	for id, grants := range s.Grants {
		if id == "" {
			continue
		}
		seen := map[string]bool{}
		kept := []string{}
		for _, one := range grants {
			if one == "" || seen[one] {
				continue
			}
			seen[one] = true
			kept = append(kept, one)
		}
		if len(kept) > 0 {
			out.Grants[id] = kept
		}
	}
	return out
}
