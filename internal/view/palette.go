package view

import (
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/fuzzy"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/skill"
)

// How many notes the palette shows, and how wide it is.
const (
	paletteLimit = 50
	paletteWidth = 560
	// skillPaletteLimit is how many skills the palette shows: there
	// are not many of them, and the notes are what it is for.
	skillPaletteLimit = 10
	// actionPaletteLimit is how many actions the palette shows.
	actionPaletteLimit = 10
	// pluginPaletteLimit is how many plugin commands the palette shows.
	pluginPaletteLimit = 10
)

// skillPrefix is what a skill is called in the palette, so that a skill
// is not taken for a note, and actionPrefix what one of the app's own
// actions is called, for the same reason. Both are said in the language
// the app shows, which may change while it stands.
func skillPrefix() string  { return locale.PaletteSkillPrefix() }
func actionPrefix() string { return locale.PaletteActionPrefix() }

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
				ui.Text(c, locale.PaletteNothing()).TextColor(
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
			row := ui.Button(c.Key(item.key()), "")
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
	if items[i].Skill != nil {
		w.useSkill(*items[i].Skill)
		return
	}
	if items[i].Action != nil {
		w.runAction(*items[i].Action)
		return
	}
	if items[i].Plugin != nil {
		if w.OnPluginCommand != nil {
			w.OnPluginCommand(items[i].Plugin.ID)
		}
		return
	}
	w.openNote(items[i].Path)
}

// useSkill is a skill taken up for the note being looked at: what it
// says is told to the model with the next question, and stays told until
// it is put down.
func (w *Workspace) useSkill(s skill.Skill) {
	n := w.Tabs.ActiveNote()
	if n == nil {
		return
	}
	session := w.aiOf(n.Path)
	session.Open = true
	session.Skill = &s
	w.status = locale.StatusUsingSkill(s.Name)
}

// openNote is the one way a note is opened from the window: its tab, and
// its place in the tree.
func (w *Workspace) openNote(path string) {
	if _, err := w.Tabs.Open(path); err != nil {
		w.status = locale.StatusCouldNotOpen(PathOf(path))
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
	// Skill is the skill this answer is, when it is not a note.
	Skill *skill.Skill
	// Action is one of the app's own actions this answer is, when it
	// is not a note either.
	Action *config.AiAction
	// Plugin is a command from an Obsidian-compatible plugin.
	Plugin *PluginCommand
}

// key is what an answer is known by in the window: a note by where it
// stands, and a skill or an action by its name, for two of them that
// stand nowhere must not be taken for each other.
func (p paletteItem) key() string {
	if p.Path != "" {
		return p.Path
	}
	if p.Plugin != nil {
		return p.Plugin.ID
	}
	return p.Name
}

// paletteItems is what the palette answers with: the notes of the vault,
// and with them the skills a model may be told to use and the app's own
// actions over a note.
func (w *Workspace) paletteItems() []paletteItem {
	items := w.noteItems()
	items = append(items, w.skillItems()...)
	items = append(items, w.actionItems()...)
	return append(items, w.pluginItems()...)
}

// pluginItems is the commands of the enabled plugins that the search
// finds. Visibility comes from the app's cached checkCallback result;
// a plugin is never called from a frame just to decide whether to draw.
func (w *Workspace) pluginItems() []paletteItem {
	names := make([]string, 0, len(w.PluginCommands))
	for _, one := range w.PluginCommands {
		if one.Visible {
			names = append(names, one.Name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	found := fuzzy.Search(names, w.paletteQuery, pluginPaletteLimit)
	items := make([]paletteItem, 0, len(found))
	for _, r := range found {
		one := pluginCommandNamed(w.PluginCommands, r.Text)
		if one == nil {
			continue
		}
		items = append(items, paletteItem{
			Name:      r.Text,
			Title:     one.ID,
			Positions: r.Positions,
			Plugin:    one,
		})
	}
	return items
}

// pluginCommandNamed is the visible command of that name.
func pluginCommandNamed(list []PluginCommand, name string) *PluginCommand {
	for i := range list {
		if list[i].Visible && list[i].Name == name {
			return &list[i]
		}
	}
	return nil
}

// actionItems is the app's own actions, found by their names, each shown
// with what it asks of the model.
func (w *Workspace) actionItems() []paletteItem {
	if len(w.Actions) == 0 {
		return nil
	}
	names := make([]string, 0, len(w.Actions))
	for _, one := range w.Actions {
		names = append(names, one.Name)
	}
	found := fuzzy.Search(names, w.paletteQuery, actionPaletteLimit)
	items := make([]paletteItem, 0, len(found))
	for _, r := range found {
		one := actionNamed(w.Actions, r.Text)
		if one == nil {
			continue
		}
		items = append(items, paletteItem{
			Name:      actionPrefix() + r.Text,
			Title:     one.Prompt,
			Positions: shifted(r.Positions, len(actionPrefix())),
			Action:    one,
		})
	}
	return items
}

// actionNamed is the action of that name, when there is one.
func actionNamed(list []config.AiAction, name string) *config.AiAction {
	for i := range list {
		if list[i].Name == name {
			return &list[i]
		}
	}
	return nil
}

// skillItems is the skills, found by their names, each shown as what it
// is for.
func (w *Workspace) skillItems() []paletteItem {
	if len(w.Skills) == 0 {
		return nil
	}
	names := make([]string, 0, len(w.Skills))
	for _, one := range w.Skills {
		names = append(names, one.Name)
	}
	found := fuzzy.Search(names, w.paletteQuery, skillPaletteLimit)
	items := make([]paletteItem, 0, len(found))
	for _, r := range found {
		one := skill.Find(w.Skills, r.Text)
		if one == nil {
			continue
		}
		items = append(items, paletteItem{
			// The prefix is part of what is shown, so the letters
			// that were typed are lit up where they stand.
			Name:      skillPrefix() + r.Text,
			Title:     one.Description,
			Positions: shifted(r.Positions, len(skillPrefix())),
			Skill:     one,
		})
	}
	return items
}

// shifted is the same places, count further along: the name a skill was
// found by is shown after a prefix.
func shifted(positions []int, count int) []int {
	out := make([]int, 0, len(positions))
	for _, at := range positions {
		out = append(out, at+count)
	}
	return out
}

// noteItems is what the palette asks of the index: every note of the
// vault, written as it stands from the root, and the best answers first.
func (w *Workspace) noteItems() []paletteItem {
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
