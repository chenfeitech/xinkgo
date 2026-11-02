package app

import (
	"encoding/json"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"reflect"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
)

// The options a setting may be one of, as the window says them and as
// they are kept.
var (
	themes = []string{
		config.ThemeSystem, config.ThemeLight, config.ThemeDark,
	}
	highlightThemes = []string{
		config.HighlightXink, config.HighlightOcean,
		config.HighlightSolarized,
	}
	syncModes = []string{
		config.SyncRatio, config.SyncAnchor, config.SyncOnStop,
	}
	imageShows = []string{config.ImageFitWidth, config.ImageHidden}
	linkOpens  = []string{config.LinkInApp, config.LinkExternal}
	densities  = []string{
		config.DensityCompact, config.DensityNormal, config.DensityRelaxed,
	}
	outputLanguages = []string{
		config.OutputAuto, config.OutputZh, config.OutputEn,
	}
)

// settings is the window of the settings: a copy of what the user chose,
// which becomes what they chose as soon as it is different from it.
type settings struct {
	cfg config.AppConfig
	// What the sliders hold: they count in whole numbers of their own,
	// and are put back into the copy as they move.
	size, lineHeight, treeSize float64
	imageWidth, width, delay   float64
	indent                     float64
	// The knobs of asking a model: how long it may take, how often it
	// is tried again, how much of a note it is shown, and how much a
	// tool may answer.
	timeout, firstByte  float64
	tries, backoff      float64
	steps               float64
	ctxChars, toolChars float64
	// scroll is where the window is scrolled to: there is more in it
	// than fits.
	scroll ui.ScrollState
	// models is what each model's row keeps between two frames.
	models map[string]*modelState
	// ollama is what the window knows of the models the machine holds:
	// the last look at them, when it was made, and what came of it. It
	// is asked for, and is kept while the window stands.
	ollama ollamaState
	// drop is the model to be forgotten, clone the one to be made into
	// two, and moveID with moveBy the one to be moved earlier or later
	// (-1 or 1). All are named by the row that was clicked and done
	// after the rows are drawn: a row must not be taken out, copied or
	// moved from under the one that follows it.
	drop   string
	clone  string
	moveID string
	moveBy int
	// vaultRename is the vault whose name is being asked for,
	// vaultName what the field holds, and vaultDrop the one to be
	// forgotten. They are named by the row that was clicked, and the
	// forgetting is done after the rows are drawn, as with the models.
	vaultRename, vaultName, vaultDrop string
	// tab is the group of the settings whose rows are on show: the
	// list on the left says which, and it is kept while the window
	// stands, and between two openings of it. shown is the group whose
	// rows were drawn last, which is how a group chosen anew is known
	// to start at its top.
	tab, shown string
}

// The groups the settings are in, as the list on the left shows them.
const (
	tabBasic   = "basic"
	tabEditor  = "editor"
	tabPreview = "preview"
	tabAi      = "ai"
	tabVault   = "vault"
	tabFile    = "file"
)

// settingsTab is one group of the settings: what the list calls it, what
// it is about, and the rows it holds.
type settingsTab struct {
	ID, Name, Note string
	Draw           func(c *ui.Context)
}

// settingsTabs is every group of the settings, in the order the list
// shows them: what the window looks like and how it is laid out first,
// then how a note is written and how it is read, then the model that is
// asked, and last the folders and the file the choices are kept in.
func (s *appState) settingsTabs() []settingsTab {
	return []settingsTab{
		{tabBasic, "Basic", "How the window looks", func(c *ui.Context) {
			s.appearance(c)
			s.window(c)
		}},
		{tabEditor, "Editor", "How a note is written",
			s.editorSettings},
		{tabPreview, "Preview", "How a note reads beside what it says",
			s.previewSettings},
		{tabAi, "AI", "Which model is asked, and how", s.ai},
		{tabVault, "Vault", "The folders the app knows",
			s.vaultSettings},
		{tabFile, "Settings file", "Given away, and taken in again",
			s.configFile},
	}
}

// openSettings shows the settings window, or brings it back when it is
// already open: there is one of them, as there is one vault.
func (s *appState) openSettings() {
	s.openSettingsTab("")
}

// openSettingsTab shows the settings window with that group of the
// settings chosen, as the vaults are asked for from the menu and from
// the tree: they are kept among the settings, and a window that stands
// already is brought back with the group chosen in it, rather than
// opened again.
func (s *appState) openSettingsTab(tab string) {
	if s.settingsWin != nil && !s.settingsWin.IsDestroyed() {
		if tab != "" {
			s.draft.tab = tab
		}
		s.settingsWin.Show()
		s.settingsWin.Focus()
		return
	}
	// The group that was on show is on show again; a first opening
	// starts with the one that is asked about most.
	if tab == "" {
		tab = s.draft.tab
	}
	if tab == "" {
		tab = tabBasic
	}
	s.draft = settings{cfg: s.config, tab: tab}
	s.draft.read(&s.config)
	s.settingsWin = mygo.NewWindow(mygo.WindowOptions{
		Title:     Name + " Settings",
		Width:     820,
		Height:    680,
		MinWidth:  640,
		MinHeight: 420,
		// The window shows the interface MyGo draws, not a web page.
		Content: ui.View(s.settingsView),
	})
}

// settingsView is the settings window: the groups in a list on the left,
// and the rows of the one chosen on the right, as Obsidian has them. A
// group is drawn only while it is chosen, so that a group thousands of
// rows deep would be no slower than a shallow one.
func (s *appState) settingsView(c *ui.Context) {
	th := c.Theme()
	ui.Row(c).Fill().Children(func() {
		s.settingsNav(c)
		ui.Box(c).FillHeight().Width(1).Background(th.Border)
		ui.Column(c).Fill().Grow(1).Children(func() {
			ui.Scroll(c).TrackScroll(&s.draft.scroll).Fill().
				Children(func() {
					ui.Column(c).FillWidth().Children(func() {
						s.chosenSettings(c)
					})
				})
		})
	})
	// What the user changed is kept, and takes effect at once.
	s.draft.write(&s.draft.cfg)
	if !reflect.DeepEqual(s.draft.cfg, s.config) {
		s.config = s.draft.cfg
		s.config.Sanitize()
		if err := s.config.Save(s.dir); err != nil {
			slog.Warn("the settings could not be written", "err", err)
		}
		s.applyConfig()
	}
}

// settingsNav is the list of the groups, on the left: choosing one shows
// its rows on the right.
func (s *appState) settingsNav(c *ui.Context) {
	ui.Sidebar(c, &s.draft.tab, func() {
		for _, tab := range s.settingsTabs() {
			ui.SidebarItem(c, tab.ID, nil, tab.Name)
		}
	}).Width(settingsNavWidth).FillHeight()
}

// chosenSettings is the group chosen: what it is about under its name,
// and its rows below.
func (s *appState) chosenSettings(c *ui.Context) {
	th := c.Theme()
	tabs := s.settingsTabs()
	if s.draft.tab == "" {
		s.draft.tab = tabs[0].ID
	}
	// A group chosen anew is scrolled to its top, and not to where
	// the one before it was left.
	if s.draft.shown != s.draft.tab {
		s.draft.shown = s.draft.tab
		s.draft.scroll = ui.ScrollState{}
	}
	for _, tab := range tabs {
		if tab.ID != s.draft.tab {
			continue
		}
		ui.Column(c).FillWidth().Gap(th.Space(0.5)).
			Padding(th.Space(6), settingsRowPad, th.Space(4)).
			Children(func() {
				ui.Text(c, tab.Name).Bold().FontSize(th.FontSize + 6)
				ui.Text(c, tab.Note).FontSize(th.FontSize - 1).
					TextColor(th.TextMuted)
			})
		tab.Draw(c)
	}
}

// appearance is how the window looks.
func (s *appState) appearance(c *ui.Context) {
	group(c, "Appearance", func() {
		choice(c, "Theme", "Light, dark, or whatever the machine has",
			&s.draft.cfg.Basic.Theme, themes)
		toggle(c, "Show every file", "Files the app cannot read are "+
			"listed by the tree too", &s.draft.cfg.Basic.ShowAllFiles)
		slider(c, "Tree font size", "How large the names in the file "+
			"tree are drawn", &s.draft.treeSize, 10, 20)
	})
}

// editorSettings is how the editor behaves.
func (s *appState) editorSettings(c *ui.Context) {
	toggle(c, "Save as I type", "The note is written to the disk "+
		"while it is written", &s.draft.cfg.Editor.AutoSave)
	slider(c, "Save after (ms)", "How long the typing pauses before "+
		"the note is written", &s.draft.delay, 200, 5000)
	toggle(c, "Wrap long lines", "A line longer than the window is "+
		"broken, rather than run out of it",
		&s.draft.cfg.Editor.SoftWrap)
	toggle(c, "Show line numbers", "A number beside every line",
		&s.draft.cfg.Editor.ShowLineNumbers)
	toggle(c, "Indent with tabs", "A tab, rather than spaces",
		&s.draft.cfg.Editor.Indent.UseTabs)
	slider(c, "Indent size", "How far an indented line starts in",
		&s.draft.indent, 1, 8)
	slider(c, "Font size", "How large the note's letters are",
		&s.draft.size, 10, 24)
	slider(c, "Line height", "How far apart the lines stand",
		&s.draft.lineHeight, 1.2, 2.2)
	choice(c, "Code theme", "The colors the code in a note is shown "+
		"in", &s.draft.cfg.Highlight.Theme, highlightThemes)
}

// previewSettings is what a note looks like beside what it says.
func (s *appState) previewSettings(c *ui.Context) {
	toggle(c, "Follow where the note is", "The preview follows the "+
		"line being written", &s.draft.cfg.Preview.SyncScroll)
	choice(c, "Follow by", "By how far down the note is, by the "+
		"heading at the top, or once the typing stops",
		&s.draft.cfg.Preview.SyncMode, syncModes)
	choice(c, "Images", "Shown as wide as the text runs, or not at "+
		"all", &s.draft.cfg.Preview.ImageDisplay, imageShows)
	slider(c, "Image width (%)", "How much of the text's width an "+
		"image is given", &s.draft.imageWidth, 20, 100)
	choice(c, "Open links", "In the app, or in the browser",
		&s.draft.cfg.Preview.LinkOpen, linkOpens)
	slider(c, "Widest the text runs", "A note is not set wider than "+
		"this, however wide the window is", &s.draft.width, 400, 1400)
	choice(c, "Density", "How much room the lines are given",
		&s.draft.cfg.Preview.Density, densities)
}

// configFile is the settings themselves: kept as a file, given away and
// taken in again.
func (s *appState) configFile(c *ui.Context) {
	buttons(c, "Kept as a file", filepath.Join(s.dir, "config.json"),
		func() {
			if ui.Button(c.Key("export-config"), "Save a copy…").
				Clicked() {
				s.exportConfig()
			}
			if ui.Button(c.Key("import-config"), "Take one in…").
				Clicked() {
				s.importConfig()
			}
			if ui.Button(c.Key("default-config"),
				"Back to the defaults").Clicked() {
				s.draft.cfg = config.Default()
				s.draft.read(&s.draft.cfg)
			}
		})
}

// exportConfig writes the settings to a file the user names, so that
// they may be taken to another machine.
func (s *appState) exportConfig() {
	go func() {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Title:       "Save the settings",
			DefaultPath: filepath.Join(s.dir, "xinkgo-config.json"),
			Filters:     []mygo.FileFilter{{Name: "JSON", Extensions: []string{"json"}}},
		})
		if err != nil || path == "" {
			return
		}
		data, err := json.MarshalIndent(s.config, "", "  ")
		if err != nil {
			slog.Warn("the settings could not be written out", "err", err)
			return
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			slog.Warn("the settings could not be written out", "err", err)
		}
	}()
}

// importConfig reads the settings from a file the user names.
func (s *appState) importConfig() {
	go func() {
		picked, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title:   "Take settings in",
			Filters: []mygo.FileFilter{{Name: "JSON", Extensions: []string{"json"}}},
		})
		if err != nil || len(picked) == 0 {
			return
		}
		data, err := os.ReadFile(picked[0])
		if err != nil {
			slog.Warn("the settings could not be read", "err", err)
			return
		}
		var cfg config.AppConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			slog.Warn("that is not a settings file", "err", err)
			return
		}
		cfg.Sanitize()
		mygo.RunOnMain(func() {
			s.config = cfg
			s.draft.cfg = cfg
			s.draft.read(&cfg)
			if err := s.config.Save(s.dir); err != nil {
				slog.Warn("the settings could not be written", "err", err)
			}
			s.applyConfig()
		})
	}()
}

// read takes what is chosen into the sliders, and write puts what the
// sliders hold back.
func (st *settings) read(c *config.AppConfig) {
	st.size = float64(c.Editor.Font.Size)
	st.lineHeight = float64(c.Editor.Font.LineHeight)
	st.treeSize = float64(c.Basic.TreeFont.Size)
	st.imageWidth = float64(c.Preview.ImageWidthPercent)
	st.width = float64(c.Preview.MaxWidth)
	st.delay = float64(c.Editor.AutoSaveDelayMs)
	st.indent = float64(c.Editor.Indent.TabSize)
	st.timeout = float64(c.Ai.TimeoutSecs)
	st.firstByte = float64(c.Ai.FirstByteTimeoutSecs)
	st.tries = float64(c.Ai.Retry.MaxAttempts)
	st.backoff = float64(c.Ai.Retry.BackoffMs)
	st.steps = float64(c.Ai.MaxSteps)
	st.ctxChars = float64(c.Ai.MaxContextChars)
	st.toolChars = float64(c.Ai.MaxToolChars)
}

func (st *settings) write(c *config.AppConfig) {
	c.Editor.Font.Size = float32(st.size)
	c.Editor.Font.LineHeight = float32(st.lineHeight)
	c.Basic.TreeFont.Size = float32(st.treeSize)
	c.Preview.ImageWidthPercent = uint32(st.imageWidth)
	c.Preview.MaxWidth = uint32(st.width)
	c.Editor.AutoSaveDelayMs = uint64(st.delay)
	c.Editor.Indent.TabSize = uint32(st.indent)
	c.Ai.TimeoutSecs = whole(st.timeout)
	c.Ai.FirstByteTimeoutSecs = whole(st.firstByte)
	c.Ai.Retry.MaxAttempts = uint8(whole(st.tries))
	c.Ai.Retry.BackoffMs = whole(st.backoff)
	c.Ai.MaxSteps = whole(st.steps)
	c.Ai.MaxContextChars = whole(st.ctxChars)
	c.Ai.MaxToolChars = whole(st.toolChars)
}

// whole is a number as a whole one: these settings count in whole
// seconds, tries and letters, and the nearest one is the one meant.
func whole(v float64) uint64 {
	return uint64(math.Round(v))
}

// window is which of the panels stand: a panel shut is given no width,
// and opening it again gives back the width it had.
func (s *appState) window(c *ui.Context) {
	if s.ws == nil {
		return
	}
	group(c, "Window", func() {
		tree := s.ws.TreeShown()
		toggle(c, "Show the file tree", "The panel on the left", &tree)
		if tree != s.ws.TreeShown() {
			s.ws.SetTreeShown(tree)
		}
		preview := s.ws.PreviewShown()
		toggle(c, "Show the preview", "The panel on the right", &preview)
		if preview != s.ws.PreviewShown() {
			s.ws.SetPreviewShown(preview)
		}
	})
}

// How wide the list of the groups is, how wide the column of the
// controls is, and how far a row starts from the window's edge: the same
// for every row, so that the controls stand in a line of their own, as
// they do in Obsidian's settings.
const (
	settingsNavWidth     float32 = 200
	settingsControlWidth float32 = 220
	settingsRowPad       float32 = 20
)

// group is one group of settings inside a category, its name above them.
func group(c *ui.Context, name string, fn func()) {
	th := c.Theme()
	ui.Column(c).FillWidth().
		Padding(th.Space(4), settingsRowPad, th.Space(1)).
		Children(func() {
			ui.Text(c, name).Bold().FontSize(th.FontSize + 2)
		})
	ui.Column(c).FillWidth().Children(fn)
}

// field is one setting: its name and what it is for on the near side,
// and what changes it on the far side, with a line under it setting it
// apart from the row below.
func field(c *ui.Context, name, note string, control func()) {
	settingRow(c, name, note, settingsControlWidth, control)
}

// buttons is a row whose far side holds buttons rather than one control:
// they stand where a control would, and take the room they need.
func buttons(c *ui.Context, name, note string, fn func()) {
	settingRow(c, name, note, 0, fn)
}

// settingRow is one row of the settings: what it is about on the near
// side, and what fn draws in a column of width on the far side, a width
// of 0 leaving it as wide as it asks to be.
func settingRow(
	c *ui.Context, name, note string, width float32, fn func(),
) {
	th := c.Theme()
	ui.Column(c).FillWidth().Children(func() {
		ui.Row(c).FillWidth().Gap(th.Space(4)).AlignItems(ui.Center).
			Padding(th.Space(2.5), settingsRowPad).Children(func() {
			ui.Column(c).Grow(1).Gap(th.Space(0.5)).Children(func() {
				ui.Text(c, name)
				if note != "" {
					ui.Text(c, note).FontSize(th.FontSize - 1).
						TextColor(th.TextMuted)
				}
			})
			control := ui.Row(c).Justify(ui.End)
			if width > 0 {
				control = control.Width(width)
			}
			control.Children(fn)
		})
		ui.Box(c).FillWidth().Height(1).Background(th.Border)
	})
}

// toggle is a setting that is on or off.
func toggle(c *ui.Context, name, note string, value *bool) {
	field(c, name, note, func() { ui.Switch(c, value).Label(name) })
}

// choice is a setting that is one of a few things.
func choice(
	c *ui.Context, name, note string, value *string, options []string,
) {
	field(c, name, note, func() { ui.Select(c, value, options) })
}

// slider is a setting between two numbers.
func slider(
	c *ui.Context, name, note string, value *float64, lo, hi float64,
) {
	field(c, name, note, func() { ui.Slider(c, value, lo, hi) })
}

// text is a setting the user writes out.
func text(
	c *ui.Context, name, note string, value *string, placeholder string,
) {
	field(c, name, note, func() {
		ui.TextInput(c, value).Placeholder(placeholder)
	})
}

// number is a setting that is a whole number between two of them: it
// says which it is, and goes up and down one at a time, which a slider
// does neither of.
func number(
	c *ui.Context, name, note string, value *float64, lo, hi float64,
) {
	field(c, name, note, func() { ui.NumberInput(c, value, lo, hi, 1) })
}
