// Package app wires Xinkgo together: it names the application, reads
// what it kept last time, opens its window, and keeps the vault's files
// and the open notes in step with the disk.
//
// It is the only package that knows about the application's lifetime;
// everything below it is a rule, a file, or a view, and can be tested
// without an application running.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/index"
	"github.com/helightxu/xinkgo/internal/keymap"
	"github.com/helightxu/xinkgo/internal/logging"
	"github.com/helightxu/xinkgo/internal/media"
	"github.com/helightxu/xinkgo/internal/session"
	"github.com/helightxu/xinkgo/internal/skill"
	"github.com/helightxu/xinkgo/internal/state"
	"github.com/helightxu/xinkgo/internal/sysinfo"
	"github.com/helightxu/xinkgo/internal/task"
	"github.com/helightxu/xinkgo/internal/vault"
	"github.com/helightxu/xinkgo/internal/view"
)

// Name is what users see. It also decides the user data directory, so it
// is set before anything asks for a path.
const Name = "Xinkgo"

// Version is the version of a build that has none linked in, such as
// `go run`. Keep it the version in mygo.json.
const Version = "0.1.0"

// current is the application as it runs: one window, one vault, and the
// notes that are open.
var current *appState

// appState is what the application has open, and where it keeps it.
type appState struct {
	// dir is the user data directory: what is kept between two runs.
	dir string
	// config is what the user chose.
	config config.AppConfig
	// vaults is the folders the user can open, and which one is open.
	vaults vault.Registry
	// layout is how wide the window's panels are.
	layout state.Layout
	// ws is the window: the tree, the tabs, the editor and the preview.
	ws *view.Workspace
	// win is the window the workspace is drawn in, kept so that a second
	// launch can bring it back rather than open another.
	win *mygo.Window
	// settingsWin and vaultsWin are the windows of the settings and of
	// the vaults: one of each, brought back rather than opened again.
	settingsWin *mygo.Window
	vaultsWin   *mygo.Window
	// draft is what the settings window is changing.
	draft settings
	// shownAll is whether the tree was last read with every file in it.
	shownAll bool

	// vaultRename is the vault whose name is being asked for.
	vaultRename string
	vaultName   string
	// root is the folder of the vault that is open.
	root string
	// idx is what is known of the whole vault. It is worked out again
	// from the disk, and is never written down.
	idx *index.Index

	watch  *mygo.FileWatcher
	cancel context.CancelFunc

	// aiTalks is what has been said to a model, one conversation to a
	// note. It is read the first time it is asked for, and written when
	// a conversation changes.
	aiTalks state.AiSessions
	// skills are the skills a model may be told to use, and skillIssues
	// the ones found but that cannot be used. Both are read off the
	// disk, and are never written down.
	skills      []skill.Skill
	skillIssues []skill.Issue
	// tasksWin is the window of the long work over the vault.
	tasksWin *mygo.Window
	// taskDraft is the task being put together there, and work the one
	// that has been planned.
	taskDraft  taskDraft
	taskScroll ui.ScrollState
	work       *task.Task

	// histWin is the window of what was kept before notes were changed:
	// one of them, brought back rather than opened again.
	histWin *mygo.Window
	// kept is what that window shows, read when it is opened and again
	// after a snapshot has been put back.
	kept []keptOne
	// histStatus is what last came of putting one back.
	histStatus string
	histScroll ui.ScrollState
}

// Run starts the application and does not return until it quits.
func Run() error {
	app := mygo.App
	app.SetName(Name)

	// One instance at a time: a second launch hands what it was asked
	// to open to this one, which shows its window, and quits.
	if !app.RequestSingleInstanceLock() {
		return nil
	}
	app.OnSecondInstance(func(args []string, workingDir string) {
		openMainWindow()
	})

	app.WhenReady(func() {
		info := sysinfo.Current(Name, version(app))
		st, err := boot(app)
		if err != nil {
			slog.Error("startup failed", "err", err)
			app.Quit()
			return
		}
		current = st
		// Light or dark as the user chose, or as the desktop has it: the
		// code themes follow the appearance on their own.
		mygo.Theme.SetSource(mygo.ThemeSource(st.config.Basic.Theme))
		// Bound for windows that show a page: mygo generate writes their
		// typed client from this and whatever else is bound here.
		mygo.Bind(system{info: info})
		current.refreshMenu()
		openMainWindow()
		st.restore()
		if err := st.watchVault(); err != nil {
			slog.Warn("could not watch the vault", "err", err)
		}
	})

	// Leaving: what was open and how the window was laid out are written
	// one last time, in case the last change was not.
	app.OnBeforeQuit(func(*mygo.QuitEvent) {
		if current == nil {
			return
		}
		current.saveLayout()
		current.saveOpenNotes()
		current.stopWatch()
	})

	if isMacOS() {
		// Stay in the Dock with no window, as Mac apps do, and open one
		// when the user activates the app again.
		app.OnWindowAllClosed(func() {})
		app.OnActivate(func(hasVisibleWindows bool) {
			if !hasVisibleWindows {
				openMainWindow()
			}
		})
	} else {
		// On Windows and Linux the app is its windows: quit with the last.
		app.OnWindowAllClosed(app.Quit)
	}
	return app.Run()
}

// boot reads what the application kept last time, and makes the window
// that shows it. Nothing here may keep the app from starting: a file that
// is missing or unreadable gives a usable value.
func boot(app *mygo.Application) (*appState, error) {
	dir, err := app.Path(mygo.PathUserData)
	if err != nil {
		return nil, fmt.Errorf("user data directory: %w", err)
	}
	cfg, err := config.Load(dir)
	if err != nil {
		// A configuration the app cannot read must not keep it from
		// starting: use the defaults and say so.
		slog.Warn("configuration unusable, using the defaults", "err", err)
		cfg = config.Default()
	}
	logs, err := app.Path(mygo.PathLogs)
	if err != nil {
		return nil, fmt.Errorf("log directory: %w", err)
	}
	if err := logging.Setup(logs, development()); err != nil {
		return nil, fmt.Errorf("logging: %w", err)
	}
	vaults, err := state.LoadVaults(dir)
	if err != nil {
		slog.Warn("the vaults could not be read", "err", err)
	}
	layout, err := state.LoadLayout(dir)
	if err != nil {
		slog.Warn("the layout could not be read", "err", err)
	}

	ws := view.NewWorkspace(cfg)
	ws.Layout = layout
	// The index is worked out again from the vault, in the background,
	// as soon as one is open.
	idx := index.New()
	ws.Index = idx
	st := &appState{
		dir:    dir,
		config: cfg,
		vaults: vaults,
		layout: layout,
		idx:    idx,
		ws:     ws,
	}
	ws.OnOpenVault = st.openVaultDialog
	ws.OnNewNote = st.newNote
	// What the tree asks for when the user acts on a file in it.
	ws.Tree.OnNewNoteIn = st.newNoteIn
	ws.Tree.OnNewFolderIn = st.newFolderIn
	ws.Tree.OnRename = st.renameItem
	ws.Tree.OnDelete = st.deleteItem
	ws.Tree.OnMove = st.moveItem
	ws.Tree.OnImport = st.importFiles
	ws.Tree.OnRefresh = st.rescan
	// The settings are asked for from the foot of the tree as well: it
	// is the app that knows the window they are opened in.
	ws.Tree.OnOpenSettings = st.openSettings
	// What is open is written down every time it changes, so that a
	// crash loses the words of a note and not which notes were open.
	ws.Tabs.OnChange = st.saveOpenNotes
	// The widths of the panels are written once a drag has settled, and
	// not once a frame while it goes on.
	ws.OnLayoutChanged = st.saveLayout
	// What the window asks of a model goes through the app: it is the
	// app that knows which model the settings name.
	ws.OnAsk = st.ask
	// The app's own actions over a note are the settings' as well.
	ws.Actions = config.EnabledActions(cfg.Ai.Actions)
	// What was said about a note is kept between two runs.
	ws.OnAiLoad = st.aiLoad
	ws.OnAiSave = st.aiSave
	// What a note said before it was changed is kept, and whether it
	// still holds together after.
	ws.OnSnapshot = st.snapshotNote
	ws.OnVerify = st.verifyNote
	// A link that is not a note is opened by the app: it is the app
	// that knows where the user wants such links to go.
	ws.OnOpenOutside = st.openOutside
	ws.Tabs.OnSave = func(path string) {
		slog.Debug("saved", "path", path)
	}
	slog.Info("starting", "data", dir, "development", development())
	return st, nil
}

// restore opens again what was open when the app last ran: the vault, and
// the notes of that vault, in the order their tabs were in.
func (s *appState) restore() {
	v := s.vaults.ResolveActive()
	if v == nil {
		return
	}
	if err := s.openVault(v.Path); err != nil {
		slog.Warn("the vault could not be opened", "path", v.Path,
			"err", err)
		return
	}
	on, err := state.LoadOpenNotes(s.dir)
	if err != nil {
		slog.Warn("the open notes could not be read", "err", err)
	}
	paths, _ := session.PartitionExisting(on.Paths)
	for _, path := range paths {
		// Only the notes of the vault that is open come back.
		if !session.Within(s.root, path) {
			continue
		}
		if _, err := s.ws.Tabs.Open(path); err != nil {
			slog.Warn("a note could not be opened again", "path", path,
				"err", err)
		}
	}
	if on.Active != nil && session.Within(s.root, *on.Active) {
		s.ws.Tabs.Activate(*on.Active)
	}
	// Written back once: the notes that are gone are dropped from it.
	s.saveOpenNotes()
}

// openVault makes the folder at path the vault that is open.
func (s *appState) openVault(path string) error {
	root, err := vault.Scan(path, s.config.Basic.ShowAllFiles)
	if err != nil {
		return err
	}
	s.root = path
	s.ws.Tree.SetRoot(root)
	s.rememberVault(path)
	s.rebuildIndex(root)
	// The skills of a vault are the vault's own: another vault, other
	// skills.
	s.discoverSkills()
	slog.Info("vault open", "path", path)
	return nil
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

// rememberVault makes the folder the one that is open, adding it to the
// list when it is not there yet.
func (s *appState) rememberVault(path string) {
	known := false
	for i := range s.vaults.Vaults {
		if s.vaults.Vaults[i].Path == path {
			id := s.vaults.Vaults[i].ID
			s.vaults.Active = &id
			known = true
			break
		}
	}
	if !known {
		e := s.vaults.Add(path)
		s.vaults.Active = &e.ID
	}
	if err := state.SaveVaults(s.dir, s.vaults); err != nil {
		slog.Warn("the vaults could not be written", "err", err)
	}
	// A vault that is not in the list yet is in the menu from now on.
	if !known {
		s.refreshMenu()
	}
}

// openVaultDialog asks the user for a folder, and opens it as a vault.
// The dialog blocks, so it is asked from another goroutine.
func (s *appState) openVaultDialog() {
	go func() {
		picked, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title:     "Open a folder as a vault",
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
	s.ws.Tabs.CloseOutside(path)
	if err := s.openVault(path); err != nil {
		s.ws.Say("that folder could not be opened")
		slog.Warn("vault", "path", path, "err", err)
		return
	}
	s.saveOpenNotes()
	if err := s.watchVault(); err != nil {
		slog.Warn("could not watch the vault", "err", err)
	}
}

// applyConfig is what the user chose taking effect: the window is told,
// and the appearance is set as it was chosen.
func (s *appState) applyConfig() {
	s.ws.Config = s.config
	s.setAppearance()
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
	// The menu says what the keys are, and the keys may have changed.
	s.refreshMenu()
}

// setAppearance is the app told how things are to look, as the user
// chose or as the desktop has it. It is the app's to tell while it runs:
// a test, or anything else that asks before the loop is up, would wait
// for a loop that is not there.
func (s *appState) setAppearance() {
	if !mygo.App.IsReady() {
		return
	}
	mygo.Theme.SetSource(mygo.ThemeSource(s.config.Basic.Theme))
}

// toggleTheme is the appearance turned the other way: light into dark and
// dark into light. Following the machine is a light appearance until the
// user says otherwise, so that turning once is a thing that shows.
func (s *appState) toggleTheme() {
	next := config.ThemeDark
	if s.config.Basic.Theme == config.ThemeDark {
		next = config.ThemeLight
	}
	s.config.Basic.Theme = next
	s.ws.Config = s.config
	s.setAppearance()
	if err := s.config.Save(s.dir); err != nil {
		slog.Warn("the choice could not be written", "err", err)
	}
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
	active := s.ws.Tabs.Active
	on := state.OpenNotes{Paths: s.ws.Tabs.Paths(), Active: &active}
	if err := state.SaveOpenNotes(s.dir, on); err != nil {
		slog.Warn("the open notes could not be written", "err", err)
	}
}

// saveLayout writes how wide the panels are.
func (s *appState) saveLayout() {
	if err := state.SaveLayout(s.dir, s.ws.Layout); err != nil {
		slog.Warn("the layout could not be written", "err", err)
	}
}

// ignoreWatched is what the watch does not look at: the same names the
// tree never shows, so that a note being saved does not wake the app.
func ignoreWatched(path string, isDir bool) bool {
	base := filepath.Base(path)
	if isDir {
		return base == ".git" || base == "node_modules" ||
			base == "target" || base == media.AIDirName
	}
	return base == ".DS_Store" || strings.HasSuffix(base, ".md.tmp")
}

// version is the version mygo build linked in, or the one in the source
// when a build has none.
func version(app *mygo.Application) string {
	if v := app.Version(); v != "" {
		return v
	}
	return Version
}

// development reports whether this is a build someone is working on: it
// logs to the terminal and keeps the development features. mygo build
// sets MYGO_ENV=production in the apps it packages.
func development() bool {
	return os.Getenv("MYGO_ENV") != "production"
}
