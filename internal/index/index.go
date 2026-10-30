// Package index is what the app knows about a whole vault: which notes
// there are, what each says, who points at whom, and what is tagged how.
//
// The notes themselves are the truth, and the disk is the truth of the
// notes: the index is worked out again from them, and is never written
// down. What a note holds is read out by markdown.ParseFacts; what is
// kept here is that, and the five tables that are worked out from it so
// that a question about the whole vault is not a walk over every note.
//
// The tables are kept up to date as notes change: a note that changes is
// taken out (detach) and put back (attach), which costs no more than the
// note itself.
package index

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/helightxu/xinkgo/internal/markdown"
)

// Index is what is known about a vault. It may be read from more than
// one goroutine: the window asks it questions while a rebuild walks the
// vault in the background.
type Index struct {
	mu sync.RWMutex

	root string
	// notes is the truth of the index: what each note holds.
	notes map[string]*Note
	// byName is how a note may be written in a link by its name alone.
	byName map[string][]string
	// byRelPath is how it may be written by its path from the root.
	byRelPath map[string]string
	// byTitle is how it may be written by its title.
	byTitle map[string][]string
	// byTag is which notes carry a tag.
	byTag map[string]map[string]bool
	// linkRefs is who wrote what: the bucket of a target holds every
	// place in the vault that named it.
	linkRefs map[string][]LinkRef
}

// Note is one note of the vault, as the index holds it.
type Note struct {
	Path  string
	Title string
	Facts markdown.DocFacts
}

// LinkRef is one place where a note was named in double brackets.
type LinkRef struct {
	From string
	Line int
	// Target is what was written, not what it came to mean: whether it
	// points at anything is asked when it is asked for.
	Target string
}

// Kind is what came of asking what a link points at.
type Kind int

const (
	// Hit is one note, and no other.
	Hit Kind = iota
	// Ambiguous is more than one note of that name: the caller says so,
	// rather than the index picking one.
	Ambiguous
	// Missing is nothing of that name — perhaps a note not yet written.
	Missing
)

// Resolution is what a link in double brackets came to mean.
type Resolution struct {
	Kind Kind
	// Path is the note, when there was exactly one.
	Path string
	// Candidates are the notes, when there was more than one.
	Candidates []string
}

// TagCount is a tag of the vault and how many notes carry it.
type TagCount struct {
	Name  string
	Count int
}

// New is an index that knows nothing yet.
func New() *Index {
	return &Index{
		notes:     map[string]*Note{},
		byName:    map[string][]string{},
		byRelPath: map[string]string{},
		byTitle:   map[string][]string{},
		byTag:     map[string]map[string]bool{},
		linkRefs:  map[string][]LinkRef{},
	}
}

// Root is the folder the index is of.
func (x *Index) Root() string {
	x.mu.RLock()
	defer x.mu.RUnlock()
	return x.root
}

// Paths is every note in the index, sorted.
func (x *Index) Paths() []string {
	x.mu.RLock()
	defer x.mu.RUnlock()
	out := make([]string, 0, len(x.notes))
	for path := range x.notes {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

// Note is what the index holds of the note at path, or nil.
func (x *Index) Note(path string) *Note {
	x.mu.RLock()
	defer x.mu.RUnlock()
	return x.notes[path]
}

// Title is how the note at path is named, or its file's name when the
// index holds nothing of it.
func (x *Index) Title(path string) string {
	x.mu.RLock()
	defer x.mu.RUnlock()
	if note := x.notes[path]; note != nil {
		return note.Title
	}
	return stem(path)
}

// Headings is the outline of the note at path.
func (x *Index) Headings(path string) []markdown.Heading {
	x.mu.RLock()
	defer x.mu.RUnlock()
	if note := x.notes[path]; note != nil {
		return note.Facts.Headings
	}
	return nil
}

// Build throws away what the index knew and reads the vault again. Only
// Markdown is taken in.
func (x *Index) Build(root string, paths []string) {
	x.mu.Lock()
	x.root = root
	x.notes = map[string]*Note{}
	x.byName = map[string][]string{}
	x.byRelPath = map[string]string{}
	x.byTitle = map[string][]string{}
	x.byTag = map[string]map[string]bool{}
	x.linkRefs = map[string][]LinkRef{}
	x.mu.Unlock()

	for _, path := range paths {
		x.Reindex(path)
	}
}

// Reindex reads the note at path again and puts what it holds in the
// index. A file that is gone is taken out of it, and one that is no
// note — or that cannot be read — is left alone.
func (x *Index) Reindex(path string) {
	if !IsNote(path) {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		x.Remove(path)
		return
	}
	x.Put(path, string(data))
}

// Put is what the note at path holds now, in text: it is read for its
// facts and put in the index, any older reading of it being taken out
// first.
func (x *Index) Put(path, text string) {
	if !IsNote(path) {
		return
	}
	facts := markdown.ParseFacts(text)
	note := &Note{Path: path, Title: titleOf(facts, path), Facts: facts}

	x.mu.Lock()
	defer x.mu.Unlock()
	x.detach(path)
	x.attachKeys(note)
	x.attachLinks(path)
}

// Remove takes the note at path out of the index.
func (x *Index) Remove(path string) {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.detach(path)
}

// Backlinks are the places in the vault that named the note at path,
// its own naming of itself left out, in the order of the notes they
// stand in and of the lines within them.
func (x *Index) Backlinks(path string) []LinkRef {
	x.mu.RLock()
	defer x.mu.RUnlock()
	if x.root == "" {
		return nil
	}
	// The ways this note may have been written; each is a bucket of
	// the places that wrote it that way.
	keys := linkTargetsOf(path, x.root)
	sort.Strings(keys)
	var refs []LinkRef
	for i, key := range keys {
		if i > 0 && keys[i-1] == key {
			continue
		}
		for _, ref := range x.linkRefs[key] {
			if ref.From == path {
				// Naming yourself is no back-link.
				continue
			}
			refs = append(refs, ref)
		}
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].From != refs[j].From {
			return refs[i].From < refs[j].From
		}
		return refs[i].Line < refs[j].Line
	})
	return refs
}

// Tags are the tags of the vault and how many notes carry each, the
// fullest first and the names in order among equals.
func (x *Index) Tags() []TagCount {
	x.mu.RLock()
	defer x.mu.RUnlock()
	out := make([]TagCount, 0, len(x.byTag))
	for name, paths := range x.byTag {
		out = append(out, TagCount{Name: name, Count: len(paths)})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// NotesWithTag are the notes carrying tag, sorted. The tag may be
// written with or without its mark, in any case of its letters.
func (x *Index) NotesWithTag(tag string) []string {
	x.mu.RLock()
	defer x.mu.RUnlock()
	paths := make([]string, 0)
	for path := range x.byTag[tagKey(tag)] {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

// Resolve is what a link in double brackets came to mean: the note of
// that path from the root, or of that name, or of that title — the
// letters' case making no difference. More than one note of the name is
// given back as it is, for the caller to say which.
func (x *Index) Resolve(target string) Resolution {
	x.mu.RLock()
	defer x.mu.RUnlock()
	return resolveIn(x.byRelPath, x.byName, x.byTitle, target)
}

// ResolveOnly is the one note a link points at, and whether there was
// one. Where the choice is not clear the answer is no: better to show a
// link as words than to send it to a note that was only guessed.
func (x *Index) ResolveOnly(target string) (string, bool) {
	if r := x.Resolve(target); r.Kind == Hit {
		return r.Path, true
	}
	return "", false
}

// detach takes the note at path out of every table. Which buckets it
// wrote into are worked out from what it held, so that a note whose
// text has changed still lets go of what it wrote before.
func (x *Index) detach(path string) {
	note, ok := x.notes[path]
	if !ok {
		return
	}
	delete(x.notes, path)

	removeFromList(x.byName, strings.ToLower(stem(path)), path)
	if rel, ok := relativeTo(path, x.root); ok {
		delete(x.byRelPath, strings.ToLower(rel))
	}
	removeFromList(x.byTitle, strings.ToLower(note.Title), path)

	for _, tag := range note.Facts.Tags {
		key := tagKey(tag.Name)
		if set := x.byTag[key]; set != nil {
			delete(set, path)
			if len(set) == 0 {
				delete(x.byTag, key)
			}
		}
	}
	for _, link := range note.Facts.WikiLinks {
		key := linkKey(link.Target)
		bucket := x.linkRefs[key]
		kept := bucket[:0]
		for _, ref := range bucket {
			if ref.From != path {
				kept = append(kept, ref)
			}
		}
		if len(kept) == 0 {
			delete(x.linkRefs, key)
		} else {
			x.linkRefs[key] = kept
		}
	}
}

// attachKeys puts the note in the tables its own name, path, title and
// tags are asked by.
func (x *Index) attachKeys(note *Note) {
	path := note.Path
	pushUnique(x.byName, strings.ToLower(stem(path)), path)
	if rel, ok := relativeTo(path, x.root); ok {
		x.byRelPath[strings.ToLower(rel)] = path
	}
	pushUnique(x.byTitle, strings.ToLower(note.Title), path)

	for _, tag := range note.Facts.Tags {
		// `#Rust` and `#rust` are one tag.
		key := tagKey(tag.Name)
		if x.byTag[key] == nil {
			x.byTag[key] = map[string]bool{}
		}
		x.byTag[key][path] = true
	}
	x.notes[path] = note
}

// attachLinks is what the note named in double brackets: it is written
// down as it was written, and not resolved — who it points at is asked
// when it is asked for, so that adding, changing and taking away a note
// all walk the same way.
func (x *Index) attachLinks(path string) {
	note := x.notes[path]
	if note == nil {
		return
	}
	for _, link := range note.Facts.WikiLinks {
		key := linkKey(link.Target)
		x.linkRefs[key] = append(x.linkRefs[key], LinkRef{
			From: path, Line: link.Line, Target: link.Target,
		})
	}
}

// resolveIn is the one rule for what a link means, wherever it is asked:
// by its path from the root, then by its name, then by its title. What
// a background goroutine makes of a link must be the same as what the
// window makes of it.
func resolveIn(
	byRelPath map[string]string,
	byName, byTitle map[string][]string,
	target string,
) Resolution {
	raw := strings.TrimSpace(target)
	if raw == "" {
		return Resolution{Kind: Missing}
	}
	key := strings.ToLower(raw)

	// 1) the path from the root, with or without its extension.
	if path, ok := byRelPath[key]; ok {
		return Resolution{Kind: Hit, Path: path}
	}
	if path, ok := byRelPath[key+".md"]; ok {
		return Resolution{Kind: Hit, Path: path}
	}

	// 2) the name it was given, then 3) its title.
	hits := append([]string{}, byName[trimMarkdown(raw)]...)
	if len(hits) == 0 {
		hits = append(hits, byTitle[trimMarkdown(raw)]...)
	}
	sort.Strings(hits)
	switch len(hits) {
	case 0:
		return Resolution{Kind: Missing}
	case 1:
		return Resolution{Kind: Hit, Path: hits[0]}
	default:
		return Resolution{Kind: Ambiguous, Candidates: hits}
	}
}

// linkTargetsOf is how the note at path may have been written in a link.
func linkTargetsOf(path, root string) []string {
	var out []string
	if s := stem(path); s != "" {
		out = append(out, strings.ToLower(s))
	}
	if rel, ok := relativeTo(path, root); ok {
		rel = strings.ToLower(rel)
		out = append(out, rel, trimMarkdown(rel))
	}
	return out
}

// titleOf is how a note is named: its first heading of the first level,
// or else its first heading, or else the name of its file.
func titleOf(facts markdown.DocFacts, path string) string {
	for _, h := range facts.Headings {
		if h.Level == 1 {
			return h.Text
		}
	}
	if len(facts.Headings) > 0 {
		return facts.Headings[0].Text
	}
	return stem(path)
}

// IsNote is whether a file is taken into the index: only Markdown is.
func IsNote(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".md")
}

// linkKey is the bucket a link goes by: what was written, without its
// extension and in one case of letters.
func linkKey(target string) string {
	return trimMarkdown(strings.TrimSpace(target))
}

// tagKey is the bucket a tag goes by: without its mark, in one case.
func tagKey(name string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(name), "#"))
}

// trimMarkdown takes the extension off a name, in one case of letters.
func trimMarkdown(name string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(name), ".md"))
}

// stem is the name of a file without its extension.
func stem(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// relativeTo is path as it is written from root, when it is under it.
func relativeTo(path, root string) (string, bool) {
	if root == "" || !strings.HasPrefix(path, root) {
		return "", false
	}
	rel := strings.TrimPrefix(strings.TrimPrefix(path, root), string(
		filepath.Separator))
	return filepath.ToSlash(rel), true
}

func pushUnique(m map[string][]string, key, path string) {
	for _, have := range m[key] {
		if have == path {
			return
		}
	}
	m[key] = append(m[key], path)
}

// removeFromList takes path out of the list at key, and the key with it
// when nothing is left: an empty bucket is no bucket.
func removeFromList(m map[string][]string, key, path string) {
	list := m[key]
	kept := list[:0]
	for _, have := range list {
		if have != path {
			kept = append(kept, have)
		}
	}
	if len(kept) == 0 {
		delete(m, key)
		return
	}
	m[key] = kept
}
