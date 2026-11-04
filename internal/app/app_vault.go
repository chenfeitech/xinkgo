// This file is the vault the app has open: opening one, keeping it, and
// switching to another — and the index that is worked out again for it.

// Package app wires Xinkgo together: it names the application, reads
// what it kept last time, opens its window, and keeps the vault's files
// and the open notes in step with the disk.
//
// It is the only package that knows about the application's lifetime;
// everything below it is a rule, a file, or a view, and can be tested
// without an application running.
package app

import (
	"log/slog"

	"github.com/egoist/mygo"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/state"
	"github.com/helightxu/xinkgo/internal/vault"
)

// vaultID is the identifier of the vault that is open, or nothing when
// the app knows of none.
func (s *appState) vaultID() string {
	if s.vaults.Active != nil {
		return *s.vaults.Active
	}
	return ""
}

// syncVaults is the list of vaults told to the window, as when one is
// added, named again or forgotten: the tree carries the way from one to
// another.
func (s *appState) syncVaults() {
	s.ws.Tree.Vaults = s.vaults
	// The menu is the platform's, and only there once the app is up:
	// a test, or anything else that asks before the loop is running,
	// would wait for a loop that is not there.
	if mygo.App.IsReady() {
		s.refreshMenu()
	}
}

// openVault makes the folder at path the vault that is open.
func (s *appState) openVault(path string) error {
	root, err := vault.Scan(path, s.config.Basic.ShowAllFiles)
	if err != nil {
		return err
	}
	s.root = path
	if s.plugins != nil {
		s.plugins.setRoot(path)
		s.plugins.refresh()
		s.plugins.notifyState("vault-open")
	}
	s.ws.Tree.SetRoot(root)
	s.rememberVault(path)
	s.rebuildIndex(root)
	// The skills of a vault are the vault's own: another vault, other
	// skills.
	s.discoverSkills()
	slog.Info("vault open", "path", path)
	return nil
}

// openVaultDialog asks the user for a folder, and opens it as a vault.
// The dialog blocks, so it is asked from another goroutine.
func (s *appState) openVaultDialog() {
	go func() {
		picked, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title:     locale.DialogOpenVault(),
			Directory: true,
		})
		if err != nil || len(picked) == 0 {
			return
		}
		path := picked[0]
		mygo.RunOnMain(func() { s.switchVault(path) })
	}()
}

// switchVault makes the vault at path the one that is open: the notes of
// the one that was are closed, for the window shows one vault at a time.
func (s *appState) switchVault(path string) {
	if path == s.root {
		return
	}
	// The notes of the vault being left are kept for it before the
	// window is given the other: they come back with it.
	s.saveOpenNotes()
	// A note of another vault is closed, and written before it goes.
	s.ws.Tabs.CloseOutside(path)
	if err := s.openVault(path); err != nil {
		s.ws.Say(locale.SayVaultNotOpened())
		slog.Warn("vault", "path", path, "err", err)
		return
	}
	s.restoreOpenNotes()
	s.saveOpenNotes()
	if err := s.watchVault(); err != nil {
		slog.Warn("could not watch the vault", "err", err)
	}
}

// switchToVault makes the vault of that identifier the one that is open,
// as the tree asks when the user chooses one.
func (s *appState) switchToVault(id string) {
	v := s.vaults.ByID(id)
	if v == nil {
		return
	}
	s.switchVault(v.Path)
}

// rememberVault makes the folder the one that is open, adding it to the
// list when it is not there yet.
func (s *appState) rememberVault(path string) {
	known := false
	if v := s.vaults.At(path); v != nil {
		id := v.ID
		s.vaults.Active = &id
		known = true
	}
	if !known {
		e := s.vaults.Add(path)
		s.vaults.Active = &e.ID
	}
	if err := state.SaveVaults(s.dir, s.vaults); err != nil {
		slog.Warn("the vaults could not be written", "err", err)
	}
	// A vault that is not in the list yet is in the menu from now on,
	// and in the tree's way from one vault to another.
	if !known {
		s.syncVaults()
	}
}

// reindexAll is the whole vault read again for what its notes say, as
// when files were brought in from outside it.
func (s *appState) reindexAll() {
	if root := s.ws.Tree.Root; root != nil {
		s.rebuildIndex(root)
	}
}

// rebuildIndex reads the whole vault for what its notes say, in the
// background: a vault of thousands of notes is no reason to keep the
// window waiting.
func (s *appState) rebuildIndex(root *vault.Entry) {
	paths := vault.Files(root)
	dir := s.root
	go func() {
		s.idx.Build(dir, paths)
		slog.Info("index built", "notes", len(paths))
	}()
}
