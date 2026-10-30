// Package search is the app looking for a few words in everything the
// vault holds.
//
// It keeps no copy of what the notes say: the index holds what they say
// about themselves, and the text itself is on the disk, so a search
// reads the disk. That costs a reading of every note, and buys this:
// what is found is always what is there.
//
// It is a plain reading, and so it may be given to a background goroutine
// while the window goes on being drawn.
package search

import (
	"os"
	"sort"
	"strings"
)

// What a search holds to: no more than this many hits are worth showing,
// a file of this size is left out rather than read line by line, and a
// line longer than this is cut to a beginning and an end.
const (
	MaxHits     = 100
	MaxFileSize = 4 * 1024 * 1024
	MaxLine     = 160
)

// Hit is one place where the words were found: the note, the line, and
// the line itself, which is what the window shows as the context.
type Hit struct {
	Path string
	// Line is 0-based, as the editor counts them.
	Line int
	Text string
}

// Files looks for query in each of the files, line by line, the case of
// the letters making no difference.
//
// A file that cannot be read — one that is gone, or locked, or not made
// of words — is passed over: a search should not fail because one file
// would not open. A file of more than MaxFileSize is not read at all.
func Files(paths []string, query string, limit int) []Hit {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	needle := strings.ToLower(query)
	if limit <= 0 || limit > MaxHits {
		limit = MaxHits
	}

	hits := make([]Hit, 0, limit)
	for _, path := range paths {
		if len(hits) >= limit {
			break
		}
		if info, err := os.Stat(path); err != nil ||
			info.Size() > MaxFileSize {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for line, text := range strings.Split(string(data), "\n") {
			if !strings.Contains(strings.ToLower(text), needle) {
				continue
			}
			hits = append(hits, Hit{Path: path, Line: line,
				Text: TrimContext(text)})
			if len(hits) >= limit {
				break
			}
		}
	}
	return hits
}

// Sortable puts the hits in the order they are shown: by the note, then
// by the line within it. A search of many files may find them in
// whatever order the reading gave them.
func Sortable(hits []Hit) {
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Path != hits[j].Path {
			return hits[i].Path < hits[j].Path
		}
		return hits[i].Line < hits[j].Line
	})
}

// TrimContext is a line of the note as it is shown beside a hit: a short
// one as it is, and a long one cut to a beginning and an end, for what
// is looked for is in the middle and the rest is only to read by.
func TrimContext(line string) string {
	chars := []rune(line)
	if len(chars) <= MaxLine {
		return strings.TrimSpace(line)
	}
	return string(chars[:80]) + "…" + string(chars[len(chars)-60:])
}
