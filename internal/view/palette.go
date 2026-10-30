package view

import (
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/fuzzy"
)

// How many notes the palette shows, and how wide it is.
const (
	paletteLimit = 50
	paletteWidth = 560
)

// OpenPalette is the palette opened: what was typed last time is
// forgotten, and the first answer is the one the keys are on.
func (w *Workspace) OpenPalette() {
	w.paletteOpen = true
	w.paletteQuery = ""
	w.paletteChosen = 0
	// The keys go to the field as the palette opens, and not every
	// frame: the user must be able to click an answer.
	w.paletteFocus = true
}

// palette is the command palette: a few letters of a note's name, and
// the note that was meant. It is drawn over the window, and goes away
// when the user looks away from it.
func (w *Workspace) palette(c *ui.Context) {
	if !w.paletteOpen {
		return
	}
	ui.Modal(c, &w.paletteOpen, func() {
		ui.Column(c).Width(paletteWidth).Children(func() {
			field := ui.TextInput(c.Key("palette"), &w.paletteQuery).Grow(1)
			if w.paletteFocus {
				field.Focus()
				w.paletteFocus = false
			}
			if field.Submitted() {
				w.choose(w.paletteChosen)
				return
			}
			if field.Shortcut(0, ui.KeyEscape) {
				w.paletteOpen = false
				return
			}
			if field.Shortcut(0, ui.KeyDown) {
				w.paletteChosen++
			}
			if field.Shortcut(0, ui.KeyUp) && w.paletteChosen > 0 {
				w.paletteChosen--
			}
			items := w.paletteItems()
			if len(items) == 0 {
				ui.Text(c, "no note of that name").TextColor(
					c.Theme().TextMuted)
				return
			}
			if w.paletteChosen >= len(items) {
				w.paletteChosen = len(items) - 1
			}
			w.paletteList(c, items)
		})
	})
}

// paletteList is the notes that were found, the one the keys are on
// marked, and the letters that were typed lit up in each.
func (w *Workspace) paletteList(c *ui.Context, items []paletteItem) {
	ui.Column(c).ClipY().MaxHeight(320).Children(func() {
		for i, item := range items {
			i, item := i, item
			row := ui.Button(c.Key(item.Path), "")
			row.Children(func() {
				ui.Column(c).Grow(1).Children(func() {
					ui.Row(c).Children(func() {
						for _, part := range fuzzy.Segments(
							item.Name, item.Positions) {
							text := ui.Text(c, part.Text)
							if part.Found {
								text.Bold()
							} else {
								text.TextColor(c.Theme().TextMuted)
							}
						}
					})
					if item.Title != "" && item.Title != item.Name {
						ui.Text(c, item.Title).TextColor(
							c.Theme().TextMuted)
					}
				})
			})
			if i == w.paletteChosen {
				row.Selected(true)
			}
			if row.Clicked() {
				w.choose(i)
			}
		}
	})
}

// choose opens the note the palette is on.
func (w *Workspace) choose(i int) {
	items := w.paletteItems()
	if i < 0 || i >= len(items) {
		return
	}
	w.paletteOpen = false
	w.openNote(items[i].Path)
}

// openNote is the one way a note is opened from the window: its tab, and
// its place in the tree.
func (w *Workspace) openNote(path string) {
	if _, err := w.Tabs.Open(path); err != nil {
		w.status = "could not open " + PathOf(path)
		return
	}
	w.Tree.Reveal(path)
}

// paletteItem is one answer the palette gives: the note, the name it was
// found by, and where the letters were found in it.
type paletteItem struct {
	Path      string
	Name      string
	Title     string
	Positions []int
}

// paletteItems is what the palette asks of the index: every note of the
// vault, written as it stands from the root, and the best answers first.
func (w *Workspace) paletteItems() []paletteItem {
	if w.Index == nil {
		return nil
	}
	root := w.Index.Root()
	paths := w.Index.Paths()
	names := make([]string, 0, len(paths))
	for _, path := range paths {
		name := path
		if root != "" {
			name = strings.TrimPrefix(strings.TrimPrefix(path, root),
				string(filepath.Separator))
		}
		names = append(names, name)
	}
	found := fuzzy.Search(names, w.paletteQuery, paletteLimit)
	items := make([]paletteItem, 0, len(found))
	for _, r := range found {
		items = append(items, paletteItem{
			Path:      filepath.Join(root, r.Text),
			Name:      r.Text,
			Title:     w.Index.Title(filepath.Join(root, r.Text)),
			Positions: r.Positions,
		})
	}
	return items
}
