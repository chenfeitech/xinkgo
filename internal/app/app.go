// Package app wires Xinkgo together: it names the application, reads what
// it needs from disk, opens its windows and menus, and runs it until the
// user quits.
//
// It is the only package that knows about the application's lifetime;
// everything below it is plain data or a view, and testable without a
// running app.
package app

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/egoist/mygo"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/logging"
	"github.com/helightxu/xinkgo/internal/sysinfo"
	"github.com/helightxu/xinkgo/internal/view"
)

// Name is what users see. It also decides the user data directory, so it
// is set before anything asks for a path.
const Name = "Xinkgo"

// Version is the version of a build that has none linked in, such as
// `go run`. Keep it the version in mygo.json.
const Version = "0.1.0"

// home is the state of the main window. The menu opens another window
// showing it, so it lives here and not in the window that created it.
var home *view.Home

// Run starts the application and does not return until it quits.
func Run() error {
	app := mygo.App
	app.SetName(Name)

	app.WhenReady(func() {
		info := sysinfo.Current(Name, version(app))
		if err := start(info); err != nil {
			slog.Error("startup failed", "err", err)
			app.Quit()
			return
		}
		home = view.NewHome(info)
		// The main window is native UI; a web window is one click away,
		// which is where a frontend joins the app later on.
		home.OnWeb = func() { openWebWindow(info) }
		app.SetMenu(mainMenu(info))
		openMainWindow()
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

// start loads the configuration, sets up logging and binds what a web
// window can call. It runs on the main thread, inside WhenReady, when the
// application's directories are known.
func start(info sysinfo.Info) error {
	app := mygo.App
	dir, err := app.Path(mygo.PathUserData)
	if err != nil {
		return fmt.Errorf("user data directory: %w", err)
	}
	cfg, err := config.Load(dir)
	if err != nil {
		// A configuration the app cannot read must not keep it from
		// starting: use the defaults and write them back.
		slog.Warn("configuration unusable, using the defaults",
			"err", err)
		cfg = config.Default()
		if err := cfg.Save(dir); err != nil {
			slog.Warn("could not rewrite the configuration", "err", err)
		}
	}
	logs, err := app.Path(mygo.PathLogs)
	if err != nil {
		return fmt.Errorf("log directory: %w", err)
	}
	if err := logging.Setup(logs, development()); err != nil {
		return fmt.Errorf("logging: %w", err)
	}
	// Bound for windows that show a page: mygo generate writes their
	// typed client from this and whatever else is bound here.
	mygo.Bind(system{info: info})
	slog.Info("starting", "app", info.String(), "data", dir,
		"development", development())
	return nil
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
