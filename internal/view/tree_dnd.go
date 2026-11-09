// This file is the tree's own drag and drop: a file or folder dragged
// from one row onto another to give it a new home in the vault.

// Package view holds the windows' interfaces, drawn by MyGo in Go.
//
// A view is a method on the state it shows: MyGo calls it on the main
// thread whenever the window needs a frame, and it is the only place that
// changes the state. Nothing here reaches outside the package, so a view
// runs in a test without a window.
package view

import (
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/vault"
)

// dragged is what a row carries when it is dragged. It is its own type so
// that a value dragged elsewhere in the window is not taken for a file of
// the tree, and a file of the tree is not dropped where it does not belong.
type dragged struct {
	path string
}

// dirOfEntry is the folder a row stands for as a place to drop a file:
// the folder itself, or the folder holding a file.
func dirOfEntry(e *vault.Entry) string {
	if e.IsDir {
		return e.Path
	}
	return filepath.Dir(e.Path)
}

// canMoveTo reports whether the file at src may be moved into dir. A move
// that changes nothing, one that would put a folder inside itself, and the
// vault's own folder are all refused, so a drag landing in the wrong place
// leaves the disk as it was.
func (t *Tree) canMoveTo(src, dir string) bool {
	if src == "" || dir == "" || t.Root == nil {
		return false
	}
	if src == t.Root.Path {
		return false // the vault itself is not a file to be moved
	}
	if dir == filepath.Dir(src) {
		return false // it already stands in this folder
	}
	rel, err := filepath.Rel(src, dir)
	if err == nil && !outside(rel) {
		return false // dir is src or a folder inside it
	}
	return true
}

// outside reports whether a path relative to a folder leaves it: an
// absolute path, or one beginning with "..". A relative path that stays
// within, such as "child" or ".", means the folder is src or below it.
func outside(rel string) bool {
	return filepath.IsAbs(rel) || rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// moveDropped asks the app to move the file at src into dir, when the move
// is one the tree allows. dir comes from the row dropped on, or from the
// vault's own ground.
func (t *Tree) moveDropped(src, dir string) {
	if !t.canMoveTo(src, dir) {
		return
	}
	if t.OnMoveTo != nil {
		t.OnMoveTo(src, dir)
	}
}

// takeInternalDrop is a file of the vault dropped on the tree's own ground,
// away from any row: it is moved into the vault's own folder. A drop on a
// row is taken by the row itself, which stands nearer the pointer, so this
// one catches only what lands where there is no row.
func (t *Tree) takeInternalDrop(e ui.Element) {
	if t.OnMoveTo == nil || t.Root == nil {
		return
	}
	if d, ok := ui.Drop[dragged](e); ok {
		t.moveDropped(d.path, t.Root.Path)
	}
}
