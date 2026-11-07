package view

import (
	"strconv"
	"strings"
	"time"

	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/templating"
)

// askModel is which model this question goes to. The four scene routes
// are all wired here: a skill in use sends a chat to the Skill route,
// and a chat that will use the read-only tool loop may go to the Agent
// route. Actions name their own route and may name their own model.
func (w *Workspace) askModel(
	s *aiSession, kind string, action *config.AiAction,
) string {
	cfg := w.Config.Ai
	if kind == config.RouteKindChat && s.Skill != nil {
		kind = config.RouteKindSkill
	}
	model := cfg.ResolveModel(kind, action, w.AiModel)
	if kind == config.RouteKindChat && cfg.Agentic &&
		w.modelHasTools(model) {
		if agent := cfg.ResolveModel(config.RouteKindAgent, nil,
			model); agent != "" {
			model = agent
		}
	}
	return model
}

// modelHasTools is whether a model has been found to call the read-only
// tools, so an agent route only applies to a request that really loops.
func (w *Workspace) modelHasTools(id string) bool {
	for _, m := range w.Config.Ai.Models {
		if m.ID == id && m.Enabled {
			return m.ToolsEnabled
		}
	}
	return false
}

// expandActionPrompt fills the variables an action may name. Unknown
// variables are left alone, so a prompt for another host is not mangled;
// a bad inline date format keeps the prompt as it was rather than
// sending half of it to the model.
func (w *Workspace) expandActionPrompt(
	prompt string, n *OpenNote, piece string,
) string {
	whole := ""
	if n != nil && n.Editor != nil {
		whole = n.Editor.Buffer.String()
	}
	extras := map[string]string{
		"title":       w.ActiveBasename(),
		"note":        whole,
		"selection":   piece,
		"known_tags":  w.knownTags(),
		"model_count": strconv.Itoa(len(w.Config.Ai.Models)),
	}
	for k, v := range w.blankExtras() {
		extras[k] = v
	}
	out, err := templating.Expand(prompt, templating.Vars{
		Extras: extras, Now: time.Now(),
	})
	if err != nil {
		return prompt
	}
	return out
}

// knownTags is the vault's tags as a short comma-separated list, for an
// action that wants to suggest tags the vault already uses.
func (w *Workspace) knownTags() string {
	if w.Index == nil {
		return ""
	}
	tags := w.Index.Tags()
	names := make([]string, 0, len(tags))
	for _, one := range tags {
		names = append(names, one.Name)
	}
	return strings.Join(names, ", ")
}

// contextMaxChars is how much of the note the model is shown. The global
// cap wins unless the chosen model declares a smaller context window;
// tokens are counted conservatively as two characters each, which is
// about right for Chinese prose and safe for English.
func (w *Workspace) contextMaxChars() int {
	max := int(w.Config.Ai.MaxContextChars)
	for _, m := range w.Config.Ai.Models {
		if m.ID != w.AiModel || m.ContextLimit == nil {
			continue
		}
		chars := int(*m.ContextLimit) * 2
		if chars > 0 && chars < max {
			return chars
		}
	}
	return max
}
