// This file is the rows of the tree: what one is drawn as, what it does
// when it is taken hold of, and what its menu holds.

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
	"time"

	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/vault"
)

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
	if m.Item(locale.TreeNewNoteHere()).Chosen() {
		t.in(e, t.OnNewNoteIn)
	}
	if m.Item(locale.TreeNewFolderHere()).Chosen() {
		t.in(e, t.OnNewFolderIn)
	}
	m.Separator()
	if m.Item(locale.TreeRename()).Chosen() {
		t.Renaming = e.Path
		t.Name = e.Name
	}
	if m.Item(locale.TreeMoveTo()).Chosen() && t.OnMove != nil {
		t.OnMove(e.Path)
	}
	if m.Item(locale.TreeDelete()).Chosen() && t.OnDelete != nil {
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
