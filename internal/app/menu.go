package app

import (
	"github.com/egoist/mygo"
	"github.com/helightxu/xinkgo/internal/keymap"
	"github.com/helightxu/xinkgo/internal/vault"
)

// mainMenu is the application's menu bar: the platform's own menus where
// it has them, and the app's items on top of them. The keys are the ones
// the configuration's keymap leaves for each action, so that a key the
// user moved is the one the menu shows, and the vaults are the ones the
// user can open.
func mainMenu(bound []keymap.Resolved, vaults vault.Registry) *mygo.Menu {
	key := func(action string) string {
		for _, r := range bound {
			if r.Action == action {
				return keymap.Accelerator(r.Keystroke)
			}
		}
		return ""
	}
	// The vaults the user can open, each saying what is wrong with it
	// when it cannot be opened.
	vaultItems := func() []*mygo.MenuItem {
		items := make([]*mygo.MenuItem, 0, len(vaults.Vaults)+2)
		for _, v := range vaults.Vaults {
			v := v
			label := v.Name
			if s := vaults.StatusOf(v.ID); !s.Usable() {
				// A vault the app cannot open says so, rather than
				// failing when it is chosen.
				label += " (" + s.String() + ")"
			}
			items = append(items, &mygo.MenuItem{
				Label: label,
				Type:  mygo.MenuItemRadio,
				Checked: vaults.Active != nil &&
					*vaults.Active == v.ID,
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if current != nil {
						current.switchVault(v.Path)
					}
				},
			})
		}
		return items
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
			{Label: "Go to Note…", Accelerator: key("OpenCommandPalette"),
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if current != nil {
						current.ws.OpenPalette()
					}
				}},
			{Label: "Search in Vault",
				Accelerator: key("OpenSearch"),
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if current != nil {
						current.ws.OpenSearch()
					}
				}},
			{Label: "Ask AI", Accelerator: key("ToggleAiChat"),
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if current != nil {
						current.ws.ToggleAI()
					}
				}},
			{Label: "Settings…", Accelerator: "CmdOrCtrl+,",
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if current != nil {
						current.openSettings()
					}
				}},
			{Label: "Library tasks…",
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if current != nil {
						current.openTasks()
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
		{Label: "Vault", Submenu: append(
			vaultItems(),
			mygo.Separator(),
			&mygo.MenuItem{Label: "Open Another Folder…",
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if current != nil {
						current.openVaultDialog()
					}
				}},
		)},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleViewMenu},
		{Role: mygo.RoleWindowMenu},
	})
}
