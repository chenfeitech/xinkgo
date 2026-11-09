package note

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/helightxu/xinkgo/internal/textfile"
)

// write puts text on disk as another program would, and moves the file's
// time forward so that the change is seen.
func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}
}

// TestOpen_readsTheDisk checks that a new note is what the file holds.
func TestOpen_readsTheDisk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.md")
	write(t, path, "# hello")
	n, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if n.Text != "# hello" || n.Dirty {
		t.Errorf("text %q dirty %v, want the file's text and no changes",
			n.Text, n.Dirty)
	}
}

// TestOpen_missingFileIsAnError checks that a note that is not there is
// said so.
func TestOpen_missingFileIsAnError(t *testing.T) {
	if _, err := Open(filepath.Join(t.TempDir(), "gone.md")); err == nil {
		t.Error("Open of a file that is not there gave no error")
	}
}

// TestOpenEncoded_keepsTheMarkAndLineEndsOnSave checks a file whose bytes
// are not plain UTF-8: what is read, typed, and written again keeps the
// mark and the line ends of the file the user opened.
func TestOpenEncoded_keepsTheMarkAndLineEndsOnSave(t *testing.T) {
	text := "one\r\n\u4e16\u754c\n"
	for _, enc := range []textfile.Encoding{
		textfile.UTF8, textfile.UTF8BOM, textfile.UTF16LE,
		textfile.UTF16BE,
	} {
		path := filepath.Join(t.TempDir(), "a.txt")
		data, err := textfile.Encode(text, enc)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		n, err := OpenEncoded(path, enc)
		if err != nil {
			t.Fatalf("OpenEncoded(%v): %v", enc, err)
		}
		if n.Text != text || n.Encoding != enc {
			t.Errorf("opened %q as %v, want %q as %v", n.Text,
				n.Encoding, text, enc)
		}
		n.SetText("two\r\n" + text)
		if err := n.Save(); err != nil {
			t.Fatalf("Save(%v): %v", enc, err)
		}
		again, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := textfile.Detect(again); got != enc {
			t.Errorf("saved mark = %v, want %v", got, enc)
		}
		reread, err := Open(path)
		if err != nil {
			t.Fatal(err)
		}
		if reread.Text != n.Text || reread.Encoding != enc {
			t.Errorf("reread %q as %v, want %q as %v", reread.Text,
				reread.Encoding, n.Text, enc)
		}
	}
}

// TestSetText_marksDirtyAndVersions checks that typing is a change the
// disk does not have, and that it is counted.
func TestSetText_marksDirtyAndVersions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.md")
	write(t, path, "one")
	n, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	before := n.Version
	n.SetText("two")
	if !n.Dirty || n.Version != before+1 {
		t.Errorf("dirty %v version %d, want dirty and one more than %d",
			n.Dirty, n.Version, before)
	}
	n.SetText("two")
	if n.Version != before+1 {
		t.Errorf("version %d, want no new version for the same text",
			n.Version)
	}
}

// TestSave_writesAtomically checks that the text reaches the file, that no
// temporary file is left, and that the note is clean afterwards.
func TestSave_writesAtomically(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.md")
	write(t, path, "old")
	n, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	n.SetText("new")
	if err := n.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Errorf("file holds %q, want new", data)
	}
	if n.Dirty {
		t.Error("the note is still dirty after being saved")
	}
	names, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if filepath.Ext(name.Name()) == ".tmp" ||
			name.Name() != "a.md" {
			t.Errorf("a temporary file is left: %s", name.Name())
		}
	}
}

// TestSave_keepsTheOldFileWhenItFails checks that a note is not lost when
// the disk cannot take the new text.
func TestSave_keepsTheOldFileWhenItFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: permissions do not apply")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "a.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, path, "old")
	n, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	n.SetText("new")
	// The directory cannot be written to, so the temporary file cannot be
	// made: the old text must still be there.
	if err := os.Chmod(filepath.Dir(path), 0o500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(filepath.Dir(path), 0o755)
	if err := n.Save(); err == nil {
		t.Error("Save wrote although the directory is read only")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old" {
		t.Errorf("file holds %q, want the old text", data)
	}
}

// TestTakeExternalText_takesWhatAnotherProgramWrote checks that the file's
// new text comes into the note.
func TestTakeExternalText_takesWhatAnotherProgramWrote(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.md")
	write(t, path, "mine")
	n, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	write(t, path, "theirs")
	text, changed := n.TakeExternalText()
	if !changed || text != "theirs" {
		t.Errorf("got %q changed %v, want theirs", text, changed)
	}
	if n.Dirty {
		t.Error("the note is dirty after taking the disk's text")
	}
}

// TestTakeExternalText_nothingChangedChecksNothing checks the case xink
// is careful about: the file's time moved but its text is the same, so
// the caret must not be moved for nothing.
func TestTakeExternalText_nothingChangedChecksNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.md")
	write(t, path, "same")
	n, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	// Same text, later time: as after another app saved without changes.
	future := time.Now().Add(time.Minute)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}
	if !n.DiskChanged() {
		t.Fatal("the change of time was not seen")
	}
	text, changed := n.TakeExternalText()
	if changed || text != "" {
		t.Errorf("got %q changed %v, want nothing to take", text, changed)
	}
	// Having taken it, the note no longer sees a change.
	if n.DiskChanged() {
		t.Error("the note still thinks the file changed")
	}
}

// TestDiskChanged checks what counts as a change outside the app.
func TestDiskChanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.md")
	write(t, path, "one")
	n, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if n.DiskChanged() {
		t.Error("a note that was just read has not changed on disk")
	}
	write(t, path, "two")
	if !n.DiskChanged() {
		t.Error("another program's edit was not seen")
	}
	if err := n.Save(); err != nil {
		t.Fatal(err)
	}
	if n.DiskChanged() {
		t.Error("a note that was just saved has not changed on disk")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if n.DiskChanged() {
		t.Error("a file that is gone does not count as changed")
	}
}

// TestSave_createsAFileThatIsNotThereYet checks saving a note the file of
// which another program removed.
func TestSave_createsAFileThatIsNotThereYet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.md")
	write(t, path, "text")
	n, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := n.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		t.Error("the file was not written again")
	}
}
