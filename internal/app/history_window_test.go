package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/index"
	"github.com/helightxu/xinkgo/internal/snapshot"
	"github.com/helightxu/xinkgo/internal/view"
)

// appWithOne is an app with one note, `a.md`, in its vault, and a folder
// of its own for what is kept.
func appWithOne(t *testing.T, text string) (*appState, string) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "a.md")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	idx := index.New()
	idx.Build(root, []string{path})
	return &appState{dir: t.TempDir(), root: root, idx: idx}, path
}

// keep is a snapshot written into the app's folder, as a revision would.
func keep(t *testing.T, s *appState, name, text string) string {
	t.Helper()
	where, err := snapshot.Write(snapshot.Dir(s.dir), name, text)
	if err != nil {
		t.Fatal(err)
	}
	return where
}

// TestPutBack_whatWasKeptComesBack checks the one thing a snapshot is
// for: the note says again what it said before it was changed.
func TestPutBack_whatWasKeptComesBack(t *testing.T) {
	s, path := appWithOne(t, "# A\n\nthe words of a model\n")
	was := keep(t, s, "a.md", "# A\n\nwhat it said\n")
	s.putBack(was)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# A\n\nwhat it said\n" {
		t.Fatalf("note = %q, want what was kept", data)
	}
	if !strings.Contains(s.histStatus, "a.md") {
		t.Fatalf("status = %q, want it said of the note", s.histStatus)
	}
}

// TestPutBack_whatStoodThereIsKeptFirst checks that the way back is
// itself a thing one can come back from: what the note says now is kept
// before it is written over.
func TestPutBack_whatStoodThereIsKeptFirst(t *testing.T) {
	s, _ := appWithOne(t, "# A\n\nthe words of a model\n")
	keep(t, s, "a.md", "# A\n\nwhat it said\n")
	s.putBack(snapshot.List(snapshot.Dir(s.dir))[0])
	all := snapshot.List(snapshot.Dir(s.dir))
	if len(all) != 2 {
		t.Fatalf("kept = %d, want the one before and the one after",
			len(all))
	}
	data, err := os.ReadFile(all[0])
	if err != nil {
		t.Fatal(err)
	}
	// The newest is what stood there before it was put back.
	if !strings.Contains(string(data), "the words of a model") {
		t.Fatalf("newest = %q, want what stood there", data)
	}
}

// TestPutBack_aNameTheVaultHasNotIsSaid checks that a snapshot of a note
// that is not in this vault is said so, rather than made into one.
func TestPutBack_aNameTheVaultHasNotIsSaid(t *testing.T) {
	s, path := appWithOne(t, "# A\n\nmine\n")
	was := keep(t, s, "z.md", "# Z\n")
	s.putBack(was)
	if !strings.Contains(s.histStatus, "not in this vault") {
		t.Fatalf("status = %q, want it said", s.histStatus)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# A\n\nmine\n" {
		t.Fatalf("note = %q, want it left alone", data)
	}
	// Nothing was made in the vault for a name it does not have.
	if _, err := os.Stat(filepath.Join(s.root, "z.md")); !os.IsNotExist(
		err) {
		t.Fatal("a note was made for a snapshot of another vault")
	}
}

// TestPutBack_twoNotesOfThatNameIsNotGuessedAt checks that a name the
// vault has twice is not put back into either of them.
func TestPutBack_twoNotesOfThatNameIsNotGuessedAt(t *testing.T) {
	root := t.TempDir()
	one := filepath.Join(root, "one", "a.md")
	two := filepath.Join(root, "two", "a.md")
	for _, path := range []string{one, two} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("# A\n\nmine\n"),
			0o644); err != nil {
			t.Fatal(err)
		}
	}
	idx := index.New()
	idx.Build(root, []string{one, two})
	s := &appState{dir: t.TempDir(), root: root, idx: idx}
	keep(t, s, "a.md", "# A\n\nkept\n")
	s.putBack(snapshot.List(snapshot.Dir(s.dir))[0])
	if !strings.Contains(s.histStatus, "more than one") {
		t.Fatalf("status = %q, want it said", s.histStatus)
	}
	for _, path := range []string{one, two} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "kept") {
			t.Fatalf("%s was written into", path)
		}
	}
}

// TestPutBack_theNoteThatIsOpenIsMadeToShowIt checks that putting back
// reaches the editor, and not only the disk: a note that says one thing
// in the editor and another on the disk is a note whose next save throws
// what was put back away.
func TestPutBack_theNoteThatIsOpenIsMadeToShowIt(t *testing.T) {
	s, path := appWithOne(t, "# A\n\nthe words of a model\n")
	was := keep(t, s, "a.md", "# A\n\nwhat it said\n")
	s.ws = view.NewWorkspace(config.Default())
	open, err := s.ws.Tabs.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	s.putBack(was)
	if got := open.Editor.Buffer.String(); got != "# A\n\nwhat it said\n" {
		t.Fatalf("editor = %q, want what was put back", got)
	}
	if open.Note.Dirty {
		t.Error("the note is said to have changes the disk does not")
	}
}

// TestLoadKept_theNewestFirstAndEachSaysWhatItWasOf checks what the
// window is shown: the newest first, each saying which note it was of
// and enough of it to tell one from another.
func TestLoadKept_theNewestFirstAndEachSaysWhatItWasOf(t *testing.T) {
	s, _ := appWithOne(t, "# A\n\nmine\n")
	keep(t, s, "a.md", "\n\n# A\n\nthe first kept\n")
	keep(t, s, "b.md", "# B\n\nthe second kept\n")
	s.loadKept()
	if len(s.kept) != 2 {
		t.Fatalf("kept = %d, want both", len(s.kept))
	}
	if s.kept[0].Name != "b.md" || s.kept[1].Name != "a.md" {
		t.Fatalf("names = %q %q, want the newest first",
			s.kept[0].Name, s.kept[1].Name)
	}
	if s.kept[0].Preview != "# B" {
		t.Fatalf("preview = %q, want the first line of it",
			s.kept[0].Preview)
	}
	// A line of nothing at the beginning is not what is shown.
	if s.kept[1].Preview != "# A" {
		t.Fatalf("preview = %q, want the first line that says something",
			s.kept[1].Preview)
	}
	if s.kept[0].When == "" || s.kept[1].When == "" {
		t.Fatal("when it was taken was not said")
	}
}

// TestFirstWords_enoughToTellWhichOneItIs checks that a long line is cut
// short, and that a snapshot with nothing in it says so.
func TestFirstWords_enoughToTellWhichOneItIs(t *testing.T) {
	long := strings.Repeat("a", 200)
	if got := firstWords(long); len([]rune(got)) != 81 {
		t.Fatalf("firstWords = %d runes, want 80 and a mark",
			len([]rune(got)))
	}
	if got := firstWords(""); got != "(nothing in it)" {
		t.Fatalf("firstWords = %q, want it said", got)
	}
}
