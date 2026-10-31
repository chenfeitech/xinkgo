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

// TestMainMenu_theSettingsAreInTheMenu checks the way to the settings a
// user is given: in the application's own menu, as on macOS, and in the
// File menu, which is the one that stands elsewhere.
func TestMainMenu_theSettingsAreInTheMenu(t *testing.T) {
	m := mainMenu(keymap.Resolve(nil), vault.Registry{})
	for _, menu := range []string{"app", "File"} {
		if _, ok := itemIn(m, menu, "Settings…"); !ok {
			t.Errorf("the %s menu has no way to the settings", menu)
		}
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
