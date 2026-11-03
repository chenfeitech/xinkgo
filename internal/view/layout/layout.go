// Package layout is the geometry of the window: how wide the panels and
// the note stand, and how far the text of a pane keeps from its edges.
//
// It draws nothing and reads no context. The window makes a Geometry from
// the configuration, the widths the user left the panels at and the shape
// the window is in, and the panes ask it as they draw. Keeping the numbers
// here keeps them out of the drawing code, and lets them be checked
// without a window to draw in.
package layout

import (
	"github.com/egoist/mygo/ui"

	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/state"
)

// Zoom is which pane has been given room that is not its own. ZoomNone is
// the usual, in which every panel keeps the room the user gave it.
type Zoom int

const (
	ZoomNone Zoom = iota
	// ZoomEditor is the note's: the tree and the preview are not drawn,
	// and the note is as wide as the window.
	ZoomEditor
	// ZoomPreview is the preview's, for reading: the note's own pane is
	// not drawn, and the preview is as wide as the window.
	ZoomPreview
	// ZoomPreviewSpread is the preview's over the note alone: the preview
	// reaches left as far as the tree, and the tree keeps the room it was
	// given, so that another note may be chosen in it while one is read.
	ZoomPreviewSpread
)

// How wide a side panel may be dragged. Below the narrowest there is
// nothing to read in it; above the widest it would cover the note.
const (
	MinPanelWidth    float32 = 120
	MaxPanelWidth    float32 = 900
	DividerGrip      float32 = 5
	DividerThickness float32 = 1
)

// What the editor keeps of its own, whatever the room its pane is given.
const (
	// EditorInsetLeft and EditorInsetRight: four, and four more when
	// there is no gutter beside the lines, to the left of them, and the
	// scroll bar's own room and four to the right of them. They come off
	// the padding the note is given, so that it is the text, and not the
	// editor, that stands that far from the edges of the pane — and as
	// far from one as from the other, which is what the padding is for.
	EditorInsetLeft  float32 = 8
	EditorInsetRight float32 = 14
	// EditorTrack is the room of the editor's own scroll bar, which it
	// draws over the last of what it is given while the pointer rests on
	// it, and not beside it: a line it breaks keeps clear of the bar and
	// of four DIPs besides, which is EditorInsetRight above, but a line
	// it does not break is cut at the very edge of what it is given, bar
	// and all.
	EditorTrack float32 = 10
	// EditorMinChars is how narrow the editor breaks a line: eight of its
	// own characters, and not a one of them less, however little room it
	// is given. A line broken that wide is written over the room the
	// editor keeps beside its lines and over the pane's own padding
	// besides, and is cut off at the very edge of the pane.
	EditorMinChars = 8
	// MonoCharRatio is how wide a character of the editor's font is, as a
	// share of the font's size: the editor measures it itself and does
	// not say what it measured, so it is said here, a little on the wide
	// side, so that the room kept is never less than the room needed.
	MonoCharRatio float32 = 0.6
	// EditorBarAir is the room the note's words keep between the last of
	// them and the editor's own scroll bar: the editor draws the bar over
	// the last of what it is given, four DIPs from a line it breaks, and
	// four DIPs is no room at all. The line is broken this much shorter
	// than the room there is, so that the bar stands in the room the pane
	// keeps beside its padding, and not against the words.
	EditorBarAir float32 = 15
)

// The rail, and the drawer beside it: the strip of buttons no pane writes
// in, for they have no background of their own and what is written under
// them is lost, and the panel that stands beside them and covers what it
// covers.
const (
	RailWidth float32 = 34
	RailGap   float32 = 6
	// RailLead is the air a pane keeps between what it writes and the
	// rail's buttons: they stand RailGap in from the window's edge, and
	// what is written hard against them is not read beside them but under
	// them, which is no plainer than writing on them.
	RailLead float32 = 8
	// RailStrip is the room the rail's own buttons take at the right edge
	// of the window.
	RailStrip = RailWidth + RailGap + RailLead
	// DrawerWidth is how wide the drawer beside the rail is.
	DrawerWidth float32 = 300
	// EditorRailAir is the room the note's words keep beside the rail's
	// buttons, on top of the padding, when its pane ends at the window's
	// own edge, as the note's does widened over the preview: the buttons
	// stand over that end of the pane, and half as much again as one of
	// them is wide — and half as much again as that — is no more than the
	// note needs to read well beside them.
	EditorRailAir float32 = RailWidth * 9 / 4
)

// Shape is the form the window is in, as far as the geometry cares: which
// pane has been given room that is not its own, how far the note has gone
// over the preview's room, and whether a drawer stands beside the rail.
type Shape struct {
	Zoom Zoom
	// Stretch is how far the note has gone over the preview's room in
	// the frame being drawn, 0 to 1.
	Stretch float32
	Drawer  bool
}

// Geometry is the geometry of one window: made from the configuration, the
// widths the user left the panels at and the shape it is in, it answers
// every question of room the panes ask as they draw. It is a value: made
// afresh whenever it is asked for, so that it cannot be behind the window.
type Geometry struct {
	Config *config.AppConfig
	Layout state.Layout
	Shape  Shape
}

// Of makes the geometry of a window.
func Of(cfg *config.AppConfig, lay state.Layout, shape Shape) Geometry {
	return Geometry{Config: cfg, Layout: lay, Shape: shape}
}

// EditorPadX is how far the note's text stands from the sides of its pane,
// as the user chose.
func (g Geometry) EditorPadX() float32 {
	return float32(g.Config.Editor.PaddingX)
}

// EditorPadLeft and EditorPadRight are how much room the pane keeps beside
// the editor, which is not the same on the two sides: the editor keeps a
// room of its own beside its lines, wider on one side than on the other,
// and it is the text that is to stand the padding away from either edge. A
// pane given less padding than the editor keeps of its own keeps none:
// what is left is the editor's own room.
func (g Geometry) EditorPadLeft() float32 {
	return max(g.EditorPadX()-EditorInsetLeft, 0)
}

// EditorPadRight is how much room the pane keeps on the right of the
// editor, which is not what it keeps on the left of it: a line the editor
// breaks is given the room the editor keeps beside it — its scroll bar's,
// and four — out of the width the editor is given, so that room comes off
// the padding the user chose. A line it does not break is cut at the very
// edge of what it is given, and the bar is drawn over the last of it:
// nothing of the editor's own is taken out of what the user chose, and the
// room of the bar is kept besides it, so that no word is written where the
// bar will stand.
func (g Geometry) EditorPadRight() float32 {
	if !g.Config.Editor.SoftWrap {
		return g.EditorPadX() + EditorTrack
	}
	return max(g.EditorPadX()-EditorInsetRight, 0)
}

// EditorMarginRight is the room the note's words keep from the right edge
// of its pane, over and above the padding: the padding is the same on
// either side of the note, and this is a margin of the note's own, asked
// for when a note reads better with more room on that side, as it does
// widened over the preview, with the rail standing at that end of it. A
// line is broken short of it — and of what the editor's own bar and the
// rail's buttons stand in besides, whichever of the two is the wider, for
// those stand at that end of the pane whether the margin is asked for or
// not.
func (g Geometry) EditorMarginRight() float32 {
	margin := EditorBarAir
	if g.NotePaneAtRail() {
		margin += EditorRailAir
	}
	return max(margin, float32(g.Config.Editor.MarginRight))
}

// NoteMinWidth is the narrowest the note's pane may be made: the room the
// pane keeps beside the editor on either side, and the room the editor
// keeps of its own — beside its lines, and for its scroll bar — and eight
// characters of the note besides, which is as narrow as the editor breaks
// a line. A pane made narrower breaks no line at all: the words are
// written over those rooms and cut off at the edge of the pane, padding
// and all.
func (g Geometry) NoteMinWidth() float32 {
	chars := EditorMinChars * MonoCharRatio * g.Config.Editor.Font.Size
	return g.EditorPadLeft() + g.EditorPadRight() +
		EditorInsetLeft + EditorInsetRight + chars
}

// WrapColumn is how many columns the note's text is broken at, when long
// lines are wrapped: as many as the room the editor is given holds, less
// the room its bar stands in and the air the words keep beside it. The
// editor breaks a line at the very room it is given, bar and all, whatever
// the pane keeps beside that: without a column to break at, the
// right-hand padding the user chose is a room the bar is drawn in, and the
// words stand four DIPs from it however wide the padding is.
func (g Geometry) WrapColumn(win, char float32) int {
	room := g.EditorWidth(win) - g.EditorPadLeft() - g.EditorPadRight()
	room -= EditorInsetLeft + EditorInsetRight + g.EditorMarginRight()
	if char <= 0 {
		return 0
	}
	// A line shorter than eight characters is not broken at all, and the
	// editor breaks none narrower than that in any case.
	return max(int(room/char), EditorMinChars)
}

// CharWidth is how wide a character of a font is: the editor measures it
// itself, of the nought, and keeps what it measured, so the nought is
// measured here as well. The note's font is a monospaced one, so every
// character of it stands as wide as the nought.
func CharWidth(font ui.Font) float32 {
	wide, _ := ui.ShapeText("0", font, 0).Size()
	return wide
}

// PreviewPadX is how far what the preview reads stands from the sides of
// its pane, and PreviewPadY from the top of it and the foot of it, as the
// user chose: the note is read from the top of the pane, and not from
// against the window's frame nor the line that parts it from the note's
// own.
func (g Geometry) PreviewPadX() float32 {
	return float32(g.Config.Preview.PaddingX)
}

// PreviewPadY is how far what the preview reads stands from the top of its
// pane and the foot of it.
func (g Geometry) PreviewPadY() float32 {
	return float32(g.Config.Preview.PaddingY)
}

// PreviewPaneWidth is how wide the preview stands: nothing when the note
// has been given the whole window, for then the preview is not drawn, and
// the rows that run the width of the window run under the rail; and what
// the note has not stretched over of it, once it has been given the
// preview's room.
func (g Geometry) PreviewPaneWidth() float32 {
	if g.Shape.Zoom == ZoomEditor {
		return 0
	}
	// Spread over the note, the preview stands as wide as it was given:
	// what the note stretched over of it is the preview's own again.
	if g.Shape.Zoom == ZoomPreviewSpread {
		return g.Layout.Right
	}
	return g.Layout.Right * (1 - g.Shape.Stretch)
}

// RailBand is how much room the rail takes at the right edge of the
// window: its own, and its drawer's when one is open.
func (g Geometry) RailBand() float32 {
	band := float32(RailStrip)
	if g.Shape.Drawer {
		band += DrawerWidth + 2
	}
	return band
}

// NotePaneAtRail is whether it is the note's pane that ends at the
// window's own right edge, which is the one the rail hangs over: it is
// when no preview stands beside the note, its room given over to the note
// or shut altogether — and when what stands there is narrower than the
// rail's strip, for the strip is then wider than the pane it stands
// beside, and the rest of it hangs over the note. That is the shape the
// note passes through as it is stretched over the preview: the preview
// grows narrower and narrower until it stands down, and for the last of
// the way it is narrower than the strip.
func (g Geometry) NotePaneAtRail() bool {
	return g.PreviewPaneWidth() < RailStrip
}

// RailClear is how much of the rail's room a row of the note's has to give
// up beside what its pane keeps already: the pane ends where the rail's
// strip begins, when it is the pane the rail hangs over, so what is left
// for the row is the drawer's own room, which covers what it covers. A row
// of a note with a preview beside it gives up nothing: the preview stands
// between it and the rail.
func (g Geometry) RailClear() float32 {
	clear := g.RailBand() - g.PreviewPaneWidth()
	// What the row gives up is what is left of the band beside that:
	// when the note's pane is the one the rail hangs over, its pane
	// keeps the strip already, and what is left is the drawer's room.
	if g.NotePaneAtRail() {
		clear -= RailStrip
	}
	if clear < 0 {
		return 0
	}
	return clear
}

// EditorWidth is how wide the note is, which is how wide its toolbar may
// be: what the panels on either side leave of the window, and what the
// rail and its drawer leave of it besides.
func (g Geometry) EditorWidth(win float32) float32 {
	// What the pane keeps at the window's right edge: the rail's strip,
	// when the note is the pane the rail hangs over, and the drawer's
	// room on top of it while one stands open beside the rail.
	clear := g.RailClear()
	if g.NotePaneAtRail() {
		clear += RailStrip
	}
	// Given the whole window, the note is as wide as it, less that.
	if g.Shape.Zoom == ZoomEditor {
		return max(win-clear, 0)
	}
	// Less the panels on either side, and the lines that part them from
	// the note: a button of the row is to stand over the note, and not
	// past the end of it.
	side := g.Layout.Left + g.PreviewPaneWidth()
	if g.Layout.Left > 0 {
		side += DividerGrip
	}
	if g.PreviewPaneWidth() >= 1 {
		side += DividerGrip
	}
	return max(win-side-clear, 0)
}

// PanelWidthDragged is how wide a drag may make a panel: what the drag
// asks for, but never so much that the note's pane is left with less than
// the note needs, once the other panel and the lines that part them have
// taken theirs. A window with no room to spare gives the panel its
// narrowest all the same: what is then too wide is cut off by the
// window's own edge, and the note is not.
func (g Geometry) PanelWidthDragged(win float32, other, v float32) float32 {
	room := win - other - g.NoteMinWidth() - 2*DividerGrip
	return min(v, max(room, MinPanelWidth))
}
