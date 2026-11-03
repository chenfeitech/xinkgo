// Package config holds the application's preferences, as xink keeps them:
// one file, `config.json`, in the application's own folder.
// one JSON file with five sections, whose fields, defaults and limits are
// the same, so a xink user can switch to Xinkgo and keep their settings.
//
// Every section is optional in the file: a field the file leaves out takes
// its default, and a field the file has but this version does not know is
// ignored. Whatever comes in is then clamped by Sanitize, which is what
// keeps a hand-edited or older file usable.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/helightxu/xinkgo/internal/keymap"
	"github.com/helightxu/xinkgo/internal/theme"
)

// FileName is the configuration file, in the application's user data
// directory.
const FileName = "config.json"

// Values the configuration's enumerations can hold.
const (
	// The appearances: light, dark, or whichever the machine has.
	AppearanceSystem = "system"
	AppearanceLight  = "light"
	AppearanceDark   = "dark"

	ImageFitWidth = "fit_width"
	ImageHidden   = "hidden"

	LinkInApp    = "in_app"
	LinkExternal = "external"

	SyncRatio  = "ratio"
	SyncAnchor = "anchor"
	SyncOnStop = "on_stop"

	DensityCompact = "compact"
	DensityNormal  = "normal"
	DensityRelaxed = "relaxed"

	HighlightXink      = "xink"
	HighlightOcean     = "ocean"
	HighlightSolarized = "solarized"

	// The sheets of paper a note may be written out on.
	ExportPageA4     = "a4"
	ExportPageLetter = "letter"

	// What an export does with the pictures a note names: written into
	// the page, put beside it, or left out.
	ExportImagesEmbed = "embed"
	ExportImagesCopy  = "copy"
	ExportImagesNone  = "none"

	// The languages the app speaks. A language is a BCP 47 tag, as
	// MyGo's locales name them, but for LanguageSystem, which follows
	// the languages the machine reads.
	LanguageSystem  = "system"
	LanguageEnglish = "en"
	LanguageChinese = "zh-Hans"
)

// How far the text of a pane stands from the edges of it, in DIPs: the
// room the note's own and the preview's are given at a first start, and
// how near to and how far from their edges they may be put. A pane whose
// text touches its edges is not one to read, and one whose room is wider
// than a few words is a pane of nothing but room.
const (
	DefaultPaddingX uint32 = 26
	DefaultPaddingY uint32 = 12
	MinPadding      uint32 = 4
	MaxPadding      uint32 = 64
	// MarginRight is how far the note's words may be made to stand from
	// the right edge of its pane over and above the padding, in DIPs: a
	// margin of the note's own, and not of the pane. None is asked for at
	// a first start, for none is needed: the room the editor's own bar
	// and the rail's buttons stand in is kept whether it is asked for or
	// not.
	DefaultMarginRight uint32 = 0
	MaxMarginRight     uint32 = 120
)

// FontConfig is a font and how it is set.
type FontConfig struct {
	Family     string  `json:"family"`
	Size       float32 `json:"size"`
	LineHeight float32 `json:"line_height"`
	Weight     string  `json:"weight"`
}

// IndentConfig is what indenting inserts.
type IndentConfig struct {
	UseTabs bool   `json:"use_tabs"`
	TabSize uint32 `json:"tab_size"`
}

// BasicConfig is what the whole application uses.
type BasicConfig struct {
	// Appearance is light, dark, or whichever the machine has.
	Appearance string `json:"appearance"`
	// Theme is the name of the color scheme the interface is drawn in,
	// as internal/theme keeps them.
	Theme        string     `json:"theme"`
	Language     string     `json:"language"`
	IconSize     uint32     `json:"icon_size"`
	TreeFont     FontConfig `json:"tree_font"`
	ShowAllFiles bool       `json:"show_all_files"`
}

// DataFileConfig is how JSON, YAML and TOML are read and written.
type DataFileConfig struct {
	Indent         uint64 `json:"indent"`
	MaxFormatBytes uint64 `json:"max_format_bytes"`
}

// EditorConfig is the editor's own settings.
type EditorConfig struct {
	AutoSave        bool         `json:"auto_save"`
	AutoSaveDelayMs uint64       `json:"auto_save_delay_ms"`
	Indent          IndentConfig `json:"indent"`
	ShowLineNumbers bool         `json:"show_line_numbers"`
	SoftWrap        bool         `json:"soft_wrap"`
	// PaddingX is how far the note's text stands from the sides of its
	// pane, in DIPs: the same on both, so that a note reads as a column
	// in the middle of the room it was given.
	PaddingX uint32 `json:"padding_x"`
	// MarginRight is how far the note's words stand from the right edge
	// of its pane over and above that padding, in DIPs: a margin of the
	// note's own, and not of the pane, asked for when a note reads better
	// with more room on that side than on the other, as it does widened
	// over the preview, with the rail standing at that end of it. A line
	// is broken short of it, and of what the editor's own bar and the
	// rail's buttons stand in besides, whichever of the two is the wider.
	MarginRight uint32            `json:"margin_right"`
	Font        FontConfig        `json:"font"`
	Keymap      map[string]string `json:"keymap"`
	DataFile    DataFileConfig    `json:"data_file"`
}

// PreviewConfig is the Markdown preview's settings.
type PreviewConfig struct {
	Font FontConfig `json:"font"`
	// LegacyCodeTheme is the code theme of versions that had it here; it
	// is read once, moved into Highlight.Theme, and never written again.
	LegacyCodeTheme   *string `json:"legacy_code_theme,omitempty"`
	ImageDisplay      string  `json:"image_display"`
	ImageWidthPercent uint32  `json:"image_width_percent"`
	LinkOpen          string  `json:"link_open"`
	SyncScroll        bool    `json:"sync_scroll"`
	SyncMode          string  `json:"sync_mode"`
	// PaddingX is how far what is read stands from the sides of the
	// preview's pane, and PaddingY from its top and its foot, in DIPs:
	// what keeps a note off the edges of the pane it is read in, and off
	// the line that parts it from the note's own.
	PaddingX        uint32 `json:"padding_x"`
	PaddingY        uint32 `json:"padding_y"`
	MaxWidth        uint32 `json:"max_width"`
	Density         string `json:"density"`
	DebugScrollSync bool   `json:"debug_scroll_sync"`
	// Wechat is the style a note is set out in for WeChat, as
	// internal/wechat names them: "" for the note as Markdown reads it,
	// which is the way a first start has it, and the way it is read
	// again when a style the user has taken away is named here.
	Wechat string `json:"wechat"`
	// WechatThemeDir is the folder the styles of WeChat are read from:
	// one file of JSON to a style, which is how a style is made without
	// touching the app. Empty is the folder of the app's own, which is
	// the way a first start has it.
	WechatThemeDir string `json:"wechat_theme_dir"`
}

// HighlightConfig is the theme of code blocks.
type HighlightConfig struct {
	Theme string `json:"theme"`
}

// ExportConfig is how a note is written out: on which sheet of paper,
// and what becomes of the pictures it names.
type ExportConfig struct {
	PageSize string `json:"page_size"`
	Images   string `json:"images"`
}

// fillMissing gives the fields the file left out their defaults.
func (x *ExportConfig) fillMissing(d ExportConfig) {
	if x.PageSize == "" {
		x.PageSize = d.PageSize
	}
	if x.Images == "" {
		x.Images = d.Images
	}
}

// AppConfig is the whole configuration.
type AppConfig struct {
	Basic     BasicConfig     `json:"basic"`
	Editor    EditorConfig    `json:"editor"`
	Preview   PreviewConfig   `json:"preview"`
	Highlight HighlightConfig `json:"highlight"`
	Export    ExportConfig    `json:"export"`
	Ai        AiConfig        `json:"ai"`
}

// Default returns the configuration of a first start.
func Default() AppConfig {
	return AppConfig{
		Basic: BasicConfig{
			Appearance:   AppearanceSystem,
			Theme:        theme.Default,
			Language:     LanguageSystem,
			IconSize:     16,
			TreeFont:     bodyFont(),
			ShowAllFiles: true,
		},
		Editor: EditorConfig{
			AutoSave:        true,
			AutoSaveDelayMs: 300,
			Indent:          IndentConfig{UseTabs: false, TabSize: 4},
			ShowLineNumbers: false,
			SoftWrap:        true,
			PaddingX:        DefaultPaddingX,
			MarginRight:     DefaultMarginRight,
			Font:            monoFont(),
			Keymap:          map[string]string{},
			DataFile:        DataFileConfig{Indent: 2, MaxFormatBytes: 1_000_000},
		},
		Preview: PreviewConfig{
			Font:              bodyFont(),
			ImageDisplay:      ImageFitWidth,
			ImageWidthPercent: 100,
			LinkOpen:          LinkInApp,
			SyncScroll:        true,
			SyncMode:          SyncRatio,
			PaddingX:          DefaultPaddingX,
			PaddingY:          DefaultPaddingY,
			MaxWidth:          0,
			Density:           DensityNormal,
			DebugScrollSync:   false,
		},
		Highlight: HighlightConfig{Theme: HighlightXink},
		Export: ExportConfig{
			PageSize: ExportPageA4,
			Images:   ExportImagesEmbed,
		},
		Ai: DefaultAi(),
	}
}

// bodyFont is the font of the interface, and of the preview.
func bodyFont() FontConfig {
	return FontConfig{
		Family:     ".SystemUIFont",
		Size:       14,
		LineHeight: 1.5,
		Weight:     "regular",
	}
}

// monoFont is the font of the editor.
func monoFont() FontConfig {
	return FontConfig{
		Family:     ".SystemMonospacedFont",
		Size:       14,
		LineHeight: 1.6,
		Weight:     "regular",
	}
}

// LocaleTag is the language the app is to show, as MyGo names it: a BCP 47
// tag of the languages the app speaks, or "" to follow the languages the
// machine reads, which is what LanguageSystem means.
func (b BasicConfig) LocaleTag() string {
	switch b.Language {
	case LanguageEnglish, LanguageChinese:
		return b.Language
	default:
		return ""
	}
}

// Load reads the configuration from dir. A file that is missing gives the
// defaults; one that cannot be read or parsed gives the defaults and an
// error, so the caller can say so and still start.
func Load(dir string) (AppConfig, error) {
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Default(), fmt.Errorf("read configuration: %w", err)
	}
	// The sections are pointers while the file is read, so that a section
	// that is there is told apart from one that is not: without that, a
	// file with only one field in a section would have the whole section
	// replaced by the defaults, and lose what the user set.
	var f fileConfig
	if err := json.Unmarshal(data, &f); err != nil {
		return Default(), fmt.Errorf("parse configuration: %w", err)
	}
	cfg := fromFile(f)
	cfg.migrate()
	cfg.Sanitize()
	return cfg, nil
}

// fileConfig is AppConfig as the file has it: a section the file leaves out
// is nil, one it has is not, however few of its fields it sets.
type fileConfig struct {
	Basic     *BasicConfig     `json:"basic"`
	Editor    *EditorConfig    `json:"editor"`
	Preview   *PreviewConfig   `json:"preview"`
	Highlight *HighlightConfig `json:"highlight"`
	Export    *ExportConfig    `json:"export"`
	Ai        *AiConfig        `json:"ai"`
}

// fromFile builds the configuration from what the file had, filling in the
// fields, and the sections, it left out.
func fromFile(f fileConfig) AppConfig {
	d := Default()
	cfg := d
	if f.Basic != nil {
		cfg.Basic = *f.Basic
		cfg.Basic.fillMissing(d.Basic)
	}
	if f.Editor != nil {
		cfg.Editor = *f.Editor
		cfg.Editor.fillMissing(d.Editor)
	}
	if f.Preview != nil {
		cfg.Preview = *f.Preview
		cfg.Preview.fillMissing(d.Preview)
	}
	if f.Highlight != nil {
		cfg.Highlight = *f.Highlight
		if cfg.Highlight.Theme == "" {
			cfg.Highlight.Theme = d.Highlight.Theme
		}
	}
	if f.Export != nil {
		cfg.Export = *f.Export
		cfg.Export.fillMissing(d.Export)
	}
	if f.Ai != nil {
		cfg.Ai = *f.Ai
		cfg.Ai.fillMissing(d.Ai)
	}
	return cfg
}

// Save writes cfg to dir.
func (c AppConfig) Save(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create configuration directory: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encode configuration: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(dir, FileName), data, 0o644); err != nil {
		return fmt.Errorf("write configuration: %w", err)
	}
	return nil
}

// fillMissing gives the fields the file left out their defaults. It is
// called only for a section the file has: one it does not keeps the
// defaults whole (see fromFile).
func (b *BasicConfig) fillMissing(d BasicConfig) {
	if b.Appearance == "" {
		b.Appearance = d.Appearance
	}
	if b.Theme == "" {
		b.Theme = d.Theme
	}
	if b.Language == "" {
		b.Language = d.Language
	}
	if b.IconSize == 0 {
		b.IconSize = d.IconSize
	}
	b.TreeFont.fillMissing(d.TreeFont)
}

func (f *FontConfig) isEmpty() bool {
	return f.Family == "" && f.Size == 0 && f.LineHeight == 0 && f.Weight == ""
}

func (f *FontConfig) fillMissing(d FontConfig) {
	if strings.TrimSpace(f.Family) == "" {
		f.Family = d.Family
	}
	if f.Size == 0 {
		f.Size = d.Size
	}
	if f.LineHeight == 0 {
		f.LineHeight = d.LineHeight
	}
	if f.Weight == "" {
		f.Weight = d.Weight
	}
}

func (e *EditorConfig) fillMissing(d EditorConfig) {
	if e.AutoSaveDelayMs == 0 {
		e.AutoSaveDelayMs = d.AutoSaveDelayMs
	}
	if e.Indent.isEmpty() {
		e.Indent = d.Indent
	}
	e.Font.fillMissing(d.Font)
	if e.PaddingX == 0 {
		e.PaddingX = d.PaddingX
	}
	if e.Keymap == nil {
		e.Keymap = map[string]string{}
	}
	if e.DataFile.isEmpty() {
		e.DataFile = d.DataFile
	}
}

func (i IndentConfig) isEmpty() bool { return !i.UseTabs && i.TabSize == 0 }

func (d DataFileConfig) isEmpty() bool {
	return d.Indent == 0 && d.MaxFormatBytes == 0
}

func (p *PreviewConfig) fillMissing(d PreviewConfig) {
	p.Font.fillMissing(d.Font)
	if p.ImageDisplay == "" {
		p.ImageDisplay = d.ImageDisplay
	}
	if p.ImageWidthPercent == 0 {
		p.ImageWidthPercent = d.ImageWidthPercent
	}
	if p.LinkOpen == "" {
		p.LinkOpen = d.LinkOpen
	}
	if p.SyncMode == "" {
		p.SyncMode = d.SyncMode
	}
	if p.PaddingX == 0 {
		p.PaddingX = d.PaddingX
	}
	if p.PaddingY == 0 {
		p.PaddingY = d.PaddingY
	}
	if p.Density == "" {
		p.Density = d.Density
	}
}

// migrate moves the settings older versions wrote elsewhere. It runs once
// per load and the old fields are not written back.
func (c *AppConfig) migrate() {
	// A version that had no schemes kept the appearance in Theme: light,
	// dark or the machine's. It is the appearance's own field now.
	switch c.Basic.Theme {
	case AppearanceSystem, AppearanceLight, AppearanceDark:
		if c.Basic.Appearance == AppearanceSystem {
			c.Basic.Appearance = c.Basic.Theme
		}
		c.Basic.Theme = theme.Default
	}
	if c.Preview.LegacyCodeTheme != nil {
		c.Highlight.Theme = highlightThemeFrom(*c.Preview.LegacyCodeTheme)
		c.Preview.LegacyCodeTheme = nil
	}
	// "realtime" is what SyncRatio was called.
	if c.Preview.SyncMode == "realtime" {
		c.Preview.SyncMode = SyncRatio
	}
}

// highlightThemeFrom maps the name of a syntect theme of the old versions
// to one of the three the app ships.
func highlightThemeFrom(name string) string {
	switch {
	case strings.HasPrefix(name, "base16-ocean"):
		return HighlightOcean
	case strings.HasPrefix(name, "Solarized"):
		return HighlightSolarized
	default:
		return HighlightXink
	}
}

// Sanitize clamps every value to what the app can use, so a file edited by
// hand, or written by a version with other limits, never breaks it.
func (c *AppConfig) Sanitize() {
	b := &c.Basic
	if !oneOf(b.Appearance, AppearanceSystem, AppearanceLight,
		AppearanceDark) {
		b.Appearance = AppearanceSystem
	}
	b.Theme = theme.Normalize(b.Theme)
	if !oneOf(b.Language, LanguageSystem, LanguageEnglish,
		LanguageChinese) {
		b.Language = LanguageSystem
	}
	b.IconSize = clampU32(b.IconSize, 12, 48)
	b.TreeFont.sanitize(bodyFont())

	e := &c.Editor
	e.AutoSaveDelayMs = clampU64(e.AutoSaveDelayMs, 200, 60000)
	e.Indent.TabSize = clampU32(e.Indent.TabSize, 1, 16)
	e.PaddingX = clampU32(e.PaddingX, MinPadding, MaxPadding)
	e.MarginRight = minU32(e.MarginRight, MaxMarginRight)
	e.Font.sanitize(monoFont())
	// A key that is stored but would not be honoured is not stored.
	keymap.Sanitize(e.Keymap)
	e.DataFile.Indent = clampU64(e.DataFile.Indent, 1, 8)
	e.DataFile.MaxFormatBytes = minU64(e.DataFile.MaxFormatBytes, 20_000_000)

	p := &c.Preview
	p.Font.sanitize(bodyFont())
	if !oneOf(p.ImageDisplay, ImageFitWidth, ImageHidden) {
		p.ImageDisplay = ImageFitWidth
	}
	p.ImageWidthPercent = clampU32(p.ImageWidthPercent, 20, 100)
	if !oneOf(p.LinkOpen, LinkInApp, LinkExternal) {
		p.LinkOpen = LinkInApp
	}
	if !oneOf(p.SyncMode, SyncRatio, SyncAnchor, SyncOnStop) {
		p.SyncMode = SyncRatio
	}
	p.PaddingX = clampU32(p.PaddingX, MinPadding, MaxPadding)
	p.PaddingY = clampU32(p.PaddingY, MinPadding, MaxPadding)
	p.MaxWidth = minU32(p.MaxWidth, 4096)
	if !oneOf(p.Density, DensityCompact, DensityNormal, DensityRelaxed) {
		p.Density = DensityNormal
	}
	// A style of WeChat is a name the app is told by a folder of its own,
	// and not one this package knows: whether the name is one of them is
	// for the app to say, and all that is asked of the file is that it
	// names one plainly, and the folder it is read from as a folder is
	// written.
	p.Wechat = strings.TrimSpace(p.Wechat)
	p.WechatThemeDir = strings.TrimSpace(p.WechatThemeDir)

	if !oneOf(c.Highlight.Theme, HighlightXink, HighlightOcean,
		HighlightSolarized) {
		c.Highlight.Theme = HighlightXink
	}

	x := &c.Export
	if !oneOf(x.PageSize, ExportPageA4, ExportPageLetter) {
		x.PageSize = ExportPageA4
	}
	if !oneOf(x.Images, ExportImagesEmbed, ExportImagesCopy,
		ExportImagesNone) {
		x.Images = ExportImagesEmbed
	}

	c.Ai.Sanitize()
}

func (f *FontConfig) sanitize(d FontConfig) {
	if strings.TrimSpace(f.Family) == "" {
		f.Family = d.Family
	}
	f.Size = clampF32(f.Size, 8, 72)
	f.LineHeight = clampF32(f.LineHeight, 0.8, 3.0)
	if !oneOf(f.Weight, "regular", "medium", "semibold", "bold") {
		f.Weight = "regular"
	}
}

func oneOf(v string, allowed ...string) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}

func clampF32(v, lo, hi float32) float32 {
	switch {
	case v < lo:
		return lo
	case v > hi:
		return hi
	default:
		return v
	}
}

func clampU32(v, lo, hi uint32) uint32 {
	switch {
	case v < lo:
		return lo
	case v > hi:
		return hi
	default:
		return v
	}
}

func clampU64(v, lo, hi uint64) uint64 {
	switch {
	case v < lo:
		return lo
	case v > hi:
		return hi
	default:
		return v
	}
}

func minU64(v, hi uint64) uint64 {
	if v > hi {
		return hi
	}
	return v
}

func minU32(v, hi uint32) uint32 {
	if v > hi {
		return hi
	}
	return v
}
