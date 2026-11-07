package view

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/skill"
)

func strPtr(s string) *string { return &s }

func TestAskModel_sceneRoutes(t *testing.T) {
	w, s, _ := workspaceWithModels(t, "chat-model", "action-model",
		"skill-model", "agent-model")
	for i := range w.Config.Ai.Models {
		w.Config.Ai.Models[i].ToolsEnabled = true
	}
	w.Config.Ai.Route = config.AiRouteConfig{
		Chat:   strPtr("chat-model"),
		Action: strPtr("action-model"),
		Skill:  strPtr("skill-model"),
		Agent:  strPtr("agent-model"),
	}
	if got := w.askModel(s, config.RouteKindChat, nil); got != "agent-model" {
		t.Errorf("a chat with tools = %q, want the agent route", got)
	}
	w.Config.Ai.Agentic = false
	if got := w.askModel(s, config.RouteKindChat, nil); got != "chat-model" {
		t.Errorf("a plain chat = %q, want the chat route", got)
	}
	s.Skill = &skill.Skill{Name: "demo"}
	if got := w.askModel(s, config.RouteKindChat, nil); got != "skill-model" {
		t.Errorf("a chat with a skill = %q, want the skill route", got)
	}
	s.Skill = nil
	if got := w.askModel(s, config.RouteKindAction,
		&config.AiAction{}); got != "action-model" {
		t.Errorf("an action = %q, want the action route", got)
	}
}

func TestExpandActionPrompt_fillsVariables(t *testing.T) {
	w, _, _ := workspaceWithModels(t, "m1")
	got := w.expandActionPrompt(
		"Title {{title}} / tags {{known_tags}} / {{selection}} / "+
			"{{date:YYYY}} / {{unknown}}",
		w.Tabs.ActiveNote(), "the piece")
	for _, gone := range []string{
		"{{title}}", "{{known_tags}}", "{{selection}}", "{{date:YYYY}}",
	} {
		if strings.Contains(got, gone) {
			t.Errorf("%s was not expanded:\n%s", gone, got)
		}
	}
	for _, want := range []string{
		"the piece", "{{unknown}}", strconv.Itoa(time.Now().Year()),
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expansion is missing %q:\n%s", want, got)
		}
	}
}

func TestExpandActionPrompt_badFormatKeepsThePrompt(t *testing.T) {
	w, _, _ := workspaceWithModels(t, "m1")
	in := "Polish {{date:YYYY-Do}}"
	if got := w.expandActionPrompt(in,
		w.Tabs.ActiveNote(), "x"); got != in {
		t.Errorf("a bad format changed the prompt: %q", got)
	}
}

func TestUsageCount(t *testing.T) {
	n := uint64(42)
	if usageCount(nil) != "-" || usageCount(&n) != "42" {
		t.Error("a cost was not shown plainly")
	}
}

func TestContextMaxChars_modelLimitWins(t *testing.T) {
	w, _, _ := workspaceWithModels(t, "m1")
	w.Config.Ai.MaxContextChars = 12_000
	w.AiModel = "m1"
	limit := uint32(2_000)
	for i := range w.Config.Ai.Models {
		w.Config.Ai.Models[i].ContextLimit = &limit
	}
	if got := w.contextMaxChars(); got != 4_000 {
		t.Errorf("context cap = %d, want 4000", got)
	}
}

func TestAiHear_recordsUsage(t *testing.T) {
	w := NewWorkspace(config.Default())
	s := &aiSession{Blocks: []aiBlock{{Role: ai.Assistant}}}
	events := make(chan ai.Event, 3)
	prompt := uint64(12)
	events <- ai.Delta("hi")
	events <- ai.Used(ai.Usage{PromptTokens: &prompt})
	events <- ai.Done()
	close(events)
	w.aiHear(s, 0, events, func() {})
	if s.Blocks[0].Usage == nil ||
		s.Blocks[0].Usage.PromptTokens == nil ||
		*s.Blocks[0].Usage.PromptTokens != 12 {
		t.Fatalf("usage = %+v", s.Blocks[0].Usage)
	}
}
