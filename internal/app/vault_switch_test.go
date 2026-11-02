package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/index"
	"github.com/helightxu/xinkgo/internal/state"
	"github.com/helightxu/xinkgo/internal/view"
)

// noteIn is a note written into the folder, and its path.
func noteIn(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("# "+name+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// appOfTwoVaults is an app that knows two folders of notes, the first of
// them open, and what is open written into its own directory.
func appOfTwoVaults(t *testing.T) (*appState, string, string) {
	t.Helper()
	s := &appState{
		dir: t.TempDir(),
		ws:  view.NewWorkspace(config.Default()),
		idx: index.New(),
	}
	one, two := t.TempDir(), t.TempDir()
	if err := s.openVault(one); err != nil {
		t.Fatalf("openVault: %v", err)
	}
	return s, one, two
}

// pathsOf is which of the paths the tabs hold.
func pathsOf(t *testing.T, paths []string, want ...string) {
	t.Helper()
	for _, w := range want {
		found := false
		for _, p := range paths {
			if p == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("the tabs hold %v, which has not %v", paths, w)
		}
	}
}

// TestSwitchVault_eachKeepsItsOwnOpenNotes checks what switching does to
// the notes that stand open: the ones of the vault that is left are kept
// for it, and come back when the user comes back to it.
func TestSwitchVault_eachKeepsItsOwnOpenNotes(t *testing.T) {
	s, one, two := appOfTwoVaults(t)
	a1 := noteIn(t, one, "a1.md")
	a2 := noteIn(t, one, "a2.md")
	b1 := noteIn(t, two, "b1.md")
	for _, p := range []string{a1, a2} {
		if _, err := s.ws.Tabs.Open(p); err != nil {
			t.Fatalf("Open %v: %v", p, err)
		}
	}

	s.switchVault(two)
	if paths := s.ws.Tabs.Paths(); len(paths) != 0 {
		t.Fatalf("the tabs hold %v, want the other vault's notes gone",
			paths)
	}
	if _, err := s.ws.Tabs.Open(b1); err != nil {
		t.Fatalf("Open %v: %v", b1, err)
	}

	// Back to the first: the two notes that stood open in it come back.
	s.switchVault(one)
	pathsOf(t, s.ws.Tabs.Paths(), a1, a2)
	if paths := s.ws.Tabs.Paths(); len(paths) != 2 {
		t.Errorf("the tabs hold %v, want just the two of this vault",
			paths)
	}
}

// TestSwitchVault_whatIsOpenIsWrittenOneListToAVault checks that what is
// kept is written down, one list to a vault: a crash while another vault
// is open loses neither.
func TestSwitchVault_whatIsOpenIsWrittenOneListToAVault(t *testing.T) {
	s, one, two := appOfTwoVaults(t)
	a1 := noteIn(t, one, "a1.md")
	b1 := noteIn(t, two, "b1.md")
	if _, err := s.ws.Tabs.Open(a1); err != nil {
		t.Fatalf("Open: %v", err)
	}
	s.switchVault(two)
	if _, err := s.ws.Tabs.Open(b1); err != nil {
		t.Fatalf("Open: %v", err)
	}
	s.switchVault(one)

	all, err := state.LoadOpenNotes(s.dir)
	if err != nil {
		t.Fatalf("LoadOpenNotes: %v", err)
	}
	// One list to a vault, both of them kept.
	if len(all) != 2 {
		t.Fatalf("the file holds %v, want one list to each vault", all)
	}
	pathsOf(t, all.Of(s.vaultID()).Paths, a1)
	for id, on := range all {
		if id == s.vaultID() {
			continue
		}
		pathsOf(t, on.Paths, b1)
	}
}
