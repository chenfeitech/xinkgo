package app

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
)

// aiReply is what an answer may be turned into besides a revision of the
// note: copied, appended to today's daily note, or appended to a note
// the user picks. A revision still goes through the diff review; these
// only add words, never replace them.
func (s *appState) aiReply(kind, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	switch kind {
	case "copy":
		mygo.Clipboard.WriteText(text)
		s.sayNow(locale.AiReplyCopied())
	case "daily":
		if !s.aiMayAppend() {
			s.sayNow(locale.AiReplyDenied())
			return
		}
		s.appendTextToDaily(text)
	case "note":
		if !s.aiMayAppend() {
			s.sayNow(locale.AiReplyDenied())
			return
		}
		go s.appendTextToPickedNote(text)
	}
}

func (s *appState) aiMayAppend() bool {
	return s.config.Ai.Permission == config.PermissionAppend
}

// aiUndo puts one note back as it was before the last revision, but only
// when the note still says exactly what the revision left there. The
// snapshots on disk remain the longer history.
func (s *appState) aiUndo(path, before, after string) string {
	data, err := os.ReadFile(path)
	if err != nil || string(data) != after {
		return locale.AiUndoConflict()
	}
	if err := s.writeNote(path, before); err != nil {
		return locale.AiUndoConflict()
	}
	s.afterAppend(path)
	return locale.AiUndoOk()
}

// appendTextToDaily writes the answer at the foot of today's note,
// making the note when it is not there yet. Daily notes' own settings
// decide the name and folder, so the answer lands beside the note the
// Daily notes plugin would open.
func (s *appState) appendTextToDaily(text string) {
	if s.root == "" {
		s.sayNow(locale.AiReplyFailed())
		return
	}
	rel, err := s.dailyRelPath(time.Now())
	if err != nil {
		s.sayNow(locale.AiReplyFailed())
		return
	}
	full, ok := s.vaultPath(rel)
	if !ok || s.appendToFile(full, text) != nil {
		s.sayNow(locale.AiReplyFailed())
		return
	}
	s.afterAppend(full)
	s.sayNow(locale.AiReplyAppended(filepath.Base(full)))
}

// appendTextToPickedNote asks for a Markdown file and appends to it.
// The dialog runs off the main thread; only the workspace updates come
// back to it.
func (s *appState) appendTextToPickedNote(text string) {
	picked, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Title: locale.AiReplyPickNote(),
		Filters: []mygo.FileFilter{{
			Name: "Markdown", Extensions: []string{"md"},
		}},
	})
	if err != nil || len(picked) == 0 {
		return
	}
	full := picked[0]
	if _, ok := s.relativeToVault(full); !ok {
		mygo.RunOnMain(func() {
			s.sayNow(locale.AiReplyFailed())
		})
		return
	}
	writeErr := s.appendToFile(full, text)
	mygo.RunOnMain(func() {
		if writeErr != nil {
			s.sayNow(locale.AiReplyFailed())
			return
		}
		s.afterAppend(full)
		s.sayNow(locale.AiReplyAppended(filepath.Base(full)))
	})
}

// appendToFile adds words at the end of a note. It never overwrites:
// an answer appended twice is two answers, which is better than a lost
// note.
func (s *appState) appendToFile(full, text string) error {
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(full,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString("\n\n" + text + "\n")
	return err
}

// afterAppend brings the tree, index and an open tab up to date with a
// file that grew on disk.
func (s *appState) afterAppend(full string) {
	if s.ws != nil {
		s.rescan()
	}
	if s.idx != nil {
		s.idx.Reindex(full)
	}
	if s.ws != nil && s.ws.Tabs != nil {
		s.ws.Tabs.PutText(full)
	}
}
