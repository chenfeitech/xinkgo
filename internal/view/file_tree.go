// Package view holds the windows' interfaces, drawn by MyGo in Go.
//
// A view is a method on the state it shows: MyGo calls it on the main
// thread whenever the window needs a frame, and it is the only place that
// changes the state. Nothing here reaches outside the package, so a view
// runs in a test without a window.
package view

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/media"
	"github.com/helightxu/xinkgo/internal/vault"
)

// Tree is the state of the file tree: which directories are open, which
// file is chosen, and what the user is shown.
type Tree struct {
	// Root is the vault's folder, as it was last scanned.
	Root *vault.Entry
	// Open is which directories are expanded, by their path.
	Open map[string]bool
	// Selected is the path of the file the tree shows as chosen.
	Selected string
	// ShowAll is whether files the app cannot open are shown too.
	ShowAll bool
	// OnOpen is called with the path of a file the user chose; the app
	// opens it in a tab.
	OnOpen func(path string)

	list ui.ListState
}

// NewTree returns a tree with nothing in it yet.
func NewTree() *Tree {
	return &Tree{Open: map[string]bool{}, ShowAll: false}
}

// row is one line the tree draws.
type row struct {
	Entry *vault.Entry
	Depth int
}

// Rows is what the tree draws: the root's children, and the children of
// every directory that is open, in the order they are shown.
func (t *Tree) Rows() []row {
	if t.Root == nil {
		return nil
	}
	var rows []row
	var add func(e *vault.Entry, depth int)
	add = func(e *vault.Entry, depth int) {
		for _, c := range e.Children {
			rows = append(rows, row{Entry: c, Depth: depth})
			if c.IsDir && t.Open[c.Path] {
				add(c, depth+1)
			}
		}
	}
	add(t.Root, 0)
	return rows
}

// SetRoot gives the tree a vault to show. Directories that are still
// there keep being open, so that a rescan does not close them.
func (t *Tree) SetRoot(root *vault.Entry) {
	t.Root = root
}

// Toggle opens or closes a directory.
func (t *Tree) Toggle(path string) {
	t.Open[path] = !t.Open[path]
}

// count is how many notes the vault holds, for the header.
func (t *Tree) count() int {
	if t.Root == nil {
		return 0
	}
	return len(vault.Files(t.Root))
}

// View draws the tree.
func (t *Tree) View(c *ui.Context) {
	rows := t.Rows()
	ui.Column(c).Fill().Children(func() {
		// The header: what is open, and how much is in it.
		ui.Row(c).Gap(8).Padding(8).Children(func() {
			name := "no vault"
			if t.Root != nil {
				name = t.Root.Name
			}
			ui.Text(c, name).Bold()
			ui.Textf(c, "%d notes", t.count()).TextColor(c.Theme().TextMuted)
		})

		ui.List(c, &t.list, len(rows), func(i int) {
			if i >= len(rows) {
				return
			}
			r := rows[i]
			ui.Row(c).Children(func() {
				// The deeper a file is, the further it starts.
				ui.Box(c).Width(float32(8 + r.Depth*12))
				ui.Text(c, marker(r.Entry)).TextColor(c.Theme().TextMuted)
				label := r.Entry.Name
				if r.Entry.IsDir {
					label = r.Entry.Name
				}
				if ui.Button(c.Key(r.Entry.Path), label).Clicked() {
					t.choose(r.Entry)
				}
			})
		}).Fill()
	})
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

// marker is what is drawn before a name: a directory is open or closed, a
// file says what it is.
func marker(e *vault.Entry) string {
	if e.IsDir {
		return "▸"
	}
	switch media.KindOf(e.Path) {
	case media.KindMarkdown:
		return "·"
	case media.KindImage:
		return "▣"
	case media.KindData:
		return "{}"
	case media.KindCode:
		return "</>"
	default:
		return "·"
	}
}

// Reveal opens the directories above path, so that the file the user is
// editing is visible in the tree.
func (t *Tree) Reveal(path string) {
	if t.Root == nil {
		return
	}
	var open func(e *vault.Entry, above []string) bool
	open = func(e *vault.Entry, above []string) bool {
		for _, c := range e.Children {
			if c.Path == path {
				for _, a := range above {
					t.Open[a] = true
				}
				t.Selected = path
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
	open(t.Root, nil)
}

// PathOf is the last part of a path, for a name the tree shows.
func PathOf(path string) string {
	if i := strings.LastIndexAny(path, "/\\"); i >= 0 {
		return path[i+1:]
	}
	return path
}
