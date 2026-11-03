package view

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/wechat"
)

// wechatWindow is the window drawn with a note in it, the styles of
// WeChat it may be set out in, and the id of the one the menu asks for.
func wechatWindow(t *testing.T) (*Workspace, *ui.Tester, *string) {
	t.Helper()
	w, te := windowDrawn(t, 600)
	w.WechatThemes = wechat.Builtin()
	asked := ""
	w.OnWechatStyle = func(id string) { asked = id }
	te.Frame()
	return w, te, &asked
}

// TestWechat_theMenuCarriesEveryStyle checks the menu of the button in the
// corner of the preview: the note as Markdown reads it first, and then
// every style a note may be set out in, the app's own and the ones the
// user brought.
func TestWechat_theMenuCarriesEveryStyle(t *testing.T) {
	w, te, _ := wechatWindow(t)
	if !w.wechatOn() {
		w.Config.Preview.Wechat = wechat.Ink
		te.Frame()
	}
	if err := te.Click(locale.WechatStyle()); err != nil {
		t.Fatalf("Click %q: %v", locale.WechatStyle(), err)
	}
	te.Frame()
	got := te.Menu()
	want := []string{locale.WechatStyleMarkdown(), "-"}
	for _, th := range w.WechatThemes {
		want = append(want, th.Name)
	}
	if len(got) != len(want) {
		t.Fatalf("the menu holds %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("the menu holds %q, want %q", got, want)
		}
	}
}

// TestWechat_choosingAStyleAsksForIt checks what a choice does: the style
// the user chose is asked for, by its id, and by no id at all when it is
// the note as Markdown reads it.
func TestWechat_choosingAStyleAsksForIt(t *testing.T) {
	w, te, asked := wechatWindow(t)
	if err := te.Click(locale.WechatStyle()); err != nil {
		t.Fatalf("Click: %v", err)
	}
	te.Frame()
	te.ChooseMenuItem(w.WechatThemes[1].Name)
	if *asked != w.WechatThemes[1].ID {
		t.Errorf("asked for %q, want %q", *asked, w.WechatThemes[1].ID)
	}

	if err := te.Click(locale.WechatStyle()); err != nil {
		t.Fatalf("Click: %v", err)
	}
	te.Frame()
	te.ChooseMenuItem(locale.WechatStyleMarkdown())
	if *asked != "" {
		t.Errorf("asked for %q, want no style at all", *asked)
	}
}

// TestWechat_theCopyIsThereWhileAStyleIs checks the button that copies the
// note as it is set out: it stands in the corner while a style is chosen,
// and is not there at all while the note is read as Markdown reads it,
// for what would be copied then is the note as it stands, which the
// note's own pane gives.
func TestWechat_theCopyIsThereWhileAStyleIs(t *testing.T) {
	w, te, _ := wechatWindow(t)
	if te.HasText(locale.WechatCopy()) {
		t.Error("the copy is there with no style chosen")
	}
	w.Config.Preview.Wechat = wechat.Ink
	te.Frame()
	if !te.HasText(locale.WechatCopy()) {
		t.Error("the copy is not there with a style chosen")
	}
}

// TestWechatScreen_theWidthOfAPhone checks the width of the screen the
// article is read on: that of a phone, and no wider, however wide the
// pane is, for the article is read on a phone and the preview is to say
// how it reads there. A pane with too little room keeps none beside the
// screen, and one narrower still gives the screen all it has.
func TestWechatScreen_theWidthOfAPhone(t *testing.T) {
	for _, c := range []struct {
		room         float32
		screen, want float32
		pad          float32
	}{
		{room: 1200, screen: phoneWidth, pad: phonePad},
		{room: 600, screen: phoneWidth, pad: phonePad},
		{room: phoneWidth + 2*phonePad, screen: phoneWidth, pad: phonePad},
		{room: phoneWidth, screen: phoneWidth},
		{room: 200, screen: 200},
		{room: 0},
	} {
		got, pad := wechatScreen(c.room)
		if got != c.screen || pad != c.pad {
			t.Errorf("a pane of %v gives a screen of %v with %v "+
				"beside it, want %v with %v", c.room, got, pad,
				c.screen, c.pad)
		}
		// The screen and the room beside it never ask for more than the
		// pane has: an article cut off at the side is no article.
		if got+2*pad > c.room {
			t.Errorf("a pane of %v is asked for %v", c.room, got+2*pad)
		}
	}
}

// TestWechat_theRailHangsBelowTheCornerOfThePane checks that the rail —
// the strip of buttons down the right of the window, the outline's and
// the tags' among them — does not stand on the buttons in the corner of
// the preview: it hangs below the row at the top of the pane, whose
// other end holds them.
func TestWechat_theRailHangsBelowTheCornerOfThePane(t *testing.T) {
	w, te, _ := wechatWindow(t)
	w.Config.Preview.Wechat = wechat.Ink
	te.Frame()
	style, ok := te.Find(locale.WechatStyle())
	if !ok {
		t.Fatal("the style button is not drawn")
	}
	drawer, ok := te.Find(drawerOutlineMark)
	if !ok {
		t.Fatal("the rail's outline button is not drawn")
	}
	if drawer.Y < style.Y+style.H {
		t.Errorf("the rail stands at y=%v, and the corner of the pane "+
			"ends at y=%v", drawer.Y, style.Y+style.H)
	}
}

// TestWechat_aPageIsShownWhenTheAppHasFilledOne checks when the preview
// shows the article: a style is chosen, and the app has filled a page for
// it. Without the page — which is what a machine without a webview has —
// what is read is the note as Markdown reads it.
func TestWechat_aPageIsShownWhenTheAppHasFilledOne(t *testing.T) {
	w := &Workspace{Config: config.Default()}
	if w.wechatShown() {
		t.Error("a page is said to be shown with no style chosen")
	}
	w.Config.Preview.Wechat = wechat.Jade
	if w.wechatShown() {
		t.Error("a page is said to be shown, and the app has filled none")
	}
}
