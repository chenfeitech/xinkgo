package app

import (
	"github.com/egoist/mygo"
)

// mainMenu is the application's menu bar: the platform's own menus where
// it has them, and the app's items on top of them.
func mainMenu() *mygo.Menu {
	return mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: "File", Submenu: []*mygo.MenuItem{
			{Label: "Open Vault…", Accelerator: "CmdOrCtrl+O",
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if current != nil {
						current.openVaultDialog()
					}
				}},
			{Label: "New Note", Accelerator: "CmdOrCtrl+N",
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if current != nil {
						current.newNote()
					}
				}},
			{Label: "Save", Accelerator: "CmdOrCtrl+S",
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if current != nil {
						if err := current.ws.Tabs.SaveActive(); err != nil {
							current.ws.Say("the note could not be saved")
						}
					}
				}},
			{Label: "Close Tab", Accelerator: "Ctrl+W",
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if current != nil {
						current.ws.Tabs.Close(current.ws.Tabs.Active)
					}
				}},
			mygo.Separator(),
			{Role: mygo.RoleClose},
		}},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleViewMenu},
		{Role: mygo.RoleWindowMenu},
	})
}
