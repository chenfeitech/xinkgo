package app

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/media"
)

// watchVault asks the system to tell the app when the files of the vault
// change, as another program edits them or the user works in another app.
// The watch is not a poll: it is told, and it says what is different.
func (s *appState) watchVault() error {
	s.stopWatch()
	if s.root == "" {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	w, err := mygo.WatchFiles(ctx, s.root, mygo.FileWatchOptions{
		Recursive: true,
		Ignore:    ignoreWatched,
	})
	if err != nil {
		cancel()
		return err
	}
	s.watch, s.cancel = w, cancel
	go s.watchLoop(ctx, w)
	return nil
}

// stopWatch ends the watch of the vault that is open, as when another
// vault is opened in its place.
func (s *appState) stopWatch() {
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	if s.watch != nil {
		_ = s.watch.Close()
		s.watch = nil
	}
}

// watchLoop takes the events as they come, and hands them to the main
// thread, which is the only one that may change what is shown.
func (s *appState) watchLoop(ctx context.Context, w *mygo.FileWatcher) {
	for {
		events, err := w.Next(ctx)
		if err != nil {
			if ctx.Err() == nil {
				slog.Warn("the watch of the vault stopped", "err", err)
			}
			return
		}
		if len(events) == 0 {
			continue
		}
		mygo.RunOnMain(func() { s.handleEvents(events) })
	}
}

// handleEvents is what the app does with a change outside it: a note that
// is open takes in the new text, unless the user has changes of their own,
// and a file that appeared or went away makes the tree read again.
func (s *appState) handleEvents(events []mygo.FileEvent) {
	structure := false
	for _, e := range events {
		if e.Op != mygo.FileWrite {
			// Created, removed, renamed: the tree is not what it was.
			structure = true
			continue
		}
		path := filepath.Join(s.root, e.Path)
		// What the index knows of this note is read again, whether or
		// not it is open.
		s.idx.Reindex(path)
		changed, conflict := s.ws.Tabs.Reload(path)
		switch {
		case conflict:
			s.ws.Say(locale.WatchChangedOutside())
		case changed:
			s.ws.Say(locale.WatchTookInOutside())
		}
	}
	if structure {
		s.rescan()
		// What notes there are is different: the index is worked out
		// again, in the background.
		s.reindexAll()
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
