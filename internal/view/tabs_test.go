package view

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// vaultWithNotes is a folder with two notes in it.
func vaultWithNotes(t *testing.T) (root, a, b string) {
	t.Helper()
	root = t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	a = filepath.Join(root, "a.md")
	b = filepath.Join(root, "notes", "b.md")
	if err := os.WriteFile(a, []byte("# a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("# b"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, a, b
}

// TestTabs_openTwiceDoesNotOpenTwice checks the rule xink keeps: one file
// is one tab, and opening it again only shows it.
func TestTabs_openTwiceDoesNotOpenTwice(t *testing.T) {
	_, a, _ := vaultWithNotes(t)
	tabs := NewTabs()
	first, err := tabs.Open(a)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	second, err := tabs.Open(a)
	if err != nil {
		t.Fatalf("Open again: %v", err)
	}
	if first != second {
		t.Error("opening the same file made another tab")
	}
	if len(tabs.Paths()) != 1 {
		t.Errorf("paths %v, want one", tabs.Paths())
	}
}

// TestTabs_openKeepsTheOrder checks that the order of the tabs is the
// order the notes were opened in.
func TestTabs_openKeepsTheOrder(t *testing.T) {
	_, a, b := vaultWithNotes(t)
	tabs := NewTabs()
	if _, err := tabs.Open(b); err != nil {
		t.Fatal(err)
	}
	if _, err := tabs.Open(a); err != nil {
		t.Fatal(err)
	}
	paths := tabs.Paths()
	if len(paths) != 2 || paths[0] != b || paths[1] != a {
		t.Errorf("paths %v, want b then a", paths)
	}
	if tabs.Active != a {
		t.Errorf("active %q, want the one opened last", tabs.Active)
	}
}

// TestTabs_openMissingFileIsAnError checks that a note that is not there
// is said so, and opens no tab.
func TestTabs_openMissingFileIsAnError(t *testing.T) {
	tabs := NewTabs()
	if _, err := tabs.Open(filepath.Join(t.TempDir(), "gone.md")); err == nil {
		t.Error("a file that is not there was opened")
	}
	if len(tabs.Paths()) != 0 {
		t.Errorf("paths %v, want none", tabs.Paths())
	}
}

// TestTabs_closeTheLastOneLeavesNothingOpen checks that closing the only
// tab leaves an empty page, not a window with no tab.
func TestTabs_closeTheLastOneLeavesNothingOpen(t *testing.T) {
	_, a, _ := vaultWithNotes(t)
	tabs := NewTabs()
	if _, err := tabs.Open(a); err != nil {
		t.Fatal(err)
	}
	tabs.Close(a)
	if len(tabs.Paths()) != 0 || tabs.Active != "" {
		t.Errorf("paths %v active %q, want nothing open", tabs.Paths(),
			tabs.Active)
	}
	if tabs.ActiveNote() != nil {
		t.Error("there is still a note shown")
	}
}

// TestTabs_closeTheActiveOneShowsAnother checks which tab is shown after
// the one that was shown is closed.
func TestTabs_closeTheActiveOneShowsAnother(t *testing.T) {
	_, a, b := vaultWithNotes(t)
	tabs := NewTabs()
	if _, err := tabs.Open(a); err != nil {
		t.Fatal(err)
	}
	if _, err := tabs.Open(b); err != nil {
		t.Fatal(err)
	}
	tabs.Close(b)
	if tabs.Active != a {
		t.Errorf("active %q, want a", tabs.Active)
	}
}

// TestTabs_closeOutsideKeepsTheVaultsNotes checks that opening another
// vault closes what is not in it.
func TestTabs_closeOutsideKeepsTheVaultsNotes(t *testing.T) {
	root, a, _ := vaultWithNotes(t)
	elsewhere := t.TempDir()
	other := filepath.Join(elsewhere, "c.md")
	if err := os.WriteFile(other, []byte("# c"), 0o644); err != nil {
		t.Fatal(err)
	}
	tabs := NewTabs()
	if _, err := tabs.Open(a); err != nil {
		t.Fatal(err)
	}
	if _, err := tabs.Open(other); err != nil {
		t.Fatal(err)
	}
	tabs.CloseOutside(root)
	paths := tabs.Paths()
	if len(paths) != 1 || paths[0] != a {
		t.Errorf("paths %v, want only the note of the vault", paths)
	}
}

// TestTabs_saveWritesTheTyping checks that what was typed reaches the
// file, and that the note stops being marked as changed.
func TestTabs_saveWritesTheTyping(t *testing.T) {
	_, a, _ := vaultWithNotes(t)
	tabs := NewTabs()
	n, err := tabs.Open(a)
	if err != nil {
		t.Fatal(err)
	}
	n.Editor.Buffer.Set("# a\n\nsomething new")
	tabs.Edited(n)
	if !tabs.Dirty(a) {
		t.Fatal("the note is not marked as changed after typing")
	}
	if err := tabs.Save(n); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# a\n\nsomething new" {
		t.Errorf("file holds %q", data)
	}
	if tabs.Dirty(a) {
		t.Error("the note is still marked as changed")
	}
}

// TestTabs_autoSaveWritesAfterThePause checks the debounce: the note is
// written once the typing stops, and not once a letter.
func TestTabs_autoSaveWritesAfterThePause(t *testing.T) {
	_, a, _ := vaultWithNotes(t)
	tabs := NewTabs()
	tabs.AutoSave = true
	tabs.AutoSaveDelay = 20 * time.Millisecond
	n, err := tabs.Open(a)
	if err != nil {
		t.Fatal(err)
	}
	n.Editor.Buffer.Set("one")
	tabs.Edited(n)
	n.Editor.Buffer.Set("one two")
	tabs.Edited(n)
	time.Sleep(150 * time.Millisecond)
	data, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "one two" {
		t.Errorf("file holds %q, want the last text", data)
	}
}

// TestTabs_autoSaveOffKeepsTheTextInMemory checks that turning auto-save
// off keeps the typing, so that nothing is lost.
func TestTabs_autoSaveOffKeepsTheTextInMemory(t *testing.T) {
	_, a, _ := vaultWithNotes(t)
	tabs := NewTabs()
	tabs.AutoSave = false
	n, err := tabs.Open(a)
	if err != nil {
		t.Fatal(err)
	}
	n.Editor.Buffer.Set("unsaved")
	tabs.Edited(n)
	time.Sleep(50 * time.Millisecond)
	data, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# a" {
		t.Errorf("file holds %q, want the old text", data)
	}
	if n.Note.Text != "unsaved" {
		t.Errorf("the note holds %q, want what was typed", n.Note.Text)
	}
}

// TestTabs_discardWritesNothing checks the tab of a file that is being
// taken away: its changes go with the file, and are not written back.
func TestTabs_discardWritesNothing(t *testing.T) {
	_, a, b := vaultWithNotes(t)
	tabs := NewTabs()
	tabs.AutoSaveDelay = 20 * time.Millisecond
	first, err := tabs.Open(a)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tabs.Open(b); err != nil {
		t.Fatal(err)
	}
	first.Editor.Buffer.Set("typed and thrown away")
	tabs.Edited(first)
	tabs.Discard(a)
	if len(tabs.Paths()) != 1 || tabs.Paths()[0] != b {
		t.Errorf("paths %v, want only b", tabs.Paths())
	}
	time.Sleep(100 * time.Millisecond)
	data, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# a" {
		t.Errorf("the file holds %q, want what it had", data)
	}
}

// TestTabs_renameKeepsTheTabInItsPlace checks a note whose file was
// renamed: the tab keeps its place and does not open a second one.
func TestTabs_renameKeepsTheTabInItsPlace(t *testing.T) {
	root, a, b := vaultWithNotes(t)
	tabs := NewTabs()
	if _, err := tabs.Open(b); err != nil {
		t.Fatal(err)
	}
	n, err := tabs.Open(a)
	if err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(root, "notes", "a.md")
	if err := os.Rename(a, moved); err != nil {
		t.Fatal(err)
	}
	tabs.Rename(a, moved)
	paths := tabs.Paths()
	if len(paths) != 2 || paths[0] != b || paths[1] != moved {
		t.Errorf("paths %v, want b then the note where it went", paths)
	}
	if tabs.Active != moved {
		t.Errorf("active %q, want the renamed note", tabs.Active)
	}
	if n.Note.Path != moved {
		t.Errorf("the note still points at %q", n.Note.Path)
	}
}

// TestTabs_reloadTakesWhatAnotherProgramWrote checks the case of a note
// edited in another app: its new text comes in.
func TestTabs_reloadTakesWhatAnotherProgramWrote(t *testing.T) {
	_, a, _ := vaultWithNotes(t)
	tabs := NewTabs()
	n, err := tabs.Open(a)
	if err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(time.Minute)
	if err := os.WriteFile(a, []byte("theirs"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(a, future, future); err != nil {
		t.Fatal(err)
	}
	changed, conflict := tabs.Reload(a)
	if !changed || conflict {
		t.Errorf("changed %v conflict %v, want the text taken in",
			changed, conflict)
	}
	if n.Note.Text != "theirs" {
		t.Errorf("the note holds %q, want theirs", n.Note.Text)
	}
	if n.Editor.Buffer.String() != "theirs" {
		t.Errorf("the editor shows %q, want theirs",
			n.Editor.Buffer.String())
	}
}

// TestTabs_reloadKeepsWhatTheUserTyped checks the other half of the rule:
// with changes of their own, the user's text stays and the app is told.
func TestTabs_reloadKeepsWhatTheUserTyped(t *testing.T) {
	_, a, _ := vaultWithNotes(t)
	tabs := NewTabs()
	n, err := tabs.Open(a)
	if err != nil {
		t.Fatal(err)
	}
	n.Editor.Buffer.Set("mine")
	tabs.Edited(n)
	future := time.Now().Add(time.Minute)
	if err := os.WriteFile(a, []byte("theirs"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(a, future, future); err != nil {
		t.Fatal(err)
	}
	changed, conflict := tabs.Reload(a)
	if changed || !conflict {
		t.Errorf("changed %v conflict %v, want a conflict and no change",
			changed, conflict)
	}
	if n.Note.Text != "mine" {
		t.Errorf("the note holds %q, want the user's text", n.Note.Text)
	}
}

// TestTabs_reloadOfTheSameTextTakesNothing checks the case xink is careful
// about: the file changed into what is already shown, so nothing happens
// and the caret is not moved.
func TestTabs_reloadOfTheSameTextTakesNothing(t *testing.T) {
	_, a, _ := vaultWithNotes(t)
	tabs := NewTabs()
	if _, err := tabs.Open(a); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(time.Minute)
	if err := os.Chtimes(a, future, future); err != nil {
		t.Fatal(err)
	}
	if changed, _ := tabs.Reload(a); changed {
		t.Error("the same text was taken in again")
	}
}

// TestTitle_markedWhenChanged checks what a tab says.
func TestTitle_markedWhenChanged(t *testing.T) {
	if got := Title("/vault/a.md", false); got != "a.md" {
		t.Errorf("title %q, want a.md", got)
	}
	if got := Title("/vault/a.md", true); got != "• a.md" {
		t.Errorf("title %q, want a mark before the name", got)
	}
}

// TestTabs_createMakesTheFileAndOpensIt checks a new note: the file is
// made, then opened.
func TestTabs_createMakesTheFileAndOpensIt(t *testing.T) {
	root, _, _ := vaultWithNotes(t)
	tabs := NewTabs()
	path := filepath.Join(root, "new.md")
	n, err := tabs.Create(path, "# new\n")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("the file was not made: %v", err)
	}
	if n.Note.Text != "# new\n" {
		t.Errorf("the note holds %q", n.Note.Text)
	}
	if len(tabs.Paths()) != 1 {
		t.Errorf("paths %v, want the new note open", tabs.Paths())
	}
}
