package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/state"
	"github.com/helightxu/xinkgo/internal/vault"
	"github.com/helightxu/xinkgo/internal/view"
)

// vaultsShown is the settings window with the group of the vaults
// chosen, drawn without a window.
func vaultsShown(t *testing.T, s *appState) *ui.Tester {
	t.Helper()
	if s.dir == "" {
		s.dir = t.TempDir()
	}
	if s.ws == nil {
		s.ws = view.NewWorkspace(config.Default())
	}
	s.config = config.Default()
	s.draft = settings{cfg: s.config, tab: tabVault}
	s.draft.read(&s.config)
	return ui.NewTester(s.settingsView, 820, 680)
}

// appOfVaults is an app that knows the folders given, and has none of
// them open.
func appOfVaults(t *testing.T, dirs ...string) (*appState, *ui.Tester) {
	t.Helper()
	s := &appState{
		dir: t.TempDir(),
		ws:  view.NewWorkspace(config.Default()),
	}
	for _, dir := range dirs {
		s.vaults.Add(dir)
	}
	return s, vaultsShown(t, s)
}

// keptVaults is the list of the vaults as it stands on the disk.
func keptVaults(t *testing.T, s *appState) []vault.VaultEntry {
	t.Helper()
	reg, err := state.LoadVaults(s.dir)
	if err != nil {
		t.Fatalf("LoadVaults: %v", err)
	}
	return reg.Vaults
}

// TestVaultSettings_everyVaultIsOnShow checks that the group of the
// vaults lists them all, and says which of them is open: the vaults are
// kept among the settings, and nowhere else.
func TestVaultSettings_everyVaultIsOnShow(t *testing.T) {
	one, two := t.TempDir(), t.TempDir()
	s, tt := appOfVaults(t, one, two)
	s.vaults.Active = &s.vaults.Vaults[0].ID
	tt.Frame()
	for _, name := range []string{filepath.Base(one), filepath.Base(two)} {
		if !tt.HasText(name) {
			t.Errorf("the group of the vaults does not say %q", name)
		}
	}
	if !tt.HasText("open") {
		t.Error("which of them is open is not said")
	}
	if !tt.HasText("Add a folder…") {
		t.Error("a folder cannot be added from here")
	}
}

// TestVaultSettings_noneOfThemYetIsSaid checks that a list of nothing is
// said to be empty, rather than left for the user to wonder at.
func TestVaultSettings_noneOfThemYetIsSaid(t *testing.T) {
	_, tt := appOfVaults(t)
	if !tt.HasText("no vaults yet") {
		t.Error("an empty list is not said to be one")
	}
}

// TestVaultSettings_aFolderAddedIsPutInTheListAndWrittenDown checks what
// adding does: the folder is one of them, the vault that is open is left
// open, and the list is written down as it is changed, as the rest of
// the settings are.
func TestVaultSettings_aFolderAddedIsPutInTheListAndWrittenDown(t *testing.T) {
	one := t.TempDir()
	s, _ := appOfVaults(t, one)
	two := t.TempDir()
	s.keepVault(two)
	if len(s.vaults.Vaults) != 2 {
		t.Fatalf("the app knows %d vaults, want 2", len(s.vaults.Vaults))
	}
	if s.vaults.At(two) == nil {
		t.Error("the folder added is not one of them")
	}
	if got := len(keptVaults(t, s)); got != 2 {
		t.Errorf("%d vaults are written down, want 2", got)
	}
	if got := len(s.ws.Tree.Vaults.Vaults); got != 2 {
		t.Errorf("the window was told of %d vaults, want 2", got)
	}
}

// TestVaultSettings_theSameFolderIsNotPutInTwice checks that adding a
// folder that is one of them already leaves the list as it was: two rows
// of one folder would be one row too many.
func TestVaultSettings_theSameFolderIsNotPutInTwice(t *testing.T) {
	one := t.TempDir()
	s, _ := appOfVaults(t, one)
	s.keepVault(one)
	if len(s.vaults.Vaults) != 1 {
		t.Errorf("the app knows %d vaults, want 1", len(s.vaults.Vaults))
	}
}

// TestVaultSettings_forgettingTakesOneOutOfTheList checks the button of
// a row: the vault is taken out of the list, and out of what is written
// down.
func TestVaultSettings_forgettingTakesOneOutOfTheList(t *testing.T) {
	one := t.TempDir()
	s, tt := appOfVaults(t, one)
	s.saveVaults()
	if err := tt.Click("Forget"); err != nil {
		t.Fatal(err)
	}
	if len(s.vaults.Vaults) != 0 {
		t.Errorf("the app still knows %d vaults, want 0",
			len(s.vaults.Vaults))
	}
	if got := len(keptVaults(t, s)); got != 0 {
		t.Errorf("%d vaults are written down, want 0", got)
	}
}

// TestVaultSettings_theVaultThatIsOpenIsNotForgotten checks the one
// thing that may not be done from here: the window shows one vault at a
// time, and which one it shows next is for the user to say.
func TestVaultSettings_theVaultThatIsOpenIsNotForgotten(t *testing.T) {
	one := t.TempDir()
	s, tt := appOfVaults(t, one)
	s.root = one
	s.vaults.Active = &s.vaults.Vaults[0].ID
	s.saveVaults()
	tt.Frame()
	if err := tt.Click("Forget"); err != nil {
		t.Fatal(err)
	}
	if len(s.vaults.Vaults) != 1 {
		t.Errorf("the vault that is open was forgotten: %d are left",
			len(s.vaults.Vaults))
	}
}

// TestVaultSettings_aVaultNamedAgainKeepsTheName checks the name as it
// is asked for in the row: what is written in the field is what the
// vault is called, and what is written down.
func TestVaultSettings_aVaultNamedAgainKeepsTheName(t *testing.T) {
	one := t.TempDir()
	s, tt := appOfVaults(t, one)
	s.saveVaults()
	if err := tt.Click("Name again"); err != nil {
		t.Fatal(err)
	}
	if s.draft.vaultRename != s.vaults.Vaults[0].ID {
		t.Fatal("no row is being named again")
	}
	if s.draft.vaultName != filepath.Base(one) {
		t.Fatalf("the field holds %q, want the name the vault has",
			s.draft.vaultName)
	}
	if err := tt.Click("The name of the vault"); err != nil {
		t.Fatal(err)
	}
	// What stands in the field is taken out before the new name is
	// written: as a user would, with the key that rubs out.
	for range s.draft.vaultName {
		tt.Key(0, ui.KeyBackspace)
	}
	tt.Type("My notes")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if got := s.vaults.Vaults[0].Name; got != "My notes" {
		t.Errorf("the vault is called %q, want %q", got, "My notes")
	}
	if got := keptVaults(t, s)[0].Name; got != "My notes" {
		t.Errorf("what is written down is %q, want %q", got, "My notes")
	}
	if s.draft.vaultRename != "" {
		t.Error("the field is still there after the name was given")
	}
}

// TestVaultSettings_aVaultThatMovedIsFollowedThere checks what finding a
// vault again does: the path is the one it stands at now, the name
// follows the folder, and the list is written down.
func TestVaultSettings_aVaultThatMovedIsFollowedThere(t *testing.T) {
	one, two := t.TempDir(), t.TempDir()
	s, _ := appOfVaults(t, one)
	id := s.vaults.Vaults[0].ID
	s.saveVaults()
	s.moveVault(id, two)
	if got := s.vaults.Vaults[0].Path; got != two {
		t.Errorf("the vault stands at %q, want %q", got, two)
	}
	if got := s.vaults.Vaults[0].Name; got != filepath.Base(two) {
		t.Errorf("the vault is called %q, want %q", got,
			filepath.Base(two))
	}
	if got := keptVaults(t, s)[0].Path; got != two {
		t.Errorf("what is written down is %q, want %q", got, two)
	}
}

// TestVaultSettings_aVaultThatIsNotThereIsSaidSo checks that a folder
// the app cannot use is not quietly listed as though it could be.
func TestVaultSettings_aVaultThatIsNotThereIsSaidSo(t *testing.T) {
	one := t.TempDir()
	_, tt := appOfVaults(t, one)
	// The folder is taken away after the app was told of it, as when a
	// disk is unmounted.
	if err := os.RemoveAll(one); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText(vault.StatusMissing.String()) {
		t.Error("a vault whose folder is gone is not said so")
	}
}
