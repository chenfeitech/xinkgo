package app

import (
	"encoding/json"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/theme"
	"github.com/helightxu/xinkgo/internal/view"
	"github.com/helightxu/xinkgo/internal/wechat"
)

// How wide the row that names the folder of the styles is: as wide as a
// path written in it is likely to be, and no wider.
const wechatDirWidth float32 = 260

// The options a setting may be one of, as the window says them and as
// they are kept.
var (
	appearances = []string{
		config.AppearanceSystem, config.AppearanceLight,
		config.AppearanceDark,
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
	// The sheets of paper a note may be written out on, and what an
	// export does with the pictures it names.
	pages        = []string{config.ExportPageA4, config.ExportPageLetter}
	exportImages = []string{
		config.ExportImagesEmbed, config.ExportImagesCopy,
		config.ExportImagesNone,
	}
	// The languages the app speaks: the one the machine reads when the
	// user leaves it to follow it, and those it is written in.
	languages = []string{
		config.LanguageSystem, config.LanguageEnglish,
		config.LanguageChinese,
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
	// How far the text of a pane stands from its edges: the note's own,
	// and the preview's, whose foot and top are a knob of their own.
	editorPad, editorMargin  float64
	previewPadX, previewPadY float64
	// The knobs of asking a model: how long it may take, how often it
	// is tried again, how much of a note it is shown, and how much a
	// tool may answer.
	timeout, firstByte  float64
	tries, backoff      float64
	steps               float64
	ctxChars, toolChars float64
	// wechatDir is the folder the styles of WeChat are read from, as it
	// is being written in the row that names it, and wechatRead the one
	// the styles below it were read from: a folder is read again when
	// the user names another, and not once a frame as they write. The
	// styles, and the files of the folder that are no styles, are what
	// the last reading of it gave; wechatBase is the folder of the app's
	// own, named when the row is empty.
	wechatDir    string
	wechatRead   string
	wechatThemes []wechat.Theme
	wechatBad    []string
	wechatBase   string
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
	// quickOpen is the quick word whose row is open to be changed,
	// quickDrop the one to be forgotten, and quickMoveID with
	// quickMoveBy the one to be moved earlier or later (-1 or 1). They
	// are named by the row that was clicked and done after the rows are
	// drawn, as with the models.
	quickOpen   string
	quickDrop   string
	quickMoveID string
	quickMoveBy int
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
// it is about, the picture the list draws it by, and the rows it holds.
type settingsTab struct {
	ID, Name, Note string
	Icon           *ui.SVG
	Draw           func(c *ui.Context)
}

// settingsTabs is every group of the settings, in the order the list
// shows them: what the window looks like and how it is laid out first,
// then how a note is written and how it is read, then the model that is
// asked, and last the folders and the file the choices are kept in.
func (s *appState) settingsTabs() []settingsTab {
	return []settingsTab{
		{tabBasic, locale.SettingsTabBasic(),
			locale.SettingsTabBasicNote(), view.IconSettingsBasic,
			func(c *ui.Context) {
				s.appearance(c)
				s.window(c)
			}},
		{tabEditor, locale.SettingsTabEditor(),
			locale.SettingsTabEditorNote(), view.IconSettingsEditor,
			s.editorSettings},
		{tabPreview, locale.SettingsTabPreview(),
			locale.SettingsTabPreviewNote(), view.IconSettingsPreview,
			s.previewSettings},
		{tabAi, locale.SettingsTabAi(), locale.SettingsTabAiNote(),
			view.IconSettingsAi, s.ai},
		{tabVault, locale.SettingsTabVault(),
			locale.SettingsTabVaultNote(), view.IconSettingsVault,
			s.vaultSettings},
		{tabFile, locale.SettingsTabFile(), locale.SettingsTabFileNote(),
			view.IconSettingsFile, s.configFile},
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
	s.draft = settings{cfg: s.config, tab: tab,
		wechatBase: filepath.Join(s.dir, wechatFolder)}
	s.draft.read(&s.config)
	s.settingsWin = mygo.NewWindow(mygo.WindowOptions{
		Title:     Name + " " + locale.TitleSettings(),
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
	theme.Apply(c, s.draft.cfg.Basic.Theme)
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
		s.followCodeTheme(&s.draft.cfg)
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
			ui.SidebarItem(c, tab.ID, tab.Icon, tab.Name)
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

// appearance is how the window looks: light or dark, and in the colors
// of which scheme.
func (s *appState) appearance(c *ui.Context) {
	group(c, locale.GroupAppearance(), func() {
		choiceNamed(c, locale.SettingLanguage(),
			locale.SettingLanguageNote(), &s.draft.cfg.Basic.Language,
			languages, languageName)
		choiceNamed(c, locale.SettingLook(), locale.SettingLookNote(),
			&s.draft.cfg.Basic.Appearance, appearances, appearanceName)
		field(c, locale.SettingTheme(), locale.SettingThemeNote(), func() {
			ui.Select(c, &s.draft.cfg.Basic.Theme, theme.Names())
		})
		s.themeCards(c)
		toggle(c, locale.SettingShowAllFiles(),
			locale.SettingShowAllFilesNote(),
			&s.draft.cfg.Basic.ShowAllFiles)
		slider(c, locale.SettingTreeFontSize(),
			locale.SettingTreeFontSizeNote(), &s.draft.treeSize, 10, 20)
	})
}

// themeCards is a card of every scheme, each drawn in the colors of the
// one it stands for and in the appearance the window has: choosing one is
// a click on it, and the one chosen is the one with the accent around it.
// They stand in a grid three cards wide, so that a card is as wide as the
// room there is for it, and not wider however narrow the window is.
func (s *appState) themeCards(c *ui.Context) {
	th := c.Theme()
	ui.Grid(c).Columns(3).Gap(th.Space(2)).FillWidth().
		Padding(th.Space(1), settingsRowPad, th.Space(3)).
		Children(func() {
			for _, sc := range theme.All() {
				s.themeCard(c, sc)
			}
		})
}

// themeCard is one card: the scheme's own colors, and its name to click.
func (s *appState) themeCard(c *ui.Context, sc theme.Scheme) {
	th := c.Theme()
	// The card is drawn in the scheme it stands for, in the appearance
	// the window has now: it is what the scheme looks like, and not a
	// picture of it.
	t := sc.Of(th.Dark)
	chosen := s.draft.cfg.Basic.Theme == sc.ID
	card := ui.Themed(c, t).FillWidth().Padding(th.Space(2)).
		Gap(th.Space(1)).Radius(6)
	if chosen {
		card = card.Border(2, t.Accent)
	} else {
		card = card.Border(1, t.Border)
	}
	card.Children(func() {
		ui.Row(c).Gap(th.Space(1)).Children(func() {
			for _, one := range []ui.Color{
				t.Background, t.Surface, t.Text, t.Accent,
			} {
				ui.Box(c).Size(themeSwatch, themeSwatch).
					Background(one).Radius(themeSwatch / 2)
			}
		})
		if chosen {
			if ui.PrimaryButton(c.Key("scheme-"+sc.ID), sc.Title).
				Clicked() {
				s.draft.cfg.Basic.Theme = sc.ID
			}
		} else if ui.Button(c.Key("scheme-"+sc.ID), sc.Title).Clicked() {
			s.draft.cfg.Basic.Theme = sc.ID
		}
		ui.Text(c, sc.Note).FontSize(th.FontSize - 2).TextColor(t.TextMuted)
	})
}

// followCodeTheme is the code preset going with the scheme the user has
// just chosen: it follows the scheme, unless the user chose a preset of
// their own, which is a choice kept.
func (s *appState) followCodeTheme(next *config.AppConfig) {
	if next.Basic.Theme == s.config.Basic.Theme {
		return
	}
	if s.config.Highlight.Theme != theme.Code(s.config.Basic.Theme) {
		return
	}
	next.Highlight.Theme = theme.Code(next.Basic.Theme)
}

// editorSettings is how the editor behaves.
func (s *appState) editorSettings(c *ui.Context) {
	toggle(c, locale.SettingAutoSave(), locale.SettingAutoSaveNote(),
		&s.draft.cfg.Editor.AutoSave)
	slider(c, locale.SettingAutoSaveDelay(),
		locale.SettingAutoSaveDelayNote(), &s.draft.delay, 200, 5000)
	toggle(c, locale.SettingSoftWrap(), locale.SettingSoftWrapNote(),
		&s.draft.cfg.Editor.SoftWrap)
	toggle(c, locale.SettingLineNumbers(),
		locale.SettingLineNumbersNote(),
		&s.draft.cfg.Editor.ShowLineNumbers)
	toggle(c, locale.SettingIndentTabs(), locale.SettingIndentTabsNote(),
		&s.draft.cfg.Editor.Indent.UseTabs)
	slider(c, locale.SettingIndentSize(), locale.SettingIndentSizeNote(),
		&s.draft.indent, 1, 8)
	slider(c, locale.SettingFontSize(), locale.SettingFontSizeNote(),
		&s.draft.size, 10, 24)
	slider(c, locale.SettingLineHeight(), locale.SettingLineHeightNote(),
		&s.draft.lineHeight, 1.2, 2.2)
	slider(c, locale.SettingEditorPadding(),
		locale.SettingEditorPaddingNote(), &s.draft.editorPad,
		float64(config.MinPadding), float64(config.MaxPadding))
	slider(c, locale.SettingEditorMargin(),
		locale.SettingEditorMarginNote(), &s.draft.editorMargin,
		float64(config.DefaultMarginRight), float64(config.MaxMarginRight))
	choiceNamed(c, locale.SettingCodeTheme(),
		locale.SettingCodeThemeNote(), &s.draft.cfg.Highlight.Theme,
		highlightThemes, highlightName)
}

// previewSettings is what a note looks like beside what it says.
func (s *appState) previewSettings(c *ui.Context) {
	toggle(c, locale.SettingSyncScroll(), locale.SettingSyncScrollNote(),
		&s.draft.cfg.Preview.SyncScroll)
	choiceNamed(c, locale.SettingSyncMode(), locale.SettingSyncModeNote(),
		&s.draft.cfg.Preview.SyncMode, syncModes, syncModeName)
	choiceNamed(c, locale.SettingImages(), locale.SettingImagesNote(),
		&s.draft.cfg.Preview.ImageDisplay, imageShows, imageShowName)
	slider(c, locale.SettingImageWidth(), locale.SettingImageWidthNote(),
		&s.draft.imageWidth, 20, 100)
	choiceNamed(c, locale.SettingLinkOpen(), locale.SettingLinkOpenNote(),
		&s.draft.cfg.Preview.LinkOpen, linkOpens, linkOpenName)
	slider(c, locale.SettingMaxWidth(), locale.SettingMaxWidthNote(),
		&s.draft.width, 400, 1400)
	slider(c, locale.SettingPreviewPadding(),
		locale.SettingPreviewPaddingNote(), &s.draft.previewPadX,
		float64(config.MinPadding), float64(config.MaxPadding))
	slider(c, locale.SettingPreviewPaddingY(),
		locale.SettingPreviewPaddingYNote(), &s.draft.previewPadY,
		float64(config.MinPadding), float64(config.MaxPadding))
	choiceNamed(c, locale.SettingDensity(), locale.SettingDensityNote(),
		&s.draft.cfg.Preview.Density, densities, densityName)
	group(c, locale.GroupExport(), func() {
		choiceNamed(c, locale.ExportPageSize(),
			locale.ExportPageSizeNote(), &s.draft.cfg.Export.PageSize,
			pages, pageSizeName)
		choiceNamed(c, locale.ExportImages(),
			locale.ExportImagesNote(), &s.draft.cfg.Export.Images,
			exportImages, exportImagesName)
	})
	group(c, locale.GroupWechat(), func() {
		s.wechatDirField(c)
		s.wechatThemeRows(c)
	})
}

// wechatDirField is the folder the styles of WeChat are read from: named
// by hand, or asked of the machine with the button beside it. One file of
// JSON in it is one style, so a folder of one's own is how a style is
// added, and shared, without touching the app.
func (s *appState) wechatDirField(c *ui.Context) {
	th := c.Theme()
	buttons(c, locale.SettingWechatDir(),
		locale.SettingWechatDirNote(s.wechatDir()), func() {
			ui.Row(c).Gap(th.Space(1)).AlignItems(ui.Center).
				Children(func() {
					ui.TextInput(c, &s.draft.wechatDir).
						Placeholder(s.wechatDir()).Width(wechatDirWidth)
					if ui.Button(c.Key("wechat-dir"),
						locale.SettingWechatDirBrowse()).Clicked() {
						s.chooseWechatDir()
					}
				})
		})
}

// wechatThemeRows is what the folder named above holds: a row to a style,
// saying its name, which of it this is, who made it, and what it says of
// itself; and after them the files of the folder that are no styles, and
// why. A folder is read again when another one is named, and not once a
// frame while it is being written.
func (s *appState) wechatThemeRows(c *ui.Context) {
	th := c.Theme()
	if s.draft.wechatRead != s.draft.wechatDir {
		s.draft.wechatRead = s.draft.wechatDir
		s.draft.wechatThemes, s.draft.wechatBad = wechat.OfDir(
			s.draft.wechatDirOf())
	}
	themes, bad := s.draft.wechatThemes, s.draft.wechatBad
	if len(themes) == 0 && len(bad) == 0 {
		field(c, locale.WechatThemesNone(), "", func() {})
		return
	}
	for _, t := range themes {
		field(c, t.Name, wechatByOf(t), func() {})
	}
	if len(themes) > 0 {
		ui.Column(c).FillWidth().Children(func() {
			ui.Text(c, locale.WechatThemesBuiltin(
				strconv.Itoa(len(wechat.Builtin())))).
				FontSize(th.FontSize - 1).TextColor(th.TextMuted)
		})
	}
	// What the folder holds that is no style: a file that will not
	// read, or that names no id, or whose id another has taken.
	for _, why := range bad {
		ui.Column(c).FillWidth().Children(func() {
			ui.Text(c, why).FontSize(th.FontSize - 1).
				TextColor(th.Danger)
		})
	}
}

// wechatByOf is what a row of a style says under its name: which of it
// this is, and who made it, when its file says, and what it says of
// itself when it says anything at all.
func wechatByOf(t wechat.Theme) string {
	var b strings.Builder
	if t.Version != "" {
		b.WriteString(t.Version)
	}
	if t.Author != "" {
		if b.Len() > 0 {
			b.WriteString(" · ")
		}
		b.WriteString(t.Author)
	}
	if t.Description != "" {
		if b.Len() > 0 {
			b.WriteString(" — ")
		}
		b.WriteString(t.Description)
	}
	return b.String()
}

// configFile is the settings themselves: kept as a file, given away and
// taken in again.
func (s *appState) configFile(c *ui.Context) {
	buttons(c, locale.GroupConfigFile(),
		filepath.Join(s.dir, "config.json"), func() {
			if ui.Button(c.Key("export-config"), locale.ConfigExport()).
				Clicked() {
				s.exportConfig()
			}
			if ui.Button(c.Key("import-config"), locale.ConfigImport()).
				Clicked() {
				s.importConfig()
			}
			if ui.Button(c.Key("default-config"),
				locale.ConfigReset()).Clicked() {
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
			Title:       locale.DialogSaveSettings(),
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
			Title:   locale.DialogImportSettings(),
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
	st.editorPad = float64(c.Editor.PaddingX)
	st.editorMargin = float64(c.Editor.MarginRight)
	st.previewPadX = float64(c.Preview.PaddingX)
	st.previewPadY = float64(c.Preview.PaddingY)
	st.timeout = float64(c.Ai.TimeoutSecs)
	st.firstByte = float64(c.Ai.FirstByteTimeoutSecs)
	st.tries = float64(c.Ai.Retry.MaxAttempts)
	st.backoff = float64(c.Ai.Retry.BackoffMs)
	st.steps = float64(c.Ai.MaxSteps)
	st.ctxChars = float64(c.Ai.MaxContextChars)
	st.toolChars = float64(c.Ai.MaxToolChars)
	st.wechatDir = c.Preview.WechatThemeDir
	// What the folder named holds is read now, and again only when
	// another folder is named: not once a frame as the row is written.
	st.wechatThemes, st.wechatBad = wechat.OfDir(st.wechatDirOf())
	st.wechatRead = st.wechatDir
}

func (st *settings) write(c *config.AppConfig) {
	c.Editor.Font.Size = float32(st.size)
	c.Editor.Font.LineHeight = float32(st.lineHeight)
	c.Basic.TreeFont.Size = float32(st.treeSize)
	c.Preview.ImageWidthPercent = uint32(st.imageWidth)
	c.Preview.MaxWidth = uint32(st.width)
	c.Editor.AutoSaveDelayMs = uint64(st.delay)
	c.Editor.Indent.TabSize = uint32(st.indent)
	c.Editor.PaddingX = whole32(st.editorPad)
	c.Editor.MarginRight = whole32(st.editorMargin)
	c.Preview.PaddingX = whole32(st.previewPadX)
	c.Preview.PaddingY = whole32(st.previewPadY)
	c.Ai.TimeoutSecs = whole(st.timeout)
	c.Ai.FirstByteTimeoutSecs = whole(st.firstByte)
	c.Ai.Retry.MaxAttempts = uint8(whole(st.tries))
	c.Ai.Retry.BackoffMs = whole(st.backoff)
	c.Ai.MaxSteps = whole(st.steps)
	c.Ai.MaxContextChars = whole(st.ctxChars)
	c.Ai.MaxToolChars = whole(st.toolChars)
	c.Preview.WechatThemeDir = strings.TrimSpace(st.wechatDir)
}

// wechatDirOf is the folder the styles are read from, as the row names
// it: the one written there, and the folder of the app's own when it is
// left empty.
func (st *settings) wechatDirOf() string {
	if dir := strings.TrimSpace(st.wechatDir); dir != "" {
		return dir
	}
	return st.wechatBase
}

// whole is a number as a whole one: these settings count in whole
// seconds, tries and letters, and the nearest one is the one meant.
func whole(v float64) uint64 {
	return uint64(math.Round(v))
}

// whole32 is a number of DIPs as a whole one: a pane's padding counts in
// them, and the nearest one is the one meant.
func whole32(v float64) uint32 { return uint32(math.Round(v)) }

// window is which of the panels stand: a panel shut is given no width,
// and opening it again gives back the width it had.
func (s *appState) window(c *ui.Context) {
	if s.ws == nil {
		return
	}
	group(c, locale.GroupWindow(), func() {
		tree := s.ws.TreeShown()
		toggle(c, locale.SettingShowTree(), locale.SettingShowTreeNote(),
			&tree)
		if tree != s.ws.TreeShown() {
			s.ws.SetTreeShown(tree)
		}
		preview := s.ws.PreviewShown()
		toggle(c, locale.SettingShowPreview(),
			locale.SettingShowPreviewNote(), &preview)
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
	// themeSwatch is how large the dots of a card's colors are: the
	// same in every card, so that a card is the same size whichever
	// scheme it stands for.
	themeSwatch float32 = 10
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

// choiceNamed is a setting that is one of a few things whose names are
// said, as the languages are: what the file keeps is not what the user
// reads, so the one chosen is looked for by the name it is given, and
// what is kept stays one of options whichever language the app shows.
func choiceNamed(
	c *ui.Context, name, note string, value *string, options []string,
	label func(string) string,
) {
	labels := make([]string, len(options))
	shown := ""
	for i, one := range options {
		labels[i] = label(one)
		if one == *value {
			shown = labels[i]
		}
	}
	field(c, name, note, func() { ui.Select(c, &shown, labels) })
	// What was chosen is put back into what is kept: the names change
	// with the language, the values do not.
	for i, one := range labels {
		if one == shown {
			*value = options[i]
			break
		}
	}
}

// What one of the options of a setting is called in the language the app
// shows. They are the names of the things themselves, as the
// configuration keeps them, and not of any one language.
func languageName(v string) string {
	switch v {
	case config.LanguageEnglish:
		return locale.LanguageEnglish()
	case config.LanguageChinese:
		return locale.LanguageChineseHans()
	default:
		return locale.LanguageFollowSystem()
	}
}

func appearanceName(v string) string {
	switch v {
	case config.AppearanceLight:
		return locale.AppearanceLight()
	case config.AppearanceDark:
		return locale.AppearanceDark()
	default:
		return locale.AppearanceSystem()
	}
}

func pageSizeName(v string) string {
	switch v {
	case config.ExportPageLetter:
		return locale.ExportPageLetter()
	default:
		return locale.ExportPageA4()
	}
}

func exportImagesName(v string) string {
	switch v {
	case config.ExportImagesCopy:
		return locale.ExportImagesCopy()
	case config.ExportImagesNone:
		return locale.ExportImagesNone()
	default:
		return locale.ExportImagesEmbed()
	}
}

func syncModeName(v string) string {
	switch v {
	case config.SyncAnchor:
		return locale.SyncAnchor()
	case config.SyncOnStop:
		return locale.SyncOnStop()
	default:
		return locale.SyncRatio()
	}
}

func imageShowName(v string) string {
	if v == config.ImageHidden {
		return locale.ImageHidden()
	}
	return locale.ImageFitWidth()
}

func linkOpenName(v string) string {
	if v == config.LinkExternal {
		return locale.LinkExternal()
	}
	return locale.LinkInApp()
}

func densityName(v string) string {
	switch v {
	case config.DensityCompact:
		return locale.DensityCompact()
	case config.DensityRelaxed:
		return locale.DensityRelaxed()
	default:
		return locale.DensityNormal()
	}
}

func highlightName(v string) string {
	switch v {
	case config.HighlightOcean:
		return locale.HighlightOcean()
	case config.HighlightSolarized:
		return locale.HighlightSolarized()
	default:
		return locale.HighlightXink()
	}
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
