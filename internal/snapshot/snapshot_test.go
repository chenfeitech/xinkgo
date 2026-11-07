package snapshot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestName_theNameSortsInTimeOrder checks the one thing the name is for:
// a folder sorted by name is a folder sorted by time, so the oldest is
// the first and there is nothing to read off the disk to know it.
func TestName_theNameSortsInTimeOrder(t *testing.T) {
	early := Name("a.md")
	if !strings.HasPrefix(early, "20") {
		t.Fatalf("name = %q, want it to start with the year", early)
	}
	if !strings.HasSuffix(early, "-a.md") {
		t.Fatalf("name = %q, want the file named in it", early)
	}
	// A name taken later sorts later.
	stamp := func(name string) string {
		return name[:len(name)-len("-a.md")]
	}
	if stamp(Name("a.md")) < stamp(early) {
		t.Fatal("a later snapshot sorts before an earlier one")
	}
}

func TestOriginal_whatItWasTakenOf(t *testing.T) {
	cases := map[string]string{
		"2026-10-31-104500-a.md":   "a.md",
		"2026-01-02-030405-b.md":   "b.md",
		"2026-10-31-104500-sub.md": "sub.md",
		"no-stamp-here":            "no-stamp-here",
	}
	for in, want := range cases {
		if got := Original(in); got != want {
			t.Errorf("Original(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestWrite_whatWasIsKept checks that a snapshot holds what the note said
// before it was changed.
func TestWrite_whatWasIsKept(t *testing.T) {
	dir := filepath.Join(t.TempDir(), DirName)
	path, err := Write(dir, "a.md", "what it said")
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "what it said" {
		t.Fatalf("snapshot holds %q", data)
	}
	if Count(dir) != 1 {
		t.Fatalf("count = %d, want one", Count(dir))
	}
}

// TestWrite_twiceInOneSecondIsNotOverwritten checks that a second
// snapshot of the same note in the same second does not write over the
// first: there would be no way back at all.
func TestWrite_twiceInOneSecondIsNotOverwritten(t *testing.T) {
	dir := filepath.Join(t.TempDir(), DirName)
	if _, err := Write(dir, "a.md", "first"); err != nil {
		t.Fatal(err)
	}
	if _, err := Write(dir, "a.md", "second"); err != nil {
		t.Fatal(err)
	}
	if got := Count(dir); got != 2 {
		t.Fatalf("count = %d, want both kept", got)
	}
}

// TestRotate_onlyTheNewestAreKept checks that a folder grown too big
// loses its oldest, and not the ones just taken.
func TestRotate_onlyTheNewestAreKept(t *testing.T) {
	dir := filepath.Join(t.TempDir(), DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"2026-01-01-000000-a.md",
		"2026-01-02-000000-a.md",
		"2026-01-03-000000-a.md",
		"2026-01-04-000000-a.md",
	} {
		if err := os.WriteFile(filepath.Join(dir, name),
			[]byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	removed, err := Rotate(dir, 2)
	if err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	if removed != 2 {
		t.Fatalf("removed = %d, want two", removed)
	}
	if got := Count(dir); got != 2 {
		t.Fatalf("count = %d, want two kept", got)
	}
	names, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if names[0].Name() != "2026-01-03-000000-a.md" {
		t.Fatalf("kept = %v, want the newest two", names)
	}
}

// TestRotate_aFolderWithinItsLimitKeepsEverything checks that nothing is
// thrown away while there is room for it.
func TestRotate_aFolderWithinItsLimitKeepsEverything(t *testing.T) {
	dir := filepath.Join(t.TempDir(), DirName)
	if _, err := Write(dir, "a.md", "x"); err != nil {
		t.Fatal(err)
	}
	if removed, err := Rotate(dir, Max); err != nil || removed != 0 {
		t.Fatalf("removed = %d err = %v, want nothing", removed, err)
	}
	// A folder that is not there at all is not a failure either.
	if removed, err := Rotate(filepath.Join(t.TempDir(), "nowhere"),
		Max); err != nil || removed != 0 {
		t.Fatalf("removed = %d err = %v, want nothing", removed, err)
	}
	if got := Count(filepath.Join(t.TempDir(), "nowhere")); got != 0 {
		t.Fatalf("count = %d, want none", got)
	}
}

// TestClear_everyOneGoesAndTheFolderStays checks that clearing empties
// the folder without taking the folder with it.
func TestClear_everyOneGoesAndTheFolderStays(t *testing.T) {
	dir := filepath.Join(t.TempDir(), DirName)
	if _, err := Write(dir, "a.md", "x"); err != nil {
		t.Fatal(err)
	}
	if _, err := Write(dir, "b.md", "x"); err != nil {
		t.Fatal(err)
	}
	removed, err := Clear(dir)
	if err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if removed != 2 {
		t.Fatalf("removed = %d, want two", removed)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("the folder is gone: %v", err)
	}
	// Clearing twice is not a failure.
	if removed, err := Clear(dir); err != nil || removed != 0 {
		t.Fatalf("removed = %d err = %v", removed, err)
	}
}

// TestDir_whereTheyStand checks that snapshots are kept with the
// application, and not in a vault the user may move away.
func TestDir_whereTheyStand(t *testing.T) {
	if got := Dir("/app"); got != filepath.Join("/app", DirName) {
		t.Fatalf("dir = %q", got)
	}
}

// A snapshot now remembers both the vault and the note's path inside
// it, so two notes of the same name are no longer confused.
func TestNameFor_scopeRoundTrip(t *testing.T) {
	want := "folder/sub/a_b~c.md"
	name := NameFor("vabc-1", want)
	path := filepath.Join(t.TempDir(), name)
	vault, rel, ok := Scope(path)
	if !ok {
		t.Fatal("a scoped snapshot was not recognised")
	}
	if vault != "vabc-1" || rel != want {
		t.Errorf("scope = %q / %q, want vabc-1 / %q", vault, rel, want)
	}
	if got := Original(path); got != want {
		t.Errorf("Original = %q, want %q", got, want)
	}
}

// A snapshot written by an older version only kept the file name; it
// must still be listed and put back by name.
func TestScope_oldSnapshotIsNotScoped(t *testing.T) {
	path := filepath.Join(t.TempDir(), Name("a.md"))
	if _, _, ok := Scope(path); ok {
		t.Error("a name-only snapshot was taken for a scoped one")
	}
	if got := Original(path); got != "a.md" {
		t.Errorf("Original = %q, want a.md", got)
	}
}

func TestWriteFor_scopeSurvivesTheFile(t *testing.T) {
	dir := t.TempDir()
	path, err := WriteFor(dir, "vabc-1", "notes/a.md", "old words")
	if err != nil {
		t.Fatal(err)
	}
	vault, rel, ok := Scope(path)
	if !ok || vault != "vabc-1" || rel != "notes/a.md" {
		t.Fatalf("scope = %q / %q / %v", vault, rel, ok)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "old words" {
		t.Fatalf("contents = %q, %v", data, err)
	}
}

// TestList_theNewestIsFirst checks the order the way back is looked for
// in: from what was kept last backwards.
func TestList_theNewestIsFirst(t *testing.T) {
	dir := filepath.Join(t.TempDir(), DirName)
	early, err := Write(dir, "a.md", "first")
	if err != nil {
		t.Fatal(err)
	}
	later, err := Write(dir, "a.md", "second")
	if err != nil {
		t.Fatal(err)
	}
	all := List(dir)
	if len(all) != 2 {
		t.Fatalf("list = %v, want both", all)
	}
	if filepath.Base(all[0]) <= filepath.Base(all[1]) {
		t.Fatalf("list = %v, want %q first", all,
			filepath.Base(later))
	}
	if all[0] != later || all[1] != early {
		t.Fatalf("list = %v, want the newest first", all)
	}
	// A folder that is not there has none, and that is not a failure.
	if got := List(filepath.Join(t.TempDir(), "nowhere")); len(got) != 0 {
		t.Fatalf("list = %v, want none", got)
	}
}

// TestTaken_theNameSaysWhen checks that the time is read back off a name
// that was written to sort in time order.
func TestTaken_theNameSaysWhen(t *testing.T) {
	when, ok := Taken("2026-10-31-104500-a.md")
	if !ok {
		t.Fatal("the time was not read")
	}
	if want := time.Date(2026, 10, 31, 10, 45, 0, 0, time.UTC); !when.Equal(
		want) {
		t.Fatalf("taken = %v, want %v", when, want)
	}
	// A name that is not one of ours says nothing, and is not a
	// failure either.
	if _, ok := Taken("no-stamp-here"); ok {
		t.Error("a name with no stamp was read as a time")
	}
	if _, ok := Taken("2026-13-45-999999-a.md"); ok {
		t.Error("a stamp that cannot be was read as a time")
	}
}
