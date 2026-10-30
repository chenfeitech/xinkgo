package highlight

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// TestNormalize_unknownNameIsTheDefault checks that a name edited by hand
// still paints code.
func TestNormalize_unknownNameIsTheDefault(t *testing.T) {
	cases := map[string]string{
		Xink:      Xink,
		Ocean:     Ocean,
		Solarized: Solarized,
		"neon":    Xink,
		"":        Xink,
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestEveryPresetHasColors checks that no preset would leave a kind of
// token unpainted.
func TestEveryPresetHasColors(t *testing.T) {
	for _, name := range Presets {
		for i, dark := range []bool{false, true} {
			p := palettes[name][i]
			if p.keyword == "" || p.str == "" || p.comment == "" ||
				p.number == "" || p.function == "" {
				t.Errorf("%s dark=%v: a color of the first rank is "+
					"missing: %+v", name, dark, p)
			}
		}
	}
}

// TestEditorTheme_paintsTheTokens checks that the theme colors the kinds
// of tokens, and that the kind a preset leaves out keeps the color the
// window's theme gives it.
func TestEditorTheme_paintsTheTokens(t *testing.T) {
	theme := EditorTheme(Ocean, ui.DarkTheme())
	if theme.Tokens[editor.TokenKeyword].Color != ui.Hex("#b48ead") {
		t.Errorf("keyword is %v, want the ocean's #b48ead",
			theme.Tokens[editor.TokenKeyword].Color)
	}
	if theme.Tokens[editor.TokenString].Color != ui.Hex("#a3be8c") {
		t.Errorf("string is %v, want the ocean's #a3be8c",
			theme.Tokens[editor.TokenString].Color)
	}
	// Ocean names no color of its own for punctuation: it keeps the
	// one the editor's default theme gives it.
	plain := editor.DefaultTheme(ui.DarkTheme())
	if theme.Tokens[editor.TokenPunctuation].Color !=
		plain.Tokens[editor.TokenPunctuation].Color {
		t.Error("punctuation is colored, want the theme's own")
	}
}

// TestEditorTheme_followsTheAppearance checks the same preset in the
// light and in the dark: the comment of a theme is one of the colors
// that change most between the two.
func TestEditorTheme_followsTheAppearance(t *testing.T) {
	light := EditorTheme(Solarized, ui.LightTheme())
	dark := EditorTheme(Solarized, ui.DarkTheme())
	if light.Tokens[editor.TokenComment].Color ==
		dark.Tokens[editor.TokenComment].Color {
		t.Error("the comment is one color in both appearances")
	}
}

// TestEditorTheme_unknownPresetIsTheDefault checks the fallback.
func TestEditorTheme_unknownPresetIsTheDefault(t *testing.T) {
	got := EditorTheme("neon", ui.LightTheme())
	want := EditorTheme(Xink, ui.LightTheme())
	if got.Tokens[editor.TokenKeyword].Color !=
		want.Tokens[editor.TokenKeyword].Color {
		t.Error("an unknown preset was not painted as the default")
	}
}
