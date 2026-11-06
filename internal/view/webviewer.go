package view

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/locale"
)

// A web tab has a fixed width, not a maximum: a page title can be
// arbitrarily long, and a tab that grows with its text pushes its close
// button and the tabs after it out of place. The title is also shortened
// before it reaches the layout, so even a title the theme cannot ellipsize
// cannot widen the tab.
const (
	webTabWidth      float32 = 150
	webTabTitleRunes         = 18
)

// shortWebTitle bounds a page title by runes, not bytes, so a Chinese
// title is cut at a character rather than in the middle of one.
func shortWebTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return locale.WebUntitled()
	}
	runes := []rune(title)
	if len(runes) <= webTabTitleRunes {
		return title
	}
	return string(runes[:webTabTitleRunes-1]) + "…"
}

// WebTabInfo is one page the Web viewer opened in the preview pane. The
// app owns the native page; the view only draws its tab and places it.
type WebTabInfo struct {
	ID      string
	Title   string
	Address string
	Page    ui.NativeWebView
	Reader  bool
	Active  bool
}

// webTabStrip is the tab bar at the top of the preview pane. The note's
// own preview is a fixed first tab; each web page follows it, with a
// button to close it and one to open another.
func (w *Workspace) webTabStrip(c *ui.Context) {
	if len(w.WebTabs) == 0 {
		return
	}
	th := c.Theme()
	ui.Row(c).FillWidth().Height(headHeight).Gap(2).
		Padding(0, cornerGap).AlignItems(ui.Center).Children(func() {
		preview := ui.ButtonBase(c.Key("web-tab-preview")).
			Label(locale.WebTabPreview()).Tooltip(locale.WebTabPreview()).
			Height(tabHeight).Radius(th.Radius).Padding(0, tabPad).
			AlignItems(ui.Center)
		preview.Draw(func(p *ui.Painter, box ui.Rect) {
			switch {
			case w.WebTabActive == "":
				p.Fill(box, th.Surface, th.Radius)
			case preview.Hovered():
				p.Fill(box, th.SurfaceHover, th.Radius)
			}
		})
		preview.Children(func() {
			ui.Text(c, locale.WebTabPreview())
		})
		if preview.Clicked() && w.OnWebShowPreview != nil {
			w.OnWebShowPreview()
		}
		ui.ScrollHorizontal(c).Grow(1).Height(headHeight).
			Children(func() {
				ui.Row(c).Gap(2).AlignItems(ui.Center).Children(func() {
					for _, one := range w.WebTabs {
						w.webTab(c, one)
					}
				})
			})
		newTab := ui.ButtonBase(c.Key("web-tab-new")).
			Label("+").Tooltip(locale.WebNewTab()).
			Size(tabHeight, tabHeight).Radius(th.Radius).
			Justify(ui.Center).AlignItems(ui.Center)
		newTab.Children(func() { ui.Text(c, "+") })
		if newTab.Clicked() && w.OnWebNewTab != nil {
			w.OnWebNewTab()
		}
	})
	ui.Box(c).FillWidth().Height(1).Background(th.Border)
}

// webTab is one web page's tab, drawn like a note's tab: lit while it is
// on show, dimmed while it is not, with its own close button.
func (w *Workspace) webTab(c *ui.Context, one WebTabInfo) {
	th := c.Theme()
	short := shortWebTitle(one.Title)
	tip := one.Title
	if tip == "" {
		tip = one.Address
	}
	tab := ui.ButtonBase(c.Key("web-tab-"+one.ID)).
		Label(short).Tooltip(tip).
		Width(webTabWidth).Height(tabHeight).Radius(th.Radius).
		Padding(0, tabPad).Gap(tabGap).AlignItems(ui.Center).
		ClipX()
	tab.Draw(func(p *ui.Painter, box ui.Rect) {
		switch {
		case tab.Pressed():
			p.Fill(box, th.SurfacePressed, th.Radius)
		case one.Active:
			p.Fill(box, th.Surface, th.Radius)
		case tab.Hovered():
			p.Fill(box, th.SurfaceHover, th.Radius)
		}
	})
	tab.Children(func() {
		ui.Row(c).FillWidth().Gap(tabGap).AlignItems(ui.Center).
			Children(func() {
				title := ui.Text(c, short).Grow(1).
					NoWrap().Ellipsis("…")
				if !one.Active {
					title.TextColor(th.TextMuted)
				}
				close := ui.ButtonBase(c.Key("web-tab-close-"+one.ID)).
					Label(locale.WebCloseTab()).
					Tooltip(locale.WebCloseTab()).
					Size(tabClose, tabClose).Radius(tabClose / 2).
					Justify(ui.Center).AlignItems(ui.Center)
				close.Children(func() {
					ui.Icon(c, iconClose).Size(tabGlyph, tabGlyph)
				})
				if close.Clicked() && w.OnWebCloseTab != nil {
					w.OnWebCloseTab(one.ID)
				}
			})
	})
	if tab.Clicked() && w.OnWebSelectTab != nil {
		w.OnWebSelectTab(one.ID)
	}
}

// webPane is the address bar and the native page of the web tab on show.
// It takes the preview pane's content area; the note's Markdown preview
// is not drawn while a web tab is active.
func (w *Workspace) webPane(c *ui.Context) {
	th := c.Theme()
	w.webToolbar(c)
	var page ui.NativeWebView
	for _, one := range w.WebTabs {
		if one.Active {
			page = one.Page
			break
		}
	}
	if page == nil {
		ui.Column(c).Fill().Center().Children(func() {
			ui.Text(c, w.WebStatus).TextColor(th.TextMuted)
		})
		return
	}
	ui.Box(c).Fill().Clip().Children(func() {
		ui.WebView(c, page).Fill()
	})
	if w.WebStatus != "" {
		ui.Text(c, w.WebStatus).FontSize(th.FontSize - 1).
			TextColor(th.TextMuted)
	}
}

// webToolbar is the address bar and the buttons that act on a page.
func (w *Workspace) webToolbar(c *ui.Context) {
	th := c.Theme()
	ui.Row(c).FillWidth().Gap(th.Space(1)).
		Padding(th.Space(1), cornerGap).AlignItems(ui.Center).
		Children(func() {
			toolButton(c, "web-back", locale.WebBack(), w.WebCanBack,
				w.OnWebBack)
			toolButton(c, "web-forward", locale.WebForward(),
				w.WebCanForward, w.OnWebForward)
			toolButton(c, "web-reload", locale.WebReload(), true,
				w.OnWebReload)
			toolButton(c, "web-home", locale.WebHome(), true, w.OnWebHome)
			field := ui.TextInput(c.Key("web-address"), &w.WebAddress).
				Placeholder(locale.WebAddressPlaceholder()).Grow(1)
			if field.Submitted() && w.OnWebNavigate != nil {
				w.OnWebNavigate(w.WebAddress)
			}
			toolButton(c, "web-reader", locale.WebReader(), true,
				w.OnWebReader)
			toolButton(c, "web-save", locale.WebSave(), true, w.OnWebSave)
			toolButton(c, "web-external", locale.WebExternal(), true,
				w.OnWebOpenExternal)
		})
	ui.Box(c).FillWidth().Height(1).Background(th.Border)
}

func toolButton(c *ui.Context, key, label string, enabled bool,
	on func()) {
	button := ui.Button(c.Key(key), label).Tooltip(label).
		Disabled(!enabled)
	if button.Clicked() && on != nil {
		on()
	}
}
