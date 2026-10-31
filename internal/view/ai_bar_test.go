package view

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
)

// workspaceWith is a window with one note open in it, whose conversation
// is answered by whatever the caller makes up.
func workspaceWith(
	t *testing.T, ask func([]ai.Message) <-chan ai.Event,
) (*Workspace, *aiSession) {
	t.Helper()
	_, a, _ := vaultWithNotes(t)
	w := NewWorkspace(config.Default())
	if _, err := w.Tabs.Open(a); err != nil {
		t.Fatalf("Open: %v", err)
	}
	w.OnAsk = func(
		context.Context, []ai.Message,
	) (<-chan ai.Event, error) {
		return ask(nil), nil
	}
	s := w.aiOf(a)
	s.Open = true
	return w, s
}

// answered is a window whose answer is the one the test sends, a piece at
// a time, and closes when it is done.
func answered(t *testing.T) (*Workspace, *aiSession, chan ai.Event) {
	t.Helper()
	events := make(chan ai.Event)
	w, s := workspaceWith(t, func([]ai.Message) <-chan ai.Event {
		return events
	})
	return w, s, events
}

// settle waits for the answer that is coming to be over.
func settle(t *testing.T, s *aiSession) {
	t.Helper()
	by := time.Now().Add(3 * time.Second)
	for time.Now().Before(by) {
		if !s.isBusy() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("the answer never came to an end")
}

// TestAiAsk_theQuestionIsWrittenBeforeTheAnswerComes checks that what was
// asked is in the conversation at once, and the answer grows into the
// block after it.
func TestAiAsk_theQuestionIsWrittenBeforeTheAnswerComes(t *testing.T) {
	events := make(chan ai.Event)
	defer close(events)
	w, s := workspaceWith(t, func([]ai.Message) <-chan ai.Event {
		return events
	})
	s.Input = "what is this note about"
	w.aiAsk(w.Tabs.ActiveNote(), s)
	blocks := s.blocks()
	if len(blocks) != 2 {
		t.Fatalf("blocks = %v, want a question and an answer", blocks)
	}
	if blocks[0].Role != ai.User ||
		blocks[0].Text != "what is this note about" {
		t.Fatalf("first block = %+v, want the question", blocks[0])
	}
	if blocks[1].Role != ai.Assistant || blocks[1].Text != "" {
		t.Fatalf("second block = %+v, want an answer yet to come",
			blocks[1])
	}
	if !s.isBusy() {
		t.Fatal("the model is not being waited for")
	}
	if s.Input != "" {
		t.Fatalf("input = %q, want it cleared", s.Input)
	}
}

// TestAiHear_theAnswerGrowsIntoItsOwnBlock checks that the words come
// into the block that waits for them, and that it is said to be over when
// the model is done.
func TestAiHear_theAnswerGrowsIntoItsOwnBlock(t *testing.T) {
	w, s, events := answered(t)
	s.Input = "go on"
	w.aiAsk(w.Tabs.ActiveNote(), s)

	events <- ai.Delta("hel")
	events <- ai.Thinking("let me see")
	events <- ai.Delta("lo")
	events <- ai.Done()
	close(events)
	settle(t, s)

	blocks := s.blocks()
	if len(blocks) != 2 {
		t.Fatalf("blocks = %v", len(blocks))
	}
	if blocks[1].Text != "hello" {
		t.Fatalf("answer = %q, want hello", blocks[1].Text)
	}
	if blocks[1].Thinking != "let me see" {
		t.Fatalf("thinking = %q", blocks[1].Thinking)
	}
	if !blocks[1].Over {
		t.Fatal("the answer was not said to be over")
	}
}

// TestAiHear_anAnswerThatBrokeOffIsSaidToBeCutShort checks that a model
// that ran out of room is known to have given only half an answer.
func TestAiHear_anAnswerThatBrokeOffIsSaidToBeCutShort(t *testing.T) {
	w, s, events := answered(t)
	s.Input = "go on"
	w.aiAsk(w.Tabs.ActiveNote(), s)
	events <- ai.Delta("half an answer")
	events <- ai.Truncated()
	events <- ai.Done()
	close(events)
	settle(t, s)
	if blocks := s.blocks(); !blocks[1].Cut {
		t.Fatalf("block = %+v, want it said to be cut short", blocks[1])
	}
}

// TestAiHear_whatWentWrongIsWrittenOnTheBlock checks that a failure is
// kept where the answer would have been.
func TestAiHear_whatWentWrongIsWrittenOnTheBlock(t *testing.T) {
	w, s, events := answered(t)
	s.Input = "go on"
	w.aiAsk(w.Tabs.ActiveNote(), s)
	events <- ai.Failed(ai.ErrTimeout)
	close(events)
	settle(t, s)

	blocks := s.blocks()
	if blocks[1].Failed == "" {
		t.Fatal("nothing was said of what went wrong")
	}
	if !strings.Contains(blocks[1].Failed, "超时") {
		t.Fatalf("failed = %q, want a sentence about the timeout",
			blocks[1].Failed)
	}
}

// TestAiAsk_aModelThatCannotBeAskedSaysSo checks that a window with no
// one to ask says so, rather than waiting forever.
func TestAiAsk_aModelThatCannotBeAskedSaysSo(t *testing.T) {
	_, a, _ := vaultWithNotes(t)
	w := NewWorkspace(config.Default())
	if _, err := w.Tabs.Open(a); err != nil {
		t.Fatalf("Open: %v", err)
	}
	s := w.aiOf(a)
	s.Input = "hello"
	w.aiAsk(w.Tabs.ActiveNote(), s)
	blocks := s.blocks()
	if len(blocks) != 2 {
		t.Fatalf("blocks = %v", len(blocks))
	}
	if !strings.Contains(blocks[1].Failed, "配置") {
		t.Fatalf("failed = %q, want to be told to configure it",
			blocks[1].Failed)
	}
	if s.isBusy() {
		t.Fatal("the window is waiting for a model it has none of")
	}
}

// TestAiAsk_aModelThatRefusesAtTheDoorSaysSo checks that an error before
// the answer began is written where the answer would have been.
func TestAiAsk_aModelThatRefusesAtTheDoorSaysSo(t *testing.T) {
	_, a, _ := vaultWithNotes(t)
	w := NewWorkspace(config.Default())
	if _, err := w.Tabs.Open(a); err != nil {
		t.Fatalf("Open: %v", err)
	}
	w.OnAsk = func(
		context.Context, []ai.Message,
	) (<-chan ai.Event, error) {
		return nil, ai.ErrAuth
	}
	s := w.aiOf(a)
	s.Input = "hello"
	w.aiAsk(w.Tabs.ActiveNote(), s)
	blocks := s.blocks()
	if len(blocks) != 2 || !strings.Contains(blocks[1].Failed, "认证") {
		t.Fatalf("blocks = %+v, want to be told of the key", blocks)
	}
}

// TestAiTold_whatTheModelIsTold checks that it is told the note, and all
// that has been said, and not the failures in between.
func TestAiTold_whatTheModelIsTold(t *testing.T) {
	events := make(chan ai.Event)
	defer close(events)
	w, s := workspaceWith(t, func([]ai.Message) <-chan ai.Event {
		return events
	})
	s.mu.Lock()
	s.Blocks = []aiBlock{
		{Role: ai.User, Text: "one", Over: true},
		{Role: ai.Assistant, Text: "two", Over: true},
		{Role: ai.Assistant, Text: "broken", Failed: "no", Over: true},
	}
	s.mu.Unlock()
	messages := w.aiTold(s, w.Tabs.ActiveNote())
	if len(messages) != 3 {
		t.Fatalf("messages = %v, want the note and two said things",
			messages)
	}
	if messages[0].Role != ai.System {
		t.Fatalf("first message = %v, want the note", messages[0].Role)
	}
	if !strings.Contains(messages[0].Content, "a.md") {
		t.Fatalf("brief = %q, want the note named", messages[0].Content)
	}
	if messages[1].Content != "one" || messages[2].Content != "two" {
		t.Fatalf("messages = %q %q", messages[1].Content,
			messages[2].Content)
	}
}

// TestAiBrief_theTongueTheAnswerIsToComeIn checks that the setting for
// the tongue is passed on to the model.
func TestAiBrief_theTongueTheAnswerIsToComeIn(t *testing.T) {
	_, a, _ := vaultWithNotes(t)
	cfg := config.Default()
	cfg.Ai.OutputLanguage = config.OutputZh
	w := NewWorkspace(cfg)
	if _, err := w.Tabs.Open(a); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if said := w.aiBrief(w.Tabs.ActiveNote()); !strings.Contains(
		said, "Answer in Chinese") {
		t.Fatalf("brief = %q, want it told to answer in Chinese", said)
	}
}

// TestAiBrief_aLongNoteIsNotSentWhole checks that a note longer than the
// setting allows is cut short rather than sent whole.
func TestAiBrief_aLongNoteIsNotSentWhole(t *testing.T) {
	path := filepath.Join(t.TempDir(), "long.md")
	if err := os.WriteFile(path, []byte(strings.Repeat("w", 500)),
		0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Ai.MaxContextChars = 64
	w := NewWorkspace(cfg)
	n, err := w.Tabs.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	said := w.aiBrief(n)
	if !strings.HasSuffix(said, strings.Repeat("w", 64)) {
		t.Fatalf("brief ends %q, want it cut at 64 characters",
			said[len(said)-70:])
	}
}

// TestToggleAI_closingOnlyHides checks what xink keeps: hiding the bar
// does not forget what was said in it.
func TestToggleAI_closingOnlyHides(t *testing.T) {
	_, a, _ := vaultWithNotes(t)
	w := NewWorkspace(config.Default())
	if _, err := w.Tabs.Open(a); err != nil {
		t.Fatalf("Open: %v", err)
	}
	s := w.aiOf(a)
	s.mu.Lock()
	s.Blocks = []aiBlock{{Role: ai.User, Text: "kept", Over: true}}
	s.mu.Unlock()

	w.ToggleAI()
	if !s.Open {
		t.Fatal("the bar did not open")
	}
	w.ToggleAI()
	if s.Open {
		t.Fatal("the bar did not hide")
	}
	if blocks := s.blocks(); len(blocks) != 1 || blocks[0].Text != "kept" {
		t.Fatalf("blocks = %v, want what was said kept", blocks)
	}
}

// TestAiForget_theConversationBeginsAgain checks that forgetting clears
// what was said.
func TestAiForget_theConversationBeginsAgain(t *testing.T) {
	events := make(chan ai.Event)
	defer close(events)
	_, s := workspaceWith(t, func([]ai.Message) <-chan ai.Event {
		return events
	})
	s.mu.Lock()
	s.Blocks = []aiBlock{{Role: ai.User, Text: "gone", Over: true}}
	s.mu.Unlock()
	s.Input = "half typed"
	(&Workspace{}).aiForget(s)
	if blocks := s.blocks(); len(blocks) != 0 {
		t.Fatalf("blocks = %v, want none", blocks)
	}
	if s.Input != "" {
		t.Fatalf("input = %q, want it cleared", s.Input)
	}
}

// TestAiStop_theAnswerIsLetGo checks that stopping cancels the asking,
// and keeps what had been said by then.
func TestAiStop_theAnswerIsLetGo(t *testing.T) {
	w, s, events := answered(t)
	s.Input = "go on"
	w.aiAsk(w.Tabs.ActiveNote(), s)
	events <- ai.Delta("half")
	w.aiStop(s)
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel == nil {
		t.Fatal("there is nothing to let go")
	}
	events <- ai.Failed(ai.ErrCanceled)
	close(events)
	settle(t, s)
	if got := s.blocks(); len(got) != 2 || got[1].Text != "half" {
		t.Fatalf("blocks = %+v, want what was said kept", got)
	}
}
