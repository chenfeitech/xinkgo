package app

import (
	"github.com/egoist/mygo"
	"github.com/helightxu/xinkgo/internal/keymap"
	xmd "github.com/helightxu/xinkgo/internal/markdown"
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
	// What the settings are opened by: the key the user gave them, or
	// the one a first start has.
	settingsKey := key("OpenSettings")
	if settingsKey == "" {
		settingsKey = "CmdOrCtrl+,"
	}
	openSettings := func(*mygo.MenuItem, *mygo.Window) {
		if current != nil {
			current.openSettings()
		}
	}
	// One menu carries the settings, and not two: on macOS that is the
	// application's own menu, where a Mac looks for them, and elsewhere
	// the File menu's item is the one that stands, for the
	// application's menu is not shown there at all.
	fileSettingsKey := settingsKey
	if isMacOS() {
		fileSettingsKey = ""
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
		// The application's own menu, where a Mac looks for the
		// settings. Off macOS this menu is not shown at all, and the
		// File menu's item is the one that stands.
		{Role: mygo.RoleAppMenu, Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleAbout},
			mygo.Separator(),
			{Label: "Settings…", Accelerator: settingsKey,
				Click: openSettings},
			mygo.Separator(),
			{Role: mygo.RoleServices},
			mygo.Separator(),
			{Role: mygo.RoleHide},
			{Role: mygo.RoleHideOthers},
			{Role: mygo.RoleUnhide},
			mygo.Separator(),
			{Role: mygo.RoleQuit},
		}},
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
			{Label: "Settings…", Accelerator: fileSettingsKey,
				Hidden: isMacOS(), Click: openSettings},
			{Label: "Library tasks…",
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if current != nil {
						current.openTasks()
					}
				}},
			{Label: "Snapshots…",
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if current != nil {
						current.openHistory()
					}
				}},
			{Label: "Save", Accelerator: key("Save"),
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if current == nil {
						return
					}
					// A revision standing before the user is not
					// written over: it is theirs to take or to
					// send back first.
					if current.ws.RevisionOpen() {
						current.ws.Say("put the revision away " +
							"first")
						return
					}
					if err := current.ws.Tabs.SaveActive(); err != nil {
						current.ws.Say("the note could not be saved")
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
			&mygo.MenuItem{Label: "Manage Vaults…",
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if current != nil {
						current.openVaultManager()
					}
				}},
		)},
		{Label: "Note", Submenu: []*mygo.MenuItem{
			{Label: "Rename…", Accelerator: key("RenameItem"),
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if current != nil {
						current.renameChosen()
					}
				}},
			{Label: "Delete", Accelerator: key("DeleteItem"),
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if current != nil {
						current.deleteChosen()
					}
				}},
		}},
		{Label: "Format", Submenu: []*mygo.MenuItem{
			{Label: "Bold", Accelerator: key("FormatBold"),
				Click: format(xmd.Bold)},
			{Label: "Italic", Accelerator: key("FormatItalic"),
				Click: format(xmd.Italic)},
			{Label: "Strikethrough",
				Accelerator: key("FormatStrikethrough"),
				Click:       format(xmd.Strikethrough)},
			{Label: "Inline code", Accelerator: key("FormatInlineCode"),
				Click: format(xmd.InlineCode)},
			{Label: "Link", Accelerator: key("FormatLink"),
				Click: format(xmd.FormatLink)},
		}},
		{
			Label: "View", Submenu: []*mygo.MenuItem{
				{Label: "Turn the light the other way",
					Accelerator: key("ToggleTheme"),
					Click: func(*mygo.MenuItem, *mygo.Window) {
						if current != nil {
							current.toggleTheme()
						}
					}},
			},
		},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleViewMenu},
		{Role: mygo.RoleWindowMenu},
	})
}

// format is what one of the Format items does: the marks go around what is
// selected in the note that is open.
func format(f xmd.Format) func(*mygo.MenuItem, *mygo.Window) {
	return func(*mygo.MenuItem, *mygo.Window) {
		if current != nil {
			current.ws.Format(f)
		}
	}
}
