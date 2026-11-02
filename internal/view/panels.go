package view

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/index"
	"github.com/helightxu/xinkgo/internal/locale"
)

// backlinks are the places in the vault that named the note that is
// open: which note, and where in it, written as Markdown, a list of
// links, as the note beside it is read.
func (w *Workspace) backlinks(c *ui.Context) {
	n := w.Tabs.ActiveNote()
	if n == nil || w.Index == nil {
		return
	}
	refs := w.Index.Backlinks(n.Path)
	if len(refs) == 0 {
		ui.Text(c, locale.PanelsNoBacklinks()).TextColor(
			c.Theme().TextMuted)
		return
	}
	var b strings.Builder
	for _, ref := range refs {
		b.WriteString("- [")
		b.WriteString(mdPlain(PathOf(ref.From)))
		b.WriteString("](")
		b.WriteString(drawerLineHref(ref.From, ref.Line))
		b.WriteString(")")
		// Where in the note it was named: the heading the line stands
		// under, which is what tells the reader why the link is there.
		// It is what the vault knows of the note, and not a line of its
		// text: the note that named this one is not the one open, and a
		// panel is not the place to read one in.
		if head := headingAbove(w.Index, ref.From, ref.Line); head != "" {
			b.WriteString(" · ")
			b.WriteString(mdPlain(head))
		}
		b.WriteString("\n")
	}
	w.drawerMarkdown(c, b.String())
}

// headingAbove is the heading a line of a note stands under, as the vault
// knows it, and empty when there is none above it.
func headingAbove(x *index.Index, path string, line int) string {
	note := x.Note(path)
	if note == nil {
		return ""
	}
	head := ""
	for _, h := range note.Facts.Headings {
		if h.Line <= line {
			head = h.Text
		}
	}
	return head
}

// tags are the tags of the whole vault and how many notes carry each,
// as a list of links: choosing one opens it, the notes that carry it,
// and choosing one of those opens it.
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
		ui.Text(c, locale.PanelsNoTags()).TextColor(
			c.Theme().TextMuted)
		return
	}
	var b strings.Builder
	for _, tag := range tags {
		b.WriteString("- [")
		b.WriteString(mdPlain(tag.Name))
		b.WriteString("](")
		b.WriteString(drawerTagHref(tag.Name))
		b.WriteString(") · ")
		b.WriteString(strconv.Itoa(tag.Count))
		b.WriteString("\n")
	}
	w.drawerMarkdown(c, b.String())
}

// tagNotes are the notes carrying the tag that was opened, and a way
// back to all of them.
func (w *Workspace) tagNotes(c *ui.Context) {
	var b strings.Builder
	b.WriteString("- [")
	// The way back: the tag it is, and all the others behind it.
	b.WriteString(mdPlain("← " + w.tagOpen))
	b.WriteString("](")
	b.WriteString(drawerTagHref(""))
	b.WriteString(")\n")
	paths := w.Index.NotesWithTag(w.tagOpen)
	for _, path := range paths {
		b.WriteString("- [")
		b.WriteString(mdPlain(w.Index.Title(path)))
		b.WriteString("](")
		b.WriteString(drawerLineHref(path, 0))
		b.WriteString(")\n")
	}
	if len(paths) == 0 {
		b.WriteString("\nno note carries it\n")
	}
	w.drawerMarkdown(c, b.String())
}

// mdPlain is text written out so that Markdown reads it as it stands:
// what would be read as marks of its own is escaped, and a line that
// was broken is kept on one. The drawer shows the names and the lines
// of notes, which are not of the reader's own writing.
func mdPlain(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	for _, mark := range []string{
		"`", "*", "_", "[", "]", "<", ">", "&", "#", "|", "-", ".",
	} {
		s = strings.ReplaceAll(s, mark, "\\"+mark)
	}
	return strings.ReplaceAll(s, "\n", " ")
}
