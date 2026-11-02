package app

import (
	"log/slog"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/state"
	"github.com/helightxu/xinkgo/internal/vault"
)

// vaultSettings is the folders the app knows: the one that is open, and
// the rest of them. They are kept here, in the window of the settings,
// and nowhere else: which folders the app knows is a thing the user
// chose, as a theme or a font is, and one app with two windows of its
// choices is a window too many.
//
// A folder may be added, opened, named again, found again where it has
// gone, and forgotten. Each is written down as it is done, as the rest
// of the settings are.
func (s *appState) vaultSettings(c *ui.Context) {
	th := c.Theme()
	group(c, locale.GroupVaultOpen(), func() {
		where := s.root
		if where == "" {
			where = locale.VaultNoneOpen()
		}
		buttons(c, locale.VaultWhere(), where, func() {
			if ui.Button(c.Key("open-other-vault"),
				locale.VaultOpenAnother()).
				Clicked() {
				s.openVaultDialog()
			}
		})
	})
	group(c, locale.GroupVaultsKnown(), func() {
		for _, v := range s.vaults.Vaults {
			s.vaultRow(c, v)
		}
		if len(s.vaults.Vaults) == 0 {
			ui.Column(c).FillWidth().
				Padding(th.Space(1), settingsRowPad).Children(func() {
				ui.Text(c, locale.VaultsNoneYet()).TextColor(th.TextMuted)
			})
		}
		buttons(c, locale.VaultAdd(), locale.VaultAddNote(),
			func() {
				if ui.Button(c.Key("add-vault"), locale.VaultAdd()).
					Clicked() {
					s.addVault()
				}
			})
	})
	// One the user asked to forget is forgotten after the rows are
	// drawn: a row must not be taken out from under the one that
	// follows it.
	if s.draft.vaultDrop != "" {
		s.forgetVault(s.draft.vaultDrop)
		s.draft.vaultDrop = ""
	}
}

// vaultRow is one vault: where it stands, what is wrong with it when the
// app cannot use it, and what may be done to it.
func (s *appState) vaultRow(c *ui.Context, v vault.VaultEntry) {
	th := c.Theme()
	open := s.vaults.Active != nil && *s.vaults.Active == v.ID
	ui.Column(c).FillWidth().Children(func() {
		ui.Row(c).FillWidth().Gap(th.Space(4)).AlignItems(ui.Center).
			Padding(th.Space(2), settingsRowPad).Children(func() {
			ui.Column(c).Grow(1).Gap(th.Space(0.5)).Children(func() {
				if s.draft.vaultRename == v.ID {
					s.vaultNameField(c, v)
					return
				}
				ui.Text(c, v.Name).Bold().SingleLine()
				ui.Text(c, v.Path).FontSize(th.FontSize - 1).
					TextColor(th.TextMuted).SingleLine()
				if status := s.vaults.StatusOf(v.ID); !status.Usable() {
					ui.Text(c, status.String()).TextColor(th.Danger).
						FontSize(th.FontSize - 1)
				}
			})
			ui.Row(c).Gap(th.Space(1)).Children(func() {
				// The one that is open is said so, rather than
				// offered to be opened again.
				if open {
					ui.Text(c, locale.VaultIsOpen()).TextColor(th.Success).
						FontSize(th.FontSize - 1)
				} else if ui.Button(c.Key("vault-open#"+v.ID),
					locale.VaultOpenButton()).
					Clicked() {
					s.switchVault(v.Path)
				}
				if ui.Button(c.Key("vault-rename#"+v.ID),
					locale.VaultRenameButton()).
					Clicked() {
					s.draft.vaultRename = v.ID
					s.draft.vaultName = v.Name
				}
				if ui.Button(c.Key("vault-relocate#"+v.ID),
					locale.VaultRelocate()).Clicked() {
					s.relocateVault(v.ID)
				}
				if ui.Button(c.Key("vault-forget#"+v.ID),
					locale.VaultForget()).
					Clicked() {
					s.draft.vaultDrop = v.ID
				}
			})
		})
		ui.Box(c).FillWidth().Height(1).Background(th.Border)
	})
}

// vaultNameField is the name of a vault being asked for: Enter gives it,
// Escape leaves it as it was.
func (s *appState) vaultNameField(c *ui.Context, v vault.VaultEntry) {
	field := ui.TextInput(c.Key("vault-name#"+v.ID), &s.draft.vaultName).
		Label(locale.VaultNameLabel()).Grow(1)
	if field.Submitted() {
		s.renameVault(v.ID, s.draft.vaultName)
		s.draft.vaultRename = ""
	}
	if field.Shortcut(0, ui.KeyEscape) {
		s.draft.vaultRename = ""
	}
}

// addVault is asked for a folder, and puts it in the list: the vault
// that is open is left open, for adding a folder is not opening it.
//
// The dialog blocks, so it is asked from another goroutine.
func (s *appState) addVault() {
	go func() {
		picked, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title:     locale.DialogAddVault(),
			Directory: true,
		})
		if err != nil || len(picked) == 0 {
			return
		}
		path := picked[0]
		mygo.RunOnMain(func() { s.keepVault(path) })
	}()
}

// keepVault puts the folder in the list of the vaults, without opening
// it. A folder that is one of them already is not put in twice, which
// would leave two rows for one folder.
func (s *appState) keepVault(path string) {
	if s.vaults.At(path) != nil {
		s.ws.Say(locale.SayVaultAlreadyKnown())
		return
	}
	s.vaults.Add(path)
	s.saveVaults()
	s.syncVaults()
}

// renameVault gives the vault another name, and writes the list down.
func (s *appState) renameVault(id, name string) {
	s.vaults.Rename(id, name)
	s.saveVaults()
	s.syncVaults()
}

// relocateVault is asked where a vault has gone, and follows it: the
// folder was moved, or named again outside the app.
func (s *appState) relocateVault(id string) {
	go func() {
		picked, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title:     locale.DialogRelocateVault(),
			Directory: true,
		})
		if err != nil || len(picked) == 0 {
			return
		}
		path := picked[0]
		mygo.RunOnMain(func() { s.moveVault(id, path) })
	}()
}

// moveVault is the vault standing somewhere else. The name follows the
// folder, unless the user had given it one of their own.
func (s *appState) moveVault(id, path string) {
	s.vaults.Relocate(id, path)
	s.saveVaults()
	s.syncVaults()
	// The vault that is open may be the one that moved.
	if s.root != "" && s.vaults.StatusOf(id).Usable() {
		s.rescan()
	}
}

// forgetVault takes a vault out of the list. The folder and the notes in
// it are left as they are: forgetting is not throwing away.
//
// The vault that is open cannot be forgotten: the window shows one vault
// at a time, and which one it shows next is for the user to say.
func (s *appState) forgetVault(id string) {
	v := s.vaults.ByID(id)
	if v != nil && s.root == v.Path {
		s.ws.Say(locale.SayOpenVaultNotForgotten())
		return
	}
	s.vaults.Remove(id)
	s.saveVaults()
	s.syncVaults()
}

// openVaultManager shows the settings with the group of the vaults
// chosen, as the menu and the tree ask for: the vaults are kept there,
// and the window is brought back rather than opened again when it
// already stands.
func (s *appState) openVaultManager() {
	s.openSettingsTab(tabVault)
}

// saveVaults writes the list of vaults down.
func (s *appState) saveVaults() {
	if err := state.SaveVaults(s.dir, s.vaults); err != nil {
		slog.Warn("the vaults could not be written", "err", err)
	}
}
