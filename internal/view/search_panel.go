package view

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/helightxu/xinkgo/internal/search"
)

// What a search holds to: how long the user is given to go on typing
// before the vault is read, and how much of it is read at once.
const (
	searchDebounce = 200 * time.Millisecond
	searchWidth    = 720
)

// OpenSearch is the search opened: the words typed last time are kept,
// for they may well be what is looked for again.
func (w *Workspace) OpenSearch() {
	w.search.open = true
	w.search.focus = true
	w.search.chosen = 0
}

// searchState is what the search holds: the words that were typed, what
// was found, and the reading that is still to come. It is kept in one
// piece, for what is found comes from another goroutine.
type searchState struct {
	open   bool
	focus  bool
	query  string
	typed  string
	chosen int
	// at is when the vault is read: the user is given a moment to go
	// on typing before it is.
	at time.Time
	// seq is which reading was asked for last, so that an older one
	// coming in later is not shown.
	seq  int
	mu   sync.Mutex
	hits []search.Hit
}

// searchPanel is the search of the whole vault: a few words, and every
// place they stand. It is drawn over the window, and the reading of the
// vault is given to the background, so that typing is never kept waiting.
func (w *Workspace) searchPanel(c *ui.Context) {
	if !w.search.open {
		return
	}
	ui.Modal(c, &w.search.open, func() {
		ui.Column(c).Width(searchWidth).Children(func() {
			field := ui.TextInput(c.Key("search"), &w.search.query).Grow(1)
			if w.search.focus {
				field.Focus()
				w.search.focus = false
			}
			if field.Shortcut(0, ui.KeyEscape) {
				w.search.open = false
				return
			}
			if field.Submitted() {
				w.openHit(w.search.chosen)
				return
			}
			if field.Shortcut(0, ui.KeyDown) {
				w.search.chosen++
			}
			if field.Shortcut(0, ui.KeyUp) && w.search.chosen > 0 {
				w.search.chosen--
			}
			// The vault is read when the user has stopped typing, and
			// not at every letter.
			if w.search.query != w.search.typed {
				w.search.typed = w.search.query
				w.search.at = time.Now().Add(searchDebounce)
			}
			if !w.search.at.IsZero() && time.Now().After(w.search.at) {
				w.search.at = time.Time{}
				go w.readVault(w.search.query)
			}
			w.searchResults(c)
		})
	})
}

// searchResults is what was found: the note and the line of each, and
// the line itself to read by.
func (w *Workspace) searchResults(c *ui.Context) {
	w.search.mu.Lock()
	hits := append([]search.Hit{}, w.search.hits...)
	w.search.mu.Unlock()

	if strings.TrimSpace(w.search.query) == "" {
		ui.Text(c, "what to look for").TextColor(c.Theme().TextMuted)
		return
	}
	if len(hits) == 0 {
		ui.Text(c, "not found in the vault").TextColor(c.Theme().TextMuted)
		return
	}
	if w.search.chosen >= len(hits) {
		w.search.chosen = len(hits) - 1
	}
	ui.Column(c).ClipY().MaxHeight(360).Children(func() {
		for i, hit := range hits {
			i, hit := i, hit
			row := ui.Button(c.Key(hit.Path+"#"+strconv.Itoa(hit.Line)), "")
			row.Children(func() {
				ui.Row(c).Children(func() {
					ui.Text(c, PathOf(hit.Path)+":"+
						strconv.Itoa(hit.Line+1)).TextColor(
						c.Theme().TextMuted)
					ui.Text(c, hit.Text).Grow(1)
				})
			})
			if i == w.search.chosen {
				row.Selected(true)
			}
			if row.Clicked() {
				w.openHit(i)
			}
		}
	})
}

// readVault reads the notes of the vault for the words, in the
// background. Only the last reading is kept: an older one that comes in
// later would show answers to words already changed.
func (w *Workspace) readVault(query string) {
	if w.Index == nil {
		return
	}
	paths := w.Index.Paths()
	w.search.mu.Lock()
	w.search.seq++
	seq := w.search.seq
	w.search.mu.Unlock()

	hits := search.Files(paths, query, search.MaxHits)
	search.Sortable(hits)

	w.search.mu.Lock()
	defer w.search.mu.Unlock()
	if seq != w.search.seq {
		return
	}
	w.search.hits = hits
	w.search.chosen = 0
}

// openHit opens the note a hit stands in, and puts the caret on its
// line, as an outline entry and a search result both ask.
func (w *Workspace) openHit(i int) {
	w.search.mu.Lock()
	hits := append([]search.Hit{}, w.search.hits...)
	w.search.mu.Unlock()
	if i < 0 || i >= len(hits) {
		return
	}
	w.search.open = false
	w.openNote(hits[i].Path)
	if n := w.Tabs.ActiveNote(); n != nil && n.Path == hits[i].Path {
		n.GoToLine(hits[i].Line)
	}
}

// GoToLine puts the caret at the start of a line and brings the line
// into view, as when a search result or an outline entry is chosen.
func (n *OpenNote) GoToLine(line int) {
	ed := n.Editor
	if line < 0 {
		line = 0
	}
	if last := ed.Buffer.LineCount() - 1; line > last {
		line = last
	}
	offset := ed.Buffer.Offset(editor.Pos{Line: line})
	ed.Select(offset, offset)
	// The line is brought a little below the top, so that what stands
	// before it can be read too.
	ed.SetScrollY(ed.LineTop(line) - 80)
}
