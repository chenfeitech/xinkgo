package config

import (
	"fmt"
	"hash/fnv"
	"net/url"
	"strings"
)

// ProviderPreset is one service the user may add without typing its
// address. Names are the brands themselves, so they read the same in
// every language; the group labels are said by the catalogs.
type ProviderPreset struct {
	ID      string
	Name    string
	BaseURL string
	Group   string
}

// The groups a preset may stand in, in the order the chooser shows them.
const (
	ProviderGroupOverseas = "overseas"
	ProviderGroupCN       = "cn"
	ProviderGroupRelay    = "relay"
	ProviderGroupCustom   = "custom"
)

// ProviderPresets are the well-known OpenAI-compatible endpoints. Ollama
// is deliberately absent: it has its own settings group, its own
// discovery and its own num_ctx / keep_alive parameters.
var ProviderPresets = []ProviderPreset{
	{"openai", "OpenAI", "https://api.openai.com/v1",
		ProviderGroupOverseas},
	{"anthropic", "Anthropic", "https://api.anthropic.com/v1",
		ProviderGroupOverseas},
	{"gemini", "Google Gemini",
		"https://generativelanguage.googleapis.com/v1beta",
		ProviderGroupOverseas},
	{"deepseek", "DeepSeek", "https://api.deepseek.com/v1",
		ProviderGroupCN},
	{"moonshot", "Kimi / Moonshot", "https://api.moonshot.cn/v1",
		ProviderGroupCN},
	{"qwen", "Qwen / DashScope",
		"https://dashscope.aliyuncs.com/compatible-mode/v1",
		ProviderGroupCN},
	{"siliconflow", "SiliconFlow", "https://api.siliconflow.cn/v1",
		ProviderGroupRelay},
	{"openrouter", "OpenRouter", "https://openrouter.ai/api/v1",
		ProviderGroupRelay},
}

// ProviderIDFor is the stable ID of a service at that address. An older
// config file has no providers, so several models of one address are
// gathered under the same ID when the file is read.
func ProviderIDFor(baseURL string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.TrimSpace(baseURL)))
	return fmt.Sprintf("p-%x", h.Sum32())
}

// providerNameOf is what a migrated provider is called: the preset's
// name when the address is known, otherwise its host.
func providerNameOf(baseURL string) string {
	for _, one := range ProviderPresets {
		if one.BaseURL == baseURL {
			return one.Name
		}
	}
	if parsed, err := url.Parse(baseURL); err == nil && parsed.Host != "" {
		return parsed.Host
	}
	return "Custom endpoint"
}

// migrateProviders gives an older config file a provider for each
// distinct address its hand-written models use, and points those models
// at it. A file that already has providers is left alone apart from
// filling in any model that lost its ProviderID.
func (a *AiConfig) migrateProviders() {
	byAddress := map[string]*AiProvider{}
	for i := range a.Providers {
		byAddress[a.Providers[i].BaseURL] = &a.Providers[i]
	}
	for i := range a.Models {
		m := &a.Models[i]
		if m.FromOllama() || strings.TrimSpace(m.BaseURL) == "" {
			continue
		}
		if m.ProviderID == "" {
			p, ok := byAddress[m.BaseURL]
			if !ok {
				id := ProviderIDFor(m.BaseURL)
				a.Providers = append(a.Providers, AiProvider{
					ID: id, Name: providerNameOf(m.BaseURL),
					BaseURL: m.BaseURL, ApiKeySource: m.ApiKeySource,
					PlainKey: m.PlainKey, KeyRef: m.KeyRef,
					ShowInPicker: true,
				})
				p = &a.Providers[len(a.Providers)-1]
				byAddress[m.BaseURL] = p
			}
			m.ProviderID = p.ID
		}
	}
	for i := range a.Providers {
		if strings.TrimSpace(a.Providers[i].ID) == "" {
			a.Providers[i].ID = ProviderIDFor(a.Providers[i].BaseURL)
		}
	}
}

// ActionBlanks are the variables of an action's prompt that the user
// must fill before it runs. The built-in ones are filled by the app, so
// they are not blanks; `{{name|default}}` keeps only the name.
func ActionBlanks(prompt string) []string {
	seen := map[string]bool{}
	out := []string{}
	text := prompt
	for {
		start := strings.Index(text, "{{")
		if start < 0 {
			return out
		}
		rest := text[start+2:]
		end := strings.Index(rest, "}}")
		if end < 0 {
			return out
		}
		name := rest[:end]
		if at := strings.Index(name, "|"); at >= 0 {
			name = name[:at]
		}
		name = strings.TrimSpace(name)
		if name != "" && !builtinActionVar(name) && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
		text = rest[end+2:]
	}
}

func builtinActionVar(name string) bool {
	switch name {
	case "title", "note", "selection", "known_tags", "model_count",
		"date", "time":
		return true
	}
	return strings.HasPrefix(name, "date:") ||
		strings.HasPrefix(name, "time:")
}
