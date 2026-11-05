// This file is the three panes of the window and what stands in each: the
// vault's tree, the note itself, and what the preview reads.

package view

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/helightxu/xinkgo/internal/datafile"
	"github.com/helightxu/xinkgo/internal/locale"
	xmd "github.com/helightxu/xinkgo/internal/markdown"
	"github.com/helightxu/xinkgo/internal/view/geom"
)

// sidebar is the file tree, and the way to open a vault when there is
// none.
func (w *Workspace) sidebar(c *ui.Context) {
	// A row centers what has no height of its own: the panel takes the
	// window's height, so that the tree stands at the top of it, and
	// the foot of it at the foot.
	ui.Column(c).Width(w.Layout.Left).FillHeight().Children(func() {
		ui.Row(c).Fill().Children(func() {
			w.pluginRibbon(c)
			ui.Box(c).FillHeight().Width(1).
				Background(c.Theme().Border)
			ui.Column(c).Grow(1).FillHeight().Children(func() {
				if w.Tree.Root == nil {
					ui.Column(c).Fill().Center().Gap(8).Padding(16).
						Children(func() {
							ui.Text(c, locale.NoVaultYet()).
								TextColor(c.Theme().TextMuted)
							if w.OnOpenVault != nil {
								if ui.PrimaryButton(c, locale.OpenFolder()).
									Label(locale.OpenFolder()).Clicked() {
									w.OnOpenVault()
								}
							}
						})
					// A tree that is not drawn draws no foot of its
					// own: the panel's foot is the one that stands, so
					// that the way to the settings is there before a
					// vault is opened too.
					w.Tree.footer(c)
					return
				}
				w.Tree.View(c)
			})
		})
	})
}

// followActiveNote is the tree showing the note that is on show: the
// folders above it are opened, it is chosen, and the tree is scrolled to
// it. Every way of showing a note comes through here, for what is on show
// and what the tree shows are one question, whatever answered it.
//
// A note the vault has not been read into yet is asked for again in the
// next frame, rather than left unshown for want of a tree to show it in.
// One the vault does not hold is asked for once: there is no row to bring
// into view, and the tree keeps what it had chosen.
func (w *Workspace) followActiveNote() {
	n := w.Tabs.ActiveNote()
	if n == nil || n.Path == w.treeFollowed || w.Tree.Root == nil {
		return
	}
	w.Tree.Reveal(n.Path)
	w.treeFollowed = n.Path
}

// center is the tabs and the editor.
func (w *Workspace) center(c *ui.Context) {
	// What the side panels leave is the note's: widening the window
	// gives the text room, not the panels. A note that ends at the
	// window's own right edge — the preview standing down, or given
	// over to the note — keeps the rail's strip besides: what is
	// written there is written under the rail's buttons, and its
	// padding with it.
	note := ui.Column(c).Grow(1).FillHeight()
	// However the room is shared out — a panel dragged wide, a window
	// squeezed narrow — the note keeps the room it needs, and the panels
	// are given what is left: a note written over the edge of its pane
	// is a note half read.
	note.MinWidth(w.geometry().NoteMinWidth())
	if w.geometry().NotePaneAtRail() {
		note.Padding(0, geom.RailStrip, 0, 0)
	}
	note.Children(func() {
		w.tabBar(c)
		w.toolbar(c)
		w.centerBody(c)
		w.aiBar(c)
	})
}

// centerBody is what stands between the toolbar and the conversation: the
// note itself, or the revision of it being looked over.
func (w *Workspace) centerBody(c *ui.Context) {
	if w.reviewStale() {
		w.review.open = false
	}
	if w.review.open {
		w.reviewPane(c)
		return
	}
	w.editor(c)
}

// reviewStale is whether the revision on show is of another note than the
// one being looked at. A diff of one note over the text of another is
// worse than no diff at all, so it is put away.
func (w *Workspace) reviewStale() bool {
	if !w.review.open {
		return false
	}
	n := w.Tabs.ActiveNote()
	return n == nil || n.Path != w.review.path
}

// editor is the note that is shown, or an empty page when none is open.
func (w *Workspace) editor(c *ui.Context) {
	n := w.Tabs.ActiveNote()
	if n == nil {
		// No notes open: one empty page, as xink keeps it.
		ui.Column(c).Fill().Center().Children(func() {
			ui.Text(c, locale.NoNoteOpen()).
				TextColor(c.Theme().TextMuted)
		})
		return
	}
	// A file the app cannot edit is shown and not written in: a picture
	// as a picture, and anything else as a card saying what it is.
	if n.Preview {
		w.filePane(c, n)
		return
	}
	// Take in what was typed since the last frame, once per change, and
	// tell the editor when what it shows has reached the disk.
	if w.Tabs.Edited(n) && w.Index != nil {
		// What the index knows of the note is what is typed, and not
		// what the disk holds: the links the user writes are known at
		// once, before the note is saved.
		w.Index.Put(n.Path, n.Editor.Buffer.String())
	}
	n.Synced()
	w.resetScroll(n.Path)
	// Code is painted with the same colors the preview paints it with.
	n.Editor.Theme = w.codeThemeFor(c)
	w.applyEditor(n)
	// What a long line is broken at: shorter than the room there is by
	// the air the words keep beside the editor's bar. A line that is not
	// broken is not broken at a column either.
	win, _ := c.Size()
	if w.Config.Editor.SoftWrap {
		room := w.geometry()
		n.Editor.WrapColumn = room.WrapColumn(win,
			geom.CharWidth(n.Editor.Font))
	} else {
		n.Editor.WrapColumn = 0
	}
	w.dataDiagnostics(n)
	// The room the note's text stands off the edges of its pane: the
	// padding the user chose, less what the editor keeps of its own.
	// The note is written in a theme of its own, whose scroll bar is
	// painted no color at all: the bar stands over the last of the room
	// the editor is given, and what is written there is the note's.
	ui.Themed(c, w.editorTheme(c)).Fill().
		Padding(0, w.geometry().EditorPadRight(), 0, w.geometry().EditorPadLeft()).
		Children(func() {
			ed := editor.View(c, n.Editor).Fill()
			// While a link is being written, what it may name is
			// offered.
			w.completion(c, ed, n)
		})
}

// dataBar is what the toolbar says of a file of data: which of the three
// it is, and whether it is well formed.
func (w *Workspace) dataBar(
	c *ui.Context, n *OpenNote, kind datafile.Kind,
) {
	ui.Row(c).Gap(2).Padding(4).Children(func() {
		ui.Text(c, kind.Name()).Bold()
		problem, bad := datafile.Check(kind, n.Note.Text)
		if bad {
			ui.Text(c, "· "+problem.Message).TextColor(c.Theme().Danger)
			return
		}
		ui.Text(c, locale.ProblemsWellFormed()).
			TextColor(c.Theme().TextMuted)
	})
}

// dataDiagnostics is what the app says of a file of data that is not well
// formed: the problem, underlined where it stands, so that the user is
// told of it as they write.
func (w *Workspace) dataDiagnostics(n *OpenNote) {
	kind := datafile.KindOf(n.Path)
	if kind == datafile.None {
		return
	}
	ed := n.Editor
	text := ed.Buffer.String()
	version := ed.Buffer.Version()
	problem, bad := datafile.Check(kind, text)
	if !bad {
		ed.SetDiagnostics(version, nil)
		return
	}
	start := ed.Buffer.Offset(editor.Pos{Line: problem.Line})
	end := ed.Buffer.Offset(editor.Pos{Line: problem.Line + 1})
	if end <= start {
		end = start + 1
	}
	ed.SetDiagnostics(version, []editor.Diagnostic{{
		Start:    start,
		End:      end,
		Severity: editor.SeverityError,
		Message:  problem.Message,
		Source:   kind.Name(),
	}})
}

// Format is what a toolbar button or a key does: the marks go around what
// is selected, or around the caret, and the change is taken in as typing
// would be.
func (w *Workspace) Format(f xmd.Format) {
	n := w.Tabs.ActiveNote()
	if n == nil {
		return
	}
	ed := n.Editor
	sel := ed.Selection()
	p := f.Plan(ed.Buffer.String(), sel.Anchor, sel.Head)
	ed.Select(p.Start, p.End)
	ed.Insert(p.Text)
	ed.Select(p.Caret, p.Caret)
	w.Tabs.Edited(n)
}

// editorTheme is the theme the note is written in: the window's own, but
// for the color of the thumb of a scroll bar, which is painted over the
// last of the room the editor is given, and so over the words standing
// there. Nothing of the bar is painted over the note now, and what shows
// of it elsewhere — the note's pane beside it, the panes of the window —
// is as it was.
func (w *Workspace) editorTheme(c *ui.Context) *ui.Theme {
	t := *c.Theme()
	t.Scrollbar = ui.Transparent
	return &t
}
