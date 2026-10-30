package view

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/highlight"
	xmd "github.com/helightxu/xinkgo/internal/markdown"
	"github.com/helightxu/xinkgo/internal/state"
	"github.com/helightxu/xinkgo/internal/sync"
)

// How wide a side panel may be dragged. Below the narrowest there is
// nothing to read in it; above the widest it would cover the note.
const (
	MinPanelWidth    float32 = 120
	MaxPanelWidth    float32 = 900
	dividerGrip      float32 = 5
	dividerThickness float32 = 1
	// layoutSettle is how long the panels stay as they were dragged
	// before what the user chose is written down.
	layoutSettle = 400 * time.Millisecond
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
	// OnLayoutChanged is called when the panels have been dragged to
	// another width, so that the app can write it down.
	OnLayoutChanged func()

	status      string
	layoutTimer *time.Timer
	// codeTheme is the colors code is painted in, made again when the
	// preset or the appearance changes.
	codeTheme *editor.Theme
	themeName string
	themeDark bool

	// What the editor and the preview follow each other by.
	previewScroll ui.ScrollState
	guard         sync.Guard
	gesture       sync.Gesture
	lastEdY       float32
	lastPrY       float32
	lastMove      time.Time
	table         sync.AnchorTable
	tablePath     string
	tableVersion  uint64
	scrollPath    string
}

// codeTheme is the colors code is painted in: one theme for the editor
// and the preview alike, in the appearance the window has now.
func (w *Workspace) codeThemeFor(c *ui.Context) *editor.Theme {
	name := w.Config.Highlight.Theme
	dark := c.Theme().Dark
	if w.codeTheme == nil || w.themeName != name || w.themeDark != dark {
		w.codeTheme = highlight.EditorTheme(name, c.Theme())
		w.themeName, w.themeDark = name, dark
	}
	return w.codeTheme
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
			// A panel of no width is one the user closed: it is drawn
			// with no line to drag it back by.
			if w.Layout.Left > 0 {
				w.sidebar(c)
				w.divider(c, "divider-tree", &w.Layout.Left, 1)
			}
			w.center(c)
			if w.Layout.Right > 0 {
				w.divider(c, "divider-preview", &w.Layout.Right, -1)
				w.previewPanel(c)
			}
		})
		w.followScroll(c)
		w.statusBar(c)
	})
}

// divider is the line between two panels, that the pointer drags to give
// one of them more room and the other less. dir is 1 when dragging to the
// right widens the panel before the line, and -1 when it narrows it.
func (w *Workspace) divider(
	c *ui.Context, key string, width *float32, dir float32,
) {
	line := ui.Box(c.Key(key)).Width(dividerGrip).FillHeight().
		Focusable().Role(ui.RoleSplitter).
		Cursor(ui.CursorResizeEW).Label(key)
	line.Draw(func(p *ui.Painter, r ui.Rect) {
		color := c.Theme().Border
		if line.Hovered() || line.Focused() {
			color = c.Theme().Accent
		}
		// The line is a hairline; the box around it is what catches the
		// pointer.
		strip := ui.Rect{
			X: r.X + (r.W-dividerThickness)/2, Y: r.Y,
			W: dividerThickness, H: r.H,
		}
		p.Fill(strip, color, 0)
	})
	dx, _, dragging := line.Dragged()
	if dragging && dx != 0 {
		line.Focus()
		w.dragPanel(width, *width+dir*dx)
	}
}

// dragPanel is what a drag does to a panel's width: it is kept between
// the narrowest and the widest, and written down once it settles.
func (w *Workspace) dragPanel(width *float32, v float32) {
	v = min(max(v, MinPanelWidth), MaxPanelWidth)
	if v == *width {
		return
	}
	*width = v
	if w.layoutTimer != nil {
		w.layoutTimer.Stop()
	}
	w.layoutTimer = time.AfterFunc(layoutSettle, func() {
		if w.OnLayoutChanged != nil {
			w.OnLayoutChanged()
		}
	})
}

// sidebar is the file tree, and the way to open a vault when there is
// none.
func (w *Workspace) sidebar(c *ui.Context) {
	ui.Column(c).Width(w.Layout.Left).Children(func() {
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
	// What the side panels leave is the note's: widening the window
	// gives the text room, not the panels.
	ui.Column(c).Grow(1).FillHeight().Children(func() {
		w.tabBar(c)
		w.toolbar(c)
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

// toolbar is the row of buttons that write Markdown for the user: one
// button wraps what is selected, and stands for a pair of marks when
// nothing is.
func (w *Workspace) toolbar(c *ui.Context) {
	if w.Tabs.ActiveNote() == nil {
		return
	}
	ui.Row(c).Gap(2).Padding(4).Children(func() {
		for _, f := range xmd.Formats {
			f := f
			if ui.Button(c.Key("fmt-"+f.Label()), f.Label()).
				Label(f.Desc()).FontSize(12).Clicked() {
				w.format(f)
			}
		}
	})
}

// format is what a toolbar button does: the marks go around what is
// selected, or around the caret, and the change is taken in as typing
// would be.
func (w *Workspace) format(f xmd.Format) {
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
	w.resetScroll(n.Path)
	// Code is painted with the same colors the preview paints it with.
	n.Editor.Theme = w.codeThemeFor(c)
	editor.View(c, n.Editor).Fill()
}

// previewPanel is what the note looks like, on the right.
func (w *Workspace) previewPanel(c *ui.Context) {
	ui.Column(c).Width(w.Layout.Right).Children(func() {
		n := w.Tabs.ActiveNote()
		if n == nil {
			return
		}
		opts := w.markdownOptions(c, n.Path)
		ui.Scroll(c).TrackScroll(&w.previewScroll).Fill().Children(func() {
			markdown.View(c, n.Note.Text, opts)
		})
	})
}

// markdownOptions is how the preview reads the note: its pictures come
// from the note's own folder, and a link to another note opens it here.
func (w *Workspace) markdownOptions(
	c *ui.Context, path string,
) *markdown.Options {
	return &markdown.Options{
		Theme:     w.previewTheme(c),
		LoadImage: markdown.FileImages(filepath.Dir(path)),
		OnLink:    func(url string) { w.openLink(url, path) },
	}
}

// previewTheme is how the preview looks: what the window's theme leaves
// to it, and the code colors of the preset the user chose, the ones the
// editor paints with.
func (w *Workspace) previewTheme(c *ui.Context) *markdown.Theme {
	return &markdown.Theme{CodeColors: w.codeThemeFor(c)}
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
