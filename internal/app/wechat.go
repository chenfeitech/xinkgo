package app

import (
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/wechat"
)

// The folder of the app's own the styles of the user's making stand in:
// one file of JSON to a style, which is how a style is added without
// touching the app.
const wechatFolder = "wechat"

// errNoWindow is the answer to a page asked for before there is a window
// to make it of, as a test, or anything else that asks before the window
// is open, does.
var errNoWindow = errors.New("xinkgo: no window to make a page of")

// wechatDir is the folder the styles are read from: the one the user
// named in the settings, or, when they named none, the folder of the
// app's own.
func (s *appState) wechatDir() string {
	if dir := strings.TrimSpace(s.config.Preview.WechatThemeDir); dir != "" {
		return dir
	}
	return filepath.Join(s.dir, wechatFolder)
}

// wechatThemes is every style a note may be set out in: the ones the app
// was made with, and then the ones the user has put in the folder they
// named. A file that is no style is said in the log, and the rest are
// none the worse.
func (s *appState) wechatThemes() []wechat.Theme {
	mine, bad := wechat.OfDir(s.wechatDir())
	if len(bad) > 0 {
		slog.Warn("styles of WeChat that are no styles", "files", bad)
	}
	return append(wechat.Builtin(), mine...)
}

// loadWechatThemes is the styles read again, as after the folder they are
// read from has been changed: the window is told of them, the page filled
// again, and a style that is no longer one of them is no longer the one
// chosen — a style taken away is the note as Markdown reads it.
func (s *appState) loadWechatThemes() {
	if s.ws == nil {
		return
	}
	s.ws.WechatThemes = s.wechatThemes()
	id := wechatStyleOf(s.ws.WechatThemes, s.config.Preview.Wechat)
	if id == s.config.Preview.Wechat {
		s.refreshWechat()
		return
	}
	s.config.Preview.Wechat = id
	s.ws.Config = s.config
	if err := s.config.Save(s.dir); err != nil {
		slog.Warn("the choice could not be written", "err", err)
	}
	s.refreshWechat()
}

// setWechatStyle is the style the preview sets the note out in: "" for
// the note as Markdown reads it, and the id of one of the styles for an
// article for WeChat. The choice is kept, and the page filled again.
func (s *appState) setWechatStyle(id string) {
	// The styles are read again: one put in the folder since the app
	// started, or taken out of it, is then one the menu shows, or does
	// not. A name that is none of them — a style taken away while the
	// app was not running — is the note as Markdown reads it.
	themes := s.wechatThemes()
	s.ws.WechatThemes = themes
	id = wechatStyleOf(themes, id)
	s.config.Preview.Wechat = id
	s.ws.Config = s.config
	if err := s.config.Save(s.dir); err != nil {
		slog.Warn("the choice could not be written", "err", err)
	}
	s.showWechat()
}

// showWechat is the note set out as an article for WeChat, and the reason
// it cannot be when it cannot: a page is a webview's, and where the
// machine has none, what is read stays the note as Markdown reads it, to
// which the copy is none the worse.
func (s *appState) showWechat() {
	if s.config.Preview.Wechat != "" && !mygo.WebViewAvailable() {
		s.say(locale.SayWechatNoWebView())
		return
	}
	s.refreshWechat()
}

// wechatStyleOf is the id of one of the styles, or none at all: a name
// that is none of them is a style the user has taken away, and the note
// is read as Markdown reads it.
func wechatStyleOf(themes []wechat.Theme, id string) string {
	if !wechat.Has(themes, id) {
		return ""
	}
	return id
}

// refreshWechat fills the page the preview shows the note in, when a
// style is chosen: the style has been changed, or another note opened, or
// the one on show written to the disk. The note is laid out off the main
// thread, for it is the work of a moment and not of a frame.
func (s *appState) refreshWechat() {
	// Until the window is up and the notes of the last run are open
	// again, there is no note on show to set out: the opening of them
	// asks for one each, and the last of them is the one that is read.
	if s.ws == nil || !s.wechatUp || s.config.Preview.Wechat == "" {
		return
	}
	path, text, ok := s.ws.ExportText()
	if !ok {
		return
	}
	// A page is what shows an article, and a page is a webview's: where
	// the machine has none there is no page to fill, and what is read
	// stays the note as Markdown reads it.
	if !mygo.WebViewAvailable() {
		return
	}
	page, err := s.wechatPage()
	if err != nil {
		slog.Warn("the page of the article could not be made", "err", err)
		return
	}
	theme := wechat.Get(s.ws.WechatThemes, s.config.Preview.Wechat)
	if theme.ID == "" {
		return
	}
	o := s.wechatOptions(path)
	o.Theme = theme
	// Which filling of the page this is: another note opened, or the one
	// on show written again, or the style changed, asks for one too, and
	// the page is filled with the last of them alone.
	seq := atomic.AddUint64(&s.wechatSeq, 1)
	go func() {
		html := wechat.Page(text, o)
		if atomic.LoadUint64(&s.wechatSeq) != seq {
			return
		}
		mygo.RunOnMain(func() { page.Page().LoadHTML(html, "") })
	}()
}

// wechatPage is the page the preview shows the article in: one of them,
// made the first time a style is chosen and kept thereafter, as a page is
// made of the window, which is the main thread's work.
func (s *appState) wechatPage() (*mygo.WebView, error) {
	if s.wechatView != nil {
		return s.wechatView, nil
	}
	if s.win == nil {
		return nil, errNoWindow
	}
	var (
		page *mygo.WebView
		err  error
	)
	mygo.RunOnMain(func() {
		page, err = s.win.NewWebView(mygo.WebViewOptions{})
	})
	if err != nil {
		return nil, err
	}
	if page == nil {
		return nil, errNoWindow
	}
	s.wechatView = page
	// The window draws what the app fills: the page is given to it as
	// what it is, a web view of the window's own.
	s.ws.WechatPage = page
	return page, nil
}

// wechatOptions is how a note is set out: its pictures read from the
// folder it stands in, and left out when the preview is asked for none of
// them, which is one of the ways the preview is set.
func (s *appState) wechatOptions(path string) wechat.Options {
	o := wechat.Options{
		BaseDir: filepath.Dir(path),
		Images:  s.config.Preview.ImageDisplay != config.ImageHidden,
	}
	if s.config.Basic.Language == config.LanguageChinese {
		o.Lang = "zh-CN"
	}
	return o
}

// copyWechat puts the note on the clipboard as it is set out, which is
// what is pasted into the editor of WeChat: the article as HTML, for the
// editor keeps nothing of a style sheet, and the words of the note as
// they stand beside it, for what reads no HTML at all.
func (s *appState) copyWechat() {
	path, text, ok := s.ws.ExportText()
	if !ok {
		s.say(locale.SayWechatNoNote())
		return
	}
	theme := wechat.Get(s.ws.WechatThemes, s.config.Preview.Wechat)
	if theme.ID == "" {
		return
	}
	o := s.wechatOptions(path)
	o.Theme = theme
	go func() {
		article := wechat.Article(text, o)
		err := mygo.Clipboard.Write(transfer.New(transfer.NewItem(
			transfer.Bytes(transfer.HTML, []byte(article)),
			transfer.Bytes(transfer.Text, []byte(text)),
		)))
		if err != nil {
			slog.Warn("the note could not be copied", "err", err)
			s.say(locale.SayWechatFailed())
			return
		}
		s.say(locale.SayWechatCopied(theme.Name))
	}()
}

// chooseWechatDir is the folder the styles are read from, asked of the
// user with a dialog of the machine's own. What the folder holds is read
// again as soon as the settings change, which they do as soon as the
// folder is named.
func (s *appState) chooseWechatDir() {
	go func() {
		picked, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title:     locale.DialogWechatDir(),
			Directory: true,
		})
		if err != nil || len(picked) == 0 {
			return
		}
		mygo.RunOnMain(func() { s.draft.wechatDir = picked[0] })
	}()
}
