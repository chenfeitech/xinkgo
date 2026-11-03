package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/view"
	"github.com/helightxu/xinkgo/internal/wechat"
)

// stateWithWechat is an app that keeps what it writes in a folder of its
// own, with a window it may set a note out in.
func stateWithWechat(t *testing.T) *appState {
	t.Helper()
	return &appState{
		dir:    t.TempDir(),
		config: config.Default(),
		ws:     view.NewWorkspace(config.Default()),
	}
}

// TestWechatThemes_aStyleOfTheUsersOwnIsOneOfThem checks how a style is
// added without touching the app: a file of JSON in the folder of the
// app's own is a style beside the ones it was made with.
func TestWechatThemes_aStyleOfTheUsersOwnIsOneOfThem(t *testing.T) {
	s := stateWithWechat(t)
	dir := filepath.Join(s.dir, wechatFolder)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "mine.json")
	if err := os.WriteFile(path, []byte(`{"id":"mine","name":"我的样式"}`),
		0o644); err != nil {
		t.Fatal(err)
	}
	themes := s.wechatThemes()
	if !wechat.Has(themes, "mine") {
		t.Fatalf("the style of the user's own is not one of them: %v",
			themeIDs(themes))
	}
	// The app's own come first, and the user's after them.
	if themes[0].ID != wechat.Ink ||
		themes[len(themes)-1].ID != "mine" {
		t.Errorf("the styles are in the order %v", themeIDs(themes))
	}
}

// TestWechatOptions_howANoteIsSetOut checks what a note is set out with:
// the pictures of the note read from the folder it stands in, unless the
// preview is asked for none of them, and the language the app speaks.
func TestWechatOptions_howANoteIsSetOut(t *testing.T) {
	s := stateWithWechat(t)
	got := s.wechatOptions("/vault/a/note.md")
	if !got.Images {
		t.Error("the pictures are left out, though the preview shows them")
	}
	if got.BaseDir != "/vault/a" {
		t.Errorf("the pictures are read from %q, want /vault/a", got.BaseDir)
	}
	s.config.Preview.ImageDisplay = config.ImageHidden
	if s.wechatOptions("/vault/a/note.md").Images {
		t.Error("the pictures are in it, though the preview shows none")
	}
	s.config.Basic.Language = config.LanguageChinese
	if got := s.wechatOptions("/vault/a/note.md"); got.Lang != "zh-CN" {
		t.Errorf("the language is %q, want zh-CN", got.Lang)
	}
}

// TestWechatStyleOf_aStyleThatIsNoneOfThemIsTheNoteAsItStands checks the
// way out of a style that has been taken away: the note as Markdown reads
// it, and not a style that is not there at all.
func TestWechatStyleOf_aStyleThatIsNoneOfThemIsTheNoteAsItStands(t *testing.T) {
	themes := wechat.Builtin()
	if got := wechatStyleOf(themes, wechat.Jade); got != wechat.Jade {
		t.Errorf("a style that is one of them gave %q, want %q", got,
			wechat.Jade)
	}
	for _, id := range []string{"gone", ""} {
		if got := wechatStyleOf(themes, id); got != "" {
			t.Errorf("the style %q gave %q, want none", id, got)
		}
	}
}

// TestWechatDir_theAppsOwnWhenNoneIsNamed checks the folder the styles
// are read from: the one the user named, and the folder of the app's own
// when they named none, which is the way a first start has it.
func TestWechatDir_theAppsOwnWhenNoneIsNamed(t *testing.T) {
	s := stateWithWechat(t)
	own := filepath.Join(s.dir, wechatFolder)
	if got := s.wechatDir(); got != own {
		t.Errorf("the folder is %q, want the app's own %q", got, own)
	}
	s.config.Preview.WechatThemeDir = "/styles/of/my/own"
	if got := s.wechatDir(); got != "/styles/of/my/own" {
		t.Errorf("the folder is %q, want the one named", got)
	}
}

// TestLoadWechatThemes_theStylesOfTheFolderAreReadAgain checks a folder
// named anew: the styles it holds are the ones the menu shows, and those
// of the folder before it are no longer among them.
func TestLoadWechatThemes_theStylesOfTheFolderAreReadAgain(t *testing.T) {
	s := stateWithWechat(t)
	dir := t.TempDir()
	write(t, filepath.Join(dir, "mine.json"), []byte(`{"id":"mine"}`))
	s.config.Preview.WechatThemeDir = dir
	s.loadWechatThemes()
	if !wechat.Has(s.ws.WechatThemes, "mine") {
		t.Errorf("the style of the folder is not one of them: %v",
			idsOfThemes(s.ws.WechatThemes))
	}
	other := t.TempDir()
	write(t, filepath.Join(other, "other.json"), []byte(`{"id":"other"}`))
	s.config.Preview.WechatThemeDir = other
	s.loadWechatThemes()
	if !wechat.Has(s.ws.WechatThemes, "other") {
		t.Errorf("the styles were not read again: %v",
			idsOfThemes(s.ws.WechatThemes))
	}
	if wechat.Has(s.ws.WechatThemes, "mine") {
		t.Errorf("a style of the folder before is still one of them: %v",
			idsOfThemes(s.ws.WechatThemes))
	}
}

// TestWechatByOf_whatARowSaysOfAStyle checks the line under a style's name
// in the settings: which of it this is, who made it, and what it says of
// itself, and nothing of any of them that its file does not say.
func TestWechatByOf_whatARowSaysOfAStyle(t *testing.T) {
	whole := wechat.Theme{
		Version: "2.1.0", Author: "谁做的", Description: "绿底红字",
	}
	if got := wechatByOf(whole); got != "2.1.0 · 谁做的 — 绿底红字" {
		t.Errorf("the row says %q", got)
	}
	if got := wechatByOf(wechat.Theme{Name: "无名"}); got != "" {
		t.Errorf("the row says %q of a style whose file says nothing", got)
	}
	if got := wechatByOf(wechat.Theme{Version: "1.0"}); got != "1.0" {
		t.Errorf("the row says %q, want 1.0", got)
	}
}

// TestSettingsWechatDirOf_theAppsOwnWhenTheRowIsEmpty checks the folder a
// row of the settings names: the one written in it, and the app's own
// when it is left empty, which is what the placeholder says.
func TestSettingsWechatDirOf_theAppsOwnWhenTheRowIsEmpty(t *testing.T) {
	st := &settings{wechatBase: "/the/apps/own"}
	if got := st.wechatDirOf(); got != "/the/apps/own" {
		t.Errorf("the folder is %q, want the app's own", got)
	}
	st.wechatDir = "  /named  "
	if got := st.wechatDirOf(); got != "/named" {
		t.Errorf("the folder is %q, want the one named", got)
	}
}

// write puts a file in the place it is named for.
func write(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// idsOfThemes is the ids of the styles, in their order.
func idsOfThemes(themes []wechat.Theme) []string {
	ids := make([]string, 0, len(themes))
	for _, t := range themes {
		ids = append(ids, t.ID)
	}
	return ids
}

// themeIDs is the ids of the styles, in their order.
func themeIDs(themes []wechat.Theme) []string {
	ids := make([]string, 0, len(themes))
	for _, t := range themes {
		ids = append(ids, t.ID)
	}
	return ids
}
