package wechat

import (
	"strings"
)

// styled is the HTML the parser wrote with the look of every part put
// into the part itself: the editor of WeChat keeps nothing of a style
// sheet, so an article has to carry its own looks in its tags.
//
// A part is looked up by the kind of it, and by the kind of it inside the
// kind that holds it — "p", and then "blockquote p" — so that a style may
// say of a paragraph in a quote what it does not say of a paragraph at
// large. What a style says nothing of has no look of its own here, and
// keeps what the tag would have had anyway.
func styled(body string, th Theme) string {
	var b strings.Builder
	b.Grow(len(body) * 2)
	// The kinds of the parts the tag at hand stands in, the nearest
	// first, as a paragraph in a quote stands in "blockquote".
	var open []string
	for at := 0; at < len(body); {
		i := strings.IndexByte(body[at:], '<')
		if i < 0 {
			b.WriteString(body[at:])
			break
		}
		i += at
		b.WriteString(body[at:i])
		// A comment is not a part of the article, and is left behind.
		if strings.HasPrefix(body[i:], "<!--") {
			end := strings.Index(body[i+4:], "-->")
			if end < 0 {
				break
			}
			at = i + 4 + end + 3
			continue
		}
		end := tagEnd(body, i)
		if end < 0 {
			b.WriteString(body[i:])
			break
		}
		tag := body[i : end+1]
		name, attrs, closing, alone := splitTag(tag)
		at = end + 1
		switch {
		case closing:
			closeOf(&open, name)
			b.WriteString(tag)
		case name == "script" || name == "style":
			// A style sheet and a script are two things the editor of
			// WeChat throws away, along with what they hold.
			at = skipThrough(body, at, name)
		case name == "input":
			// A task's box is an <input>, which the editor of WeChat
			// drops: the box is written as the character it looks
			// like, and the task stands as it did.
			b.WriteString(boxOf(attrs))
		default:
			b.WriteString(withStyle(name, attrs, alone, th.lookOf(name, open)))
			if !alone && !void[name] {
				open = append(open, name)
			}
		}
	}
	// A tag left open, as one of a note's own making may be, is closed
	// so that what follows it is not inside it.
	for i := len(open) - 1; i >= 0; i-- {
		b.WriteString("</" + open[i] + ">")
	}
	return b.String()
}

// lookOf is the look of a part of a kind, as the style writes it: the one
// it gives that kind inside the kind that holds it, and failing that the
// one it gives the kind at large.
func (t Theme) lookOf(name string, open []string) string {
	if len(open) > 0 {
		if s, ok := t.Style[open[len(open)-1]+">"+name]; ok {
			return s
		}
	}
	return t.Style[name]
}

// withStyle is the tag with a look of its own, written first among its
// attributes: what it already says of itself — where a picture is read
// from, what a link points at — is kept as it was.
func withStyle(name, attrs string, alone bool, look string) string {
	var b strings.Builder
	b.WriteString("<" + name)
	if look != "" {
		b.WriteString(` style="` + escapeStyle(look) + `"`)
	}
	if attrs != "" {
		b.WriteString(" " + attrs)
	}
	if alone || void[name] {
		b.WriteString(" />")
	} else {
		b.WriteByte('>')
	}
	return b.String()
}

// escapeStyle is a look written inside a tag: an attribute ends at the
// next quote, so a look that holds one — a family of fonts named in
// quotes — would end the tag before its time.
func escapeStyle(look string) string {
	look = strings.ReplaceAll(look, "&", "&amp;")
	look = strings.ReplaceAll(look, `"`, "&quot;")
	look = strings.ReplaceAll(look, "<", "&lt;")
	return strings.ReplaceAll(look, ">", "&gt;")
}

// boxOf is a task's box as a character: an empty one, or one with a tick
// in it for a task that is done.
func boxOf(attrs string) string {
	if strings.Contains(attrs, "checked") {
		return "☑"
	}
	return "☐"
}

// splitTag is what a tag is made of: the kind of the part it opens, what
// else it says, whether it closes one instead, and whether it stands on
// its own, as `<br />` does.
func splitTag(tag string) (name, attrs string, closing, alone bool) {
	inner := strings.TrimSuffix(strings.TrimPrefix(tag, "<"), ">")
	if strings.HasPrefix(inner, "/") {
		return strings.TrimSpace(strings.TrimPrefix(inner, "/")), "", true, false
	}
	inner = strings.TrimSpace(inner)
	if strings.HasSuffix(inner, "/") {
		inner, alone = strings.TrimSpace(strings.TrimSuffix(inner, "/")), true
	}
	if i := strings.IndexAny(inner, " \t\n"); i >= 0 {
		return inner[:i], strings.TrimSpace(inner[i+1:]), false, alone
	}
	return inner, "", false, alone
}

// tagEnd is where the tag begun at i ends: the '>' that closes it, which
// is not one that stands inside a quote, as in the title of a picture.
func tagEnd(s string, i int) int {
	for j := i + 1; j < len(s); j++ {
		switch s[j] {
		case '"':
			if k := strings.IndexByte(s[j+1:], '"'); k >= 0 {
				j += k + 1
			}
		case '>':
			return j
		}
	}
	return -1
}

// skipThrough is the place in s after the tag that closes the one of a
// kind opened before at: what a style sheet or a script holds.
func skipThrough(s string, at int, name string) int {
	if i := strings.Index(s[at:], "</"+name); i >= 0 {
		if end := strings.IndexByte(s[at+i:], '>'); end >= 0 {
			return at + i + end + 1
		}
	}
	return len(s)
}

// closeOf takes the kind of the part that has just ended off the ones the
// next tag stands in: an end tag that closes nothing, as one of a note's
// own making may, leaves them as they are.
func closeOf(open *[]string, name string) {
	for i := len(*open) - 1; i >= 0; i-- {
		if (*open)[i] == name {
			*open = (*open)[:i]
			return
		}
	}
}

// void is the kinds of part that hold nothing: ones that stand on their
// own, and never have an end tag to them.
var void = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}
