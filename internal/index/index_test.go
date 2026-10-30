package index

import (
	"os"
	"path/filepath"
	"testing"
)

// vault makes a folder of notes, and gives back the root and the paths
// of each note written in it.
func vault(t *testing.T, files map[string]string) (string, []string) {
	t.Helper()
	root := t.TempDir()
	var paths []string
	for name, text := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	return root, paths
}

// TestIndex_backlinksAreWhoNamedTheNote checks the question the
// back-links answer.
func TestIndex_backlinksAreWhoNamedTheNote(t *testing.T) {
	root, paths := vault(t, map[string]string{
		"a.md": "# A\n\nsee [[b]]\n",
		"b.md": "# B\n",
	})
	x := New()
	x.Build(root, paths)

	refs := x.Backlinks(filepath.Join(root, "b.md"))
	if len(refs) != 1 {
		t.Fatalf("backlinks %v, want one", refs)
	}
	if refs[0].From != filepath.Join(root, "a.md") {
		t.Errorf("the back-link is from %q, want a", refs[0].From)
	}
	if refs[0].Line != 2 {
		t.Errorf("the back-link is at line %d, want 2", refs[0].Line)
	}
	if refs[0].Target != "b" {
		t.Errorf("the target is %q, want it as it was written",
			refs[0].Target)
	}
}

// TestIndex_aNoteNamingItselfIsNoBacklink checks that a note pointing at
// itself does not appear among its own back-links.
func TestIndex_aNoteNamingItselfIsNoBacklink(t *testing.T) {
	root, paths := vault(t, map[string]string{"a.md": "[[a]] and [[a]]\n"})
	x := New()
	x.Build(root, paths)
	if refs := x.Backlinks(filepath.Join(root, "a.md")); len(refs) != 0 {
		t.Errorf("backlinks %v, want none", refs)
	}
}

// TestIndex_aNoteMayBeWrittenSeveralWays checks that naming a note by
// its name and by its path both come to the same back-links.
func TestIndex_aNoteMayBeWrittenSeveralWays(t *testing.T) {
	root, paths := vault(t, map[string]string{
		"notes/b.md": "# B\n",
		"a.md":       "[[b]]\n",
		"c.md":       "[[notes/b]]\n",
		"d.md":       "[[notes/b.md]]\n",
	})
	x := New()
	x.Build(root, paths)
	refs := x.Backlinks(filepath.Join(root, "notes", "b.md"))
	if len(refs) != 3 {
		t.Errorf("backlinks %d, want 3: one for each way of writing it",
			len(refs))
	}
}

// TestIndex_aNoteThatChangesLetGoOfWhatItWrote checks that a note read
// again stops pointing where it no longer points.
func TestIndex_aNoteThatChangesLetGoOfWhatItWrote(t *testing.T) {
	root, paths := vault(t, map[string]string{
		"a.md": "[[b]]\n",
		"b.md": "# B\n",
		"c.md": "# C\n",
	})
	x := New()
	x.Build(root, paths)
	a := filepath.Join(root, "a.md")
	x.Put(a, "[[c]]\n")

	if refs := x.Backlinks(filepath.Join(root, "b.md")); len(refs) != 0 {
		t.Errorf("b has backlinks %v, want none", refs)
	}
	refs := x.Backlinks(filepath.Join(root, "c.md"))
	if len(refs) != 1 || refs[0].From != a {
		t.Errorf("c has backlinks %v, want the one from a", refs)
	}
}

// TestIndex_aNoteTakenAwayIsNoLongerNamed checks that deleting a note
// takes away what it pointed at.
func TestIndex_aNoteTakenAwayIsNoLongerNamed(t *testing.T) {
	root, paths := vault(t, map[string]string{
		"a.md": "[[b]]\n",
		"b.md": "# B\n",
	})
	x := New()
	x.Build(root, paths)
	x.Remove(filepath.Join(root, "a.md"))
	if refs := x.Backlinks(filepath.Join(root, "b.md")); len(refs) != 0 {
		t.Errorf("backlinks %v, want none", refs)
	}
	if len(x.Paths()) != 1 {
		t.Errorf("paths %v, want only b", x.Paths())
	}
}

// TestIndex_resolveByPathByNameByTitle checks the order a link is read
// in, and that the case of its letters makes no difference.
func TestIndex_resolveByPathByNameByTitle(t *testing.T) {
	root, paths := vault(t, map[string]string{
		"notes/target.md": "# A title of its own\n",
		"other.md":        "text\n",
	})
	x := New()
	x.Build(root, paths)
	want := filepath.Join(root, "notes", "target.md")

	if got, ok := x.ResolveOnly("notes/target.md"); !ok || got != want {
		t.Errorf("by path: %q %v, want the note", got, ok)
	}
	if got, ok := x.ResolveOnly("notes/target"); !ok || got != want {
		t.Errorf("by path without extension: %q %v, want the note", got, ok)
	}
	if got, ok := x.ResolveOnly("TARGET"); !ok || got != want {
		t.Errorf("by name: %q %v, want the note", got, ok)
	}
	if got, ok := x.ResolveOnly("A TITLE OF ITS OWN"); !ok || got != want {
		t.Errorf("by title: %q %v, want the note", got, ok)
	}
}

// TestIndex_twoNotesOfOneNameAreAmbiguous checks that a name that could
// be either of two notes is given back as it is, rather than guessed.
func TestIndex_twoNotesOfOneNameAreAmbiguous(t *testing.T) {
	root, paths := vault(t, map[string]string{
		"one/same.md": "# one\n",
		"two/same.md": "# two\n",
	})
	x := New()
	x.Build(root, paths)
	r := x.Resolve("same")
	if r.Kind != Ambiguous || len(r.Candidates) != 2 {
		t.Fatalf("resolution %+v, want two candidates", r)
	}
	if _, ok := x.ResolveOnly("same"); ok {
		t.Error("an ambiguous name was resolved to one note")
	}
}

// TestIndex_aLinkToNothingIsMissing checks a link to a note that is not
// there yet.
func TestIndex_aLinkToNothingIsMissing(t *testing.T) {
	root, paths := vault(t, map[string]string{"a.md": "[[not written]]\n"})
	x := New()
	x.Build(root, paths)
	if r := x.Resolve("not written"); r.Kind != Missing {
		t.Errorf("resolution %+v, want missing", r)
	}
}

// TestIndex_tagsAreCountedAndOfOneCase checks how the tags of a vault
// are counted, and that one tag written two ways is one tag.
func TestIndex_tagsAreCountedAndOfOneCase(t *testing.T) {
	root, paths := vault(t, map[string]string{
		"a.md": "# A\n\n#rust #idea\n",
		"b.md": "#Rust #idea\n",
		"c.md": "#thinking\n",
	})
	x := New()
	x.Build(root, paths)

	tags := x.Tags()
	if len(tags) != 3 {
		t.Fatalf("tags %v, want 3", tags)
	}
	// idea is on two notes, rust on two, thinking on one.
	if tags[0].Name != "idea" || tags[0].Count != 2 {
		t.Errorf("first tag %+v, want idea with 2", tags[0])
	}
	if tags[1].Name != "rust" || tags[1].Count != 2 {
		t.Errorf("second tag %+v, want rust with 2", tags[1])
	}
	if tags[2].Name != "thinking" || tags[2].Count != 1 {
		t.Errorf("third tag %+v, want thinking with 1", tags[2])
	}
	notes := x.NotesWithTag("#RUST")
	if len(notes) != 2 {
		t.Errorf("notes with the tag %v, want both", notes)
	}
}

// TestIndex_onlyMarkdownIsIndexed checks that what is not a note is not
// taken in.
func TestIndex_onlyMarkdownIsIndexed(t *testing.T) {
	root, paths := vault(t, map[string]string{
		"a.md":   "# A\n",
		"b.txt":  "# B\n",
		"c.json": "{}\n",
	})
	x := New()
	x.Build(root, paths)
	if got := x.Paths(); len(got) != 1 || filepath.Base(got[0]) != "a.md" {
		t.Errorf("paths %v, want only the note", got)
	}
}

// TestIndex_theTitleIsTheFirstHeading checks how a note is named when no
// title is asked for.
func TestIndex_theTitleIsTheFirstHeading(t *testing.T) {
	root, paths := vault(t, map[string]string{
		"a.md":     "text\n# The heading\n## second\n",
		"plain.md": "no heading at all\n",
		"# B\n":    "",
		"empty.md": "",
		"named.md": "",
	})
	x := New()
	x.Build(root, paths)
	if got := x.Title(filepath.Join(root, "a.md")); got != "The heading" {
		t.Errorf("title %q, want the first heading", got)
	}
	if got := x.Title(filepath.Join(root, "plain.md")); got != "plain" {
		t.Errorf("title %q, want the name of the file", got)
	}
	if got := x.Title(filepath.Join(root, "empty.md")); got != "empty" {
		t.Errorf("title %q, want the name of the file", got)
	}
}

// TestIndex_aNoteOutsideTheRootIsForgotten checks a vault that was
// opened anew: what the index knew of the last one is thrown away.
func TestIndex_aNoteOutsideTheRootIsForgotten(t *testing.T) {
	root, paths := vault(t, map[string]string{"a.md": "# A\n"})
	x := New()
	x.Build(root, paths)
	other, others := vault(t, map[string]string{"z.md": "# Z\n"})
	x.Build(other, others)
	if got := x.Paths(); len(got) != 1 || filepath.Base(got[0]) != "z.md" {
		t.Errorf("paths %v, want only the note of the new vault", got)
	}
	if x.Root() != other {
		t.Errorf("root %q, want %q", x.Root(), other)
	}
}

// TestIndex_headingsOfANote checks the outline the index holds.
func TestIndex_headingsOfANote(t *testing.T) {
	root, paths := vault(t, map[string]string{
		"a.md": "# One\ntext\n## Two\n",
	})
	x := New()
	x.Build(root, paths)
	headings := x.Headings(filepath.Join(root, "a.md"))
	if len(headings) != 2 || headings[1].Text != "Two" {
		t.Errorf("headings %v, want two, the second being Two", headings)
	}
}
