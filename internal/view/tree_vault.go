// This file is the vault the tree is showing: which one it is, and the
// way another is chosen.

// Package view holds the windows' interfaces, drawn by MyGo in Go.
//
// A view is a method on the state it shows: MyGo calls it on the main
// thread whenever the window needs a frame, and it is the only place that
// changes the state. Nothing here reaches outside the package, so a view
// runs in a test without a window.
package view

import (
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/locale"
)

// vaultSwitch is the head of the tree: the vault that is open, and the
// way to another. The folders the user can open stand in its menu, the
// one that is open marked, and a folder the app cannot open says so
// rather than failing when it is chosen.
func (t *Tree) vaultSwitch(c *ui.Context, name string) {
	th := c.Theme()
	label := locale.TreeSwitchVault()
	head := ui.ButtonBase(c.Key("vault-switch")).Label(label).
		Tooltip(label).Grow(1).MinWidth(0).Radius(th.Radius)
	head.Menu(func(m *ui.Menu) { t.vaultMenu(m) })
	head.Children(func() {
		ui.Row(c).FillWidth().Gap(treeGap).AlignItems(ui.Center).
			Children(func() {
				ui.Icon(c, iconFolder).Size(treeIcon, treeIcon).
					TextColor(th.TextMuted)
				ui.Text(c, name).Bold().Grow(1).Ellipsis("…").NoWrap()
				ui.Icon(c, iconChevron).Size(treeChevron, treeChevron).
					TextColor(th.TextMuted)
			})
	})
}

// vaultMenu is what the head of the tree asks: which vault is open, and
// what may be done about them.
func (t *Tree) vaultMenu(m *ui.Menu) {
	for _, v := range t.Vaults.Vaults {
		v := v
		label := v.Name
		if s := t.Vaults.StatusOf(v.ID); !s.Usable() {
			label += " (" + s.String() + ")"
		}
		if m.Item(label).Radio(t.vaultID() == v.ID).Chosen() &&
			t.vaultID() != v.ID && t.OnSwitchVault != nil {
			t.OnSwitchVault(v.ID)
		}
	}
	if len(t.Vaults.Vaults) == 0 {
		m.Item(locale.TreeNoOtherVault()).Disabled(true)
	}
	m.Separator()
	if m.Item(locale.TreeOpenAnotherFolder()).Chosen() &&
		t.OnOpenFolder != nil {
		t.OnOpenFolder()
	}
	if m.Item(locale.TreeManageVaults()).Chosen() &&
		t.OnManageVaults != nil {
		t.OnManageVaults()
	}
}

// vaultName is what the tree calls the vault that is open: the name the
// user gave it, when it is one the app knows, and the folder's own
// otherwise.
// vaultID is the identifier of the vault that is open, or nothing when
// the app knows of none.
func (t *Tree) vaultID() string {
	if t.Vaults.Active != nil {
		return *t.Vaults.Active
	}
	return ""
}

func (t *Tree) vaultName() string {
	for _, v := range t.Vaults.Vaults {
		if v.ID == t.vaultID() && v.Name != "" {
			return v.Name
		}
	}
	if t.Root != nil {
		return t.Root.Name
	}
	return locale.TreeNoVault()
}
