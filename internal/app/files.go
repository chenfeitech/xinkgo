package app

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/helightxu/xinkgo/internal/fs"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/view"
)

// What the user can ask the tree to do to a file: make one, name it
// again, move it, take it away, or bring one in from outside.
//
// They are asked from another goroutine when they need a dialog, which
// waits for the user, and the vault is changed on the main thread, which
// is the only one that may change what is shown.

// newNote makes a note in the vault's own folder, and opens it.
func (s *appState) newNote() {
	s.newNoteIn(s.root)
}

// newNoteIn makes a note in dir, and opens it.
func (s *appState) newNoteIn(dir string) {
	if dir == "" {
		dir = s.root
	}
	if dir == "" {
		return
	}
	name := fs.UniqueName(dir, "untitled.md")
	path := filepath.Join(dir, name)
	title := strings.TrimSuffix(name, ".md")
	if _, err := s.ws.Tabs.Create(path, "# "+title+"\n"); err != nil {
		slog.Warn("a note could not be made", "path", path, "err", err)
		s.ws.Say(locale.SayNoteNotMade())
		return
	}
	s.rescan()
	s.idx.Reindex(path)
	s.ws.Tree.Reveal(path)
}

// newFolderIn makes a folder in dir, and shows it in the tree.
func (s *appState) newFolderIn(dir string) {
	if dir == "" {
		dir = s.root
	}
	if dir == "" {
		return
	}
	path := filepath.Join(dir, fs.UniqueName(dir, "new folder"))
	if err := os.MkdirAll(path, 0o755); err != nil {
		slog.Warn("a folder could not be made", "path", path, "err", err)
		s.ws.Say(locale.SayFolderNotMade())
		return
	}
	s.rescan()
	s.ws.Tree.Reveal(path)
}

// renameItem gives the file at path the name the user typed. A name that
// is the one it has already, but for the case of its letters, is no
// change at all.
func (s *appState) renameItem(path, name string) {
	if path == s.root {
		s.ws.Say(locale.SayVaultNotRenamed())
		return
	}
	newPath := fs.RenameName(path, name)
	if strings.EqualFold(newPath, path) {
		return
	}
	if err := fs.Move(path, newPath); err != nil {
		slog.Warn("a file could not be named again", "path", path,
			"err", err)
		s.ws.Say(locale.SayFileNotRenamed())
		return
	}
	// A note that is open follows its file: its tab keeps its place.
	s.ws.Tabs.Rename(path, newPath)
	s.idx.Remove(path)
	s.idx.Reindex(newPath)
	s.rescan()
	s.ws.Tree.Reveal(newPath)
}

// deleteItem takes the file at path away, after asking: a folder goes
// with everything in it, and the vault itself never does.
func (s *appState) deleteItem(path string) {
	if path == s.root {
		s.ws.Say(locale.SayVaultNotRemoved())
		return
	}
	go func() {
		answer, err := mygo.Dialog.Message(mygo.MessageOptions{
			Type:    mygo.MessageQuestion,
			Title:   locale.DeleteTitle(),
			Message: locale.DeleteMessage(view.PathOf(path)),
			Detail:  path,
			Buttons: []string{locale.TakeAway(), locale.Cancel()},
			// Cancel is what Enter and Escape do: taking a file away
			// should never be the answer to a slip of the key.
			DefaultButton: 1,
			CancelButton:  1,
		})
		if err != nil || answer.Button != 0 {
			return
		}
		mygo.RunOnMain(func() { s.removeItem(path) })
	}()
}

// chosen is the file the user has their eye on: the one the tree shows as
// chosen, and, when nothing is chosen there, the note that is open.
func (s *appState) chosen() string {
	if s.ws == nil {
		return ""
	}
	if s.ws.Tree.Selected != "" {
		return s.ws.Tree.Selected
	}
	if s.ws.Tabs != nil {
		return s.ws.Tabs.Active
	}
	return ""
}

// renameChosen asks for the name of the file the user has their eye on.
// The asking is the tree's: it is there that a file stands, and there
// that a name is given.
func (s *appState) renameChosen() {
	path := s.chosen()
	if path == "" {
		s.ws.Say(locale.SayNothingChosen())
		return
	}
	s.ws.Tree.Renaming = path
	s.ws.Tree.Name = view.PathOf(path)
}

// deleteChosen takes the file the user has their eye on away.
func (s *appState) deleteChosen() {
	path := s.chosen()
	if path == "" {
		s.ws.Say(locale.SayNothingChosen())
		return
	}
	s.deleteItem(path)
}

// removeItem is the removal itself, once the user has said yes.
func (s *appState) removeItem(path string) {
	// Forget the note before its file goes: a tab writing it back would
	// put it right where it was taken from.
	s.ws.Tabs.Discard(path)
	if err := fs.RemoveAll(s.root, path); err != nil {
		slog.Warn("a file could not be taken away", "path", path,
			"err", err)
		s.ws.Say(locale.SayNotRemoved())
		return
	}
	s.idx.Remove(path)
	s.rescan()
	s.saveOpenNotes()
}

// moveItem moves the file at path to a folder the user chooses.
func (s *appState) moveItem(path string) {
	if path == s.root {
		s.ws.Say(locale.SayVaultNotMoved())
		return
	}
	go func() {
		picked, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title:     locale.DialogMoveTo(view.PathOf(path)),
			Directory: true,
		})
		if err != nil || len(picked) == 0 {
			return
		}
		dir := picked[0]
		mygo.RunOnMain(func() { s.moveItemTo(path, dir) })
	}()
}

// moveItemTo is the move itself: a file of one name already there gets
// another, as one made in the folder would.
func (s *appState) moveItemTo(path, dir string) {
	name := filepath.Base(path)
	target := filepath.Join(dir, name)
	if dir != filepath.Dir(path) {
		target = filepath.Join(dir, fs.UniqueName(dir, name))
	}
	if err := fs.Move(path, target); err != nil {
		slog.Warn("a file could not be moved", "path", path, "err", err)
		s.ws.Say(locale.SayNotMoved())
		return
	}
	s.ws.Tabs.Rename(path, target)
	s.idx.Remove(path)
	s.idx.Reindex(target)
	s.rescan()
	s.ws.Tree.Reveal(target)
}

// importFiles brings files from outside the vault into dir: they are
// copied, so that what the user dragged in stays where it was too.
func (s *appState) importFiles(dir string, paths []string) {
	if dir == "" {
		dir = s.root
	}
	if dir == "" {
		return
	}
	go func() {
		brought, failed := 0, 0
		for _, path := range paths {
			name := fs.UniqueName(dir, filepath.Base(path))
			if err := fs.CopyAll(path, filepath.Join(dir, name)); err != nil {
				failed++
				slog.Warn("a file could not be brought in", "path", path,
					"err", err)
				continue
			}
			brought++
		}
		mygo.RunOnMain(func() {
			s.rescan()
			s.reindexAll()
			switch {
			case brought > 0 && failed > 0:
				s.ws.Say(locale.SayBroughtIn(brought) + ", " +
					locale.SayCouldNotBeBroughtIn(failed))
			case brought > 0:
				s.ws.Say(locale.SayBroughtIn(brought))
			default:
				s.ws.Say(locale.SayNothingBroughtIn())
			}
		})
	}()
}

// pasteIntoVault brings what the clipboard holds into the vault: files
// are copied in, a picture is written as one, and words become a note.
func (s *appState) pasteIntoVault() {
	dir := s.ws.Tree.DirOf()
	go func() {
		if files, err := mygo.Clipboard.ReadFiles(); err == nil &&
			len(files) > 0 {
			mygo.RunOnMain(func() { s.importFiles(dir, files) })
			return
		}
		if picture := mygo.Clipboard.ReadImage(); len(picture) > 0 {
			mygo.RunOnMain(func() { s.pastePicture(dir, picture) })
			return
		}
		text := strings.TrimSpace(mygo.Clipboard.ReadText())
		if text == "" {
			return
		}
		mygo.RunOnMain(func() { s.pasteText(dir, text) })
	}()
}

// pastePicture writes the picture the clipboard holds into dir.
func (s *appState) pastePicture(dir string, picture []byte) {
	if dir == "" {
		dir = s.root
	}
	path := filepath.Join(dir, fs.UniqueName(dir, "pasted.png"))
	if err := os.WriteFile(path, picture, 0o644); err != nil {
		slog.Warn("the picture could not be written", "path", path,
			"err", err)
		s.ws.Say(locale.SayPictureNotWritten())
		return
	}
	s.rescan()
	s.idx.Reindex(path)
	s.ws.Tree.Reveal(path)
}

// pasteText writes the words the clipboard holds into a note of their
// own, and opens it.
func (s *appState) pasteText(dir, text string) {
	if dir == "" {
		dir = s.root
	}
	name := fs.UniqueName(dir, "pasted.md")
	path := filepath.Join(dir, name)
	if _, err := s.ws.Tabs.Create(path, text+"\n"); err != nil {
		slog.Warn("the words could not be written", "path", path, "err",
			err)
		s.ws.Say(locale.SayTextNotWritten())
		return
	}
	s.rescan()
	s.idx.Reindex(path)
	s.ws.Tree.Reveal(path)
}
