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

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/index"
	"github.com/helightxu/xinkgo/internal/logging"
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
const Version = "0.1.1"

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
	// wechatView is the page the preview shows a note in when it is set
	// out as an article for WeChat: made of win the first time a style
	// is chosen, and kept, for a page is not made again and again.
	wechatView *mygo.WebView
	// plugins is the Obsidian-compatible plugin host: discovery, the
	// custom protocol, isolated pages, and the plugin manager window.
	plugins *pluginHost
	// wechatSeq is which filling of that page was asked for last, and
	// wechatUp whether the window is up, and the note on show with it:
	// only the last filling is put in the page, and none before the
	// window is.
	wechatSeq uint64
	wechatUp  bool
	// settingsWin is the window of the settings, which is where the
	// vaults are kept too: one of it, brought back rather than opened
	// again.
	settingsWin *mygo.Window
	// draft is what the settings window is changing.
	draft settings
	// shownAll is whether the tree was last read with every file in it.
	shownAll bool

	// root is the folder of the vault that is open.
	root string
	// openNotes is what notes were open in each vault: the window
	// shows one vault at a time, and one the user comes back to has
	// the notes they left open in it.
	openNotes state.OpenNotesByVault
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
	// web is the Web viewer plugin's window, tabs and native pages. It is
	// empty until the user opens the viewer for the first time.
	web webViewerState
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
		// The language before the menu: the menu takes its words from
		// the one the app shows, which is the configuration's.
		current.applyLanguage()
		// Light or dark as the user chose, or as the desktop has it: the
		// code themes follow the appearance on their own, and every
		// window puts the scheme the user chose on top of it.
		mygo.Theme.SetSource(mygo.ThemeSource(st.config.Basic.Appearance))
		// Bound for windows that show a page: mygo generate writes their
		// typed client from this and whatever else is bound here.
		mygo.Bind(system{info: info})
		current.refreshMenu()
		// A language chosen later, or the machine's changing while the
		// app follows it, is said again by the menu.
		mygo.App.OnLocaleChanged(func(string) {
			if current != nil {
				current.refreshMenu()
			}
		})
		openMainWindow()
		st.restore()
		// A style chosen the last time the app ran is the one the note
		// is set out in now. The window is up, and the notes of the
		// last run are open again: the one on show is set out, and not
		// each of them as they were opened.
		st.wechatUp = true
		st.showWechat()
		if st.plugins != nil {
			st.plugins.refresh()
			st.plugins.updateState()
			// The built-in core features stand or fall with the
			// choices in plugins.json, before any plugin page loads.
			st.plugins.publishCoreContributions()
			st.plugins.startEnabled()
		}
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
		current.webQuit()
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
	pluginState, err := state.LoadPlugins(dir)
	if err != nil {
		slog.Warn("the plugin state could not be read", "err", err)
	}
	resources, err := app.Path(mygo.PathResources)
	if err != nil {
		slog.Warn("the resources directory could not be read", "err", err)
	}

	ws := view.NewWorkspace(cfg)
	ws.Layout = layout
	ws.RecentModels = state.LoadRecentModels(dir)
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
	plugins, err := newPluginHost(st, dir, resources, pluginState)
	if err != nil {
		slog.Warn("the plugin host could not start", "err", err)
	} else {
		st.plugins = plugins
	}
	ws.OnOpenVault = st.openVaultDialog
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
	// The way from one vault to another stands in the tree's head: it
	// is the app that knows which folders the user can open.
	ws.Tree.OnSwitchVault = st.switchToVault
	ws.Tree.OnManageVaults = st.openVaultManager
	ws.Tree.OnOpenFolder = st.openVaultDialog
	// What is open is written down every time it changes, so that a
	// crash loses the words of a note and not which notes were open.
	// Another note on show is another article for WeChat, if that is
	// how the preview is set out.
	ws.Tabs.OnChange = func() {
		st.saveOpenNotes()
		st.refreshWechat()
		st.pluginStateChanged("file-open")
	}
	ws.Tabs.OnEdit = func(path string) {
		st.pluginStateChanged("editor-change")
	}
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
	// A file the app cannot edit is opened by the program the machine
	// has for it, as a PDF is previewed: the file itself, and not a
	// window of this app's.
	ws.OnOpenExternally = st.openExternally
	// A note is written out by the app: it is the app that asks where
	// the file is to go, and that puts it there.
	ws.OnExport = st.exportNote
	// A note set out as an article for WeChat, and the styles it may be
	// set out in: the app keeps the choice, fills the page, and owns
	// the clipboard it is copied to.
	st.loadWechatThemes()
	ws.OnWechatStyle = st.setWechatStyle
	ws.OnWechatCopy = st.copyWechat
	if st.plugins != nil {
		// Built-in core plugins and compatible plugins share the same
		// surfaces; one dispatcher decides who owns an ID.
		ws.OnPluginCommand = st.runCommand
		ws.OnPluginAction = st.runAction
		ws.OnPluginView = st.plugins.openPluginView
	}
	// Web viewer tabs live in the preview pane, so their controls are
	// wired on the workspace rather than on a window of their own.
	ws.OnWebNavigate = st.webNavigate
	ws.OnWebBack = st.webBack
	ws.OnWebForward = st.webForward
	ws.OnWebReload = st.webReload
	ws.OnWebHome = st.webHome
	ws.OnWebReader = st.webToggleReader
	ws.OnWebSave = st.webSaveToVault
	ws.OnWebOpenExternal = st.webOpenInBrowser
	ws.OnWebNewTab = func() { st.webNewTab("about:blank") }
	ws.OnWebSelectTab = st.webSelectTab
	ws.OnWebCloseTab = st.webCloseTab
	ws.OnWebShowPreview = st.webShowPreview
	ws.OnAiReply = st.aiReply
	ws.OnAiUndo = st.aiUndo
	ws.Tabs.OnSave = func(path string) {
		slog.Debug("saved", "path", path)
		// What is saved is what is set out: the article is laid out
		// again, so that it is the note as it stands now.
		st.refreshWechat()
		st.pluginStateChanged("vault-modify")
	}
	slog.Info("starting", "data", dir, "development", development())
	return st, nil
}

// restore opens again what was open when the app last ran: the vault, and
// the notes of that vault, in the order their tabs were in.
func (s *appState) restore() {
	// The vaults the user can open are told to the window before the
	// one that is open: the tree carries the way between them.
	s.syncVaults()
	v := s.vaults.ResolveActive()
	if v == nil {
		return
	}
	if err := s.openVault(v.Path); err != nil {
		slog.Warn("the vault could not be opened", "path", v.Path,
			"err", err)
		return
	}
	// What was open is kept one list to a vault: the file of a version
	// that knew one vault is read as the open vault's.
	all, err := state.LoadOpenNotes(s.dir)
	if err != nil {
		slog.Warn("the open notes could not be read", "err", err)
	}
	if len(all) == 0 {
		on, err := state.LoadLegacyOpenNotes(s.dir)
		if err != nil {
			slog.Warn("the open notes could not be read", "err", err)
		}
		if len(on.Paths) > 0 && s.vaults.Active != nil {
			all = all.WithOpenNotes(*s.vaults.Active, on)
		}
	}
	s.openNotes = all
	s.restoreOpenNotes()
	// The daily notes plugin may open today's note on startup. It runs
	// after the previous tabs are back, and Tabs.Open deduplicates by
	// path, so a note already restored is not opened twice.
	s.dailyOpenOnStartup()
}

// restoreOpenNotes opens again what was open in the vault that is open:
// the notes of another vault are left alone, for they are kept for the
// vault they are of.
func (s *appState) restoreOpenNotes() {
	on := s.openNotes.Of(s.vaultID())
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
