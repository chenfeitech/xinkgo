package view

import (
	"context"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
)

// workspaceWithModels is a window whose settings hold those models — every
// one of them turned on, and the first two of them standing for a model
// the machine holds and one written in by hand — and whose asking is
// written down: which model was asked of.
func workspaceWithModels(
	t *testing.T, models ...string,
) (*Workspace, *aiSession, *string) {
	t.Helper()
	_, a, _ := vaultWithNotes(t)
	w := NewWorkspace(config.Default())
	w.Config.Ai.Enabled = true
	w.Config.Ai.Models = nil
	for _, id := range models {
		w.Config.Ai.Models = append(w.Config.Ai.Models,
			config.ModelEntry{ID: id, Label: id, Enabled: true})
	}
	if _, err := w.Tabs.Open(a); err != nil {
		t.Fatalf("Open: %v", err)
	}
	asked := ""
	w.OnAsk = func(
		_ context.Context, _ []ai.Message, model string,
	) (<-chan ai.Event, error) {
		asked = model
		out := make(chan ai.Event, 2)
		out <- ai.Delta("here")
		out <- ai.Done()
		close(out)
		return out, nil
	}
	s := w.aiOf(a)
	s.Open = true
	return w, s, &asked
}

// shown is the place to ask, drawn once: what the window shows of a model
// is decided as it is drawn, as it is for any other setting.
func composeShown(t *testing.T, w *Workspace, s *aiSession) {
	t.Helper()
	n := w.Tabs.ActiveNote()
	if n == nil {
		t.Fatal("no note is open")
	}
	tester := ui.NewTester(func(c *ui.Context) {
		w.aiCompose(c, s, n)
	}, 720, 120)
	tester.Frame()
}

// TestAiModelNames_onlyWhatCanBeAsked checks that what the bar offers is
// what it can ask: a model turned off is not offered, and neither is one
// with no name.
func TestAiModelNames_onlyWhatCanBeAsked(t *testing.T) {
	w := NewWorkspace(config.Default())
	w.Config.Ai.Models = []config.ModelEntry{
		{ID: "qwen3:0.6b", Enabled: true},
		{ID: "llava:7b", Enabled: false},
		{ID: "  ", Enabled: true},
	}
	got := w.aiModelNames()
	if len(got) != 1 || got[0] != "qwen3:0.6b" {
		t.Fatalf("offered = %v, want only the one that is turned on", got)
	}
}

// TestAiBar_theModelTheSettingsNameIsAsked checks that a bar nobody has
// picked in asks the model the settings name.
func TestAiBar_theModelTheSettingsNameIsAsked(t *testing.T) {
	w, s, asked := workspaceWithModels(t, "qwen3:0.6b", "llava:7b")
	w.Config.Ai.DefaultModel = "llava:7b"
	composeShown(t, w, s)

	n := w.Tabs.ActiveNote()
	s.Input = "hello"
	w.aiAsk(n, s)
	settle(t, s)
	if *asked != "llava:7b" {
		t.Fatalf("asked %q, want the one the settings name", *asked)
	}
}

// TestAiBar_theModelPickedInTheBarIsAsked checks the whole of what the
// list in the place to ask is for: a model picked there is the one asked,
// and not the one the settings name.
func TestAiBar_theModelPickedInTheBarIsAsked(t *testing.T) {
	w, s, asked := workspaceWithModels(t, "qwen3:0.6b", "llava:7b")
	w.Config.Ai.DefaultModel = "llava:7b"
	composeShown(t, w, s)
	// Picking one in the bar is the hand being laid on it.
	w.AiModel = "qwen3:0.6b"
	w.aiPicked = true

	n := w.Tabs.ActiveNote()
	s.Input = "hello"
	w.aiAsk(n, s)
	settle(t, s)
	if *asked != "qwen3:0.6b" {
		t.Fatalf("asked %q, want the one picked in the bar", *asked)
	}
}

// TestAiBar_aModelThatHasGoneSinceIsLetGoOf checks that a model taken out
// of the settings while it was the one picked is not asked and found
// wanting: the bar goes back to the one the settings name.
func TestAiBar_aModelThatHasGoneSinceIsLetGoOf(t *testing.T) {
	w, s, asked := workspaceWithModels(t, "qwen3:0.6b", "llava:7b")
	w.Config.Ai.DefaultModel = "llava:7b"
	composeShown(t, w, s)
	w.AiModel = "qwen3:0.6b"
	w.aiPicked = true
	// It is taken out of the settings, or turned off in them.
	w.Config.Ai.Models = w.Config.Ai.Models[1:]
	composeShown(t, w, s)
	if w.AiModel != "llava:7b" {
		t.Fatalf("asking %q, want the one the settings name", w.AiModel)
	}

	n := w.Tabs.ActiveNote()
	s.Input = "hello"
	w.aiAsk(n, s)
	settle(t, s)
	if *asked != "llava:7b" {
		t.Fatalf("asked %q, want the one the settings name", *asked)
	}
}

// TestAiBar_theModelsAreOfferedInThePlaceToAsk checks that the place to
// ask shows which model is asked, so that it may be changed there and not
// only in the settings.
func TestAiBar_theModelsAreOfferedInThePlaceToAsk(t *testing.T) {
	w, s, _ := workspaceWithModels(t, "qwen3:0.6b", "llava:7b")
	w.Config.Ai.DefaultModel = "qwen3:0.6b"
	n := w.Tabs.ActiveNote()
	tester := ui.NewTester(func(c *ui.Context) {
		w.aiCompose(c, s, n)
	}, 720, 120)
	tester.Frame()
	if !tester.HasText("qwen3:0.6b") {
		t.Errorf("the place to ask does not say which model is asked: %v",
			tester.Texts())
	}
}
