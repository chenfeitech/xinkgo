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
