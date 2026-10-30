package fs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestUniqueName_keepsTheExtension checks the name of the second file of
// the same name.
func TestUniqueName_keepsTheExtension(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.md"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := UniqueName(dir, "a.md"); got != "a-1.md" {
		t.Errorf("got %q, want a-1.md", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "a-1.md"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := UniqueName(dir, "a.md"); got != "a-2.md" {
		t.Errorf("got %q, want a-2.md", got)
	}
}

// TestUniqueName_freeNameIsUntouched checks a name nothing has.
func TestUniqueName_freeNameIsUntouched(t *testing.T) {
	if got := UniqueName(t.TempDir(), "notes.md"); got != "notes.md" {
		t.Errorf("got %q, want notes.md", got)
	}
}

// TestRenameName_addsMarkdownExtension checks that renaming a note drops
// nothing: "b" stays a note.
func TestRenameName_addsMarkdownExtension(t *testing.T) {
	got := RenameName("/vault/a.md", "b")
	if got != filepath.Join("/vault", "b.md") {
		t.Errorf("got %q, want /vault/b.md", got)
	}
}

// TestRenameName_keepsOtherExtensions checks that a data file keeps the
// extension the user gave it.
func TestRenameName_keepsOtherExtensions(t *testing.T) {
	if got := RenameName("/vault/a.json", "b"); got != filepath.Join("/vault",
		"b") {
		t.Errorf("got %q, want /vault/b", got)
	}
	if got := RenameName("/vault/a.md", "b.txt"); got != filepath.Join("/vault",
		"b.txt") {
		t.Errorf("got %q, want /vault/b.txt", got)
	}
}

// TestRenameName_emptyNameChangesNothing checks an empty name from the
// dialog is not turned into a file called ".md".
func TestRenameName_emptyNameChangesNothing(t *testing.T) {
	if got := RenameName("/vault/a.md", "   "); got != "/vault/a.md" {
		t.Errorf("got %q, want the old path", got)
	}
}

// TestWriteNewFile_refusesToOverwrite checks that creating a note never
// replaces one that is there.
func TestWriteNewFile_refusesToOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.md")
	if err := WriteNewFile(path, "first"); err != nil {
		t.Fatal(err)
	}
	if err := WriteNewFile(path, "second"); err == nil {
		t.Error("the second write overwrote the first")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "first" {
		t.Errorf("contents %q, want the first ones", data)
	}
}

// TestMove_withinADirectory checks the common case: a rename.
func TestMove_withinADirectory(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.md")
	if err := os.WriteFile(src, []byte("text"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "sub", "b.md")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Move(src, dst); err != nil {
		t.Fatalf("Move: %v", err)
	}
	if _, err := os.Stat(src); !errors.Is(err, os.ErrNotExist) {
		t.Error("the source is still there")
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "text" {
		t.Errorf("contents %q, want text", data)
	}
}

// TestRemoveAll_refusesTheVaultItself checks that the vault cannot be
// deleted from inside the app.
func TestRemoveAll_refusesTheVaultItself(t *testing.T) {
	root := t.TempDir()
	if err := RemoveAll(root, root); err == nil {
		t.Error("the vault was removed")
	}
}

// TestRemoveAll_refusesOutsideTheVault checks that a path outside the
// vault is never touched.
func TestRemoveAll_refusesOutsideTheVault(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	file := filepath.Join(outside, "a.md")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveAll(root, file); err == nil {
		t.Error("a file outside the vault was removed")
	}
	if _, err := os.Stat(file); err != nil {
		t.Error("the file outside the vault is gone")
	}
}

// TestRemoveAll_removesADirectoryWithItsFiles checks deleting a folder
// that is not empty.
func TestRemoveAll_removesADirectoryWithItsFiles(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a.md"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveAll(root, sub); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}
	if _, err := os.Stat(sub); !errors.Is(err, os.ErrNotExist) {
		t.Error("the directory is still there")
	}
}
