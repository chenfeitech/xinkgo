package app

import (
	"github.com/egoist/mygo"
	"github.com/helightxu/xinkgo/internal/sysinfo"
)

// mainMenu is the application's menu bar: the platform's own menus where
// it has them, and the app's items on top of them.
func mainMenu(info sysinfo.Info) *mygo.Menu {
	return mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: "File", Submenu: []*mygo.MenuItem{
			{Label: "New Window", Accelerator: "CmdOrCtrl+N",
				Click: func(*mygo.MenuItem, *mygo.Window) {
					openMainWindow()
				}},
			{Label: "Open Web Window",
				Click: func(*mygo.MenuItem, *mygo.Window) {
					openWebWindow(info)
				}},
			mygo.Separator(),
			{Role: mygo.RoleClose},
		}},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleViewMenu},
		{Role: mygo.RoleWindowMenu},
	})
}
