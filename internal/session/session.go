// Package session holds the rules of the editor's tabs: which notes are
// open, in which order, and which one is active. It keeps no file of its
// own — the notes that are open are `state`'s to write — and is rules
// over paths alone.
//
// They are rules over paths alone, with no window and no application, so
// they can be tested on their own, and so that opening a note at start-up
// is the same decision as opening one by hand.
package session

import (
	"os"
	"path/filepath"
	"strings"
)

// TabKey is the identity of a tab: the file's canonical path, so that one
// file reached by two spellings, or through a link, is one tab and not
// two. A file that is gone keeps the path it was opened by.
func TabKey(path string) string {
	if canonical, err := filepath.EvalSymlinks(path); err == nil {
		return canonical
	}
	return path
}

// DedupPaths removes a path that is already in the list, keeping the
// order in which each was first opened.
func DedupPaths(paths []string) []string {
	seen := make(map[string]bool, len(paths))
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		key := TabKey(p)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, p)
	}
	return out
}

// PartitionExisting splits paths into the ones that are files now and the
// ones that are not: a note deleted outside the app must not come back
// when the app starts.
func PartitionExisting(paths []string) (existing, missing []string) {
	existing = make([]string, 0, len(paths))
	missing = make([]string, 0)
	for _, p := range paths {
		if IsExisting(p) {
			existing = append(existing, p)
			continue
		}
		missing = append(missing, p)
	}
	return existing, missing
}

// IsExisting reports whether path is a file that is there now.
func IsExisting(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// ResolveActive picks the active path: the one asked for, when it is one
// of the paths; the last one, when it is not, as xink falls back.
func ResolveActive(paths []string, active string) string {
	if active == "" {
		if len(paths) == 0 {
			return ""
		}
		return paths[len(paths)-1]
	}
	key := TabKey(active)
	for _, p := range paths {
		if TabKey(p) == key {
			return p
		}
	}
	if len(paths) == 0 {
		return ""
	}
	return paths[len(paths)-1]
}

// Within reports whether a path belongs to the vault at root, so that
// switching vaults can close what is not theirs. The vault's own root is
// within it; a path that goes up out of it is not.
func Within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return !filepath.IsAbs(rel)
}
