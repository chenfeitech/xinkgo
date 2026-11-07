package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateProviders_oldFileIsGrouped(t *testing.T) {
	c := DefaultAi()
	c.Models = []ModelEntry{
		{ID: "a", BaseURL: "https://one.test/v1", Enabled: true,
			ApiKeySource: KeyPlain},
		{ID: "b", BaseURL: "https://one.test/v1", Enabled: true,
			ApiKeySource: KeyPlain},
		{ID: "c", BaseURL: "https://two.test/v1", Enabled: true,
			ApiKeySource: KeyEnv},
		{ID: "local", Source: ModelSourceOllama, Enabled: true},
	}
	c.Sanitize()
	if len(c.Providers) != 2 {
		t.Fatalf("providers = %+v, want two addresses", c.Providers)
	}
	if c.Models[0].ProviderID != c.Models[1].ProviderID {
		t.Error("two models of one address were not grouped")
	}
	if c.Models[0].ProviderID == c.Models[2].ProviderID {
		t.Error("different addresses were grouped together")
	}
	if c.Models[3].ProviderID != "" {
		t.Error("an Ollama model was given a provider")
	}
	// Running again must not add duplicate providers.
	c.Sanitize()
	if len(c.Providers) != 2 {
		t.Errorf("a second sanitize duplicated providers: %d",
			len(c.Providers))
	}
}

func TestSanitizeProviders_uniqueIDsAndNames(t *testing.T) {
	c := DefaultAi()
	c.Providers = []AiProvider{
		{ID: "p1", BaseURL: "https://one.test/v1"},
		{ID: "p1", BaseURL: "https://two.test/v1"},
		{ID: "", BaseURL: "https://three.test/v1"},
	}
	c.Sanitize()
	if len(c.Providers) != 2 {
		t.Fatalf("providers = %+v", c.Providers)
	}
	for _, p := range c.Providers {
		if p.ID == "" || p.Name == "" {
			t.Errorf("an unnamed provider was kept: %+v", p)
		}
	}
}

func TestProviderIDFor_stable(t *testing.T) {
	a := ProviderIDFor("https://one.test/v1")
	b := ProviderIDFor("https://one.test/v1")
	if a != b || a == ProviderIDFor("https://two.test/v1") {
		t.Error("a provider ID is not stable per address")
	}
}

func TestActionBlanks(t *testing.T) {
	got := ActionBlanks(
		"{{title}} {{known_tags}} {{voice|formal}} {{date}} " +
			"{{date:YYYY}} {{voice}}")
	if len(got) != 1 || got[0] != "voice" {
		t.Errorf("blanks = %+v, want only voice", got)
	}
}

// A config.json written before providers existed must still load, with
// its models grouped into providers rather than losing them.
func TestLoad_oldAiConfigMigratesProviders(t *testing.T) {
	dir := t.TempDir()
	data := `{"ai":{"enabled":true,"models":[` +
		`{"id":"a","base_url":"https://one.test/v1",` +
		`"api_key_source":"plain","enabled":true},` +
		`{"id":"b","base_url":"https://one.test/v1",` +
		`"api_key_source":"plain","enabled":true}]}}`
	if err := os.WriteFile(filepath.Join(dir, FileName),
		[]byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Ai.Models) != 2 || len(cfg.Ai.Providers) != 1 {
		t.Fatalf("models = %d providers = %d", len(cfg.Ai.Models),
			len(cfg.Ai.Providers))
	}
	if cfg.Ai.Models[0].ProviderID == "" {
		t.Error("a model was not pointed at its provider")
	}
}
