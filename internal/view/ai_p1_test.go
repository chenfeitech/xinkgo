package view

import (
	"strings"
	"testing"

	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
)

func TestSyncContexts_removedContextIsNotSent(t *testing.T) {
	w, s, _ := workspaceWithModels(t, "m1")
	n := w.Tabs.ActiveNote()
	whole := n.Editor.Buffer.String()
	end := 4
	if len(whole) < end {
		end = len(whole)
	}
	n.Editor.Select(0, end)
	w.syncContexts(s, n)
	if len(s.Contexts) != 2 {
		t.Fatalf("contexts = %+v, want the note and the selection",
			s.Contexts)
	}
	brief := w.aiBrief(s, n)
	if !strings.Contains(brief, "The selected text is") {
		t.Errorf("the selection was not sent:\n%s", brief)
	}
	s.Contexts[0].Removed = true
	brief = w.aiBrief(s, n)
	if strings.Contains(brief, "The note being edited is") {
		t.Error("a removed context was still sent")
	}
}

func TestModelLines_recentThenGroups(t *testing.T) {
	w := NewWorkspace(config.Default())
	w.Config.Ai.Models = []config.ModelEntry{
		{ID: "local", Source: config.ModelSourceOllama, Enabled: true},
		{ID: "cloud-a", ProviderID: "p1", Enabled: true},
		{ID: "hand", Enabled: true},
	}
	w.Config.Ai.Providers = []config.AiProvider{
		{ID: "p1", Name: "One Service"},
	}
	w.RecentModels = []string{"cloud-a"}
	lines := w.modelLines("")
	if len(lines) != 3 {
		t.Fatalf("lines = %+v", lines)
	}
	if lines[0].ID != "cloud-a" ||
		lines[0].Group != locale.AiModelRecent() {
		t.Errorf("recent model is not first: %+v", lines[0])
	}
	for _, one := range lines[1:] {
		if one.Group == locale.AiModelRecent() {
			t.Errorf("a recent group was repeated: %+v", one)
		}
	}
	if got := w.modelLines("local"); len(got) != 1 ||
		got[0].ID != "local" {
		t.Errorf("query = %+v", got)
	}
	if got := w.modelLines("cloud-a"); len(got) != 1 {
		t.Errorf("an unknown ID was not filterable: %+v", got)
	}
}

func TestPickModel_setsHandPicked(t *testing.T) {
	w, _, _ := workspaceWithModels(t, "m1", "m2")
	w.pickModel("m2")
	if w.AiModel != "m2" || !w.aiPicked {
		t.Errorf("picked = %q / %v", w.AiModel, w.aiPicked)
	}
	w.pickModel("not-there")
	if w.AiModel != "m2" {
		t.Error("an unknown model replaced the picked one")
	}
}

func TestQuickRun_blanksWaitForTheForm(t *testing.T) {
	w, s, asked := workspaceWithModels(t, "m1")
	w.Actions = []config.AiAction{{
		ID: "x", Name: "X", Prompt: "Use {{voice}} please",
		Enable: true, Scope: config.ActionScopeSelection,
	}}
	w.quickRun(w.Tabs.ActiveNote(), s, w.Actions[0], "")
	if !w.aiP1.blankOpen {
		t.Fatal("a prompt with blanks ran without its form")
	}
	if *asked != "" {
		t.Error("a prompt with blanks was sent anyway")
	}
	w.runBlankAction(w.Tabs.ActiveNote(), s)
	// runAction expands the blank into an empty answer and asks.
	if *asked == "" {
		t.Error("the form never sent the prompt")
	}
}

func TestQuickWords_pinnedFirst(t *testing.T) {
	w := NewWorkspace(config.Default())
	w.Actions = []config.AiAction{
		{ID: "a", Name: "A", Prompt: "a"},
		{ID: "b", Name: "B", Prompt: "b", Pinned: true},
	}
	got := w.quickWords("")
	if len(got) != 2 || got[0].ID != "b" {
		t.Errorf("quick words = %+v, want the pinned one first", got)
	}
}

func TestAiReviewDone_replacementReviewsOnlyTheSelection(t *testing.T) {
	w, s, _ := workspaceWithModels(t, "m1")
	n := w.Tabs.ActiveNote()
	whole := n.Editor.Buffer.String()
	end := 4
	if len(whole) < end {
		end = len(whole)
	}
	n.Editor.Select(0, end)
	w.syncContexts(s, n)
	s.selectionFrom, s.selectionTo = 0, end
	s.Blocks = []aiBlock{{
		Role: ai.Assistant,
		Text: "Here is why.\n<replacement>new words</replacement>",
		Over: true,
	}}
	w.aiReviewDone()
	if !w.review.open || !w.review.part || w.review.from != 0 ||
		w.review.to != end {
		t.Fatalf("review = %+v, want a piece review of 0..%d",
			w.review, end)
	}
}
