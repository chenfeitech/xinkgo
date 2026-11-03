// This file is the geometry of the window: how wide the panels and the
// note stand, how far the text of a pane keeps from its edges, and where a
// line is broken. Nothing here draws: it is numbers, from the
// configuration and the layout the user left the window in. See
// docs/design/modularity.md.

package view

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// How wide a side panel may be dragged. Below the narrowest there is
// nothing to read in it; above the widest it would cover the note.
const (
	MinPanelWidth    float32 = 120
	MaxPanelWidth    float32 = 900
	dividerGrip      float32 = 5
	dividerThickness float32 = 1
	// What the editor keeps to its own text, whatever the room its pane
	// is given: four, and four more when there is no gutter beside the
	// lines, to the left of them, and the scroll bar's own room and four
	// to the right of them. They come off the padding the note is given,
	// so that it is the text, and not the editor, that stands that far
	// from the edges of the pane — and as far from one as from the
	// other, which is what the padding is for.
	editorInsetLeft  float32 = 8
	editorInsetRight float32 = 14
	// editorTrack is the room of the editor's own scroll bar, which it
	// draws over the last of what it is given while the pointer rests
	// on it, and not beside it: a line it breaks keeps clear of the bar
	// and of four DIPs besides, which is the editorInsetRight above,
	// but a line it does not break is cut at the very edge of what it
	// is given, bar and all.
	editorTrack float32 = 10
	// editorMinChars is how narrow the editor breaks a line: eight of
	// its own characters, and not a one of them less, however little
	// room it is given. A line broken that wide is written over the room
	// the editor keeps beside its lines and over the pane's own padding
	// besides, and is cut off at the very edge of the pane: so a pane
	// narrower than those together is a pane the note is not written in,
	// and none is ever made.
	editorMinChars = 8
	// monoCharRatio is how wide a character of the editor's font is, as
	// a share of the font's size: the editor measures it itself and does
	// not say what it measured, so it is said here, a little on the wide
	// side, so that the room kept is never less than the room needed.
	monoCharRatio float32 = 0.6
	// editorBarAir is the room the note's words keep between the last of
	// them and the editor's own scroll bar: the editor draws the bar over
	// the last of what it is given, four DIPs from a line it breaks, and
	// four DIPs is no room at all. The line is broken this much shorter
	// than the room there is — half as much again as it was, for the words
	// stood nearer the bar than a note reads well beside it — so that the
	// bar stands in the room the pane keeps beside its padding, and not
	// against the words.
	editorBarAir float32 = 15
	// editorRailAir is the room the note's words keep beside the rail's
	// buttons, on top of the padding, when its pane ends at the window's
	// own edge, as the note's does widened over the preview: the buttons
	// stand over that end of the pane, and half as much again as one of
	// them is wide — and half as much again as that — is no more than the
	// note needs to read well beside them.
	editorRailAir float32 = railWidth * 9 / 4
	// layoutSettle is how long the panels stay as they were dragged
	// before what the user chose is written down.
	layoutSettle = 400 * time.Millisecond
)

// editorPadX is how far the note's text stands from the sides of its
// pane, as the user chose.
func (w *Workspace) editorPadX() float32 {
	return float32(w.Config.Editor.PaddingX)
}

// editorPadLeft and editorPadRight are how much room the pane keeps
// beside the editor, which is not the same on the two sides: the editor
// keeps a room of its own beside its lines, wider on one side than on the
// other, and it is the text that is to stand the padding away from either
// edge. A pane given less padding than the editor keeps of its own keeps
// none: what is left is the editor's own room.
func (w *Workspace) editorPadLeft() float32 {
	return max(w.editorPadX()-editorInsetLeft, 0)
}

// editorPadRight is how much room the pane keeps on the right of the
// editor, which is not what it keeps on the left of it: a line the
// editor breaks is given the room the editor keeps beside it — its
// scroll bar's, and four — out of the width the editor is given, so
// that room comes off the padding the user chose. A line it does not
// break is cut at the very edge of what it is given, and the bar is
// drawn over the last of it: nothing of the editor's own is taken out
// of what the user chose, and the room of the bar is kept besides it,
// so that no word is written where the bar will stand.
func (w *Workspace) editorPadRight() float32 {
	if !w.Config.Editor.SoftWrap {
		return w.editorPadX() + editorTrack
	}
	return max(w.editorPadX()-editorInsetRight, 0)
}

// noteMinWidth is the narrowest the note's pane may be made: the room
// the pane keeps beside the editor on either side, and the room the
// editor keeps of its own — beside its lines, and for its scroll bar —
// and eight characters of the note besides, which is as narrow as the
// editor breaks a line. A pane made narrower breaks no line at all: the
// words are written over those rooms and cut off at the edge of the
// pane, padding and all.
func (w *Workspace) noteMinWidth() float32 {
	chars := editorMinChars * monoCharRatio * w.Config.Editor.Font.Size
	return w.editorPadLeft() + w.editorPadRight() +
		editorInsetLeft + editorInsetRight + chars
}

// wrapColumn is how many columns the note's text is broken at, when
// long lines are wrapped: as many as the room the editor is given holds,
// less the room its bar stands in and the air the words keep beside it.
// The editor breaks a line at the very room it is given, bar and all,
// whatever the pane keeps beside that: without a column to break at, the
// right-hand padding the user chose is a room the bar is drawn in, and
// the words stand four DIPs from it however wide the padding is.
func (w *Workspace) wrapColumn(c *ui.Context, n *OpenNote) int {
	room := w.editorWidth(c) - w.editorPadLeft() - w.editorPadRight()
	room -= editorInsetLeft + editorInsetRight + w.editorMarginRight()
	char := w.editorCharWidth(n)
	if char <= 0 {
		return 0
	}
	// A line shorter than eight characters is not broken at all, and the
	// editor breaks none narrower than that in any case.
	return max(int(room/char), editorMinChars)
}

// editorTheme is the theme the note is written in: the window's own, but
// for the color of the thumb of a scroll bar, which is painted over the
// last of the room the editor is given, and so over the words standing
// there. Nothing of the bar is painted over the note now, and what shows
// of it elsewhere — the note's pane beside it, the panes of the window —
// is as it was.
func (w *Workspace) editorTheme(c *ui.Context) *ui.Theme {
	t := *c.Theme()
	t.Scrollbar = ui.Transparent
	return &t
}

// editorMarginRight is the room the note's words keep from the right edge
// of its pane, over and above the padding: the padding is the same on
// either side of the note, and this is a margin of the note's own, asked
// for when a note reads better with more room on that side, as it does
// widened over the preview, with the rail standing at that end of it. A
// line is broken short of it — and of what the editor's own bar and the
// rail's buttons stand in besides, whichever of the two is the wider, for
// those stand at that end of the pane whether the margin is asked for or
// not.
func (w *Workspace) editorMarginRight() float32 {
	margin := editorBarAir
	if w.notePaneAtRail() {
		margin += editorRailAir
	}
	return max(margin, float32(w.Config.Editor.MarginRight))
}

// editorCharWidth is how wide a character of the note's font is: the
// editor measures it itself, of the nought, and keeps what it measured,
// so the nought is measured here as well. The note's font is a
// monospaced one, so every character of it stands as wide as the nought.
func (w *Workspace) editorCharWidth(n *OpenNote) float32 {
	wide, _ := ui.ShapeText("0", n.Editor.Font, 0).Size()
	return wide
}

// previewPadX is how far what the preview reads stands from the sides of
// its pane, and previewPadY from the top of it and the foot of it, as the
// user chose: the note is read from the top of the pane, and not from
// against the window's frame nor the line that parts it from the note's
// own.
func (w *Workspace) previewPadX() float32 {
	return float32(w.Config.Preview.PaddingX)
}

func (w *Workspace) previewPadY() float32 {
	return float32(w.Config.Preview.PaddingY)
}

// previewPaneWidth is how wide the preview stands: nothing when the note
// has been given the whole window, for then the preview is not drawn,
// and the rows that run the width of the window run under the rail; and
// what the note has not stretched over of it, once it has been given the
// preview's room.
func (w *Workspace) previewPaneWidth() float32 {
	if w.zoom == zoomEditor {
		return 0
	}
	// Spread over the note, the preview stands as wide as it was given:
	// what the note stretched over of it is the preview's own again.
	if w.zoom == zoomPreviewSpread {
		return w.Layout.Right
	}
	return w.Layout.Right * (1 - w.stretch)
}

// railBand is how much room the rail takes at the right edge of the
// window: its own, and its drawer's when one is open.
func (w *Workspace) railBand() float32 {
	band := railStrip
	if w.drawer != "" {
		band += drawerWidth + 2
	}
	return band
}

// notePaneAtRail is whether it is the note's pane that ends at the
// window's own right edge, which is the one the rail hangs over: it is
// when no preview stands beside the note, its room given over to the
// note or shut altogether — and when what stands there is narrower than
// the rail's strip, for the strip is then wider than the pane it stands
// beside, and the rest of it hangs over the note. That is the shape the
// note passes through as it is stretched over the preview: the preview
// grows narrower and narrower until it stands down, and for the last of
// the way it is narrower than the strip.
func (w *Workspace) notePaneAtRail() bool {
	return w.previewPaneWidth() < railStrip
}

// railClear is how much of the rail's room a row of the note's has to
// give up beside what its pane keeps already: the pane ends where the
// rail's strip begins, when it is the pane the rail hangs over, so what
// is left for the row is the drawer's own room, which covers what it
// covers. A row of a note with a preview beside it gives up nothing: the
// preview stands between it and the rail.
func (w *Workspace) railClear() float32 {
	clear := w.railBand() - w.previewPaneWidth()
	// What the row gives up is what is left of the band beside that:
	// when the note's pane is the one the rail hangs over, its pane
	// keeps the strip already, and what is left is the drawer's room.
	if w.notePaneAtRail() {
		clear -= railStrip
	}
	if clear < 0 {
		return 0
	}
	return clear
}

// markdownWidth is how wide the row of the Markdown buttons may be: what
// the panels leave, less the room the way to the model takes beside it, so
// that it stands beside the row and not under the rail.
func (w *Workspace) markdownWidth(c *ui.Context) float32 {
	room := w.editorWidth(c) - aiBand
	// Narrower than a few buttons there is nothing to show but the menu
	// at the end, which is still worth its place.
	if room < iconSize*3 {
		return iconSize * 3
	}
	return room
}

// editorWidth is how wide the note is, which is how wide its toolbar may
// be: what the panels on either side leave of the window, and what the
// rail and its drawer leave of it besides.
func (w *Workspace) editorWidth(c *ui.Context) float32 {
	width, _ := c.Size()
	// What the pane keeps at the window's right edge: the rail's strip,
	// when the note is the pane the rail hangs over, and the drawer's
	// room on top of it while one stands open beside the rail.
	clear := w.railClear()
	if w.notePaneAtRail() {
		clear += railStrip
	}
	// Given the whole window, the note is as wide as it, less that.
	if w.zoom == zoomEditor {
		return max(width-clear, 0)
	}
	// Less the panels on either side, and the lines that part them from
	// the note: a button of the row is to stand over the note, and not
	// past the end of it.
	side := w.Layout.Left + w.previewPaneWidth()
	if w.Layout.Left > 0 {
		side += dividerGrip
	}
	if w.previewPaneWidth() >= 1 {
		side += dividerGrip
	}
	return max(width-side-clear, 0)
}
