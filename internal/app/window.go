package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// openMainWindow opens the app's window: the vault's tree, the notes that
// are open, and what a note looks like, all drawn by MyGo in Go.
func openMainWindow() {
	mygo.NewWindow(mygo.WindowOptions{
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
