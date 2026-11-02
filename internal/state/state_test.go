package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/helightxu/xinkgo/internal/vault"
)

func str(s string) *string { return &s }

// TestLoadOpenNotes_missingFileOpensNothing checks a first start.
func TestLoadOpenNotes_missingFileOpensNothing(t *testing.T) {
	all, err := LoadOpenNotes(t.TempDir())
	if err != nil {
		t.Fatalf("LoadOpenNotes: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("got %+v, want nothing open", all)
	}
}

// TestLoadOpenNotes_brokenFileOpensNothing checks that a file the app
// cannot read does not keep it from starting.
func TestLoadOpenNotes_brokenFileOpensNothing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, OpenNotesFile), []byte("{"),
		0o644); err != nil {
		t.Fatal(err)
	}
	all, err := LoadOpenNotes(dir)
	if err == nil {
		t.Error("a file that is not JSON was accepted")
	}
	if len(all) != 0 {
		t.Errorf("got %v, want nothing open", all)
	}
}

// TestSaveLoadOpenNotes_roundTrip checks that what the app writes it reads
// back, in the same order, one list to a vault.
func TestSaveLoadOpenNotes_roundTrip(t *testing.T) {
	dir := t.TempDir()
	want := OpenNotesByVault{
		"v1": {Paths: []string{"b.md", "a.md", "c.md"}, Active: str("a.md")},
		"v2": {Paths: []string{"d.md"}, Active: str("d.md")},
	}
	if err := SaveOpenNotes(dir, want); err != nil {
		t.Fatalf("SaveOpenNotes: %v", err)
	}
	got, err := LoadOpenNotes(dir)
	if err != nil {
		t.Fatalf("LoadOpenNotes: %v", err)
	}
	first := got.Of("v1")
	if strings.Join(first.Paths, ",") != "b.md,a.md,c.md" {
		t.Errorf("paths %v, want the order they were saved in", first.Paths)
	}
	if first.Active == nil || *first.Active != "a.md" {
		t.Errorf("active %v, want a.md", first.Active)
	}
	// The other vault keeps its own, and one the app knows nothing of
	// has none.
	second := got.Of("v2")
	if strings.Join(second.Paths, ",") != "d.md" {
		t.Errorf("paths %v, want the other vault's own", second.Paths)
	}
	if len(got.Of("nope").Paths) != 0 {
		t.Errorf("a vault the app does not know has %v open",
			got.Of("nope").Paths)
	}
}

// TestWithOpenNotes_oneVaultKeepsItsOwn checks that giving one vault its
// notes leaves the others as they were: switching vaults is not allowed
// to forget what was open in the rest.
func TestWithOpenNotes_oneVaultKeepsItsOwn(t *testing.T) {
	all := OpenNotesByVault{"v1": {Paths: []string{"a.md"}}}
	all = all.WithOpenNotes("v2", OpenNotes{Paths: []string{"b.md"}})
	if strings.Join(all.Of("v1").Paths, ",") != "a.md" {
		t.Errorf("v1 = %v, want what it had", all.Of("v1").Paths)
	}
	if strings.Join(all.Of("v2").Paths, ",") != "b.md" {
		t.Errorf("v2 = %v, want what it was given", all.Of("v2").Paths)
	}
	// A vault of no identifier keeps nothing: there is no vault to
	// bring its notes back to.
	if before := len(all); len(all.WithOpenNotes("",
		OpenNotes{Paths: []string{"c.md"}})) != before {
		t.Error("notes were kept for no vault at all")
	}
}

// TestLoadLegacyOpenNotes_theOpenVaultsNotesAreRead checks the file a
// version that knew one vault wrote: its list is read as that vault's,
// so that the notes open then are not lost by the move to many.
func TestLoadLegacyOpenNotes_theOpenVaultsNotesAreRead(t *testing.T) {
	dir := t.TempDir()
	old := `{"paths":["a.md","b.md"],"active":"b.md"}`
	if err := os.WriteFile(filepath.Join(dir, OpenNotesFile),
		[]byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	on, err := LoadLegacyOpenNotes(dir)
	if err != nil {
		t.Fatalf("LoadLegacyOpenNotes: %v", err)
	}
	if strings.Join(on.Paths, ",") != "a.md,b.md" {
		t.Errorf("paths %v, want the two of the old file", on.Paths)
	}
	if on.Active == nil || *on.Active != "b.md" {
		t.Errorf("active %v, want b.md", on.Active)
	}
}

// TestLoadLegacyOpenNotes_aFileOfVaultsGivesNothing checks that a file
// written since the vaults are known is not read as one vault's.
func TestLoadLegacyOpenNotes_aFileOfVaultsGivesNothing(t *testing.T) {
	dir := t.TempDir()
	if err := SaveOpenNotes(dir, OpenNotesByVault{
		"v1": {Paths: []string{"a.md"}},
	}); err != nil {
		t.Fatal(err)
	}
	on, err := LoadLegacyOpenNotes(dir)
	if err != nil {
		t.Fatalf("LoadLegacyOpenNotes: %v", err)
	}
	if len(on.Paths) != 0 {
		t.Errorf("paths %v, want none: the file is one vault to each",
			on.Paths)
	}
}

// TestNormalizeOpenNotes_activeMustBeOneOfThem checks that the active note
// is a tab that is open, so that no tab is active and hidden at once.
func TestNormalizeOpenNotes_activeMustBeOneOfThem(t *testing.T) {
	on := NormalizeOpenNotes(OpenNotes{
		Paths:  []string{"a.md", "b.md"},
		Active: str("gone.md"),
	})
	if on.Active == nil || *on.Active != "b.md" {
		t.Errorf("active %v, want the last open note", on.Active)
	}
}

// TestNormalizeOpenNotes_removesDuplicates checks one tab per note.
func TestNormalizeOpenNotes_removesDuplicates(t *testing.T) {
	on := NormalizeOpenNotes(OpenNotes{
		Paths: []string{"a.md", "b.md", "a.md"},
	})
	if len(on.Paths) != 2 {
		t.Errorf("paths %v, want two", on.Paths)
	}
}

// TestNormalizeOpenNotes_noPathsHasNoActive checks the shape of nothing
// being open.
func TestNormalizeOpenNotes_noPathsHasNoActive(t *testing.T) {
	on := NormalizeOpenNotes(OpenNotes{Paths: nil, Active: str("a.md")})
	if on.Active != nil {
		t.Errorf("active %v, want none", on.Active)
	}
	if on.Paths == nil {
		t.Error("paths is nil, want an empty list so the file is []")
	}
}

// TestLoadVaults_migratesTheOldFile checks that a vault of an older
// version becomes the first of the list, and that the list is written.
func TestLoadVaults_migratesTheOldFile(t *testing.T) {
	dir := t.TempDir()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, LegacyVaultFile),
		[]byte(`{"root":`+quote(t, root)+`}`), 0o644); err != nil {
		t.Fatal(err)
	}
	reg, err := LoadVaults(dir)
	if err != nil {
		t.Fatalf("LoadVaults: %v", err)
	}
	if len(reg.Vaults) != 1 || reg.Vaults[0].Path != root {
		t.Fatalf("vaults %+v, want the one folder", reg.Vaults)
	}
	if reg.Active == nil || *reg.Active != reg.Vaults[0].ID {
		t.Errorf("active %v, want the vault that was migrated", reg.Active)
	}
	if _, err := os.Stat(filepath.Join(dir, VaultsFile)); err != nil {
		t.Errorf("the list was not written: %v", err)
	}
}

// TestLoadVaults_oldFileWithAGoneFolderIsIgnored checks that a folder that
// no longer exists is not migrated.
func TestLoadVaults_oldFileWithAGoneFolderIsIgnored(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, LegacyVaultFile),
		[]byte(`{"root":"/gone/forever"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	reg, err := LoadVaults(dir)
	if err != nil {
		t.Fatalf("LoadVaults: %v", err)
	}
	if len(reg.Vaults) != 0 {
		t.Errorf("vaults %+v, want none", reg.Vaults)
	}
}

// TestLoadVaults_missingFileGivesAnEmptyList checks a first start.
func TestLoadVaults_missingFileGivesAnEmptyList(t *testing.T) {
	reg, err := LoadVaults(t.TempDir())
	if err != nil {
		t.Fatalf("LoadVaults: %v", err)
	}
	if reg.Vaults == nil {
		t.Error("vaults is nil, want an empty list")
	}
}

// TestSaveVaults_isAtomic checks that the list is replaced only once the
// new one is on disk, and that no temporary file is left.
func TestSaveVaults_isAtomic(t *testing.T) {
	dir := t.TempDir()
	reg := vault.Registry{Vaults: []vault.VaultEntry{}}
	e := reg.Add("/notes")
	reg.Active = &e.ID
	if err := SaveVaults(dir, reg); err != nil {
		t.Fatalf("SaveVaults: %v", err)
	}
	got, err := LoadVaults(dir)
	if err != nil {
		t.Fatalf("LoadVaults: %v", err)
	}
	if len(got.Vaults) != 1 || got.Vaults[0].Path != "/notes" {
		t.Errorf("vaults %+v, want /notes", got.Vaults)
	}
	names, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		if strings.HasSuffix(n.Name(), ".tmp") {
			t.Errorf("a temporary file is left: %s", n.Name())
		}
	}
}

// TestLoadLayout_otherVersionGivesTheDefault checks that a layout written
// by another version is thrown away, rather than drawn wrong.
func TestLoadLayout_otherVersionGivesTheDefault(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, LayoutFile),
		[]byte(`{"version":6,"left":999,"right":999}`), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err := LoadLayout(dir)
	if err != nil {
		t.Fatalf("LoadLayout: %v", err)
	}
	if l != DefaultLayout() {
		t.Errorf("got %+v, want the default %+v", l, DefaultLayout())
	}
}

// TestSaveLoadLayout_roundTrip checks that the widths come back.
func TestSaveLoadLayout_roundTrip(t *testing.T) {
	dir := t.TempDir()
	want := Layout{Left: 300, Right: 500}
	if err := SaveLayout(dir, want); err != nil {
		t.Fatalf("SaveLayout: %v", err)
	}
	got, err := LoadLayout(dir)
	if err != nil {
		t.Fatalf("LoadLayout: %v", err)
	}
	if got.Left != 300 || got.Right != 500 || got.Version != LayoutVersion {
		t.Errorf("got %+v, want %+v at version %d", got, want,
			LayoutVersion)
	}
}

// TestLoadLayout_missingFileGivesTheDefault checks a first start.
func TestLoadLayout_missingFileGivesTheDefault(t *testing.T) {
	l, err := LoadLayout(t.TempDir())
	if err != nil {
		t.Fatalf("LoadLayout: %v", err)
	}
	if l != DefaultLayout() {
		t.Errorf("got %+v, want %+v", l, DefaultLayout())
	}
}

func quote(t *testing.T, s string) string {
	t.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
