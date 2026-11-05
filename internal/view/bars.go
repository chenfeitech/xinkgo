// This file is the rows across the panes: the tab bar over the note, the
// toolbar under it, and the line the window says at its foot.

package view

import (
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/datafile"
	"github.com/helightxu/xinkgo/internal/locale"
	xmd "github.com/helightxu/xinkgo/internal/markdown"
	"github.com/helightxu/xinkgo/internal/view/geom"
)

// iconClose is the × a tab is closed by: two strokes of a 24 by 24 box,
// drawn in the color of the text beside it.
var iconClose = svgIcon(`<path d="M18 6 6 18"/><path d="m6 6 12 12"/>`)

// How a tab is put together: its height, the room its name and its close
// button keep inside it, and how wide or narrow it may be.
const (
	tabHeight   float32 = 26
	tabPad      float32 = 8
	tabGap      float32 = 6
	tabGlyph    float32 = 12
	tabClose    float32 = 16
	tabMaxWidth float32 = 180
	// tabNameMax is how wide a tab's name may be: what is left of a tab
	// of tabMaxWidth once its padding, its close button and the room
	// between the two are taken out.
	tabNameMax float32 = 142
	// cornerGap is the room a corner button keeps from the edge of the
	// pane it stands in: the button at the end of the tab bar and the
	// one in the preview's head stand as far from the line that parts
	// the panes — which is drawn down the middle of its grip — as one
	// another, so that the two are read as one row across it.
	cornerGap float32 = 6
	// headHeight is how tall the row at the top of a pane stands: the
	// tab bar's, which is the note's own top, and the preview's head.
	// Each of the two carries a button in its corner, and the two are
	// read as one row across the window, so the row is of one height in
	// either pane and the buttons stand level with each other.
	headHeight = tabHeight + 8
)

// markdownWidth is how wide the row of the Markdown buttons may be: what
// the panels leave, less the room the way to the model takes beside it, so
// that it stands beside the row and not under the rail.
func (w *Workspace) markdownWidth(c *ui.Context) float32 {
	win, _ := c.Size()
	room := w.geometry().EditorWidth(win) - aiBand
	// Narrower than a few buttons there is nothing to show but the menu
	// at the end, which is still worth its place.
	if room < iconSize*3 {
		return iconSize * 3
	}
	return room
}

// tabBar is one tab per open note, and the way to make a new one: the
// tabs are read from left to right, and the strip is scrolled sideways
// when there are more of them than there is room for, as the tabs of the
// editors the reader knows are. The button that makes a note stands
// outside what is scrolled, so that it is always there.
func (w *Workspace) tabBar(c *ui.Context) {
	paths := w.Tabs.Paths()
	ui.Row(c).AlignItems(ui.Center).
		PaddingEnd(w.geometry().RailClear() + cornerGap).Children(func() {
		ui.ScrollHorizontal(c).Grow(1).Height(headHeight).
			Children(func() {
				ui.Row(c).Gap(2).Padding(4).AlignItems(ui.Center).
					Children(func() {
						for _, path := range paths {
							path := path
							w.tab(c, path)
						}
					})
			})
		w.widenButton(c)
	})
}

// tab is one open note: its name, marked when it has changes the disk
// does not, and the button that closes it, both inside the one tab, as
// the tabs of the editors the reader knows. The note on show is lit, and
// so is the one under the pointer.
func (w *Workspace) tab(c *ui.Context, path string) {
	th := c.Theme()
	on := w.Tabs.IsActive(path)
	name := PathOf(path)
	tab := ui.ButtonBase(c.Key("tab-"+path)).Label(name).Tooltip(path).
		Height(tabHeight).Radius(th.Radius).Padding(0, tabPad).
		Gap(tabGap).AlignItems(ui.Center).MaxWidth(tabMaxWidth)
	tab.Draw(func(p *ui.Painter, box ui.Rect) {
		switch {
		case tab.Pressed():
			p.Fill(box, th.SurfacePressed, th.Radius)
		case on:
			p.Fill(box, th.Surface, th.Radius)
		case tab.Hovered():
			p.Fill(box, th.SurfaceHover, th.Radius)
		}
	})
	tab.Children(func() {
		ui.Row(c).FillWidth().Gap(tabGap).AlignItems(ui.Center).
			Children(func() {
				title := ui.Text(c, Title(path, w.Tabs.Dirty(path))).
					Grow(1).MaxWidth(tabNameMax).Ellipsis("…").NoWrap()
				if !on {
					title.TextColor(th.TextMuted)
				}
				w.tabClose(c, path, on)
			})
	})
	if tab.Clicked() {
		w.Tabs.Activate(path)
	}
}

// tabClose is the × of a tab: it stands within the tab, and closes the
// note without showing it first.
func (w *Workspace) tabClose(c *ui.Context, path string, on bool) {
	th := c.Theme()
	name := PathOf(path)
	b := ui.ButtonBase(c.Key(path+"#close")).Label(locale.TabCloseLabel(name)).
		Tooltip(locale.TabCloseTooltip(name)).
		Size(tabClose, tabClose).
		Radius(tabClose / 2).Justify(ui.Center).AlignItems(ui.Center)
	b.Draw(func(p *ui.Painter, box ui.Rect) {
		switch {
		case b.Pressed():
			p.Fill(box, th.SurfacePressed, tabClose/2)
		case b.Hovered():
			p.Fill(box, th.SurfaceHover, tabClose/2)
		}
	})
	b.Children(func() {
		glyph := ui.Icon(c, iconClose).Size(tabGlyph, tabGlyph)
		if !on && !b.Hovered() {
			glyph.TextColor(th.TextMuted)
		}
	})
	if b.Clicked() {
		w.Tabs.Close(path)
		w.status = ""
	}
}

// widenButton is the button at the end of the tab bar, which is the
// note's own top corner: pressed, the note is stretched over the preview,
// out to the window's right edge, and the tree keeps the room it was
// given. Pressed again the preview comes back as wide as it was. It is
// not drawn while the preview stands shut, for then there is no room of
// its for the note to stretch into.
func (w *Workspace) widenButton(c *ui.Context) {
	if !w.widenOffered() {
		return
	}
	// A note standing over the preview is one this button takes it from,
	// and so it says the way back: the arrow turned round, and the press
	// named for what it does.
	icon, label, tip := iconWiden, locale.WidenNote(), locale.WidenNoteTooltip()
	if w.mode.Stretched {
		icon, label, tip = iconWidenBack, locale.WidenBack(),
			locale.WidenBackTooltip()
	}
	// A button of no look: the picture alone, standing on the bar as the
	// words of the tabs do, and not in a box of its own. Nothing is
	// filled in behind it — not when the pointer rests on it, and not
	// while the note stands over the preview, for the arrow turned round
	// is what says so. Only the picture changes: it is quiet until the
	// pointer comes to it, or until the note has the room. What it does,
	// and whether it is pressed, are still said to the keys and to what
	// reads the window aloud, which is what the box was for.
	button := ui.ButtonBase(c.Key("widen")).
		Label(label).Tooltip(tip).
		Role(ui.RoleToggleButton).Checked(w.mode.Stretched).
		Padding(4, 5)
	button.Children(func() {
		glyph := ui.Icon(c, icon).Size(iconSize, iconSize)
		if !w.mode.Stretched && !button.Hovered() {
			glyph.TextColor(c.Theme().TextMuted)
		}
	})
	if button.Clicked() {
		w.mode.Stretched = !w.mode.Stretched
	}
}

// widenOffered is whether the button that widens the note is drawn: it
// takes the preview's room, so there is nothing for it to do where the
// preview stands shut — unless the note stands over that room now, and
// has to be given the way back.
func (w *Workspace) widenOffered() bool {
	return w.mode.Stretched || w.Layout.Right > 0
}

// toolbar is the row of buttons that write Markdown for the user: one
// button wraps what is selected, and stands for a pair of marks when
// nothing is. A button is a picture, and what it does is said in words
// only when the pointer rests on it, as in a text editor's toolbar; what
// there is no room for goes into a menu at its end, so that the row never
// runs wider than the note it writes in.
func (w *Workspace) toolbar(c *ui.Context) {
	n := w.Tabs.ActiveNote()
	if n == nil {
		return
	}
	// A file of data has no use of the buttons that write Markdown: it
	// is told which it is, and whether it reads. Nor has a file the app
	// only shows, which has nothing to write in at all.
	if kind := datafile.KindOf(n.Path); kind != datafile.None {
		w.dataBar(c, n, kind)
		return
	}
	if n.Preview {
		w.fileBar(c, n)
		return
	}
	// The buttons are as close together as an editor's are, so that
	// all of them are seen side by side rather than in the menu. What
	// there is no room for goes into the menu at the toolbar's end;
	// the way to the model is not one of them, so it stands outside it.
	ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
		ui.Toolbar(c, func() {
			for i, group := range xmd.Groups {
				if i > 0 {
					w.toolbarDivider(c)
				}
				for _, f := range group {
					f := f
					if ui.Button(c.Key("fmt-"+f.Label()), "").
						Label(f.Label()).Tooltip(f.Desc()).
						Padding(4, 5).
						Children(func() {
							ui.Icon(c, formatIcon(f)).
								Size(iconSize, iconSize)
						}).Clicked() {
						w.Format(f)
					}
				}
			}
		}).Gap(2).Padding(3, 4).MaxWidth(w.markdownWidth(c)).
			Label(locale.ToolbarMarkdown())
		w.toolbarDivider(c)
		w.aiButton(c)
	})
}

// zoomButton is one button that gives a pane the whole window: the note's
// at the end of the Markdown toolbar, the preview's at the top of its
// pane. Pressed again it gives the room back, and while the pane has the
// window the button stands lit, as the way to the model does.
func (w *Workspace) zoomButton(
	c *ui.Context, key string, pane geom.Zoom, icon *ui.SVG,
	label, tip string,
) {
	// A pane that has the window is one this button takes it from, and
	// so it says the way back: the picture of a pane giving up the room
	// it took, and what the press will do.
	if w.mode.Zoom == pane {
		icon = iconZoomBack
	}
	button := ui.Button(c.Key(key), "").
		Label(label).Tooltip(w.zoomTip(pane, tip)).
		Padding(4, 5).
		Children(func() {
			ui.Icon(c, icon).Size(iconSize, iconSize)
		})
	if w.mode.Zoom == pane {
		button.Selected(true)
	}
	if button.Clicked() {
		w.setZoom(pane)
	}
}

// zoomTip is what the button says when the pointer rests on it: what the
// press will do, which is the room given back when the pane has it.
func (w *Workspace) zoomTip(pane geom.Zoom, tip string) string {
	if w.mode.Zoom == pane {
		return locale.ZoomBackTooltip()
	}
	return tip
}

// aiButton is the way to the model: a button of the toolbar's own kind,
// a picture with a word said of it when the pointer rests on it, and the
// conversation under the note opened or hidden by a click.
func (w *Workspace) aiButton(c *ui.Context) {
	n := w.Tabs.ActiveNote()
	if n == nil {
		return
	}
	button := ui.Button(c.Key("ai-open"), "").
		Label(locale.AiAssist()).Tooltip(locale.AiAssistTooltip()).
		Padding(4, 5).
		Children(func() {
			ui.Icon(c, iconAi).Size(iconSize, iconSize)
		})
	if w.aiOpen(n.Path) {
		button.Selected(true)
	}
	if button.Clicked() {
		w.ToggleAI()
	}
}

// toolbarDivider is the line that sets two groups of the toolbar apart.
func (w *Workspace) toolbarDivider(c *ui.Context) {
	ui.Box(c).Width(1).Height(iconDivider).
		Background(c.Theme().Border)
}

// statusBar is what the window says about what happened. When the
// word-count core plugin stands it also says how much the note on show
// holds, after the message the window has.
func (w *Workspace) statusBar(c *ui.Context) {
	text := w.status
	if text == "" {
		n := w.Tabs.ActiveNote()
		if n != nil {
			text = languageName(n.Path)
			if n.Note.Dirty {
				text += " · not saved"
			}
		}
	}
	count := ""
	if w.coreOn(CoreWordCount) {
		if n := w.Tabs.ActiveNote(); n != nil && !n.Preview {
			count = locale.StatusWordCount(strconv.Itoa(wordsIn(n.Note.Text)))
		}
	}
	if text == "" && count == "" {
		return
	}
	ui.Row(c).Padding(8).Children(func() {
		if text != "" {
			ui.Text(c, text).FontSize(12).TextColor(c.Theme().TextMuted)
		}
		if count != "" {
			ui.Box(c).Grow(1)
			ui.Text(c, count).FontSize(12).TextColor(c.Theme().TextMuted)
		}
	})
}

// wordsIn counts the words of a note: runs of whitespace between them in
// Latin text, and each CJK character as one, as a reader of either counts.
func wordsIn(text string) int {
	words := 0
	inWord := false
	for _, r := range text {
		switch {
		case r == '\n':
			inWord = false
		case isSpace(r):
			inWord = false
		case isCJK(r):
			words++
			inWord = false
		default:
			if !inWord {
				words++
				inWord = true
			}
		}
	}
	return words
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\r'
}

func isCJK(r rune) bool {
	return r >= 0x3400 && r <= 0x9fff || r >= 0xf900 && r <= 0xfaff
}
