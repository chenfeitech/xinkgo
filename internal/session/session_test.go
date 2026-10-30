package session

import (
	"os"
	"path/filepath"
	"testing"
)

// touch makes an empty file, so that it is "there" for these rules.
func touch(t *testing.T, path string) string {
	t.Helper()
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestTabKey_sameFileThroughALink checks that one file reached two ways is
// one tab.
func TestTabKey_sameFileThroughALink(t *testing.T) {
	dir := t.TempDir()
	file := touch(t, filepath.Join(dir, "a.md"))
	link := filepath.Join(dir, "link.md")
	if err := os.Symlink(file, link); err != nil {
		t.Skip("symlinks are not supported here")
	}
	if TabKey(file) != TabKey(link) {
		t.Errorf("%q and %q are the same file but different tabs", file, link)
	}
}

// TestTabKey_fileThatIsGone checks that a note deleted outside the app
// keeps the path it was opened by, so it can still be named.
func TestTabKey_fileThatIsGone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gone.md")
	if got := TabKey(path); got != path {
		t.Errorf("TabKey = %q, want %q", got, path)
	}
}

// TestDedupPaths_keepsFirstOrder checks the order the tabs are shown in.
func TestDedupPaths_keepsFirstOrder(t *testing.T) {
	got := DedupPaths([]string{"b.md", "a.md", "b.md", "c.md", "a.md"})
	want := []string{"b.md", "a.md", "c.md"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// TestPartitionExisting_missingNotesDoNotComeBack checks that a note
// deleted outside the app is not restored at the next start.
func TestPartitionExisting_missingNotesDoNotComeBack(t *testing.T) {
	dir := t.TempDir()
	there := touch(t, filepath.Join(dir, "a.md"))
	gone := filepath.Join(dir, "b.md")
	existing, missing := PartitionExisting([]string{there, gone})
	if len(existing) != 1 || existing[0] != there {
		t.Errorf("existing %v, want %v", existing, []string{there})
	}
	if len(missing) != 1 || missing[0] != gone {
		t.Errorf("missing %v, want %v", missing, []string{gone})
	}
}

// TestPartitionExisting_aDirectoryIsNotANote checks that a path that is a
// directory now is not opened as one.
func TestPartitionExisting_aDirectoryIsNotANote(t *testing.T) {
	dir := t.TempDir()
	_, missing := PartitionExisting([]string{dir})
	if len(missing) != 1 {
		t.Errorf("a directory counted as a note: missing %v", missing)
	}
}

// TestResolveActive checks which tab is active when the app starts.
func TestResolveActive(t *testing.T) {
	paths := []string{"a.md", "b.md", "c.md"}
	if got := ResolveActive(paths, "b.md"); got != "b.md" {
		t.Errorf("active %q, want b.md", got)
	}
	if got := ResolveActive(paths, "gone.md"); got != "c.md" {
		t.Errorf("active %q, want the last one, c.md", got)
	}
	if got := ResolveActive(paths, ""); got != "c.md" {
		t.Errorf("active %q, want the last one, c.md", got)
	}
	if got := ResolveActive(nil, "gone.md"); got != "" {
		t.Errorf("active %q, want none", got)
	}
	if got := ResolveActive(nil, ""); got != "" {
		t.Errorf("active %q, want none", got)
	}
}

// TestWithin checks which notes belong to the vault that is open, so that
// switching vaults closes the others.
func TestWithin(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "notes")
	cases := map[string]bool{
		filepath.Join(root, "a.md"):        true,
		filepath.Join(root, "sub", "b.md"): true,
		filepath.Join(root, "sub"):         true,
		filepath.Join(root, "..", "c.md"):  false,
		filepath.Join("/other", "d.md"):    false,
	}
	for path, want := range cases {
		if got := Within(root, path); got != want {
			t.Errorf("Within(%q, %q) = %v, want %v", root, path, got, want)
		}
	}
}
