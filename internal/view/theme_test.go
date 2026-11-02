package view

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/theme"
)

// TestWorkspace_isDrawnInTheSchemeChosen checks that the scheme the user
// chose is the one the window is drawn in, and that it is drawn in it in
// whichever appearance the machine has: the two are chosen apart, and a
// window is where they meet.
func TestWorkspace_isDrawnInTheSchemeChosen(t *testing.T) {
	cfg := config.Default()
	cfg.Basic.Theme = theme.Ocean
	w := NewWorkspace(cfg)
	var got *ui.Theme
	tt := ui.NewTester(func(c *ui.Context) {
		w.View(c)
		got = c.Theme()
	}, 640, 400)
	if got != theme.Get(theme.Ocean).Of(false) {
		t.Fatalf("the window is not drawn in the scheme chosen")
	}
	tt.SetDark(true)
	if got != theme.Get(theme.Ocean).Of(true) {
		t.Fatalf("the window is not drawn in the dark of the scheme chosen")
	}
	if got.Accent != theme.Get(theme.Ocean).Of(true).Accent {
		t.Errorf("the accent is %v, want %v", got.Accent,
			theme.Get(theme.Ocean).Of(true).Accent)
	}
}

// TestWorkspace_aSchemeThatIsNotOneOfThemIsStillDrawn checks that a name
// a file edited by hand, or another version, put there leaves the window
// in colors of its own, and not in none.
func TestWorkspace_aSchemeThatIsNotOneOfThemIsStillDrawn(t *testing.T) {
	cfg := config.Default()
	cfg.Basic.Theme = "neon"
	w := NewWorkspace(cfg)
	var got *ui.Theme
	ui.NewTester(func(c *ui.Context) {
		w.View(c)
		got = c.Theme()
	}, 640, 400)
	if got != theme.Get(theme.Default).Of(false) {
		t.Errorf("the window is not drawn in the default scheme")
	}
}

// TestWorkspace_isPaintedInTheSchemeChosen checks that the colors are
// painted, and not only told to the widgets: the window's own ground is
// the scheme's, which is what the user sees.
func TestWorkspace_isPaintedInTheSchemeChosen(t *testing.T) {
	for _, dark := range []bool{false, true} {
		cfg := config.Default()
		cfg.Basic.Theme = theme.Plum
		w := NewWorkspace(cfg)
		tt := ui.NewTester(w.View, 640, 400)
		tt.SetDark(dark)
		want := theme.Get(theme.Plum).Of(dark).Background
		pixel := tt.Image().At(2, 2)
		r, g, b, a := pixel.RGBA()
		got := ui.RGB(uint8(r>>8), uint8(g>>8), uint8(b>>8))
		got.A = uint8(a >> 8)
		if got != want {
			t.Errorf("dark=%v: the window's ground is %v, want %v",
				dark, got, want)
		}
	}
}
