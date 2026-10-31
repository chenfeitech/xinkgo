package app

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/helightxu/xinkgo/internal/keymap"
	"github.com/helightxu/xinkgo/internal/vault"
)

// itemIn is the item of that label in the submenu of the menu of that
// name, and whether there is one.
func itemIn(m *mygo.Menu, menu, label string) (*mygo.MenuItem, bool) {
	for _, it := range m.Items() {
		if it.Label != menu && it.Role != mygo.RoleAppMenu {
			continue
		}
		if it.Role == mygo.RoleAppMenu && menu != "app" {
			continue
		}
		for _, sub := range it.Submenu {
			if sub.Label == label {
				return sub, true
			}
		}
	}
	return nil, false
}

// TestMainMenu_theSettingsStandInOneMenu checks the way to the settings a
// user is given: one menu carries them, and not two — the
// application's own menu, as on macOS, or the File menu's item, which is
// the one that stands where that menu is not shown.
func TestMainMenu_theSettingsStandInOneMenu(t *testing.T) {
	m := mainMenu(keymap.Resolve(nil), vault.Registry{})
	shown := 0
	for _, menu := range []string{"app", "File"} {
		it, ok := itemIn(m, menu, "Settings…")
		if !ok || it.Hidden {
			continue
		}
		shown++
	}
	if shown != 1 {
		t.Errorf("the settings stand in %d menus, want one", shown)
	}
}

// TestMainMenu_theSettingsSayTheKeyTheyAreOpenedBy checks that the item
// says the key the keymap gives the settings, so that a user who moved
// the key is shown the one that opens them.
func TestMainMenu_theSettingsSayTheKeyTheyAreOpenedBy(t *testing.T) {
	m := mainMenu(keymap.Resolve(nil), vault.Registry{})
	app, ok := itemIn(m, "app", "Settings…")
	if !ok {
		t.Fatal("the application's menu has no way to the settings")
	}
	if app.Accelerator != "CmdOrCtrl+," {
		t.Errorf("accelerator = %q, want the key a first start has",
			app.Accelerator)
	}
	moved := mainMenu(keymap.Resolve(map[string]string{
		"OpenSettings": "cmd-shift-,",
	}), vault.Registry{})
	app, _ = itemIn(moved, "app", "Settings…")
	if app.Accelerator != "CmdOrCtrl+Shift+," {
		t.Errorf("accelerator = %q, want the key the user moved it to",
			app.Accelerator)
	}
}
