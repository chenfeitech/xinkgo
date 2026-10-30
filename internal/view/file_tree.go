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

	"github.com/egoist/mygo/transfer"
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
	// OnNewNoteIn makes a note in dir, OnNewFolderIn a folder.
	OnNewNoteIn   func(dir string)
	OnNewFolderIn func(dir string)
	// OnRename is the name the user gave the file at path.
	OnRename func(path, name string)
	// OnDelete asks the app to take the file at path away.
	OnDelete func(path string)
	// OnMove asks the app to move the file at path to another folder.
	OnMove func(path string)
	// OnImport brings files from outside the vault into dir.
	OnImport func(dir string, paths []string)

	// Renaming is the path whose name is being asked for, and Name the
	// name the user has typed so far.
	Renaming string
	Name     string

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
	tree := ui.Column(c).Fill()
	defer func() { t.takeDrop(tree) }()
	tree.Children(func() {
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
				if t.Renaming == r.Entry.Path {
					t.renameField(c, r.Entry)
					return
				}
				ui.Text(c, marker(r.Entry)).TextColor(c.Theme().TextMuted)
				name := ui.Button(c.Key(r.Entry.Path), r.Entry.Name)
				name.ContextMenu(func(m *ui.Menu) { t.menu(m, r.Entry) })
				if name.Clicked() {
					t.choose(r.Entry)
				}
			})
		}).Fill()
	})
}

// menu is what a click of the other button asks: what can be done to
// this file or folder.
func (t *Tree) menu(m *ui.Menu, e *vault.Entry) {
	if m.Item("New note here").Chosen() {
		t.in(e, t.OnNewNoteIn)
	}
	if m.Item("New folder here").Chosen() {
		t.in(e, t.OnNewFolderIn)
	}
	m.Separator()
	if m.Item("Rename…").Chosen() {
		t.Renaming = e.Path
		t.Name = e.Name
	}
	if m.Item("Move to…").Chosen() && t.OnMove != nil {
		t.OnMove(e.Path)
	}
	if m.Item("Delete").Chosen() && t.OnDelete != nil {
		t.OnDelete(e.Path)
	}
}

// in runs fn on the folder the entry stands for: itself when it is one,
// the one holding it when it is a file.
func (t *Tree) in(e *vault.Entry, fn func(dir string)) {
	if fn == nil {
		return
	}
	if e.IsDir {
		fn(e.Path)
		return
	}
	fn(filepath.Dir(e.Path))
}

// renameField is the name of a file being asked for: Enter gives it,
// Escape leaves the file as it was.
func (t *Tree) renameField(c *ui.Context, e *vault.Entry) {
	field := ui.TextInput(c.Key(e.Path+"#name"), &t.Name).Grow(1)
	if field.Submitted() {
		t.commitRename(e.Path)
		return
	}
	if field.Shortcut(0, ui.KeyEscape) {
		t.Renaming = ""
	}
}

// commitRename is the name the user gave: an empty one, or the name the
// file already has, leaves the file alone.
func (t *Tree) commitRename(path string) {
	name := strings.TrimSpace(t.Name)
	t.Renaming = ""
	if name == "" {
		return
	}
	if name == PathOf(path) {
		return
	}
	if t.OnRename != nil {
		t.OnRename(path, name)
	}
}

// takeDrop is a file dragged into the tree from another app: it is
// brought into the vault, into the folder it was dropped on.
func (t *Tree) takeDrop(e ui.Element) {
	if t.OnImport == nil {
		return
	}
	drop, ok := ui.DropData(e, transfer.DropOptions{
		Formats:    []transfer.Format{transfer.URIList, transfer.FileList},
		Operations: transfer.Copy,
	})
	if !ok {
		return
	}
	paths, err := drop.Data.Files()
	if err != nil || len(paths) == 0 {
		return
	}
	t.OnImport(t.DirOf(), paths)
}

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
