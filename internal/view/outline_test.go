package view

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/locale"
)

// railOf is where the rail's buttons stand in a window of the width given,
// with the preview as wide as the preview says, and the drawer named open,
// or none of them when it is named "".
func railOf(t *testing.T, width, preview float32, drawer string) ui.Rect {
	t.Helper()
	w, _ := workspaceShown(t)
	w.Layout.Right = preview
	w.drawer = drawer
	tt := ui.NewTester(w.View, int(width), 600)
	r, ok := tt.Find(locale.DrawerOutlineLabel())
	if !ok {
		t.Fatalf("no rail button in the window")
	}
	return r
}

// TestRailView_theRailStandsAtTheWindowEdge checks where the buttons hang:
// at the right edge of the window, and not at the right end of the note,
// which is where the preview begins.
func TestRailView_theRailStandsAtTheWindowEdge(t *testing.T) {
	const width, preview = float32(1000), float32(320)
	r := railOf(t, width, preview, "")
	// The buttons are the window's, so they stand at its edge however
	// wide the panels are: 320 in from it is where the note ends.
	if end := r.X + r.W; end < width-40 || end > width {
		t.Fatalf("the rail ends at %v, want it at the window's own edge %v",
			end, width)
	}
}

// TestRailView_theRailStaysPutWhenThePreviewIsShut checks that shutting the
// preview, which widens the note, does not move the buttons: they are not
// the note's to move.
func TestRailView_theRailStaysPutWhenThePreviewIsShut(t *testing.T) {
	const width = float32(1000)
	open := railOf(t, width, 320, "")
	shut := railOf(t, width, 0, "")
	if open.X != shut.X {
		t.Fatalf("the rail moved from %v to %v when the preview was shut",
			shut.X, open.X)
	}
}

// drawerStrip is how many of the pixels of a thin strip inside where the
// drawer stands are not the face of the window: the drawer's own face,
// once it is open, hides the note behind it, and the strip is then all
// one color, while the note behind it is not.
func drawerStrip(t *testing.T, open bool) int {
	t.Helper()
	const width, preview, height = float32(1000), float32(320), 600
	w, _ := workspaceShown(t)
	w.Layout.Right = preview
	// A note long enough that the preview fills the panel with lines of
	// text, so that there is something behind the drawer to hide.
	w.Tabs.ActiveNote().Editor.Buffer.Set("# a heading\n\n" +
		strings.Repeat("word ", 3000))
	if open {
		w.drawer = drawerOutline
	}
	tt := ui.NewTester(w.View, int(width), height)
	tt.Frame()
	tt.Frame()
	img := tt.Image()
	face := ui.LightTheme().Background
	rail, ok := tt.Find(locale.DrawerOutlineLabel())
	if !ok {
		t.Fatalf("no rail button in the window")
	}
	// The strip stands well inside the drawer: a hundred DIPs in from the
	// rail, where nothing of what it says reaches, as what it says reads
	// from the left, and over the middle of the height it was given
	// beside the rail, which is where the note behind it has lines of
	// text whatever the pane keeps at the top of its own.
	other := 0
	for y := height/2 - 30; y <= height/2+30; y++ {
		r, g, b, _ := img.At(int(rail.X)-100, y).RGBA()
		if int(r>>8) != int(face.R) || int(g>>8) != int(face.G) ||
			int(b>>8) != int(face.B) {
			other++
		}
	}
	return other
}

// TestDrawerPanel_theDrawerStandsOnItsOwnFace checks that a drawer open
// over the note is read on its own face, and not on the note it covers.
func TestDrawerPanel_theDrawerStandsOnItsOwnFace(t *testing.T) {
	if behind := drawerStrip(t, false); behind == 0 {
		t.Fatalf("the note behind the drawer shows nothing, so the test " +
			"would pass with a drawer of no face at all")
	}
	if over := drawerStrip(t, true); over != 0 {
		t.Errorf("%d of the drawer's pixels are the note behind it, "+
			"want none of them", over)
	}
}

// TestRailView_theButtonsKeepToTheTop checks that a drawer as tall as the
// window does not take the buttons down with it: they hang from the top
// of the rail, and stand there whether a drawer is open or not.
func TestRailView_theButtonsKeepToTheTop(t *testing.T) {
	const width, preview = float32(1000), float32(320)
	shut := railOf(t, width, preview, "")
	open := railOf(t, width, preview, drawerOutline)
	if shut.Y != open.Y {
		t.Errorf("the buttons stand at %v with the drawer shut and %v "+
			"with it open, want them not to move", shut.Y, open.Y)
	}
	if shut.Y != drawerTop {
		t.Errorf("the buttons stand at %v, want them at the top of the "+
			"rail, %v", shut.Y, drawerTop)
	}
}

// drawerEdge is the top and the bottom line of the window that the
// drawer's own edge is drawn over, in a window of the height given: the
// note beside it says nothing, so that nothing but the drawer paints
// there.
func drawerEdge(t *testing.T, height int) (top, bottom int) {
	t.Helper()
	const width = 1000
	w, _ := workspaceShown(t)
	w.Layout.Right = 320
	w.Tabs.ActiveNote().Editor.Buffer.Set("")
	w.drawer = drawerOutline
	tt := ui.NewTester(w.View, width, height)
	tt.Frame()
	tt.Frame()
	img := tt.Image()
	border := ui.LightTheme().Border
	// The drawer stands its own width and the rail's in from the right
	// edge, as the rail stands railGap in from it.
	x := width - int(railWidth) - int(drawerWidth) - int(railGap)
	top, bottom = -1, -1
	for y := 0; y < height; y++ {
		r, g, b, _ := img.At(x, y).RGBA()
		if near(int(r>>8), int(border.R)) && near(int(g>>8), int(border.G)) &&
			near(int(b>>8), int(border.B)) {
			if top < 0 {
				top = y
			}
			bottom = y
		}
	}
	if top < 0 {
		t.Fatalf("no drawer was drawn in the window")
	}
	return top, bottom
}

// near is whether a channel of the pixel read is the channel of the
// color, as near as a line drawn over the window's own face is.
func near(pixel, color int) bool {
	d := pixel - color
	return d < 12 && d > -12
}

// TestDrawerPanel_theDrawerIsAsTallAsThePanels checks what the drawer
// was opened for: it is not a card of a height of its own, but a panel
// as tall as the note and the preview it stands over, and it grows and
// shrinks with the window.
func TestDrawerPanel_theDrawerIsAsTallAsThePanels(t *testing.T) {
	short, shortFoot := drawerEdge(t, 600)
	tall, tallFoot := drawerEdge(t, 900)
	if short < int(drawerTop) || short > int(drawerTop)+12 {
		t.Errorf("the drawer starts at %d, want it at the top of the "+
			"rail, %v", short, drawerTop)
	}
	if tall != short {
		t.Errorf("the drawer starts at %d in a tall window and %d in a "+
			"short one, want it to start at the same place", tall, short)
	}
	// A window 300 deeper gives the drawer 300 more: a drawer of a
	// height of its own would give it none.
	if grew := tallFoot - shortFoot; grew < 250 {
		t.Errorf("the drawer ends %d lower in a window 300 deeper, want "+
			"it to end about 300 lower", grew)
	}
}

// TestDrawerPanel_theDrawerStandsLevelWithThePreview checks the line the
// drawer was asked to stand on: the top of the preview beside it, and not
// somewhere down the window.
func TestDrawerPanel_theDrawerStandsLevelWithThePreview(t *testing.T) {
	const width, height = 1000, 600
	w, _ := workspaceShown(t)
	w.Layout.Right = 320
	w.Tabs.ActiveNote().Editor.Buffer.Set("# a heading\n")
	// The drawer shut: the top of the preview's pane, which is where the
	// button of its own corner stands, is the line to stand level with.
	// What is read in the pane begins under that row, and not under the
	// drawer, but the pane itself begins there and nowhere else.
	tt := ui.NewTester(w.View, width, height)
	tt.Frame()
	head, ok := tt.Find(locale.SpreadPreview())
	if !ok {
		t.Fatalf("the preview's pane draws no button of its own")
	}
	top, _ := drawerEdge(t, height)
	if off := float32(top) - head.Y; off > 12 || off < -12 {
		t.Errorf("the drawer begins at %v and the preview at %v, want them "+
			"level with each other", top, head.Y)
	}
}

// TestRailView_aButtonOpensItsDrawer checks that the buttons are pressed
// where they stand: over the panels, in the row that holds them, and not
// in an overlay of their own.
func TestRailView_aButtonOpensItsDrawer(t *testing.T) {
	const width, preview = float32(1000), float32(320)
	w, _ := workspaceShown(t)
	w.Layout.Right = preview
	tt := ui.NewTester(w.View, int(width), 600)
	tt.Frame()
	if err := tt.Click(locale.DrawerOutlineLabel()); err != nil {
		t.Fatalf("Click: %v", err)
	}
	tt.Frame()
	if w.drawer != drawerOutline {
		t.Errorf("the drawer is %q, want the outline", w.drawer)
	}
	if err := tt.Click(locale.DrawerOutlineLabel()); err != nil {
		t.Fatalf("Click: %v", err)
	}
	tt.Frame()
	if w.drawer != "" {
		t.Errorf("the drawer is %q, want it shut again", w.drawer)
	}
}

// drawerClick is a click on a link in the drawer: a link is only as wide
// as what it says, and the rest of the line is the panel's, so the click
// goes to its left end and not to the middle of the line.
func drawerClick(t *testing.T, tt *ui.Tester, s string) {
	t.Helper()
	r, ok := tt.Find(s)
	if !ok {
		t.Fatalf("the drawer shows no %q: %v", s, tt.Texts())
	}
	tt.ClickAt(r.X+8, r.Y+r.H/2)
	tt.Frame()
}

// TestDrawerOutline_theHeadingsReadAsAList checks how the drawer shows
// the outline: as Markdown, a list of links, as the note beside it is
// read, and not as a column of buttons.
func TestDrawerOutline_theHeadingsReadAsAList(t *testing.T) {
	w, _ := workspaceShown(t)
	// No preview: with one open the note is read twice over, once in the
	// panel and once in the drawer, and the test could not tell which of
	// the two it had found.
	w.Layout.Right = 0
	n := w.Tabs.ActiveNote()
	n.Editor.Buffer.Set("one\ntwo\nthree\n# a heading\n")
	w.drawer = drawerOutline
	tt := ui.NewTester(w.View, 1000, 600)
	tt.Frame()
	if _, ok := tt.Find("•"); !ok {
		t.Fatalf("the outline is no list: %v", tt.Texts())
	}
	drawerClick(t, tt, "a heading")
	line := n.Editor.Buffer.Pos(n.Editor.Cursor()).Line
	if line != 3 {
		t.Fatalf("the caret is on line %d, want the 3 the heading is on",
			line)
	}
}

// TestDrawerTags_theTagsReadAsAList checks the same of the tags: a list
// of links, and a click on one opens the notes that carry it.
func TestDrawerTags_theTagsReadAsAList(t *testing.T) {
	w, a, _ := windowWithTwoNotes(t)
	w.Layout.Right = 0
	w.Index.Put(a, "# a\n\n#work\n")
	w.drawer = drawerTags
	tt := ui.NewTester(w.View, 1000, 600)
	tt.Frame()
	if _, ok := tt.Find("•"); !ok {
		t.Fatalf("the tags are no list: %v", tt.Texts())
	}
	drawerClick(t, tt, "work")
	if w.tagOpen != "work" {
		t.Fatalf("the tag open is %q, want the one that was clicked",
			w.tagOpen)
	}
}

// TestDrawerLinks_theBacklinksReadAsAList checks the same of the notes
// that name this one: a list of links, and a click on one opens the note
// at the line it was named on.
func TestDrawerLinks_theBacklinksReadAsAList(t *testing.T) {
	w, a, b := windowWithTwoNotes(t)
	w.Layout.Right = 0
	w.Index.Put(a, "# a\n\nsee [[b]] here\n")
	if _, err := w.Tabs.Open(b); err != nil {
		t.Fatalf("Open: %v", err)
	}
	w.drawer = drawerLinks
	tt := ui.NewTester(w.View, 1000, 600)
	tt.Frame()
	drawerClick(t, tt, "a.md")
	if got := w.Tabs.ActiveNote().Path; got != a {
		t.Fatalf("the note open is %q, want the one that was clicked %q",
			got, a)
	}
}

// TestDrawerLineHref_whatADestinationNames checks the round trip of a
// link of the drawer's own: which note, and which line of it.
func TestDrawerLineHref_whatADestinationNames(t *testing.T) {
	for _, want := range []struct {
		path string
		line int
	}{{"", 3}, {"/vault/a.md", 12}, {"/vault/a b.md", 0}} {
		path, line, ok := drawerLineTarget(drawerLineHref(want.path,
			want.line))
		if !ok {
			t.Fatalf("%v is no destination of the drawer's", want)
		}
		if path != want.path || line != want.line {
			t.Errorf("got %q line %d, want %q line %d", path, line,
				want.path, want.line)
		}
	}
}

// TestDrawerTagHref_whatADestinationNames checks the same of a link to a
// tag, empty for all of them again.
func TestDrawerTagHref_whatADestinationNames(t *testing.T) {
	name, ok := drawerTagTarget(drawerTagHref("work"))
	if !ok || name != "work" {
		t.Fatalf("got %q (%v), want the tag that was written", name, ok)
	}
	name, ok = drawerTagTarget(drawerTagHref(""))
	if !ok || name != "" {
		t.Fatalf("got %q (%v), want every tag of the vault", name, ok)
	}
}

// TestMdPlain_whatIsWrittenIsWhatIsRead checks that what the drawer
// shows of a note - its name, and a line of it - is read as it stands,
// and not as marks of Markdown.
func TestMdPlain_whatIsWrittenIsWhatIsRead(t *testing.T) {
	for _, s := range []string{"a [b] c", "*stars*", "1. two", "# head",
		"a `tick`"} {
		got := mdPlain(s)
		if got == s {
			t.Errorf("%q is written as it stands, want it escaped", s)
		}
	}
	if got := mdPlain("one\ntwo"); strings.Contains(got, "\n") {
		t.Errorf("%q still breaks a line", got)
	}
}
