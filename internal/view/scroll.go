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

// followScroll keeps the note and what it looks like at the same place.
// The side the user moved drives, and the other is moved to it: a moment
// of quiet is left after a gesture, and a lock keeps the two from
// pushing each other back and forth.
func (w *Workspace) followScroll(c *ui.Context) {
	n := w.Tabs.ActiveNote()
	if n == nil || w.Layout.Right <= 0 || !w.Config.Preview.SyncScroll {
		return
	}
	now := time.Now()
	ed := n.Editor
	edY, prY := float32(ed.ScrollY()), w.previewScroll.Y
	edMoved, prMoved := moved(edY, w.lastEdY), moved(prY, w.lastPrY)
	w.lastEdY, w.lastPrY = edY, prY
	if edMoved || prMoved {
		w.gesture.NoteUserMovement(now)
		w.lastMove = now
	}
	if w.gesture.IsQuiet(now) {
		return
	}
	// Waiting for the scrolling to stop: don't move the other side
	// while this one is still going.
	if sync.ModeOf(w.Config.Preview.SyncMode) == sync.OnStop &&
		now.Sub(w.lastMove) < sync.StopQuiet {
		return
	}
	table := w.anchorTable(n)
	switch {
	case edMoved && !w.guard.IsEcho(sync.LineAnchor(topLine(ed)),
		lineTolerance(ed), now) && w.guard.Claim(sync.Editor, now):
		w.toPreview(ed, table, topLine(ed))
	case prMoved && !w.guard.IsEcho(w.previewAnchor(), w.previewTolerance(),
		now) && w.guard.Claim(sync.Preview, now):
		w.toEditor(ed, table, w.previewRatio())
	}
}

// toPreview moves the preview to where the editor is, and remembers the
// place it was sent to, so that seeing it come back is known for an echo.
func (w *Workspace) toPreview(
	ed *editor.State, table sync.AnchorTable, line float32,
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
	w.previewScroll.Y = ratio * w.previewScroll.MaxY
	w.guard.Expect(sync.RatioAnchor(ratio))
}

// toEditor moves the editor to where the preview is, the other way round.
func (w *Workspace) toEditor(
	ed *editor.State, table sync.AnchorTable, ratio float32,
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
	w.guard.Expect(sync.LineAnchor(line))
}

// previewRatio is how far down the preview is scrolled, 0 to 1.
func (w *Workspace) previewRatio() float32 {
	if w.previewScroll.MaxY <= 0 {
		return 0
	}
	return w.previewScroll.Y / w.previewScroll.MaxY
}

// previewAnchor is the place the preview is at, as the two sides pass
// them to each other.
func (w *Workspace) previewAnchor() sync.Anchor {
	return sync.RatioAnchor(w.previewRatio())
}

// previewTolerance is how near the place that comes back has to be to
// count as the app's own: two pixels of the preview.
func (w *Workspace) previewTolerance() float32 {
	if w.previewScroll.MaxY <= 0 {
		return 0
	}
	return 2 / w.previewScroll.MaxY
}

// anchorTable is the blocks of the note the preview drew, made again
// when the note's text has changed.
func (w *Workspace) anchorTable(n *OpenNote) sync.AnchorTable {
	version := n.Editor.Buffer.Version()
	if w.tablePath != n.Path || w.tableVersion != version {
		text := n.Editor.Buffer.String()
		w.table = sync.NewAnchorTable(text, n.Editor.Buffer.LineCount())
		w.tablePath, w.tableVersion = n.Path, version
	}
	return w.table
}

// resetScroll forgets what the two sides knew, as when another note is
// shown: without it the new one is jumped to the old one's place.
func (w *Workspace) resetScroll(path string) {
	if w.scrollPath == path {
		return
	}
	w.scrollPath = path
	w.guard.Reset()
	w.gesture.Reset()
	w.lastEdY, w.lastPrY = 0, 0
	w.lastMove = time.Time{}
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
