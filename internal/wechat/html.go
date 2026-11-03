package wechat

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui/markdown/parser"
)

// Options is how a note is set out.
type Options struct {
	// Theme is the style it is set out in.
	Theme Theme
	// BaseDir is where the pictures a note names by a relative path are
	// read from: the folder of the note itself.
	BaseDir string
	// Images is whether the pictures a note names are written into the
	// article. The editor of WeChat has no folder of a note's to read
	// them from, so one put beside the page is of no use to it: a
	// picture either stands in the article itself, or is left out for
	// the words that describe it, as the preview leaves it out.
	Images bool
	// Lang is the language of the article, as a BCP 47 tag: "en" or
	// "zh-CN".
	Lang string
}

// Article is the note in src as an article: a piece of HTML whose every
// part carries its own look, which is what is put on the clipboard and
// pasted into the editor of WeChat.
func Article(src string, o Options) string {
	body := styled(pictures(parser.Parse(src).HTML(), o), o.Theme)
	if body == "" {
		return ""
	}
	return `<div style="` + escapeStyle(o.Theme.Style["body"]) + `">` +
		body + `</div>`
}

// pagePad is the room a page keeps beside the article it shows, in CSS
// pixels: the screen of a phone keeps an article off its edges, as
// WeChat does, and an article against the edge of the screen is not one
// read on a phone. It is the page's, and not the article's: what is
// pasted into the editor of WeChat keeps the room that editor gives it.
const pagePad = 20

// Page is the same article in a page of its own, which is what the
// preview shows: the same words and the same looks, so that what is seen
// is what is pasted.
func Page(src string, o Options) string {
	article := Article(src, o)
	if o.Lang == "" {
		o.Lang = "en"
	}
	var b strings.Builder
	b.WriteString("<!doctype html>\n<html lang=" + `"` + o.Lang + `"` + ">\n")
	b.WriteString("<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString(`<meta name="viewport" content="width=device-width, initial-scale=1">` + "\n")
	b.WriteString("<title>Xinkgo</title>\n</head>\n")
	// The article is read on the ground it is pasted onto, which is the
	// white of WeChat's own page, whatever the window is drawn in, and
	// it stands off the sides of the screen, as it does on a phone.
	b.WriteString(`<body style="margin: 0; background: #ffffff; ` +
		`padding: 0 ` + strconv.Itoa(pagePad) + `px;">` + "\n")
	b.WriteString(article + "\n")
	b.WriteString("</body>\n</html>\n")
	return b.String()
}

// The tag the parser writes for a picture, and the end it gives it.
const (
	imgOpen  = `<img src="`
	imgClose = `" />`
)

// pictures is the article with every picture dealt with: written into it,
// or left out for the words that describe it. One that is not a file of
// the note's own — one it names by its address on the web — is left to
// whoever reads the article.
func pictures(body string, o Options) string {
	var b strings.Builder
	at := 0
	for {
		from := strings.Index(body[at:], imgOpen)
		if from < 0 {
			b.WriteString(body[at:])
			return b.String()
		}
		from += at
		b.WriteString(body[at:from])
		to := strings.Index(body[from:], imgClose)
		if to < 0 {
			b.WriteString(body[from:])
			return b.String()
		}
		tag := body[from : from+to+len(imgClose)]
		alt := attribute(tag, `alt="`)
		dest := attribute(tag, `src="`)
		if src, kept := picture(dest, o); kept {
			b.WriteString(imgOpen + src + `" alt="` + alt + imgClose)
		} else {
			b.WriteString(`<span>` + alt + `</span>`)
		}
		at = from + len(tag)
	}
}

// picture is one picture: where the article is to read it from, and
// whether it stands in the article at all. One that is not there, or
// that could not be read, is left out.
func picture(dest string, o Options) (string, bool) {
	if !o.Images {
		return "", false
	}
	if strings.Contains(dest, "://") || strings.HasPrefix(dest, "data:") {
		return dest, true
	}
	path := dest
	if !filepath.IsAbs(path) && o.BaseDir != "" {
		path = filepath.Join(o.BaseDir, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return "data:" + mimeOf(path) + ";base64," +
		base64.StdEncoding.EncodeToString(data), true
}

// attribute is what an attribute of a tag holds, as the parser wrote it:
// `alt="` asks for the words a picture is described in.
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
