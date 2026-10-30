// Package view holds the windows' interfaces, drawn by MyGo in Go.
//
// A view is a method on the state it shows: MyGo calls it on the main
// thread whenever the window needs a frame, and it is the only place that
// changes the state. Nothing here reaches outside the package, so a view
// runs in a test without a window.
package view

import (
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/sysinfo"
)

// Home is the state of the main window.
type Home struct {
	info  sysinfo.Info
	name  string
	count int

	// OnWeb opens a window that shows a page in the system webview. The
	// main window is native UI; this is the seam the app grows a web
	// frontend into, and it is nil until the app sets it.
	OnWeb func()
}

// NewHome returns the state of a main window showing info.
func NewHome(info sysinfo.Info) *Home {
	return &Home{info: info}
}

// Count is the value the window shows, for tests and for the app's state.
func (h *Home) Count() int { return h.count }

// View draws the main window from the state.
func (h *Home) View(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Center().Gap(16).Padding(24).Children(func() {
		ui.Text(c, h.info.Name).FontSize(26).Bold()
		ui.Text(c, "version "+h.info.Version).FontSize(12).
			TextColor(t.TextMuted)

		ui.TextInput(c, &h.name).Placeholder("Your name").
			Label("Name").Width(240)
		greeting := "Hello!"
		if h.name != "" {
			greeting = "Hello, " + h.name + "!"
		}
		ui.Text(c, greeting).TextColor(t.TextMuted)

		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			if ui.Button(c.Key("decrement"), "−").
				Label("Decrement").Width(44).Clicked() {
				h.count--
			}
			ui.Textf(c, "%d", h.count).FontSize(20).Width(48).
				TextAlign(ui.Center)
			if ui.PrimaryButton(c.Key("increment"), "+").
				Label("Increment").Width(44).Clicked() {
				h.count++
			}
		})

		if h.OnWeb != nil {
			if ui.Button(c.Key("web"), "Open a web window").
				Label("Open a web window").Clicked() {
				h.OnWeb()
			}
		}
	})
}
