package view

import (
	"testing"

	"github.com/helightxu/xinkgo/internal/config"
)

// workspaceAndNote is a window with one note open, holding the text given.
func workspaceAndNote(t *testing.T, text string) (*Workspace, *OpenNote) {
	t.Helper()
	_, a, _ := vaultWithNotes(t)
	w := NewWorkspace(config.Default())
	n, err := w.Tabs.Open(a)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	n.Editor.Buffer.Set(text)
	return w, n
}

// TestStartReview_whatTheModelMadeIsPutBeforeTheUser checks that a
// revision is held back rather than written into the note: the note is
// still as it was, and the revision stands beside it.
func TestStartReview_whatTheModelMadeIsPutBeforeTheUser(t *testing.T) {
	w, n := workspaceAndNote(t, "one\ntwo\nthree\n")
	w.StartReview("one\nTWO\nthree\n")
	if !w.review.open {
		t.Fatal("the revision was not put before the user")
	}
	if w.review.path != n.Path {
		t.Fatalf("path = %q, want the note's", w.review.path)
	}
	if w.review.before != "one\ntwo\nthree\n" {
		t.Fatalf("before = %q, want the note as it was", w.review.before)
	}
	if n.Editor.Buffer.String() != "one\ntwo\nthree\n" {
		t.Fatalf("note = %q, want it left alone",
			n.Editor.Buffer.String())
	}
	if hunks := w.review.diff.Hunks(); len(hunks) != 1 {
		t.Fatalf("hunks = %v, want one change", hunks)
	}
}

// TestStartReview_theSameWordsAreNoRevision checks that a model which
// said what the note says is told so, rather than being reviewed.
func TestStartReview_theSameWordsAreNoRevision(t *testing.T) {
	w, _ := workspaceAndNote(t, "one\ntwo\n")
	w.StartReview("one\ntwo\n")
	if w.review.open {
		t.Fatal("a revision of nothing was put before the user")
	}
	if w.status == "" {
		t.Fatal("nothing was said of it")
	}
}

// TestStartReview_noNoteHasNothingToRevise checks that a window with no
// note open does not open a revision.
func TestStartReview_noNoteHasNothingToRevise(t *testing.T) {
	w := NewWorkspace(config.Default())
	w.StartReview("anything\n")
	if w.review.open {
		t.Fatal("a revision was opened with no note in the window")
	}
}

// TestReviewEach_everyChangeTaken checks that taking them all leaves the
// revision with nothing left to show, and the new words where they are.
func TestReviewEach_everyChangeTaken(t *testing.T) {
	w, _ := workspaceAndNote(t, "one\ntwo\nthree\n")
	w.StartReview("one\nTWO\nTHREE\n")
	w.reviewEach(true)
	if hunks := w.review.diff.Hunks(); len(hunks) != 0 {
		t.Fatalf("hunks = %v, want none left", hunks)
	}
	if got := w.review.diff.New.String(); got != "one\nTWO\nTHREE\n" {
		t.Fatalf("new = %q, want the model's words kept", got)
	}
}

// TestReviewEach_everyChangeSentBack checks that sending them all back
// leaves the note's own words.
func TestReviewEach_everyChangeSentBack(t *testing.T) {
	w, _ := workspaceAndNote(t, "one\ntwo\nthree\n")
	w.StartReview("one\nTWO\nTHREE\n")
	w.reviewEach(false)
	if hunks := w.review.diff.Hunks(); len(hunks) != 0 {
		t.Fatalf("hunks = %v, want none left", hunks)
	}
	if got := w.review.diff.New.String(); got != "one\ntwo\nthree\n" {
		t.Fatalf("new = %q, want the note's own words back", got)
	}
}

// TestWriteReview_whatIsTakenGoesIntoTheNote checks that writing puts
// what is left into the note and puts the revision away.
func TestWriteReview_whatIsTakenGoesIntoTheNote(t *testing.T) {
	w, n := workspaceAndNote(t, "one\ntwo\nthree\n")
	w.StartReview("one\nTWO\nthree\n")
	w.writeReview()
	if got := n.Editor.Buffer.String(); got != "one\nTWO\nthree\n" {
		t.Fatalf("note = %q, want the change taken", got)
	}
	if w.review.open {
		t.Fatal("the revision was not put away")
	}
}

// TestWriteReview_whatIsSentBackLeavesTheNoteAlone checks that sending
// every change back and writing is the same as never having asked.
func TestWriteReview_whatIsSentBackLeavesTheNoteAlone(t *testing.T) {
	w, n := workspaceAndNote(t, "one\ntwo\nthree\n")
	w.StartReview("one\nTWO\nTHREE\n")
	w.reviewEach(false)
	w.writeReview()
	if got := n.Editor.Buffer.String(); got != "one\ntwo\nthree\n" {
		t.Fatalf("note = %q, want it as it was", got)
	}
	if w.review.open {
		t.Fatal("the revision was not put away")
	}
	if w.status == "" {
		t.Fatal("nothing was said of it")
	}
}

// TestReviewStale_anotherNotePutsTheRevisionAway checks that a revision
// of one note is not left over the text of another.
func TestReviewStale_anotherNotePutsTheRevisionAway(t *testing.T) {
	w, _ := workspaceAndNote(t, "one\ntwo\n")
	w.StartReview("one\nTWO\n")
	if !w.review.open {
		t.Fatal("the revision was not opened")
	}
	if w.reviewStale() {
		t.Fatal("the revision of the note being looked at is not stale")
	}
	w.review.path = "/somewhere/else.md"
	if !w.reviewStale() {
		t.Fatal("a revision of another note was not seen as stale")
	}
}
