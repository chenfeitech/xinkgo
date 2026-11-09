// Package vault is the folder the user's notes are in: what is in it, and
// the list of folders they can open.
//
// The scan reads the whole tree, not one directory at a time, because the
// tree needs the children of a directory to know whether to draw the
// arrow that opens it.
package vault

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/media"
	"github.com/helightxu/xinkgo/internal/textfile"
)

// ignored are the names that are never shown, whatever they hold.
var ignored = map[string]bool{
	".git":          true,
	"node_modules":  true,
	"target":        true,
	".DS_Store":     true,
	".obsidian":     true,
	media.AIDirName: true,
}

// Entry is one file or directory of the tree.
type Entry struct {
	Name     string
	Path     string
	IsDir    bool
	Children []*Entry
	// Kind is what the tree calls the file, and Text whether the editor
	// can take it. They are filled by the scan, so a name that says
	// nothing is not read again for every icon or search.
	Kind media.Kind
	Text bool
}

// Scan reads the tree under root. With showAll it shows every file;
// without it, only what the app can open: text, pictures, and every
// directory, so that the way into them is never hidden. maxTextBytes is
// the largest text file the editor will hold; zero leaves it unchecked.
func Scan(root string, showAll bool, maxTextBytes uint64) (*Entry, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("scan %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}
	return scanDir(root, showAll, maxTextBytes)
}

func scanDir(dir string, showAll bool, maxTextBytes uint64) (*Entry, error) {
	names, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	e := &Entry{
		Name:  filepath.Base(dir),
		Path:  dir,
		IsDir: true,
		Kind:  media.KindDir,
	}
	for _, name := range names {
		base := name.Name()
		if ignored[base] || strings.HasSuffix(base, ".md.tmp") {
			continue
		}
		path := filepath.Join(dir, base)
		if name.IsDir() {
			if media.IsAIDir(path) {
				continue
			}
			child, err := scanDir(path, showAll, maxTextBytes)
			if err != nil {
				// One directory that cannot be read must not hide the
				// rest of the tree.
				continue
			}
			e.Children = append(e.Children, child)
			continue
		}
		kind := media.KindOf(path)
		text := false
		info, known := textfile.ByName(path)
		if showAll {
			text = known && info.Text
		} else {
			image := media.IsImage(path)
			if !image {
				if known && !info.Text {
					continue
				}
				got, err := textfile.Classify(path, maxTextBytes)
				if err != nil || !got.Text {
					continue
				}
				text = true
				if kind == media.KindOther || kind == media.KindText {
					if got.Language != "" {
						kind = media.KindCode
					} else {
						kind = media.KindText
					}
				}
			}
		}
		e.Children = append(e.Children, &Entry{
			Name:  base,
			Path:  path,
			IsDir: false,
			Kind:  kind,
			Text:  text,
		})
	}
	sortEntries(e.Children)
	return e, nil
}

// sortEntries puts directories first, and the rest in the order of their
// names, without regard to case.
func sortEntries(entries []*Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) <
			strings.ToLower(entries[j].Name)
	})
}

// Walk calls fn for the entry and everything under it.
func Walk(e *Entry, fn func(*Entry) error) error {
	if err := fn(e); err != nil {
		return err
	}
	for _, c := range e.Children {
		if err := Walk(c, fn); err != nil {
			return err
		}
	}
	return nil
}

// Files lists the files under the entry, without the directories.
func Files(e *Entry) []string {
	var out []string
	_ = Walk(e, func(e *Entry) error {
		if !e.IsDir {
			out = append(out, e.Path)
		}
		return nil
	})
	return out
}

// Status is the state of a vault's folder: whether the app can use it, and
// what is wrong when it cannot.
type Status int

// The states a vault's folder can be in.
const (
	StatusOk Status = iota
	StatusMissing
	StatusNotDir
	StatusNoPermission
	StatusDuplicate
	StatusNested
)

// String is what a user is told about the folder, in the language the app
// shows.
func (s Status) String() string {
	switch s {
	case StatusOk:
		return locale.VaultStatusOk()
	case StatusMissing:
		return locale.VaultStatusMissing()
	case StatusNotDir:
		return locale.VaultStatusNotDir()
	case StatusNoPermission:
		return locale.VaultStatusNoPermission()
	case StatusDuplicate:
		return locale.VaultStatusDuplicate()
	case StatusNested:
		return locale.VaultStatusNested()
	default:
		return locale.VaultStatusUnknown()
	}
}

// Usable reports whether notes can be opened from the folder. A nested
// vault is a warning, not a fault: it works, it is just inside another.
func (s Status) Usable() bool {
	return s == StatusOk || s == StatusNested
}

// VaultEntry is one folder the user can open.
type VaultEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}

// Registry is the list of vaults, and which one is open.
type Registry struct {
	Vaults []VaultEntry `json:"vaults"`
	Active *string      `json:"active"`
}

// idSeq keeps two vaults added one after the other from getting the same
// identifier, which they would if the clock had not moved on.
var idSeq atomic.Uint64

// NewID gives a vault an identifier that no other has.
func NewID(path string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(path))
	return fmt.Sprintf("v%x-%x", h.Sum32(),
		uint64(time.Now().UnixNano())+idSeq.Add(1))
}

// Add puts the folder at path in the list, and returns what it added.
func (r *Registry) Add(path string) VaultEntry {
	e := VaultEntry{
		ID:   NewID(path),
		Name: filepath.Base(path),
		Path: path,
	}
	if strings.TrimSpace(e.Name) == "" {
		e.Name = path
	}
	r.Vaults = append(r.Vaults, e)
	return e
}

// At finds the vault of that folder, or nothing when the folder is not
// one of them: a folder is put in the list once, for two rows of one
// folder would be one row too many.
func (r *Registry) At(path string) *VaultEntry {
	for i := range r.Vaults {
		if r.Vaults[i].Path == path {
			return &r.Vaults[i]
		}
	}
	return nil
}

// ByID finds a vault by its identifier.
func (r *Registry) ByID(id string) *VaultEntry {
	for i := range r.Vaults {
		if r.Vaults[i].ID == id {
			return &r.Vaults[i]
		}
	}
	return nil
}

// Remove takes a vault out of the list, and forgets it as the open one.
func (r *Registry) Remove(id string) {
	kept := r.Vaults[:0:0]
	for _, v := range r.Vaults {
		if v.ID != id {
			kept = append(kept, v)
		}
	}
	r.Vaults = kept
	if r.Active != nil && *r.Active == id {
		r.Active = nil
	}
}

// Rename gives the vault another name. A name of nothing at all leaves
// it the one the folder has.
func (r *Registry) Rename(id, name string) {
	v := r.ByID(id)
	if v == nil {
		return
	}
	v.Name = strings.TrimSpace(name)
	if v.Name == "" {
		v.Name = filepath.Base(v.Path)
	}
}

// Relocate is the vault standing somewhere else: the folder was moved,
// or named again outside the app. The name follows the folder, unless
// the user had given it one of their own.
func (r *Registry) Relocate(id, path string) {
	v := r.ByID(id)
	if v == nil {
		return
	}
	if v.Name == "" || v.Name == filepath.Base(v.Path) {
		v.Name = filepath.Base(path)
	}
	v.Path = path
}

// StatusOf says what is wrong with the folder of a vault, comparing it
// with the others to find one that is the same or above it.
func (r *Registry) StatusOf(id string) Status {
	v := r.ByID(id)
	if v == nil {
		return StatusMissing
	}
	info, err := os.Stat(v.Path)
	if err != nil {
		if os.IsPermission(err) {
			return StatusNoPermission
		}
		return StatusMissing
	}
	if !info.IsDir() {
		return StatusNotDir
	}
	// A folder that cannot be read at all is no use.
	if _, err := os.ReadDir(v.Path); err != nil && os.IsPermission(err) {
		return StatusNoPermission
	}
	for _, other := range r.Vaults {
		if other.ID == id {
			continue
		}
		if samePath(other.Path, v.Path) {
			return StatusDuplicate
		}
		if isAbove(other.Path, v.Path) {
			return StatusNested
		}
	}
	return StatusOk
}

// ResolveActive returns the vault that is open: the one the user chose,
// when it can still be used; the first one that can, when it cannot;
// nothing, when none of them can.
func (r *Registry) ResolveActive() *VaultEntry {
	if r.Active != nil {
		if v := r.ByID(*r.Active); v != nil &&
			r.StatusOf(v.ID).Usable() {
			return v
		}
	}
	for _, v := range r.Vaults {
		if r.StatusOf(v.ID).Usable() {
			id := v.ID
			r.Active = &id
			return r.ByID(id)
		}
	}
	r.Active = nil
	return nil
}

// samePath reports whether two paths are the same folder.
func samePath(a, b string) bool {
	ia, ea := os.Stat(a)
	ib, eb := os.Stat(b)
	if ea != nil || eb != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return os.SameFile(ia, ib)
}

// isAbove reports whether a is a folder above b.
func isAbove(a, b string) bool {
	rel, err := filepath.Rel(a, b)
	if err != nil {
		return false
	}
	return rel != "." && !filepath.IsAbs(rel) &&
		!strings.HasPrefix(rel, "..")
}

// Readable reports whether a directory can be listed.
func Readable(path string) bool {
	_, err := os.ReadDir(path)
	return err == nil
}
