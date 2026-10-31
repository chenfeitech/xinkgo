package view

import (
	"context"
	"strings"
	"testing"

	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
)

// workspaceAsking is a window with one note open, that answers with the
// words given and remembers what it was asked.
func workspaceAsking(
	t *testing.T, answer string, told *[]ai.Message,
) (*Workspace, *OpenNote, *aiSession) {
	t.Helper()
	_, a, _ := vaultWithNotes(t)
	w := NewWorkspace(config.Default())
	n, err := w.Tabs.Open(a)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	n.Editor.Buffer.Set("one\ntwo\nthree\n")
	w.OnAsk = func(ctx context.Context, got []ai.Message) (
		<-chan ai.Event, error,
	) {
		*told = got
		events := make(chan ai.Event, 2)
		events <- ai.Delta(answer)
		events <- ai.Done()
		close(events)
		return events, nil
	}
	return w, n, w.aiOf(a)
}

// polish is one of the actions the app ships with.
func polish() config.AiAction {
	for _, one := range config.DefaultActions() {
		if one.ID == "polish" {
			return one
		}
	}
	panic("the app no longer ships a polish action")
}

// TestActionItems_theActionsAreOfferedByName checks that the app's own
// actions are found in the palette by their names, as the skills are.
func TestActionItems_theActionsAreOfferedByName(t *testing.T) {
	w := NewWorkspace(config.Default())
	w.Actions = config.EnabledActions(config.DefaultActions())
	w.paletteQuery = ""
	found := map[string]bool{}
	for _, item := range w.paletteItems() {
		found[item.Name] = true
	}
	if !found["Action · "+polish().Name] {
		t.Fatalf("items = %v, want the actions in it", found)
	}
	// A few letters of the name find it, and the letters stand out.
	w.paletteQuery = "pol"
	items := w.actionItems()
	if len(items) != 1 || items[0].Action == nil {
		t.Fatalf("items = %v, want the one action", items)
	}
	if items[0].Title != polish().Prompt {
		t.Fatalf("title = %q, want what the action asks", items[0].Title)
	}
	if len(items[0].Positions) == 0 {
		t.Fatal("the letters that were typed are not lit up")
	}
	// A window with none has none to offer.
	bare := NewWorkspace(config.Default())
	if len(bare.actionItems()) != 0 {
		t.Fatal("a window with no actions offered some")
	}
}

// TestRunAction_whatTheActionSaysIsAsked checks the one thing an action
// is for: the words it keeps are the words the model is asked.
func TestRunAction_whatTheActionSaysIsAsked(t *testing.T) {
	told := []ai.Message{}
	w, _, s := workspaceAsking(t, "ONE\ntwo\nthree\n", &told)
	w.runAction(polish())
	if len(told) == 0 {
		t.Fatal("nothing was asked")
	}
	last := told[len(told)-1]
	if last.Role != ai.User {
		t.Fatalf("role = %v, want the question", last.Role)
	}
	if !strings.Contains(last.Content, polish().Prompt) {
		t.Fatalf("asked = %q, want what the action says", last.Content)
	}
	// What was asked is said in the conversation, as it is with
	// anything else one asks.
	blocks := s.blocks()
	if len(blocks) != 2 || !strings.Contains(blocks[0].Text,
		polish().Prompt) {
		t.Fatalf("blocks = %+v, want the asking in it", blocks)
	}
}

// TestAiReviewDone_theAnswerIsPutBeforeTheUser checks that an action's
// answer is held back as a revision and not written into the note: the
// same road any other revision goes by.
func TestAiReviewDone_theAnswerIsPutBeforeTheUser(t *testing.T) {
	told := []ai.Message{}
	w, n, s := workspaceAsking(t, "one\nTWO\nthree\n", &told)
	w.runAction(polish())
	settle(t, s)
	w.aiReviewDone()
	if !w.review.open {
		t.Fatal("the answer was not put before the user")
	}
	if got := n.Editor.Buffer.String(); got != "one\ntwo\nthree\n" {
		t.Fatalf("note = %q, want it left alone", got)
	}
	// It is put before the user once, and not again every frame.
	w.review.open = false
	w.aiReviewDone()
	if w.review.open {
		t.Fatal("the same answer was put before the user twice")
	}
}

// TestPieceOf_aSelectionIsThePieceAndWithoutOneItIsTheWhole checks what
// an action is done to: what is selected when something is, and the
// whole note when nothing is.
func TestPieceOf_aSelectionIsThePieceAndWithoutOneItIsTheWhole(t *testing.T) {
	_, n := workspaceAndNote(t, "one\ntwo\nthree\n")
	one := polish()
	text, from, to, part := pieceOf(n, one)
	if part || text != "one\ntwo\nthree\n" {
		t.Fatalf("piece = %q part = %v, want the whole note", text, part)
	}
	if from != 0 || to != 0 {
		t.Fatalf("from,to = %d,%d, want the whole of it", from, to)
	}
	n.Editor.Select(4, 7) // the word "two"
	text, from, to, part = pieceOf(n, one)
	if !part || text != "two" {
		t.Fatalf("piece = %q part = %v, want what is selected",
			text, part)
	}
	if from != 4 || to != 7 {
		t.Fatalf("from,to = %d,%d, want where it stands", from, to)
	}
	// An action that is done to the whole note goes by the whole of it,
	// selected or not.
	whole := one
	whole.Scope = config.ActionScopeWhole
	if text, _, _, part = pieceOf(n, whole); part ||
		text != "one\ntwo\nthree\n" {
		t.Fatalf("piece = %q part = %v, want the whole note", text, part)
	}
}

// TestAskingOf_aPieceIsGivenAndTheWholeIsNot checks what the model is
// told: the note as a whole it already knows, so only a piece is sent
// with the asking.
func TestAskingOf_aPieceIsGivenAndTheWholeIsNot(t *testing.T) {
	one := polish()
	if got := askingOf(one, "one\ntwo\n", false); got != one.Prompt {
		t.Fatalf("asked = %q, want what the action says and no more",
			got)
	}
	got := askingOf(one, "two", true)
	if !strings.Contains(got, one.Prompt) ||
		!strings.Contains(got, "two") {
		t.Fatalf("asked = %q, want the piece with it", got)
	}
}

// TestStartPieceReview_onlyThePieceIsRevised checks that a revision of a
// piece is of the piece and of nothing else: the rest of the note is not
// in it.
func TestStartPieceReview_onlyThePieceIsRevised(t *testing.T) {
	w, n := workspaceAndNote(t, "one\ntwo\nthree\n")
	w.StartPieceReview(4, 7, "TWO")
	if !w.review.open {
		t.Fatal("the revision was not put before the user")
	}
	if !w.review.part {
		t.Fatal("the revision is not of a piece")
	}
	if w.review.before != "two" {
		t.Fatalf("before = %q, want the piece as it was", w.review.before)
	}
	if got := w.review.diff.New.String(); got != "TWO" {
		t.Fatalf("new = %q, want what the model made of the piece", got)
	}
	if got := n.Editor.Buffer.String(); got != "one\ntwo\nthree\n" {
		t.Fatalf("note = %q, want it left alone", got)
	}
	// A piece that is not in the note any more cannot be put back.
	w.review.open = false
	w.StartPieceReview(100, 120, "TWO")
	if w.review.open {
		t.Fatal("a piece that is not there was revised")
	}
}

// TestWriteReview_aPieceGoesBackWhereItWasTakenFrom checks that taking a
// revision of a piece writes the piece and leaves the rest alone.
func TestWriteReview_aPieceGoesBackWhereItWasTakenFrom(t *testing.T) {
	w, n := workspaceAndNote(t, "one\ntwo\nthree\n")
	w.OnSnapshot = func(path, text string) string {
		kept = text
		return "/somewhere"
	}
	defer func() { kept = "" }()
	w.StartPieceReview(4, 7, "TWO")
	w.writeReview()
	if got := n.Editor.Buffer.String(); got != "one\nTWO\nthree\n" {
		t.Fatalf("note = %q, want the piece and nothing else", got)
	}
	// The way back holds the whole note, and not only the piece: it is
	// the whole note that is written over.
	if kept != "one\ntwo\nthree\n" {
		t.Fatalf("kept %q, want the whole note as it was", kept)
	}
	if w.review.open {
		t.Fatal("the revision was not put away")
	}
}

// TestWriteReview_aPieceSentBackLeavesTheNoteAsItWas checks that sending
// the change back is the same as never having asked.
func TestWriteReview_aPieceSentBackLeavesTheNoteAsItWas(t *testing.T) {
	w, n := workspaceAndNote(t, "one\ntwo\nthree\n")
	w.StartPieceReview(4, 7, "TWO")
	w.reviewEach(false)
	w.writeReview()
	if got := n.Editor.Buffer.String(); got != "one\ntwo\nthree\n" {
		t.Fatalf("note = %q, want it as it was", got)
	}
}

// TestEnabledActions_oneTurnedOffIsNotOffered checks that an action the
// user turned off is not put before them.
func TestEnabledActions_oneTurnedOffIsNotOffered(t *testing.T) {
	all := config.DefaultActions()
	all[0].Enable = false
	got := config.EnabledActions(all)
	if len(got) != len(all)-1 {
		t.Fatalf("enabled = %d, want one fewer", len(got))
	}
	for _, one := range got {
		if one.ID == all[0].ID {
			t.Fatalf("%q was offered though it is turned off", one.ID)
		}
	}
}
