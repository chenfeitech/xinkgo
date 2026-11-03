package view

import (
	"strings"
	"testing"

	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/view/layout"
)

// TestPad_theNoteStandsAsFarFromOneSideAsTheOther checks what the padding
// of the note is for: its text stands the room the user chose from the
// left edge of its pane, and the very same room from the right one. The
// editor keeps a room of its own beside its lines, wider on one side than
// on the other, and that room is taken out of what the pane is given, so
// that it is the text, and not the editor, that is being measured.
//
// The tree and the preview stand down for the measurement, so that the
// note's pane is the window but for the rail's strip, which no pane
// writes in, and where it begins and ends is not in question.
func TestPad_theNoteStandsAsFarFromOneSideAsTheOther(t *testing.T) {
	const width = 900
	w, tt := zoomTester(t, width)
	w.Layout.Left, w.Layout.Right = 0, 0
	for _, pad := range []uint32{16, 32, config.MaxPadding} {
		w.Config.Editor.PaddingX = pad
		tt.Frame()
		ed, ok := tt.Find(editorLabel)
		if !ok {
			t.Fatalf("with a padding of %d, the note is not drawn", pad)
		}
		// Where the note's text stands: the room the pane was given,
		// and what the editor keeps of its own beside the lines. The
		// pane ends where the rail's strip begins, and not at the
		// window's own edge.
		for side, got := range map[string]float32{
			"left": ed.X + layout.EditorInsetLeft,
			"right": width - layout.RailStrip - (ed.X + ed.W) +
				layout.EditorInsetRight,
		} {
			if got != float32(pad) {
				t.Errorf("with a padding of %d, the note stands %.0f "+
					"from the %s edge of its pane", pad, got, side)
			}
		}
	}
}

// TestPad_aPaneGivenLessThanTheEditorKeepsKeepsNone checks the end of the
// range: a padding narrower than the room the editor keeps of its own is
// given to the pane as no room at all, for there is none to give — the
// text stands as near the edges as the editor itself puts it.
func TestPad_aPaneGivenLessThanTheEditorKeepsKeepsNone(t *testing.T) {
	const width = 900
	w, tt := zoomTester(t, width)
	w.Layout.Left, w.Layout.Right = 0, 0
	w.Config.Editor.PaddingX = config.MinPadding
	tt.Frame()
	ed, ok := tt.Find(editorLabel)
	if !ok {
		t.Fatal("the note is not drawn")
	}
	if ed.X != 0 {
		t.Errorf("the note's pane keeps %.0f beside the editor, want "+
			"none: the padding is narrower than what the editor keeps",
			ed.X)
	}
	// The other side of the pane is where the rail's strip begins: the
	// pane keeps nothing beside it of its own.
	if got := width - layout.RailStrip - (ed.X + ed.W); got != 0 {
		t.Errorf("the note's pane keeps %.0f on the other side, want none",
			got)
	}
}

// TestPad_theRoomTheNoteIsGivenIsTheRoomItKeeps checks that the padding is
// what decides where the note stands, and that it is given to the two
// sides alike: more of it brings the note in from both edges at once, and
// by as much on one as on the other.
func TestPad_theRoomTheNoteIsGivenIsTheRoomItKeeps(t *testing.T) {
	w, tt := zoomTester(t, 900)
	w.Config.Editor.PaddingX = 16
	tt.Frame()
	near, ok := tt.Find(editorLabel)
	if !ok {
		t.Fatal("the note is not drawn")
	}
	w.Config.Editor.PaddingX = 40
	tt.Frame()
	far, ok := tt.Find(editorLabel)
	if !ok {
		t.Fatal("the note is not drawn")
	}
	if got := far.X - near.X; got != 24 {
		t.Errorf("the note came in %v on the left, want the 24 of the "+
			"padding it was given more of", got)
	}
	if got := (near.X + near.W) - (far.X + far.W); got != 24 {
		t.Errorf("the note came in %v on the right, want 24 alike", got)
	}
}

// TestPad_whatIsReadStandsOffTheEdgesOfItsPane checks the preview's own:
// the room the user chose on either side of what is read, and the room
// over its top, which is a choice of its own. The preview is given the
// whole window for the measurement, as the note's pane is in the other
// test, so that where its pane begins is not in question.
func TestPad_whatIsReadStandsOffTheEdgesOfItsPane(t *testing.T) {
	w, tt := zoomTester(t, 900)
	w.setZoom(layout.ZoomPreview)
	for _, pad := range []uint32{config.MinPadding, 16, 40} {
		w.Config.Preview.PaddingX = pad
		tt.Frame()
		text, ok := tt.Find("head")
		if !ok {
			t.Fatalf("with a padding of %d, the note is not read", pad)
		}
		if got := text.X; got != float32(pad) {
			t.Errorf("with a padding of %d, what is read stands %.0f "+
				"from the side of its pane", pad, got)
		}
	}
	// The room over the top of it is a choice of its own: what is read
	// stands that much lower down. A preview given the window keeps a row
	// of its own above it, which is the way out of a pane the note's
	// toolbar is not drawn in, so it is the room that grows that is
	// measured, and not where it begins.
	w.Config.Preview.PaddingY = 12
	tt.Frame()
	near, ok := tt.Find("head")
	if !ok {
		t.Fatal("the note is not read")
	}
	w.Config.Preview.PaddingY = 36
	tt.Frame()
	far, ok := tt.Find("head")
	if !ok {
		t.Fatal("the note is not read")
	}
	if got := far.Y - near.Y; got != 24 {
		t.Errorf("what is read came down %v, want the 24 of the padding "+
			"it was given more of", got)
	}
}

// TestPad_whatIsReadKeepsItsPaddingSpreadOverTheNote checks the padding
// of a preview given the note's room as well as its own: what is read
// stands the room the user chose from the tree's line on one side and
// from the rail's strip on the other, and not hard against the window's
// own edge, which is where a box as wide as the pane puts it.
func TestPad_whatIsReadKeepsItsPaddingSpreadOverTheNote(t *testing.T) {
	const width = 1100
	w, tt := zoomTester(t, width)
	w.setZoom(layout.ZoomPreviewSpread)
	for _, pad := range []uint32{config.MinPadding, 16, 40} {
		w.Config.Preview.PaddingX = pad
		tt.Frame()
		text, ok := tt.Find("head")
		if !ok {
			t.Fatalf("with a padding of %d, the note is not read", pad)
		}
		want := float32(pad)
		if got := text.X - (w.Layout.Left + layout.DividerGrip); got != want {
			t.Errorf("with a padding of %d, what is read stands %.0f "+
				"from the line of the tree, want %d", pad, got, pad)
		}
		if got := float32(width) - layout.RailStrip - (text.X + text.W); got != want {
			t.Errorf("with a padding of %d, what is read stands %.0f "+
				"from the rail's strip, want %d", pad, got, pad)
		}
	}
}

// TestPad_thePaneEndsWhereTheRailBegins checks that no pane writes in the
// rail's strip: the buttons of the rail have no face of their own, so
// what is written there is lost, and the padding of the pane with it.
// Opening the drawer beside the rail covers what it covers, and takes
// none of the room the note is writing in.
func TestPad_thePaneEndsWhereTheRailBegins(t *testing.T) {
	const width = 1100
	w, tt := zoomTester(t, width)
	if err := tt.Click(locale.WidenNote()); err != nil {
		t.Fatalf("Click: %v", err)
	}
	tt.Advance(widenAnim.Duration)
	tt.Frame()
	ed, ok := tt.Find(editorLabel)
	if !ok {
		t.Fatal("the note is not drawn")
	}
	end := ed.X + ed.W
	if got := float32(width) - layout.RailStrip - end; got != w.geometry().EditorPadRight() {
		t.Errorf("the note's pane keeps %.0f beside the rail, want the "+
			"%.0f of the padding", got, w.geometry().EditorPadRight())
	}
	// A drawer opened beside the rail covers what it covers.
	w.mode.Drawer = drawerOutline
	tt.Frame()
	tt.Frame()
	open, ok := tt.Find(editorLabel)
	if !ok {
		t.Fatal("the note is not drawn with the drawer open")
	}
	if open.X+open.W != end {
		t.Errorf("the note ends at %v with the drawer open, want the %v "+
			"it ends at with it shut", open.X+open.W, end)
	}
}

// TestPad_aLineThatIsNotBrokenKeepsItsRoomBesideTheEditor checks the
// padding of a note whose lines the editor does not break: such a line is
// cut at the very edge of what the editor is given, and the editor draws
// its scroll bar over the last of it, so the pane keeps the room the user
// chose and the room of the bar besides, and no word is written where the
// bar will stand.
func TestPad_aLineThatIsNotBrokenKeepsItsRoomBesideTheEditor(t *testing.T) {
	w, tt := zoomTester(t, 1100)
	line, ok := tt.Find(cornerDivider)
	if !ok {
		t.Fatal("the line that parts the note from the preview is not drawn")
	}
	pane := line.X
	w.Config.Editor.SoftWrap = false
	for _, pad := range []uint32{16, 40} {
		w.Config.Editor.PaddingX = pad
		tt.Frame()
		ed, ok := tt.Find(editorLabel)
		if !ok {
			t.Fatalf("with a padding of %d, the note is not drawn", pad)
		}
		if got := pane - (ed.X + ed.W); got != float32(pad)+layout.EditorTrack {
			t.Errorf("with a padding of %d, the pane keeps %.0f beside "+
				"the editor, want the %d the user chose and the %.0f "+
				"of its scroll bar", pad, got, pad, layout.EditorTrack)
		}
	}
}

// TestPad_aPaneNarrowerThanTheNoteNeedsIsNeverMade checks the end of the
// range the other way: a panel given more room than the window has beside
// the note does not take the note's with it. The editor breaks a line no
// narrower than eight of its own characters, so a pane narrower than
// those and the rooms beside them writes its words over them and cuts
// them off at the edge of the pane, padding and all.
func TestPad_aPaneNarrowerThanTheNoteNeedsIsNeverMade(t *testing.T) {
	w, tt := zoomTester(t, 900)
	want := w.geometry().NoteMinWidth() - w.geometry().EditorPadLeft() - w.geometry().EditorPadRight()
	for _, name := range []string{
		"the preview given more than there is", "the window squeezed",
	} {
		switch name {
		case "the preview given more than there is":
			w.Layout.Right = 700
		case "the window squeezed":
			w.Layout.Right = 300
			tt.SetSize(420, 320)
		}
		tt.Frame()
		ed, ok := tt.Find(editorLabel)
		if !ok {
			t.Fatalf("with %s, the note is not drawn", name)
		}
		if ed.W < want {
			t.Errorf("with %s, the note is given %v, want the %v it needs:"+
				" narrower than that it is written over the edge of its"+
				" pane", name, ed.W, want)
		}
	}
}

// TestPad_aDragTakesNoMoreThanTheWindowHasBesideTheNote checks that a
// drag does not put the note in a pane it cannot be written in: what the
// panel is given by the drag is what the window has beside the note, so
// that letting go and taking hold again does not find the room already
// spent.
func TestPad_aDragTakesNoMoreThanTheWindowHasBesideTheNote(t *testing.T) {
	const width = 900
	w, tt := zoomTester(t, width)
	div, ok := tt.Find("divider-preview")
	if !ok {
		t.Fatal("there is no line to drag the preview by")
	}
	x, y := div.X+div.W/2, div.Y+80
	tt.Move(x, y)
	tt.Press(x, y)
	tt.Move(x-400, y) // the preview asked for more than the window has
	tt.Release(x-400, y)
	tt.Frame()
	ed, ok := tt.Find(editorLabel)
	if !ok {
		t.Fatal("the note is not drawn")
	}
	want := w.geometry().NoteMinWidth() - w.geometry().EditorPadLeft() - w.geometry().EditorPadRight()
	if ed.W < want {
		t.Errorf("the drag left the note %v, want the %v it needs", ed.W, want)
	}
	if got, room := w.Layout.Right,
		width-w.Layout.Left-w.geometry().NoteMinWidth()-2*layout.DividerGrip; got > room {
		t.Errorf("the drag gave the preview %v, want no more than the %v"+
			" beside the note", got, room)
	}
}

// TestPad_theWordsKeepTheirAirBesideTheEditorBar checks that the padding
// is air and not the room a scroll bar stands in. The editor draws its
// bar over the last of what it is given, four DIPs from a line it breaks,
// so a line is broken short of the bar's room by the air the words keep:
// without that, the bar is drawn against the last word however wide the
// padding is chosen.
func TestPad_theWordsKeepTheirAirBesideTheEditorBar(t *testing.T) {
	w, tt := zoomTester(t, 900)
	w.Config.Editor.PaddingX = 29
	n := w.Tabs.ActiveNote()
	n.Editor.Buffer.Set(strings.Repeat("M", 400))
	tt.Frame()
	ed, ok := tt.Find(editorLabel)
	if !ok {
		t.Fatal("the note is not drawn")
	}
	// The room the editor breaks a line at when it is left to itself:
	// what it is given, less what it keeps of its own beside the lines.
	natural := ed.W - layout.EditorInsetLeft - layout.EditorInsetRight
	broken := float32(n.Editor.WrapColumn) * layout.CharWidth(n.Editor.Font)
	if n.Editor.WrapColumn <= 0 || broken > natural {
		broken = natural
	}
	if got := natural - broken; got < layout.EditorBarAir {
		t.Errorf("the words end %v short of the room there is, want the %v"+
			" of air they keep beside the bar", got, layout.EditorBarAir)
	}
}

// TestPad_aPreviewNarrowerThanTheRailIsNoCoverForIt checks the shape the
// note passes through as it is stretched over the preview: the preview
// grows narrower and narrower, and while it is narrower than the rail's
// strip the strip hangs over the note as well as over the preview. The
// note keeps the strip's room all the same — its padding with it — and
// not only once the preview has stood down altogether.
func TestPad_aPreviewNarrowerThanTheRailIsNoCoverForIt(t *testing.T) {
	const width = 1100
	w, tt := zoomTester(t, width)
	// Narrower than the strip and still drawn: as the preview is for the
	// last of the stretch, before it stands down.
	w.Layout.Right = 1
	tt.Frame()
	ed, ok := tt.Find(editorLabel)
	if !ok {
		t.Fatal("the note is not drawn")
	}
	if end := ed.X + ed.W + w.geometry().EditorPadRight(); end > float32(width)-layout.RailStrip {
		t.Errorf("the note's pane ends at %v, want it to end at %v and no"+
			" further: the rail's strip hangs over what is past it",
			end, float32(width)-layout.RailStrip)
	}
}

// TestPad_theWordsKeepTheRailsRoomBesideThePadding checks the pane that
// ends at the window's own edge, as the note's does widened over the
// preview: the rail's buttons stand over that end of it, so the words
// keep their air beside the buttons on top of the padding — the padding
// alone was chosen for a pane with a pane beside it, and not the rail.
func TestPad_theWordsKeepTheRailsRoomBesideThePadding(t *testing.T) {
	w, tt := zoomTester(t, 1100)
	n := w.Tabs.ActiveNote()
	if err := tt.Click(locale.WidenNote()); err != nil {
		t.Fatalf("Click: %v", err)
	}
	tt.Advance(widenAnim.Duration)
	tt.Frame()
	ed, ok := tt.Find(editorLabel)
	if !ok {
		t.Fatal("the note is not drawn")
	}
	// Where the words would end were the line broken at the very room
	// there is, and where they do end at the column they are broken at.
	natural := ed.X + ed.W - layout.EditorInsetRight
	broken := ed.X + layout.EditorInsetLeft +
		float32(n.Editor.WrapColumn)*layout.CharWidth(n.Editor.Font)
	if got := natural - broken; got < layout.EditorBarAir+layout.EditorRailAir {
		t.Errorf("the words end %v short of the room there is, want the %v"+
			" of air beside the bar and the rail's buttons", got,
			layout.EditorBarAir+layout.EditorRailAir)
	}
}

// TestPad_theBarOfTheNoteIsPaintedNoColor checks that the editor's own
// scroll bar paints nothing over the note: the bar stands over the last
// of the room the editor is given, and what is written there is the
// note's, and not the bar's to cover. The bars of the window's own panes
// are left as they were.
func TestPad_theBarOfTheNoteIsPaintedNoColor(t *testing.T) {
	w, tt := zoomTester(t, 900)
	n := w.Tabs.ActiveNote()
	// A note long enough to scroll, scrolled past its first lines so that
	// the mark the bar shows of the caret is out of the way: what is left
	// standing in the bar's room is the bar itself.
	var b strings.Builder
	for i := 0; i < 600; i++ {
		b.WriteString("a line of the note\n")
	}
	n.Editor.Buffer.Set(b.String())
	tt.Frame()
	n.Editor.SetScrollY(4000)
	ed, ok := tt.Find(editorLabel)
	if !ok {
		t.Fatal("the note is not drawn")
	}
	tt.Move(ed.X+100, ed.Y+100) // the pointer over the note: the bar shows
	tt.Frame()
	img := tt.Image()
	// The last of the pane's height is where a bar that runs along the
	// foot of it stands, and is not this one's: the bar asked about runs
	// down the side of the note.
	for y := int(ed.Y); y < int(ed.Y+ed.H)-int(layout.EditorTrack); y++ {
		for x := int(ed.X+ed.W) - int(layout.EditorTrack); x < int(ed.X+ed.W); x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if (r>>8+g>>8+b>>8)/3 < 250 {
				t.Fatalf("the bar paints at %v,%v: what stands there is"+
					" written over", x, y)
			}
		}
	}
}

// TestPad_theMarginIsRoomTheWordsKeepBesidesThePadding checks the note's
// own margin: room its words keep from the right edge of the pane, over
// and above the padding, which is the same on either side. A wider margin
// brings the words in from the right, and leaves the left alone.
func TestPad_theMarginIsRoomTheWordsKeepBesidesThePadding(t *testing.T) {
	w, tt := zoomTester(t, 900)
	n := w.Tabs.ActiveNote()
	n.Editor.Buffer.Set(strings.Repeat("M", 400))
	w.Config.Editor.PaddingX = 29
	w.Config.Editor.MarginRight = 0
	tt.Frame()
	ed, ok := tt.Find(editorLabel)
	if !ok {
		t.Fatal("the note is not drawn")
	}
	near := ed.X + layout.EditorInsetLeft +
		float32(n.Editor.WrapColumn)*layout.CharWidth(n.Editor.Font)
	w.Config.Editor.MarginRight = 90
	tt.Frame()
	far, ok := tt.Find(editorLabel)
	if !ok {
		t.Fatal("the note is not drawn with a margin")
	}
	if far.X != ed.X {
		t.Errorf("the note begins at %v with the margin and %v without,"+
			" want it to begin in the same place", far.X, ed.X)
	}
	words := far.X + layout.EditorInsetLeft +
		float32(n.Editor.WrapColumn)*layout.CharWidth(n.Editor.Font)
	// A line is broken at a whole character, and the room the editor's
	// own bar stands in is kept whether the margin is asked for or not:
	// what the words come in by is the margin asked for, less that room,
	// and less what is left of a character besides.
	if got := near - words; got < 60 {
		t.Errorf("the margin brought the words in %v, want the 90 asked"+
			" for, less the %v the bar stands in and the rest of a"+
			" character", got, layout.EditorBarAir)
	}
}
