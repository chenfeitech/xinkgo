package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
)

// aiSettingsShown is the settings of the AI as the window would draw them:
// two models the machine holds, one of them already asked, and one the
// user wrote in by hand.
func aiSettingsShown(t *testing.T) *appState {
	t.Helper()
	s := &appState{}
	s.draft = settings{cfg: config.Default()}
	cfg := &s.draft.cfg.Ai
	cfg.Enabled = true
	s.draft.ollama.models = []ai.OllamaModel{
		{Name: "llava:7b", Size: 4_700_000_000, Family: "llama",
			Families: []string{"llama", "clip"},
			Capabilities: []string{
				ai.CapCompletion, ai.CapVision,
			}, Guessed: true},
		{Name: "qwen3:0.6b", Size: 522_928_832, Family: "qwen3",
			ParameterSize: "0.6B", Quantization: "Q8_0",
			Capabilities: []string{ai.CapCompletion, ai.CapTools}},
	}
	s.draft.ollama.at = time.Now()
	cfg.Models = []config.ModelEntry{
		{ID: "qwen3:0.6b", Source: config.ModelSourceOllama,
			Kind: config.ModelOllama, Label: "qwen3:0.6b",
			Enabled:      true,
			Capabilities: []string{ai.CapCompletion, ai.CapTools}},
		{ID: "gpt-4o-mini", Source: config.ModelSourceCustom,
			Kind: config.ModelOpenAICompatible, Label: "gpt-4o-mini",
			Enabled: true, BaseURL: "https://api.openai.com/v1",
			ApiKeySource: config.KeyPlain},
	}
	cfg.DefaultModel = "qwen3:0.6b"
	return s
}

// TestAiSettings_theWindowSaysWhatTheMachineHolds checks the drawn window:
// the models the machine holds are listed with what they can do, the one
// that is asked is marked, and the model written in by hand is on show
// beside them.
func TestAiSettings_theWindowSaysWhatTheMachineHolds(t *testing.T) {
	s := aiSettingsShown(t)
	tester := ui.NewTester(func(c *ui.Context) { s.ai(c) }, 760, 1000)
	tester.Frame()
	tester.Frame()
	for _, want := range []string{
		"Ollama on this machine", "Refresh", "2 models",
		"qwen3:0.6b", "llava:7b", "499 MB", "4.4 GB",
		"Chat completion", "Tool calling", "Vision", "Default",
		"Which one is asked", "gpt-4o-mini",
	} {
		if !tester.HasText(want) {
			t.Errorf("the window does not say %q", want)
		}
	}
}

// TestAiSettings_aModelWrittenInByHandMayBeOpened checks that the model the
// user wrote in has the fields it is written in with: its name, where it
// is asked, what it is asked with, and what way it is spoken to.
func TestAiSettings_aModelWrittenInByHandMayBeOpened(t *testing.T) {
	s := aiSettingsShown(t)
	tester := ui.NewTester(func(c *ui.Context) { s.ai(c) }, 760, 1000)
	tester.Frame()
	if err := tester.Click("Open"); err != nil {
		t.Fatalf("the row could not be opened: %v", err)
	}
	tester.Frame()
	for _, want := range []string{
		"Model", "Address", "Key", "Speaks", "What it can do",
		"List them", "Make it the default",
	} {
		if !tester.HasText(want) {
			t.Errorf("an opened row does not say %q", want)
		}
	}
}

// TestAiSettings_tickingAModelTheMachineHoldsAsksIt checks the whole point
// of asking the machine: a model it holds is taken into the settings by
// ticking it, as one of the machine's own.
func TestAiSettings_tickingAModelTheMachineHoldsAsksIt(t *testing.T) {
	s := aiSettingsShown(t)
	tester := ui.NewTester(func(c *ui.Context) { s.ai(c) }, 760, 1000)
	tester.Frame()
	if err := tester.Click("llava:7b"); err != nil {
		t.Fatalf("the model could not be ticked: %v", err)
	}
	tester.Frame()
	on, there := s.draft.ollamaOn("llava:7b")
	if !on || !there {
		t.Fatalf("on = %v there = %v, want it asked", on, there)
	}
}

// The AI settings show one group at a time. The models are the first
// screen; switching to Asking hides them and shows the asking knobs.
func TestAiSettings_sectionsSwitch(t *testing.T) {
	s := aiSettingsShown(t)
	tester := ui.NewTester(func(c *ui.Context) { s.ai(c) }, 760, 1000)
	tester.Frame()
	if !tester.HasText("Which one is asked") {
		t.Fatal("the models section is not the first screen")
	}
	s.draft.aiSection = "chat"
	tester.Frame()
	if !tester.HasText("Answer in") {
		t.Error("the asking section was not drawn")
	}
	if tester.HasText("Which one is asked") {
		t.Error("the models section stayed on show")
	}
	s.draft.aiSection = "tools"
	tester.Frame()
	if !tester.HasText("Skills") {
		t.Error("the tools section was not drawn")
	}
}
