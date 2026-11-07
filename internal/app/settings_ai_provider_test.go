package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/view"
)

func providerApp(t *testing.T) *appState {
	t.Helper()
	s := &appState{root: t.TempDir(), dir: t.TempDir()}
	s.config = config.Default()
	s.draft = settings{cfg: config.Default()}
	s.draft.cfg.Ai.Enabled = true
	s.ws = view.NewWorkspace(config.Default())
	return s
}

func TestCommitProvider_addsOneService(t *testing.T) {
	s := providerApp(t)
	s.draft.aiProviders.draft = aiProviderDraft{
		open: true, preset: "openai", name: "OpenAI",
		url: "https://api.openai.com/v1",
		key: "sk-test", keyFrom: config.KeyPlain,
	}
	s.commitProvider()
	if len(s.draft.cfg.Ai.Providers) != 1 {
		t.Fatalf("providers = %+v", s.draft.cfg.Ai.Providers)
	}
	p := s.draft.cfg.Ai.Providers[0]
	if p.ID == "" || p.Name != "OpenAI" || !p.ShowInPicker {
		t.Errorf("provider = %+v", p)
	}
	if s.draft.aiProviders.draft.open {
		t.Error("the add form stayed open")
	}
	// Adding the same address again must not duplicate it.
	s.draft.aiProviders.draft = aiProviderDraft{
		open: true, name: "Again", url: "https://api.openai.com/v1",
	}
	s.commitProvider()
	if len(s.draft.cfg.Ai.Providers) != 1 {
		t.Error("a service was duplicated")
	}
}

func TestAddProviderModel_inheritsAddressAndKey(t *testing.T) {
	s := providerApp(t)
	s.draft.cfg.Ai.Providers = []config.AiProvider{{
		ID: "p1", Name: "One", BaseURL: "https://one.test/v1",
		ApiKeySource: config.KeyEnv, PlainKey: "MY_KEY",
	}}
	s.addProviderModel("p1", "gpt-x")
	if len(s.draft.cfg.Ai.Models) != 1 {
		t.Fatalf("models = %+v", s.draft.cfg.Ai.Models)
	}
	m := s.draft.cfg.Ai.Models[0]
	if m.ID != "gpt-x" || m.ProviderID != "p1" ||
		m.BaseURL != "https://one.test/v1" ||
		m.ApiKeySource != config.KeyEnv || !m.Enabled {
		t.Errorf("model = %+v", m)
	}
	s.addProviderModel("p1", "gpt-x")
	if len(s.draft.cfg.Ai.Models) != 1 {
		t.Error("a model was added twice")
	}
}

func TestForgetProvider_keepsItsModels(t *testing.T) {
	s := providerApp(t)
	s.draft.cfg.Ai.Providers = []config.AiProvider{{ID: "p1"}}
	s.draft.cfg.Ai.Models = []config.ModelEntry{
		{ID: "a", ProviderID: "p1"},
	}
	s.forgetProvider("p1")
	if len(s.draft.cfg.Ai.Providers) != 0 ||
		len(s.draft.cfg.Ai.Models) != 1 ||
		s.draft.cfg.Ai.Models[0].ProviderID != "" {
		t.Fatalf("provider = %+v models = %+v",
			s.draft.cfg.Ai.Providers, s.draft.cfg.Ai.Models)
	}
}

func TestAiReply_appendToDailyNote(t *testing.T) {
	s := providerApp(t)
	s.config.Ai.Permission = config.PermissionAppend
	s.aiReply("daily", "an answer from the model")
	rel, err := s.dailyRelPath(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	full, ok := s.vaultPath(rel)
	if !ok {
		t.Fatal("the daily note path is outside the vault")
	}
	data, err := os.ReadFile(full)
	if err != nil {
		t.Fatalf("the daily note was not made: %v", err)
	}
	if !strings.Contains(string(data), "an answer from the model") {
		t.Errorf("daily note = %q", data)
	}
}

func TestAppendToFile_neverOverwrites(t *testing.T) {
	s := providerApp(t)
	full := filepath.Join(s.root, "a.md")
	if err := os.WriteFile(full, []byte("first"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.appendToFile(full, "second"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(full)
	if string(data) != "first\n\nsecond\n" {
		t.Errorf("file = %q", data)
	}
}
