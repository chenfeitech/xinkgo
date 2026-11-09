package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/index"
	"github.com/helightxu/xinkgo/internal/view"
)

// TestTreeOnMoveTo_theDragMovesTheFileOnDisk checks the end of a drag: the
// callback the tree is wired to (as boot wires it) gives the file a new
// path, follows it in the index, and shows it where it now stands.
func TestTreeOnMoveTo_theDragMovesTheFileOnDisk(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(root, "a.md")
	if err := os.WriteFile(src, []byte("# a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := &appState{
		dir:    t.TempDir(),
		root:   root,
		ws:     view.NewWorkspace(config.Default()),
		idx:    index.New(),
		config: config.Default(),
	}
	s.ws.Tree.OnMoveTo = s.moveItemTo

	target := filepath.Join(sub, "a.md")
	s.ws.Tree.OnMoveTo(src, sub)

	if _, err := os.Stat(src); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the old path still stands: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "# a\n" {
		t.Fatalf("the file at the new path is %q (%v)", data, err)
	}
	if s.idx.Note(src) != nil || s.idx.Note(target) == nil {
		t.Error("the index did not follow the file to its new path")
	}
	if s.ws.Tree.Selected != target {
		t.Errorf("the tree chose %q, want the moved file %q",
			s.ws.Tree.Selected, target)
	}
}

// TestTreeOnMoveTo_anOpenNoteFollowsItsFile checks that a note open in a tab
// keeps its place when its file is dragged into another folder.
func TestTreeOnMoveTo_anOpenNoteFollowsItsFile(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(root, "a.md")
	if err := os.WriteFile(src, []byte("# a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := &appState{
		dir:    t.TempDir(),
		root:   root,
		ws:     view.NewWorkspace(config.Default()),
		idx:    index.New(),
		config: config.Default(),
	}
	if _, err := s.ws.Tabs.Open(src); err != nil {
		t.Fatal(err)
	}
	s.ws.Tree.OnMoveTo = s.moveItemTo

	target := filepath.Join(sub, "a.md")
	s.ws.Tree.OnMoveTo(src, sub)

	if !s.ws.Tabs.IsActive(target) {
		t.Errorf("the open note is not at %q, want %q",
			s.ws.Tabs.Active, target)
	}
	paths := s.ws.Tabs.Paths()
	if len(paths) != 1 || paths[0] != target {
		t.Errorf("the open notes are %q, want only %q", paths, target)
	}
}
