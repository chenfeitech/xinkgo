package theme

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestEveryScheme_keepsTheSizeOfTheText checks that a palette's colors
// do not take the size of the text with them: a theme made out of
// nothing leaves FontSize at 0, and a window whose text is asked for at
// no size draws its headings — the name of a group of the settings, say —
// too small to read, which is what they were.
func TestEveryScheme_keepsTheSizeOfTheText(t *testing.T) {
	want := ui.LightTheme().FontSize
	for _, s := range All() {
		for _, c := range []struct {
			what  string
			theme *ui.Theme
		}{{"light", s.Of(false)}, {"dark", s.Of(true)}} {
			if c.theme.FontSize != want {
				t.Errorf("scheme %q %s: the text is drawn at %v, "+
					"want %v, as MyGo has it",
					s.ID, c.what, c.theme.FontSize, want)
			}
			if c.theme.Spacing <= 0 {
				t.Errorf("scheme %q %s: the room between things is %v",
					s.ID, c.what, c.theme.Spacing)
			}
		}
	}
}

// TestEveryScheme_hasBothAppearances checks that a scheme added later is
// whole: two palettes that are not the same, and a code preset that is
// one of the three the app has.
func TestEveryScheme_hasBothAppearances(t *testing.T) {
	if len(All()) < 3 {
		t.Fatalf("%d schemes, want at least three", len(All()))
	}
	for _, s := range All() {
		if s.ID == "" || s.Title == "" || s.Note == "" {
			t.Errorf("scheme %q is not named in full: %+v", s.ID, s)
		}
		if s.Light == s.Dark {
			t.Errorf("scheme %q is the same in both appearances", s.ID)
		}
		switch s.Code {
		case CodeXink, CodeOcean, CodeSolarized:
		default:
			t.Errorf("scheme %q paints code in %q, want one of the three",
				s.ID, s.Code)
		}
		for _, c := range []struct {
			what  string
			theme *ui.Theme
		}{{"light", s.Of(false)}, {"dark", s.Of(true)}} {
			if c.theme.Dark != (c.what == "dark") {
				t.Errorf("scheme %q: the %s theme says Dark=%v",
					s.ID, c.what, c.theme.Dark)
			}
			if c.theme.Background == c.theme.Text {
				t.Errorf("scheme %q %s: text is the color of the ground",
					s.ID, c.what)
			}
			if c.theme.Selection.A == 0 || c.theme.Focus.A == 0 {
				t.Errorf("scheme %q %s: selection or focus is invisible",
					s.ID, c.what)
			}
		}
	}
}

// TestEveryScheme_isReadable checks the contrast of what is drawn on
// what: a scheme is of no use if its text cannot be read on its
// background, or the label of a button on the button's accent.
func TestEveryScheme_isReadable(t *testing.T) {
	for _, s := range All() {
		for _, dark := range []bool{false, true} {
			th := s.Of(dark)
			checks := []struct {
				what         string
				text, ground ui.Color
				least        float64
			}{
				{"text", th.Text, th.Background, 4.5},
				{"muted text", th.TextMuted, th.Background, 4.0},
				{"text on a control", th.Text, th.SurfaceHover, 4.0},
				{"a label on the accent", th.AccentText, th.Accent, 4.5},
			}
			for _, c := range checks {
				if got := contrast(c.text, c.ground); got < c.least {
					t.Errorf("scheme %q (dark=%v): %s stands at %.2f:1, "+
						"want at least %.1f:1", s.ID, dark, c.what,
						got, c.least)
				}
			}
		}
	}
}

// TestNamesAndTitlesAgree checks that a drop-down of the names offers
// every scheme, and nothing that is not one.
func TestNamesAndTitlesAgree(t *testing.T) {
	names := Names()
	if len(names) != len(All()) {
		t.Fatalf("%d names for %d schemes", len(names), len(All()))
	}
	for i, s := range All() {
		if names[i] != s.ID {
			t.Errorf("name %d is %q, want %q", i, names[i], s.ID)
		}
	}
}

// TestNormalize_unknownNameIsTheDefault checks that a name the app does
// not know, from a file edited by hand or kept by another version, draws
// the default scheme rather than nothing.
func TestNormalize_unknownNameIsTheDefault(t *testing.T) {
	if got := Normalize("neon"); got != Default {
		t.Errorf("normalize(neon) = %q, want %q", got, Default)
	}
	if got := Normalize(Ocean); got != Ocean {
		t.Errorf("normalize(ocean) = %q, want %q", got, Ocean)
	}
	if got := Get("neon").ID; got != Default {
		t.Errorf("get(neon) = %q, want %q", got, Default)
	}
	if got := Code("neon"); got != Code(Default) {
		t.Errorf("code(neon) = %q, want %q", got, Code(Default))
	}
}

// TestOf_isTheSameThemeEveryFrame checks that a view, which is drawn
// every frame, is given the same theme as the frame before: the runtime
// knows a theme by which one it is.
func TestOf_isTheSameThemeEveryFrame(t *testing.T) {
	for _, s := range All() {
		if s.Of(false) != s.Of(false) || s.Of(true) != s.Of(true) {
			t.Errorf("scheme %q gives a new theme every time", s.ID)
		}
	}
}

// TestApply_putsTheSchemeOnTheWindow checks that a window drawn in a
// scheme is drawn in its colors, in whichever appearance it has.
func TestApply_putsTheSchemeOnTheWindow(t *testing.T) {
	for _, dark := range []bool{false, true} {
		for _, s := range All() {
			var got *ui.Theme
			tt := ui.NewTester(func(c *ui.Context) {
				Apply(c, s.ID)
				got = c.Theme()
			}, 200, 120)
			tt.SetDark(dark)
			want := s.Of(dark)
			if got != want {
				t.Fatalf("scheme %q (dark=%v): the window is drawn in "+
					"another theme", s.ID, dark)
			}
			if got.Accent != want.Accent {
				t.Errorf("scheme %q (dark=%v): accent %v, want %v",
					s.ID, dark, got.Accent, want.Accent)
			}
		}
	}
}

// TestApply_anUnknownNameIsTheDefault checks that a window whose scheme
// is not one of them is still drawn in colors of its own.
func TestApply_anUnknownNameIsTheDefault(t *testing.T) {
	var got *ui.Theme
	ui.NewTester(func(c *ui.Context) {
		Apply(c, "neon")
		got = c.Theme()
	}, 200, 120)
	if got != Get(Default).Of(false) {
		t.Error("a scheme that is not one of them is not the default one")
	}
}

// contrast is how far apart two colors stand, as WCAG counts it: 1 is
// the same color, 21 black on white.
func contrast(a, b ui.Color) float64 {
	la, lb := luminance(a), luminance(b)
	hi, lo := la, lb
	if lb > la {
		hi, lo = lb, la
	}
	return (hi + 0.05) / (lo + 0.05)
}

// luminance is how bright a color is, as WCAG counts it: 0 is black and
// 1 white, with the sRGB curve taken out of it.
func luminance(c ui.Color) float64 {
	line := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*line(c.R) + 0.7152*line(c.G) + 0.0722*line(c.B)
}
