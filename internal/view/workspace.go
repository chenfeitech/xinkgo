package view

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/state"
)

// Workspace is the window: the vault's tree on the left, the notes in the
// middle, and what a note looks like on the right.
type Workspace struct {
	// Tree is the file tree on the left.
	Tree *Tree
	// Tabs is the notes that are open, and which one is shown.
	Tabs *Tabs
	// Layout is how wide the panels are.
	Layout state.Layout
	// Config is what the user chose; it decides what is drawn.
	Config config.AppConfig

	// OnOpenVault opens the dialog that chooses a folder, when there is
	// no vault yet.
	OnOpenVault func()
	// OnNewNote makes a note in the vault.
	OnNewNote func()
	// OnStatus is told what the window says at the bottom, such as that
	// a file changed outside the app.
	OnStatus func(text string)

	status string
}

// NewWorkspace returns a window showing nothing yet: no vault, no notes.
func NewWorkspace(cfg config.AppConfig) *Workspace {
	w := &Workspace{
		Tree:   NewTree(),
		Tabs:   NewTabs(),
		Layout: state.DefaultLayout(),
		Config: cfg,
	}
	w.Tree.ShowAll = cfg.Basic.ShowAllFiles
	w.Tabs.AutoSave = cfg.Editor.AutoSave
	w.Tabs.AutoSaveDelay = milliseconds(cfg.Editor.AutoSaveDelayMs)
	w.Tabs.SoftWrap = cfg.Editor.SoftWrap
	w.Tree.OnOpen = func(path string) {
		if _, err := w.Tabs.Open(path); err != nil {
			w.status = "could not open " + PathOf(path)
			return
		}
		w.Tree.Reveal(path)
	}
	return w
}

// milliseconds turns the configuration's delay into a duration.
func milliseconds(ms uint64) time.Duration {
	return time.Duration(ms) * time.Millisecond
}

// View draws the window.
func (w *Workspace) View(c *ui.Context) {
	ui.Column(c).Fill().Children(func() {
		ui.Row(c).Fill().Children(func() {
			w.sidebar(c)
			w.center(c)
			w.previewPanel(c)
		})
		w.statusBar(c)
	})
}

// sidebar is the file tree, and the way to open a vault when there is
// none.
func (w *Workspace) sidebar(c *ui.Context) {
	width := float32(w.Layout.Left)
	if width <= 0 {
		return
	}
	ui.Column(c).Width(width).Children(func() {
		if w.Tree.Root == nil {
			ui.Column(c).Fill().Center().Gap(8).Padding(16).Children(func() {
				ui.Text(c, "No vault yet").
					TextColor(c.Theme().TextMuted)
				if w.OnOpenVault != nil {
					if ui.PrimaryButton(c, "Open a folder").
						Label("Open a folder").Clicked() {
						w.OnOpenVault()
					}
				}
			})
			return
		}
		w.Tree.View(c)
	})
}

// center is the tabs and the editor.
func (w *Workspace) center(c *ui.Context) {
	ui.Column(c).Fill().Children(func() {
		w.tabBar(c)
		w.editor(c)
	})
}

// tabBar is one button per open note, and the way to close them.
func (w *Workspace) tabBar(c *ui.Context) {
	paths := w.Tabs.Paths()
	ui.Row(c).Gap(4).Padding(4).Children(func() {
		for _, path := range paths {
			path := path
			title := Title(path, w.Tabs.Dirty(path))
			if ui.Button(c.Key(path), title).Label(title).Clicked() {
				w.Tabs.Activate(path)
			}
			if ui.Button(c.Key(path+"#close"), "×").
				Label("close " + PathOf(path)).Clicked() {
				w.Tabs.Close(path)
				w.status = ""
			}
		}
		if w.OnNewNote != nil && w.Tree.Root != nil {
			if ui.Button(c.Key("new"), "+").Label("New note").Clicked() {
				w.OnNewNote()
			}
		}
	})
}

// editor is the note that is shown, or an empty page when none is open.
func (w *Workspace) editor(c *ui.Context) {
	n := w.Tabs.ActiveNote()
	if n == nil {
		// No notes open: one empty page, as xink keeps it.
		ui.Column(c).Fill().Center().Children(func() {
			ui.Text(c, "No note open").
				TextColor(c.Theme().TextMuted)
		})
		return
	}
	// Take in what was typed since the last frame, once per change, and
	// tell the editor when what it shows has reached the disk.
	w.Tabs.Edited(n)
	n.Synced()
	editor.View(c, n.Editor).Fill()
}

// previewPanel is what the note looks like, on the right.
func (w *Workspace) previewPanel(c *ui.Context) {
	width := float32(w.Layout.Right)
	if width <= 0 {
		return
	}
	ui.Column(c).Width(width).Children(func() {
		n := w.Tabs.ActiveNote()
		if n == nil {
			return
		}
		opts := w.markdownOptions(n.Path)
		markdown.View(c, n.Note.Text, opts).Fill()
	})
}

// markdownOptions is how the preview reads the note: its pictures come
// from the note's own folder, and a link to another note opens it here.
func (w *Workspace) markdownOptions(path string) *markdown.Options {
	return &markdown.Options{
		LoadImage: markdown.FileImages(filepath.Dir(path)),
		OnLink:    func(url string) { w.openLink(url, path) },
	}
}

// openLink is what a click in the preview does: another note of the vault
// is opened here, everything else goes to the browser.
func (w *Workspace) openLink(url, from string) {
	if strings.HasPrefix(url, "http://") ||
		strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "mailto:") {
		return // the view opens these itself
	}
	if strings.HasPrefix(url, "#") {
		return
	}
	target := url
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(from), target)
	}
	if _, err := w.Tabs.Open(target); err != nil {
		w.status = "no note at " + PathOf(target)
	}
}

// statusBar is what the window says about what happened.
func (w *Workspace) statusBar(c *ui.Context) {
	text := w.status
	if text == "" {
		n := w.Tabs.ActiveNote()
		if n != nil {
			text = languageName(n.Path)
			if n.Note.Dirty {
				text += " · not saved"
			}
		}
	}
	if text == "" {
		return
	}
	ui.Row(c).Padding(8).Children(func() {
		ui.Text(c, text).FontSize(12).TextColor(c.Theme().TextMuted)
	})
}

// Say puts a line at the bottom of the window.
func (w *Workspace) Say(text string) {
	w.status = text
}
