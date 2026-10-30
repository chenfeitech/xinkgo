// Package sync is the ground the editor and the preview share when they
// follow each other.
//
// The two do not hold the same thing: one has the lines of a Markdown
// note, the other the blocks it was drawn as. Copying "how far down the
// scroll bar is" from one to the other therefore misses, so what crosses
// between them is a place in the note — a line, or the ratio when there
// are no lines to speak of — and each side says where that is for it.
//
// Nothing here touches a window: it is numbers, and can be tested alone.
package sync

import (
	"strings"
	"time"
)

// BlockSpan is the lines of the note that one block of the preview was
// drawn from, from 0 and including both ends.
type BlockSpan struct {
	StartLine int
	EndLine   int
}

// height is how many lines of the note the block stands for, at least 1.
func (b BlockSpan) height() float32 {
	return float32(b.EndLine-b.StartLine) + 1
}

// AnchorTable is the preview's blocks and the lines they came from: what
// a line of the note and a block of the preview are to each other.
type AnchorTable struct {
	blocks     []BlockSpan
	totalLines int
}

// NewAnchorTable is the table of a note: its blocks, and how many lines
// it has.
func NewAnchorTable(source string, totalLines int) AnchorTable {
	return AnchorTable{blocks: BlockSpans(source), totalLines: totalLines}
}

// Len is how many blocks the preview drew.
func (t AnchorTable) Len() int { return len(t.blocks) }

// IsEmpty reports whether there is nothing to align to.
func (t AnchorTable) IsEmpty() bool { return len(t.blocks) == 0 }

// TotalLines is how many lines the note has, at least 1.
func (t AnchorTable) TotalLines() int {
	if t.totalLines < 1 {
		return 1
	}
	return t.totalLines
}

// Locate is the block a line of the note is in, and how far into it.
func (t AnchorTable) Locate(line float32) (int, float32) {
	if len(t.blocks) == 0 {
		return 0, 0
	}
	if line < 0 {
		line = 0
	}
	target := int(line)
	ix := len(t.blocks) - 1
	for i, b := range t.blocks {
		if target <= b.EndLine {
			ix = i
			break
		}
	}
	b := t.blocks[ix]
	frac := (line - float32(b.StartLine)) / b.height()
	return ix, clampf(frac, 0, 1)
}

// LineOf is the line of the note a place in a block stands for: the
// other way round. A block that is not there gives the end of the note,
// and never a made-up number.
func (t AnchorTable) LineOf(ix int, frac float32) float32 {
	if ix < 0 || ix >= len(t.blocks) {
		return float32(t.TotalLines())
	}
	b := t.blocks[ix]
	return float32(b.StartLine) + clampf(frac, 0, 1)*b.height()
}

// BlockSpans is the lines each top-level block of a note was drawn from.
//
// Blocks are parted by blank lines, with the lines of a fenced code
// block kept in one piece whatever they hold: that is the parting the
// eye makes, and it is what the preview draws as one block.
func BlockSpans(source string) []BlockSpan {
	lines := strings.Split(source, "\n")
	var (
		out     []BlockSpan
		start   = -1
		fence   string
		inFence bool
	)
	flush := func(end int) {
		if start >= 0 && end >= start {
			out = append(out, BlockSpan{StartLine: start, EndLine: end})
		}
		start = -1
	}
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		switch {
		case inFence:
			// A fence closes on a line of its own mark; a blank line
			// inside it parts nothing.
			if strings.HasPrefix(trimmed, fence) {
				inFence, fence = false, ""
			}
		case strings.HasPrefix(trimmed, "```"):
			inFence, fence = true, "```"
		case strings.HasPrefix(trimmed, "~~~"):
			inFence, fence = true, "~~~"
		case strings.TrimSpace(line) == "":
			flush(i - 1)
			continue
		}
		if start < 0 {
			start = i
		}
	}
	flush(len(lines) - 1)
	return out
}

// Kind is what a place between the two is measured in.
type Kind int

const (
	// KindLine is a line of the note, fractions of one between them.
	KindLine Kind = iota
	// KindRatio is how far down the note is scrolled, 0 to 1, for what
	// has no lines to speak of.
	KindRatio
)

// Anchor is a place in the note, as one side sees it and the other can
// find: a line, or a ratio.
type Anchor struct {
	Kind  Kind
	Value float32
}

// LineAnchor is a place given as a line of the note.
func LineAnchor(line float32) Anchor { return Anchor{Kind: KindLine, Value: line} }

// RatioAnchor is a place given as how far down the note is scrolled.
func RatioAnchor(r float32) Anchor { return Anchor{Kind: KindRatio, Value: r} }

// Clamped is the anchor brought into the note: a line to between 0 and
// maxLine, a ratio to between 0 and 1.
func (a Anchor) Clamped(maxLine float32) Anchor {
	switch a.Kind {
	case KindLine:
		return Anchor{Kind: KindLine, Value: clampf(a.Value, 0, max(maxLine, 0))}
	default:
		return Anchor{Kind: KindRatio, Value: clampf(a.Value, 0, 1)}
	}
}

// SameKind reports whether two anchors can be compared at all: a line
// and a ratio cannot.
func (a Anchor) SameKind(other Anchor) bool { return a.Kind == other.Kind }

// Side is which of the two is speaking.
type Side int

const (
	Editor Side = iota
	Preview
)

// How long one side keeps the direction to itself, and how long a place
// it wrote still counts as its own echo.
const (
	// OwnerTimeout covers writing a scroll position and reading it back
	// the next frame, and no more: long enough to stop the two pushing
	// each other, short enough not to eat the user's next scroll.
	OwnerTimeout = 200 * time.Millisecond
	// EchoTimeout is how long a place the app wrote is read back as its
	// own. Writing is deferred and the landing place is adjusted, so
	// what comes back is near what was written, and not quite it.
	EchoTimeout = 500 * time.Millisecond
)

// Guard is the direction lock: the side that moved first owns the sync
// for a moment, and what the other side sees meanwhile is the echo of
// what was written, not the user scrolling back.
type Guard struct {
	owner      *Side
	since      time.Time
	expected   *Anchor
	expectedAt time.Time
}

// Claim asks whether side may drive the sync now: yes when nobody owns
// it, when it owns it already, or when the owner's turn has run out.
func (g *Guard) Claim(side Side, now time.Time) bool {
	expired := g.owner == nil || now.Sub(g.since) >= OwnerTimeout
	switch {
	case g.owner == nil, expired:
		g.owner, g.since = &side, now
		return true
	case *g.owner == side:
		g.since = now
		return true
	default:
		// The other side is driving: this change is its echo.
		return false
	}
}

// Expect remembers the place the app wrote, so that reading it back is
// known for what it is.
func (g *Guard) Expect(a Anchor) {
	g.expected, g.expectedAt = &a, time.Now()
}

// IsEcho reports whether the place now read is the one the app wrote,
// within tol: a tolerance the caller gives in the units of the anchor,
// as a pixel of the scroll is.
func (g *Guard) IsEcho(a Anchor, tol float32, now time.Time) bool {
	if g.expected == nil || now.Sub(g.expectedAt) > EchoTimeout {
		return false
	}
	if !g.expected.SameKind(a) {
		return false
	}
	return absf(g.expected.Value-a.Value) <= max(tol, 0)
}

// Reset forgets everything, as when another note is shown: without it
// the new note would be jumped to the old one's place.
func (g *Guard) Reset() {
	g.owner, g.expected = nil, nil
	g.since, g.expectedAt = time.Time{}, time.Time{}
}

// QuietPeriod is how long a gesture is left alone after the user's last
// scroll: the tail of one flick of a trackpad.
const QuietPeriod = 250 * time.Millisecond

// Gesture keeps a window open after the user scrolls, in which the app
// writes nothing: a scroll written over the user's own is a scroll that
// will not move, or that is pulled back.
type Gesture struct {
	quietUntil time.Time
}

// NoteUserMovement opens the window, and keeps it open while the user
// goes on scrolling.
func (g *Gesture) NoteUserMovement(now time.Time) {
	g.quietUntil = now.Add(QuietPeriod)
}

// IsQuiet reports whether the app should write nothing now.
func (g *Gesture) IsQuiet(now time.Time) bool {
	return !g.quietUntil.IsZero() && now.Before(g.quietUntil)
}

// Reset ends the window at once, as when another note is shown.
func (g *Gesture) Reset() { g.quietUntil = time.Time{} }

// Mode is how the two follow each other.
type Mode int

const (
	// Ratio follows how far down each is, and nothing else.
	Ratio Mode = iota
	// Anchor follows the block the note is at, through the table.
	AnchorMode
	// OnStop waits for the scrolling to stop.
	OnStop
	// StopQuiet is how long a scroll has to stand still in OnStop
	// before the other side is moved.
	StopQuiet = 120 * time.Millisecond
)

// ModeOf is the mode a name stands for; one that is not a mode is Ratio.
func ModeOf(name string) Mode {
	switch name {
	case "anchor":
		return AnchorMode
	case "on_stop":
		return OnStop
	default:
		return Ratio
	}
}

func clampf(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func absf(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
