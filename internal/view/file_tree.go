// Package view holds the windows' interfaces, drawn by MyGo in Go.
//
// A view is a method on the state it shows: MyGo calls it on the main
// thread whenever the window needs a frame, and it is the only place that
// changes the state. Nothing here reaches outside the package, so a view
// runs in a test without a window.
package view

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/locale"
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
	// OnMoveTo asks the app to move the file at path into dir at once,
	// as when it is dragged onto a folder in the tree.
	OnMoveTo func(path, dir string)
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
	// focusRename is whether the rename field is to take the keyboard
	// focus as it appears, so a new name can be typed at once.
	focusRename bool

	// revealPath is the file the tree is to bring into view when next it
	// draws. What was asked for while the tree was not being drawn at
	// all, as when its panel stands shut, is not forgotten: it is shown
	// when the tree comes back.
	revealPath string

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
	return locale.TreeCounts(locale.NoteCount(t.count()),
		locale.FolderCount(t.dirs()))
}

// View draws the tree: the vault at the head, and under it one line for
// each file and folder, a folder with a chevron that turns as it opens,
// a file with the icon of what it is, and the one that is open marked.
func (t *Tree) View(c *ui.Context) {
	rows := t.Rows()
	t.bringIntoView(rows)
	tree := ui.Column(c).Fill()
	defer func() {
		t.takeDrop(tree)         // files dragged in from another app
		t.takeInternalDrop(tree) // a file of the vault, onto its own ground
	}()
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
	line := ui.ButtonBase(c.Key("tree-settings")).
		Label(locale.TreeSettings()).
		Tooltip(locale.TreeSettings()).
		FillWidth().Height(treeRowHeight).
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
			ui.Text(c, locale.TreeSettings()).Grow(1).Ellipsis("…").
				NoWrap()
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
						locale.TreeRefresh(), t.refresh)
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
					t.button(c, "new-note", iconFilePlus,
						locale.TreeNewNoteHere(),
						func() { t.OnNewNoteIn(t.DirOf()) })
				}
				if t.OnNewFolderIn != nil {
					t.button(c, "new-folder", iconFolderPlus,
						locale.TreeNewFolderHere(),
						func() { t.OnNewFolderIn(t.DirOf()) })
				}
				if t.AnyOpen() {
					t.button(c, "fold", iconFoldAll,
						locale.TreeFoldAll(), t.CollapseAll)
				} else {
					t.button(c, "fold", iconUnfoldAll,
						locale.TreeUnfoldAll(), t.ExpandAll)
				}
			})
	})
	ui.Box(c).FillWidth().Height(1).Background(th.Border)
}
