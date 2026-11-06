package app

import (
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/fs"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/templating"
	"github.com/helightxu/xinkgo/internal/view"
	"github.com/helightxu/xinkgo/internal/webviewer"
)

// webViewerAssets are compiled into the binary. A script the app runs
// itself and a rule list it compiles are not served to a page, so
// embedding them removes the risk of a packaged build missing a resource.
//
//go:embed webviewer_assets/reader.js webviewer_assets/content-rules.json
var webViewerAssets embed.FS

const (
	webReaderAsset = "webviewer_assets/reader.js"
	webRulesAsset  = "webviewer_assets/content-rules.json"
)

// webMaxTabs bounds native WebViews. MyGo keeps a page alive while it is
// not shown, so each tab costs a real web view until it is closed.
const webMaxTabs = 8

// webTab is one page of the Web viewer. reader is made only when the
// reader view is first used, and both pages belong to the same window.
type webTab struct {
	id         string
	url        string
	title      string
	page       *mygo.WebView
	reader     *mygo.WebView
	readerOn   bool
	canBack    bool
	canForward bool
}

// webViewerState is everything the plugin owns. It is empty until the
// user first opens a page: no native WebView is made at startup. Pages
// belong to the main window and are shown in its preview pane.
type webViewerState struct {
	tabs         []*webTab
	active       string
	session      *mygo.Session
	rules        *mygo.ContentRules
	rulesLoading bool
	readerJS     string
	seq          uint64
	status       string
}

func (s *appState) webActiveTab() *webTab {
	for _, one := range s.web.tabs {
		if one.id == s.web.active {
			return one
		}
	}
	return nil
}

func webRules(r *mygo.ContentRules) []*mygo.ContentRules {
	if r == nil {
		return nil
	}
	return []*mygo.ContentRules{r}
}

// webEnsureHost reports whether the main window can host a web page.
// ui.WebView only accepts pages made by the window that draws it, so
// every page belongs to the main window and lives in its preview pane.
func (s *appState) webEnsureHost() bool {
	if !mygo.WebViewAvailable() || !mygo.App.IsReady() {
		return false
	}
	if s.win == nil || s.win.IsDestroyed() {
		return false
	}
	s.webEnsureSession()
	return true
}

// webOpen shows the preview pane with a web tab, making a first tab if
// there is none. It no longer opens a window of its own.
func (s *appState) webOpen() {
	if !s.coreEnabled("web-viewer") {
		return
	}
	if !s.webEnsureHost() {
		s.sayNow(locale.SayWebNoWebView())
		return
	}
	if len(s.web.tabs) == 0 {
		o := s.coreOptions().WebViewer
		address := "about:blank"
		if home := strings.TrimSpace(o.Homepage); home != "" {
			if norm, err := webviewer.Normalize(home, o.SearchEngine,
				o.AllowLocal); err == nil {
				address = norm
			}
		}
		s.webNewTab(address)
	} else if s.web.active == "" {
		s.web.active = s.web.tabs[0].id
	}
	s.showWebPane()
	s.webPublish()
}

// showWebPane makes sure the preview pane is on show: a web tab drawn in
// a hidden pane is a page the user cannot see.
func (s *appState) showWebPane() {
	if s.ws != nil && !s.ws.PreviewShown() {
		s.ws.SetPreviewShown(true)
	}
}

// webOpenURL is the path an external link takes when links open in the
// app. It returns false when the address is refused, so the caller can
// fall back to the user's browser.
func (s *appState) webOpenURL(address string) bool {
	if !s.coreEnabled("web-viewer") {
		return false
	}
	o := s.coreOptions().WebViewer
	norm, err := webviewer.Normalize(address, o.SearchEngine, o.AllowLocal)
	if err != nil {
		return false
	}
	if !s.webEnsureHost() {
		return false
	}
	s.webNewTab(norm)
	s.showWebPane()
	return true
}

// webNewTab makes one native page and installs the remote-page boundary.
func (s *appState) webNewTab(address string) {
	if !s.coreEnabled("web-viewer") || !s.webEnsureHost() {
		s.sayNow(locale.SayWebNoWebView())
		return
	}
	if len(s.web.tabs) >= webMaxTabs {
		s.web.status = locale.WebLoadFailed()
		s.webPublish()
		return
	}
	s.webEnsureSession()
	s.webEnsureRules()
	s.web.seq++
	id := fmt.Sprintf("web-%d", s.web.seq)
	page, err := s.win.NewWebView(mygo.WebViewOptions{
		URL: address,
		Page: mygo.PageOptions{
			Untrusted:    true,
			Session:      s.web.session,
			ContentRules: webRules(s.web.rules),
		},
	})
	if err != nil {
		slog.Warn("web viewer page could not be made", "err", err)
		s.sayNow(locale.SayWebNoWebView())
		return
	}
	tab := &webTab{id: id, url: address, title: locale.WebUntitled(),
		page: page, canBack: true, canForward: true}
	s.web.tabs = append(s.web.tabs, tab)
	s.web.active = id
	s.web.status = ""
	s.installWebHandlers(tab)
	s.webPublish()
}

func (s *appState) webEnsureSession() {
	if s.web.session == nil {
		s.web.session = mygo.NewSession("xinkgo-webviewer")
	}
}

// installWebHandlers is the security boundary for a remote page: no Go
// bindings, no downloads, no permissions, no script-opened popups, only
// http(s) navigation, and no loopback/private hosts unless allowed.
func (s *appState) installWebHandlers(tab *webTab) {
	p := tab.page.Page()
	p.SetPermissionHandler(func(mygo.PermissionRequest) bool { return false })
	p.OnWillDownload(func(e *mygo.DownloadEvent) { e.PreventDefault() })
	p.SetWindowOpenHandler(func(req mygo.WindowOpenRequest) *mygo.WindowOptions {
		if req.UserInitiated && webviewer.IsAllowed(req.URL,
			s.coreOptions().WebViewer.AllowLocal) {
			url := req.URL
			// Defer out of the navigation callback: making a WebView
			// while the engine is asking to open one is re-entrant.
			go func() {
				mygo.RunOnMain(func() { s.webNewTab(url) })
			}()
		}
		return nil
	})
	p.OnWillNavigate(func(e *mygo.NavigateEvent) {
		if strings.HasPrefix(e.URL, "about:") {
			return
		}
		if !webviewer.IsAllowed(e.URL,
			s.coreOptions().WebViewer.AllowLocal) {
			e.PreventDefault()
			mygo.RunOnMain(func() { s.sayNow(locale.SayWebBlocked()) })
		}
	})
	p.OnDidNavigate(func(url string) {
		mygo.RunOnMain(func() {
			tab.url = url
			s.webPublish()
		})
	})
	p.OnPageTitleUpdated(func(e *mygo.TitleEvent) {
		mygo.RunOnMain(func() {
			tab.title = e.Title
			s.webPublish()
		})
	})
	p.OnDidFinishLoad(func() {
		mygo.RunOnMain(func() {
			s.web.status = ""
			s.webPublish()
		})
	})
	p.OnDidFailLoad(func(e *mygo.LoadError) {
		mygo.RunOnMain(func() {
			s.web.status = locale.WebLoadFailed()
			s.webPublish()
		})
	})
}

func (s *appState) webSelectTab(id string) {
	for _, one := range s.web.tabs {
		if one.id == id {
			s.web.active = id
			s.webPublish()
			return
		}
	}
}

// webShowPreview puts the note's Markdown preview back, keeping the web
// tabs alive in the strip so they can be chosen again.
func (s *appState) webShowPreview() {
	s.web.active = ""
	s.webPublish()
}

func (s *appState) webCloseTab(id string) {
	for i, one := range s.web.tabs {
		if one.id != id {
			continue
		}
		if one.page != nil {
			one.page.Destroy()
		}
		if one.reader != nil {
			one.reader.Destroy()
		}
		s.web.tabs = append(s.web.tabs[:i], s.web.tabs[i+1:]...)
		if s.web.active == id {
			s.web.active = ""
			if len(s.web.tabs) > 0 {
				s.web.active = s.web.tabs[len(s.web.tabs)-1].id
			}
		}
		s.webPublish()
		return
	}
}

// webPublish pushes the app's cached state to the workspace's preview
// pane. It is called on events, never once a frame.
func (s *appState) webPublish() {
	publish := func() {
		if s.ws == nil {
			return
		}
		tabs := make([]view.WebTabInfo, 0, len(s.web.tabs))
		for _, one := range s.web.tabs {
			var page ui.NativeWebView
			if one.readerOn && one.reader != nil {
				page = one.reader
			} else if one.page != nil {
				page = one.page
			}
			tabs = append(tabs, view.WebTabInfo{
				ID: one.id, Title: one.title, Address: one.url,
				Page: page, Reader: one.readerOn,
				Active: one.id == s.web.active,
			})
		}
		s.ws.WebTabs = tabs
		s.ws.WebTabActive = s.web.active
		s.ws.WebStatus = s.web.status
		if t := s.webActiveTab(); t != nil {
			s.ws.WebAddress = t.url
			s.ws.WebCanBack, s.ws.WebCanForward =
				t.canBack, t.canForward
		} else {
			s.ws.WebAddress = ""
		}
	}
	// Before the app is up there is no loop to run on; writing the state
	// directly is enough, and it lets tests see the published tabs.
	if !mygo.App.IsReady() {
		publish()
		return
	}
	mygo.RunOnMain(publish)
}

func (s *appState) webNavigate(address string) {
	t := s.webActiveTab()
	if t == nil || t.page == nil {
		s.webNewTab(address)
		return
	}
	o := s.coreOptions().WebViewer
	norm, err := webviewer.Normalize(address, o.SearchEngine, o.AllowLocal)
	if err != nil {
		s.sayNow(locale.SayWebBlocked())
		return
	}
	if t.readerOn {
		t.readerOn = false
	}
	_ = t.page.Page().LoadURL(norm)
	s.webPublish()
}

func (s *appState) webBack() {
	if t := s.webActiveTab(); t != nil && t.page != nil &&
		!t.readerOn {
		t.page.Page().GoBack()
	}
}

func (s *appState) webForward() {
	if t := s.webActiveTab(); t != nil && t.page != nil &&
		!t.readerOn {
		t.page.Page().GoForward()
	}
}

func (s *appState) webReload() {
	if t := s.webActiveTab(); t != nil && t.page != nil &&
		!t.readerOn {
		t.page.Page().Reload()
	}
}

func (s *appState) webHome() {
	o := s.coreOptions().WebViewer
	if home := strings.TrimSpace(o.Homepage); home != "" {
		s.webNavigate(home)
	}
}

// webReaderJS loads the extraction script once. It is a resource of the
// app, checked into the package by check-resources.sh.
func (s *appState) webReaderJS() (string, error) {
	if s.web.readerJS != "" {
		return s.web.readerJS, nil
	}
	data, err := webViewerAssets.ReadFile(webReaderAsset)
	if err != nil {
		return "", err
	}
	s.web.readerJS = string(data)
	return s.web.readerJS, nil
}

// webExtract runs the reader script in the page from Go. Eval may wait on
// the web engine, so it runs off the main thread; the result comes back to
// it before anything in the window is changed.
func (s *appState) webExtract(page *mygo.WebView,
	done func(map[string]any, error)) {
	js, err := s.webReaderJS()
	if err != nil {
		done(nil, err)
		return
	}
	go func() {
		value, err := page.Page().Eval(js +
			"\nreturn window.__xinkgoWebExtract();")
		fields, ok := value.(map[string]any)
		if err == nil && !ok {
			err = fmt.Errorf("the reader script gave no fields")
		}
		mygo.RunOnMain(func() { done(fields, err) })
	}()
}

func webField(fields map[string]any, key string) string {
	if text, ok := fields[key].(string); ok {
		return text
	}
	return ""
}

// webToggleReader swaps which of the tab's two pages the window places.
// The original page stays alive, so returning to it costs nothing.
func (s *appState) webToggleReader() {
	t := s.webActiveTab()
	if t == nil || t.page == nil {
		s.sayNow(locale.SayWebNoPage())
		return
	}
	if t.readerOn {
		t.readerOn = false
		s.webPublish()
		return
	}
	s.webExtract(t.page, func(fields map[string]any, err error) {
		if t != s.webActiveTab() {
			return
		}
		if err != nil {
			slog.Warn("reader view failed", "err", err)
			s.sayNow(locale.WebLoadFailed())
			return
		}
		if t.reader == nil {
			reader, err := s.win.NewWebView(mygo.WebViewOptions{
				URL: "about:blank",
				Page: mygo.PageOptions{
					Untrusted: true, Session: s.web.session,
				},
			})
			if err != nil {
				slog.Warn("reader page could not be made", "err", err)
				s.sayNow(locale.WebLoadFailed())
				return
			}
			t.reader = reader
		}
		t.reader.Page().LoadHTML(webField(fields, "html"), t.url)
		t.readerOn = true
		s.webPublish()
	})
}

// webSaveToVault writes the page into the vault as a Markdown note. It
// reuses the same ensureNote and template expansion the other plugins use.
func (s *appState) webSaveToVault() {
	t := s.webActiveTab()
	if t == nil || t.page == nil {
		s.sayNow(locale.SayWebNoPage())
		return
	}
	if s.root == "" {
		s.sayNow(locale.SayWebNoVault())
		return
	}
	s.webExtract(t.page, func(fields map[string]any, err error) {
		if err != nil {
			slog.Warn("page could not be extracted", "err", err)
			s.sayNow(locale.SayWebSaveFailed())
			return
		}
		title := webviewer.CleanTitle(webField(fields, "title"))
		address := webField(fields, "url")
		if address == "" {
			address = t.url
		}
		o := s.coreOptions()
		date, err := templating.Format(time.Now(), o.Templates.DateFormat)
		if err != nil {
			date = time.Now().Format("2006-01-02")
		}
		// A date format may contain slashes or colons; a note name may
		// not, or it would silently create nested folders.
		date = strings.NewReplacer("/", "-", "\\", "-", ":", "-").
			Replace(date)
		content := fmt.Sprintf(
			"---\nsource: %s\ndate: %s\n---\n\n# %s\n\n%s\n",
			address, date, title, webField(fields, "text"))
		if o.WebViewer.SaveTemplate != "" {
			text, err := s.readTemplate(o.WebViewer.SaveTemplate)
			if err == nil {
				expanded, err := templating.ExpandFile(
					o.WebViewer.SaveTemplate, text,
					s.templateVars(title))
				if err == nil {
					content = expanded
				} else {
					slog.Warn("save template not expanded", "err", err)
				}
			} else {
				slog.Warn("save template not read", "err", err)
			}
		}
		dir, ok := s.vaultPath(o.WebViewer.SaveFolder)
		if !ok {
			s.sayNow(locale.SayOptionOutsideVault())
			return
		}
		file := fs.UniqueName(dir, date+"-"+title+".md")
		rel := file
		if o.WebViewer.SaveFolder != "" {
			rel = filepath.Join(o.WebViewer.SaveFolder, file)
		}
		full, err := s.ensureNote(rel, content)
		if err != nil {
			slog.Warn("page not saved", "err", err)
			s.sayNow(locale.SayWebSaveFailed())
			return
		}
		s.sayNow(locale.SayWebSaved(filepath.Base(full)))
	})
}

func (s *appState) webOpenInBrowser() {
	t := s.webActiveTab()
	if t == nil || t.url == "" || strings.HasPrefix(t.url, "about:") {
		s.sayNow(locale.SayWebNoPage())
		return
	}
	if err := mygo.Shell.OpenExternal(t.url); err != nil {
		slog.Warn("page not opened in browser", "err", err)
	}
}

// webEnsureRules starts compiling the content rules off the main thread.
func (s *appState) webEnsureRules() {
	if !s.coreOptions().WebViewer.AdBlock || s.web.rules != nil ||
		s.web.rulesLoading || !mygo.WebViewAvailable() ||
		!mygo.App.IsReady() {
		return
	}
	s.web.rulesLoading = true
	go s.webCompileRules()
}

func (s *appState) webCompileRules() {
	o := s.coreOptions()
	data, err := webViewerAssets.ReadFile(webRulesAsset)
	if err != nil {
		slog.Warn("built-in content rules not embedded", "err", err)
		return
	}
	if custom := strings.TrimSpace(o.WebViewer.RuleFile); custom != "" {
		if path, ok := s.vaultPath(custom); ok {
			if own, err := os.ReadFile(path); err == nil {
				data = own
			} else {
				slog.Warn("custom content rules not read", "err", err)
			}
		}
	}
	rules, err := mygo.CompileContentRules("xinkgo-webviewer", string(data))
	mygo.RunOnMain(func() {
		s.web.rulesLoading = false
		if err != nil {
			slog.Warn("content rules not compiled", "err", err)
			s.sayNow(locale.SayWebRuleBad())
			return
		}
		s.web.rules = rules
		for _, one := range s.web.tabs {
			if one.page != nil {
				one.page.Page().SetContentRules(rules)
			}
		}
	})
}

// webOptionsChanged re-applies rules and availability after a settings
// change. It is called from setCoreOptions, on the main thread.
func (s *appState) webOptionsChanged() {
	if !s.coreEnabled("web-viewer") || !mygo.WebViewAvailable() {
		return
	}
	s.webEnsureRules()
}

// webDispose is the onDisable release hook: every native page and the
// pages and the preview tabs go away. It runs on the main thread.
func (s *appState) webDispose() {
	release := func() {
		for _, one := range s.web.tabs {
			if one.page != nil {
				one.page.Destroy()
			}
			if one.reader != nil {
				one.reader.Destroy()
			}
		}
		s.web.tabs = nil
		s.web.active = ""
		s.web.status = ""
		if s.ws != nil {
			s.ws.WebTabs = nil
			s.ws.WebTabActive = ""
			s.ws.WebAddress = ""
			s.ws.WebStatus = ""
		}
	}
	// Before the app is up there is no loop to run on, and no native page
	// could have been made; releasing the state directly is enough.
	if !mygo.App.IsReady() {
		release()
		return
	}
	mygo.RunOnMain(release)
}

// webQuit is called before the app exits. It clears the session only when
// the user asked for it, then releases the native pages.
func (s *appState) webQuit() {
	if s.coreEnabled("web-viewer") &&
		s.coreOptions().WebViewer.ClearOnQuit &&
		s.web.session != nil {
		if err := s.web.session.ClearData(mygo.SiteDataAll); err != nil {
			slog.Warn("web viewer data not cleared", "err", err)
		}
	}
	s.webDispose()
}

// webViewerSettings is the third Plugin options page.
func (s *appState) webViewerSettings(c *ui.Context) {
	th := c.Theme()
	if !mygo.WebViewAvailable() {
		ui.Text(c, locale.WebNoWebView()).
			FontSize(th.FontSize - 1).TextColor(th.TextMuted)
		return
	}
	o := &s.draft.coreOpts.WebViewer
	group(c, locale.GroupWebView(), func() {
		s.coreTextSetting(c, "web.homepage", locale.WebHomepage(),
			locale.WebHomepageNote(), &o.Homepage, false)
		s.coreTextSetting(c, "web.search", locale.WebSearchEngine(),
			locale.WebSearchEngineNote(), &o.SearchEngine, false)
		s.coreFolderSetting(c, "web.folder", locale.WebSaveFolder(),
			locale.WebSaveFolderNote(), &o.SaveFolder, true)
		s.coreFolderSetting(c, "web.template", locale.WebSaveTemplate(),
			locale.WebSaveTemplateNote(), &o.SaveTemplate, false)
		s.coreBoolSetting(c, "web.adblock", locale.WebAdBlock(),
			locale.WebAdBlockNote(), &o.AdBlock)
		s.coreTextSetting(c, "web.rule", locale.WebRuleFile(),
			locale.WebRuleFileNote(), &o.RuleFile, false)
		s.coreBoolSetting(c, "web.local", locale.WebAllowLocal(),
			locale.WebAllowLocalNote(), &o.AllowLocal)
		s.coreBoolSetting(c, "web.clear", locale.WebClearOnQuit(),
			locale.WebClearOnQuitNote(), &o.ClearOnQuit)
		buttons(c, locale.WebClearData(), "", func() {
			if ui.Button(c.Key("web-clear-data"),
				locale.WebClearData()).Clicked() {
				go s.webClearData()
			}
		})
		ui.Text(c, locale.WebLinkOpenNote()).
			FontSize(th.FontSize - 1).TextColor(th.TextMuted)
		if ui.Button(c.Key("web-go-preview"),
			locale.WebLinkOpenButton()).Clicked() {
			s.openSettingsTab(tabPreview)
		}
	})
}

func (s *appState) webClearData() {
	answer, err := mygo.Dialog.Message(mygo.MessageOptions{
		Type:          mygo.MessageQuestion,
		Title:         locale.WebClearConfirmTitle(),
		Message:       locale.WebClearConfirmMessage(),
		Buttons:       []string{locale.WebClearConfirmOk(), locale.Cancel()},
		DefaultButton: 1,
		CancelButton:  1,
	})
	if err != nil || answer.Button != 0 {
		return
	}
	mygo.RunOnMain(func() {
		s.webEnsureSession()
		if err := s.web.session.ClearData(mygo.SiteDataAll); err != nil {
			slog.Warn("web viewer data not cleared", "err", err)
		}
	})
}
