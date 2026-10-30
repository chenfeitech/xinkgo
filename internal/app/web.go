package app

import (
	"html"
	"strings"

	"github.com/egoist/mygo"
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
