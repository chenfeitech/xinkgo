package sync

import (
	"testing"
	"time"
)

// TestBlockSpans_blankLinesPartTheBlocks checks the parting the eye
// makes, and the preview draws.
func TestBlockSpans_blankLinesPartTheBlocks(t *testing.T) {
	got := BlockSpans("# one\n\ntext one\nmore\n\n## two\n")
	want := []BlockSpan{{0, 0}, {2, 3}, {5, 5}}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("block %d is %v, want %v", i, got[i], want[i])
		}
	}
}

// TestBlockSpans_aFenceIsOneBlock checks that a code block is one block
// however many lines it has, and whatever those lines look like.
func TestBlockSpans_aFenceIsOneBlock(t *testing.T) {
	src := "text\n\n```go\n\nif x {\n}\n\n```\n\nafter\n"
	got := BlockSpans(src)
	want := []BlockSpan{{0, 0}, {2, 7}, {9, 9}}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("block %d is %v, want %v", i, got[i], want[i])
		}
	}
}

// TestAnchorTable_locateFindsTheBlock checks a line in the middle of a
// block of three lines.
func TestAnchorTable_locateFindsTheBlock(t *testing.T) {
	table := NewAnchorTable("# one\n\ntext one\nmore\nlast\n", 5)
	ix, frac := table.Locate(3)
	if ix != 1 {
		t.Errorf("block %d, want 1", ix)
	}
	// The block is lines 2 to 4, so line 3 stands a third into it.
	if frac < 0.32 || frac > 0.34 {
		t.Errorf("frac %v, want a third of the way through", frac)
	}
}

// TestAnchorTable_lineOfIsTheOtherWayRound checks that a block and a
// place in it come back as the line they came from.
func TestAnchorTable_lineOfIsTheOtherWayRound(t *testing.T) {
	table := NewAnchorTable("# one\n\ntext one\nmore\nlast\n", 5)
	if got := table.LineOf(1, 0); got != 2 {
		t.Errorf("line %v, want 2, the block's first", got)
	}
	if got := table.LineOf(1, 1); got != 5 {
		t.Errorf("line %v, want 5, the end of the block", got)
	}
}

// TestAnchorTable_outOfRangeGivesTheEnd checks that a block that is not
// there gives the end of the note, never a made-up line.
func TestAnchorTable_outOfRangeGivesTheEnd(t *testing.T) {
	table := NewAnchorTable("# one\n", 1)
	if got := table.LineOf(9, 0.5); got != 1 {
		t.Errorf("line %v, want the end of the note", got)
	}
	if ix, _ := table.Locate(99); ix != 0 {
		t.Errorf("block %d, want the last one", ix)
	}
}

// TestAnchorTable_withoutBlocksGivesNothing checks a note that has none.
func TestAnchorTable_withoutBlocksGivesNothing(t *testing.T) {
	table := NewAnchorTable("", 0)
	if !table.IsEmpty() {
		t.Error("an empty note has blocks")
	}
	if ix, frac := table.Locate(3); ix != 0 || frac != 0 {
		t.Errorf("got %d %v, want nothing", ix, frac)
	}
	if table.TotalLines() != 1 {
		t.Errorf("total lines %d, want at least 1", table.TotalLines())
	}
}

// TestAnchor_kindsDoNotMix checks that a line is never taken for a
// ratio, which would put the note somewhere else entirely.
func TestAnchor_kindsDoNotMix(t *testing.T) {
	line, ratio := LineAnchor(4), RatioAnchor(0.5)
	if line.SameKind(ratio) {
		t.Error("a line and a ratio are the same kind")
	}
	if !line.SameKind(LineAnchor(9)) {
		t.Error("two lines are not the same kind")
	}
}

// TestAnchor_clampedStaysInTheNote checks the bounds.
func TestAnchor_clampedStaysInTheNote(t *testing.T) {
	if got := LineAnchor(-2).Clamped(10).Value; got != 0 {
		t.Errorf("line %v, want 0", got)
	}
	if got := LineAnchor(40).Clamped(10).Value; got != 10 {
		t.Errorf("line %v, want 10", got)
	}
	if got := RatioAnchor(3).Clamped(10).Value; got != 1 {
		t.Errorf("ratio %v, want 1", got)
	}
}

// TestGuard_theFirstSideKeepsTheDirection checks that the side that
// moved first drives, and the other is told it is an echo.
func TestGuard_theFirstSideKeepsTheDirection(t *testing.T) {
	var g Guard
	now := time.Now()
	if !g.Claim(Editor, now) {
		t.Fatal("the editor could not take the direction")
	}
	if g.Claim(Preview, now.Add(10*time.Millisecond)) {
		t.Error("the preview took the direction while it was held")
	}
}

// TestGuard_theTurnRunsOut checks that a side is not locked out for
// longer than the turn lasts.
func TestGuard_theTurnRunsOut(t *testing.T) {
	var g Guard
	now := time.Now()
	g.Claim(Editor, now)
	if !g.Claim(Preview, now.Add(OwnerTimeout)) {
		t.Error("the preview could not take the direction after a turn")
	}
}

// TestGuard_theSameSideKeepsIt checks that going on scrolling renews a
// side's turn.
func TestGuard_theSameSideKeepsIt(t *testing.T) {
	var g Guard
	now := time.Now()
	g.Claim(Editor, now)
	if !g.Claim(Editor, now.Add(150*time.Millisecond)) {
		t.Error("the editor lost the direction while it was scrolling")
	}
	if !g.Claim(Editor, now.Add(300*time.Millisecond)) {
		t.Error("the editor lost the direction though it renewed it")
	}
}

// TestGuard_isEcho checks the place the app wrote coming back, and that
// it stops counting as one when it is too old or too far away.
func TestGuard_isEcho(t *testing.T) {
	var g Guard
	now := time.Now()
	g.Expect(LineAnchor(10), now)
	if !g.IsEcho(LineAnchor(10.2), 0.5, now) {
		t.Error("a place near what was written is not an echo")
	}
	if g.IsEcho(LineAnchor(30), 0.5, now) {
		t.Error("a place far from what was written is an echo")
	}
	if g.IsEcho(RatioAnchor(10), 0.5, now) {
		t.Error("a ratio is an echo of a line")
	}
	old := now.Add(EchoTimeout + time.Millisecond)
	if g.IsEcho(LineAnchor(10), 0.5, old) {
		t.Error("a place read after the echo's time is still one")
	}
}

// TestGuard_resetForgetsEverything checks what showing another note
// does: the new one must not be jumped to the old one's place.
func TestGuard_resetForgetsEverything(t *testing.T) {
	var g Guard
	now := time.Now()
	g.Claim(Editor, now)
	g.Expect(LineAnchor(10), now)
	g.Reset()
	if !g.Claim(Preview, now.Add(time.Millisecond)) {
		t.Error("the preview could not take the direction after a reset")
	}
	if g.IsEcho(LineAnchor(10), 0.5, now) {
		t.Error("the echo was remembered across the reset")
	}
}

// TestGesture_isQuietWhileTheUserScrolls checks the window in which the
// app writes nothing at all.
func TestGesture_isQuietWhileTheUserScrolls(t *testing.T) {
	var g Gesture
	now := time.Now()
	if g.IsQuiet(now) {
		t.Error("it is quiet before anything happened")
	}
	g.NoteUserMovement(now)
	if !g.IsQuiet(now.Add(QuietPeriod - time.Millisecond)) {
		t.Error("it is not quiet just after the user scrolled")
	}
	if g.IsQuiet(now.Add(QuietPeriod + time.Millisecond)) {
		t.Error("it is still quiet after the window closed")
	}
	g.Reset()
	if g.IsQuiet(now.Add(2 * QuietPeriod)) {
		t.Error("it is quiet after a reset")
	}
}

// TestModeOf_names checks the three modes, and a name that is none.
func TestModeOf_names(t *testing.T) {
	cases := map[string]Mode{
		"ratio":   Ratio,
		"anchor":  AnchorMode,
		"on_stop": OnStop,
		"moon":    Ratio,
	}
	for in, want := range cases {
		if got := ModeOf(in); got != want {
			t.Errorf("ModeOf(%q) = %d, want %d", in, got, want)
		}
	}
}
