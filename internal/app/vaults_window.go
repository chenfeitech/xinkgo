package app

import (
	"log/slog"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/state"
	"github.com/helightxu/xinkgo/internal/vault"
)

// openVaultManager shows the window of the vaults the app knows, or
// brings it back when it is already open: there is one of them.
func (s *appState) openVaultManager() {
	if s.vaultsWin != nil && !s.vaultsWin.IsDestroyed() {
		s.vaultsWin.Show()
		s.vaultsWin.Focus()
		return
	}
	s.vaultsWin = mygo.NewWindow(mygo.WindowOptions{
		Title:     "Vaults",
		Width:     620,
		Height:    420,
		MinWidth:  480,
		MinHeight: 300,
		// The window shows the interface MyGo draws, not a web page.
		Content: ui.View(s.vaultsView),
	})
}

// vaultsView is the window of the vaults: where each stands, what is
// wrong with one that cannot be opened, and what may be done to them.
func (s *appState) vaultsView(c *ui.Context) {
	ui.Column(c).Fill().Children(func() {
		for _, v := range s.vaults.Vaults {
			v := v
			ui.Row(c).Children(func() {
				if s.vaultRename == v.ID {
					s.vaultNameField(c, v)
					return
				}
				ui.Column(c).Grow(1).Children(func() {
					ui.Text(c, v.Name).Bold()
					ui.Text(c, v.Path).TextColor(c.Theme().TextMuted)
				})
				if status := s.vaults.StatusOf(v.ID); !status.Usable() {
					ui.Text(c, status.String()).TextColor(
						c.Theme().Danger)
				}
				if ui.Button(c.Key("open-"+v.ID), "Open").Clicked() {
					s.switchVault(v.Path)
				}
				if ui.Button(c.Key("rename-"+v.ID), "Name again").
					Clicked() {
					s.vaultRename = v.ID
					s.vaultName = v.Name
				}
				if ui.Button(c.Key("relocate-"+v.ID), "Find it again").
					Clicked() {
					s.relocateVault(v.ID)
				}
				if ui.Button(c.Key("forget-"+v.ID), "Forget").Clicked() {
					s.forgetVault(v.ID)
				}
			})
		}
		if len(s.vaults.Vaults) == 0 {
			ui.Text(c, "no vaults yet").TextColor(c.Theme().TextMuted)
		}
		ui.Row(c).Children(func() {
			if ui.Button(c.Key("add-vault"), "Add a folder…").Clicked() {
				s.openVaultDialog()
			}
		})
	})
}

// vaultNameField is the name of a vault being asked for: Enter gives it,
// Escape leaves it as it was.
func (s *appState) vaultNameField(c *ui.Context, v vault.VaultEntry) {
	field := ui.TextInput(c.Key("vault-name-"+v.ID), &s.vaultName).Grow(1)
	if field.Submitted() {
		s.vaults.Rename(v.ID, s.vaultName)
		s.vaultRename = ""
		s.saveVaults()
		s.refreshMenu()
	}
	if field.Shortcut(0, ui.KeyEscape) {
		s.vaultRename = ""
	}
}

// relocateVault is asked where a vault has gone, and follows it: the
// folder was moved or named again outside the app.
func (s *appState) relocateVault(id string) {
	go func() {
		picked, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title:     "Where has the vault gone?",
			Directory: true,
		})
		if err != nil || len(picked) == 0 {
			return
		}
		path := picked[0]
		mygo.RunOnMain(func() {
			s.vaults.Relocate(id, path)
			s.saveVaults()
			s.refreshMenu()
			// The vault that is open may be the one that moved.
			if s.root != "" && s.vaults.StatusOf(id).Usable() {
				s.rescan()
			}
		})
	}()
}

// forgetVault takes a vault out of the list. The folder and the notes in
// it are left as they are: forgetting is not throwing away.
func (s *appState) forgetVault(id string) {
	v := s.vaults.ByID(id)
	if v != nil && s.root == v.Path {
		s.ws.Say("the vault that is open cannot be forgotten")
		return
	}
	s.vaults.Remove(id)
	s.saveVaults()
	s.refreshMenu()
}

// saveVaults writes the list of vaults down.
func (s *appState) saveVaults() {
	if err := state.SaveVaults(s.dir, s.vaults); err != nil {
		slog.Warn("the vaults could not be written", "err", err)
	}
}
