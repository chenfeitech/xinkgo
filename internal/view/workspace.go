package view

import (
	"context"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/highlight"
	"github.com/helightxu/xinkgo/internal/index"
	"github.com/helightxu/xinkgo/internal/locale"
	xmd "github.com/helightxu/xinkgo/internal/markdown"
	"github.com/helightxu/xinkgo/internal/skill"
	"github.com/helightxu/xinkgo/internal/state"
	"github.com/helightxu/xinkgo/internal/theme"
	"github.com/helightxu/xinkgo/internal/view/geom"
	"github.com/helightxu/xinkgo/internal/wechat"
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
	// PluginCommands are the commands of the Obsidian-compatible plugins
	// that are enabled. The app fills the list; the palette only draws the
	// visible ones and reports which was chosen.
	PluginCommands []PluginCommand
	// OnPluginCommand is told the ID of a plugin command chosen in the
	// palette. It is the app's work to deliver it to the plugin runtime.
	OnPluginCommand func(id string)
	// AiModel is the model the bar asks, of the ones the settings hold:
	// the one the settings name until another is picked in the bar, and
	// that one again when the picked one is gone or turned off. It is
	// not written down: a run of the app starts by asking what the
	// settings say.
	AiModel string
	// aiPicked is whether a model was picked in the bar, rather than
	// followed from the settings.
	aiPicked bool

	// WechatPage is the page the preview shows the note in when it is
	// set out as an article for WeChat, as internal/app filled it: the
	// app is the one that makes it, of the window and on the main
	// thread, and the one that fills it, for laying a note out is not
	// the work of a frame. It is nil until a style is chosen, and where
	// the machine has no webview at all.
	WechatPage ui.NativeWebView
	// WechatThemes are the styles a note may be set out in for WeChat:
	// the ones the app was made with, and the ones the user has put in
	// its folder, in the order the menu shows them.
	WechatThemes []wechat.Theme
	// OnOpenExternally opens a file with the program the machine has
	// for it, as a PDF is previewed: the window shows it, and does not
	// pretend to be the editor of it.
	OnOpenExternally func(path string)
	// OnExport writes the note that is open out as a file another
	// program reads: a page of HTML, or the PDF printed of it. It is the
	// app that asks where it is to be put, and that writes it.
	OnExport func(kind string)
	// OnWechatStyle is told the style the preview is to set the note
	// out in for WeChat: "" for the note as Markdown reads it, and the
	// id of one of the styles for an article. It is the app that keeps
	// the choice, and that fills the page the article stands in.
	OnWechatStyle func(id string)
	// OnWechatCopy puts the note on the clipboard as it is set out, to
	// be pasted into the editor of WeChat: it is the app that reads the
	// note's pictures, and that owns the clipboard.
	OnWechatCopy func()
	// OnOpenVault opens the dialog that chooses a folder, when there is
	// no vault yet.
	OnOpenVault func()
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

	// scroll is what the note and the preview follow each other by: one
	// is scrolled by the user, and the other is brought to it.
	scroll scrollFollow

	// The command palette: whether it is open, what was typed in it,
	// and which of its answers the keys are on.
	paletteOpen   bool
	paletteQuery  string
	paletteChosen int
	paletteFocus  bool

	// search is the search of the whole vault: what was typed, and
	// what was found.
	search searchState

	// mode is the shape the window is in: which pane has been given
	// room that is not its own, how far the note has gone over the
	// preview's, and which drawer stands beside the rail. None of it is
	// written down: a run of the app starts in the shape the user left
	// the panels in, and not in a pane given the window.
	mode struct {
		// Zoom is which pane has been given the whole window, and
		// ZoomNone for the usual, in which the panels stand as the
		// user left them.
		Zoom geom.Zoom
		// Stretched is whether the note has been given the preview's
		// room: it stands over the preview, out to the window's right
		// edge, and the tree keeps the room it was given. What the
		// preview was given is left alone, so that it comes back as
		// wide as it was.
		Stretched bool
		// Stretch is how far the note has gone over that room in the
		// frame being drawn, 0 to 1: it follows Stretched by an
		// animation, so that the note is seen to stretch over the
		// preview rather than to jump over it.
		Stretch float32
		// Drawer is which panel the buttons on the rail have open
		// beside the note, empty for none.
		Drawer string
	}
	// outlineHeadings are the headings of the note that is open, read
	// again when its text has changed.
	outlineHeadings []xmd.Heading
	outlinePath     string
	outlineVersion  uint64
	// tagOpen is the tag whose notes the drawer is showing, empty for
	// all of them.
	tagOpen string

	// treeFollowed is the note the tree was last made to show. Another
	// coming on show makes the tree open the way to it and choose it,
	// whichever way the note came to be on show: a tab clicked, a link
	// followed, a note found in the palette, or the notes of the last
	// run come back.
	treeFollowed string
	// The quick words of the place to ask: what was typed into it after
	// a slash, as it was when the keys were last taken, and which of
	// the words the keys are on.
	quickQuery  string
	quickChosen int

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

// geometry is the geometry of the window as it stands now: the numbers the
// panes ask for as they draw, made afresh so that they are never behind
// the window they belong to.
func (w *Workspace) geometry() geom.Geometry {
	return geom.Of(&w.Config, w.Layout, geom.Shape{
		Zoom:    w.mode.Zoom,
		Stretch: w.mode.Stretch,
		Drawer:  w.mode.Drawer != "",
	})
}

// layoutSettle is how long the panels stay as they were dragged
// before what the user chose is written down.
const layoutSettle = 400 * time.Millisecond

// View draws the window.
func (w *Workspace) View(c *ui.Context) {
	// The colors the window is drawn in, before anything is drawn: the
	// scheme the user chose, in the appearance the window has.
	theme.Apply(c, w.Config.Basic.Theme)
	// The tree is told what is on show before it is drawn, so that the
	// note the user has just brought forward is seen in it as soon as it
	// is drawn again.
	w.followActiveNote()
	// What the user stopped typing is written now, and not by a timer of
	// its own: only this thread may touch a note. A revision standing
	// before the user is not written over the top of: what stands in
	// the note is what they have yet to take or send back.
	if !w.review.open {
		if err := w.Tabs.SaveDue(); err != nil {
			w.Say(locale.SayNoteNotSaved())
		}
	}
	// What an action's answer came to is put before the user as a
	// revision, now that it is over: a revision is made out of the
	// note's own editor, and only this thread may touch that.
	w.aiReviewDone()
	ui.Column(c).Fill().Children(func() {
		// How far the note has gone over the preview's room, read from
		// the animation before the panels are laid out from it: the
		// value is the row's own, and frames keep coming while it
		// moves.
		panels := ui.Row(c).Key("panels").Fill()
		target := float32(0)
		if w.mode.Stretched {
			target = 1
		}
		w.mode.Stretch = panels.AnimateTo("stretch", target, widenAnim)
		panels.Children(func() {
			switch w.mode.Zoom {
			case geom.ZoomEditor:
				// The note takes the window: the tree and the preview
				// are left out, without giving up the width they
				// were given, so that they come back as they were.
				w.center(c)
			case geom.ZoomPreview:
				w.previewPanel(c, 0)
			case geom.ZoomPreviewSpread:
				// The preview reaches over the note to the tree, which
				// keeps its place: what is read is read beside the
				// tree, and not over it, so that the note to read next
				// may be chosen without giving the room back first.
				// The note's own pane is not drawn at all, and, as when
				// the note takes the window, the widths are kept, so
				// that everything comes back as it was.
				if w.Layout.Left > 0 {
					w.sidebar(c)
					w.divider(c, "divider-tree", &w.Layout.Left, 1)
				}
				w.previewPanel(c, 0)
			default:
				// A panel of no width is one the user closed: it is
				// drawn with no line to drag it back by.
				if w.Layout.Left > 0 {
					w.sidebar(c)
					w.divider(c, "divider-tree", &w.Layout.Left, 1)
				}
				w.center(c)
				// The preview stands as wide as the user left it,
				// less what the note has stretched over of it: that
				// room is the preview's still, and comes back whole.
				if right := w.geometry().PreviewPaneWidth(); right >= 1 {
					w.divider(c, "divider-preview", &w.Layout.Right, -1)
					w.previewPanel(c, right)
				}
			}
			// The rail and its drawer hang over the panels, within the
			// row that holds them: the bottom of the row is where they
			// end, whatever the window's height is and whatever the
			// line at the bottom of it says.
			w.rail(c)
			// What is written out is what the preview reads, so the
			// way out of it stands over the preview and nowhere else.
			w.exportBar(c)
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

// SetPreviewShown opens the preview, or shuts it. Opening it gives back
// the width it had when it was shut. A preview that stands down takes
// the note's stretch with it: there is no room left for the note to
// stand over, nor a button to say the way back; and one that was spread
// over the note gives the note its room back, for there is no pane left
// to spread.
func (w *Workspace) SetPreviewShown(shown bool) {
	w.setShown(shown, &w.Layout.Right, &w.Layout.LastRight)
	if !shown {
		w.mode.Stretched = false
		if w.mode.Zoom == geom.ZoomPreviewSpread {
			w.mode.Zoom = geom.ZoomNone
		}
	}
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
	line := ui.Box(c.Key(key)).Width(geom.DividerGrip).FillHeight().
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
			X: r.X + (r.W-geom.DividerThickness)/2, Y: r.Y,
			W: geom.DividerThickness, H: r.H,
		}
		p.Fill(strip, color, 0)
	})
	dx, _, dragging := line.Dragged()
	if dragging && dx != 0 {
		line.Focus()
		win, _ := c.Size()
		w.dragPanel(width, w.panelWidthDragged(win, width, *width+dir*dx))
	}
}

// panelWidthDragged is how wide a drag may make a panel: what the drag
// asks for, but never so much that the note's pane is left with less
// than the note needs, once the other panel and the lines that part them
// have taken theirs. A window with no room to spare gives the panel its
// narrowest all the same: what is then too wide is cut off by the
// window's own edge, and the note is not.
func (w *Workspace) panelWidthDragged(
	win float32, width *float32, v float32,
) float32 {
	other := w.geometry().PreviewPaneWidth()
	if width == &w.Layout.Right {
		other = w.Layout.Left
	}
	room := win - other - w.geometry().NoteMinWidth() - 2*geom.DividerGrip
	return min(v, max(room, geom.MinPanelWidth))
}

// dragPanel is what a drag does to a panel's width: it is kept between
// the narrowest and the widest, and written down once it settles.
func (w *Workspace) dragPanel(width *float32, v float32) {
	v = min(max(v, geom.MinPanelWidth), geom.MaxPanelWidth)
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

// Say puts a line at the bottom of the window.
func (w *Workspace) Say(text string) {
	w.status = text
}
