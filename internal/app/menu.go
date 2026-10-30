package app

import (
	"github.com/egoist/mygo"
	"github.com/helightxu/xinkgo/internal/keymap"
)

// mainMenu is the application's menu bar: the platform's own menus where
// it has them, and the app's items on top of them. The keys are the ones
// the configuration's keymap leaves for each action, so that a key the
// user moved is the one the menu shows, and the vaults are the ones the
// user can open.
func mainMenu(bound []keymap.Resolved) *mygo.Menu {
	key := func(action string) string {
		for _, r := range bound {
			if r.Action == action {
				return keymap.Accelerator(r.Keystroke)
			}
		}
		return ""
	}
	return mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: "File", Submenu: []*mygo.MenuItem{
			{Label: "Open Vault…", Accelerator: key("OpenFolder"),
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if current != nil {
						current.openVaultDialog()
					}
				}},
			{Label: "New Note", Accelerator: key("NewNote"),
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if current != nil {
						current.newNote()
					}
				}},
			{Label: "New Folder", Accelerator: key("NewFolder"),
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if current != nil {
						current.newFolderIn(current.ws.Tree.DirOf())
					}
				}},
			{Label: "Paste into Vault", Accelerator: "CmdOrCtrl+Shift+V",
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if current != nil {
						current.pasteIntoVault()
					}
				}},
			{Label: "Save", Accelerator: key("Save"),
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if current != nil {
						if err := current.ws.Tabs.SaveActive(); err != nil {
							current.ws.Say("the note could not be saved")
						}
					}
				}},
			{Label: "Close Tab", Accelerator: key("ClosePanel"),
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
