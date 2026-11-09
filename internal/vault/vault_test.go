package vault

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/helightxu/xinkgo/internal/media"
)

// tree makes a vault with files the app opens and files it does not.
func tree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"b.md":              "# b",
		"a.md":              "# a",
		"notes/c.md":        "# c",
		"pic.png":           "",
		"data.json":         "{}",
		"movie.mp4":         "",
		".hidden.md":        "",
		".git/config":       "",
		"node_modules/x.md": "",
		".xink/ai.md":       "",
		"draft.md.tmp":      "",
	}
	for name, text := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// names of the files and directories, in the order they are shown.
func names(e *Entry) []string {
	out := []string{e.Name}
	for _, c := range e.Children {
		out = append(out, c.Name)
	}
	return out
}

// TestScan_showsOnlyWhatTheAppOpens checks the default: text and pictures,
// every directory, and none of what is never shown.
func TestScan_showsOnlyWhatTheAppOpens(t *testing.T) {
	root := tree(t)
	e, err := Scan(root, false, 0)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	got := map[string]bool{}
	for _, n := range names(e) {
		got[n] = true
	}
	for _, want := range []string{"a.md", "b.md", "pic.png", "data.json",
		"notes", ".hidden.md"} {
		if !got[want] {
			t.Errorf("%q is not shown", want)
		}
	}
	for _, unwanted := range []string{"movie.mp4", ".git", "node_modules",
		".xink", "draft.md.tmp"} {
		if got[unwanted] {
			t.Errorf("%q should not be shown", unwanted)
		}
	}
}

// TestScan_showAllShowsEverything checks the switch: with it, a video is
// there too, but what is never shown is still not.
func TestScan_showAllShowsEverything(t *testing.T) {
	root := tree(t)
	e, err := Scan(root, true, 0)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	got := map[string]bool{}
	for _, n := range names(e) {
		got[n] = true
	}
	if !got["movie.mp4"] {
		t.Error("a video is not shown although everything is asked for")
	}
	for _, unwanted := range []string{".git", "node_modules", ".xink",
		"draft.md.tmp"} {
		if got[unwanted] {
			t.Errorf("%q should never be shown", unwanted)
		}
	}
}

// TestScan_aNameThatSaysNothingIsRead checks the files a name alone
// cannot place: a Makefile and a LICENSE are text the editor can take,
// and a file with NUL in it is not.
func TestScan_aNameThatSaysNothingIsRead(t *testing.T) {
	root := t.TempDir()
	files := map[string][]byte{
		"Makefile": []byte("all:\n\techo hi\n"),
		"LICENSE":  []byte("MIT\n"),
		"blob":     {'a', 0, 'b'},
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(root, name), data,
			0o644); err != nil {
			t.Fatal(err)
		}
	}
	e, err := Scan(root, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]*Entry{}
	for _, c := range e.Children {
		got[c.Name] = c
	}
	if got["blob"] != nil {
		t.Error("a binary file with no extension was shown")
	}
	for name, kind := range map[string]media.Kind{
		"Makefile": media.KindCode,
		"LICENSE":  media.KindText,
	} {
		one := got[name]
		if one == nil {
			t.Errorf("%q is not shown", name)
			continue
		}
		if !one.Text || one.Kind != kind {
			t.Errorf("%q is text=%v kind=%v, want text=true kind=%v",
				name, one.Text, one.Kind, kind)
		}
	}
}

// TestScan_directoriesFirstAndNamesInOrder checks the order the tree is
// drawn in.
func TestScan_directoriesFirstAndNamesInOrder(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"z.md", "a.md", "M.md", "sub"} {
		path := filepath.Join(root, name)
		if name == "sub" {
			if err := os.Mkdir(path, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	e, err := Scan(root, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	children := e.Children
	if len(children) == 0 || !children[0].IsDir {
		t.Fatalf("the first entry is not a directory: %+v", children)
	}
	var order []string
	for _, c := range children[1:] {
		order = append(order, c.Name)
	}
	want := []string{"a.md", "M.md", "z.md"}
	for i := range want {
		if i >= len(order) || order[i] != want[i] {
			t.Fatalf("order %v, want %v", order, want)
		}
	}
}

// TestScan_fileInsteadOfDirectoryIsAnError checks a vault whose folder is
// a file.
func TestScan_fileInsteadOfDirectoryIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.md")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Scan(path, false, 0); err == nil {
		t.Error("Scan of a file gave no error")
	}
}

// TestFiles_listsEveryNote checks what the index and the search go
// through.
func TestFiles_listsEveryNote(t *testing.T) {
	e, err := Scan(tree(t), false, 0)
	if err != nil {
		t.Fatal(err)
	}
	files := Files(e)
	if len(files) == 0 {
		t.Fatal("no files found")
	}
	for _, path := range files {
		if filepath.Base(path) == "notes" {
			t.Errorf("a directory is among the files: %s", path)
		}
	}
}

// TestRegistry_addRemove checks the list of vaults.
func TestRegistry_addRemove(t *testing.T) {
	var r Registry
	a := r.Add("/notes")
	if len(r.Vaults) != 1 || r.Vaults[0].Name != "notes" {
		t.Fatalf("vaults %+v, want one named notes", r.Vaults)
	}
	r.Add("/other")
	r.Remove(a.ID)
	if len(r.Vaults) != 1 || r.Vaults[0].Path != "/other" {
		t.Errorf("vaults %+v, want only /other", r.Vaults)
	}
	if r.ByID(a.ID) != nil {
		t.Error("the removed vault is still found")
	}
}

// TestNewID_givesDifferentIdentifiers checks that two vaults of the same
// folder are told apart.
func TestNewID_givesDifferentIdentifiers(t *testing.T) {
	if NewID("/notes") == NewID("/notes") {
		t.Error("two vaults got the same identifier")
	}
}

// TestStatusOf checks what is said about a folder.
func TestStatusOf(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "notes")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "file.md")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	var r Registry
	good := r.Add(dir)
	gone := r.Add(filepath.Join(root, "gone"))
	notDir := r.Add(file)
	if s := r.StatusOf(good.ID); s != StatusOk {
		t.Errorf("a folder that is there: %v", s)
	}
	if s := r.StatusOf(gone.ID); s != StatusMissing {
		t.Errorf("a folder that is not there: %v", s)
	}
	if s := r.StatusOf(notDir.ID); s != StatusNotDir {
		t.Errorf("a file instead of a folder: %v", s)
	}
}

// TestStatusOf_duplicateAndNested checks two vaults that are the same
// folder, and one inside another.
func TestStatusOf_duplicateAndNested(t *testing.T) {
	root := t.TempDir()
	inner := filepath.Join(root, "sub")
	if err := os.Mkdir(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	var r Registry
	outer := r.Add(root)
	inside := r.Add(inner)
	if s := r.StatusOf(inside.ID); s != StatusNested {
		t.Errorf("a vault inside another: %v", s)
	}
	if !StatusNested.Usable() {
		t.Error("a nested vault should still be usable")
	}
	// Two vaults of the same folder: neither is used until one is removed,
	// so that the same note is not opened from two places.
	again := r.Add(root)
	if s := r.StatusOf(again.ID); s != StatusDuplicate {
		t.Errorf("a vault that is another one: %v", s)
	}
	if s := r.StatusOf(outer.ID); s != StatusDuplicate {
		t.Errorf("the first of two that are the same: %v", s)
	}
	if StatusDuplicate.Usable() {
		t.Error("a duplicate vault should not be used")
	}
}

// TestActive_fallsBackToAUsableVault checks what happens when the folder
// the user had open is gone: another one is opened, and starting up is
// not blocked.
func TestActive_fallsBackToAUsableVault(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "notes")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var r Registry
	gone := r.Add(filepath.Join(root, "gone"))
	notes := r.Add(dir)
	r.Active = &gone.ID
	if v := r.ResolveActive(); v == nil || v.Path != notes.Path {
		t.Fatalf("active %+v, want notes", v)
	}
}

// TestActive_noneUsableGivesNothing checks that a list of folders that are
// all gone opens no vault at all.
func TestActive_noneUsableGivesNothing(t *testing.T) {
	var r Registry
	gone := r.Add(filepath.Join(t.TempDir(), "gone"))
	r.Active = &gone.ID
	if v := r.ResolveActive(); v != nil {
		t.Errorf("active %+v, want nothing", v)
	}
	if r.Active != nil {
		t.Error("the app still remembers a vault that cannot be used")
	}
}

// TestRemove_forgetsTheOpenVault checks that removing the vault that is
// open leaves none open.
func TestRemove_forgetsTheOpenVault(t *testing.T) {
	var r Registry
	v := r.Add("/notes")
	id := v.ID
	r.Active = &id
	r.Remove(id)
	if r.Active != nil {
		t.Error("the app still remembers the vault that was removed")
	}
}

// TestAt_theVaultOfThatFolder checks that a folder is found by where it
// stands: a folder is put in the list once, and two rows of one folder
// would be one row too many.
func TestAt_theVaultOfThatFolder(t *testing.T) {
	var r Registry
	notes := r.Add("/notes")
	other := r.Add("/other")
	if v := r.At("/notes"); v == nil || v.ID != notes.ID {
		t.Errorf("At(/notes) = %+v, want %+v", v, notes)
	}
	if v := r.At("/other"); v == nil || v.ID != other.ID {
		t.Errorf("At(/other) = %+v, want %+v", v, other)
	}
	if v := r.At("/nowhere"); v != nil {
		t.Errorf("At(/nowhere) = %+v, want nothing", v)
	}
}
