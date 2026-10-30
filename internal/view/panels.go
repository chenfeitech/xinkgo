package view

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// backlinks are the places in the vault that named the note that is
// open: which note, and the line it was named on.
func (w *Workspace) backlinks(c *ui.Context) {
	n := w.Tabs.ActiveNote()
	if n == nil || w.Index == nil {
		return
	}
	refs := w.Index.Backlinks(n.Path)
	if len(refs) == 0 {
		ui.Text(c, "no other note names this one").TextColor(
			c.Theme().TextMuted)
		return
	}
	// The line each was named on, as the note holds it: it is the line
	// that tells the reader why the link is there.
	text := n.Editor.Buffer.String()
	for _, ref := range refs {
		ref := ref
		row := ui.Button(c.Key("backlink-"+ref.From+"#"+strconv.Itoa(
			ref.Line)), "")
		row.Children(func() {
			ui.Column(c).Children(func() {
				ui.Text(c, PathOf(ref.From)).TextColor(
					c.Theme().TextMuted)
				ui.Text(c, lineOf(text, ref.Line))
			})
		})
		if row.Clicked() {
			w.openNote(ref.From)
			if opened := w.Tabs.ActiveNote(); opened != nil &&
				opened.Path == ref.From {
				opened.GoToLine(ref.Line)
			}
		}
	}
}

// tags are the tags of the whole vault and how many notes carry each.
// Choosing one opens it: the notes that carry it, and choosing one of
// those opens it.
func (w *Workspace) tags(c *ui.Context) {
	if w.Index == nil {
		return
	}
	if w.tagOpen != "" {
		w.tagNotes(c)
		return
	}
	tags := w.Index.Tags()
	if len(tags) == 0 {
		ui.Text(c, "nothing is tagged in this vault").TextColor(
			c.Theme().TextMuted)
		return
	}
	for _, tag := range tags {
		tag := tag
		if ui.Button(c.Key("tag-"+tag.Name), tag.Name+"  "+
			strconv.Itoa(tag.Count)).Clicked() {
			w.tagOpen = tag.Name
		}
	}
}

// tagNotes are the notes carrying the tag that was opened, and a way
// back to all of them.
func (w *Workspace) tagNotes(c *ui.Context) {
	if ui.Button(c.Key("tag-back"), "← "+w.tagOpen).Clicked() {
		w.tagOpen = ""
		return
	}
	paths := w.Index.NotesWithTag(w.tagOpen)
	if len(paths) == 0 {
		ui.Text(c, "no note carries it").TextColor(c.Theme().TextMuted)
		return
	}
	for _, path := range paths {
		path := path
		ui.Row(c).Children(func() {
			ui.Text(c, w.Index.Title(path)).Grow(1)
			if ui.Button(c.Key("tag-note-"+path), "打开").Clicked() {
				w.openNote(path)
			}
		})
	}
}

// lineOf is a line of a text, as the drawer shows it: short enough to
// read beside a name, and the whole of it when it is.
func lineOf(text string, line int) string {
	lines := strings.Split(text, "\n")
	if line < 0 || line >= len(lines) {
		return ""
	}
	out := strings.TrimSpace(lines[line])
	if len([]rune(out)) > 60 {
		return string([]rune(out)[:60]) + "…"
	}
	return out
}
