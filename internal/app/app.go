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

	"github.com/egoist/mygo"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/fs"
	"github.com/helightxu/xinkgo/internal/logging"
	"github.com/helightxu/xinkgo/internal/media"
	"github.com/helightxu/xinkgo/internal/session"
	"github.com/helightxu/xinkgo/internal/state"
	"github.com/helightxu/xinkgo/internal/sysinfo"
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
	// root is the folder of the vault that is open.
	root string

	watch  *mygo.FileWatcher
	cancel context.CancelFunc
}

// Run starts the application and does not return until it quits.
func Run() error {
	app := mygo.App
	app.SetName(Name)

	app.WhenReady(func() {
		info := sysinfo.Current(Name, version(app))
		st, err := boot(app)
		if err != nil {
			slog.Error("startup failed", "err", err)
			app.Quit()
			return
		}
		current = st
		// Bound for windows that show a page: mygo generate writes their
		// typed client from this and whatever else is bound here.
		mygo.Bind(system{info: info})
		app.SetMenu(mainMenu())
		openMainWindow()
		st.restore()
		if err := st.watchVault(); err != nil {
			slog.Warn("could not watch the vault", "err", err)
		}
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
	st := &appState{
		dir:    dir,
		config: cfg,
		vaults: vaults,
		layout: layout,
		ws:     ws,
	}
	ws.OnOpenVault = st.openVaultDialog
	ws.OnNewNote = st.newNote
	// What is open is written down every time it changes, so that a
	// crash loses the words of a note and not which notes were open.
	ws.Tabs.OnChange = st.saveOpenNotes
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
	slog.Info("vault open", "path", path)
	return nil
}

// rememberVault makes the folder the one that is open, adding it to the
// list when it is not there yet.
func (s *appState) rememberVault(path string) {
	for i := range s.vaults.Vaults {
		if s.vaults.Vaults[i].Path == path {
			id := s.vaults.Vaults[i].ID
			s.vaults.Active = &id
			return
		}
	}
	e := s.vaults.Add(path)
	s.vaults.Active = &e.ID
	if err := state.SaveVaults(s.dir, s.vaults); err != nil {
		slog.Warn("the vaults could not be written", "err", err)
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
		mygo.RunOnMain(func() {
			// Notes of another vault are closed: the window shows one
			// vault at a time.
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
		})
	}()
}

// newNote makes a note in the vault, and opens it.
func (s *appState) newNote() {
	if s.root == "" {
		return
	}
	name := fs.UniqueName(s.root, "untitled.md")
	path := filepath.Join(s.root, name)
	title := strings.TrimSuffix(name, ".md")
	if _, err := s.ws.Tabs.Create(path, "# "+title+"\n"); err != nil {
		slog.Warn("a note could not be made", "path", path, "err", err)
		return
	}
	s.rescan()
	s.ws.Tree.Reveal(path)
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
