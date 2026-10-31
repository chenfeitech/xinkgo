package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/index"
	"github.com/helightxu/xinkgo/internal/search"
	"github.com/helightxu/xinkgo/internal/skill"
)

// vaultHost is what a tool may see of the vault.
//
// It is made on the main thread, of a snapshot of what is known, and
// given to the loop to read: nothing it holds is being written while
// the loop reads it.
type vaultHost struct {
	root   string
	idx    *index.Index
	skills []skill.Skill
}

// host is what the tools are given to look through. It is made now, of
// what is known now.
func (s *appState) host() ai.Host {
	return vaultHost{
		root:   s.root,
		idx:    s.idx,
		skills: s.skills,
	}
}

// ReadNote is a note's text, read off the disk: the disk is the truth of
// a note, and not the index, which is only made of it.
//
// A note is read from within the vault and nowhere else. A tool is asked
// for a path by a model, and a model is not to be given the run of the
// machine.
func (h vaultHost) ReadNote(path string) (string, bool) {
	resolved, ok := h.within(path)
	if !ok {
		return "", false
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", false
	}
	return string(data), true
}

// within is the path as it stands inside the vault, and whether it does
// stand there. A path that reaches out of the vault — one beginning with
// dots, or an absolute one pointing elsewhere — is not one a tool is
// given.
func (h vaultHost) within(path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", false
	}
	resolved := path
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(h.root, resolved)
	}
	resolved = filepath.Clean(resolved)
	if h.root == "" {
		return resolved, true
	}
	rel, err := filepath.Rel(h.root, resolved)
	if err != nil || rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return resolved, true
}

// Search is the lines the words asked for stand in.
func (h vaultHost) Search(query string, limit int) []search.Hit {
	if h.idx == nil {
		return nil
	}
	return search.Files(h.idx.Paths(), query, limit)
}

// Tags is every tag in the vault, and how many notes use it.
func (h vaultHost) Tags() []index.TagCount {
	if h.idx == nil {
		return nil
	}
	return h.idx.Tags()
}

// Backlinks is where a note is pointed at from.
func (h vaultHost) Backlinks(path string) []index.LinkRef {
	if h.idx == nil {
		return nil
	}
	resolved, ok := h.within(path)
	if !ok {
		return nil
	}
	return h.idx.Backlinks(resolved)
}

// Outline is the headings of a note.
func (h vaultHost) Outline(path string) []string {
	if h.idx == nil {
		return nil
	}
	resolved, ok := h.within(path)
	if !ok {
		return nil
	}
	headings := h.idx.Headings(resolved)
	out := make([]string, 0, len(headings))
	for _, heading := range headings {
		out = append(out, strings.Repeat("  ", heading.Level)+
			heading.Text)
	}
	return out
}

// ListNotes is the notes of the vault, with what they are called.
func (h vaultHost) ListNotes(limit int) []ai.NoteRef {
	if h.idx == nil {
		return nil
	}
	paths := h.idx.Paths()
	if limit > 0 && limit < len(paths) {
		paths = paths[:limit]
	}
	out := make([]ai.NoteRef, 0, len(paths))
	for _, path := range paths {
		out = append(out, ai.NoteRef{
			Path: path, Title: h.idx.Title(path),
		})
	}
	return out
}

// Root is the vault's folder.
func (h vaultHost) Root() string { return h.root }

// Skill is the skill of that name, when there is one.
func (h vaultHost) Skill(name string) (*skill.Skill, bool) {
	// Find gives back nothing when there is no such skill, so it is
	// asked whether it found one.
	one := skill.Find(h.skills, name)
	return one, one != nil
}
