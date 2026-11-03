// This file is how a path in the vault is found in the tree, chosen in
// it, and brought into view.

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

	"github.com/helightxu/xinkgo/internal/vault"
)

// DirOf is the folder things are made in: the one holding what is
// chosen, or the vault's own when nothing is.
func (t *Tree) DirOf() string {
	if t.Selected != "" {
		if e := t.find(t.Selected); e != nil && e.IsDir {
			return e.Path
		}
		return filepath.Dir(t.Selected)
	}
	if t.Root != nil {
		return t.Root.Path
	}
	return ""
}

// find is the entry at path, as the tree has it.
func (t *Tree) find(path string) *vault.Entry {
	if t.Root == nil {
		return nil
	}
	var found *vault.Entry
	vault.Walk(t.Root, func(e *vault.Entry) error {
		if e.Path == path {
			found = e
		}
		return nil
	})
	return found
}

// choose is what a click does: a directory opens or closes, a file is
// chosen and opened.
func (t *Tree) choose(e *vault.Entry) {
	if e.IsDir {
		t.Toggle(e.Path)
		return
	}
	t.Selected = e.Path
	if t.OnOpen != nil {
		t.OnOpen(e.Path)
	}
}

// Reveal opens the directories above path and chooses the file, so that
// the note the user is editing is seen where it stands in the vault, and
// asks for it to be brought into view. It says whether the vault holds
// the file: one it does not, as a note of another vault, is not in the
// tree to be shown, and what the tree had chosen is left alone.
func (t *Tree) Reveal(path string) bool {
	if t.Root == nil {
		return false
	}
	var open func(e *vault.Entry, above []string) bool
	open = func(e *vault.Entry, above []string) bool {
		for _, c := range e.Children {
			if c.Path == path {
				for _, a := range above {
					t.Open[a] = true
				}
				t.Selected = path
				t.revealPath = path
				return true
			}
			if c.IsDir {
				if open(c, append(above, c.Path)) {
					return true
				}
			}
		}
		return false
	}
	return open(t.Root, nil)
}

// bringIntoView scrolls the tree to the file it was last asked to reveal.
// The list is told before it is built, for it plans its rows around the
// one it is to show: a file far down a long vault is drawn at once, and
// not left below the fold until the user scrolls to it.
func (t *Tree) bringIntoView(rows []row) {
	if t.revealPath == "" {
		return
	}
	for i, r := range rows {
		if r.Entry.Path == t.revealPath {
			t.list.ScrollIntoView(i)
			break
		}
	}
	t.revealPath = ""
}

// PathOf is the last part of a path, for a name the tree shows.
func PathOf(path string) string {
	if i := strings.LastIndexAny(path, "/\\"); i >= 0 {
		return path[i+1:]
	}
	return path
}
