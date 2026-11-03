// This file is what a change of the configuration does to the running
// app: the appearance it takes, the language it speaks, and what is
// written down before the app is closed.

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
	"time"

	"github.com/egoist/mygo"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/keymap"
	"github.com/helightxu/xinkgo/internal/state"
	"github.com/helightxu/xinkgo/internal/vault"
)

// applyConfig is what the user chose taking effect: the window is told,
// and the appearance is set as it was chosen.
func (s *appState) applyConfig() {
	was := s.ws.Config.Basic.Theme
	s.ws.Config = s.config
	s.setAppearance()
	s.applyLanguage()
	// A scheme is put on by the view that is drawn, so a window that is
	// drawing nothing would keep the colors it had: the windows are told
	// to draw again when the scheme changed. The appearance says so of
	// itself.
	if s.config.Basic.Theme != was {
		s.redrawWindows()
	}
	// Saving as the user types, after the pause they chose.
	s.ws.Tabs.AutoSave = s.config.Editor.AutoSave
	s.ws.Tabs.AutoSaveDelay = time.Duration(
		s.config.Editor.AutoSaveDelayMs) * time.Millisecond
	s.ws.Tree.FontSize = s.config.Basic.TreeFont.Size
	// Which skills may be used is a choice, and so are the app's own
	// actions over a note.
	s.publishSkills()
	s.publishActions()
	// Which files the tree shows is a choice too: one that changed is
	// read again.
	if s.shownAll != s.config.Basic.ShowAllFiles {
		s.shownAll = s.config.Basic.ShowAllFiles
		s.rescan()
	}
	// The styles of WeChat are read from a folder the user names, which
	// this may have changed.
	s.loadWechatThemes()
	// The menu says what the keys are, and the keys may have changed.
	s.refreshMenu()
}

// applyLanguage is the language the app shows: the one the user chose in
// the settings, or the languages the machine reads when they left it to
// follow them. MyGo is told, and says its own controls, menus and dialogs
// in it as well; the menu is made again by OnLocaleChanged, whose words
// are the ones of the language the app shows now.
func (s *appState) applyLanguage() {
	if !mygo.App.IsReady() {
		return
	}
	mygo.App.SetLocale(s.config.Basic.LocaleTag())
}

// setAppearance is the app told how light or dark things are to be, as
// the user chose or as the desktop has it. It is the app's to tell while
// it runs: a test, or anything else that asks before the loop is up,
// would wait for a loop that is not there. The colors are the scheme's,
// and every window puts them on as it is drawn.
func (s *appState) setAppearance() {
	if !mygo.App.IsReady() {
		return
	}
	mygo.Theme.SetSource(mygo.ThemeSource(s.config.Basic.Appearance))
}

// toggleAppearance is the appearance turned the other way: light into
// dark and dark into light. Following the machine is a light appearance
// until the user says otherwise, so that turning once is a thing that
// shows.
func (s *appState) toggleAppearance() {
	next := config.AppearanceDark
	if s.config.Basic.Appearance == config.AppearanceDark {
		next = config.AppearanceLight
	}
	s.config.Basic.Appearance = next
	s.ws.Config = s.config
	s.setAppearance()
	s.redrawWindows()
	if err := s.config.Save(s.dir); err != nil {
		slog.Warn("the choice could not be written", "err", err)
	}
}

// redrawWindows asks every window to draw again: a scheme is put on by
// the view that is drawn, so a window that is drawing nothing keeps the
// colors it had until it draws the next frame. Telling the appearance
// module what it already knows is what makes them draw, and changes
// nothing else.
func (s *appState) redrawWindows() {
	if !mygo.App.IsReady() {
		return
	}
	mygo.Theme.SetSource(mygo.Theme.Source())
}

// refreshMenu builds the menu again: what it lists of the vaults changes
// when one is added, or another one is opened.
func (s *appState) refreshMenu() {
	mygo.App.SetMenu(mainMenu(
		keymap.Resolve(s.config.Editor.Keymap), s.vaults))
}

// rescan reads the vault's folder again, as when a file was created or
// removed outside the app.
func (s *appState) rescan() {
	if s.root == "" {
		return
	}
	root, err := vault.Scan(s.root, s.config.Basic.ShowAllFiles)
	if err != nil {
		slog.Warn("the vault could not be read again", "err", err)
		return
	}
	s.ws.Tree.SetRoot(root)
}

// saveOpenNotes writes which notes are open, and which one is shown.
func (s *appState) saveOpenNotes() {
	if s.dir == "" {
		return
	}
	// What is open belongs to the vault that is open: another keeps
	// its own, and has them back when it is the one shown.
	if id := s.vaultID(); id != "" {
		active := s.ws.Tabs.Active
		on := state.OpenNotes{Paths: s.ws.Tabs.Paths(), Active: &active}
		s.openNotes = s.openNotes.WithOpenNotes(id, on)
	}
	if err := state.SaveOpenNotes(s.dir, s.openNotes); err != nil {
		slog.Warn("the open notes could not be written", "err", err)
	}
}

// saveLayout writes how wide the panels are.
func (s *appState) saveLayout() {
	if err := state.SaveLayout(s.dir, s.ws.Layout); err != nil {
		slog.Warn("the layout could not be written", "err", err)
	}
}
