package view

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/helightxu/xinkgo/internal/locale"
	xmd "github.com/helightxu/xinkgo/internal/markdown"
)

// How wide the drawer is, and where it hangs. Its height is not one of
// them: it is as tall as the panels it stands over. It hangs railGap in
// from the top, the right and the foot of them, as the preview begins at
// the top of the window and not lower down, and its buttons hang from
// the top of it.
const (
	drawerWidth float32 = 300
	drawerTop           = railGap
	drawerFoot          = railGap
	railWidth   float32 = 34
	railGap     float32 = 6
)

// How a link in the drawer says which line of which note it goes to, and
// which tag it opens. Each is a scheme of its own, so that a click on it
// is known for what it is, and is not taken for a page on the web nor
// for a note of the vault.
const (
	drawerLineScheme = "xinkgo-line:"
	drawerTagScheme  = "xinkgo-tag:"
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
	// its place when the panels are dragged wider or shut. It hangs from
	// the top of the window, where the preview begins, and its buttons
	// under it: a drawer as tall as the window is must not push them
	// down the note.
	ui.Row(c).Absolute().Right(railGap).Top(drawerTop).Bottom(drawerFoot).
		AlignItems(ui.Start).Children(func() {
		if w.drawer != "" {
			w.drawerPanel(c)
		}
		ui.Column(c).Width(railWidth).Children(func() {
			w.railButton(c, drawerOutline, "◫",
				locale.DrawerOutlineLabel())
			w.railButton(c, drawerLinks, "⇄",
				locale.DrawerLinksLabel())
			w.railButton(c, drawerTags, "＃", locale.DrawerTagsLabel())
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
// furthest in, and each of them a place to go to. It is written as
// Markdown, as the note beside it is read: a list of links, and not a
// column of buttons.
func (w *Workspace) outline(c *ui.Context) {
	n := w.Tabs.ActiveNote()
	if n == nil {
		return
	}
	headings := w.outlineOf(n)
	if len(headings) == 0 {
		ui.Text(c, locale.OutlineNoHeadings()).TextColor(
			c.Theme().TextMuted)
		return
	}
	var b strings.Builder
	for _, h := range headings {
		// The deeper a heading is, the further it stands in: two spaces
		// for each level, which is what nests a list.
		b.WriteString(strings.Repeat("  ", max(h.Level-1, 0)))
		b.WriteString("- [")
		b.WriteString(mdPlain(h.Text))
		b.WriteString("](")
		b.WriteString(drawerLineHref("", h.Line))
		b.WriteString(")\n")
	}
	w.drawerMarkdown(c, b.String())
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

// drawerMarkdown is what the drawer says, read as Markdown, as the note
// beside it is read: a list of links, and not a column of buttons.
func (w *Workspace) drawerMarkdown(c *ui.Context, text string) {
	from := ""
	if n := w.Tabs.ActiveNote(); n != nil {
		from = n.Path
	}
	opts := &markdown.Options{
		Theme: w.previewTheme(c),
		// A link of the drawer's own is not one of the note's: it is
		// told apart here, and the rest is left to the preview's way
		// with a link.
		OnLink: func(url string) { w.drawerLink(url, from) },
	}
	markdown.View(c, text, opts)
}

// drawerLink is a click on a link in the drawer: one of the drawer's own
// goes where it says, and any other is one as the preview takes them.
func (w *Workspace) drawerLink(url, from string) {
	if path, line, ok := drawerLineTarget(url); ok {
		if path != "" {
			w.openNote(path)
		}
		if n := w.Tabs.ActiveNote(); n != nil && line > 0 &&
			(path == "" || n.Path == path) {
			n.GoToLine(line)
		}
		return
	}
	if name, ok := drawerTagTarget(url); ok {
		w.tagOpen = name
		return
	}
	w.openLink(url, from)
}

// drawerLineHref is where a link to a line of a note points, for the
// drawer: which note, and which line of it. A link into the note that is
// open names no note of its own.
func drawerLineHref(path string, line int) string {
	u := &url.URL{Path: path}
	if line > 0 {
		u.Fragment = strconv.Itoa(line)
	}
	return drawerLineScheme + u.String()
}

// drawerLineTarget is what such a destination names: the note, empty for
// the note that is open, the line, 0 when it names none, and whether it
// is such a destination at all.
func drawerLineTarget(href string) (path string, line int, ok bool) {
	if !strings.HasPrefix(href, drawerLineScheme) {
		return "", 0, false
	}
	u, err := url.Parse(strings.TrimPrefix(href, drawerLineScheme))
	if err != nil {
		return "", 0, false
	}
	if u.Fragment != "" {
		line, _ = strconv.Atoi(u.Fragment)
	}
	return u.Path, line, true
}

// drawerTagHref is where a link to a tag points, for the drawer: empty
// for all of them again.
func drawerTagHref(name string) string {
	return drawerTagScheme + (&url.URL{Path: name}).String()
}

// drawerTagTarget is what such a destination names, and whether it is
// one at all: empty for all the tags of the vault.
func drawerTagTarget(href string) (name string, ok bool) {
	if !strings.HasPrefix(href, drawerTagScheme) {
		return "", false
	}
	u, err := url.Parse(strings.TrimPrefix(href, drawerTagScheme))
	if err != nil {
		return "", false
	}
	return u.Path, true
}
