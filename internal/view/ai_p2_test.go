package view

import (
	"context"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
)

func TestActivityOf_namesTheVerbAndTarget(t *testing.T) {
	w := NewWorkspace(config.Default())
	one := w.activityOf(ai.ToolCall{Name: ai.ToolReadNote,
		Arguments: `{"path":"notes/a.md"}`})
	if one.Verb != "read" || one.Target != "notes/a.md" {
		t.Errorf("activity = %+v", one)
	}
	one = w.activityOf(ai.ToolCall{Name: ai.ToolSearchNotes,
		Arguments: `{"query":"hello"}`})
	if one.Verb != "search" || one.Target != "hello" {
		t.Errorf("search activity = %+v", one)
	}
}

func TestAnswerQuestion_startsANewTurn(t *testing.T) {
	w, s, _ := workspaceWithModels(t, "m1")
	w.OnAsk = func(_ context.Context,
		_ []ai.Message, model string) (<-chan ai.Event, error) {
		out := make(chan ai.Event, 1)
		out <- ai.Done()
		close(out)
		return out, nil
	}
	s.PendingQuestion = &ai.Question{Text: "Which tone?"}
	w.answerQuestion(s, w.Tabs.ActiveNote(), "Formal")
	if s.PendingQuestion != nil {
		t.Error("the card stayed up after an answer")
	}
	blocks := s.blocks()
	if len(blocks) < 2 || blocks[0].Text != "Formal" {
		t.Errorf("blocks = %+v", blocks)
	}
}

func TestAnswerQuestion_skipEndsQuietly(t *testing.T) {
	w, s, asked := workspaceWithModels(t, "m1")
	s.PendingQuestion = &ai.Question{Text: "Which tone?"}
	w.answerQuestion(s, w.Tabs.ActiveNote(), "")
	if s.PendingQuestion != nil || *asked != "" {
		t.Error("a skipped question started a turn")
	}
}

func TestAnswerQuestion_secretIsNotStored(t *testing.T) {
	w, s, _ := workspaceWithModels(t, "m1")
	s.PendingQuestion = &ai.Question{Text: "Key?", Secret: true}
	w.answerQuestion(s, w.Tabs.ActiveNote(), "s3cret")
	blocks := s.blocks()
	if !blocks[0].Secret {
		t.Error("the secret answer was not marked")
	}
	stored := w.aiStored(blocks)
	if len(stored) != 0 {
		t.Errorf("a secret answer was written down: %+v", stored)
	}
}

func TestEditUserBlock_truncatesLaterTurns(t *testing.T) {
	w, s, _ := workspaceWithModels(t, "m1")
	s.Blocks = []aiBlock{
		{Role: ai.User, Text: "first", Over: true},
		{Role: ai.Assistant, Text: "answer", Over: true},
		{Role: ai.User, Text: "second", Over: true},
	}
	w.editUserBlock(s, 2, "edited")
	blocks := s.blocks()
	if len(blocks) != 2 || s.Input != "edited" || !s.focus {
		t.Fatalf("blocks = %+v input = %q focus = %v", blocks,
			s.Input, s.focus)
	}
}

func TestAttachRef_addsAContext(t *testing.T) {
	w, s, _ := workspaceWithModels(t, "m1")
	s.RefTexts = map[string]string{"b.md": "the attached note"}
	s.Refs = []string{"b.md"}
	n := w.Tabs.ActiveNote()
	w.syncContexts(s, n)
	found := false
	for _, one := range s.Contexts {
		if one.Kind == "ref" && one.Text == "the attached note" {
			found = true
		}
	}
	if !found {
		t.Errorf("contexts = %+v", s.Contexts)
	}
}

func TestUndoRound_checksWithTheApp(t *testing.T) {
	w, _, _ := workspaceWithModels(t, "m1")
	called := false
	w.OnAiUndo = func(path, before, after string) string {
		called = true
		return "done"
	}
	w.LastChange = &roundChange{Path: "a.md", Before: "old",
		After: "new"}
	w.undoRound()
	if !called || w.LastChange != nil {
		t.Error("the undo did not go through the app")
	}
}

func TestQuestionCard_rendersAndSkips(t *testing.T) {
	w, s, _ := workspaceWithModels(t, "m1")
	s.PendingQuestion = &ai.Question{Text: "Which tone?",
		Options: []string{"Formal", "Casual"}}
	tt := ui.NewTester(func(c *ui.Context) {
		w.questionCard(c, s, w.Tabs.ActiveNote())
	}, 420, 220)
	tt.Frame()
	if !tt.HasText("Which tone?") || !tt.HasText("A  Formal") {
		t.Fatal("the question card was not drawn")
	}
	if err := tt.Click(locale.AiQuestionSkip()); err != nil {
		t.Fatal(err)
	}
	if s.PendingQuestion != nil {
		t.Error("skipping left the card up")
	}
}

func TestAiTalk_replyButtonsFollowPermission(t *testing.T) {
	w, s, _ := workspaceWithModels(t, "m1")
	n := w.Tabs.ActiveNote()
	s.Blocks = []aiBlock{{Role: ai.Assistant, Text: "answer",
		Over: true}}
	w.OnAiReply = func(string, string) {}
	tt := ui.NewTester(func(c *ui.Context) {
		w.aiTalk(c, s, n)
	}, 700, 400)
	tt.Frame()
	if tt.HasText(locale.AiReplyDaily()) {
		t.Error("read-only still offers to append to a note")
	}
	w.Config.Ai.Permission = config.PermissionAppend
	tt.Frame()
	if !tt.HasText(locale.AiReplyDaily()) {
		t.Error("append permission did not show the reply actions")
	}
}

func TestTimelineLine_foldsUntilAsked(t *testing.T) {
	w, s, _ := workspaceWithModels(t, "m1")
	b := aiBlock{Activities: []aiActivity{
		{Verb: "read", Target: "a.md"},
		{Verb: "search", Target: "hello"},
	}}
	tt := ui.NewTester(func(c *ui.Context) {
		w.timelineLine(c, b, s)
	}, 500, 200)
	tt.Frame()
	if !tt.HasText(locale.AiTimeline("1", "1")) {
		t.Error("the folded line was not drawn")
	}
	if tt.HasText(locale.AiTimelineStep("read", "a.md")) {
		t.Error("the steps were shown before being asked for")
	}
	if err := tt.Click(locale.AiTimelineShow()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText(locale.AiTimelineStep("read", "a.md")) {
		t.Error("the steps did not unfold")
	}
}

func TestUsageBar_needsAModelLimit(t *testing.T) {
	w, _, _ := workspaceWithModels(t, "m1")
	w.AiModel = "m1"
	b := aiBlock{Over: true}
	if w.contextLimitOf("m1") != 0 {
		t.Fatal("a model without a limit must show no bar")
	}
	limit := uint32(100)
	for i := range w.Config.Ai.Models {
		w.Config.Ai.Models[i].ContextLimit = &limit
	}
	used := uint64(50)
	b.Usage = &ai.Usage{PromptTokens: &used}
	tt := ui.NewTester(func(c *ui.Context) {
		w.usageBar(c, b)
	}, 300, 80)
	tt.Frame()
	if !tt.HasText(locale.AiUsagePct("50")) {
		t.Error("the context bar was not drawn at 50%")
	}
}
