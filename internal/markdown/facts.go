// Facts: what a note says about itself.
//
// A note is read for its headings, its links, its double brackets and
// its tags, and where in the note each of them stands. The outline, the
// back-links, the tags and the index all read the same facts, so they
// are taken out in one place, and never by a renderer: what is inside a
// code block or between backticks is not a fact about the note, it is
// text the note is showing.
//
// Line numbers are 0-based, as the editor counts them.
package markdown

import (
	"strings"
	"unicode"
)

// DocFacts is what one note holds.
type DocFacts struct {
	Headings  []Heading
	Links     []Link
	WikiLinks []WikiLink
	Tags      []Tag
}

// Heading is an ATX heading: how deep it is, what it says, where it is.
type Heading struct {
	Level int // 1–6
	Text  string
	Line  int
}

// Link is an inline link, an image among them: where it points, what it
// reads, and where it stands.
type Link struct {
	Target string
	Text   string
	Line   int
}

// WikiLink is a link in double brackets: `[[target]]`, with an alias
// after `|` and a heading of the target after `#`, either of which may
// be absent.
type WikiLink struct {
	Target string
	Alias  string // empty when there is none
	Anchor string // empty when there is none
	Line   int
}

// Tag is a `#tag` of the note's own, or one listed in its frontmatter.
// The name is kept without the leading `#`.
type Tag struct {
	Name string
	Line int
}

// ParseFacts reads a note and takes out what it says about itself.
//
// It walks the lines once, keeping two things in mind: whether it is
// inside a fenced block of code, and whether it is inside the
// frontmatter at the top. What stands between backticks is masked out —
// made spaces of the same length, so that what follows keeps its place —
// before links, double brackets and tags are looked for. Headings are
// read from the line as it is, for a heading cannot be inside code.
func ParseFacts(text string) DocFacts {
	var facts DocFacts
	fence := ""
	inFence := false
	inFrontmatter := false

	for lineNo, raw := range strings.Split(text, "\n") {
		// The frontmatter is only at the top, fenced by `---`.
		if lineNo == 0 {
			if strings.TrimSpace(raw) == "---" {
				inFrontmatter = true
				continue
			}
		} else if inFrontmatter {
			if strings.TrimSpace(raw) == "---" {
				inFrontmatter = false
			} else {
				collectFrontmatterTags(raw, lineNo, &facts.Tags)
			}
			continue
		}

		if marker, ok := fenceMarker(raw); ok {
			switch {
			case !inFence:
				fence, inFence = marker, true
			case fence == marker:
				fence, inFence = "", false
			}
			// The fence's own line is never a fact either.
			continue
		}
		if inFence {
			continue
		}

		if heading, ok := parseHeading(raw, lineNo); ok {
			facts.Headings = append(facts.Headings, heading)
		}
		masked := maskInlineCode(raw)
		collectLinks(masked, lineNo, &facts.Links)
		collectWikiLinks(masked, lineNo, &facts.WikiLinks)
		collectTags(masked, lineNo, &facts.Tags)
	}
	return facts
}

// fenceMarker is the fence a line opens or closes, if any: a run of
// three backticks or three tildes at the start of the line, and nothing
// else.
func fenceMarker(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	runes := []rune(trimmed)
	if len(runes) < 3 {
		return "", false
	}
	switch marker := string(runes[:3]); marker {
	case "```", "~~~":
		return marker, true
	default:
		return "", false
	}
}

// parseHeading reads an ATX heading: how many marks, then a space, then
// the text, whose closing marks are not part of it. `#tag` is no
// heading; it may well be a tag.
func parseHeading(line string, lineNo int) (Heading, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level > 6 {
		return Heading{}, false
	}
	rest := trimmed[level:]
	if rest != "" && !strings.HasPrefix(rest, " ") {
		return Heading{}, false
	}
	text := strings.Trim(strings.TrimRight(strings.TrimSpace(rest), "#"),
		" \t")
	if text == "" {
		return Heading{}, false
	}
	return Heading{Level: level, Text: text, Line: lineNo}, true
}

// maskInlineCode makes spaces of what stands between backticks, of the
// same length, so that the rest of the line keeps its place. An opening
// run is closed by a run of the same length; one that is never closed
// masks the rest of the line.
func maskInlineCode(line string) string {
	chars := []rune(line)
	for i := 0; i < len(chars); i++ {
		if chars[i] != '`' {
			continue
		}
		opening := runLength(chars, i, '`')
		closed := false
		for j := i + opening; j < len(chars); {
			if chars[j] != '`' {
				j++
				continue
			}
			closing := runLength(chars, j, '`')
			if closing == opening {
				for k := i; k < j+closing; k++ {
					chars[k] = ' '
				}
				i = j + closing - 1
				closed = true
				break
			}
			j += closing
		}
		if !closed {
			for k := i; k < len(chars); k++ {
				chars[k] = ' '
			}
			break
		}
	}
	return string(chars)
}

// runLength is how many of the same rune stand from here on.
func runLength(chars []rune, from int, c rune) int {
	n := 0
	for from+n < len(chars) && chars[from+n] == c {
		n++
	}
	return n
}

// collectLinks finds `[text](target)` and `![text](target)`, the target
// being what stands before any title in quotes, without angle brackets.
func collectLinks(line string, lineNo int, out *[]Link) {
	for i := 0; i < len(line); {
		open := -1
		switch {
		case line[i] == '[':
			open = i
		case line[i] == '!' && i+1 < len(line) && line[i+1] == '[':
			open = i + 1
		default:
			i++
			continue
		}
		closeBracket := strings.IndexByte(line[open+1:], ']')
		if closeBracket < 0 {
			i++
			continue
		}
		closeBracket += open + 1
		if closeBracket+1 >= len(line) || line[closeBracket+1] != '(' {
			i++
			continue
		}
		end := strings.IndexByte(line[closeBracket+2:], ')')
		if end < 0 {
			i++
			continue
		}
		end += closeBracket + 2

		target := line[closeBracket+2 : end]
		if fields := strings.Fields(target); len(fields) > 0 {
			target = fields[0]
		}
		target = strings.Trim(target, "<>")
		if target != "" {
			*out = append(*out, Link{
				Target: target,
				Text:   strings.TrimSpace(line[open+1 : closeBracket]),
				Line:   lineNo,
			})
		}
		i = end + 1
	}
}

// collectWikiLinks finds `[[target]]`, and reads its alias and its
// anchor. A pair of brackets with nothing in it is no link at all.
func collectWikiLinks(line string, lineNo int, out *[]WikiLink) {
	for i := 0; i+1 < len(line); {
		if line[i] != '[' || line[i+1] != '[' {
			i++
			continue
		}
		end := strings.Index(line[i+2:], "]]")
		if end < 0 {
			// No closing brackets: the rest of the line is no link.
			break
		}
		end += i + 2
		if link, ok := parseWikiInner(line[i+2:end], lineNo); ok {
			*out = append(*out, link)
		}
		i = end + 2
	}
}

// parseWikiInner is what stands inside the brackets: the target, and
// after a `|` the alias and after a `#` the heading it points at. The
// alias is cut off first, so that `[[a#b|c]]` reads as it looks.
func parseWikiInner(inner string, lineNo int) (WikiLink, bool) {
	left, alias := inner, ""
	if i := strings.Index(inner, "|"); i >= 0 {
		left, alias = inner[:i], strings.TrimSpace(inner[i+1:])
	}
	target, anchor := left, ""
	if i := strings.Index(left, "#"); i >= 0 {
		target, anchor = left[:i], strings.TrimSpace(left[i+1:])
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return WikiLink{}, false
	}
	return WikiLink{Target: target, Alias: alias, Anchor: anchor,
		Line: lineNo}, true
}

// collectTags finds `#tag` in the text. What a tag may hold: letters
// and digits of any language, `_`, `-` and `/` — the last for a tag
// inside a tag — and no tag is made of digits alone.
func collectTags(line string, lineNo int, out *[]Tag) {
	chars := []rune(line)
	for i := 0; i < len(chars); {
		if chars[i] != '#' || !tagCanStartAt(chars, i) {
			i++
			continue
		}
		start := i + 1
		end := start
		for end < len(chars) && isTagChar(chars[end]) {
			end++
		}
		digits := true
		for _, c := range chars[start:end] {
			if !isASCIIDigit(c) {
				digits = false
				break
			}
		}
		if end > start && !digits {
			*out = append(*out, Tag{Name: string(chars[start:end]),
				Line: lineNo})
		}
		i = end
	}
}

// tagCanStartAt says whether a `#` here opens a tag: it does at the
// start of the line, after a space, or after one of a few marks that
// part a word. Without this, `a.md#top` and `#fff` would be tags.
func tagCanStartAt(chars []rune, i int) bool {
	if i == 0 {
		return true
	}
	switch prev := chars[i-1]; {
	case unicode.IsSpace(prev):
		return true
	default:
		return strings.ContainsRune("([{\"'（“，：、", prev)
	}
}

// isTagChar says whether a rune may stand in a tag.
func isTagChar(c rune) bool {
	return unicode.IsLetter(c) || unicode.IsDigit(c) ||
		strings.ContainsRune("_-/", c) || c > unicode.MaxASCII
}

func isASCIIDigit(c rune) bool { return c >= '0' && c <= '9' }

// collectFrontmatterTags reads `tags: [a, b]` — or `tags: a, b`, with or
// without quotes and marks — from a line of the frontmatter.
func collectFrontmatterTags(line string, lineNo int, out *[]Tag) {
	rest := strings.TrimLeft(line, " \t")
	rest, ok := cutPrefix(rest, "tags:")
	if !ok {
		rest, ok = cutPrefix(rest, "tag:")
	}
	if !ok {
		return
	}
	rest = strings.Trim(strings.TrimSpace(rest), "[]")
	for _, part := range strings.Split(rest, ",") {
		name := strings.Trim(part, " \t\"'#")
		if name != "" {
			*out = append(*out, Tag{Name: name, Line: lineNo})
		}
	}
}

// cutPrefix takes prefix off s, as strings.CutPrefix does, kept for the
// Go version this builds with.
func cutPrefix(s, prefix string) (string, bool) {
	if strings.HasPrefix(s, prefix) {
		return s[len(prefix):], true
	}
	return s, false
}
