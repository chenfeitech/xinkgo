// Package export writes a note out as a file another program reads: a
// page of HTML that stands on its own, whose PDF the system's own webview
// prints (see internal/app).
//
// The words are the ones the preview reads. The parser that draws the
// preview writes HTML of its own, as cmark-gfm renders it, so nothing
// here reads Markdown: this is that HTML, put in a page of the app's own
// making — in the colors of the scheme the window is drawn in, with the
// rules that keep a table, a picture and a block of code from being cut
// in two by the edge of a sheet of paper.
//
// What the preview does as it draws, and the document does not, is done
// here as well: the pictures of a note are read from the note's own
// folder, and are either written into the page, put beside it, or left
// out for their descriptions.
package export

import (
	"encoding/base64"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown/parser"
	"github.com/helightxu/xinkgo/internal/theme"
)

// Assets is the folder the pictures of a note are put in when they stand
// beside the page rather than in it.
const Assets = "assets"

// Images is what an export does with the pictures a note names.
type Images int

const (
	// ImagesNone leaves the pictures out and says their descriptions in
	// their place, as the preview does when the user asks for none of
	// them.
	ImagesNone Images = iota
	// ImagesEmbed writes every picture into the page itself, which then
	// stands alone and may be sent as it is, and is as large as the
	// pictures are.
	ImagesEmbed
	// ImagesCopy puts the pictures beside the page, in Assets, which
	// stays small but is two things to keep together.
	ImagesCopy
)

// Options is how a page is written out.
type Options struct {
	// Title is what the page is called: the name of the note, as the
	// window's title and the file's name say it.
	Title string
	// Lang is the language of the page, as a BCP 47 tag: "en" or "zh-CN".
	Lang string
	// Scheme is the name of the color scheme the window is drawn in, and
	// Dark whether it is drawn in its dark colors.
	Scheme string
	Dark   bool
	// Font is the family the page is set in, as CSS names it.
	Font string
	// BaseDir is where the pictures a note names by a relative path are
	// read from: the folder of the note itself.
	BaseDir string
	// Images is what is done with the pictures.
	Images Images
	// Copy is asked to keep the bytes of a picture under that name, for
	// ImagesCopy: beside the page, in Assets. It is given the name the
	// page then writes in the tag, and not only the file's own.
	Copy func(name string, data []byte) error
}

// HTML is the note in src as a page that stands on its own.
func HTML(src string, o Options) ([]byte, error) {
	w := &writer{o: o, used: map[string]bool{}}
	body, err := w.pictures(parser.Parse(src).HTML())
	if err != nil {
		return nil, err
	}
	return []byte(w.page(body)), nil
}

// writer writes one page: it keeps the names of the pictures it has put
// beside it, so that two of one name do not take each other's place.
type writer struct {
	o    Options
	used map[string]bool
}

// The tag the parser writes for a picture, and the end it gives it.
const (
	imgOpen  = `<img src="`
	imgClose = `" />`
)

// pictures is the body of the page with every picture dealt with: written
// into it, put beside it, or left out for its description.
func (w *writer) pictures(body string) (string, error) {
	var b strings.Builder
	at := 0
	for {
		from := strings.Index(body[at:], imgOpen)
		if from < 0 {
			b.WriteString(body[at:])
			return b.String(), nil
		}
		from += at
		b.WriteString(body[at:from])
		// A tag the parser wrote ends where it says; one it did not is
		// left as it stands.
		to := strings.Index(body[from:], imgClose)
		if to < 0 {
			b.WriteString(body[from:])
			return b.String(), nil
		}
		tag := body[from : from+to+len(imgClose)]
		alt := attribute(tag, `alt="`)
		dest := attribute(tag, `src="`)
		src, kept, err := w.picture(dest, alt)
		if err != nil {
			return "", err
		}
		if kept {
			b.WriteString(imgOpen + src + `" alt="` + alt + imgClose)
		} else {
			b.WriteString(`<span class="pict">` + alt + `</span>`)
		}
		at = from + len(tag)
	}
}

// picture is one picture: where the page is to read it from, and whether
// it is in the page at all. One that is not there, or that could not be
// read, is left out, and its description stands in its place.
func (w *writer) picture(dest, _ string) (string, bool, error) {
	if w.o.Images == ImagesNone {
		return "", false, nil
	}
	// A picture that is not a file — one the note names by its address on
	// the web — is left to whoever reads the page.
	if strings.Contains(dest, "://") || strings.HasPrefix(dest, "data:") {
		return dest, true, nil
	}
	path := dest
	if !filepath.IsAbs(path) && w.o.BaseDir != "" {
		path = filepath.Join(w.o.BaseDir, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false, nil
	}
	switch w.o.Images {
	case ImagesCopy:
		if w.o.Copy == nil {
			return "", false, nil
		}
		name := w.nameOf(filepath.Base(path))
		if err := w.o.Copy(name, data); err != nil {
			return "", false, err
		}
		return name, true, nil
	default:
		return "data:" + mimeOf(path) + ";base64," +
			base64.StdEncoding.EncodeToString(data), true, nil
	}
}

// nameOf is the name a picture is kept under in Assets: its own, until
// another picture of that name has taken it, and then one with a number
// in it.
func (w *writer) nameOf(base string) string {
	name := Assets + "/" + base
	if !w.used[name] {
		w.used[name] = true
		return name
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	for n := 2; ; n++ {
		name = fmt.Sprintf("%s/%s-%d%s", Assets, stem, n, ext)
		if !w.used[name] {
			w.used[name] = true
			return name
		}
	}
}

// attribute is what an attribute of a tag holds, as the parser wrote it:
// `src="` asks for where a picture is read from.
func attribute(tag, name string) string {
	i := strings.Index(tag, name)
	if i < 0 {
		return ""
	}
	rest := tag[i+len(name):]
	if j := strings.IndexByte(rest, '"'); j >= 0 {
		return rest[:j]
	}
	return rest
}

// mimeOf is what a picture is, as a browser asks to be told: the kinds
// the app shows, and nothing said of the ones it does not.
func mimeOf(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	case ".avif":
		return "image/avif"
	case ".gif":
		return "image/gif"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	default:
		return "application/octet-stream"
	}
}

// page is the body in a document of the app's own making.
func (w *writer) page(body string) string {
	o := w.o
	if o.Lang == "" {
		o.Lang = "en"
	}
	t := theme.Get(o.Scheme).Of(o.Dark)
	var b strings.Builder
	fmt.Fprintf(&b, "<!doctype html>\n<html lang=%q>\n<head>\n", o.Lang)
	b.WriteString(`<meta charset="utf-8">` + "\n")
	b.WriteString(`<meta name="viewport" content="width=device-width, initial-scale=1">` + "\n")
	b.WriteString(`<meta name="generator" content="Xinkgo">` + "\n")
	fmt.Fprintf(&b, "<title>%s</title>\n", html.EscapeString(o.Title))
	b.WriteString("<style>\n" + style(t, o.Font) + "</style>\n")
	b.WriteString("</head>\n<body>\n<article>\n")
	b.WriteString(body)
	b.WriteString("\n</article>\n</body>\n</html>\n")
	return b.String()
}

// style is how the page looks: the colors of the scheme the window is
// drawn in, and, for paper, the rules that keep a thing whole.
func style(t *ui.Theme, font string) string {
	if font == "" {
		font = "system-ui, -apple-system, sans-serif"
	}
	mono := "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace"
	var b strings.Builder
	fmt.Fprintf(&b, `:root {
  --bg: %s; --fg: %s; --muted: %s; --accent: %s; --border: %s; --surface: %s;
}
`, hex(t.Background), hex(t.Text), hex(t.TextMuted), hex(t.Accent),
		hex(t.Border), hex(t.Surface))
	fmt.Fprintf(&b, `body {
  margin: 0; background: var(--bg); color: var(--fg);
  font-family: %s; font-size: 16px; line-height: 1.65;
}
article { max-width: 44rem; margin: 0 auto; padding: 2rem 1.5rem 4rem; }
h1, h2, h3, h4, h5, h6 { line-height: 1.3; margin: 1.6em 0 .6em; }
h1 { font-size: 1.8em; } h2 { font-size: 1.4em; } h3 { font-size: 1.2em; }
p, ul, ol, blockquote, table, pre { margin: 0 0 1em; }
ul, ol { padding-left: 1.5em; }
li > ul, li > ol { margin: .3em 0; }
a { color: var(--accent); }
hr { border: 0; border-top: 1px solid var(--border); margin: 2em 0; }
blockquote {
  margin-left: 0; padding: 0 0 0 1em; border-left: 3px solid var(--border);
  color: var(--muted);
}
code, pre, kbd { font-family: %s; }
:not(pre) > code {
  background: var(--surface); border: 1px solid var(--border);
  border-radius: 3px; padding: .1em .3em; font-size: .9em;
}
pre {
  background: var(--surface); border: 1px solid var(--border);
  border-radius: 4px; padding: .8em 1em; overflow-x: auto; font-size: .9em;
}
pre code { background: none; border: 0; padding: 0; }
table { border-collapse: collapse; display: block; overflow-x: auto; }
th, td { border: 1px solid var(--border); padding: .35em .7em; }
th { background: var(--surface); }
img { max-width: 100%%; height: auto; }
.pict { color: var(--muted); font-style: italic; }
.math { font-family: %s; }
.footnotes { margin-top: 2em; border-top: 1px solid var(--border); font-size: .9em; }
`, font, mono, mono)
	b.WriteString(`@media print {
  article { max-width: none; margin: 0; padding: 0; }
  pre, blockquote, table, img, .math.display { break-inside: avoid; }
  h1, h2, h3, h4, h5, h6 { break-after: avoid; }
  p { orphans: 2; widows: 2; }
  a { color: inherit; text-decoration: none; }
}
`)
	return b.String()
}

// hex is a color as CSS writes it.
func hex(c ui.Color) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}
