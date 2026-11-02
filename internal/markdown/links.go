package markdown

import (
	"net/url"
	"strings"
)

// WikiScheme is how a link to a note of the vault is written in what the
// preview is given: a click on it is then known for what it is, and is
// not taken for a page on the web.
const WikiScheme = "xinkgo:"

// WikiHref is where a link to what `target` names points, for the
// preview. The target is written as a path would be, so that a name with
// a space in it is still one link.
func WikiHref(target string) string {
	return WikiScheme + (&url.URL{Path: target}).String()
}

// WikiTarget is what such a destination names, and whether it is one at
// all. A destination may reach here already undone by what read it;
// undoing it again leaves it as it stands.
func WikiTarget(href string) (string, bool) {
	if !strings.HasPrefix(href, WikiScheme) {
		return "", false
	}
	target := strings.TrimPrefix(href, WikiScheme)
	if plain, err := url.PathUnescape(target); err == nil {
		target = plain
	}
	if target == "" {
		return "", false
	}
	return target, true
}

// Links is a note written out to be shown: every link in double brackets
// that names a note of the vault is made into one that can be clicked.
//
// One that names nothing is left as it was written. A vault may well not
// have the note yet — a draft is allowed to point at what is still to be
// written — and sending the reader to a note that was only guessed at is
// worse than leaving the words as they are.
//
// What stands in a fence of code, or between backticks on a line, is left
// alone: `[[x]]` there is code, and not a link.
func Links(text string, resolve func(target string) (string, bool)) string {
	if resolve == nil || !strings.Contains(text, "[[") {
		return text
	}
	lines := strings.Split(text, "\n")
	fence := ""
	inFence := false
	for i, line := range lines {
		if marker, ok := fenceMarker(line); ok {
			switch {
			case !inFence:
				fence, inFence = marker, true
			case fence == marker:
				fence, inFence = "", false
			}
			continue
		}
		if inFence {
			continue
		}
		lines[i] = linkedLine(line, resolve)
	}
	return strings.Join(lines, "\n")
}

// linkedLine is one line written out: what its double brackets name made
// clickable, and its code left as it stands.
func linkedLine(
	line string, resolve func(target string) (string, bool),
) string {
	kept := []rune(line)
	// What stands between backticks is not looked at, and keeps its
	// place, so that the two read the same way along.
	masked := []rune(maskInlineCode(line))
	var out strings.Builder
	for i := 0; i < len(masked); {
		if i+1 >= len(masked) || masked[i] != '[' || masked[i+1] != '[' {
			out.WriteRune(kept[i])
			i++
			continue
		}
		end := closingBrackets(masked, i+2)
		if end < 0 {
			// No closing brackets: the rest of the line is no
			// link at all.
			out.WriteString(string(kept[i:]))
			return out.String()
		}
		link, ok := parseWikiInner(string(masked[i+2:end]), 0)
		if ok {
			if _, found := resolve(link.Target); found {
				out.WriteString(written(link))
				i = end + 2
				continue
			}
		}
		out.WriteString(string(kept[i : end+2]))
		i = end + 2
	}
	return out.String()
}

// closingBrackets is where the next pair of closing brackets stands at or
// after from, counted in runes: the line is held in runes, and what a
// count of bytes would give — as strings.Index does — runs past the end
// of a line whose characters take more than a byte each, as Chinese ones
// do, and then reads what is not there.
func closingBrackets(chars []rune, from int) int {
	for i := from; i+1 < len(chars); i++ {
		if chars[i] == ']' && chars[i+1] == ']' {
			return i
		}
	}
	return -1
}

// written is a link in double brackets as the preview is given it: the
// words the reader sees are the alias that was written, or else what the
// link names.
func written(l WikiLink) string {
	seen := l.Target
	if l.Anchor != "" {
		seen += "#" + l.Anchor
	}
	if l.Alias != "" {
		seen = l.Alias
	}
	return "[" + seen + "](" + WikiHref(l.Target) + ")"
}
