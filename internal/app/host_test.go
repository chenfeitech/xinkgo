package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/helightxu/xinkgo/internal/index"
	"github.com/helightxu/xinkgo/internal/skill"
)

// hostOf is a vault of two notes, with an index built of them, for the
// tools to look through.
func hostOf(t *testing.T) (vaultHost, string) {
	t.Helper()
	root := t.TempDir()
	a := filepath.Join(root, "a.md")
	b := filepath.Join(root, "b.md")
	if err := os.WriteFile(a, []byte("# A\n\nsome words\n#tag-a\n"),
		0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("# B\n\nsee [[a]]\n#tag-a\n"),
		0o644); err != nil {
		t.Fatal(err)
	}
	idx := index.New()
	idx.Build(root, []string{a, b})
	return vaultHost{
		root: root,
		idx:  idx,
		skills: []skill.Skill{
			{Name: "demo", Description: "a demo"},
		},
	}, root
}

// TestHost_aNoteIsReadOffTheDisk checks that the disk is the truth of a
// note, and not the index, which is only made of it.
func TestHost_aNoteIsReadOffTheDisk(t *testing.T) {
	h, root := hostOf(t)
	got, ok := h.ReadNote(filepath.Join(root, "a.md"))
	if !ok {
		t.Fatal("the note could not be read")
	}
	if !strings.Contains(got, "some words") {
		t.Fatalf("text = %q", got)
	}
	// A relative path is one from the root, for that is how a model
	// names a note.
	if got, ok := h.ReadNote("a.md"); !ok ||
		!strings.Contains(got, "some words") {
		t.Fatalf("a.md = %q %v, want the note", got, ok)
	}
}

// TestHost_aToolIsNotGivenTheRunOfTheMachine checks that a model asking
// for a path out of the vault is not given it: a tool reads the vault,
// and that is all it reads.
func TestHost_aToolIsNotGivenTheRunOfTheMachine(t *testing.T) {
	h, root := hostOf(t)
	outside := filepath.Join(filepath.Dir(root), "outside.md")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"../outside.md",
		outside,
		"a/../../outside.md",
		"",
		"   ",
	} {
		if _, ok := h.ReadNote(path); ok {
			t.Errorf("%q was read although it is not in the vault", path)
		}
	}
}

// TestHost_theSearchIsAskedOfTheWholeVault checks that a tool's search
// looks through every note, and finds the line the words stand in.
func TestHost_theSearchIsAskedOfTheWholeVault(t *testing.T) {
	h, _ := hostOf(t)
	hits := h.Search("some words", 10)
	if len(hits) != 1 {
		t.Fatalf("hits = %v, want one", hits)
	}
	if filepath.Base(hits[0].Path) != "a.md" {
		t.Fatalf("hit = %v", hits[0])
	}
	if got := h.Search("words that are nowhere", 10); len(got) != 0 {
		t.Fatalf("hits = %v, want none", got)
	}
}

// TestHost_theTagsAreSaidWithHowManyNotesUseThem checks the count the
// index holds.
func TestHost_theTagsAreSaidWithHowManyNotesUseThem(t *testing.T) {
	h, _ := hostOf(t)
	tags := h.Tags()
	if len(tags) != 1 || tags[0].Name != "tag-a" || tags[0].Count != 2 {
		t.Fatalf("tags = %v, want tag-a twice", tags)
	}
}

// TestHost_theBacklinksAreWhereANoteIsPointedAtFrom checks what the
// index knows of who points at whom.
func TestHost_theBacklinksAreWhereANoteIsPointedAtFrom(t *testing.T) {
	h, root := hostOf(t)
	links := h.Backlinks(filepath.Join(root, "a.md"))
	if len(links) != 1 || filepath.Base(links[0].From) != "b.md" {
		t.Fatalf("links = %v, want b.md", links)
	}
	// A path out of the vault is not looked up at all.
	if got := h.Backlinks("../outside.md"); len(got) != 0 {
		t.Fatalf("links = %v, want none", got)
	}
}

// TestHost_theOutlineIsTheHeadingsOfANote checks the headings, set out
// by how deep they are.
func TestHost_theOutlineIsTheHeadingsOfANote(t *testing.T) {
	h, root := hostOf(t)
	got := h.Outline("a.md")
	if len(got) != 1 || !strings.Contains(got[0], "A") {
		t.Fatalf("outline = %v, want A", got)
	}
	_ = root
}

// TestHost_theNotesAreSaidWithWhatTheyAreCalled checks that a tool is
// given the paths and the titles, and no more than it asked for.
func TestHost_theNotesAreSaidWithWhatTheyAreCalled(t *testing.T) {
	h, _ := hostOf(t)
	if got := h.ListNotes(0); len(got) != 2 {
		t.Fatalf("notes = %v, want two", got)
	}
	if got := h.ListNotes(1); len(got) != 1 {
		t.Fatalf("notes = %v, want one", got)
	}
	for _, note := range h.ListNotes(0) {
		if note.Title == "" {
			t.Fatalf("note %v has no title", note)
		}
	}
}

// TestHost_theSkillOfThatName checks that a skill a tool asks for is
// found, and that one there is no such skill of is said not to be.
func TestHost_theSkillOfThatName(t *testing.T) {
	h, _ := hostOf(t)
	one, ok := h.Skill("demo")
	if !ok || one == nil || one.Name != "demo" {
		t.Fatalf("skill = %v %v, want demo", one, ok)
	}
	if one, ok := h.Skill("nope"); ok || one != nil {
		t.Fatalf("skill = %v %v, want none", one, ok)
	}
}

// TestHost_aVaultWithNoIndexHasNothingToLookThrough checks that a host
// with nothing known of the vault gives nothing, and does not fall over.
func TestHost_aVaultWithNoIndexHasNothingToLookThrough(t *testing.T) {
	h := vaultHost{root: t.TempDir()}
	if got := h.Search("x", 5); len(got) != 0 {
		t.Errorf("hits = %v", got)
	}
	if got := h.Tags(); len(got) != 0 {
		t.Errorf("tags = %v", got)
	}
	if got := h.Backlinks("a.md"); len(got) != 0 {
		t.Errorf("links = %v", got)
	}
	if got := h.Outline("a.md"); len(got) != 0 {
		t.Errorf("outline = %v", got)
	}
	if got := h.ListNotes(5); len(got) != 0 {
		t.Errorf("notes = %v", got)
	}
	// The note itself is still read off the disk, which is the truth
	// of it whether it is indexed or not.
	if _, ok := h.ReadNote("a.md"); ok {
		t.Error("a note that is not there was read")
	}
}
