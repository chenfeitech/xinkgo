package view

import (
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/locale"
)

// wechatOn is whether a style has been chosen for the note: the preview
// shows the note set out as an article for WeChat, and not as Markdown
// reads it.
func (w *Workspace) wechatOn() bool { return w.Config.Preview.Wechat != "" }

// wechatShown is whether that article can be shown: a style is chosen,
// and the app has made the page it stands in. The app makes it, of the
// window and on the main thread, and there is none where the machine has
// no webview — in which case what is read is still the note as Markdown
// reads it, and the copy of it is none the worse.
func (w *Workspace) wechatShown() bool {
	return w.wechatOn() && w.WechatPage != nil
}

// wechatButtons are the two ways of a note set out as an article for
// WeChat, in the corner of the preview: the style it is set out in,
// chosen in the menu of the one, and the copy of it that is pasted into
// the editor of WeChat, taken by the other. The copy is there while a
// style is: with no style chosen, what would be copied is the note as it
// stands, which the note's own pane gives.
func (w *Workspace) wechatButtons(c *ui.Context) {
	if w.Tabs.ActiveNote() == nil {
		return
	}
	id := w.Config.Preview.Wechat
	ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
		if id != "" {
			button := ui.ButtonBase(c.Key("wechat-copy")).
				Label(locale.WechatCopy()).
				Tooltip(locale.WechatCopyLabel()).Padding(4, 5)
			button.Children(func() {
				ui.Icon(c, iconCopy).Size(iconSize, iconSize)
			})
			if button.Clicked() && w.OnWechatCopy != nil {
				w.OnWechatCopy()
			}
		}
		ui.MenuButton(c, locale.WechatStyle(), func(m *ui.Menu) {
			// The note as Markdown reads it is the way a first start
			// has it, and the way back from a style.
			if m.Item(locale.WechatStyleMarkdown()).Radio(id == "").
				Chosen() {
				if w.OnWechatStyle != nil {
					w.OnWechatStyle("")
				}
			}
			if len(w.WechatThemes) > 0 {
				m.Separator()
			}
			for _, th := range w.WechatThemes {
				if m.Item(th.Name).Radio(th.ID == id).Chosen() {
					if w.OnWechatStyle != nil {
						w.OnWechatStyle(th.ID)
					}
				}
			}
		}).Padding(2, 4).Tooltip(locale.WechatStyleLabel())
	})
}

// The screen an article of WeChat is read on, as the preview draws it:
// the width of a phone's, how round its corners are, and the room it
// keeps from the edges of the pane it stands in. The width is that of a
// large phone, whose screen is wider than the article on it: the page
// keeps the article off the edges of the screen, as WeChat does, and
// what is left is the room an article has on a phone.
const (
	phoneWidth  float32 = 414
	phoneRadius float32 = 8
	phonePad    float32 = 12
)

// wechatScreen is how wide that screen stands, and how much room it keeps
// from the edges of the pane: the width of a phone's, with the room
// beside it, however wide the pane is — for an article of WeChat is read
// on a phone, and the preview is to say how it reads there. A pane with
// less room than the screen and its room beside it keeps none of that
// room, and a pane with less than the screen gives what it has: an
// article cut off at the side is no article at all.
func wechatScreen(room float32) (screen, pad float32) {
	if room <= 0 {
		return 0, 0
	}
	if room-2*phonePad >= phoneWidth {
		return phoneWidth, phonePad
	}
	return min(phoneWidth, room), 0
}

// wechatArticle is the note set out as an article for WeChat, shown in
// the page the app filled: the same words and the same looks as what the
// copy of it puts on the clipboard, so that what is seen is what is
// pasted.
//
// It stands in the middle of the pane as the screen of a phone, and not
// as wide as the pane gives: what is read is read on a phone, and the
// look of an article — where its lines break, how wide a picture is
// drawn — is the look it has at that width and no other. The note as
// Markdown reads it is the other way, and takes the room it is given.
func (w *Workspace) wechatArticle(c *ui.Context, room, width float32) {
	// The room the screen has is the width of the pane it stands in,
	// which is the width the pane was given; the preview that has the
	// whole window has the room of the window instead.
	if width > 0 {
		room = width
	}
	t := c.Theme()
	screen, pad := wechatScreen(room)
	ui.Row(c).Fill().Justify(ui.Center).
		Padding(pad, pad).Children(func() {
		ui.Box(c).FillHeight().Width(screen).
			Radius(phoneRadius).Border(1, t.Border).Clip().
			Children(func() {
				ui.WebView(c, w.WechatPage).Fill()
			})
	})
}
