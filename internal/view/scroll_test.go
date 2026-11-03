package view

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/sync"
)

// How wide the preview is in the tests, and where the pointer is put to
// scroll the note and to scroll the preview: the note stands on the left
// of the window and the preview on its right.
const (
	previewWidth = 400
	noteX        = 200
	previewX     = 800
)

// longNote is a note long enough to be scrolled on both sides: a head
// and a line of words, with a blank line between one and the next.
func longNote() string {
	var b strings.Builder
	for i := 0; i < 120; i++ {
		fmt.Fprintf(&b, "## head %d\n\nwords %d\n\n", i, i)
	}
	return b.String()
}

// scrolledWindow is a window with the long note open in it, the preview
// beside it, and a clock the tests move themselves.
func scrolledWindow(t *testing.T) (*Workspace, *OpenNote, *ui.Tester) {
	t.Helper()
	_, a, _ := vaultWithNotes(t)
	w := NewWorkspace(config.Default())
	n, err := w.Tabs.Open(a)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	n.Editor.Buffer.Set(longNote())
	w.Layout.Left = 0
	w.Layout.Right = previewWidth
	te := ui.NewTester(w.View, 1000, 600)
	te.Frame()
	if w.scroll.Preview.MaxY <= 0 {
		t.Fatalf("the preview is not scrollable: %+v", w.scroll.Preview)
	}
	return w, n, te
}

// userScrolls is the user scrolling at x: the pointer is put there, and
// the wheel turned.
func userScrolls(te *ui.Tester, x float32, dy float32) {
	te.Move(x, 300)
	te.Scroll(x, 300, 0, dy)
}

// TestFollowScroll_thePreviewComesToTheNote checks the way from the note
// to what it looks like: the user scrolls the note, and the preview is
// brought to the same place, and not left at the top.
func TestFollowScroll_thePreviewComesToTheNote(t *testing.T) {
	w, n, te := scrolledWindow(t)
	userScrolls(te, noteX, 400)
	te.Frame()
	if n.Editor.ScrollY() <= 0 {
		t.Fatalf("the note did not scroll: %v", n.Editor.ScrollY())
	}
	if w.scroll.Preview.Y <= 0 {
		t.Fatalf("the preview stayed at the top: %+v", w.scroll.Preview)
	}
	// The two stand at the same place in the note, and have not merely
	// both moved: how far down its lines the note is is how far down
	// what it drew the preview is.
	lineHeight := n.Editor.LineTop(1) - n.Editor.LineTop(0)
	line := float32(n.Editor.ScrollY()) / float32(lineHeight)
	want := line / float32(n.Editor.Buffer.LineCount()) *
		w.scroll.Preview.MaxY
	// A line of the note is a good part of the preview: how far off
	// the two may be and still be called the same place.
	tol := w.scroll.Preview.MaxY / float32(n.Editor.Buffer.LineCount())
	if d := w.scroll.Preview.Y - want; d < -tol || d > tol {
		t.Errorf("the preview is at %v, want it at %v (of %v)",
			w.scroll.Preview.Y, want, w.scroll.Preview.MaxY)
	}
}

// TestFollowScroll_theNoteComesToThePreview checks the way back: the
// user scrolls the preview, and the note is brought to it.
func TestFollowScroll_theNoteComesToThePreview(t *testing.T) {
	w, n, te := scrolledWindow(t)
	userScrolls(te, previewX, 500)
	te.Frame()
	if w.scroll.Preview.Y <= 0 {
		t.Fatalf("the preview did not scroll: %+v", w.scroll.Preview)
	}
	if n.Editor.ScrollY() <= 0 {
		t.Fatalf("the note stayed at the top: %v", n.Editor.ScrollY())
	}
}

// TestFollowScroll_theSideTheUserScrolledIsLeftAlone checks that the two
// do not push each other: what the app wrote on one side is not read
// back as the user scrolling it, which would pull the note back to
// where the preview was put.
func TestFollowScroll_theSideTheUserScrolledIsLeftAlone(t *testing.T) {
	w, n, te := scrolledWindow(t)
	userScrolls(te, noteX, 400)
	te.Frame()
	at := float32(n.Editor.ScrollY())
	preview := w.scroll.Preview.Y
	// Time enough for what was written to come back, and be read as
	// the user's own were it not known for what it is.
	for i := 0; i < 4; i++ {
		te.Advance(80 * time.Millisecond)
		te.Frame()
	}
	if got := float32(n.Editor.ScrollY()); moved(got, at) {
		t.Errorf("the note was pulled from %v to %v", at, got)
	}
	if got := w.scroll.Preview.Y; moved(got, preview) {
		t.Errorf("the preview moved on from %v to %v", preview, got)
	}
}

// TestFollowScroll_itFollowsAsTheUserGoes checks that the other side
// comes along while the user is still scrolling, and not only once they
// have stopped: the preview is moved on the very frame after the note's
// first move, and again after the next.
func TestFollowScroll_itFollowsAsTheUserGoes(t *testing.T) {
	w, _, te := scrolledWindow(t)
	userScrolls(te, noteX, 200)
	te.Frame()
	first := w.scroll.Preview.Y
	if first <= 0 {
		t.Fatalf("the preview did not come: %+v", w.scroll.Preview)
	}
	userScrolls(te, noteX, 200)
	te.Frame()
	if second := w.scroll.Preview.Y; second <= first {
		t.Errorf("the preview stood at %v, want it on to %v and further",
			second, first)
	}
}

// TestFollowScroll_onStopWaitsForTheScrollToEnd checks the mode that
// follows only when the scrolling has stood still: nothing is moved
// while the user is still at it, and the other side comes once they are
// not.
func TestFollowScroll_onStopWaitsForTheScrollToEnd(t *testing.T) {
	w, _, te := scrolledWindow(t)
	w.Config.Preview.SyncMode = config.SyncOnStop
	userScrolls(te, noteX, 400)
	te.Frame()
	if w.scroll.Preview.Y != 0 {
		t.Fatalf("the preview came at once: %+v, want it to wait",
			w.scroll.Preview)
	}
	// Still scrolling: a second move within the moment it waits.
	userScrolls(te, noteX, 100)
	te.Frame()
	if w.scroll.Preview.Y != 0 {
		t.Fatalf("the preview came while the user was at it: %+v",
			w.scroll.Preview)
	}
	te.Advance(sync.StopQuiet + 20*time.Millisecond)
	te.Frame()
	if w.scroll.Preview.Y <= 0 {
		t.Errorf("the preview never came: %+v", w.scroll.Preview)
	}
}

// TestFollowScroll_turnedOffNeitherFollowsTheOther checks the setting:
// with the following off, each side stays where the user put it.
func TestFollowScroll_turnedOffNeitherFollowsTheOther(t *testing.T) {
	w, n, te := scrolledWindow(t)
	w.Config.Preview.SyncScroll = false
	userScrolls(te, noteX, 400)
	te.Frame()
	if w.scroll.Preview.Y != 0 {
		t.Errorf("the preview came though the following is off: %+v",
			w.scroll.Preview)
	}
	userScrolls(te, previewX, 400)
	te.Frame()
	if n.Editor.ScrollY() == 0 {
		t.Fatalf("the preview did not scroll")
	}
	if w.scroll.Preview.Y <= 0 {
		t.Fatalf("the preview did not scroll: %+v", w.scroll.Preview)
	}
}

// TestFollowScroll_anotherNoteStartsAtItsTop checks that showing another
// note does not bring it to where the last one was left.
func TestFollowScroll_anotherNoteStartsAtItsTop(t *testing.T) {
	w, _, te := scrolledWindow(t)
	userScrolls(te, noteX, 400)
	te.Frame()
	if w.scroll.Preview.Y <= 0 {
		t.Fatalf("the preview did not come: %+v", w.scroll.Preview)
	}
	_, _, b := vaultWithNotes(t)
	if _, err := w.Tabs.Open(b); err != nil {
		t.Fatalf("Open: %v", err)
	}
	te.Frame()
	if w.scroll.Preview.Y != 0 {
		t.Errorf("the other note was put at %v, want it at its top",
			w.scroll.Preview.Y)
	}
}
