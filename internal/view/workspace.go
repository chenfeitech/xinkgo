package view

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/datafile"
	"github.com/helightxu/xinkgo/internal/highlight"
	"github.com/helightxu/xinkgo/internal/index"
	xmd "github.com/helightxu/xinkgo/internal/markdown"
	"github.com/helightxu/xinkgo/internal/media"
	"github.com/helightxu/xinkgo/internal/skill"
	"github.com/helightxu/xinkgo/internal/state"
	"github.com/helightxu/xinkgo/internal/sync"
	"github.com/helightxu/xinkgo/internal/theme"
)

// How wide a side panel may be dragged. Below the narrowest there is
// nothing to read in it; above the widest it would cover the note.
const (
	MinPanelWidth    float32 = 120
	MaxPanelWidth    float32 = 900
	dividerGrip      float32 = 5
	dividerThickness float32 = 1
	// previewPad is how far the preview's text stands from the edges of
	// its panel: the note is read from the top of it, and not from
	// against the window's frame.
	previewPadX float32 = 16
	previewPadY float32 = 12
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
	// Index is what is known of the whole vault: who points at whom,
	// and what is tagged how. The palette and the panels ask it.
	Index *index.Index
	// Skills are the skills a model may be told to use, found in the
	// vault and in the application's own folder. Those turned off are
	// not here at all: the app is the one that says which they are.
	Skills []skill.Skill
	// Actions are the app's own shortcuts over a note — polish,
	// condense, translate and the rest: a thing often asked for, kept
	// in the settings and offered by name. Those turned off are not
	// here either.
	Actions []config.AiAction
	// AiModel is the model the bar asks, of the ones the settings hold:
	// the one the settings name until another is picked in the bar, and
	// that one again when the picked one is gone or turned off. It is
	// not written down: a run of the app starts by asking what the
	// settings say.
	AiModel string
	// aiPicked is whether a model was picked in the bar, rather than
	// followed from the settings.
	aiPicked bool

	// OnOpenExternally opens a file with the program the machine has
	// for it, as a PDF is previewed: the window shows it, and does not
	// pretend to be the editor of it.
	OnOpenExternally func(path string)
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
	// OnAsk is the app asking a model: it is given all that has been
	// said, and the model to ask of the ones the settings hold (empty
	// for the one the settings name), and gives back what the model
	// answers, a piece at a time.
	OnAsk func(
		ctx context.Context, messages []ai.Message, model string,
	) (<-chan ai.Event, error)
	// OnAiLoad is asked for what was said about a note, the first time
	// the bar is opened for it.
	OnAiLoad func(path string) []state.AiBlock
	// OnAiSave is told what has been said about a note, so that it can
	// be kept between two runs.
	OnAiSave func(path string, blocks []state.AiBlock)
	// OnSnapshot is told what a note said before a revision was written
	// into it, so that there is a way back. It gives back where it was
	// put, if anywhere.
	OnSnapshot func(path, text string) string
	// OnVerify is asked whether a note still holds together after it
	// has been changed.
	OnVerify func(text string) []ai.Issue
	// OnOpenOutside is asked to open a link that is not a note: a page
	// on the web, in the user's browser or in a window of this app's
	// own, as the settings say.
	OnOpenOutside func(url string)

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
	// edGesture and prGesture are how long each side is left to the
	// user after they scrolled it; follow is the side the user moved
	// whose other side has yet to be brought over.
	edGesture    sync.Gesture
	prGesture    sync.Gesture
	follow       *sync.Side
	lastEdY      float32
	lastPrY      float32
	lastMove     time.Time
	table        sync.AnchorTable
	tablePath    string
	tableVersion uint64
	scrollPath   string

	// The command palette: whether it is open, what was typed in it,
	// and which of its answers the keys are on.
	paletteOpen   bool
	paletteQuery  string
	paletteChosen int
	paletteFocus  bool

	// search is the search of the whole vault: what was typed, and
	// what was found.
	search searchState

	// drawer is which panel the buttons on the rail have open beside
	// the note, empty for none.
	drawer string
	// outlineHeadings are the headings of the note that is open, read
	// again when its text has changed.
	outlineHeadings []xmd.Heading
	outlinePath     string
	outlineVersion  uint64
	// tagOpen is the tag whose notes the drawer is showing, empty for
	// all of them.
	tagOpen string

	// The completion of a link: whether it is open, where the brackets
	// stand, and which of the names offered is chosen.
	completeOpen   bool
	completeStart  int
	completeChosen int

	// ai is what has been said to a model, one conversation to a note.
	// It is kept here, and not in the note: a note is what stands on
	// the disk, and this is what was said about it.
	ai map[string]*aiSession
	// review is a revision of a note being looked over, one change at a
	// time. There is one at a time, as there is one note being looked
	// at.
	review reviewState
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
	w.Tree.OnOpen = w.openNote
	return w
}

// milliseconds turns the configuration's delay into a duration.
func milliseconds(ms uint64) time.Duration {
	return time.Duration(ms) * time.Millisecond
}

// View draws the window.
func (w *Workspace) View(c *ui.Context) {
	// The colors the window is drawn in, before anything is drawn: the
	// scheme the user chose, in the appearance the window has.
	theme.Apply(c, w.Config.Basic.Theme)
	// What the user stopped typing is written now, and not by a timer of
	// its own: only this thread may touch a note. A revision standing
	// before the user is not written over the top of: what stands in
	// the note is what they have yet to take or send back.
	if !w.review.open {
		if err := w.Tabs.SaveDue(); err != nil {
			w.Say("the note could not be saved")
		}
	}
	// What an action's answer came to is put before the user as a
	// revision, now that it is over: a revision is made out of the
	// note's own editor, and only this thread may touch that.
	w.aiReviewDone()
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
			// The rail and its drawer hang over the panels, within the
			// row that holds them: the bottom of the row is where they
			// end, whatever the window's height is and whatever the
			// line at the bottom of it says.
			w.rail(c)
		})
		w.followScroll(c)
		w.statusBar(c)
	})
	w.palette(c)
	w.searchPanel(c)
}

// TreeShown is whether the file tree stands: a panel of no width is one
// that was shut.
func (w *Workspace) TreeShown() bool { return w.Layout.TreeShown() }

// PreviewShown is whether the preview stands.
func (w *Workspace) PreviewShown() bool { return w.Layout.PreviewShown() }

// SetTreeShown opens the file tree, or shuts it. Opening it gives back
// the width it had when it was shut.
func (w *Workspace) SetTreeShown(shown bool) {
	w.setShown(shown, &w.Layout.Left, &w.Layout.LastLeft)
}

// SetPreviewShown opens the preview, or shuts it.
func (w *Workspace) SetPreviewShown(shown bool) {
	w.setShown(shown, &w.Layout.Right, &w.Layout.LastRight)
}

// setShown is one panel opened or shut. What it was is kept when it is
// shut, so that it comes back the same; and a panel that was never
// given a width is given the one a first start gives.
func (w *Workspace) setShown(shown bool, width, kept *float32) {
	if shown == (*width > 0) {
		return
	}
	if shown {
		*width = *kept
		if *width <= 0 {
			*width = w.defaultWidth(width)
		}
	} else {
		*kept = *width
		*width = 0
	}
	if w.OnLayoutChanged != nil {
		w.OnLayoutChanged()
	}
}

// defaultWidth is the width a panel of that kind is given on a first
// start.
func (w *Workspace) defaultWidth(width *float32) float32 {
	if width == &w.Layout.Left {
		return state.DefaultLayout().Left
	}
	return state.DefaultLayout().Right
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
	// A row centers what has no height of its own: the panel takes the
	// window's height, so that the tree stands at the top of it, and
	// the foot of it at the foot.
	ui.Column(c).Width(w.Layout.Left).FillHeight().Children(func() {
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
			// A tree that is not drawn draws no foot of its own: the
			// panel's foot is the one that stands, so that the way to
			// the settings is there before a vault is opened too.
			w.Tree.footer(c)
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

// iconClose is the × a tab is closed by: two strokes of a 24 by 24 box,
// drawn in the color of the text beside it.
var iconClose = svgIcon(`<path d="M18 6 6 18"/><path d="m6 6 12 12"/>`)

// How a tab is put together: its height, the room its name and its close
// button keep inside it, and how wide or narrow it may be.
const (
	tabHeight   float32 = 26
	tabPad      float32 = 8
	tabGap      float32 = 6
	tabGlyph    float32 = 12
	tabClose    float32 = 16
	tabMaxWidth float32 = 180
	// tabNameMax is how wide a tab's name may be: what is left of a tab
	// of tabMaxWidth once its padding, its close button and the room
	// between the two are taken out.
	tabNameMax float32 = 142
)

// railBand is how much room the rail takes at the right edge of the
// window: its own, and its drawer's when one is open.
func (w *Workspace) railBand() float32 {
	band := railWidth + railGap + 2
	if w.drawer != "" {
		band += drawerWidth + 2
	}
	return band
}

// railClear is how much of that room a row has to give up: the rail
// hangs from the top of the window over whatever ends there, so a row as
// wide as the window, as the tab bar is when the preview is shut, would
// end its last button under it. The preview, which ends at the window's
// own edge, is not in the way: what it leaves of the band is what a row
// beside it gives up.
func (w *Workspace) railClear() float32 {
	clear := w.railBand() - w.Layout.Right
	if clear < 0 {
		return 0
	}
	return clear
}

// tabBar is one tab per open note, and the way to make a new one: the
// tabs are read from left to right, and the strip is scrolled sideways
// when there are more of them than there is room for, as the tabs of the
// editors the reader knows are. The button that makes a note stands
// outside what is scrolled, so that it is always there.
func (w *Workspace) tabBar(c *ui.Context) {
	paths := w.Tabs.Paths()
	ui.Row(c).AlignItems(ui.Center).PaddingEnd(w.railClear()).
		Children(func() {
			ui.ScrollHorizontal(c).Grow(1).Height(tabHeight + 8).
				Children(func() {
					ui.Row(c).Gap(2).Padding(4).AlignItems(ui.Center).
						Children(func() {
							for _, path := range paths {
								path := path
								w.tab(c, path)
							}
						})
				})
			if w.OnNewNote != nil && w.Tree.Root != nil {
				if ui.Button(c.Key("new"), "+").Label("New note").Clicked() {
					w.OnNewNote()
				}
			}
		})
}

// tab is one open note: its name, marked when it has changes the disk
// does not, and the button that closes it, both inside the one tab, as
// the tabs of the editors the reader knows. The note on show is lit, and
// so is the one under the pointer.
func (w *Workspace) tab(c *ui.Context, path string) {
	th := c.Theme()
	on := w.Tabs.IsActive(path)
	name := PathOf(path)
	tab := ui.ButtonBase(c.Key("tab-"+path)).Label(name).Tooltip(path).
		Height(tabHeight).Radius(th.Radius).Padding(0, tabPad).
		Gap(tabGap).AlignItems(ui.Center).MaxWidth(tabMaxWidth)
	tab.Draw(func(p *ui.Painter, box ui.Rect) {
		switch {
		case tab.Pressed():
			p.Fill(box, th.SurfacePressed, th.Radius)
		case on:
			p.Fill(box, th.Surface, th.Radius)
		case tab.Hovered():
			p.Fill(box, th.SurfaceHover, th.Radius)
		}
	})
	tab.Children(func() {
		ui.Row(c).FillWidth().Gap(tabGap).AlignItems(ui.Center).
			Children(func() {
				title := ui.Text(c, Title(path, w.Tabs.Dirty(path))).
					Grow(1).MaxWidth(tabNameMax).Ellipsis("…").NoWrap()
				if !on {
					title.TextColor(th.TextMuted)
				}
				w.tabClose(c, path, on)
			})
	})
	if tab.Clicked() {
		w.Tabs.Activate(path)
	}
}

// tabClose is the × of a tab: it stands within the tab, and closes the
// note without showing it first.
func (w *Workspace) tabClose(c *ui.Context, path string, on bool) {
	th := c.Theme()
	name := PathOf(path)
	b := ui.ButtonBase(c.Key(path+"#close")).Label("close "+name).
		Tooltip("Close "+name).Size(tabClose, tabClose).
		Radius(tabClose / 2).Justify(ui.Center).AlignItems(ui.Center)
	b.Draw(func(p *ui.Painter, box ui.Rect) {
		switch {
		case b.Pressed():
			p.Fill(box, th.SurfacePressed, tabClose/2)
		case b.Hovered():
			p.Fill(box, th.SurfaceHover, tabClose/2)
		}
	})
	b.Children(func() {
		glyph := ui.Icon(c, iconClose).Size(tabGlyph, tabGlyph)
		if !on && !b.Hovered() {
			glyph.TextColor(th.TextMuted)
		}
	})
	if b.Clicked() {
		w.Tabs.Close(path)
		w.status = ""
	}
}

// toolbar is the row of buttons that write Markdown for the user: one
// button wraps what is selected, and stands for a pair of marks when
// nothing is. A button is a picture, and what it does is said in words
// only when the pointer rests on it, as in a text editor's toolbar; what
// there is no room for goes into a menu at its end, so that the row never
// runs wider than the note it writes in.
func (w *Workspace) toolbar(c *ui.Context) {
	n := w.Tabs.ActiveNote()
	if n == nil {
		return
	}
	// A file of data has no use of the buttons that write Markdown: it
	// is told which it is, and whether it reads. Nor has a file the app
	// only shows, which has nothing to write in at all.
	if kind := datafile.KindOf(n.Path); kind != datafile.None {
		w.dataBar(c, n, kind)
		return
	}
	if n.Preview {
		w.fileBar(c, n)
		return
	}
	// The buttons are as close together as an editor's are, so that
	// all of them are seen side by side rather than in the menu. What
	// there is no room for goes into the menu at the toolbar's end;
	// the way to the model is not one of them, so it stands outside it.
	ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
		ui.Toolbar(c, func() {
			for i, group := range xmd.Groups {
				if i > 0 {
					w.toolbarDivider(c)
				}
				for _, f := range group {
					f := f
					if ui.Button(c.Key("fmt-"+f.Label()), "").
						Label(f.Label()).Tooltip(f.Desc()).
						Padding(4, 5).
						Children(func() {
							ui.Icon(c, formatIcon(f)).
								Size(iconSize, iconSize)
						}).Clicked() {
						w.Format(f)
					}
				}
			}
		}).Gap(2).Padding(3, 4).MaxWidth(w.markdownWidth(c)).
			Label("Markdown")
		w.toolbarDivider(c)
		w.aiButton(c)
	})
}

// markdownWidth is how wide the row of the Markdown buttons may be: what
// the panels leave, less the room the way to the model takes beside it, so
// that it stands beside the row and not under the rail.
func (w *Workspace) markdownWidth(c *ui.Context) float32 {
	room := w.editorWidth(c) - aiBand
	// Narrower than a few buttons there is nothing to show but the menu
	// at the end, which is still worth its place.
	if room < iconSize*3 {
		return iconSize * 3
	}
	return room
}

// aiButton is the way to the model: a button of the toolbar's own kind,
// a picture with a word said of it when the pointer rests on it, and the
// conversation under the note opened or hidden by a click.
func (w *Workspace) aiButton(c *ui.Context) {
	n := w.Tabs.ActiveNote()
	if n == nil {
		return
	}
	button := ui.Button(c.Key("ai-open"), "").
		Label("AI 辅助").Tooltip("AI 辅助：在笔记下方打开对话框").
		Padding(4, 5).
		Children(func() {
			ui.Icon(c, iconAi).Size(iconSize, iconSize)
		})
	if w.aiOpen(n.Path) {
		button.Selected(true)
	}
	if button.Clicked() {
		w.ToggleAI()
	}
}

// toolbarDivider is the line that sets two groups of the toolbar apart.
func (w *Workspace) toolbarDivider(c *ui.Context) {
	ui.Box(c).Width(1).Height(iconDivider).
		Background(c.Theme().Border)
}

// editorWidth is how wide the note is, which is how wide its toolbar may
// be: what the panels on either side leave of the window, and what the
// rail and its drawer leave of it besides.
func (w *Workspace) editorWidth(c *ui.Context) float32 {
	width, _ := c.Size()
	return max(width-w.Layout.Left-w.Layout.Right-w.railClear(), 0)
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
		ui.Text(c, "· well formed").TextColor(c.Theme().TextMuted)
	})
}

// previewText is what the preview is given: a note as it is, its links
// in double brackets made into ones that can be clicked, and a file of
// data as a block of code, written out evenly when it can be.
func (w *Workspace) previewText(n *OpenNote) string {
	// A file the app only shows is not read as Markdown: a picture is
	// shown as the picture it is, and the rest are said in a line.
	if n.Preview {
		return w.previewTextOfFile(n)
	}
	kind := datafile.KindOf(n.Path)
	if kind == datafile.None {
		return xmd.Links(n.Note.Text, w.resolveLink)
	}
	text := n.Note.Text
	// A file of megabytes is shown as it stands.
	if !datafile.TooLarge(uint64(len(text)),
		w.Config.Editor.DataFile.MaxFormatBytes) {
		if even, ok := datafile.Format(kind, text,
			int(w.Config.Editor.DataFile.Indent)); ok {
			text = even
		}
	}
	return "```" + strings.ToLower(kind.Name()) + "\n" + text + "\n```"
}

// resolveLink is what a link in double brackets names: the note of the
// vault, when there is one and only one.
func (w *Workspace) resolveLink(target string) (string, bool) {
	if w.Index == nil {
		return "", false
	}
	return w.Index.ResolveOnly(target)
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
	w.dataDiagnostics(n)
	ed := editor.View(c, n.Editor).Fill()
	// While a link is being written, what it may name is offered.
	w.completion(c, ed, n)
}

// previewPanel is what the note looks like, on the right.
func (w *Workspace) previewPanel(c *ui.Context) {
	// As the tree's panel: a note is read from its first line, and not
	// from the middle of the window.
	ui.Column(c).Width(w.Layout.Right).FillHeight().Children(func() {
		n := w.Tabs.ActiveNote()
		if n == nil {
			return
		}
		opts := w.markdownOptions(c, n.Path)
		ui.Scroll(c).TrackScroll(&w.previewScroll).Fill().Children(func() {
			width, _ := c.Size()
			ui.Box(c).MaxWidth(w.previewWidth(width)).
				Padding(previewPadY, previewPadX).Children(func() {
				markdown.View(c, w.previewText(n), opts)
			})
		})
	})
}

// previewTextOfFile is what the preview is given of a file the app only
// shows: the picture itself, whose name the preview reads as one of the
// note's own, or a line saying what the file is.
func (w *Workspace) previewTextOfFile(n *OpenNote) string {
	switch {
	case media.IsImage(n.Path):
		return "![<" + PathOf(n.Path) + ">](<" + PathOf(n.Path) + ">)"
	default:
		return "_" + w.sayOfFile(n) + "_"
	}
}

// markdownOptions is how the preview reads the note: its pictures come
// from the note's own folder, and a link to another note opens it here.
func (w *Workspace) markdownOptions(
	c *ui.Context, path string,
) *markdown.Options {
	opts := &markdown.Options{
		Theme:  w.previewTheme(c),
		OnLink: func(url string) { w.openLink(url, path) },
	}
	// Pictures come from the note's own folder, unless the user has
	// asked for none of them.
	if w.Config.Preview.ImageDisplay != config.ImageHidden {
		opts.LoadImage = markdown.FileImages(filepath.Dir(path))
	}
	return opts
}

// openLink is what a click in the preview does: a link in double brackets
// opens the note it names, a page on the web goes where the user said
// links should go, and anything else is a file beside the note.
func (w *Workspace) openLink(url, from string) {
	if target, ok := xmd.WikiTarget(url); ok {
		w.openWikiLink(target)
		return
	}
	// A page on the web is not opened by the view when the window says
	// what a click does: it is for the app to open, or for nobody.
	if outsideLink(url) {
		if w.OnOpenOutside != nil {
			w.OnOpenOutside(url)
		}
		return
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

// openWikiLink is a click on a link in double brackets: the note it names
// is opened. It is looked up again now, and not when the note was
// shown: the vault may have gained the note since.
func (w *Workspace) openWikiLink(target string) {
	path, ok := w.resolveLink(target)
	if !ok {
		w.status = "no note called " + target
		return
	}
	if _, err := w.Tabs.Open(path); err != nil {
		w.status = "could not open " + PathOf(path)
	}
}

// outsideLink is whether a link is a page on the web or a letter, and so
// not a file of the vault.
func outsideLink(url string) bool {
	return strings.HasPrefix(url, "http://") ||
		strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "mailto:")
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
