package config

import "testing"

func ptr(s string) *string { return &s }

func modelsForRoute() AiConfig {
	c := DefaultAi()
	c.Models = []ModelEntry{
		{ID: "chat-model", Enabled: true},
		{ID: "action-model", Enabled: true},
		{ID: "skill-model", Enabled: true},
		{ID: "agent-model", Enabled: true, ToolsEnabled: true},
		{ID: "off-model", Enabled: false},
	}
	c.DefaultModel = "chat-model"
	return c
}

func TestResolveModel_defaultAndPicked(t *testing.T) {
	c := modelsForRoute()
	if got := c.ResolveModel(RouteKindChat, nil, ""); got != "chat-model" {
		t.Errorf("default = %q", got)
	}
	if got := c.ResolveModel(RouteKindChat, nil,
		"action-model"); got != "action-model" {
		t.Errorf("picked = %q", got)
	}
}

func TestResolveModel_actionWinsOverRoute(t *testing.T) {
	c := modelsForRoute()
	c.Route.Action = ptr("skill-model")
	a := AiAction{Model: ptr("action-model")}
	if got := c.ResolveModel(RouteKindAction, &a,
		"chat-model"); got != "action-model" {
		t.Errorf("action model = %q, want the action's own", got)
	}
}

func TestResolveModel_fourRoutes(t *testing.T) {
	c := modelsForRoute()
	c.Route = AiRouteConfig{
		Chat:   ptr("chat-model"),
		Action: ptr("action-model"),
		Skill:  ptr("skill-model"),
		Agent:  ptr("agent-model"),
	}
	for kind, want := range map[string]string{
		RouteKindChat:   "chat-model",
		RouteKindAction: "action-model",
		RouteKindSkill:  "skill-model",
		RouteKindAgent:  "agent-model",
	} {
		if got := c.ResolveModel(kind, nil, ""); got != want {
			t.Errorf("route %s = %q, want %q", kind, got, want)
		}
	}
}

func TestResolveModel_disabledRouteIsSkipped(t *testing.T) {
	c := modelsForRoute()
	c.Route.Chat = ptr("off-model")
	c.Route.Action = ptr("off-model")
	if got := c.ResolveModel(RouteKindChat, nil, ""); got != "chat-model" {
		t.Errorf("a disabled route won: %q", got)
	}
	if got := c.ResolveModel(RouteKindChat, nil,
		"off-model"); got != "chat-model" {
		t.Errorf("a disabled picked model won: %q", got)
	}
}

func TestResolveModel_noEnabledModels(t *testing.T) {
	c := modelsForRoute()
	for i := range c.Models {
		c.Models[i].Enabled = false
	}
	if got := c.ResolveModel(RouteKindChat, nil, ""); got != "" {
		t.Errorf("no models = %q, want empty", got)
	}
}

func TestPermission_defaultsToReadOnly(t *testing.T) {
	c := DefaultAi()
	if c.Permission != PermissionRead {
		t.Fatalf("default permission = %q", c.Permission)
	}
	c.Permission = "not-a-level"
	c.Sanitize()
	if c.Permission != PermissionRead {
		t.Errorf("sanitized permission = %q", c.Permission)
	}
}
