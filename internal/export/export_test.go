package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// note is a note with something of everything in it, as the preview reads
// it: its links in double brackets already made into ones that can be
// clicked.
const note = `# A heading

Some words, and *one* of them in italics.

- one thing
- [ ] a thing not done yet

| a | b |
|---|---|
| 1 | 2 |

> said by someone else

` + "```go\nfmt.Println(\"hi <there>\")\n```" + `

![a picture](pict.png)
`

// TestHTML_theWordsOfTheNote checks that what is written out is the note,
// as cmark-gfm renders it, and not another reading of it: the headings are
// headings, the table a table, and what would be read as markup is not.
func TestHTML_theWordsOfTheNote(t *testing.T) {
	page := page(t, note, Options{})
	for _, want := range []string{
		"<h1>A heading</h1>", "<em>one</em>", "<ul>", "<table>",
		"<blockquote>", "<pre><code", "&quot;hi &lt;there&gt;&quot;",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the page does not say %q", want)
		}
	}
}

// TestHTML_thePageStandsOnItsOwn checks that it is a document and not a
// piece of one: it says how it is written, what the note is called, and
// how it is to look when it is put on paper.
func TestHTML_thePageStandsOnItsOwn(t *testing.T) {
	page := page(t, note, Options{Title: "A note", Scheme: "xink"})
	for _, want := range []string{
		"<!doctype html>", `<meta charset="utf-8">`, "<title>A note</title>",
		"--fg:", "@media print", "break-inside: avoid",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the page does not say %q", want)
		}
	}
}

// TestHTML_theTitleIsEscaped checks a name the browser would otherwise
// read as markup of the page's own.
func TestHTML_theTitleIsEscaped(t *testing.T) {
	page := page(t, note, Options{Title: `a <b> & "c"`})
	if !strings.Contains(page, "<title>a &lt;b&gt; &amp; &#34;c&#34;</title>") {
		t.Error("the title is not escaped")
	}
}

// TestHTML_aPictureIsWrittenIntoThePage checks the way a page stands
// alone: the picture is in it, and there is nothing beside it to keep.
func TestHTML_aPictureIsWrittenIntoThePage(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pict.png"), []byte("a picture"), 0o644)
	page := page(t, note, Options{
		BaseDir: dir, Images: ImagesEmbed,
	})
	if !strings.Contains(page, `src="data:image/png;base64,YSBwaWN0dXJl"`) {
		t.Error("the picture is not written into the page")
	}
}

// TestHTML_aPictureIsPutBesideThePage checks the other way: the page says
// where the picture is, and the bytes are given to whoever asked for the
// page to be written.
func TestHTML_aPictureIsPutBesideThePage(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pict.png"), []byte("a picture"), 0o644)
	var kept []string
	page := page(t, note, Options{
		BaseDir: dir, Images: ImagesCopy,
		Copy: func(name string, data []byte) error {
			kept = append(kept, name)
			if string(data) != "a picture" {
				t.Errorf("%q: the bytes are %q", name, data)
			}
			return nil
		},
	})
	if len(kept) != 1 || kept[0] != "assets/pict.png" {
		t.Fatalf("kept %q, want [assets/pict.png]", kept)
	}
	if !strings.Contains(page, `src="assets/pict.png"`) {
		t.Error("the page does not read the picture from beside it")
	}
}

// TestHTML_twoPicturesOfOneName checks two pictures of one name, from two
// folders: neither is to take the other's place beside the page.
func TestHTML_twoPicturesOfOneName(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "one"), 0o755)
	os.MkdirAll(filepath.Join(dir, "two"), 0o755)
	os.WriteFile(filepath.Join(dir, "one", "pict.png"), []byte("one"), 0o644)
	os.WriteFile(filepath.Join(dir, "two", "pict.png"), []byte("two"), 0o644)
	var kept []string
	page := page(t, "![a](one/pict.png)\n\n![b](two/pict.png)\n", Options{
		BaseDir: dir, Images: ImagesCopy,
		Copy: func(name string, data []byte) error {
			kept = append(kept, name)
			return nil
		},
	})
	if len(kept) != 2 || kept[0] == kept[1] {
		t.Fatalf("kept %q, want two names", kept)
	}
	if !strings.Contains(page, `src="`+kept[0]+`"`) ||
		!strings.Contains(page, `src="`+kept[1]+`"`) {
		t.Errorf("the page does not read both: %s", page)
	}
}

// TestHTML_aPictureLeftOut checks the third way: the picture is not in
// the page at all, and what the note calls it stands in its place.
func TestHTML_aPictureLeftOut(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pict.png"), []byte("a picture"), 0o644)
	page := page(t, note, Options{BaseDir: dir, Images: ImagesNone})
	if strings.Contains(page, "<img") {
		t.Error("the picture is in the page after all")
	}
	if !strings.Contains(page, `<span class="pict">a picture</span>`) {
		t.Error("the description of the picture is not in its place")
	}
}

// TestHTML_aPictureThatIsNotThere checks a note that names a picture the
// folder does not hold: the page still stands, and says what it called it.
func TestHTML_aPictureThatIsNotThere(t *testing.T) {
	page := page(t, note, Options{
		BaseDir: t.TempDir(), Images: ImagesEmbed,
	})
	if strings.Contains(page, "<img") {
		t.Error("a picture that is not there is in the page")
	}
	if !strings.Contains(page, `<span class="pict">a picture</span>`) {
		t.Error("the description is not in its place")
	}
}

// TestHTML_aPictureOnTheWebIsLeftAsItIs checks a picture the note names
// by its address: it is not read into the page, for it is not a file of
// the note's own.
func TestHTML_aPictureOnTheWebIsLeftAsItIs(t *testing.T) {
	page := page(t, "![a](https://example.com/a.png)\n", Options{
		BaseDir: t.TempDir(), Images: ImagesEmbed,
	})
	if !strings.Contains(page, `src="https://example.com/a.png"`) {
		t.Error("the address is not left as it was")
	}
}

// page is the note written out, as the tests ask for it.
func page(t *testing.T, src string, o Options) string {
	t.Helper()
	got, err := HTML(src, o)
	if err != nil {
		t.Fatalf("the page could not be written: %v", err)
	}
	return string(got)
}
