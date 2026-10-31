package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/view"
)

// appWithWindow is an app with a window, in folders of its own.
func appWithWindow(t *testing.T) *appState {
	t.Helper()
	root := t.TempDir()
	return &appState{
		dir:  t.TempDir(),
		root: root,
		ws:   view.NewWorkspace(config.Default()),
	}
}

// TestToggleTheme_theOtherWayRound checks that turning the light once
// turns it, and that turning it twice is the same as never having: a key
// that only ever goes one way is no way to turn anything.
func TestToggleTheme_theOtherWayRound(t *testing.T) {
	s := appWithWindow(t)
	s.config = config.Default()
	s.config.Basic.Theme = config.ThemeLight
	s.toggleTheme()
	if s.config.Basic.Theme != config.ThemeDark {
		t.Fatalf("theme = %q, want dark", s.config.Basic.Theme)
	}
	// The window is told, and not only the settings: it is what is
	// asked how things are to look.
	if s.ws.Config.Basic.Theme != config.ThemeDark {
		t.Fatalf("window = %q, want dark", s.ws.Config.Basic.Theme)
	}
	s.toggleTheme()
	if s.config.Basic.Theme != config.ThemeLight {
		t.Fatalf("theme = %q, want light again", s.config.Basic.Theme)
	}
	// It is a choice that is kept: the app comes up as it was left.
	got, err := config.Load(s.dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Basic.Theme != config.ThemeLight {
		t.Fatalf("kept = %q, want light", got.Basic.Theme)
	}
}

// TestChosen_theFileTheUserHasTheirEyeOn checks what a key does its work
// on: what the tree shows as chosen, and otherwise the note that is open.
func TestChosen_theFileTheUserHasTheirEyeOn(t *testing.T) {
	s := appWithWindow(t)
	path := filepath.Join(s.root, "a.md")
	if err := os.WriteFile(path, []byte("# a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := s.chosen(); got != "" {
		t.Fatalf("chosen = %q, want nothing", got)
	}
	s.ws.Tree.Selected = path
	if got := s.chosen(); got != path {
		t.Fatalf("chosen = %q, want the one the tree knows", got)
	}
	// Nothing chosen in the tree: it is the note that is open.
	s.ws.Tree.Selected = ""
	if _, err := s.ws.Tabs.Open(path); err != nil {
		t.Fatal(err)
	}
	if got := s.chosen(); got != path {
		t.Fatalf("chosen = %q, want the note that is open", got)
	}
}

// TestRenameChosen_theAskingIsTheTreeS checks that a rename is asked for
// where the file stands, and that nothing is asked when there is nothing
// chosen.
func TestRenameChosen_theAskingIsTheTrees(t *testing.T) {
	s := appWithWindow(t)
	path := filepath.Join(s.root, "a.md")
	if err := os.WriteFile(path, []byte("# a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s.renameChosen()
	if s.ws.Tree.Renaming == path {
		t.Fatal("a rename was asked for with nothing chosen")
	}
	s.ws.Tree.Selected = path
	s.renameChosen()
	if s.ws.Tree.Renaming != path {
		t.Fatalf("renaming = %q, want the file chosen",
			s.ws.Tree.Renaming)
	}
	if s.ws.Tree.Name != "a.md" {
		t.Fatalf("name = %q, want the name it stands under",
			s.ws.Tree.Name)
	}
}
