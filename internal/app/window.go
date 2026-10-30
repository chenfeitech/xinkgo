package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// openMainWindow opens the app's window: native UI, drawn by MyGo from
// the view in internal/view.
func openMainWindow() {
	mygo.NewWindow(mygo.WindowOptions{
		Title:     Name,
		Width:     520,
		Height:    400,
		MinWidth:  360,
		MinHeight: 300,
		// Remembers where and how large the user left it.
		StateKey: "main",
		// The window shows the interface MyGo draws, not a web page.
		Content: ui.View(home.View),
	})
}
