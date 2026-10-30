package config

import "strings"

// Model kinds, and where a model's key comes from.
const (
	ModelOllama            = "ollama"
	ModelOpenAICompatible  = "open_ai_compatible"
	KeyKeychain            = "keychain"
	KeyEnv                 = "env"
	KeyPlain               = "plain"
	OutputAuto             = "auto"
	OutputZh               = "zh"
	OutputEn               = "en"
	ActionScopeSelection   = "selection"
	ActionScopeWhole       = "whole"
	ActionApplyReplace     = "replace"
	ActionApplyInsert      = "insert"
	maxActions             = 32
	maxActionNameLen       = 32
	maxActionPromptLen     = 2000
	actionPolish           = "polish"
	actionCondense         = "condense"
	actionTranslate        = "translate"
	actionRetitle          = "retitle"
	actionSummarize        = "summarize"
	actionTag              = "tag"
	defaultNumCtx          = 8192
	defaultKeepAlive       = "5m"
	defaultNumPredict      = 8192
	defaultTemperature     = 0.7
	defaultOllamaBaseURL   = "http://localhost:11434"
	defaultOllamaKeepAlive = "5m"
)

// AiConfig is everything the AI features use.
type AiConfig struct {
	Enabled              bool          `json:"enabled"`
	DefaultModel         string        `json:"default_model"`
	Models               []ModelEntry  `json:"models"`
	Ollama               OllamaConfig  `json:"ollama"`
	TimeoutSecs          uint64        `json:"timeout_secs"`
	MaxContextChars      uint64        `json:"max_context_chars"`
	MaxSteps             uint64        `json:"max_steps"`
	Agentic              bool          `json:"agentic"`
	MaxToolChars         uint64        `json:"max_tool_chars"`
	OutputLanguage       string        `json:"output_language"`
	AutoApply            bool          `json:"auto_apply"`
	SkillsEnabled        bool          `json:"skills_enabled"`
	DisabledSkills       []string      `json:"disabled_skills"`
	MemoryEnabled        bool          `json:"memory_enabled"`
	Actions              []AiAction    `json:"actions"`
	Route                AiRouteConfig `json:"route"`
	FallbackModel        string        `json:"fallback_model"`
	Retry                AiRetryConfig `json:"retry"`
	FirstByteTimeoutSecs uint64        `json:"first_byte_timeout_secs"`
}

// OllamaConfig is the local Ollama server.
type OllamaConfig struct {
	BaseURL   string `json:"base_url"`
	KeepAlive string `json:"keep_alive"`
	NumCtx    uint32 `json:"num_ctx"`
}

// AiRetryConfig is what a failed request does next.
type AiRetryConfig struct {
	MaxAttempts uint8  `json:"max_attempts"`
	BackoffMs   uint64 `json:"backoff_ms"`
}

// AiRouteConfig sends each kind of request to a model of its own, when set.
type AiRouteConfig struct {
	Chat   *string `json:"chat"`
	Action *string `json:"action"`
	Skill  *string `json:"skill"`
	Agent  *string `json:"agent"`
}

// ModelEntry is one model the user can pick.
type ModelEntry struct {
	ID                 string        `json:"id"`
	Kind               string        `json:"kind"`
	Label              string        `json:"label"`
	BaseURL            string        `json:"base_url"`
	ApiKeySource       string        `json:"api_key_source"`
	PlainKey           string        `json:"plain_key"`
	Params             AiModelParams `json:"params"`
	ContextLimit       *uint32       `json:"context_limit"`
	Capabilities       []string      `json:"capabilities"`
	CapabilitiesPinned bool          `json:"capabilities_pinned"`
	ToolsEnabled       bool          `json:"tools_enabled"`
	Enabled            bool          `json:"enabled"`
	LastCheckedMs      uint64        `json:"last_checked_ms"`
}

// AiModelParams is what a request to a model asks for.
type AiModelParams struct {
	NumCtx      *uint32 `json:"num_ctx,omitempty"`
	NumPredict  uint32  `json:"num_predict"`
	Temperature float32 `json:"temperature"`
	Think       bool    `json:"think"`
	KeepAlive   *string `json:"keep_alive,omitempty"`
}

// AiAction is a shortcut the user runs on a note or a selection.
type AiAction struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Prompt string  `json:"prompt"`
	Scope  string  `json:"scope"`
	Apply  string  `json:"apply"`
	Enable bool    `json:"enabled"`
	Model  *string `json:"model,omitempty"`
}

// DefaultAi returns the AI settings of a first start: disabled, with the
// built-in actions and the local Ollama server.
func DefaultAi() AiConfig {
	return AiConfig{
		Enabled:              false,
		Models:               []ModelEntry{},
		Ollama:               OllamaConfig{BaseURL: defaultOllamaBaseURL, KeepAlive: defaultOllamaKeepAlive, NumCtx: defaultNumCtx},
		TimeoutSecs:          60,
		MaxContextChars:      12_000,
		MaxSteps:             8,
		Agentic:              true,
		MaxToolChars:         4_000,
		OutputLanguage:       OutputAuto,
		AutoApply:            false,
		SkillsEnabled:        true,
		DisabledSkills:       []string{},
		MemoryEnabled:        false,
		Actions:              DefaultActions(),
		Retry:                AiRetryConfig{MaxAttempts: 3, BackoffMs: 500},
		FirstByteTimeoutSecs: 30,
	}
}

// DefaultActions returns the actions the app ships with.
func DefaultActions() []AiAction {
	builtin := []struct {
		id     string
		name   string
		prompt string
	}{
		{actionPolish, "Polish", "Polish the writing: fix grammar and " +
			"awkward phrasing, keep the meaning and the voice."},
		{actionCondense, "Condense", "Condense this: say the same in " +
			"fewer words, keep every fact."},
		{actionTranslate, "Translate", "Translate this into English, " +
			"keeping the structure and the formatting."},
		{actionRetitle, "Retitle", "Suggest a short title for this note, " +
			"and only the title."},
		{actionSummarize, "Summarize", "Summarize this in a few lines."},
		{actionTag, "Tag", "Suggest tags for this note, as a list of " +
			"#tags."},
	}
	actions := make([]AiAction, 0, len(builtin))
	for _, b := range builtin {
		actions = append(actions, AiAction{
			ID:     b.id,
			Name:   b.name,
			Prompt: b.prompt,
			Scope:  ActionScopeSelection,
			Apply:  ActionApplyReplace,
			Enable: true,
		})
	}
	return actions
}

func (o OllamaConfig) isEmpty() bool {
	return o.BaseURL == "" && o.KeepAlive == "" && o.NumCtx == 0
}

func (r AiRetryConfig) isEmpty() bool {
	return r.MaxAttempts == 0 && r.BackoffMs == 0
}

func (a *AiConfig) fillMissing(d AiConfig) {
	if a.Ollama.isEmpty() {
		a.Ollama = d.Ollama
	}
	if a.TimeoutSecs == 0 {
		a.TimeoutSecs = d.TimeoutSecs
	}
	if a.MaxContextChars == 0 {
		a.MaxContextChars = d.MaxContextChars
	}
	if a.MaxSteps == 0 {
		a.MaxSteps = d.MaxSteps
	}
	if a.MaxToolChars == 0 {
		a.MaxToolChars = d.MaxToolChars
	}
	if a.OutputLanguage == "" {
		a.OutputLanguage = d.OutputLanguage
	}
	if a.DisabledSkills == nil {
		a.DisabledSkills = []string{}
	}
	if a.Actions == nil {
		a.Actions = DefaultActions()
	}
	if a.Retry.isEmpty() {
		a.Retry = d.Retry
	}
	if a.FirstByteTimeoutSecs == 0 {
		a.FirstByteTimeoutSecs = d.FirstByteTimeoutSecs
	}
	if a.Models == nil {
		a.Models = []ModelEntry{}
	}
}

// Sanitize clamps the AI settings, drops the models and actions that cannot
// be used, and clears the routes that point at nothing.
func (a *AiConfig) Sanitize() {
	a.TimeoutSecs = clampU64(a.TimeoutSecs, 5, 600)
	a.MaxContextChars = clampU64(a.MaxContextChars, 1_000, 200_000)
	a.MaxSteps = clampU64(a.MaxSteps, 1, 64)
	a.MaxToolChars = clampU64(a.MaxToolChars, 500, 20_000)
	a.FirstByteTimeoutSecs = clampU64(a.FirstByteTimeoutSecs, 5, 600)
	a.Retry.MaxAttempts = uint8(clampU64(uint64(a.Retry.MaxAttempts), 1, 5))
	a.Retry.BackoffMs = clampU64(a.Retry.BackoffMs, 100, 10_000)
	if !oneOf(a.OutputLanguage, OutputAuto, OutputZh, OutputEn) {
		a.OutputLanguage = OutputAuto
	}
	// Auto-applying an AI's edit without showing it is never on.
	a.AutoApply = false
	a.Ollama.NumCtx = clampU32(a.Ollama.NumCtx, 512, 1_048_576)

	models := a.Models[:0:0]
	known := map[string]bool{}
	for i := range a.Models {
		m := &a.Models[i]
		if strings.TrimSpace(m.ID) == "" || known[m.ID] {
			continue
		}
		if !oneOf(m.Kind, ModelOllama, ModelOpenAICompatible) {
			m.Kind = ModelOllama
		}
		if !oneOf(m.ApiKeySource, KeyKeychain, KeyEnv, KeyPlain) {
			m.ApiKeySource = KeyKeychain
		}
		m.sanitizeParams(a.Ollama.NumCtx)
		known[m.ID] = true
		models = append(models, *m)
	}
	a.Models = models

	a.sanitizeActions()

	// A model that is gone, or turned off, is not a default or a route.
	usable := func(id string) bool {
		if id == "" {
			return false
		}
		for i := range a.Models {
			if a.Models[i].ID == id && a.Models[i].Enabled {
				return true
			}
		}
		return false
	}
	if !usable(a.DefaultModel) {
		a.DefaultModel = ""
	}
	if !usable(a.FallbackModel) {
		a.FallbackModel = ""
	}
	r := &a.Route
	if r.Chat != nil && !usable(*r.Chat) {
		r.Chat = nil
	}
	if r.Action != nil && !usable(*r.Action) {
		r.Action = nil
	}
	if r.Skill != nil && !usable(*r.Skill) {
		r.Skill = nil
	}
	if r.Agent != nil && !usable(*r.Agent) {
		r.Agent = nil
	}
}

func (m *ModelEntry) sanitizeParams(ollamaCtx uint32) {
	p := &m.Params
	p.NumPredict = clampU32(p.NumPredict, 16, 32_768)
	if p.Temperature < 0 || p.Temperature > 2 || p.Temperature != p.Temperature {
		p.Temperature = defaultTemperature
	}
	if p.NumCtx != nil {
		// The context cannot be larger than the model allows.
		hi := uint32(1_048_576)
		if m.ContextLimit != nil && *m.ContextLimit < hi {
			hi = *m.ContextLimit
		}
		v := clampU32(*p.NumCtx, 512, hi)
		// The value the default would have is the same as not setting it.
		if v == defaultNumCtx && (m.ContextLimit == nil || ollamaCtx == defaultNumCtx) {
			p.NumCtx = nil
		} else {
			p.NumCtx = &v
		}
	}
	if p.KeepAlive != nil && *p.KeepAlive == defaultKeepAlive {
		p.KeepAlive = nil
	}
}

func (a *AiConfig) sanitizeActions() {
	actions := make([]AiAction, 0, len(a.Actions))
	seen := map[string]bool{}
	for i := range a.Actions {
		act := a.Actions[i]
		act.ID = strings.TrimSpace(act.ID)
		if act.ID == "" || seen[act.ID] {
			continue
		}
		act.Name = truncate(strings.TrimSpace(act.Name), maxActionNameLen)
		if act.Name == "" {
			act.Name = act.ID
		}
		act.Prompt = truncate(strings.TrimSpace(act.Prompt), maxActionPromptLen)
		if !oneOf(act.Scope, ActionScopeSelection, ActionScopeWhole) {
			act.Scope = ActionScopeSelection
		}
		if !oneOf(act.Apply, ActionApplyReplace, ActionApplyInsert) {
			act.Apply = ActionApplyReplace
		}
		seen[act.ID] = true
		actions = append(actions, act)
	}
	if len(actions) > maxActions {
		actions = actions[:maxActions]
	}
	a.Actions = actions
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
