// Package snapshot is the way back: what a note was before a model
// changed it, kept in the application's own folder, one folder to a
// vault. It is not a stack of undone things and nothing of it is in
// memory: what is saved is on the disk.
//
// The app has no stack of undone things — what is saved is on the disk —
// so a snapshot is the only cheap way back from a change someone did not
// mean to keep.
package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DirName is the folder the snapshots stand in, under the application's
// own data folder: they belong to no vault, for a vault is a thing the
// user may move or forget.
const DirName = "ai-history"

// Max is how many snapshots are kept: more than this and the oldest are
// thrown away, so that using the app longer does not mean piling up
// forever.
const Max = 200

// stampFormat is how the time is written in a snapshot's name: a fixed
// number of digits, so that it is both sorted and read back.
const stampFormat = "2006-01-02-150405"

// Name is what a snapshot of that file is called:
// `<yyyy-mm-dd-HHMMSS>-<文件名>`.
//
// A name that reads in time order also sorts in time order, so the
// oldest is the first of a sorted list and there is nothing to read off
// the disk to find out which it is.
func Name(fileName string) string {
	stamp := time.Now().UTC().Format(stampFormat)
	return stamp + "-" + fileName
}

// Dir is where the snapshots of this application stand.
func Dir(appData string) string {
	return filepath.Join(appData, DirName)
}

// Write keeps what the file was, and gives back where it was put.
//
// A rotation that fails is not said to the caller: the snapshot is
// already written, and a folder with too many old snapshots in it is not
// a reason to tell the user their save failed.
func Write(dir, fileName, content string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("make %s: %w", dir, err)
	}
	path := filepath.Join(dir, Name(fileName))
	if _, err := os.Stat(path); err == nil {
		// Twice in the same second: a number is put after it, rather
		// than one overwriting the other.
		for n := 1; ; n++ {
			other := filepath.Join(dir, fmt.Sprintf("%s-%d",
				Name(fileName), n))
			if _, err := os.Stat(other); err != nil {
				path = other
				break
			}
		}
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	_, _ = Rotate(dir, Max)
	return path, nil
}

// List is the snapshots of the folder, the newest first: the way back is
// looked for from the newest backwards, and the names sort in time order,
// so this is the same reading of the folder the other way round.
//
// A folder that is not there has none, and that is not a failure.
func List(dir string) []string {
	paths, ok := list(dir)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(paths))
	for i := len(paths) - 1; i >= 0; i-- {
		out = append(out, paths[i])
	}
	return out
}

// Taken is when a snapshot was taken, and whether the name says it: a
// name that sorts in time order is a name that says the time.
func Taken(snapshotPath string) (time.Time, bool) {
	base := filepath.Base(snapshotPath)
	if len(base) < len(stampFormat) {
		return time.Time{}, false
	}
	when, err := time.Parse(stampFormat, base[:len(stampFormat)])
	if err != nil {
		return time.Time{}, false
	}
	return when, true
}

// Count is how many snapshots there are. A folder that is not there has
// none, and that is not a failure.
func Count(dir string) int {
	paths, ok := list(dir)
	if !ok {
		return 0
	}
	return len(paths)
}

// Rotate keeps only the newest `keep` snapshots, and gives back how many
// it threw away.
func Rotate(dir string, keep int) (int, error) {
	paths, ok := list(dir)
	if !ok {
		return 0, nil
	}
	if len(paths) <= keep {
		return 0, nil
	}
	// The names sort in time order, so the oldest are at the front.
	sort.Strings(paths)
	removed := 0
	for _, path := range paths[:len(paths)-keep] {
		if err := os.Remove(path); err != nil {
			return removed, fmt.Errorf("remove %s: %w", path, err)
		}
		removed++
	}
	return removed, nil
}

// Clear throws every snapshot away, and gives back how many there were.
func Clear(dir string) (int, error) {
	paths, ok := list(dir)
	if !ok {
		return 0, nil
	}
	removed := 0
	for _, path := range paths {
		if err := os.Remove(path); err != nil {
			return removed, fmt.Errorf("remove %s: %w", path, err)
		}
		removed++
	}
	return removed, nil
}

// list is the snapshot files of the folder, and whether it could be read.
func list(dir string) ([]string, bool) {
	names, err := os.ReadDir(dir)
	if err != nil {
		return nil, false
	}
	out := []string{}
	for _, name := range names {
		if name.IsDir() {
			continue
		}
		out = append(out, filepath.Join(dir, name.Name()))
	}
	sort.Strings(out)
	return out, true
}

// Original is the name of the file a snapshot was taken of: what stands
// after the first dash-and-a-name's worth of stamp.
//
// It is how a snapshot is shown: `2026-10-31-104500-a.md` is a snapshot
// of a.md.
func Original(snapshotPath string) string {
	base := filepath.Base(snapshotPath)
	// The stamp is `yyyy-mm-dd-HHMMSS`, so the name of the file comes
	// after the fourth dash.
	parts := strings.SplitN(base, "-", 4)
	if len(parts) < 4 {
		return base
	}
	_, name, found := strings.Cut(parts[3], "-")
	if !found || name == "" {
		return base
	}
	return name
}
