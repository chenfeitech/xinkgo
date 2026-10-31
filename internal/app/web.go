package app

import (
	"html"
	"log/slog"
	"net/url"
	"strings"

	"github.com/egoist/mygo"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/sysinfo"
)

// system is a service a page can call: `mygo generate` writes its typed
// TypeScript client from the services the app binds. Xinkgo's windows are
// native UI today; this is what a web window reaches when one is added.
type system struct {
	info sysinfo.Info
}

// Info returns what the app knows about itself and its machine.
func (s system) Info() sysinfo.Info { return s.info }

// webPage is the page of a web window: a window MyGo does not draw, whose
// contents the system's webview shows (WebView2 on Windows). It has the
// app's values where a real frontend would call Go for them.
const webPage = `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>{{name}}</title>
  <style>
    :root { color-scheme: light dark;
            font-family: system-ui, sans-serif; }
    body { height: 100vh; margin: 0;
           display: grid; place-content: center; text-align: center; }
    code { font-size: 12px; opacity: .7; }
  </style>
</head>
<body>
  <h1>{{name}}</h1>
  <p id="info">…</p>
  <p><code>{{runtime}}</code></p>
  <script>
    mygo.call("system.Info").then((info) => {
      document.getElementById("info").textContent =
        info.name + " " + info.version;
    });
  </script>
</body>
</html>`

// openWebWindow opens a window whose page the system webview shows. It
// needs the webview of the platform: WebView2 on Windows, which Windows 11
// ships and Windows 10 installs as the WebView2 Runtime.
func openWebWindow(info sysinfo.Info) {
	page := strings.NewReplacer(
		"{{name}}", html.EscapeString(info.Name),
		"{{runtime}}", html.EscapeString(info.String()),
	).Replace(webPage)

	win := mygo.NewWindow(mygo.WindowOptions{
		Title:  Name + " · Web",
		Width:  720,
		Height: 480,
	})
	win.Page().LoadHTML(page, "")
}

// openWebURL opens a page in a window of this app's own, and not in the
// user's browser: what a link is asked to do when the settings say links
// are opened in the app.
//
// It needs the webview of the platform — WebView2 on Windows, which
// Windows 11 ships and Windows 10 installs as the WebView2 Runtime — and
// says so when there is none rather than leaving a click that does
// nothing.
func openWebURL(address string) error {
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:  pageTitle(address),
		Width:  900,
		Height: 640,
	})
	return win.Page().LoadURL(address)
}

// pageTitle is what such a window is called: where the page is, and not
// the whole of its address, which is longer than a title can hold.
func pageTitle(address string) string {
	parsed, err := url.Parse(address)
	if err != nil || parsed.Host == "" {
		return Name + " · Web"
	}
	return parsed.Host
}

// openOutside is a link that is not a note of the vault: a page on the web
// goes where the user said links should go — their own browser, or a
// window of this app's — and a link that cannot be opened is said in the
// log, for there is nothing the user can do about it.
func (s *appState) openOutside(address string) {
	if s.config.Preview.LinkOpen == config.LinkInApp {
		if err := openWebURL(address); err != nil {
			slog.Warn("the page could not be opened here",
				"url", address, "err", err)
			// Out of the app rather than nowhere at all: a link
			// that does nothing is worse than one that opens
			// somewhere else.
			if err := mygo.Shell.OpenExternal(address); err != nil {
				slog.Warn("the link could not be opened",
					"url", address, "err", err)
			}
		}
		return
	}
	if err := mygo.Shell.OpenExternal(address); err != nil {
		slog.Warn("the link could not be opened", "url", address,
			"err", err)
	}
}
