// Package theme holds the color schemes the interface is drawn in.
//
// A scheme is one set of colors for a light appearance and another for a
// dark one, and the code preset that goes with them: the user chooses a
// scheme, the machine (or the user) the appearance, and the window is
// drawn in the scheme's colors for whichever appearance it has. The
// names of the schemes are what the configuration keeps, so this is the
// one place a scheme is added: the settings window, and the validation
// of what a hand-edited file may hold, both read them from here.
package theme

import (
	"github.com/egoist/mygo/ui"
)

// The schemes the user may choose between, in the order the settings
// show them. Their names are what the configuration keeps.
const (
	// Xink is the neutral scheme: gray on white, gray on near-black,
	// with a blue accent. It is what the app has always looked like.
	Xink = "xink"
	// Ocean is cool: slate with a teal accent.
	Ocean = "ocean"
	// Moss is green: gray-green with a green accent.
	Moss = "moss"
	// Plum is violet: gray-mauve with a violet accent.
	Plum = "plum"
	// Sepia is warm: paper and ink with an amber accent, for reading
	// a long note.
	Sepia = "sepia"

	// Default is the scheme of a first start, and of a name that is
	// not one of these.
	Default = Xink
)

// The code presets a scheme paints code in, as internal/highlight names
// them, and as the configuration keeps them in Highlight.Theme.
const (
	CodeXink      = "xink"
	CodeOcean     = "ocean"
	CodeSolarized = "solarized"
)

// Palette is the colors of one scheme in one appearance. Focus and
// Selection are not in it: they are the accent, made faint, and are
// worked out from it, so that a scheme is one accent away from being
// whole.
type Palette struct {
	Background     ui.Color
	Surface        ui.Color
	SurfaceHover   ui.Color
	SurfacePressed ui.Color
	Border         ui.Color
	Text           ui.Color
	TextMuted      ui.Color
	Accent         ui.Color
	AccentHover    ui.Color
	AccentPressed  ui.Color
	AccentText     ui.Color
	Danger         ui.Color
	Warning        ui.Color
	Success        ui.Color
	Scrollbar      ui.Color
}

// build is the palette as one of MyGo's themes, in the appearance dark
// asks for.
func (p Palette) build(dark bool) *ui.Theme {
	t := &ui.Theme{
		Dark:           dark,
		Background:     p.Background,
		Surface:        p.Surface,
		SurfaceHover:   p.SurfaceHover,
		SurfacePressed: p.SurfacePressed,
		Border:         p.Border,
		Text:           p.Text,
		TextMuted:      p.TextMuted,
		Accent:         p.Accent,
		AccentHover:    p.AccentHover,
		AccentPressed:  p.AccentPressed,
		AccentText:     p.AccentText,
		Danger:         p.Danger,
		Warning:        p.Warning,
		Success:        p.Success,
		Scrollbar:      p.Scrollbar,
		Radius:         6,
		Spacing:        4,
	}
	// Selected text and the focus ring are the accent over what is
	// behind it, and text under them must still be read: less of the
	// accent in a dark appearance, where the accent is brighter.
	if dark {
		t.Selection, t.Focus = p.Accent.Alpha(0.4), p.Accent.Alpha(0.6)
	} else {
		t.Selection, t.Focus = p.Accent.Alpha(0.25), p.Accent.Alpha(0.55)
	}
	return t
}

// Scheme is one set of colors the user may choose: a title and a line
// about it for the settings, the colors of a light and of a dark
// appearance, and the code preset that goes with them.
type Scheme struct {
	ID    string
	Title string
	Note  string
	Light Palette
	Dark  Palette
	Code  string
}

// schemes is every scheme, in the order the settings show them.
var schemes = []Scheme{
	{
		ID: Xink, Title: "Xink", Note: "Neutral gray, blue accent",
		Code: CodeXink,
		Light: Palette{
			Background:     ui.Hex("#ffffff"),
			Surface:        ui.Hex("#f4f4f5"),
			SurfaceHover:   ui.Hex("#e9e9ec"),
			SurfacePressed: ui.Hex("#dddde1"),
			Border:         ui.Hex("#d9d9de"),
			Text:           ui.Hex("#18181b"),
			TextMuted:      ui.Hex("#71717a"),
			Accent:         ui.Hex("#2563eb"),
			AccentHover:    ui.Hex("#1d4ed8"),
			AccentPressed:  ui.Hex("#1e40af"),
			AccentText:     ui.Hex("#ffffff"),
			Danger:         ui.Hex("#dc2626"),
			Warning:        ui.Hex("#d97706"),
			Success:        ui.Hex("#16a34a"),
			Scrollbar:      ui.Hex("#00000052"),
		},
		Dark: Palette{
			Background:     ui.Hex("#18181b"),
			Surface:        ui.Hex("#27272a"),
			SurfaceHover:   ui.Hex("#323236"),
			SurfacePressed: ui.Hex("#3c3c41"),
			Border:         ui.Hex("#3f3f46"),
			Text:           ui.Hex("#f4f4f5"),
			TextMuted:      ui.Hex("#a1a1aa"),
			// A blue a shade deeper than the one a light appearance
			// has: a label on it has to be read, and white on a
			// brighter blue cannot be.
			Accent:        ui.Hex("#2f6ae0"),
			AccentHover:   ui.Hex("#60a5fa"),
			AccentPressed: ui.Hex("#2563eb"),
			AccentText:    ui.Hex("#ffffff"),
			Danger:        ui.Hex("#ef4444"),
			Warning:       ui.Hex("#f59e0b"),
			Success:       ui.Hex("#22c55e"),
			Scrollbar:     ui.Hex("#ffffff59"),
		},
	},
	{
		ID: Ocean, Title: "Ocean", Note: "Cool slate, teal accent",
		Code: CodeOcean,
		Light: Palette{
			Background:     ui.Hex("#fbfdfe"),
			Surface:        ui.Hex("#eff5f9"),
			SurfaceHover:   ui.Hex("#e3edf4"),
			SurfacePressed: ui.Hex("#d6e5ee"),
			Border:         ui.Hex("#cbdce7"),
			Text:           ui.Hex("#102029"),
			TextMuted:      ui.Hex("#5a7484"),
			Accent:         ui.Hex("#0f766e"),
			AccentHover:    ui.Hex("#0c5f58"),
			AccentPressed:  ui.Hex("#115e56"),
			AccentText:     ui.Hex("#ffffff"),
			Danger:         ui.Hex("#b91c1c"),
			Warning:        ui.Hex("#b45309"),
			Success:        ui.Hex("#15803d"),
			Scrollbar:      ui.Hex("#1020294d"),
		},
		Dark: Palette{
			Background:     ui.Hex("#0b1620"),
			Surface:        ui.Hex("#142430"),
			SurfaceHover:   ui.Hex("#1b3040"),
			SurfacePressed: ui.Hex("#223c4e"),
			Border:         ui.Hex("#23404f"),
			Text:           ui.Hex("#e6f0f6"),
			TextMuted:      ui.Hex("#8fa9b8"),
			Accent:         ui.Hex("#2dd4bf"),
			AccentHover:    ui.Hex("#5eead4"),
			AccentPressed:  ui.Hex("#14b8a6"),
			AccentText:     ui.Hex("#04231f"),
			Danger:         ui.Hex("#f87171"),
			Warning:        ui.Hex("#fbbf24"),
			Success:        ui.Hex("#34d399"),
			Scrollbar:      ui.Hex("#ffffff4d"),
		},
	},
	{
		ID: Moss, Title: "Moss", Note: "Green gray, green accent",
		Code: CodeSolarized,
		Light: Palette{
			Background:     ui.Hex("#fdfff9"),
			Surface:        ui.Hex("#f1f6ee"),
			SurfaceHover:   ui.Hex("#e4edde"),
			SurfacePressed: ui.Hex("#d7e4cf"),
			Border:         ui.Hex("#d2e2c7"),
			Text:           ui.Hex("#16211a"),
			TextMuted:      ui.Hex("#5f7360"),
			Accent:         ui.Hex("#4d7c0f"),
			AccentHover:    ui.Hex("#3f6212"),
			AccentPressed:  ui.Hex("#365314"),
			AccentText:     ui.Hex("#ffffff"),
			Danger:         ui.Hex("#b91c1c"),
			Warning:        ui.Hex("#a16207"),
			Success:        ui.Hex("#15803d"),
			Scrollbar:      ui.Hex("#16211a4d"),
		},
		Dark: Palette{
			Background:     ui.Hex("#111612"),
			Surface:        ui.Hex("#1b241d"),
			SurfaceHover:   ui.Hex("#232f26"),
			SurfacePressed: ui.Hex("#2b3a30"),
			Border:         ui.Hex("#2c3a30"),
			Text:           ui.Hex("#e8f0e9"),
			TextMuted:      ui.Hex("#9db3a1"),
			Accent:         ui.Hex("#84cc16"),
			AccentHover:    ui.Hex("#a3e635"),
			AccentPressed:  ui.Hex("#65a30d"),
			AccentText:     ui.Hex("#14210a"),
			Danger:         ui.Hex("#f87171"),
			Warning:        ui.Hex("#fbbf24"),
			Success:        ui.Hex("#4ade80"),
			Scrollbar:      ui.Hex("#ffffff4d"),
		},
	},
	{
		ID: Plum, Title: "Plum", Note: "Mauve gray, violet accent",
		Code: CodeOcean,
		Light: Palette{
			Background:     ui.Hex("#fffefe"),
			Surface:        ui.Hex("#f6f3f8"),
			SurfaceHover:   ui.Hex("#ece6f2"),
			SurfacePressed: ui.Hex("#e1d9ea"),
			Border:         ui.Hex("#dfd7e7"),
			Text:           ui.Hex("#1b1620"),
			TextMuted:      ui.Hex("#6f6478"),
			Accent:         ui.Hex("#7c3aed"),
			AccentHover:    ui.Hex("#6d28d9"),
			AccentPressed:  ui.Hex("#5b21b6"),
			AccentText:     ui.Hex("#ffffff"),
			Danger:         ui.Hex("#b91c1c"),
			Warning:        ui.Hex("#b45309"),
			Success:        ui.Hex("#15803d"),
			Scrollbar:      ui.Hex("#1b16204d"),
		},
		Dark: Palette{
			Background:     ui.Hex("#16121a"),
			Surface:        ui.Hex("#241d2a"),
			SurfaceHover:   ui.Hex("#2e2636"),
			SurfacePressed: ui.Hex("#382f42"),
			Border:         ui.Hex("#3a3145"),
			Text:           ui.Hex("#f0ebf5"),
			TextMuted:      ui.Hex("#a99cb4"),
			Accent:         ui.Hex("#a78bfa"),
			AccentHover:    ui.Hex("#c4b5fd"),
			AccentPressed:  ui.Hex("#8b5cf6"),
			AccentText:     ui.Hex("#1a1026"),
			Danger:         ui.Hex("#f87171"),
			Warning:        ui.Hex("#fbbf24"),
			Success:        ui.Hex("#4ade80"),
			Scrollbar:      ui.Hex("#ffffff4d"),
		},
	},
	{
		ID: Sepia, Title: "Sepia", Note: "Warm paper, amber accent",
		Code: CodeSolarized,
		Light: Palette{
			Background:     ui.Hex("#fdf8f1"),
			Surface:        ui.Hex("#f5ecdf"),
			SurfaceHover:   ui.Hex("#eee2d1"),
			SurfacePressed: ui.Hex("#e6d7c2"),
			Border:         ui.Hex("#e3d5c1"),
			Text:           ui.Hex("#2b2119"),
			TextMuted:      ui.Hex("#7a6a58"),
			Accent:         ui.Hex("#b45309"),
			AccentHover:    ui.Hex("#92400e"),
			AccentPressed:  ui.Hex("#78350f"),
			AccentText:     ui.Hex("#ffffff"),
			Danger:         ui.Hex("#a83232"),
			Warning:        ui.Hex("#a16207"),
			Success:        ui.Hex("#4d7c0f"),
			Scrollbar:      ui.Hex("#2b21194d"),
		},
		Dark: Palette{
			Background:     ui.Hex("#1c1712"),
			Surface:        ui.Hex("#262019"),
			SurfaceHover:   ui.Hex("#2f2820"),
			SurfacePressed: ui.Hex("#383027"),
			Border:         ui.Hex("#3a3229"),
			Text:           ui.Hex("#f3e9dc"),
			TextMuted:      ui.Hex("#b0a08c"),
			Accent:         ui.Hex("#f59e0b"),
			AccentHover:    ui.Hex("#fbbf24"),
			AccentPressed:  ui.Hex("#d97706"),
			AccentText:     ui.Hex("#241a08"),
			Danger:         ui.Hex("#f87171"),
			Warning:        ui.Hex("#eab308"),
			Success:        ui.Hex("#a3e635"),
			Scrollbar:      ui.Hex("#ffffff4d"),
		},
	},
}

// built is every scheme's two themes, made once: a view is drawn every
// frame, and the theme it is given is one the runtime may know again.
var built = func() map[string][2]*ui.Theme {
	m := make(map[string][2]*ui.Theme, len(schemes))
	for _, s := range schemes {
		m[s.ID] = [2]*ui.Theme{s.Light.build(false), s.Dark.build(true)}
	}
	return m
}()

// All is every scheme, in the order the settings show them.
func All() []Scheme {
	return append([]Scheme(nil), schemes...)
}

// Names is the name of every scheme, in the order the settings show
// them: what a drop-down of them is given.
func Names() []string {
	names := make([]string, 0, len(schemes))
	for _, s := range schemes {
		names = append(names, s.ID)
	}
	return names
}

// Normalize is the scheme a name stands for: one that is not one of them
// is the default, so that a name edited by hand, or kept by a version
// that had others, never leaves the window uncolored.
func Normalize(name string) string {
	if _, ok := built[name]; ok {
		return name
	}
	return Default
}

// Get is the scheme of that name, or the default one.
func Get(name string) Scheme {
	for _, s := range schemes {
		if s.ID == name {
			return s
		}
	}
	return Get(Default)
}

// Of is the scheme's theme for the appearance dark asks for.
func (s Scheme) Of(dark bool) *ui.Theme {
	both := built[Normalize(s.ID)]
	if dark {
		return both[1]
	}
	return both[0]
}

// Code is the code preset of the scheme of that name.
func Code(name string) string {
	return Get(name).Code
}

// Apply paints the window in the scheme of that name, in the appearance
// the window has: light or dark as the machine, or as the user, has it.
// It is called once, at the top of a view, and before anything is drawn:
// the appearance is read off the theme the window came with, and is
// known no more after this has put the scheme's on.
func Apply(c *ui.Context, name string) {
	c.SetTheme(Get(name).Of(c.Theme().Dark))
}
