package app

import (
	"encoding/json"
	"log/slog"
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
	// scroll is where the window is scrolled to: there is more in it
	// than fits.
	scroll ui.ScrollState
	// models is what each model's row keeps between two frames.
	models map[string]*modelState
	// drop is the model to be forgotten, named by the row that was
	// clicked and taken out after the rows are drawn.
	drop string
}

// openSettings shows the settings window, or brings it back when it is
// already open: there is one of them, as there is one vault.
func (s *appState) openSettings() {
	if s.settingsWin != nil && !s.settingsWin.IsDestroyed() {
		s.settingsWin.Show()
		s.settingsWin.Focus()
		return
	}
	s.draft = settings{cfg: s.config}
	s.draft.read(&s.config)
	s.settingsWin = mygo.NewWindow(mygo.WindowOptions{
		Title:    Name + " Settings",
		Width:    560,
		Height:   720,
		MinWidth: 420,
		// The window shows the interface MyGo draws, not a web page.
		Content: ui.View(s.settingsView),
	})
}

// settingsView is the settings window: what the user chose, grouped by
// what it is about.
func (s *appState) settingsView(c *ui.Context) {
	ui.Scroll(c).TrackScroll(&s.draft.scroll).Fill().Children(func() {
		ui.Column(c).Fill().Children(func() {
			s.window(c)
			s.appearance(c)
			s.editorSettings(c)
			s.previewSettings(c)
			s.vaultSettings(c)
			s.ai(c)
			s.configFile(c)
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

// appearance is how the window looks.
func (s *appState) appearance(c *ui.Context) {
	group(c, "Appearance", func() {
		choice(c, "Theme", &s.draft.cfg.Basic.Theme, themes)
		ui.Checkbox(c, &s.draft.cfg.Basic.ShowAllFiles,
			"Show every file, not only the ones that can be read")
		slider(c, "Tree font size", &s.draft.treeSize, 10, 20)
	})
}

// editorSettings is how the editor behaves.
func (s *appState) editorSettings(c *ui.Context) {
	group(c, "Editor", func() {
		ui.Checkbox(c, &s.draft.cfg.Editor.AutoSave,
			"Save as I type")
		slider(c, "Save after (ms)", &s.draft.delay, 200, 5000)
		ui.Checkbox(c, &s.draft.cfg.Editor.SoftWrap, "Wrap long lines")
		ui.Checkbox(c, &s.draft.cfg.Editor.ShowLineNumbers,
			"Show line numbers")
		ui.Checkbox(c, &s.draft.cfg.Editor.Indent.UseTabs,
			"Indent with tabs")
		slider(c, "Indent size", &s.draft.indent, 1, 8)
		slider(c, "Font size", &s.draft.size, 10, 24)
		slider(c, "Line height", &s.draft.lineHeight, 1.2, 2.2)
		choice(c, "Code theme", &s.draft.cfg.Highlight.Theme,
			highlightThemes)
	})
}

// previewSettings is what a note looks like beside what it says.
func (s *appState) previewSettings(c *ui.Context) {
	group(c, "Preview", func() {
		ui.Checkbox(c, &s.draft.cfg.Preview.SyncScroll,
			"Follow where the note is")
		choice(c, "Follow by", &s.draft.cfg.Preview.SyncMode, syncModes)
		choice(c, "Images", &s.draft.cfg.Preview.ImageDisplay, imageShows)
		slider(c, "Image width (%)", &s.draft.imageWidth, 20, 100)
		choice(c, "Open links", &s.draft.cfg.Preview.LinkOpen, linkOpens)
		slider(c, "Widest the text runs", &s.draft.width, 400, 1400)
		choice(c, "Density", &s.draft.cfg.Preview.Density, densities)
	})
}

// vaultSettings is the folders the app knows, and the window that keeps
// them.
func (s *appState) vaultSettings(c *ui.Context) {
	group(c, "Vault", func() {
		ui.Row(c).Children(func() {
			ui.Text(c, filepath.Base(s.root)).Grow(1)
			if ui.Button(c.Key("manage-vaults"), "Manage…").Clicked() {
				s.openVaultManager()
			}
			if ui.Button(c.Key("open-other-vault"), "Open another…").
				Clicked() {
				s.openVaultDialog()
			}
		})
	})
}

// configFile is the settings themselves: kept as a file, given away and
// taken in again.
func (s *appState) configFile(c *ui.Context) {
	group(c, "Settings file", func() {
		ui.Row(c).Children(func() {
			if ui.Button(c.Key("export-config"), "Save a copy…").
				Clicked() {
				s.exportConfig()
			}
			if ui.Button(c.Key("import-config"), "Take one in…").
				Clicked() {
				s.importConfig()
			}
			if ui.Button(c.Key("default-config"), "Back to the defaults").
				Clicked() {
				s.draft.cfg = config.Default()
				s.draft.read(&s.draft.cfg)
			}
		})
		ui.Text(c, filepath.Join(s.dir, "config.json")).TextColor(
			c.Theme().TextMuted)
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
}

func (st *settings) write(c *config.AppConfig) {
	c.Editor.Font.Size = float32(st.size)
	c.Editor.Font.LineHeight = float32(st.lineHeight)
	c.Basic.TreeFont.Size = float32(st.treeSize)
	c.Preview.ImageWidthPercent = uint32(st.imageWidth)
	c.Preview.MaxWidth = uint32(st.width)
	c.Editor.AutoSaveDelayMs = uint64(st.delay)
	c.Editor.Indent.TabSize = uint32(st.indent)
}

// window is which of the panels stand: a panel shut is given no width,
// and opening it again gives back the width it had.
func (s *appState) window(c *ui.Context) {
	if s.ws == nil {
		return
	}
	group(c, "Window", func() {
		tree := s.ws.TreeShown()
		ui.Checkbox(c, &tree, "Show the file tree")
		if tree != s.ws.TreeShown() {
			s.ws.SetTreeShown(tree)
		}
		preview := s.ws.PreviewShown()
		ui.Checkbox(c, &preview, "Show the preview")
		if preview != s.ws.PreviewShown() {
			s.ws.SetPreviewShown(preview)
		}
	})
}

// group is one group of settings, with its name above them.
func group(c *ui.Context, name string, fn func()) {
	ui.Column(c).Fill().Children(func() {
		ui.Text(c, name).Bold()
		ui.Column(c).Children(fn)
	})
}

// choice is a setting that is one of a few things.
func choice(c *ui.Context, label string, value *string, options []string) {
	ui.Row(c).Children(func() {
		ui.Text(c, label).Grow(1)
		ui.Select(c, value, options)
	})
}

// slider is a setting between two numbers.
func slider(c *ui.Context, label string, value *float64, lo, hi float64) {
	ui.Row(c).Children(func() {
		ui.Text(c, label).Grow(1)
		ui.Slider(c, value, lo, hi)
	})
}
