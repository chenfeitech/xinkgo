package wechat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sample is a note with a part of every kind in it.
const sample = "# Title\n\nSome *leaning* words and `code`.\n\n" +
	"> A quote, and\n>\n> - a list in it\n\n" +
	"- [ ] a task not done\n- [x] one that is\n\n" +
	"| a | b |\n|---|---|\n| 1 | 2 |\n\n" +
	"```go\nfmt.Println()\n```\n\n![a picture](p.png)\n"

// article is the note set out in the style named.
func article(t *testing.T, src, id string, o Options) string {
	t.Helper()
	o.Theme = Get(Builtin(), id)
	return Article(src, o)
}

// TestArticle_everyPartCarriesItsOwnLook checks the one thing an article
// for WeChat cannot do without: that every part of it says how it looks
// in the tag itself, and not in a style sheet the editor would throw
// away.
func TestArticle_everyPartCarriesItsOwnLook(t *testing.T) {
	got := article(t, sample, Ink, Options{})
	for _, tag := range []string{"<div style=", "<h1 style=", "<p style=",
		"<em style=", "<code style=", "<blockquote style=", "<li style=",
		"<th style=", "<td style=", "<pre style="} {
		if !strings.Contains(got, tag) {
			t.Errorf("the article has no %q in it:\n%s", tag, got)
		}
	}
	if strings.Contains(got, "<style") || strings.Contains(got, "<script") {
		t.Errorf("the article carries a style sheet or a script, which "+
			"the editor of WeChat throws away:\n%s", got)
	}
}

// TestArticle_aPartInsideAnotherIsSetOutAsThatOneIs checks that a style
// may say of a part inside another what it does not say of the part at
// large: a paragraph in a quote stands as a quote's paragraphs do.
func TestArticle_aPartInsideAnotherIsSetOutAsThatOneIs(t *testing.T) {
	got := article(t, "> quoted\n", Ink, Options{})
	quote := Get(Builtin(), Ink).Style["blockquote"]
	inner := Get(Builtin(), Ink).Style["blockquote>p"]
	at := strings.Index(got, "<blockquote")
	if at < 0 || !strings.Contains(got[at:], `style="`+inner+`"`) {
		t.Errorf("the paragraph in the quote is not set out as one:\n%s", got)
	}
	if !strings.Contains(got, `style="`+quote+`"`) {
		t.Errorf("the quote is not set out as one:\n%s", got)
	}
}

// TestArticle_codeInABlockIsNotCodeInAWord checks the two kinds of code:
// the code of a block, which a style may set on a dark ground of its
// own, and the code of a word inside a line.
func TestArticle_codeInABlockIsNotCodeInAWord(t *testing.T) {
	th := Get(Builtin(), Lapis)
	got := Article("`a word`\n\n```\na block\n```\n",
		Options{Theme: th})
	block := th.Style["pre>code"]
	if !strings.Contains(got, `style="`+block+`"`) {
		t.Errorf("the code of the block is not set out as a block's:\n%s", got)
	}
	if strings.Count(got, `style="`+block+`"`) != 1 {
		t.Errorf("the code of a word is set out as a block's:\n%s", got)
	}
	if !strings.Contains(got, `style="`+th.Style["code"]+`"`) {
		t.Errorf("the code of a word is not set out at all:\n%s", got)
	}
}

// TestArticle_aTaskIsWrittenAsTheBoxItLooksLike checks the boxes of a
// list of tasks: the editor of WeChat drops an <input>, so the box is a
// character instead, ticked for a task that is done.
func TestArticle_aTaskIsWrittenAsTheBoxItLooksLike(t *testing.T) {
	got := article(t, "- [ ] not done\n- [x] done\n", Ink, Options{})
	for _, want := range []string{"☐ not done", "☑ done"} {
		if !strings.Contains(got, want) {
			t.Errorf("the tasks are not written as boxes: want %q in\n%s",
				want, got)
		}
	}
	if strings.Contains(got, "<input") {
		t.Errorf("the article holds an <input>, which the editor of "+
			"WeChat throws away:\n%s", got)
	}
}

// TestArticle_aPictureIsWrittenIntoTheArticle checks a picture of the
// note's own: the editor of WeChat has no folder of the note's to read
// it from, so it is written into the article itself.
func TestArticle_aPictureIsWrittenIntoTheArticle(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "p.png"), []byte("png"))
	got := article(t, "![a picture](p.png)\n", Ink,
		Options{BaseDir: dir, Images: true})
	if !strings.Contains(got, `src="data:image/png;base64,`) {
		t.Errorf("the picture is not written into the article:\n%s", got)
	}
}

// TestArticle_aPictureLeftOutIsSaidInWords checks the other way with a
// picture: the words that describe it stand in its place, as they do in
// the preview when the user asks for none of them.
func TestArticle_aPictureLeftOutIsSaidInWords(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "p.png"), []byte("png"))
	got := article(t, "![a picture](p.png)\n", Ink, Options{BaseDir: dir})
	if strings.Contains(got, "<img") {
		t.Errorf("the picture is in the article all the same:\n%s", got)
	}
	if !strings.Contains(got, "a picture") {
		t.Errorf("nothing says what the picture was:\n%s", got)
	}
}

// TestArticle_aPictureThatIsNotAFileIsLeftAsItStands checks a picture
// the note names by its address on the web: it is not the app's to read,
// and is left to whoever reads the article.
func TestArticle_aPictureThatIsNotAFileIsLeftAsItStands(t *testing.T) {
	got := article(t, "![a](https://x.test/a.png)\n", Ink,
		Options{Images: true})
	if !strings.Contains(got, `src="https://x.test/a.png"`) {
		t.Errorf("the picture is not left as it stands:\n%s", got)
	}
}

// TestArticle_aLookWithAQuoteInItDoesNotEndTheTag checks a style whose
// words hold a quote, as a family of fonts named in quotes does: the
// look is written inside the tag, and a quote of its own would end it
// before its time.
func TestArticle_aLookWithAQuoteInItDoesNotEndTheTag(t *testing.T) {
	th := Get(Builtin(), Ink)
	th.Style["p"] = `font-family: "PingFang SC"; margin: 0;`
	got := Article("words\n", Options{Theme: th})
	if !strings.Contains(got, `style="font-family: &quot;PingFang SC&quot;; margin: 0;"`) {
		t.Errorf("the quote in the look is not written as an entity:\n%s", got)
	}
	if strings.Count(got, "<p ") != 1 {
		t.Errorf("the paragraph is not one paragraph:\n%s", got)
	}
}

// TestArticle_aPartOfANotesOwnMakingIsLeftAsItIs checks what a note may
// hold of its own: a tag with an attribute of its own in it, one that
// stands alone, a style sheet and a script — which the editor of WeChat
// throws away, and which must not throw the article out of shape.
func TestArticle_aPartOfANotesOwnMakingIsLeftAsItIs(t *testing.T) {
	src := "words\n\n<br />\n\n<style>p { margin: 1px }</style>\n\n" +
		"<script>x()</script>\n\n<div data-x=\"a > b\">more</div>\n"
	got := article(t, src, Ink, Options{})
	if strings.Contains(got, "<style") || strings.Contains(got, "x()") {
		t.Errorf("the style sheet or the script is in the article:\n%s", got)
	}
	if !strings.Contains(got, `data-x="a > b"`) {
		t.Errorf("the attribute of the note's own is not as it was:\n%s", got)
	}
	if !strings.Contains(got, "<br />") || strings.Contains(got, "</br>") {
		t.Errorf("a tag that stands alone is not as it was:\n%s", got)
	}
}

// TestPage_theArticleStandsOffTheEdgesOfTheScreen checks the room the
// page keeps beside the article: the screen of a phone keeps an article
// off its edges, as WeChat does. The article itself keeps none of it,
// for what is pasted is given the room of the editor it is pasted into.
func TestPage_theArticleStandsOffTheEdgesOfTheScreen(t *testing.T) {
	th := Get(Builtin(), Ink)
	page := Page("words\n", Options{Theme: th})
	if !strings.Contains(page, "padding: 0 20px") {
		t.Errorf("the page keeps no room beside the article:\n%s", page)
	}
	if got := Article("words\n", Options{Theme: th}); strings.Contains(got,
		"padding: 0 20px") {
		t.Errorf("the article keeps a room that is the page's:\n%s", got)
	}
}

// TestPage_theArticleInAPageOfItsOwn checks what the preview shows: the
// same article, in a page of its own, so that what is seen is what is
// pasted.
func TestPage_theArticleInAPageOfItsOwn(t *testing.T) {
	got := Page(sample, Options{Theme: Get(Builtin(), Ink), Lang: "zh-CN"})
	for _, want := range []string{"<!doctype html>", `<html lang="zh-CN">`,
		`<meta charset="utf-8">`, "<body", "</html>"} {
		if !strings.Contains(got, want) {
			t.Errorf("the page has no %q in it:\n%s", want, got)
		}
	}
	if !strings.Contains(got, ">Title<") {
		t.Errorf("the page does not hold the article:\n%s", got)
	}
}

// TestOfDir_aStyleOfTheUsersOwnIsOneOfThem checks how a style is added
// without touching the app: a file of JSON in a folder of the app's own
// is a style beside the ones it was made with.
func TestOfDir_aStyleOfTheUsersOwnIsOneOfThem(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "mine.json"), []byte(`{
		"id": "mine",
		"name": "我的样式",
		"style": {"p": "margin: 0 0 2em;"}
	}`))
	write(t, filepath.Join(dir, "broken.json"), []byte(`{`))
	themes, bad := OfDir(dir)
	if len(themes) != 1 || themes[0].ID != "mine" {
		t.Fatalf("the folder gave %v, want one style, mine", themes)
	}
	if len(bad) != 1 || !strings.HasPrefix(bad[0], "broken.json") {
		t.Errorf("the file that is no style is not said: %v", bad)
	}
	// A style of the user's own says what it says, and keeps the look of
	// the base for what it says nothing of.
	if themes[0].Style["p"] != "margin: 0 0 2em;" {
		t.Errorf("the style's own look is not the one its file gives")
	}
	if themes[0].Style["h2"] != base["h2"] {
		t.Errorf("what the style says nothing of is not the base's: %q",
			themes[0].Style["h2"])
	}
	if themes[0].Name != "我的样式" {
		t.Errorf("the style is called %q, want the name of its file",
			themes[0].Name)
	}
}

// TestOfDir_aStyleSaysWhoMadeIt checks what a file of a style says beside
// its looks: which of it this is, who made it, and what it says of
// itself — what a folder of styles shared between machines tells of each
// of them.
func TestOfDir_aStyleSaysWhoMadeIt(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "mine.json"), []byte(`{
		"id": "mine",
		"name": "我的样式",
		"version": "2.1.0",
		"author": "谁做的",
		"description": "绿底红字，标题居中"
	}`))
	themes, bad := OfDir(dir)
	if len(bad) != 0 {
		t.Fatalf("the file is said to be no style: %v", bad)
	}
	got := themes[0]
	if got.Version != "2.1.0" || got.Author != "谁做的" ||
		got.Description != "绿底红字，标题居中" {
		t.Errorf("the style says %q %q %q of itself, want what its "+
			"file says", got.Version, got.Author, got.Description)
	}
}

// TestOfDir_aStyleMadeOverAnother checks extends: the looks of the style
// named, with this one's own over them, so that a style of a few lines
// may be made of one that is whole. It may name one the app was made
// with, or one read before it in the same folder.
func TestOfDir_aStyleMadeOverAnother(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "one.json"), []byte(
		`{"id":"one","style":{"p":"margin: 0 0 3em;"}}`))
	write(t, filepath.Join(dir, "two.json"), []byte(
		`{"id":"two","extends":"one","style":{"h2":"color: #123456;"}}`))
	write(t, filepath.Join(dir, "three.json"), []byte(
		`{"id":"three","extends":"`+Ink+`"}`))
	themes, bad := OfDir(dir)
	if len(bad) != 0 {
		t.Fatalf("the files are said to be no styles: %v", bad)
	}
	if len(themes) != 3 {
		t.Fatalf("the folder gave %d styles, want 3", len(themes))
	}
	// Over one of the folder's own, which is read before it: its looks,
	// and then its own over them.
	two := Get(themes, "two")
	if two.Style["p"] != "margin: 0 0 3em;" {
		t.Errorf("the style made over another does not have its looks: %q",
			two.Style["p"])
	}
	if two.Style["h2"] != "color: #123456;" {
		t.Errorf("the style made over another lost its own looks: %q",
			two.Style["h2"])
	}
	if two.Style["h3"] != base["h3"] {
		t.Errorf("what neither says is not the base's: %q", two.Style["h3"])
	}
	// Over one the app was made with: that one's looks, and nothing of
	// the base's where it says one of its own.
	three := Get(themes, "three")
	if three.Style["h2"] != Get(Builtin(), Ink).Style["h2"] {
		t.Errorf("the style made over one of the app's own does not "+
			"have its looks: %q", three.Style["h2"])
	}
}

// TestOfDir_aStyleThatExtendsWhatIsNotThereIsSaid checks the way out of a
// style that extends one that is not there: the base's looks, and the
// file said, for a style that comes out wrong is worse than one that
// says so.
func TestOfDir_aStyleThatExtendsWhatIsNotThereIsSaid(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "mine.json"), []byte(
		`{"id":"mine","extends":"gone"}`))
	themes, bad := OfDir(dir)
	if len(themes) != 1 {
		t.Fatalf("the folder gave %d styles, want 1", len(themes))
	}
	if len(bad) != 1 || !strings.Contains(bad[0], "gone") {
		t.Errorf("what is wrong with the file is not said: %v", bad)
	}
	if themes[0].Style["p"] != base["p"] {
		t.Errorf("the style kept no looks at all")
	}
}

// TestOfDir_twoFilesOfOneIdChecksTheSecond checks two files named for one
// style: the first is the style, and the second is said, for two styles
// of one name cannot both be chosen by it.
func TestOfDir_twoFilesOfOneIdChecksTheSecond(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "a.json"),
		[]byte(`{"id":"mine","name":"先读到的"}`))
	write(t, filepath.Join(dir, "b.json"),
		[]byte(`{"id":"mine","name":"后读到的"}`))
	themes, bad := OfDir(dir)
	if len(themes) != 1 || themes[0].Name != "先读到的" {
		t.Fatalf("the folder gave %v, want the first of the two", themes)
	}
	if len(bad) != 1 || !strings.HasPrefix(bad[0], "b.json") {
		t.Errorf("the second file is not said: %v", bad)
	}
}

// TestOfDir_aStyleOfNoIdIsSaid checks a file that names no id: there is
// nothing to choose a style by, and nothing to keep in the settings.
func TestOfDir_aStyleOfNoIdIsSaid(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "mine.json"),
		[]byte(`{"name":"没有 id 的样式"}`))
	themes, bad := OfDir(dir)
	if len(themes) != 0 {
		t.Errorf("a style of no id was given: %v", themes)
	}
	if len(bad) != 1 {
		t.Errorf("the file is not said: %v", bad)
	}
}

// TestThemes_theAppsOwnComeFirst checks the order the menu shows: the
// styles the app was made with, and then the ones the user brought.
func TestThemes_theAppsOwnComeFirst(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "mine.json"),
		[]byte(`{"id":"mine","name":"mine"}`))
	got := Themes(dir)
	if len(got) != len(builtin)+1 {
		t.Fatalf("there are %d styles, want %d", len(got), len(builtin)+1)
	}
	if got[0].ID != Ink || got[len(got)-1].ID != "mine" {
		t.Errorf("the styles are in the order %v", idsOf(got))
	}
}

// TestGet_aNameThatIsNoneOfThemIsTheFirst checks the way out of a style
// the user has taken away: the first of them, which is the one a first
// start has, and never one that is not there at all.
func TestGet_aNameThatIsNoneOfThemIsTheFirst(t *testing.T) {
	themes := Builtin()
	if got := Get(themes, "gone"); got.ID != themes[0].ID {
		t.Errorf("a name that is none of them gave %q, want %q", got.ID,
			themes[0].ID)
	}
	if got := Get(nil, "gone"); got.Style["p"] != base["p"] {
		t.Errorf("the base is not what a style with none of them has")
	}
	if !Has(themes, Lapis) || Has(themes, "gone") {
		t.Errorf("Has is wrong about the styles")
	}
}

// idsOf is the ids of the styles, in their order.
func idsOf(themes []Theme) []string {
	ids := make([]string, 0, len(themes))
	for _, t := range themes {
		ids = append(ids, t.ID)
	}
	return ids
}

// write puts a file in the place it is named for.
func write(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
