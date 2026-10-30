package view

import (
	"strconv"

	"github.com/egoist/mygo/ui"
	xmd "github.com/helightxu/xinkgo/internal/markdown"
)

// How wide the drawer is, how deep it may grow, and how far a heading
// stands in for each level of it.
const (
	drawerWidth  float32 = 300
	drawerHeight float32 = 420
	drawerTop    float32 = 72
	railWidth    float32 = 34
	indentStep   float32 = 12
)

// Drawers are the panels the buttons on the rail open: the outline of
// the note, and later who points at it and what it is tagged with.
const drawerOutline = "outline"

// rail is the strip of buttons down the right of the note, and the
// drawer they open beside it. It floats over the window rather than
// taking a place in the layout: a panel that is open for a moment
// should not move the note the user is reading.
func (w *Workspace) rail(c *ui.Context) {
	if w.Tabs.ActiveNote() == nil {
		return
	}
	// The drawer stands beside the preview, or at the window's edge
	// when there is none.
	right := w.Layout.Right
	ui.Overlay(c, func() {
		ui.Row(c).Absolute().Right(right).Top(drawerTop).Children(func() {
			if w.drawer != "" {
				w.drawerPanel(c)
			}
			ui.Column(c).Width(railWidth).Children(func() {
				w.railButton(c, drawerOutline, "◫", "大纲")
			})
		})
	})
}

// railButton is one button of the rail: it opens its drawer, and closes
// it when pressed again, for a panel that is wanted for a moment is
// wanted to go away as easily.
func (w *Workspace) railButton(
	c *ui.Context, drawer, mark, label string,
) {
	button := ui.Button(c.Key("rail-"+drawer), mark).Label(label)
	if w.drawer == drawer {
		button.Selected(true)
	}
	if button.Clicked() {
		if w.drawer == drawer {
			w.drawer = ""
			return
		}
		w.drawer = drawer
	}
}

// drawerPanel is what the open drawer holds.
func (w *Workspace) drawerPanel(c *ui.Context) {
	ui.Column(c).Width(drawerWidth).ClipY().MaxHeight(
		drawerHeight).Children(func() {
		if w.drawer == drawerOutline {
			w.outline(c)
		}
	})
}

// outline is the headings of the note that is open, the deepest standing
// furthest in, and each of them a place to go to.
func (w *Workspace) outline(c *ui.Context) {
	n := w.Tabs.ActiveNote()
	if n == nil {
		return
	}
	headings := w.outlineOf(n)
	if len(headings) == 0 {
		ui.Text(c, "no headings in this note").TextColor(
			c.Theme().TextMuted)
		return
	}
	for _, h := range headings {
		h := h
		ui.Row(c).Children(func() {
			// The deeper a heading is, the further it stands in.
			ui.Box(c).Width(float32(h.Level-1) * indentStep)
			if ui.Button(c.Key("outline-"+n.Path+"#"+strconv.Itoa(h.Line)),
				h.Text).Clicked() {
				n.GoToLine(h.Line)
			}
		})
	}
}

// outlineOf is the headings of the note as it is now: read again when
// the text has changed, and not every time the window is drawn.
func (w *Workspace) outlineOf(n *OpenNote) []xmd.Heading {
	version := n.Editor.Buffer.Version()
	if w.outlinePath != n.Path || w.outlineVersion != version {
		w.outlineHeadings = xmd.ParseFacts(
			n.Editor.Buffer.String()).Headings
		w.outlinePath, w.outlineVersion = n.Path, version
	}
	return w.outlineHeadings
}
