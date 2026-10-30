package view

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/egoist/mygo/ui/editor"
	"github.com/helightxu/xinkgo/internal/fs"
	"github.com/helightxu/xinkgo/internal/media"
	"github.com/helightxu/xinkgo/internal/note"
	"github.com/helightxu/xinkgo/internal/session"
)

// OpenNote is a note the app has open: the file, what the editor shows,
// and whether the disk has it too.
type OpenNote struct {
	// Path is where the file is.
	Path string
	// Note is the file: its text, and whether the disk has it.
	Note *note.Note
	// Editor is the editor that shows it, with its own carets and
	// selections, kept while the tab is closed and opened again.
	Editor *editor.State
	// Version is the buffer's version when it was last looked at, so
	// that typing is taken in once and not again for the same text.
	Version uint64

	saveTimer *time.Timer
	saving    bool
}

// Tabs is the notes the app has open, in the order their tabs are in.
//
// Open is the only way in: every other way a note is opened, at start-up,
// from the tree, from a link, goes through it, so that the same file is
// never open twice and what is open is written down every time.
type Tabs struct {
	// Order is the paths of the open notes, in the order of their tabs.
	// It is the one place their order is kept: a map's order is not one.
	Order []string
	// Active is the path of the tab that is shown.
	Active string
	// Notes is what is open, by the identity of its file.
	Notes map[string]*OpenNote

	// AutoSave is whether typing is written to the disk, and
	// AutoSaveDelay how long after the last change.
	AutoSave      bool
	AutoSaveDelay time.Duration
	// SoftWrap is whether the editor wraps long lines.
	SoftWrap bool

	// OnChange is called whenever a note is opened, closed or made
	// active, so that the app can write down what is open at once.
	OnChange func()
	// OnSave is called after a note has been written to the disk.
	OnSave func(path string)
	// OnEdit is called when the text of a note changes, so that the app
	// can refresh what it shows elsewhere.
	OnEdit func(path string)

	mu sync.Mutex
}

// NewTabs returns a tab bar with nothing open.
func NewTabs() *Tabs {
	return &Tabs{
		Notes:         map[string]*OpenNote{},
		AutoSave:      true,
		AutoSaveDelay: 300 * time.Millisecond,
		SoftWrap:      true,
	}
}

// Open opens the note at path: the only way a note is opened. A note that
// is already open is only made active: it is not read again, its carets
// and where it is scrolled to are not moved, and its tab does not move.
func (t *Tabs) Open(path string) (*OpenNote, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := session.TabKey(path)
	if n, ok := t.Notes[key]; ok {
		t.Active = n.Path
		t.changed()
		return n, nil
	}
	n, err := note.Open(path)
	if err != nil {
		return nil, err
	}
	open := &OpenNote{
		Path:    path,
		Note:    n,
		Editor:  t.newEditor(path, n.Text),
		Version: 0,
	}
	t.Notes[key] = open
	t.Order = append(t.Order, path)
	t.Active = path
	t.changed()
	return open, nil
}

// newEditor is the editor of a note: the language follows the file's
// name, so that Markdown, JSON and the rest are highlighted.
func (t *Tabs) newEditor(path, text string) *editor.State {
	st := &editor.State{Buffer: editor.NewBuffer(text)}
	st.SoftWrap = t.SoftWrap
	if lang := editor.LanguageFor(filepath.Base(path)); lang != nil {
		st.Buffer.SetLanguage(lang)
	}
	return st
}

// Close closes the note at path. The last one leaves nothing open, which
// the window shows as an empty page rather than as a window with no tab.
func (t *Tabs) Close(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := session.TabKey(path)
	n, ok := t.Notes[key]
	if !ok {
		return
	}
	// A note that has not been saved is written before it is closed: the
	// user asked for the tab to go away, not for their words to.
	if n.Note.Dirty {
		_ = n.Note.Save()
	}
	if n.saveTimer != nil {
		n.saveTimer.Stop()
	}
	delete(t.Notes, key)
	kept := make([]string, 0, len(t.Order))
	for _, p := range t.Order {
		if session.TabKey(p) != key {
			kept = append(kept, p)
		}
	}
	t.Order = kept
	if session.TabKey(t.Active) == key {
		t.Active = session.ResolveActive(t.Order, "")
	}
	t.changed()
}

// Activate makes the note at path the one that is shown.
func (t *Tabs) Activate(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.Notes[session.TabKey(path)]; !ok {
		return
	}
	t.Active = path
	t.changed()
}

// ActiveNote is the note that is shown, or nil when none is open.
func (t *Tabs) ActiveNote() *OpenNote {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Notes[session.TabKey(t.Active)]
}

// Paths is what is open, in the order of the tabs: what is written down
// and read again at the next start.
func (t *Tabs) Paths() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]string, len(t.Order))
	copy(out, t.Order)
	return out
}

// CloseOutside closes the notes that are not in the vault at root, as
// when the user opens another vault.
func (t *Tabs) CloseOutside(root string) {
	t.mu.Lock()
	outside := make([]string, 0, len(t.Order))
	for _, p := range t.Order {
		if !session.Within(root, p) {
			outside = append(outside, p)
		}
	}
	t.mu.Unlock()
	for _, p := range outside {
		t.Close(p)
	}
}

// Edited is what the view calls whenever the window draws: the editor's
// text is compared with the note's, and what was typed is taken in, once
// per change.
func (t *Tabs) Edited(n *OpenNote) {
	if n == nil {
		return
	}
	version := n.Editor.Buffer.Version()
	if version == n.Version {
		return
	}
	n.Version = version
	n.Note.SetText(n.Editor.Buffer.String())
	t.scheduleSave(n)
	if t.OnEdit != nil {
		t.OnEdit(n.Path)
	}
}

// scheduleSave writes the note after the user has stopped typing. A new
// change puts off the writing, so that typing saves once and not once a
// letter.
func (t *Tabs) scheduleSave(n *OpenNote) {
	if !t.AutoSave {
		return
	}
	if n.saveTimer != nil {
		n.saveTimer.Stop()
	}
	delay := t.AutoSaveDelay
	if delay <= 0 {
		delay = 300 * time.Millisecond
	}
	n.saveTimer = time.AfterFunc(delay, func() { t.Save(n) })
}

// Save writes the note to its file now, as Cmd+S does. It writes, and
// nothing more: the editor is told its text is on the disk by the view,
// which is the only one that may touch it, and only from the main thread.
func (t *Tabs) Save(n *OpenNote) error {
	if n == nil {
		return nil
	}
	n.saving = true
	err := n.Note.Save()
	n.saving = false
	if err != nil {
		return err
	}
	if t.OnSave != nil {
		t.OnSave(n.Path)
	}
	return nil
}

// Synced tells the editor that what it shows is on the disk, so that it
// stops showing the note as changed. The view calls it from the main
// thread, which is the only one that may.
func (n *OpenNote) Synced() {
	if !n.Note.Dirty && n.Editor.Buffer.Modified() {
		n.Editor.Buffer.MarkSaved()
	}
}

// SaveActive writes the note that is shown, as Cmd+S does.
func (t *Tabs) SaveActive() error {
	return t.Save(t.ActiveNote())
}

// Dirty reports whether the note at path has changes the disk does not.
func (t *Tabs) Dirty(path string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	n, ok := t.Notes[session.TabKey(path)]
	return ok && n.Note.Dirty
}

// Reload brings what another program wrote into the note, when the user
// has no changes of their own; when they do, their text stays and the app
// is told, so that it can say so.
func (t *Tabs) Reload(path string) (bool, bool) {
	t.mu.Lock()
	n, ok := t.Notes[session.TabKey(path)]
	t.mu.Unlock()
	if !ok {
		return false, false
	}
	if n.Note.Dirty {
		return false, true // changed on disk, but the user has edits
	}
	text, changed := n.Note.TakeExternalText()
	if !changed {
		return false, false
	}
	n.Editor.Buffer.Set(text)
	n.Version = n.Editor.Buffer.Version()
	return true, false
}

// changed tells the app that what is open is different now.
func (t *Tabs) changed() {
	if t.OnChange != nil {
		t.OnChange()
	}
}

// OpenText opens the text of a note that is not on the disk yet, as a new
// note is: the file is made, then opened.
func (t *Tabs) Create(path, text string) (*OpenNote, error) {
	if err := fs.WriteNewFile(path, text); err != nil {
		return nil, fmt.Errorf("create %s: %w", path, err)
	}
	return t.Open(path)
}

// Title is what a tab says: the file's name, and a mark when it has
// changes the disk does not.
func Title(path string, dirty bool) string {
	name := PathOf(path)
	if dirty {
		return "• " + name
	}
	return name
}

// languageName is the name the editor highlights a file as, for logs.
func languageName(path string) string {
	if l := media.Language(path); l != "" {
		return l
	}
	return "plain text"
}
