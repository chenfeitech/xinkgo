package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// openMainWindow shows the app's window: the vault's tree, the notes that
// are open, and what a note looks like, all drawn by MyGo in Go. A window
// that is already there is brought back rather than opened again, which is
// what a second launch and a click on the dock's icon ask for.
func openMainWindow() {
	if current == nil || current.ws == nil {
		return
	}
	if current.win != nil && !current.win.IsDestroyed() {
		current.win.Show()
		current.win.Focus()
		return
	}
	current.win = mygo.NewWindow(mygo.WindowOptions{
		Title:     Name,
		Width:     1100,
		Height:    720,
		MinWidth:  640,
		MinHeight: 480,
		// Remembers where and how large the user left it.
		StateKey: "main",
		// The window shows the interface MyGo draws, not a web page.
		Content: ui.View(current.ws.View),
	})
}
