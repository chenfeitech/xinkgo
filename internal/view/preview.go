// This file is the preview: the pane a note is read in, what is read in it,
// the buttons in its head, the way out of it as a page or a PDF, and where
// a link in it goes.

package view

import (
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/datafile"
	"github.com/helightxu/xinkgo/internal/locale"
	xmd "github.com/helightxu/xinkgo/internal/markdown"
	"github.com/helightxu/xinkgo/internal/media"
	"github.com/helightxu/xinkgo/internal/view/geom"
)

// previewPanel is what the note looks like, on the right.
func (w *Workspace) previewPanel(c *ui.Context, width float32) {
	// As the tree's panel: a note is read from its first line, and not
	// from the middle of the window. A width of none is the whole
	// window, which is what the pane is given when it is maximized.
	pane := ui.Column(c).FillHeight().Padding(0, geom.RailStrip, 0, 0)
	if width > 0 {
		pane.Width(width)
	} else {
		pane.Grow(1)
	}
	pane.Children(func() {
		// A preview given the whole window keeps the bar that is its
		// way back. One in a pane of its own — spread over the note,
		// or not — keeps the button of its corner instead.
		if w.mode.Zoom == geom.ZoomPreview {
			w.previewBar(c)
		} else {
			w.previewHead(c)
		}
		// The Web viewer's tabs stand in the preview pane. A web tab on
		// show takes the pane's content instead of the note's Markdown
		// preview, until the Preview tab is chosen again.
		w.webTabStrip(c)
		if w.WebTabActive != "" {
			w.webPane(c)
			return
		}
		n := w.Tabs.ActiveNote()
		if n == nil {
			return
		}
		// A style chosen, the preview shows the note as an article for
		// WeChat: what is seen is what the copy of it pastes, and it is
		// seen as on the screen of a phone.
		if w.wechatShown() {
			room, _ := c.Size()
			w.wechatArticle(c, room, width)
			return
		}
		opts := w.markdownOptions(c, n.Path)
		ui.Scroll(c).TrackScroll(&w.scroll.Preview).Fill().Children(func() {
			room, _ := c.Size()
			// Given the whole window, what is read is given the width
			// it is allowed and stands in the middle of what is left:
			// a column to read, and not a line against the window's
			// edge. In its own pane it is that pane's width already.
			if width <= 0 {
				ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
					ui.Box(c).FillWidth().MaxWidth(w.previewWidth(room)).
						Padding(w.geometry().PreviewPadY(), w.geometry().PreviewPadX()).
						Children(func() {
							markdown.View(c, w.previewText(n), opts)
						})
				})
				return
			}
			ui.Box(c).MaxWidth(w.previewWidth(room)).
				Padding(w.geometry().PreviewPadY(), w.geometry().PreviewPadX()).
				Children(func() {
					markdown.View(c, w.previewText(n), opts)
				})
		})
	})
}

// The kinds of file the note that is open may be written out as, as the
// preview asks the app for them.
const (
	// ExportPage is a page of HTML that stands on its own.
	ExportPage = "html"
	// ExportPDF is that page printed onto paper by the machine's own
	// webview: what to write on a sheet of paper, and where to break it,
	// is a browser's business and not ours.
	ExportPDF = "pdf"
)

// exportButton is the way out of the preview: the note on show, written out
// as a page of HTML or as a PDF. It stands in the head of the preview beside
// the buttons of WeChat, so the ways out of what is read are found together.
// A file the app only shows, a picture or a PDF, has nothing in it to write
// out, and gets no button.
func (w *Workspace) exportButton(c *ui.Context) {
	n := w.Tabs.ActiveNote()
	if n == nil || n.Preview {
		return
	}
	ui.MenuButton(c, locale.PreviewExport(), func(m *ui.Menu) {
		if m.Item(locale.MenuExportPage()).Chosen() {
			if w.OnExport != nil {
				w.OnExport(ExportPage)
			}
		}
		if m.Item(locale.MenuExportPdf()).Chosen() {
			if w.OnExport != nil {
				w.OnExport(ExportPDF)
			}
		}
	}).Padding(2, 4).Tooltip(locale.PreviewExportLabel())
}

// previewActions are the ways out of what the preview reads: set out for
// WeChat, or written to a file. They stand together at the end of the head,
// and a narrow pane gives what does not fit to the toolbar's overflow menu
// rather than drawing the buttons over one another.
func (w *Workspace) previewActions(c *ui.Context) {
	ui.Toolbar(c, func() {
		w.wechatButtons(c)
		w.exportButton(c)
	}).Gap(2).Padding(0, 0).Label(locale.CorePagePreviewName())
}

// ExportText is what an export is given: the very text the preview reads
// of the note that is open, and where the note stands, which is where the
// pictures it names are. A file the app only shows has nothing in it to
// write out.
func (w *Workspace) ExportText() (path, text string, ok bool) {
	n := w.Tabs.ActiveNote()
	if n == nil || n.Preview {
		return "", "", false
	}
	return n.Path, w.previewText(n), true
}

// previewHead is the row at the top of the preview's own pane: the button
// that spreads the preview over the note at one end, and the buttons that
// write what is read out and set it out for WeChat at the other. The spread
// button stands in the corner the note's toolbar is not in. The row takes
// the pane's own top, so what is read begins under it and never under the
// buttons themselves, and is as tall as the tab bar's, which stands level
// with it across the window: the corner buttons of the two panes are then
// drawn on one line, each keeping cornerGap from the line that parts them.
// What the window shows and does not read — a picture, a PDF — is spread
// as much as a note is: it is the same pane, read wider.
func (w *Workspace) previewHead(c *ui.Context) {
	if w.Tabs.ActiveNote() == nil {
		return
	}
	ui.Row(c).FillWidth().Height(headHeight).AlignItems(ui.Center).
		Gap(4).Padding(0, cornerGap).Children(func() {
		w.spreadButton(c)
		ui.Box(c).Grow(1)
		w.previewActions(c)
	})
}

// spreadButton is the button in the corner of the preview's pane, which is
// the pane's own top: pressed, the preview is spread over the note, out as
// far as the tree, and the tree keeps the room it was given. Pressed again
// the note comes back as wide as it was. It is the other half of the one
// at the end of the tab bar, which stretches the note over the preview.
func (w *Workspace) spreadButton(c *ui.Context) {
	spread := w.mode.Zoom == geom.ZoomPreviewSpread
	// A preview standing over the note is one this button takes it from,
	// and so it says the way back: the arrow turned round, and the press
	// named for what it does.
	icon, label, tip := iconSpreadPreview, locale.SpreadPreview(),
		locale.SpreadPreviewTooltip()
	if spread {
		icon, label, tip = iconSpreadPreviewBack, locale.SpreadPreviewBack(),
			locale.SpreadPreviewBackTooltip()
	}
	// A button of no look, as the one that widens the note is: the picture
	// alone, standing in the pane's head as what is read does, and not in
	// a box of its own. Nothing is filled in behind it — not when the
	// pointer rests on it, and not while the preview stands over the note,
	// for the arrow turned round is what says so. Only the picture changes:
	// it is quiet until the pointer comes to it, or until the preview has
	// the room. What it does, and whether it is pressed, are still said to
	// the keys and to what reads the window aloud, which is what the box
	// was for.
	button := ui.ButtonBase(c.Key("preview-spread")).
		Label(label).Tooltip(tip).
		Role(ui.RoleToggleButton).Checked(spread).
		Padding(4, 5)
	button.Children(func() {
		glyph := ui.Icon(c, icon).Size(iconSize, iconSize)
		if !spread && !button.Hovered() {
			glyph.TextColor(c.Theme().TextMuted)
		}
	})
	if button.Clicked() {
		w.setZoom(geom.ZoomPreviewSpread)
	}
}

// previewBar is the row at the top of the preview when the preview has
// the whole window: the buttons that write the note out and set it out for
// WeChat, and the button that takes it away again, which is the way
// out of a pane the note's toolbar is not drawn in. It stands at the
// pane's own end, clear of the rail, whose buttons hang over it. A
// preview in a pane of its own has the button of its corner there
// instead, which is smaller and stands at the other end.
func (w *Workspace) previewBar(c *ui.Context) {
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(4).
		Padding(4, 6).PaddingEnd(w.geometry().RailBand() - geom.RailStrip + 6).
		Children(func() {
			ui.Box(c).Grow(1)
			w.previewActions(c)
			w.zoomButton(c, "zoom-back", geom.ZoomPreview, iconZoomPreview,
				locale.ZoomBack(), locale.ZoomBackTooltip())
		})
}

// previewTextOfFile is what the preview is given of a file the app only
// shows: the picture itself, whose name the preview reads as one of the
// note's own, or a line saying what the file is.
func (w *Workspace) previewTextOfFile(n *OpenNote) string {
	switch {
	case media.IsImage(n.Path):
		return "![<" + PathOf(n.Path) + ">](<" + PathOf(n.Path) + ">)"
	default:
		return "_" + w.sayOfFile(n) + "_"
	}
}

// markdownOptions is how the preview reads the note: its pictures come
// from the note's own folder, and a link to another note opens it here.
func (w *Workspace) markdownOptions(
	c *ui.Context, path string,
) *markdown.Options {
	opts := &markdown.Options{
		Theme:  w.previewTheme(c),
		OnLink: func(url string) { w.openLink(url, path) },
	}
	// Pictures come from the note's own folder, unless the user has
	// asked for none of them.
	if w.Config.Preview.ImageDisplay != config.ImageHidden {
		opts.LoadImage = markdown.FileImages(filepath.Dir(path))
	}
	return opts
}

// previewText is what the preview is given: a note as it is, its links
// in double brackets made into ones that can be clicked, and a file of
// data as a block of code, written out evenly when it can be.
func (w *Workspace) previewText(n *OpenNote) string {
	// A file the app only shows is not read as Markdown: a picture is
	// shown as the picture it is, and the rest are said in a line.
	if n.Preview {
		return w.previewTextOfFile(n)
	}
	kind := datafile.KindOf(n.Path)
	if kind == datafile.None {
		return xmd.Links(n.Note.Text, w.resolveLink)
	}
	text := n.Note.Text
	// A file of megabytes is shown as it stands.
	if !datafile.TooLarge(uint64(len(text)),
		w.Config.Editor.DataFile.MaxFormatBytes) {
		if even, ok := datafile.Format(kind, text,
			int(w.Config.Editor.DataFile.Indent)); ok {
			text = even
		}
	}
	return "```" + strings.ToLower(kind.Name()) + "\n" + text + "\n```"
}

// openLink is what a click in the preview does: a link in double brackets
// opens the note it names, a page on the web goes where the user said
// links should go, and anything else is a file beside the note.
func (w *Workspace) openLink(url, from string) {
	if target, ok := xmd.WikiTarget(url); ok {
		w.openWikiLink(target)
		return
	}
	// A page on the web is not opened by the view when the window says
	// what a click does: it is for the app to open, or for nobody.
	if outsideLink(url) {
		if w.OnOpenOutside != nil {
			w.OnOpenOutside(url)
		}
		return
	}
	if strings.HasPrefix(url, "#") {
		return
	}
	target := url
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(from), target)
	}
	if _, err := w.Tabs.Open(target); err != nil {
		w.status = locale.StatusNoNoteAt(PathOf(target))
	}
}

// openWikiLink is a click on a link in double brackets: the note it names
// is opened. It is looked up again now, and not when the note was
// shown: the vault may have gained the note since.
func (w *Workspace) openWikiLink(target string) {
	path, ok := w.resolveLink(target)
	if !ok {
		w.status = locale.StatusNoNoteCalled(target)
		return
	}
	if _, err := w.Tabs.Open(path); err != nil {
		w.status = locale.StatusCouldNotOpen(PathOf(path))
	}
}

// outsideLink is whether a link is a page on the web or a letter, and so
// not a file of the vault.
func outsideLink(url string) bool {
	return strings.HasPrefix(url, "http://") ||
		strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "mailto:")
}

// resolveLink is what a link in double brackets names: the note of the
// vault, when there is one and only one.
func (w *Workspace) resolveLink(target string) (string, bool) {
	if w.Index == nil {
		return "", false
	}
	return w.Index.ResolveOnly(target)
}
