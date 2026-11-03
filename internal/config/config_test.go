package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/helightxu/xinkgo/internal/theme"
)

// load reads a configuration whose file has the given contents.
func load(t *testing.T, contents string) AppConfig {
	t.Helper()
	dir := t.TempDir()
	if contents != "" {
		if err := os.WriteFile(filepath.Join(dir, FileName),
			[]byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}

// TestLoad_missingFileGivesDefaults checks a first start.
func TestLoad_missingFileGivesDefaults(t *testing.T) {
	cfg := load(t, "")
	if cfg.Basic.Appearance != AppearanceSystem || !cfg.Basic.ShowAllFiles {
		t.Errorf("basic %+v, want the defaults", cfg.Basic)
	}
	if cfg.Editor.AutoSaveDelayMs != 300 || !cfg.Editor.SoftWrap {
		t.Errorf("editor %+v, want the defaults", cfg.Editor)
	}
	if cfg.Preview.SyncMode != SyncRatio || cfg.Preview.ImageDisplay != ImageFitWidth {
		t.Errorf("preview %+v, want the defaults", cfg.Preview)
	}
	if cfg.Highlight.Theme != HighlightXink {
		t.Errorf("highlight %q, want %q", cfg.Highlight.Theme, HighlightXink)
	}
}

// TestLoad_emptyObjectGivesDefaults checks that a file without the sections
// fills every one of them in, instead of leaving zeroes.
func TestLoad_emptyObjectGivesDefaults(t *testing.T) {
	cfg := load(t, `{}`)
	if cfg.Basic.IconSize != Default().Basic.IconSize {
		t.Errorf("icon size %d, want %d", cfg.Basic.IconSize,
			Default().Basic.IconSize)
	}
	if cfg.Editor.Font.Family != Default().Editor.Font.Family {
		t.Errorf("editor font %q, want %q", cfg.Editor.Font.Family,
			Default().Editor.Font.Family)
	}
	if cfg.Ai.TimeoutSecs != Default().Ai.TimeoutSecs {
		t.Errorf("ai timeout %d, want %d", cfg.Ai.TimeoutSecs,
			Default().Ai.TimeoutSecs)
	}
}

// TestLoad_unknownFieldsAreIgnored checks that a newer file, with fields
// this version does not have, still loads.
func TestLoad_unknownFieldsAreIgnored(t *testing.T) {
	cfg := load(t, `{"basic":{"appearance":"dark","something_new":42}}`)
	if cfg.Basic.Appearance != AppearanceDark {
		t.Errorf("appearance %q, want %q", cfg.Basic.Appearance,
			AppearanceDark)
	}
}

// TestLoad_anOldFileKeepsItsAppearance checks a file of a version that
// kept the appearance in Theme, where the schemes are now: light and
// dark are not names of a scheme, and are moved to the appearance.
func TestLoad_anOldFileKeepsItsAppearance(t *testing.T) {
	cfg := load(t, `{"basic":{"theme":"dark"}}`)
	if cfg.Basic.Appearance != AppearanceDark {
		t.Errorf("appearance %q, want %q", cfg.Basic.Appearance,
			AppearanceDark)
	}
	if cfg.Basic.Theme != theme.Default {
		t.Errorf("theme %q, want %q", cfg.Basic.Theme, theme.Default)
	}
}

// TestLoad_aSchemeIsKept checks that the scheme the user chose comes
// back, and is not taken for an appearance.
func TestLoad_aSchemeIsKept(t *testing.T) {
	cfg := load(t, `{"basic":{"appearance":"light","theme":"ocean"}}`)
	if cfg.Basic.Appearance != AppearanceLight ||
		cfg.Basic.Theme != theme.Ocean {
		t.Errorf("basic %+v, want light and ocean", cfg.Basic)
	}
}

// TestLoad_brokenFileGivesDefaultsAndError checks that a file the app
// cannot read is reported, and does not keep the app from starting.
func TestLoad_brokenFileGivesDefaultsAndError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("{"),
		0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err == nil {
		t.Error("Load accepted a file that is not JSON")
	}
	if cfg.Basic.Appearance != AppearanceSystem {
		t.Errorf("appearance %q, want the defaults", cfg.Basic.Appearance)
	}
}

// TestSaveLoad_roundTrip checks that what the app writes it reads back.
func TestSaveLoad_roundTrip(t *testing.T) {
	dir := t.TempDir()
	want := Default()
	want.Basic.Appearance = AppearanceDark
	want.Basic.Theme = theme.Ocean
	want.Editor.AutoSaveDelayMs = 1500
	want.Preview.Density = DensityRelaxed
	if err := want.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Basic.Appearance != AppearanceDark ||
		got.Basic.Theme != theme.Ocean ||
		got.Editor.AutoSaveDelayMs != 1500 ||
		got.Preview.Density != DensityRelaxed {
		t.Errorf("got basic=%+v delay=%d density=%q", got.Basic,
			got.Editor.AutoSaveDelayMs, got.Preview.Density)
	}
}

// TestSanitize_clampsOutOfRangeValues checks the limits of what a user can
// put in the file by hand.
func TestSanitize_clampsOutOfRangeValues(t *testing.T) {
	cfg := load(t, `{
		"basic":{"appearance":"twilight","theme":"neon","icon_size":999},
		"editor":{"auto_save_delay_ms":1,"indent":{"tab_size":99}},
		"preview":{"image_width_percent":5,"max_width":99999,"density":"airy"}
	}`)
	if cfg.Basic.Appearance != AppearanceSystem {
		t.Errorf("appearance %q, want %q", cfg.Basic.Appearance,
			AppearanceSystem)
	}
	if cfg.Basic.Theme != theme.Default {
		t.Errorf("theme %q, want %q", cfg.Basic.Theme, theme.Default)
	}
	if cfg.Basic.IconSize != 48 {
		t.Errorf("icon size %d, want 48", cfg.Basic.IconSize)
	}
	if cfg.Editor.AutoSaveDelayMs != 200 {
		t.Errorf("delay %d, want 200", cfg.Editor.AutoSaveDelayMs)
	}
	if cfg.Editor.Indent.TabSize != 16 {
		t.Errorf("tab size %d, want 16", cfg.Editor.Indent.TabSize)
	}
	if cfg.Preview.ImageWidthPercent != 20 {
		t.Errorf("image width %d, want 20", cfg.Preview.ImageWidthPercent)
	}
	if cfg.Preview.MaxWidth != 4096 {
		t.Errorf("max width %d, want 4096", cfg.Preview.MaxWidth)
	}
	if cfg.Preview.Density != DensityNormal {
		t.Errorf("density %q, want %q", cfg.Preview.Density, DensityNormal)
	}
}

// TestDefault_theTextOfAPaneStandsOffItsEdges checks the room a pane keeps
// beside what it shows at a first start: the same on both sides of the
// note, and the same on both sides of the preview, whose top and foot
// keep a room of their own.
func TestDefault_theTextOfAPaneStandsOffItsEdges(t *testing.T) {
	cfg := Default()
	if cfg.Editor.PaddingX != DefaultPaddingX {
		t.Errorf("the note's padding is %d, want %d", cfg.Editor.PaddingX,
			DefaultPaddingX)
	}
	if cfg.Preview.PaddingX != DefaultPaddingX {
		t.Errorf("the preview's padding is %d, want %d",
			cfg.Preview.PaddingX, DefaultPaddingX)
	}
	if cfg.Preview.PaddingY != DefaultPaddingY {
		t.Errorf("the preview's padding over its top and foot is %d, "+
			"want %d", cfg.Preview.PaddingY, DefaultPaddingY)
	}
}

// TestLoad_aFileThatNamesNoPaddingKeepsTheDefaultRoom checks that a
// configuration written before the padding was a thing to choose keeps the
// room a first start gives, and not none at all.
func TestLoad_aFileThatNamesNoPaddingKeepsTheDefaultRoom(t *testing.T) {
	cfg := load(t, `{"editor":{"soft_wrap":false},"preview":{"density":"compact"}}`)
	if cfg.Editor.PaddingX != DefaultPaddingX {
		t.Errorf("the note's padding is %d, want %d", cfg.Editor.PaddingX,
			DefaultPaddingX)
	}
	if cfg.Preview.PaddingX != DefaultPaddingX ||
		cfg.Preview.PaddingY != DefaultPaddingY {
		t.Errorf("the preview's padding is %d by %d, want %d by %d",
			cfg.Preview.PaddingX, cfg.Preview.PaddingY,
			DefaultPaddingX, DefaultPaddingY)
	}
}

// TestDefault_theNoteAsksForNoMarginOfItsOwn checks the margin the note
// keeps of its own: none at a first start, for the room its words keep
// from the right edge of the pane is the padding's to give, and the room
// the editor's own bar and the rail's buttons stand in is kept besides.
func TestDefault_theNoteAsksForNoMarginOfItsOwn(t *testing.T) {
	cfg := Default()
	if cfg.Editor.MarginRight != DefaultMarginRight {
		t.Errorf("the note's margin is %d, want %d", cfg.Editor.MarginRight,
			DefaultMarginRight)
	}
	if DefaultMarginRight != 0 {
		t.Errorf("a first start asks for a margin of %d, want none:"+
			" the padding is the room a note is given", DefaultMarginRight)
	}
}

// TestSanitize_theMarginStaysWithinItsRange checks the limit of the margin:
// a margin wider than the pane has room for is a pane of nothing but
// margin, and is not asked for.
func TestSanitize_theMarginStaysWithinItsRange(t *testing.T) {
	cfg := load(t, `{"editor":{"margin_right":9999}}`)
	if cfg.Editor.MarginRight != MaxMarginRight {
		t.Errorf("the note's margin is %d, want %d", cfg.Editor.MarginRight,
			MaxMarginRight)
	}
}

// TestSanitize_thePaddingStaysWithinItsRange checks the limits of the room
// a pane may keep: a pane whose text touches its edges is not one to read,
// and one of nothing but room is no pane at all.
func TestSanitize_thePaddingStaysWithinItsRange(t *testing.T) {
	cfg := load(t, `{"editor":{"padding_x":1},
		"preview":{"padding_x":999,"padding_y":999}}`)
	if cfg.Editor.PaddingX != MinPadding {
		t.Errorf("the note's padding is %d, want %d", cfg.Editor.PaddingX,
			MinPadding)
	}
	if cfg.Preview.PaddingX != MaxPadding {
		t.Errorf("the preview's padding is %d, want %d",
			cfg.Preview.PaddingX, MaxPadding)
	}
	if cfg.Preview.PaddingY != MaxPadding {
		t.Errorf("the preview's padding over its top and foot is %d, "+
			"want %d", cfg.Preview.PaddingY, MaxPadding)
	}
	// A nought is what a file that never named the room has, and is read
	// as one: it gives the room a first start gives, and not none at
	// all, as for every other number the configuration keeps.
	zero := load(t, `{"editor":{"padding_x":0},"preview":{"padding_y":0}}`)
	if zero.Editor.PaddingX != DefaultPaddingX {
		t.Errorf("the note's padding is %d, want %d", zero.Editor.PaddingX,
			DefaultPaddingX)
	}
	if zero.Preview.PaddingY != DefaultPaddingY {
		t.Errorf("the preview's padding over its top and foot is %d, "+
			"want %d", zero.Preview.PaddingY, DefaultPaddingY)
	}
}

// TestSaveLoad_thePaddingComesBack checks that the room a pane keeps is
// written out and read back, as a choice the user made.
func TestSaveLoad_thePaddingComesBack(t *testing.T) {
	dir := t.TempDir()
	want := Default()
	want.Editor.PaddingX = 28
	want.Preview.PaddingX, want.Preview.PaddingY = 40, 8
	if err := want.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Editor.PaddingX != 28 {
		t.Errorf("the note's padding is %d, want 28", got.Editor.PaddingX)
	}
	if got.Preview.PaddingX != 40 || got.Preview.PaddingY != 8 {
		t.Errorf("the preview's padding is %d by %d, want 40 by 8",
			got.Preview.PaddingX, got.Preview.PaddingY)
	}
}

// TestSanitize_keymapDropsUnusableEntries checks that a key the app would
// not honour is not kept, as a hand-edited file may well have one.
func TestSanitize_keymapDropsUnusableEntries(t *testing.T) {
	cfg := load(t, `{"editor":{"keymap":{"Save":" CMD-SHIFT-S ",
		"OpenSearch":"shift-","NotAnAction":"cmd-j"}}}`)
	if got := cfg.Editor.Keymap["Save"]; got != "cmd-shift-s" {
		t.Errorf("Save is %q, want cmd-shift-s", got)
	}
	if _, ok := cfg.Editor.Keymap["OpenSearch"]; ok {
		t.Error("a key that cannot be bound was kept")
	}
	if _, ok := cfg.Editor.Keymap["NotAnAction"]; ok {
		t.Error("an action the app does not have was kept")
	}
}

// TestSanitize_unknownEnumFallsBack checks the enumerations.
func TestSanitize_unknownEnumFallsBack(t *testing.T) {
	cfg := load(t, `{"preview":{"image_display":"tile","link_open":"nowhere",
		"sync_mode":"moon"},"highlight":{"theme":"neon"}}`)
	if cfg.Preview.ImageDisplay != ImageFitWidth ||
		cfg.Preview.LinkOpen != LinkInApp ||
		cfg.Preview.SyncMode != SyncRatio ||
		cfg.Highlight.Theme != HighlightXink {
		t.Errorf("got %+v / %q", cfg.Preview, cfg.Highlight.Theme)
	}
}

// TestDefaults_theNoteIsReadAsMarkdown checks the style a note is set out
// in for WeChat on a first start: none, which is the note read as
// Markdown reads it.
func TestDefaults_theNoteIsReadAsMarkdown(t *testing.T) {
	if got := Default().Preview.Wechat; got != "" {
		t.Errorf("wechat %q, want none", got)
	}
}

// TestDefaults_theStylesAreReadFromTheAppsOwnFolder checks a first start:
// no folder of styles is named, so the styles are read from the folder of
// the app's own.
func TestDefaults_theStylesAreReadFromTheAppsOwnFolder(t *testing.T) {
	if got := Default().Preview.WechatThemeDir; got != "" {
		t.Errorf("the folder of the styles is %q, want the app's own", got)
	}
}

// TestSanitize_theFolderOfTheStylesIsNamedPlainly checks the folder the
// styles are read from: written as a folder is written, and empty for a
// name of nothing but room.
func TestSanitize_theFolderOfTheStylesIsNamedPlainly(t *testing.T) {
	cfg := load(t, `{"preview":{"wechat_theme_dir":" /styles "}}`)
	if cfg.Preview.WechatThemeDir != "/styles" {
		t.Errorf("the folder is %q, want /styles", cfg.Preview.WechatThemeDir)
	}
	if cfg := load(t, `{"preview":{"wechat_theme_dir":"   "}}`); cfg.Preview.WechatThemeDir != "" {
		t.Errorf("the folder is %q, want the app's own",
			cfg.Preview.WechatThemeDir)
	}
}

// TestSanitize_theStyleOfWeChatIsNamedPlainly checks the name of a style:
// the app is told the styles by a folder of its own, so all that is asked
// of the file is that it name one without the room around it.
func TestSanitize_theStyleOfWeChatIsNamedPlainly(t *testing.T) {
	if cfg := load(t, `{"preview":{"wechat":" jade "}}`); cfg.Preview.Wechat != "jade" {
		t.Errorf("wechat %q, want jade", cfg.Preview.Wechat)
	}
	if cfg := load(t, `{"preview":{"wechat":"   "}}`); cfg.Preview.Wechat != "" {
		t.Errorf("wechat %q, want none", cfg.Preview.Wechat)
	}
}

// TestLoad_theLanguageOfAMachineIsFollowed checks a first start: no
// language is chosen, so the app speaks the one the machine reads.
func TestLoad_theLanguageOfAMachineIsFollowed(t *testing.T) {
	cfg := load(t, "")
	if cfg.Basic.Language != LanguageSystem {
		t.Errorf("language %q, want %q", cfg.Basic.Language,
			LanguageSystem)
	}
	if tag := cfg.Basic.LocaleTag(); tag != "" {
		t.Errorf("LocaleTag = %q, want it to follow the machine", tag)
	}
}

// TestSanitize_unknownLanguageFallsBack checks a language the app does not
// speak, as a file written by hand or by another version may have.
func TestSanitize_unknownLanguageFallsBack(t *testing.T) {
	cfg := load(t, `{"basic":{"language":"klingon"}}`)
	if cfg.Basic.Language != LanguageSystem {
		t.Errorf("language %q, want %q", cfg.Basic.Language,
			LanguageSystem)
	}
}

// TestLocaleTag_theLanguageMyGoIsTold checks what the configuration's
// language is to MyGo: a tag it knows, or "" to follow the machine.
func TestLocaleTag_theLanguageMyGoIsTold(t *testing.T) {
	cases := map[string]string{
		LanguageSystem:  "",
		LanguageEnglish: "en",
		LanguageChinese: "zh-Hans",
	}
	for lang, want := range cases {
		cfg := load(t, `{"basic":{"language":"`+lang+`"}}`)
		if got := cfg.Basic.LocaleTag(); got != want {
			t.Errorf("%q: tag %q, want %q", lang, got, want)
		}
	}
}

// TestSaveLoad_theLanguageComesBack checks that the language is kept, as
// the rest of the settings are.
func TestSaveLoad_theLanguageComesBack(t *testing.T) {
	dir := t.TempDir()
	cfg := Default()
	cfg.Basic.Language = LanguageChinese
	if err := cfg.Save(dir); err != nil {
		t.Fatal(err)
	}
	again, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if again.Basic.Language != LanguageChinese {
		t.Errorf("language %q, want %q", again.Basic.Language,
			LanguageChinese)
	}
}

// TestLoad_aFirstStartWritesItsNotesOutOnA4 checks what a note is
// written out as before the user has said otherwise: the sheet the
// machine's own printer takes, and the pictures in the page itself, so
// that it stands alone.
func TestLoad_aFirstStartWritesItsNotesOutOnA4(t *testing.T) {
	cfg := load(t, "")
	if cfg.Export.PageSize != ExportPageA4 {
		t.Errorf("page %q, want %q", cfg.Export.PageSize, ExportPageA4)
	}
	if cfg.Export.Images != ExportImagesEmbed {
		t.Errorf("images %q, want %q", cfg.Export.Images,
			ExportImagesEmbed)
	}
}

// TestSanitize_anExportTheAppCannotMakeFallsBack checks a sheet or a way
// with the pictures the app does not know, as a file written by hand may
// have.
func TestSanitize_anExportTheAppCannotMakeFallsBack(t *testing.T) {
	cfg := load(t, `{"export":{"page_size":"a3","images":"burn"}}`)
	if cfg.Export.PageSize != ExportPageA4 {
		t.Errorf("page %q, want %q", cfg.Export.PageSize, ExportPageA4)
	}
	if cfg.Export.Images != ExportImagesEmbed {
		t.Errorf("images %q, want %q", cfg.Export.Images,
			ExportImagesEmbed)
	}
}

// TestSaveLoad_theExportComesBack checks that the sheet and the way with
// the pictures are kept, as the rest of the settings are.
func TestSaveLoad_theExportComesBack(t *testing.T) {
	dir := t.TempDir()
	cfg := Default()
	cfg.Export.PageSize = ExportPageLetter
	cfg.Export.Images = ExportImagesCopy
	if err := cfg.Save(dir); err != nil {
		t.Fatal(err)
	}
	again, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if again.Export != cfg.Export {
		t.Errorf("export %+v, want %+v", again.Export, cfg.Export)
	}
}

// TestMigrate_syncModeAlias checks the name SyncRatio used to have.
func TestMigrate_syncModeAlias(t *testing.T) {
	if got := load(t, `{"preview":{"sync_mode":"realtime"}}`).
		Preview.SyncMode; got != SyncRatio {
		t.Errorf("sync mode %q, want %q", got, SyncRatio)
	}
}

// TestMigrate_legacyCodeTheme checks the code theme of the old versions,
// which moves to highlight.theme and is not written back.
func TestMigrate_legacyCodeTheme(t *testing.T) {
	cases := map[string]string{
		"base16-ocean.dark":  HighlightOcean,
		"base16-ocean.light": HighlightOcean,
		"Solarized (dark)":   HighlightSolarized,
		"InspiredGitHub":     HighlightXink,
	}
	for from, want := range cases {
		cfg := load(t, `{"preview":{"legacy_code_theme":`+
			mustQuote(from)+`},"highlight":{"theme":"ocean"}}`)
		if cfg.Highlight.Theme != want {
			t.Errorf("%q: theme %q, want %q", from, cfg.Highlight.Theme, want)
		}
		if cfg.Preview.LegacyCodeTheme != nil {
			t.Errorf("%q: the old field is still there", from)
		}
	}
}

// TestSave_doesNotWriteLegacyField checks that the migrated field never
// comes back into the file.
func TestSave_doesNotWriteLegacyField(t *testing.T) {
	dir := t.TempDir()
	cfg := Default()
	cfg.Preview.LegacyCodeTheme = nil
	if err := cfg.Save(dir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "legacy_code_theme") {
		t.Error("the old field was written back")
	}
}

// TestDefault_hasBuiltinActions checks the shortcuts a first start has.
func TestDefault_hasBuiltinActions(t *testing.T) {
	actions := Default().Ai.Actions
	if len(actions) != 6 {
		t.Errorf("%d actions, want 6", len(actions))
	}
	for _, a := range actions {
		if a.ID == "" || a.Prompt == "" || !a.Enable {
			t.Errorf("action %+v is not complete", a)
		}
	}
}

// TestSanitize_aiDropsUnusableEntries checks the AI settings: models without
// an id go, routes to models that are gone are cleared, and auto-apply
// stays off however the file sets it.
func TestSanitize_aiDropsUnusableEntries(t *testing.T) {
	cfg := load(t, `{"ai":{"auto_apply":true,
		"models":[{"id":" "},{"id":"m1","enabled":true},{"id":"m1"}],
		"default_model":"gone","fallback_model":"m1",
		"route":{"chat":"gone","action":"m1"}}}`)
	if cfg.Ai.AutoApply {
		t.Error("auto-apply stayed on")
	}
	if len(cfg.Ai.Models) != 1 || cfg.Ai.Models[0].ID != "m1" {
		t.Errorf("models %+v, want only m1", cfg.Ai.Models)
	}
	if cfg.Ai.DefaultModel != "" {
		t.Errorf("default model %q, want it cleared", cfg.Ai.DefaultModel)
	}
	if cfg.Ai.FallbackModel != "m1" {
		t.Errorf("fallback model %q, want m1", cfg.Ai.FallbackModel)
	}
	if cfg.Ai.Route.Chat != nil {
		t.Error("the chat route still points at a model that is gone")
	}
	if cfg.Ai.Route.Action == nil || *cfg.Ai.Route.Action != "m1" {
		t.Error("the action route was cleared although its model is there")
	}
}

// TestSanitize_aiActionsDeduplicated checks the limits on the shortcuts.
func TestSanitize_aiActionsDeduplicated(t *testing.T) {
	cfg := load(t, `{"ai":{"actions":[
		{"id":"a","name":"A","prompt":"p"},
		{"id":"a","name":"A again","prompt":"p"},
		{"id":"","name":"no id","prompt":"p"}]}}`)
	if len(cfg.Ai.Actions) != 1 || cfg.Ai.Actions[0].ID != "a" {
		t.Errorf("actions %+v, want only one a", cfg.Ai.Actions)
	}
}

// TestSanitize_aiParamsClamped checks the limits of a model's parameters.
func TestSanitize_aiParamsClamped(t *testing.T) {
	cfg := load(t, `{"ai":{"models":[{"id":"m","params":{"num_predict":1,
		"temperature":99,"num_ctx":4,"keep_alive":"5m"}}]}}`)
	p := cfg.Ai.Models[0].Params
	if p.NumPredict != 16 {
		t.Errorf("num_predict %d, want 16", p.NumPredict)
	}
	if p.Temperature != 0.7 {
		t.Errorf("temperature %v, want 0.7", p.Temperature)
	}
	if p.NumCtx == nil || *p.NumCtx != 512 {
		t.Errorf("num_ctx %v, want 512", p.NumCtx)
	}
	if p.KeepAlive != nil {
		t.Error("keep_alive of the default was kept as if it were set")
	}
}

// TestSave_writesIndentAndSections checks the file's shape, which a user
// may edit by hand.
func TestSave_writesIndentAndSections(t *testing.T) {
	dir := t.TempDir()
	if err := Default().Save(dir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "\n  \"basic\"") {
		t.Error("the file is not indented with two spaces")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	for _, section := range []string{"basic", "editor", "preview",
		"highlight", "ai"} {
		if _, ok := raw[section]; !ok {
			t.Errorf("the file has no %q section", section)
		}
	}
}

func mustQuote(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// TestModelEntry_cloneIsChangedWithoutTheOtherChanging checks the one
// thing a copy is for: it may be changed afterwards. A copy that shares
// its capabilities, its context or its keep-alive with the original is
// one that changes the original as well.
func TestModelEntry_cloneIsChangedWithoutTheOtherChanging(t *testing.T) {
	ctx := uint32(4096)
	alive := "5m"
	limit := uint32(8192)
	one := ModelEntry{
		ID:           "first",
		Label:        "first",
		Capabilities: []string{"tools"},
		ContextLimit: &limit,
		Params: AiModelParams{
			NumCtx:     &ctx,
			KeepAlive:  &alive,
			NumPredict: 512,
		},
	}
	two := one.Clone("second")
	if two.ID != "second" {
		t.Fatalf("id = %q, want the name it was given", two.ID)
	}
	if two.Label != "second" {
		t.Fatalf("label = %q, want it not to wear the other's name",
			two.Label)
	}
	two.Capabilities[0] = "vision"
	*two.Params.NumCtx = 128
	*two.Params.KeepAlive = "0"
	*two.ContextLimit = 1
	if one.Capabilities[0] != "tools" || *one.Params.NumCtx != 4096 ||
		*one.Params.KeepAlive != "5m" || *one.ContextLimit != 8192 {
		t.Fatalf("the one it came from changed too: %+v", one)
	}
	// What was not said stays as it was.
	if two.Params.NumPredict != 512 {
		t.Fatalf("num_predict = %d, want the same", two.Params.NumPredict)
	}
}

// TestModelEntry_cloneKeepsANameOfItsOwn checks that a label the user
// wrote is left alone: only one that echoed what the model is known by
// follows the new name.
func TestModelEntry_cloneKeepsANameOfItsOwn(t *testing.T) {
	one := ModelEntry{ID: "first", Label: "The one at home"}
	if got := one.Clone("second").Label; got != "The one at home" {
		t.Fatalf("label = %q, want the one the user wrote", got)
	}
}

// TestSanitize_modelsWithoutASourceAreToldApart checks that models that
// stood in the settings before the two kinds were told apart are put on
// the right side of the line: one that speaks to a local Ollama and names
// no address of its own is one of the machine's, and the rest are the
// user's own.
func TestSanitize_modelsWithoutASourceAreToldApart(t *testing.T) {
	cfg := Default()
	cfg.Ai.Models = []ModelEntry{
		{ID: "local", Kind: ModelOllama, Enabled: true},
		{ID: "far", Kind: ModelOllama, BaseURL: "http://elsewhere:11434",
			Enabled: true},
		{ID: "cloud", Kind: ModelOpenAICompatible, Enabled: true},
	}
	cfg.Sanitize()
	want := map[string]string{
		"local": ModelSourceOllama,
		"far":   ModelSourceCustom,
		"cloud": ModelSourceCustom,
	}
	if len(cfg.Ai.Models) != 3 {
		t.Fatalf("models = %d, want three", len(cfg.Ai.Models))
	}
	for _, m := range cfg.Ai.Models {
		if m.Source != want[m.ID] {
			t.Errorf("%q source = %q, want %q", m.ID, m.Source,
				want[m.ID])
		}
	}
	// What the user said is what is believed, and is not overwritten.
	cfg.Ai.Models[0].Source = ModelSourceCustom
	cfg.Sanitize()
	if cfg.Ai.Models[0].Source != ModelSourceCustom {
		t.Fatalf("source = %q, want the user's kept",
			cfg.Ai.Models[0].Source)
	}
}

// TestSanitize_aKeyIsEitherWrittenOutOrNamed checks the three places a
// key may come from: one written out here, one the machine keeps under a
// name, or one the keyring of the system keeps. There is no fourth, and
// a key that cannot be found is worse than one that was written out.
func TestSanitize_aKeyIsEitherWrittenOutOrNamed(t *testing.T) {
	cfg := Default()
	cfg.Ai.Models = []ModelEntry{
		{ID: "a", Enabled: true},
		{ID: "b", Enabled: true, ApiKeySource: KeyKeychain},
		{ID: "c", Enabled: true, ApiKeySource: KeyEnv},
	}
	cfg.Sanitize()
	want := map[string]string{
		"a": KeyPlain, "b": KeyKeychain, "c": KeyEnv,
	}
	for _, m := range cfg.Ai.Models {
		if m.ApiKeySource != want[m.ID] {
			t.Errorf("%q key from = %q, want %q", m.ID,
				m.ApiKeySource, want[m.ID])
		}
	}
}
