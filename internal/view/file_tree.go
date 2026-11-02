// Package view holds the windows' interfaces, drawn by MyGo in Go.
//
// A view is a method on the state it shows: MyGo calls it on the main
// thread whenever the window needs a frame, and it is the only place that
// changes the state. Nothing here reaches outside the package, so a view
// runs in a test without a window.
package view

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
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
	// OnRefresh reads the vault's folder again, when the user asks.
	OnRefresh func()
	// Vaults are the folders the user can open, and which one is open.
	// OnSwitchVault is asked for another, by its identifier, and
	// OnManageVaults for the window that keeps them.
	Vaults         vault.Registry
	OnSwitchVault  func(id string)
	OnManageVaults func()
	OnOpenFolder   func()
	// OnOpenSettings opens the settings window, asked for at the foot
	// of the tree, which stands there whether a vault is open or not.
	OnOpenSettings func()

	// FontSize is how large the names are drawn, as the user chose; 0
	// takes the size the window's theme has.
	FontSize float32

	// Renaming is the path whose name is being asked for, and Name the
	// name the user has typed so far.
	Renaming string
	Name     string

	refreshed time.Time
	list      ui.ListState
}

// How the tree is drawn: the height of a line, how far a line starts for
// every level it is in, and the sizes of the chevron, the icon and the
// header's buttons.
const (
	treeRowHeight float32 = 26
	treeIndent    float32 = 14
	treePad       float32 = 6
	treeGap       float32 = 6
	treeChevron   float32 = 12
	treeIcon      float32 = 14
	treeButton    float32 = 22
)

// RefreshQuiet is how long a second press of the refresh button is
// ignored, so that pressing it twice reads the folder once.
const RefreshQuiet = 300 * time.Millisecond

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

// CollapseAll closes every directory, so that only the vault's own
// children are left.
func (t *Tree) CollapseAll() {
	t.Open = map[string]bool{}
}

// ExpandAll opens every directory the vault holds.
func (t *Tree) ExpandAll() {
	if t.Root == nil {
		return
	}
	vault.Walk(t.Root, func(e *vault.Entry) error {
		if e.IsDir {
			t.Open[e.Path] = true
		}
		return nil
	})
}

// AnyOpen is whether a directory stands open, which the header's button
// asks before it decides whether it folds them or opens them.
func (t *Tree) AnyOpen() bool {
	for _, open := range t.Open {
		if open {
			return true
		}
	}
	return false
}

// count is how many notes the vault holds, and dirs how many folders,
// for the header.
func (t *Tree) count() int {
	if t.Root == nil {
		return 0
	}
	return len(vault.Files(t.Root))
}

// dirs is how many folders the vault holds, its own not counted.
func (t *Tree) dirs() int {
	if t.Root == nil {
		return 0
	}
	n := 0
	vault.Walk(t.Root, func(e *vault.Entry) error {
		if e.IsDir && e != t.Root {
			n++
		}
		return nil
	})
	return n
}

// counts is what the header says the vault holds.
func (t *Tree) counts() string {
	return fmt.Sprintf("%s · %s", plural(t.count(), "note"),
		plural(t.dirs(), "folder"))
}

// plural is a number and the word for what it counts, with an s when it
// is not one of them.
func plural(n int, word string) string {
	if n == 1 {
		return strconv.Itoa(n) + " " + word
	}
	return strconv.Itoa(n) + " " + word + "s"
}

// View draws the tree: the vault at the head, and under it one line for
// each file and folder, a folder with a chevron that turns as it opens,
// a file with the icon of what it is, and the one that is open marked.
func (t *Tree) View(c *ui.Context) {
	rows := t.Rows()
	tree := ui.Column(c).Fill()
	defer func() { t.takeDrop(tree) }()
	tree.Children(func() {
		t.header(c)
		ui.List(c, &t.list, len(rows), func(i int) {
			if i >= len(rows) {
				return
			}
			t.line(c, rows[i])
		}).Fill()
		t.footer(c)
	})
}

// footer is the foot of the tree: the way to the settings, which stands
// below the files whatever is in the vault, and is drawn on its own when
// there is no vault at all.
func (t *Tree) footer(c *ui.Context) {
	if t.OnOpenSettings == nil {
		return
	}
	th := c.Theme()
	ui.Box(c).FillWidth().Height(1).Background(th.Border)
	line := ui.ButtonBase(c.Key("tree-settings")).Label("Settings").
		Tooltip("Settings").FillWidth().Height(treeRowHeight).
		Radius(th.Radius).Justify(ui.Start).AlignItems(ui.Center)
	line.Draw(func(p *ui.Painter, box ui.Rect) {
		if line.Hovered() {
			p.Fill(box, th.SurfaceHover, th.Radius)
		}
	})
	line.Children(func() {
		ui.Row(c).FillWidth().Height(treeRowHeight).Gap(treeGap).
			AlignItems(ui.Center).
			PaddingStart(treePad).PaddingEnd(treePad).Children(func() {
			ui.Icon(c, iconSettings).Size(treeIcon, treeIcon).
				TextColor(th.TextMuted)
			ui.Text(c, "Settings").Grow(1).Ellipsis("…").NoWrap()
		})
	})
	if line.Clicked() {
		t.OnOpenSettings()
	}
}

// header is the head of the tree: which vault it is, and the way to
// another; what it holds, and the buttons that make things in it, fold it
// and read it again.
func (t *Tree) header(c *ui.Context) {
	th := c.Theme()
	name := t.vaultName()
	ui.Column(c).FillWidth().Padding(8, treePad, 6).Gap(4).Children(func() {
		ui.Row(c).FillWidth().Gap(treeGap).AlignItems(ui.Center).
			Children(func() {
				t.vaultSwitch(c, name)
				if t.OnRefresh != nil {
					t.button(c, "refresh", iconRefresh,
						"Read the folder again", t.refresh)
				}
			})
		ui.Row(c).FillWidth().Gap(4).AlignItems(ui.Center).
			Children(func() {
				ui.Text(c, t.counts()).FontSize(th.FontSize - 1).
					TextColor(th.TextMuted).Grow(1).Ellipsis("…").NoWrap()
				if t.Root == nil {
					return
				}
				if t.OnNewNoteIn != nil {
					t.button(c, "new-note", iconFilePlus, "New note here",
						func() { t.OnNewNoteIn(t.DirOf()) })
				}
				if t.OnNewFolderIn != nil {
					t.button(c, "new-folder", iconFolderPlus,
						"New folder here",
						func() { t.OnNewFolderIn(t.DirOf()) })
				}
				if t.AnyOpen() {
					t.button(c, "fold", iconFoldAll,
						"Fold every folder", t.CollapseAll)
				} else {
					t.button(c, "fold", iconUnfoldAll,
						"Open every folder", t.ExpandAll)
				}
			})
	})
	ui.Box(c).FillWidth().Height(1).Background(th.Border)
}

// vaultSwitch is the head of the tree: the vault that is open, and the
// way to another. The folders the user can open stand in its menu, the
// one that is open marked, and a folder the app cannot open says so
// rather than failing when it is chosen.
func (t *Tree) vaultSwitch(c *ui.Context, name string) {
	th := c.Theme()
	label := "Switch vault"
	head := ui.ButtonBase(c.Key("vault-switch")).Label(label).
		Tooltip(label).Grow(1).MinWidth(0).Radius(th.Radius)
	head.Menu(func(m *ui.Menu) { t.vaultMenu(m) })
	head.Children(func() {
		ui.Row(c).FillWidth().Gap(treeGap).AlignItems(ui.Center).
			Children(func() {
				ui.Icon(c, iconFolder).Size(treeIcon, treeIcon).
					TextColor(th.TextMuted)
				ui.Text(c, name).Bold().Grow(1).Ellipsis("…").NoWrap()
				ui.Icon(c, iconChevron).Size(treeChevron, treeChevron).
					TextColor(th.TextMuted)
			})
	})
}

// vaultMenu is what the head of the tree asks: which vault is open, and
// what may be done about them.
func (t *Tree) vaultMenu(m *ui.Menu) {
	for _, v := range t.Vaults.Vaults {
		v := v
		label := v.Name
		if s := t.Vaults.StatusOf(v.ID); !s.Usable() {
			label += " (" + s.String() + ")"
		}
		if m.Item(label).Radio(t.vaultID() == v.ID).Chosen() &&
			t.vaultID() != v.ID && t.OnSwitchVault != nil {
			t.OnSwitchVault(v.ID)
		}
	}
	if len(t.Vaults.Vaults) == 0 {
		m.Item("no other vault").Disabled(true)
	}
	m.Separator()
	if m.Item("Open another folder…").Chosen() && t.OnOpenFolder != nil {
		t.OnOpenFolder()
	}
	if m.Item("Manage vaults…").Chosen() && t.OnManageVaults != nil {
		t.OnManageVaults()
	}
}

// vaultName is what the tree calls the vault that is open: the name the
// user gave it, when it is one the app knows, and the folder's own
// otherwise.
// vaultID is the identifier of the vault that is open, or nothing when
// the app knows of none.
func (t *Tree) vaultID() string {
	if t.Vaults.Active != nil {
		return *t.Vaults.Active
	}
	return ""
}

func (t *Tree) vaultName() string {
	for _, v := range t.Vaults.Vaults {
		if v.ID == t.vaultID() && v.Name != "" {
			return v.Name
		}
	}
	if t.Root != nil {
		return t.Root.Name
	}
	return "no vault"
}

// button is one of the header's buttons: it shows no face of its own
// until the pointer is over it, and says what it does when it is.
func (t *Tree) button(
	c *ui.Context, key string, img *ui.SVG, label string, fn func(),
) {
	th := c.Theme()
	b := ui.ButtonBase(c.Key("tree-"+key)).Label(label).Tooltip(label).
		Size(treeButton, treeButton).Radius(th.Radius).
		Justify(ui.Center).AlignItems(ui.Center)
	b.Draw(func(p *ui.Painter, box ui.Rect) {
		switch {
		case b.Pressed():
			p.Fill(box, th.SurfacePressed, th.Radius)
		case b.Hovered():
			p.Fill(box, th.SurfaceHover, th.Radius)
		}
	})
	b.Children(func() {
		ui.Icon(c, img).Size(treeIcon, treeIcon).TextColor(th.TextMuted)
	})
	if b.Clicked() {
		fn()
	}
}

// line is one file or folder: it starts further in for every level it is
// in, shows what it is, and is chosen by a click. The one that is open is
// marked, and the one under the pointer is lit, as in the file explorers
// the reader knows.
func (t *Tree) line(c *ui.Context, r row) {
	e := r.Entry
	th := c.Theme()
	chosen := t.Selected == e.Path
	open := t.Open[e.Path]
	line := ui.ButtonBase(c.Key(e.Path)).Label(e.Name).
		FillWidth().Height(treeRowHeight).Radius(th.Radius).
		ContextMenu(func(m *ui.Menu) { t.menu(m, e) })
	line.Draw(func(p *ui.Painter, box ui.Rect) {
		switch {
		case chosen:
			p.Fill(box, th.Accent.Alpha(0.18), th.Radius)
		case line.Hovered():
			p.Fill(box, th.SurfaceHover, th.Radius)
		}
	})
	line.Children(func() {
		ui.Row(c).FillWidth().Height(treeRowHeight).Gap(treeGap).
			AlignItems(ui.Center).
			PaddingStart(treePad + float32(r.Depth)*treeIndent).
			PaddingEnd(treePad).Children(func() {
			if t.Renaming == e.Path {
				t.renameField(c, e)
				return
			}
			if e.IsDir {
				chevron := ui.Icon(c, iconChevron).
					Size(treeChevron, treeChevron).TextColor(th.TextMuted)
				if open {
					chevron.Rotate(90)
				}
			} else {
				// A file has no chevron, but its icon stands where a
				// folder's does.
				ui.Box(c).Size(treeChevron, treeChevron)
			}
			ui.Icon(c, iconOf(e, open)).Size(treeIcon, treeIcon).
				TextColor(th.TextMuted)
			name := ui.Text(c, e.Name).Grow(1).Ellipsis("…").NoWrap()
			if t.FontSize > 0 {
				name.FontSize(t.FontSize)
			}
		})
	})
	if line.Clicked() {
		t.choose(e)
	}
}

// refresh reads the folder again when the user asks. Pressing the button
// twice quickly reads it once: what the eye wanted was one reading, and
// the second press only says it again.
func (t *Tree) refresh() {
	now := time.Now()
	if now.Sub(t.refreshed) < RefreshQuiet {
		return
	}
	t.refreshed = now
	if t.OnRefresh != nil {
		t.OnRefresh()
	}
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
