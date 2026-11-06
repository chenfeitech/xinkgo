package view

import (
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/fuzzy"
	"github.com/helightxu/xinkgo/internal/locale"
)

// pickerLimit is how many choices the template picker shows. A template
// folder is small; this keeps the modal from covering the window.
const pickerLimit = 20

// Picker is a small chooser the app opens for a command. The view draws
// it and reports which item was chosen; it never reads the disk itself.
type Picker struct {
	Open   bool
	Title  string
	Items  []string
	OnPick func(name string)

	query  string
	chosen int
	focus  bool
}

// OpenPicker shows one chooser. Opening another replaces the first, so at
// most one modal is ever on show, as with the plugin view drawer.
func (w *Workspace) OpenPicker(title string, items []string,
	onPick func(string)) {
	w.pickerState = Picker{
		Open: true, Title: title, Items: items, OnPick: onPick,
		focus: true,
	}
}

// picker is the template chooser. It is drawn over the window and goes
// away when the user looks away from it, as the command palette does.
func (w *Workspace) picker(c *ui.Context) {
	p := &w.pickerState
	if !p.Open {
		return
	}
	ui.Modal(c, &p.Open, func() {
		ui.Column(c).Width(paletteWidth).Children(func() {
			ui.Text(c, p.Title).Bold()
			field := ui.TextInput(c.Key("picker"), &p.query).Grow(1)
			if p.focus {
				field.Focus()
				p.focus = false
			}
			if field.Submitted() {
				w.choosePicker(p.chosen)
				return
			}
			if field.Shortcut(0, ui.KeyEscape) {
				p.Open = false
				return
			}
			if field.Shortcut(0, ui.KeyDown) {
				p.chosen++
			}
			if field.Shortcut(0, ui.KeyUp) && p.chosen > 0 {
				p.chosen--
			}
			found := fuzzy.Search(p.Items, p.query, pickerLimit)
			if len(found) == 0 {
				ui.Text(c, locale.PaletteNothing()).
					TextColor(c.Theme().TextMuted)
				return
			}
			if p.chosen >= len(found) {
				p.chosen = len(found) - 1
			}
			ui.Column(c).ClipY().MaxHeight(320).Children(func() {
				for i, one := range found {
					i := i
					row := ui.Button(c.Key("picker-"+one.Text), "")
					row.Children(func() {
						ui.Text(c, one.Text)
					})
					if i == p.chosen {
						row.Selected(true)
					}
					if row.Clicked() {
						w.choosePicker(i)
					}
				}
			})
		})
	})
}

// choosePicker closes the modal and reports the chosen name to the app.
func (w *Workspace) choosePicker(i int) {
	p := &w.pickerState
	found := fuzzy.Search(p.Items, p.query, pickerLimit)
	if i < 0 || i >= len(found) {
		return
	}
	name := found[i].Text
	onPick := p.OnPick
	p.Open = false
	if onPick != nil {
		onPick(name)
	}
}
