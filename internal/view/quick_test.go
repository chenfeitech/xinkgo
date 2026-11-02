package view

import (
	"context"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
)

// quickWorkspace is a window with one note open and the bar open under
// it, whose answer has already come to an end, and which keeps what the
// model was asked.
func quickWorkspace(t *testing.T) (*Workspace, *aiSession, *[]ai.Message) {
	t.Helper()
	_, a, _ := vaultWithNotes(t)
	w := NewWorkspace(config.Default())
	if _, err := w.Tabs.Open(a); err != nil {
		t.Fatalf("Open: %v", err)
	}
	// The words as the app gives them to the window: as they were
	// written in, and then as the settings keep them, which is what
	// gives each of them a word to be run by.
	cfg := config.Default()
	cfg.Sanitize()
	w.Actions = config.EnabledActions(cfg.Ai.Actions)
	asked := &[]ai.Message{}
	w.OnAsk = func(
		_ context.Context, msgs []ai.Message, _ string,
	) (<-chan ai.Event, error) {
		*asked = append([]ai.Message{}, msgs...)
		events := make(chan ai.Event)
		close(events)
		return events, nil
	}
	// The bar opened as the user opens it, which asks for the keys to
	// be given to the place to ask.
	w.ToggleAI()
	return w, w.aiOf(a), asked
}

// askedLast is the last thing the user said to the model, of all that it
// was told.
func askedLast(msgs []ai.Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == ai.User {
			return msgs[i].Content
		}
	}
	return ""
}

// TestWordBeingTyped_theWordAndWhatGoesWithIt checks what is taken for a
// word: the letters after the slash, up to the first space, and after it
// the words that go with the asking.
func TestWordBeingTyped_theWordAndWhatGoesWithIt(t *testing.T) {
	for _, one := range []struct {
		input       string
		query, rest string
		typing      bool
	}{
		{"", "", "", false},
		{"what is this about", "", "", false},
		{"/", "", "", true},
		{"/pol", "pol", "", true},
		{"/pol keep it short", "pol", "keep it short", true},
		{"/pol  ", "pol", "", true},
		{"say /pol", "", "", false},
	} {
		query, rest, typing := wordBeingTyped(one.input)
		if query != one.query || rest != one.rest || typing != one.typing {
			t.Errorf("wordBeingTyped(%q) = %q, %q, %v, want %q, %q, %v",
				one.input, query, rest, typing,
				one.query, one.rest, one.typing)
		}
	}
}

// TestQuickWords_whatWasTypedIsFoundInThem checks that a few letters find
// the words they stand in, whether they were typed as the word is run or
// as it is called.
func TestQuickWords_whatWasTypedIsFoundInThem(t *testing.T) {
	w := NewWorkspace(config.Default())
	w.Actions = config.EnabledActions(config.DefaultActions())
	if got := len(w.quickWords("pol")); got != 1 {
		t.Fatalf("%d words found for pol, want one", got)
	}
	if got := w.quickWords("pol")[0].ID; got != "polish" {
		t.Fatalf("the word found is %q, want polish", got)
	}
	// What it is called, and not what it is run by: the two are the
	// same in what the app ships with, and need not be afterwards.
	w.Actions[0].Trigger = "shine"
	if got := len(w.quickWords("shine")); got != 1 {
		t.Fatalf("%d words found for shine, want one", got)
	}
	if got := len(w.quickWords("polish")); got != 1 {
		t.Fatalf("%d words found for polish, want one by its name", got)
	}
	if got := len(w.quickWords("zzz")); got != 0 {
		t.Fatalf("%d words found for zzz, want none", got)
	}
}

// TestQuickRun_aWordAsksWhatItSays checks that choosing a word asks the
// model what the word says, and that what comes back is to be put before
// the user as a revision, as any other answer to an action is.
func TestQuickRun_aWordAsksWhatItSays(t *testing.T) {
	w, s, asked := quickWorkspace(t)
	n := w.Tabs.ActiveNote()
	w.quickRun(n, s, w.Actions[0], "")
	if got := askedLast(*asked); got != w.Actions[0].Prompt {
		t.Fatalf("asked %q, want what the word says", got)
	}
	if s.waiting == nil {
		t.Fatal("the answer is not to be looked over as a revision")
	}
	if s.Input != "" {
		t.Fatalf("the place to ask still says %q", s.Input)
	}
}

// TestQuickRun_wordsAfterTheWordGoWithIt checks that what was typed after
// the word goes with what it asks: a word may be made to ask for a
// little more this once, without being changed.
func TestQuickRun_wordsAfterTheWordGoWithIt(t *testing.T) {
	w, s, asked := quickWorkspace(t)
	n := w.Tabs.ActiveNote()
	w.quickRun(n, s, w.Actions[0], "keep it short")
	got := askedLast(*asked)
	if !strings.HasPrefix(got, w.Actions[0].Prompt) ||
		!strings.HasSuffix(got, "keep it short") {
		t.Fatalf("asked %q, want the word and then what was added", got)
	}
	if strings.Contains(w.Actions[0].Prompt, "keep it short") {
		t.Fatal("the word itself was changed")
	}
}

// TestRunAction_aWordWithNothingToAskAsksNothing checks that a word the
// user has written nothing into is not asked of the model at all: the
// note is left as it is, and they are told why.
func TestRunAction_aWordWithNothingToAskAsksNothing(t *testing.T) {
	w, s, asked := quickWorkspace(t)
	word := w.Actions[0]
	word.Prompt = "   "
	w.runAction(word)
	if len(*asked) != 0 {
		t.Fatalf("the model was asked %q all the same", askedLast(*asked))
	}
	if s.waiting != nil {
		t.Fatal("an answer is waited for that will not come")
	}
	if w.status == "" {
		t.Fatal("the user was told nothing")
	}
}

// TestQuickStrip_theWordsStandAboveThePlaceToAsk checks where the words
// stand: in a row of their own above the place to ask, and not in a
// panel floating over the note, so that one press asks a word and what
// is being typed can still be read.
func TestQuickStrip_theWordsStandAboveThePlaceToAsk(t *testing.T) {
	w, _, _ := quickWorkspace(t)
	tt := ui.NewTester(w.View, 900, 700)
	tt.SetFocused(true)
	tt.Frame()
	word, ok := tt.Find("Polish")
	if !ok {
		t.Fatal("the words do not stand above the place to ask")
	}
	ask, ok := tt.Find(locale.AiAsk())
	if !ok {
		t.Fatal("the way to ask was not found")
	}
	if word.Y+word.H > ask.Y {
		t.Fatalf("the word stands at %v to %v, which is not above "+
			"the place to ask at %v", word.Y, word.Y+word.H, ask.Y)
	}
}

// TestQuickStrip_aPressAsksWhatTheWordAsks checks the words as the user
// meets them: they stand there without a slash being typed, and one
// press on one of them asks the model what that word asks.
func TestQuickStrip_aPressAsksWhatTheWordAsks(t *testing.T) {
	w, s, asked := quickWorkspace(t)
	tt := ui.NewTester(w.View, 900, 700)
	tt.SetFocused(true)
	tt.Frame()
	if err := tt.Click("Polish"); err != nil {
		t.Fatalf("the word could not be pressed: %v", err)
	}
	tt.Frame()
	if got := askedLast(*asked); !strings.Contains(got, "Polish the writing") {
		t.Fatalf("asked %q, want what the word says", got)
	}
	if s.Input != "" {
		t.Fatalf("the place to ask still says %q", s.Input)
	}
}

// TestQuickStrip_aSlashTypedPicksAmongThem checks the other way in: what
// is typed after a slash leaves the words it was not found in out of the
// row, and Enter asks the one it was found in.
func TestQuickStrip_aSlashTypedPicksAmongThem(t *testing.T) {
	w, _, asked := quickWorkspace(t)
	tt := ui.NewTester(w.View, 900, 700)
	tt.SetFocused(true)
	tt.Frame()
	if !tt.HasText("Polish") || !tt.HasText("Condense") {
		t.Fatal("the words are not on show")
	}
	tt.Type("/cond")
	tt.Frame()
	if tt.HasText("Polish") {
		t.Fatal("what was typed did not pick among the words")
	}
	if !tt.HasText("Condense") {
		t.Fatal("the word that was typed is not on show")
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if got := askedLast(*asked); !strings.Contains(got, "Condense") {
		t.Fatalf("asked %q, want what the word says", got)
	}
}

// TestQuickStrip_theArrowsMoveAmongTheWordsAndEnterAsks checks the keys
// of the row: the arrows move from one word to the next, and Enter asks
// what the one they stand on asks.
func TestQuickStrip_theArrowsMoveAmongTheWordsAndEnterAsks(t *testing.T) {
	w, _, asked := quickWorkspace(t)
	tt := ui.NewTester(w.View, 900, 700)
	tt.SetFocused(true)
	tt.Frame()
	tt.Type("/")
	tt.Frame()
	// Twice down from the first of them: polish, condense, translate.
	for range 2 {
		tt.Key(0, ui.KeyDown)
		tt.Frame()
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if got := askedLast(*asked); !strings.Contains(got, "Translate this") {
		t.Fatalf("asked %q, want what the third word says", got)
	}
}
