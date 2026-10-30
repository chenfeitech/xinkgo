// Package fs is what the app does to the files of a vault: writing a new
// one without overwriting another, renaming, moving between devices, and
// importing from outside.
//
// Every operation leaves the disk as it was when it fails, as far as the
// file systems allow: a move that cannot be done must not lose the file.
package fs

import (
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"
)

// UniqueName returns a name in dir that nothing has yet: "a.md" when it
// is free, "a-1.md" when it is not, keeping the extension.
func UniqueName(dir, name string) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	candidate := name
	for n := 1; ; n++ {
		if _, err := os.Stat(filepath.Join(dir, candidate)); errors.Is(err,
			os.ErrNotExist) {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d%s", base, n, ext)
	}
}

// RenameName is the new path of a file renamed to newName: Markdown gets
// its extension back, every other kind keeps the one it was given.
func RenameName(oldPath, newName string) string {
	dir := filepath.Dir(oldPath)
	name := strings.TrimSpace(newName)
	if name == "" {
		return oldPath
	}
	if filepath.Ext(name) == "" && strings.EqualFold(filepath.Ext(oldPath),
		".md") {
		name += ".md"
	}
	return filepath.Join(dir, name)
}

// WriteNewFile writes content to path, which must not be there yet.
func WriteNewFile(path, content string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s is already there", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// Move moves src to dst, within a device or across them: a rename that
// the file system refuses, because the two are on different devices, is
// done as a copy whose size is checked before the source is removed.
func Move(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	// Rename fails across devices: copy, check, then remove the source.
	if err := Copy(src, dst); err != nil {
		return err
	}
	same, err := sameSize(src, dst)
	if err != nil {
		return err
	}
	if !same {
		// Keep the source: better two copies than none.
		if err := os.Remove(dst); err != nil {
			return fmt.Errorf("remove the copy: %w", err)
		}
		return fmt.Errorf("moved %s: the copy is not the same size", src)
	}
	if err := os.Remove(src); err != nil {
		return fmt.Errorf("remove %s after moving it: %w", src, err)
	}
	return nil
}

// Copy copies the file src to dst.
func Copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a file", src)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return nil
}

// CopyAll copies src to dst, a file as a file and a folder with all that
// is in it, as a file or folder dragged into the vault is.
func CopyAll(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return Copy(src, dst)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(
		path string, d iofs.DirEntry, err error,
	) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return Copy(path, target)
	})
}

// sameSize reports whether two files have the same number of bytes.
func sameSize(a, b string) (bool, error) {
	ia, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	ib, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	return ia.Size() == ib.Size(), nil
}

// RemoveAll removes a file, or a directory with what is in it. It refuses
// to remove the vault's root, or anything outside it.
func RemoveAll(root, path string) error {
	if !within(root, path) {
		return fmt.Errorf("%s is not in the vault %s", path, root)
	}
	if filepath.Clean(root) == filepath.Clean(path) {
		return fmt.Errorf("the vault %s cannot be removed", root)
	}
	return os.RemoveAll(path)
}

// within reports whether path is root or something under it.
func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (!filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."))
}
