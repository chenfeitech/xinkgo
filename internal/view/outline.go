package view

import (
	"strconv"

	"github.com/egoist/mygo/ui"
	xmd "github.com/helightxu/xinkgo/internal/markdown"
)

// How wide the drawer is, where it hangs, and how far a heading stands
// in for each level of it. Its height is not one of them: it is as tall
// as the panels it stands over.
const (
	drawerWidth float32 = 300
	drawerTop   float32 = 72
	drawerFoot  float32 = 6
	railWidth   float32 = 34
	railGap     float32 = 6
	indentStep  float32 = 12
)

// Drawers are the panels the buttons on the rail open: the outline of
// the note, and later who points at it and what it is tagged with.
const (
	drawerOutline = "outline"
	drawerLinks   = "backlinks"
	drawerTags    = "tags"
)

// rail is the strip of buttons down the right of the window, and the
// drawer they open beside them. It floats over the panels rather than
// taking a place in the layout: a panel that is open for a moment
// should not move the note the user is reading.
func (w *Workspace) rail(c *ui.Context) {
	if w.Tabs.ActiveNote() == nil {
		return
	}
	// The rail belongs to the window, and not to the note: it stands at
	// the window right edge, over whatever panel is there, and so keeps
	// its place when the panels are dragged wider or shut. Its buttons
	// keep to the top of it: a drawer as tall as the window is must not
	// push them down the note.
	ui.Row(c).Absolute().Right(railGap).Top(drawerTop).Bottom(drawerFoot).
		AlignItems(ui.Start).Children(func() {
		if w.drawer != "" {
			w.drawerPanel(c)
		}
		ui.Column(c).Width(railWidth).Children(func() {
			w.railButton(c, drawerOutline, "◫", "大纲")
			w.railButton(c, drawerLinks, "⇄", "反向链接")
			w.railButton(c, drawerTags, "＃", "标签")
		})
	})
}

// railButton is one button of the rail: it opens its drawer, and closes
// it when pressed again, for a panel that is wanted for a moment is
// wanted to go away as easily.
func (w *Workspace) railButton(
	c *ui.Context, drawer, mark, label string,
) {
	button := ui.Button(c.Key("rail-"+drawer), mark).
		Label(label).Tooltip(label)
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

// drawerPanel is what the open drawer holds: a panel of the theme's own
// look, as tall as the panels it stands over, so that what it says is
// read over the note and not mixed into the note it is over. What it
// says is read from its first line down, and scrolled when it says more
// than there is room for, as the note beside it is.
func (w *Workspace) drawerPanel(c *ui.Context) {
	th := c.Theme()
	ui.Column(c).Width(drawerWidth).FillHeight().ClipY().
		Padding(th.Space(2)).Background(th.Background).
		Border(1, th.Border).Radius(th.Radius+2).
		Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, 0.18)).Children(func() {
		ui.Scroll(c).Fill().Children(func() {
			ui.Column(c).FillWidth().Children(func() {
				switch w.drawer {
				case drawerOutline:
					w.outline(c)
				case drawerLinks:
					w.backlinks(c)
				case drawerTags:
					w.tags(c)
				}
			})
		})
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
