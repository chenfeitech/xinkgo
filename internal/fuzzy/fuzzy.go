// Package fuzzy is the way the app finds what the user meant by a few
// letters: the command palette, and the completion of a link.
//
// A name is found when its letters were typed, in that order, with other
// letters between them. Every name in a vault is not many, so what is
// looked for is a good order rather than a clever one: letters standing
// together are worth more than letters far apart, and a letter at the
// start of a word — after a slash, a dot or a hyphen, or where a small
// letter turns to a capital — is worth more than one in the middle.
package fuzzy

import (
	"sort"
	"strings"
	"unicode"
)

// How much each thing is worth. They only matter beside one another.
const (
	// hit is a letter that was found.
	hit = 10
	// together is a letter found right after another one found: what
	// was typed stands in the name as it was typed.
	together = 8
	// atWord is a letter found where a word begins.
	atWord = 6
	// atStart is the very first letter of the name.
	atStart = 4
	// prefix is the whole of what was typed standing at the front.
	prefix = 12
	// long is taken off for every eight letters of the name, so that
	// a short name that fits is not beaten by a long one that fits too.
	long = 8
)

// separators are what ends a word in a name of a note or a path.
const separators = " ./-\\_:"

// Result is one name a query was found in, how well it fitted, and
// where the letters were found — the last so that they may be lit up in
// what the window shows.
type Result struct {
	Text      string
	Score     int
	Positions []int
}

// Found is whether the query was found in the name at all.
func (r Result) Found() bool { return len(r.Positions) > 0 }

// Match looks for query in text, letter by letter, each after the one
// before it.
func Match(text, query string) Result {
	if query == "" {
		return Result{Text: text}
	}
	chars := []rune(strings.ToLower(text))
	want := []rune(strings.ToLower(query))
	lower := []rune(text)

	positions := make([]int, 0, len(want))
	score := 0
	found := 0
	for i := range chars {
		if found >= len(want) {
			break
		}
		if chars[i] != want[found] {
			continue
		}
		// What this letter is worth: it was found, and perhaps beside
		// the one before it, and perhaps where a word begins.
		score += hit
		if len(positions) > 0 && positions[len(positions)-1] == i-1 {
			score += together
		}
		if i == 0 {
			score += atStart
		} else if strings.ContainsRune(separators, lower[i-1]) ||
			(unicode.IsLower(lower[i-1]) && unicode.IsUpper(lower[i])) {
			score += atWord
		}
		positions = append(positions, i)
		found++
	}
	if found < len(want) {
		return Result{Text: text}
	}
	if strings.HasPrefix(strings.ToLower(text), strings.ToLower(query)) {
		score += prefix
	}
	score -= len(chars) / long
	return Result{Text: text, Score: score, Positions: positions}
}

// Search is what was looked for among many names: the best first, the
// names in order among equals, and no more than limit of them. A query
// of nothing gives the names as they are, for there is nothing to
// choose between them yet.
func Search(names []string, query string, limit int) []Result {
	out := make([]Result, 0, len(names))
	for _, name := range names {
		if r := Match(name, query); query == "" || r.Found() {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Text < out[j].Text
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// Segments breaks a name into what was found and what was not, so that
// the window may light up the one and not the other.
func Segments(text string, positions []int) []Segment {
	if len(positions) == 0 {
		if text == "" {
			return nil
		}
		return []Segment{{Text: text}}
	}
	chars := []rune(text)
	if len(chars) == 0 {
		return nil
	}
	marked := make([]bool, len(chars))
	for _, i := range positions {
		if i >= 0 && i < len(chars) {
			marked[i] = true
		}
	}
	var out []Segment
	start := 0
	for i := 1; i <= len(chars); i++ {
		if i < len(chars) && marked[i] == marked[start] {
			continue
		}
		out = append(out, Segment{Text: string(chars[start:i]),
			Found: marked[start]})
		start = i
	}
	return out
}

// Segment is a piece of a name: some of its letters, and whether they
// are ones the query was found in.
type Segment struct {
	Text  string
	Found bool
}
