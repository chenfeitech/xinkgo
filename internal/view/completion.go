package view

import (
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/fuzzy"
)

// How many notes the completion offers, for more than this is no help.
const completionLimit = 30

// completion is what the app offers while a link is being written: the
// notes whose names hold the letters typed between the brackets, the
// best first.
//
// It is shown while the caret stands in a pair of brackets that are not
// closed yet, and goes away as soon as they are.
func (w *Workspace) completion(c *ui.Context, anchor ui.Element, n *OpenNote) {
	if w.Index == nil || n == nil {
		return
	}
	text := n.Editor.Buffer.String()
	start, query, typing := linkBeingTyped(text, n.Editor.Cursor())
	if !typing {
		w.completeOpen = false
		return
	}
	w.completeStart = start
	items := fuzzy.Search(w.noteNames(), query, completionLimit)
	if len(items) == 0 {
		w.completeOpen = false
		return
	}
	if w.completeChosen >= len(items) {
		w.completeChosen = 0
	}
	w.completeOpen = true
	ui.Popover(c, anchor, &w.completeOpen, func() {
		ui.Column(c).Width(320).ClipY().MaxHeight(240).Children(func() {
			for _, item := range items {
				item := item
				name := item.Text
				row := ui.Button(c.Key("complete-"+name), "")
				row.Children(func() {
					ui.Column(c).Children(func() {
						for _, part := range fuzzy.Segments(
							name, item.Positions) {
							text := ui.Text(c, part.Text)
							if !part.Found {
								text.TextColor(c.Theme().TextMuted)
							}
						}
						if title := w.Index.Title(nameOf(w.Index.Root(),
							name)); title != name {
							ui.Text(c, title).TextColor(
								c.Theme().TextMuted)
						}
					})
				})
				if row.Clicked() {
					w.complete(n, name)
				}
			}
		})
	})
}

// complete writes the note into the link that is being written: as a
// path from the vault's root, which is how a link is always written
// inside a vault, so that a note may be moved with the folder it stands
// in and still be found.
func (w *Workspace) complete(n *OpenNote, name string) {
	ed := n.Editor
	ed.Select(w.completeStart, ed.Cursor())
	ed.Insert("[[" + name + "]]")
	w.Tabs.Edited(n)
	w.completeOpen = false
	w.completeChosen = 0
}

// linkBeingTyped is the link the caret stands in: the letters after the
// last `[[` before it, when no `]]` has closed them yet, and where the
// brackets stand.
func linkBeingTyped(text string, cursor int) (int, string, bool) {
	if cursor > len(text) {
		cursor = len(text)
	}
	before := text[:cursor]
	start := strings.LastIndex(before, "[[")
	if start < 0 {
		return 0, "", false
	}
	query := before[start+2:]
	// A link that was closed already is not being written.
	if strings.Contains(query, "]]") {
		return 0, "", false
	}
	// Nor is one whose letters were left behind by an earlier line.
	if strings.ContainsAny(query, "\n") {
		return 0, "", false
	}
	return start, query, true
}

// noteNames is every note of the vault, written as it stands from its
// root: what a link inside the vault is written as.
func (w *Workspace) noteNames() []string {
	if w.Index == nil {
		return nil
	}
	root := w.Index.Root()
	paths := w.Index.Paths()
	names := make([]string, 0, len(paths))
	for _, path := range paths {
		if root == "" {
			names = append(names, path)
			continue
		}
		names = append(names, strings.TrimPrefix(strings.TrimPrefix(
			path, root), string(filepath.Separator)))
	}
	return names
}

// nameOf is the path a name of a note stands for.
func nameOf(root, name string) string {
	if root == "" {
		return name
	}
	return filepath.Join(root, filepath.FromSlash(name))
}
