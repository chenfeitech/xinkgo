package layout

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/state"
)

// of is the geometry of a window of the shape named, as wide as told.
func of(zoom Zoom, stretch float32, drawer bool) Geometry {
	cfg := config.Default()
	return Of(&cfg, state.DefaultLayout(), Shape{
		Zoom:    zoom,
		Stretch: stretch,
		Drawer:  drawer,
	})
}

// TestGeometry_theNoteStandsAsFarFromOneSideAsTheOther checks what the
// padding is for: the note's text stands the room the user chose from the
// left edge of its pane, and the very same from the right one. The editor
// keeps a room of its own beside its lines, wider on one side than on the
// other, and that room is taken out of what the pane is given, so that it
// is the text, and not the editor, that is being measured.
func TestGeometry_theNoteStandsAsFarFromOneSideAsTheOther(t *testing.T) {
	for _, pad := range []uint32{16, 32, config.MaxPadding} {
		cfg := config.Default()
		cfg.Editor.PaddingX = pad
		g := Of(&cfg, state.DefaultLayout(), Shape{})
		if got := g.EditorPadLeft() + EditorInsetLeft; got != float32(pad) {
			t.Errorf("with a padding of %d, the note stands %.0f from the"+
				" left edge of its pane", pad, got)
		}
		if got := g.EditorPadRight() + EditorInsetRight; got != float32(pad) {
			t.Errorf("with a padding of %d, the note stands %.0f from the"+
				" right edge of its pane", pad, got)
		}
	}
}

// TestGeometry_aPaneGivenLessThanTheEditorKeepsKeepsNone checks the end of
// the range: a padding narrower than the room the editor keeps of its own
// is given to the pane as no room at all, for there is none to give.
func TestGeometry_aPaneGivenLessThanTheEditorKeepsKeepsNone(t *testing.T) {
	cfg := config.Default()
	cfg.Editor.PaddingX = config.MinPadding
	g := Of(&cfg, state.DefaultLayout(), Shape{})
	if got := g.EditorPadLeft(); got != 0 {
		t.Errorf("the pane keeps %.0f beside the editor, want none", got)
	}
	if got := g.EditorPadRight(); got != 0 {
		t.Errorf("the pane keeps %.0f on the other side, want none", got)
	}
}

// TestGeometry_theWordsKeepTheirAirBesideTheBarAndTheRail checks that the
// room the editor's own bar and the rail's buttons stand in is kept
// whether the note asks for a margin or not: at rest that is the bar's
// room, and widened over the preview — with the note's pane ending at the
// window's own edge — the rail's besides it.
func TestGeometry_theWordsKeepTheirAirBesideTheBarAndTheRail(t *testing.T) {
	if got := of(ZoomNone, 0, false).EditorMarginRight(); got < EditorBarAir {
		t.Errorf("the words keep %.0f beside the bar, want the %.0f of air",
			got, EditorBarAir)
	}
	widened := of(ZoomNone, 1, false)
	if !widened.NotePaneAtRail() {
		t.Error("widened over the preview, the note's pane does not end" +
			" at the window's edge")
	}
	if got := widened.EditorMarginRight(); got < EditorBarAir+EditorRailAir {
		t.Errorf("the words keep %.0f beside the rail's buttons, want the"+
			" %.0f of the bar and the rail besides", got,
			EditorBarAir+EditorRailAir)
	}
}

// TestGeometry_theRailIsClearedByThePaneItHangsOver checks what a row of
// the note's has to give up of the rail's room: nothing while a preview
// stands between it and the rail, and nothing but the drawer's own room
// when the note's pane is the one the rail hangs over, for its pane keeps
// the strip already.
func TestGeometry_theRailIsClearedByThePaneItHangsOver(t *testing.T) {
	if got := of(ZoomNone, 0, false).RailClear(); got != 0 {
		t.Errorf("a row gives up %.0f with a preview beside the note,"+
			" want nothing", got)
	}
	if got := of(ZoomNone, 1, false).RailClear(); got != 0 {
		t.Errorf("a row gives up %.0f widened over the preview, want"+
			" nothing", got)
	}
	// A drawer beside the rail covers what it covers: the room it takes
	// is what the row gives up, and no more.
	want := DrawerWidth + 2
	if got := of(ZoomNone, 1, true).RailClear(); got != want {
		t.Errorf("a row gives up %.0f with the drawer open, want the %.0f"+
			" of the drawer", got, want)
	}
}

// TestGeometry_theNoteKeepsTheRoomItNeeds checks the narrowest the note's
// pane may be made: the padding on either side, the rooms the editor keeps
// of its own, and eight characters of the note, which is as narrow as the
// editor breaks a line.
func TestGeometry_theNoteKeepsTheRoomItNeeds(t *testing.T) {
	g := of(ZoomNone, 0, false)
	chars := EditorMinChars * MonoCharRatio * g.Config.Editor.Font.Size
	want := g.EditorPadLeft() + g.EditorPadRight() +
		EditorInsetLeft + EditorInsetRight + chars
	if got := g.NoteMinWidth(); got != want {
		t.Errorf("the note's narrowest pane is %.0f, want %.0f", got, want)
	}
}

// TestGeometry_aLineIsBrokenShortOfTheRoomThereIs checks what a line is
// broken at: as many columns as the room the editor is given holds, less
// the air the words keep beside the bar. Without that, the padding the
// user chose is a room the bar is drawn in.
func TestGeometry_aLineIsBrokenShortOfTheRoomThereIs(t *testing.T) {
	const win = 900
	g := of(ZoomNone, 0, false)
	char := CharWidth(ui.Font{
		Family: g.Config.Editor.Font.Family,
		Size:   g.Config.Editor.Font.Size,
	})
	if char <= 0 {
		t.Fatal("a character of the note's font is measured as nothing")
	}
	cols := g.WrapColumn(win, char)
	room := g.EditorWidth(win) - g.EditorPadLeft() - g.EditorPadRight()
	if got := room - float32(cols)*char; got < g.EditorMarginRight() {
		t.Errorf("the words stand %.0f short of the room there is, want"+
			" the %.0f of air beside the bar and more", got,
			g.EditorMarginRight())
	}
	// And never narrower than eight characters, however little room the
	// pane is given.
	if got := g.WrapColumn(0, char); got < EditorMinChars {
		t.Errorf("a line is broken at %d columns in a pane of no room,"+
			" want no fewer than %d", got, EditorMinChars)
	}
}
