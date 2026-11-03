package view

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/helightxu/xinkgo/internal/sync"
)

// How much of a scroll counts as the user moving: less than a pixel is
// the app's own rounding.
const scrollMoved = 0.5

// scrollFollow is what the note and the preview follow each other by: the
// side the user scrolls drives, and the other is brought to it, once the
// user's hand is off it and what they moved has stood still a moment. What
// the two are lined up by is a table of anchors, made for the note on show.
type scrollFollow struct {
	// Preview is the scroll of the preview, tracked by its own view.
	Preview ui.ScrollState
	// Guard keeps the two from pushing each other back and forth: the
	// side the user moved keeps the direction for a moment, and what
	// the app wrote is known again for what it is when it is read back.
	Guard sync.Guard
	// Ed and Pr are how long each side is left to the user after they
	// scrolled it; Side is the side they moved, whose other side has
	// yet to be brought over.
	Ed   sync.Gesture
	Pr   sync.Gesture
	Side *sync.Side
	// LastEdY and LastPrY are where the two stood when last read, and
	// LastMove when the user last moved either of them.
	LastEdY  float32
	LastPrY  float32
	LastMove time.Time
	// Table is the anchors the two are lined up by, TablePath and
	// TableVersion the note and the revision of it they were made for.
	Table        sync.AnchorTable
	TablePath    string
	TableVersion uint64
	// ResetPath is the note the scroll was last put back to the top
	// for.
	ResetPath string
}

// followScroll keeps the note and what it looks like at the same place.
// The side the user scrolls drives, and the other is brought to it: the
// side under the user's hand is theirs alone, and the app writes nothing
// to it, but there is no cause to wait before moving the other, so the
// two follow each other as the user goes, and once more when they stop.
//
// A lock keeps the two from pushing each other back and forth: the side
// the user moved keeps the direction for a moment, and what the app
// wrote is known again for what it is when it is read back.
func (w *Workspace) followScroll(c *ui.Context) {
	n := w.Tabs.ActiveNote()
	// The preview is not there to follow the note when the note has been
	// given the whole window: it is not drawn, and what it would say is
	// not said.
	if n == nil || w.geometry().PreviewPaneWidth() <= 0 ||
		!w.Config.Preview.SyncScroll {
		return
	}
	now := c.Now()
	ed := n.Editor
	edY, prY := float32(ed.ScrollY()), w.scroll.Preview.Y
	edMoved, prMoved := moved(edY, w.scroll.LastEdY), moved(prY, w.scroll.LastPrY)
	w.scroll.LastEdY, w.scroll.LastPrY = edY, prY
	// A move is the user's unless it is the app's own, read back.
	if edMoved && !w.scroll.Guard.IsEcho(sync.LineAnchor(topLine(ed)),
		lineTolerance(ed), now) {
		w.userScrolled(sync.Editor, now)
	}
	if prMoved && !w.scroll.Guard.IsEcho(w.previewAnchor(),
		w.previewTolerance(), now) {
		w.userScrolled(sync.Preview, now)
	}
	if w.scroll.Side == nil {
		return
	}
	// Following on stop: the other side is moved once the scrolling has
	// stood still for a moment, and not under the user's hand.
	if sync.ModeOf(w.Config.Preview.SyncMode) == sync.OnStop &&
		now.Sub(w.scroll.LastMove) < sync.StopQuiet {
		return
	}
	// What the app writes, it writes to the other side, and only while
	// the user's hand is not on that one either: a side they are
	// scrolling themselves is left as they leave it.
	if w.gestureOf(theOther(*w.scroll.Side)).IsQuiet(now) {
		return
	}
	table := w.anchorTable(n)
	switch *w.scroll.Side {
	case sync.Editor:
		if !w.scroll.Guard.Claim(sync.Editor, now) {
			return
		}
		w.toPreview(ed, table, topLine(ed), now)
	case sync.Preview:
		if !w.scroll.Guard.Claim(sync.Preview, now) {
			return
		}
		w.toEditor(ed, table, w.previewRatio(), now)
	}
	// The other side has been brought over: nothing is owed it until
	// the user moves again.
	w.scroll.Side = nil
}

// userScrolled is the user moving one side: that side is theirs for a
// moment, and the other owes them the place they left it at.
func (w *Workspace) userScrolled(side sync.Side, now time.Time) {
	w.gestureOf(side).NoteUserMovement(now)
	w.scroll.LastMove = now
	w.scroll.Side = &side
}

// gestureOf is the gesture of one side: how long the user's scrolling of
// it keeps the app from writing to it.
func (w *Workspace) gestureOf(side sync.Side) *sync.Gesture {
	if side == sync.Editor {
		return &w.scroll.Ed
	}
	return &w.scroll.Pr
}

// theOther is the side that is not the one given.
func theOther(side sync.Side) sync.Side {
	if side == sync.Editor {
		return sync.Preview
	}
	return sync.Editor
}

// toPreview moves the preview to where the editor is, and remembers the
// place it was sent to, so that seeing it come back is known for an echo.
func (w *Workspace) toPreview(
	ed *editor.State, table sync.AnchorTable, line float32, now time.Time,
) {
	ratio := w.previewRatio()
	if !table.IsEmpty() && sync.ModeOf(w.Config.Preview.SyncMode) ==
		sync.AnchorMode {
		// Through the blocks: the block the line is in, and how far
		// into it, rather than how far down the note is.
		ix, frac := table.Locate(line)
		if n := table.Len(); n > 0 {
			ratio = (float32(ix) + frac) / float32(n)
		}
	} else {
		ratio = line / float32(table.TotalLines())
	}
	ratio = sync.RatioAnchor(ratio).Clamped(1).Value
	w.scroll.Preview.Y = ratio * w.scroll.Preview.MaxY
	w.scroll.Guard.Expect(sync.RatioAnchor(ratio), now)
}

// toEditor moves the editor to where the preview is, the other way round.
func (w *Workspace) toEditor(
	ed *editor.State, table sync.AnchorTable, ratio float32, now time.Time,
) {
	line := ratio * float32(table.TotalLines())
	if !table.IsEmpty() && sync.ModeOf(w.Config.Preview.SyncMode) ==
		sync.AnchorMode {
		if n := table.Len(); n > 0 {
			p := ratio * float32(n)
			ix := int(p)
			line = table.LineOf(ix, p-float32(ix))
		}
	}
	line = sync.LineAnchor(line).
		Clamped(float32(table.TotalLines())).Value
	ed.SetScrollY(lineTop(ed, line))
	w.scroll.Guard.Expect(sync.LineAnchor(line), now)
}

// previewRatio is how far down the preview is scrolled, 0 to 1.
func (w *Workspace) previewRatio() float32 {
	if w.scroll.Preview.MaxY <= 0 {
		return 0
	}
	return w.scroll.Preview.Y / w.scroll.Preview.MaxY
}

// previewAnchor is the place the preview is at, as the two sides pass
// them to each other.
func (w *Workspace) previewAnchor() sync.Anchor {
	return sync.RatioAnchor(w.previewRatio())
}

// previewTolerance is how near the place that comes back has to be to
// count as the app's own: two pixels of the preview.
func (w *Workspace) previewTolerance() float32 {
	if w.scroll.Preview.MaxY <= 0 {
		return 0
	}
	return 2 / w.scroll.Preview.MaxY
}

// anchorTable is the blocks of the note the preview drew, made again
// when the note's text has changed.
func (w *Workspace) anchorTable(n *OpenNote) sync.AnchorTable {
	version := n.Editor.Buffer.Version()
	if w.scroll.TablePath != n.Path || w.scroll.TableVersion != version {
		text := n.Editor.Buffer.String()
		w.scroll.Table = sync.NewAnchorTable(text, n.Editor.Buffer.LineCount())
		w.scroll.TablePath, w.scroll.TableVersion = n.Path, version
	}
	return w.scroll.Table
}

// resetScroll forgets what the two sides knew, as when another note is
// shown: without it the new one is jumped to the old one's place.
func (w *Workspace) resetScroll(path string) {
	if w.scroll.ResetPath == path {
		return
	}
	w.scroll.ResetPath = path
	w.scroll.Guard.Reset()
	w.scroll.Ed.Reset()
	w.scroll.Pr.Reset()
	w.scroll.Side = nil
	w.scroll.LastEdY, w.scroll.LastPrY = 0, 0
	w.scroll.LastMove = time.Time{}
}

// moved reports whether a scroll position is different from what it was,
// by more than the app's own rounding.
func moved(now, before float32) bool {
	if now < before {
		return before-now > scrollMoved
	}
	return now-before > scrollMoved
}

// topLine is the line of the note at the top of what the editor shows:
// a line and a part of one, as the scroll does not stop at a line.
func topLine(ed *editor.State) float32 {
	y := ed.ScrollY()
	last := ed.Buffer.LineCount() - 1
	if last < 0 {
		return 0
	}
	// The last line whose top is at or above where the editor shows.
	lo, hi := 0, last
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if lineTop(ed, float32(mid)) <= y {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	top := lineTop(ed, float32(lo))
	next := lineTop(ed, float32(lo+1))
	if next <= top {
		return float32(lo)
	}
	return float32(lo) + float32((y-top)/(next-top))
}

// lineTop is where a line of the note begins in the editor, a part of a
// line between two of them.
func lineTop(ed *editor.State, line float32) float64 {
	whole := int(line)
	top := ed.LineTop(whole)
	if whole+1 > ed.Buffer.LineCount() {
		return top
	}
	next := ed.LineTop(whole + 1)
	if next <= top {
		return top
	}
	return top + (next-top)*float64(line-float32(whole))
}

// lineTolerance is how near a line that comes back has to be to count as
// the app's own: two pixels of the editor.
func lineTolerance(ed *editor.State) float32 {
	top, next := ed.LineTop(0), ed.LineTop(1)
	if next <= top {
		return 0
	}
	return 2 / float32(next-top)
}
