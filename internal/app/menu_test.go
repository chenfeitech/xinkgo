package app

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/i18n"
	"github.com/helightxu/xinkgo/internal/keymap"
	"github.com/helightxu/xinkgo/internal/locale"
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

// TestMainMenu_theLanguageItIsBuiltIn checks that the menu says its items
// in the language the app shows: unlike the window, a menu is built once
// and not drawn again every frame, so OnLocaleChanged makes it again.
func TestMainMenu_theLanguageItIsBuiltIn(t *testing.T) {
	// The language the catalogs are asked in, as MyGo asks them.
	i18n.SetLocaleFunc(func() string { return "zh-Hans" })
	t.Cleanup(func() { i18n.SetLocaleFunc(func() string { return "" }) })
	m := mainMenu(keymap.Resolve(nil), vault.Registry{})
	for label, want := range map[string]string{
		"文件":  locale.MenuFile(),
		"设置…": locale.MenuSettings(),
	} {
		if label != want {
			t.Fatalf("the test and the catalog disagree: %q, %q", label,
				want)
		}
	}
	if _, ok := itemIn(m, locale.MenuFile(), locale.MenuOpenVault()); !ok {
		t.Errorf("no way to open a vault: %v", labelsOf(m))
	}
	if _, ok := itemIn(m, "app", locale.MenuSettings()); !ok {
		t.Errorf("no way to the settings: %v", labelsOf(m))
	}
}

// labelsOf is every label of a menu and of its submenus, as a test says
// them when it cannot find one.
func labelsOf(m *mygo.Menu) []string {
	var out []string
	for _, it := range m.Items() {
		out = append(out, it.Label)
		for _, sub := range it.Submenu {
			out = append(out, sub.Label)
		}
	}
	return out
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
