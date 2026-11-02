package app

import (
	"testing"

	"github.com/helightxu/xinkgo/internal/config"
)

// modelsTurnedOn is settings holding those models: the first two turned
// on, and one that is not.
func modelsTurnedOn() config.AiConfig {
	cfg := config.DefaultAi()
	cfg.Models = []config.ModelEntry{
		{ID: "qwen3:0.6b", Enabled: true},
		{ID: "llava:7b", Enabled: true},
		{ID: "turned-off", Enabled: false},
	}
	return cfg
}

// TestModelFor_theModelNamedIsAsked checks the whole of what the list in
// the place to ask is for: a model picked in the window is the one asked,
// and not the one the settings name.
func TestModelFor_theModelNamedIsAsked(t *testing.T) {
	s := &appState{}
	cfg := modelsTurnedOn()
	cfg.DefaultModel = "llava:7b"
	got, ok := s.modelFor(cfg, "qwen3:0.6b")
	if !ok || got.ID != "qwen3:0.6b" {
		t.Fatalf("asked %q (ok %v), want the one named", got.ID, ok)
	}
}

// TestModelFor_oneThatCannotBeAskedIsNotAsked checks that a name the
// settings do not hold, or one that is turned off in them, is not asked:
// asking is then as it would have been with nothing named at all.
func TestModelFor_oneThatCannotBeAskedIsNotAsked(t *testing.T) {
	s := &appState{}
	cfg := modelsTurnedOn()
	cfg.DefaultModel = "llava:7b"
	for _, named := range []string{"turned-off", "gone", ""} {
		got, ok := s.modelFor(cfg, named)
		if !ok || got.ID != cfg.DefaultModel {
			t.Fatalf("named %q asked %q (ok %v), want the default",
				named, got.ID, ok)
		}
	}
}

// TestModelFor_whatTheSettingsNameComesBeforeTheFirstTurnedOn checks the
// two ways a model is come by when the window has named none: the one the
// settings name, and failing that the first one turned on.
func TestModelFor_whatTheSettingsNameComesBeforeTheFirstTurnedOn(t *testing.T) {
	s := &appState{}
	cfg := modelsTurnedOn()
	cfg.DefaultModel = ""
	got, ok := s.modelFor(cfg, "")
	if !ok || got.ID != "qwen3:0.6b" {
		t.Fatalf("asked %q (ok %v), want the first turned on",
			got.ID, ok)
	}
	// One that is named but turned off is not the one asked either.
	cfg.DefaultModel = "turned-off"
	got, ok = s.modelFor(cfg, "")
	if !ok || got.ID != "qwen3:0.6b" {
		t.Fatalf("asked %q (ok %v), want the first turned on",
			got.ID, ok)
	}
}

// TestModelFor_nothingToAskIsSaid checks that a window with every model
// turned off says so, rather than asking a model that is not there.
func TestModelFor_nothingToAskIsSaid(t *testing.T) {
	s := &appState{}
	cfg := modelsTurnedOn()
	cfg.Models[0].Enabled = false
	cfg.Models[1].Enabled = false
	if _, ok := s.modelFor(cfg, ""); ok {
		t.Fatal("a model was asked although none is turned on")
	}
}
