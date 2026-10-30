// Package logging sets up the structured logger the rest of the app uses
// through log/slog's default logger.
//
// Development runs log to the terminal, which is where a developer looks;
// installed apps write to a file in the application's log directory,
// because on Windows they have no console to write to. The file is opened
// in append mode and slog writes it unbuffered, so nothing is lost when
// the process quits without closing it.
package logging

import (
	"log/slog"
	"os"
	"path/filepath"
)

// FileName is the log file of an installed app.
const FileName = "xinkgo.log"

// LevelEnv is the environment variable that sets how much the app logs,
// for a developer who wants more than the default without editing files.
const LevelEnv = "XINKGO_LOG_LEVEL"

// Level turns a name into a slog.Level, falling back to info for one it
// does not know.
func Level(name string) slog.Level {
	switch name {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Setup makes slog's default logger write where the app wants it: the
// terminal while it is developed, dir/FileName otherwise.
func Setup(dir string, development bool) error {
	opts := &slog.HandlerOptions{Level: Level(os.Getenv(LevelEnv))}
	if development || dir == "" {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, opts)))
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, FileName),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(f, opts)))
	return nil
}
