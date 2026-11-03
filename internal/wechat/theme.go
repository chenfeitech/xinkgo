// Package wechat sets a note out as an article for WeChat: the words are
// the ones the preview reads, and the look is one of the styles this
// package keeps, put into the tags themselves.
//
// The editor of WeChat keeps nothing of a style sheet: a page pasted into
// it keeps what each of its tags says of itself, and loses the rest. So a
// style is a table of the looks of the parts of a note — a paragraph, a
// heading, a block of code, a cell of a table — as CSS writes each one,
// and this package gives every tag the look its kind asks for.
//
// A style is data, and not code: the ones the app was made with stand
// here, and the ones the user makes stand in a folder of the app's own,
// one file to a style. A style that says nothing of a part gets the look
// the base gives it, so a new style is a handful of lines and nothing
// else.
package wechat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// The styles the app was made with, in the order the menu shows them.
// Their ids are what the configuration keeps.
const (
	// Ink is black and white, with a seal-red accent and headings ruled
	// beneath: the look of a printed essay.
	Ink = "ink"
	// Jade is green: a jade bar down the side of every heading.
	Jade = "jade"
	// Lapis is blue, with its code on a dark ground.
	Lapis = "lapis"
	// Amber is warm: paper the color of candlelight.
	Amber = "amber"
	// Rose is red, with headings written in the accent itself.
	Rose = "rose"
)

// Theme is one way a note is set out: what it is called, who made it,
// and how every part of a note looks in it.
//
// A style the app was made with has all of this in the code; one the
// user made has it in a file of JSON in the folder they named, one file
// to a style:
//
//	{
//	  "id": "jade",
//	  "name": "青玉",
//	  "version": "1.0.0",
//	  "author": "谁做的",
//	  "description": "绿色系：标题带玉色条，引用淡绿底",
//	  "extends": "ink",
//	  "style": {
//	    "h2": "border-left: 4px solid #0f9d58; padding-left: .6em;",
//	    "code": "background: #f0f9f4; color: #0f7a45;"
//	  }
//	}
//
// Only the id is asked for, and only the style says how a note looks;
// the rest is what the settings tell of a style, and what its maker
// says of it.
type Theme struct {
	// ID is what the configuration keeps, and what a style the user
	// made is known by: two files may not hold one id between them.
	ID string `json:"id"`
	// Name is what the menu says. It is the name of the style itself,
	// as the color schemes of internal/theme are named: one the user
	// made is named as their file says, and by its id when their file
	// says no name.
	Name string `json:"name"`
	// Version is which of the style this is, as its maker numbers
	// them, and Author who made it: what a folder of styles shared
	// between machines says of each of them.
	Version string `json:"version,omitempty"`
	Author  string `json:"author,omitempty"`
	// Description is what the style says of itself: what it is for,
	// and what it looks like.
	Description string `json:"description,omitempty"`
	// Extends is the id of the style this one is made over: its looks
	// are that style's, with this one's own over them, so that a style
	// of a handful of lines may be made of one that is whole. It may
	// name one the app was made with, or one of the files read before
	// it in the same folder.
	Extends string `json:"extends,omitempty"`
	// Style is the look of a part of a note, as CSS writes it: "p" is
	// the look of a paragraph, "blockquote p" of one inside a quote,
	// and "pre>code" of the code inside a block of code. A part a
	// style says nothing of keeps the look of the one it extends, or
	// of the base when it extends none.
	Style map[string]string `json:"style"`
	// file is the name of the file a style the user made came from,
	// which is what a folder is told of when the file is no style. It
	// is not of the file's own: the app fills it in as it reads.
	file string
}

// base is the look every part of a note starts from, in the words of CSS:
// what a style that says nothing of its own keeps.
var base = map[string]string{
	"body": `font-family: -apple-system, BlinkMacSystemFont, 'Helvetica Neue',` +
		` 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif;` +
		` font-size: 15px; line-height: 1.75; color: #3f3f3f;` +
		` letter-spacing: .5px; word-break: break-word;` +
		` padding: 4px 2px;`,
	"p":  `margin: 0 0 1.2em;`,
	"h1": `margin: 1.4em 0 1em; font-size: 22px; font-weight: 700; line-height: 1.4; text-align: center;`,
	"h2": `margin: 1.6em 0 .8em; font-size: 19px; font-weight: 700; line-height: 1.4;`,
	"h3": `margin: 1.4em 0 .8em; font-size: 17px; font-weight: 700;`,
	"h4": `margin: 1.3em 0 .7em; font-size: 16px; font-weight: 700;`,
	"h5": `margin: 1.2em 0 .7em; font-size: 15px; font-weight: 700;`,
	"h6": `margin: 1.2em 0 .7em; font-size: 15px; font-weight: 700; color: #8a8a8a;`,
	"blockquote": `margin: 1.2em 0; padding: .6em 1em;` +
		` border-left: 4px solid #d9d9de; background: #f7f7f9; color: #6b6b70;`,
	"blockquote>p": `margin: 0;`,
	"ul":           `margin: 0 0 1.2em; padding-left: 1.6em;`,
	"ol":           `margin: 0 0 1.2em; padding-left: 1.6em;`,
	"li":           `margin: .3em 0;`,
	"pre": `margin: 0 0 1.2em; padding: 1em 1.1em; background: #f7f7f9;` +
		` border-radius: 4px; font-size: 13.5px; line-height: 1.65;` +
		` white-space: pre-wrap; word-break: break-word; overflow-x: auto;`,
	"pre>code": `font-family: Menlo, Consolas, 'Courier New', monospace;` +
		` background: none; padding: 0; color: #3f3f3f;`,
	"code": `font-family: Menlo, Consolas, 'Courier New', monospace;` +
		` font-size: .88em; padding: .15em .4em; background: #f2f2f5;` +
		` border-radius: 3px; color: #c0392b;`,
	"table": `margin: 0 0 1.2em; border-collapse: collapse; width: 100%;` +
		` font-size: 14px;`,
	"th": `border: 1px solid #d9d9de; padding: .4em .6em;` +
		` background: #f4f4f5; font-weight: 700;`,
	"td":     `border: 1px solid #d9d9de; padding: .4em .6em;`,
	"a":      `color: #2563eb; text-decoration: none; font-weight: 500;`,
	"img":    `max-width: 100%; height: auto; display: block; margin: .8em auto; border-radius: 4px;`,
	"hr":     `margin: 2em 0; border: 0; border-top: 1px solid #e4e4e7;`,
	"strong": `font-weight: 700; color: #1f1f23;`,
	"em":     `font-style: italic;`,
	"del":    `text-decoration: line-through; color: #8a8a8a;`,
	"section": `margin-top: 2em; padding-top: 1em;` +
		` border-top: 1px solid #e4e4e7; font-size: 14px; color: #8a8a8a;`,
}

// What the styles the app was made with say of themselves: which of them
// this is, and who made them.
const (
	builtinVersion = "1.0"
	builtinAuthor  = "Xinkgo"
)

// builtin is every style the app was made with, in the order the menu
// shows them. Each says what it makes its own, and nothing of the rest.
var builtin = []Theme{
	{
		ID:      Ink,
		Name:    "墨黑",
		Version: builtinVersion,
		Author:  builtinAuthor,
		Description: "黑白：标题下压一道墨线，朱红作强调，" +
			"如印出来的文章",
		Style: map[string]string{
			"h1": `margin: 1.4em 0 1em; font-size: 23px; font-weight: 700;` +
				` line-height: 1.4; text-align: center; color: #1f1f23;`,
			"h2": `margin: 1.8em 0 .9em; padding-bottom: .3em;` +
				` font-size: 19px; font-weight: 700; color: #1f1f23;` +
				` border-bottom: 2px solid #1f1f23;`,
			"h3": `margin: 1.5em 0 .8em; font-size: 17px; font-weight: 700;` +
				` color: #1f1f23; border-left: 4px solid #1f1f23;` +
				` padding-left: .5em;`,
			"blockquote": `margin: 1.2em 0; padding: .7em 1em;` +
				` border-left: 4px solid #c0392b; background: #fbf4f3;` +
				` color: #6b4a47;`,
			"a":      `color: #c0392b; text-decoration: none; font-weight: 500;`,
			"code":   `font-family: Menlo, Consolas, monospace; font-size: .88em; padding: .15em .4em; background: #f4f1f0; border-radius: 3px; color: #c0392b;`,
			"th":     `border: 1px solid #e0d9d7; padding: .4em .6em; background: #f7f3f2; font-weight: 700;`,
			"td":     `border: 1px solid #e0d9d7; padding: .4em .6em;`,
			"img":    `max-width: 100%; height: auto; display: block; margin: .8em auto; border-radius: 2px;`,
			"hr":     `margin: 2.4em 0; border: 0; border-top: 1px dashed #c9c2c0;`,
			"strong": `font-weight: 700; color: #c0392b;`,
		},
	},
	{
		ID:      Jade,
		Name:    "青玉",
		Version: builtinVersion,
		Author:  builtinAuthor,
		Description: "青绿：标题左侧一道玉色条，" +
			"引用与行内代码都是淡绿底",
		Style: map[string]string{
			"h1": `margin: 1.4em 0 1em; font-size: 22px; font-weight: 700;` +
				` line-height: 1.4; color: #0f7a45;`,
			"h2": `margin: 1.6em 0 .8em; padding: .35em .6em;` +
				` font-size: 18px; font-weight: 700; color: #0f7a45;` +
				` background: #f0f9f4; border-left: 4px solid #0f9d58;`,
			"h3": `margin: 1.4em 0 .8em; font-size: 17px; font-weight: 700;` +
				` color: #0f7a45;`,
			"blockquote": `margin: 1.2em 0; padding: .7em 1em;` +
				` border-left: 4px solid #0f9d58; background: #f2faf5;` +
				` color: #4a6b57;`,
			"a":      `color: #0f9d58; text-decoration: none; font-weight: 500;`,
			"code":   `font-family: Menlo, Consolas, monospace; font-size: .88em; padding: .15em .4em; background: #f0f9f4; border-radius: 3px; color: #0f7a45;`,
			"th":     `border: 1px solid #cfe8d9; padding: .4em .6em; background: #f0f9f4; font-weight: 700;`,
			"td":     `border: 1px solid #cfe8d9; padding: .4em .6em;`,
			"hr":     `margin: 2em 0; border: 0; border-top: 1px solid #cfe8d9;`,
			"strong": `font-weight: 700; color: #0f7a45;`,
		},
	},
	{
		ID:      Lapis,
		Name:    "靛蓝",
		Version: builtinVersion,
		Author:  builtinAuthor,
		Description: "靛蓝：标题带浅蓝下划线，" +
			"代码块是深蓝底浅字",
		Style: map[string]string{
			"h1": `margin: 1.4em 0 1em; font-size: 22px; font-weight: 700;` +
				` line-height: 1.4; text-align: center; color: #1e40af;`,
			"h2": `margin: 1.6em 0 .8em; font-size: 18px; font-weight: 700;` +
				` color: #1e3a8a; border-bottom: 1px solid #bfdbfe;` +
				` padding-bottom: .3em;`,
			"h3": `margin: 1.4em 0 .8em; font-size: 17px; font-weight: 700;` +
				` color: #1e40af;`,
			"blockquote": `margin: 1.2em 0; padding: .7em 1em;` +
				` border-left: 4px solid #1e6fd9; background: #eff6ff;` +
				` color: #41547a;`,
			"a": `color: #1e6fd9; text-decoration: none; font-weight: 500;`,
			"pre": `margin: 0 0 1.2em; padding: 1em 1.1em;` +
				` background: #1e293b; border-radius: 6px;` +
				` font-size: 13.5px; line-height: 1.65;` +
				` white-space: pre-wrap; word-break: break-word;`,
			"pre>code": `font-family: Menlo, Consolas, monospace;` +
				` background: none; padding: 0; color: #e2e8f0;`,
			"code":   `font-family: Menlo, Consolas, monospace; font-size: .88em; padding: .15em .4em; background: #eff6ff; border-radius: 3px; color: #1e6fd9;`,
			"th":     `border: 1px solid #cfe0f7; padding: .4em .6em; background: #eff6ff; font-weight: 700;`,
			"td":     `border: 1px solid #cfe0f7; padding: .4em .6em;`,
			"hr":     `margin: 2em 0; border: 0; border-top: 1px solid #cfe0f7;`,
			"strong": `font-weight: 700; color: #1e3a8a;`,
		},
	},
	{
		ID:          Amber,
		Name:        "琥珀",
		Version:     builtinVersion,
		Author:      builtinAuthor,
		Description: "暖色：烛光色的纸与字，标题左侧一道琥珀条",
		Style: map[string]string{
			"body": `font-family: -apple-system, BlinkMacSystemFont, 'Helvetica Neue',` +
				` 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif;` +
				` font-size: 15px; line-height: 1.8; color: #4b3b2a;` +
				` letter-spacing: .5px; word-break: break-word;` +
				` padding: 4px 2px;`,
			"h1": `margin: 1.4em 0 1em; font-size: 22px; font-weight: 700;` +
				` line-height: 1.4; text-align: center; color: #92400e;`,
			"h2": `margin: 1.6em 0 .8em; padding-left: .6em; font-size: 18px;` +
				` font-weight: 700; color: #92400e;` +
				` border-left: 6px solid #f59e0b;`,
			"h3": `margin: 1.4em 0 .8em; font-size: 17px; font-weight: 700;` +
				` color: #b45309;`,
			"blockquote": `margin: 1.2em 0; padding: .7em 1em;` +
				` border-left: 4px solid #f59e0b; background: #fffbeb;` +
				` color: #7c5c34;`,
			"a": `color: #d97706; text-decoration: none; font-weight: 500;`,
			"pre": `margin: 0 0 1.2em; padding: 1em 1.1em; background: #fdf6e3;` +
				` border-radius: 6px; font-size: 13.5px; line-height: 1.65;` +
				` white-space: pre-wrap; word-break: break-word;`,
			"code":   `font-family: Menlo, Consolas, monospace; font-size: .88em; padding: .15em .4em; background: #fdf0d5; border-radius: 3px; color: #b45309;`,
			"th":     `border: 1px solid #f3e0bb; padding: .4em .6em; background: #fdf6e3; font-weight: 700;`,
			"td":     `border: 1px solid #f3e0bb; padding: .4em .6em;`,
			"hr":     `margin: 2em 0; border: 0; border-top: 1px solid #f3e0bb;`,
			"strong": `font-weight: 700; color: #92400e;`,
		},
	},
	{
		ID:          Rose,
		Name:        "胭脂",
		Version:     builtinVersion,
		Author:      builtinAuthor,
		Description: "胭脂：标题与强调都是玫红，分隔线为虚线",
		Style: map[string]string{
			"h1": `margin: 1.4em 0 1em; font-size: 22px; font-weight: 700;` +
				` line-height: 1.4; text-align: center; color: #e11d48;`,
			"h2": `margin: 1.6em 0 .8em; font-size: 18px; font-weight: 700;` +
				` color: #e11d48; border-bottom: 1px dashed #f9a8bf;` +
				` padding-bottom: .35em;`,
			"h3": `margin: 1.4em 0 .8em; font-size: 17px; font-weight: 700;` +
				` color: #be123c;`,
			"blockquote": `margin: 1.2em 0; padding: .7em 1em;` +
				` border-left: 4px solid #e11d48; background: #fff1f4;` +
				` color: #7a4453;`,
			"a":      `color: #e11d48; text-decoration: none; font-weight: 500;`,
			"code":   `font-family: Menlo, Consolas, monospace; font-size: .88em; padding: .15em .4em; background: #fff1f4; border-radius: 3px; color: #e11d48;`,
			"th":     `border: 1px solid #f9c9d6; padding: .4em .6em; background: #fff1f4; font-weight: 700;`,
			"td":     `border: 1px solid #f9c9d6; padding: .4em .6em;`,
			"hr":     `margin: 2em 0; border: 0; border-top: 1px dashed #f9c9d6;`,
			"strong": `font-weight: 700; color: #e11d48;`,
			"img":    `max-width: 100%; height: auto; display: block; margin: .8em auto; border-radius: 8px;`,
		},
	},
}

// Builtin is every style the app was made with, each with the look of the
// base where it says nothing of its own.
func Builtin() []Theme {
	out := make([]Theme, 0, len(builtin))
	for _, t := range builtin {
		out = append(out, t.withBase())
	}
	return out
}

// OfDir is every style the user has put in dir, one to a file of JSON:
// a style of their own making, which is how a style is added without
// touching the app. They are read in the order the folder gives them,
// which is by name.
//
// A file that is no style is said in bad, with what is wrong with it —
// it will not read, it names no id, its id is one another file has
// taken, or it extends a style that is not there — and the rest are
// given all the same. A folder that is not there is no styles, and not
// a fault: the user has made none yet.
func OfDir(dir string) (themes []Theme, bad []string) {
	if dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []string{dir + ": " + err.Error()}
	}
	// What has been read so far, the app's own styles first: a style
	// that extends another is given the looks of one read before it, so
	// that no two of them may extend each other.
	read := Builtin()
	var files []Theme
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			bad = append(bad, e.Name()+": "+err.Error())
			continue
		}
		var t Theme
		if err := json.Unmarshal(data, &t); err != nil {
			bad = append(bad, e.Name()+": "+err.Error())
			continue
		}
		t.ID = strings.TrimSpace(t.ID)
		t.Extends = strings.TrimSpace(t.Extends)
		if t.ID == "" {
			bad = append(bad, e.Name()+": "+errNoID)
			continue
		}
		if t.Name == "" {
			t.Name = t.ID
		}
		t.file = e.Name()
		if Has(read, t.ID) || Has(files, t.ID) {
			bad = append(bad, e.Name()+": "+errIDTaken+t.ID)
			continue
		}
		files = append(files, t)
	}
	for _, t := range files {
		// Extends is kept as the file wrote it, found or not: it says
		// what the style was made over, which is what the settings tell
		// of it. The looks are what may come of it going wrong.
		if t.Extends != "" && !Has(read, t.Extends) {
			bad = append(bad, t.file+": "+errNoExtends+t.Extends)
		}
		t = t.withLooks(read)
		read = append(read, t)
		themes = append(themes, t)
	}
	return themes, bad
}

// What is wrong with a file of a folder that is no style.
const (
	errNoID      = "no id"
	errIDTaken   = "the id is taken: "
	errNoExtends = "extends a style that is not there: "
)

// Themes is every style the menu shows: the ones the app was made with,
// and then the ones the user brought.
func Themes(dir string) []Theme {
	mine, _ := OfDir(dir)
	return append(Builtin(), mine...)
}

// Get is the style an id names out of themes: one of them, or, for a name
// that is none of theirs — a style the user has taken away — the first,
// which is the one a first start has.
func Get(themes []Theme, id string) Theme {
	for _, t := range themes {
		if t.ID == id {
			return t
		}
	}
	if len(themes) == 0 {
		return Theme{Style: base}
	}
	return themes[0]
}

// Has is whether an id is one of the styles.
func Has(themes []Theme, id string) bool {
	for _, t := range themes {
		if t.ID == id {
			return true
		}
	}
	return false
}

// withBase is the style with the look of the base where it says nothing of
// its own: a new style is a handful of lines, and not a whole sheet.
func (t Theme) withBase() Theme {
	return t.over(base)
}

// withLooks is the style with the looks of the one it extends under its
// own: what it says of a part is kept, and what it says nothing of is
// the other's. One that extends none is given the base's.
//
// A style may extend one of the app's own, or one read before it in the
// same folder, which is what keeps two of them from extending each
// other.
func (t Theme) withLooks(read []Theme) Theme {
	for _, from := range read {
		if from.ID == t.Extends {
			return t.over(from.Style)
		}
	}
	return t.withBase()
}

// over is the style with the looks of another under its own: its own
// where it says one, and the other's where it says none.
func (t Theme) over(under map[string]string) Theme {
	style := make(map[string]string, len(under)+len(t.Style))
	for k, v := range under {
		style[k] = v
	}
	for k, v := range t.Style {
		style[k] = v
	}
	t.Style = style
	return t
}
