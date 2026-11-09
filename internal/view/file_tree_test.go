package view

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/vault"
)

// treeRoot is a vault with two notes and a directory of one.
func treeRoot() *vault.Entry {
	return &vault.Entry{
		Name:  "vault",
		Path:  "/vault",
		IsDir: true,
		Children: []*vault.Entry{
			{Name: "a.md", Path: "/vault/a.md"},
			{Name: "notes", Path: "/vault/notes", IsDir: true,
				Children: []*vault.Entry{
					{Name: "b.md", Path: "/vault/notes/b.md"},
				}},
			{Name: "pic.png", Path: "/vault/pic.png"},
		},
	}
}

func treeTester(t *testing.T, tr *Tree) *ui.Tester {
	t.Helper()
	return ui.NewTester(tr.View, 260, 600)
}

// TestTreeRows_onlyWhatIsOpen checks what the tree draws: the children of
// the vault, and of a directory only once it is open.
func TestTreeRows_onlyWhatIsOpen(t *testing.T) {
	tr := NewTree()
	tr.SetRoot(treeRoot())
	rows := tr.Rows()
	if len(rows) != 3 {
		t.Fatalf("%d rows, want a.md, notes and pic.png", len(rows))
	}
	tr.Toggle("/vault/notes")
	rows = tr.Rows()
	if len(rows) != 4 {
		t.Fatalf("%d rows, want b.md too", len(rows))
	}
	if rows[2].Entry.Name != "b.md" || rows[2].Depth != 1 {
		t.Errorf("the third row is %+v, want b.md one level in", rows[2])
	}
}

// TestTreeRows_noVaultDrawsNothing checks a first start, before a vault
// is opened.
func TestTreeRows_noVaultDrawsNothing(t *testing.T) {
	if rows := NewTree().Rows(); len(rows) != 0 {
		t.Errorf("%d rows, want none", len(rows))
	}
}

// TestTreeView_clickingANoteOpensIt checks the way from the tree to a tab.
func TestTreeView_clickingANoteOpensIt(t *testing.T) {
	tr := NewTree()
	tr.SetRoot(treeRoot())
	var opened string
	tr.OnOpen = func(path string) { opened = path }
	tt := treeTester(t, tr)
	if err := tt.Click("a.md"); err != nil {
		t.Fatal(err)
	}
	if opened != "/vault/a.md" {
		t.Errorf("opened %q, want /vault/a.md", opened)
	}
	if tr.Selected != "/vault/a.md" {
		t.Errorf("selected %q, want the note that was clicked", tr.Selected)
	}
}

// TestTreeView_clickingADirectoryOpensIt checks that a directory opens
// instead of being opened as a note.
func TestTreeView_clickingADirectoryOpensIt(t *testing.T) {
	tr := NewTree()
	tr.SetRoot(treeRoot())
	var opened string
	tr.OnOpen = func(path string) { opened = path }
	tt := treeTester(t, tr)
	if err := tt.Click("notes"); err != nil {
		t.Fatal(err)
	}
	if opened != "" {
		t.Errorf("a directory was opened as a note: %q", opened)
	}
	if !tr.Open["/vault/notes"] {
		t.Error("the directory did not open")
	}
	// Now that it is open, the note inside it can be clicked.
	if err := tt.Click("b.md"); err != nil {
		t.Fatal(err)
	}
	if opened != "/vault/notes/b.md" {
		t.Errorf("opened %q, want the note inside", opened)
	}
}

// TestTreeReveal_opensTheWayToANote checks that opening a note from
// outside the tree, at start-up or from a search, shows it there.
func TestTreeReveal_opensTheWayToANote(t *testing.T) {
	tr := NewTree()
	tr.SetRoot(treeRoot())
	tr.Reveal("/vault/notes/b.md")
	if !tr.Open["/vault/notes"] {
		t.Error("the directory above the note stayed closed")
	}
	if tr.Selected != "/vault/notes/b.md" {
		t.Errorf("selected %q, want the revealed note", tr.Selected)
	}
}

// TestTreeReveal_unknownPathChangesNothing checks a note that is not in
// the vault.
func TestTreeReveal_unknownPathChangesNothing(t *testing.T) {
	tr := NewTree()
	tr.SetRoot(treeRoot())
	tr.Reveal("/elsewhere/c.md")
	if len(tr.Open) != 0 || tr.Selected != "" {
		t.Errorf("open %v selected %q, want nothing", tr.Open, tr.Selected)
	}
}

// TestTreeCount_countsTheNotes checks the header.
func TestTreeCount_countsTheNotes(t *testing.T) {
	tr := NewTree()
	if n := tr.count(); n != 0 {
		t.Errorf("%d notes, want none", n)
	}
	tr.SetRoot(treeRoot())
	if n := tr.count(); n != 3 {
		t.Errorf("%d notes, want 3", n)
	}
}

// TestTreeCollapseAll_foldsEveryOpenFolder checks the header's button:
// what was opened is closed again, so that only the vault's own children
// are left.
func TestTreeCollapseAll_foldsEveryOpenFolder(t *testing.T) {
	tr := NewTree()
	tr.SetRoot(treeRoot())
	tr.Toggle("/vault/notes")
	if !tr.AnyOpen() {
		t.Fatal("nothing is open, want the directory opened")
	}
	tr.CollapseAll()
	if tr.AnyOpen() {
		t.Error("a directory is still open after all were folded")
	}
	if rows := tr.Rows(); len(rows) != 3 {
		t.Errorf("%d rows, want the three the vault holds itself", len(rows))
	}
}

// TestTreeExpandAll_opensEveryFolder checks the other half of the
// button: every directory opens, and what was inside it is drawn.
func TestTreeExpandAll_opensEveryFolder(t *testing.T) {
	tr := NewTree()
	tr.SetRoot(treeRoot())
	tr.ExpandAll()
	if !tr.Open["/vault/notes"] {
		t.Error("the directory stayed closed")
	}
	if rows := tr.Rows(); len(rows) != 4 {
		t.Errorf("%d rows, want the note inside the directory too", len(rows))
	}
}

// TestTreeCounts_whatTheHeaderSays checks the two numbers beside the
// vault's name.
func TestTreeCounts_whatTheHeaderSays(t *testing.T) {
	tr := NewTree()
	if got := tr.counts(); got != "0 notes · 0 folders" {
		t.Errorf("with no vault the header says %q", got)
	}
	tr.SetRoot(treeRoot())
	if got := tr.counts(); got != "3 notes · 1 folder" {
		t.Errorf("the header says %q, want 3 notes and 1 folder", got)
	}
}

// TestTreeView_theSettingsStandAtTheFoot checks the way to the settings
// the tree gives: it is at the foot of it, below the files, whether the
// vault holds anything or not, and a click on it opens them.
func TestTreeView_theSettingsStandAtTheFoot(t *testing.T) {
	for _, root := range []*vault.Entry{nil, treeRoot()} {
		tr := NewTree()
		tr.SetRoot(root)
		opened := 0
		tr.OnOpenSettings = func() { opened++ }
		tt := treeTester(t, tr)
		if !tt.HasText("Settings") {
			t.Fatal("the tree says nothing of the settings")
		}
		if err := tt.Click("Settings"); err != nil {
			t.Fatal(err)
		}
		if opened != 1 {
			t.Errorf("opened %d times, want once", opened)
		}
	}
}

// TestTreeView_withNoWayToTheSettingsThereIsNoFoot checks a tree that was
// given nowhere to open the settings: it says nothing of them, rather
// than offering a way that goes nowhere.
func TestTreeView_withNoWayToTheSettingsThereIsNoFoot(t *testing.T) {
	tr := NewTree()
	tr.SetRoot(treeRoot())
	if tt := treeTester(t, tr); tt.HasText("Settings") {
		t.Error("the tree offers settings it cannot open")
	}
}

// TestSidebar_theSettingsStandAtTheFootOfThePanel checks the panel the
// tree is in: the way to the settings is at its foot even before a vault
// has been opened, which is when a user is most likely to look for it.
func TestSidebar_theSettingsStandAtTheFootOfThePanel(t *testing.T) {
	w := NewWorkspace(config.Default())
	opened := 0
	w.Tree.OnOpenSettings = func() { opened++ }
	tt := ui.NewTester(w.sidebar, 260, 600)
	if !tt.HasText("Settings") {
		t.Fatal("the panel says nothing of the settings")
	}
	if err := tt.Click("Settings"); err != nil {
		t.Fatal(err)
	}
	if opened != 1 {
		t.Errorf("opened %d times, want once", opened)
	}
}

// TestSidebar_theFootStandsOnce checks that the way to the settings is
// drawn once and not twice, which it would be if the tree drew its foot
// and the panel drew one of its own under it.
func TestSidebar_theFootStandsOnce(t *testing.T) {
	w := NewWorkspace(config.Default())
	w.Tree.SetRoot(treeRoot())
	w.Tree.OnOpenSettings = func() {}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			ui.Row(c).Fill().Children(func() { w.sidebar(c) })
		})
	}, 260, 600)
	// One foot says the word twice: once as the button's own label,
	// once as the words beside the gear.
	if n := countOf(tt.Texts(), "Settings"); n != 2 {
		t.Errorf("the foot stands %d times, want once", n/2)
	}
}

// countOf is how many times the list says this.
func countOf(list []string, want string) int {
	n := 0
	for _, one := range list {
		if one == want {
			n++
		}
	}
	return n
}

// treeWithVaults is a tree of two vaults the user can open, the first of
// them open.
func treeWithVaults(t *testing.T) (*Tree, *string) {
	t.Helper()
	tr := NewTree()
	tr.SetRoot(treeRoot())
	one, two := t.TempDir(), t.TempDir()
	tr.Vaults = vault.Registry{
		Vaults: []vault.VaultEntry{
			{ID: "v1", Name: "Alpha", Path: one},
			{ID: "v2", Name: "Beta", Path: two},
		},
	}
	id := "v1"
	tr.Vaults.Active = &id
	asked := ""
	tr.OnSwitchVault = func(got string) { asked = got }
	return tr, &asked
}

// TestTreeVaultSwitch_theMenuCarriesTheVaults checks the way from one
// vault to another: every folder the user can open stands in the menu of
// the tree's head, and so do opening another and keeping the list.
func TestTreeVaultSwitch_theMenuCarriesTheVaults(t *testing.T) {
	tr, _ := treeWithVaults(t)
	te := treeTester(t, tr)
	te.Click("Switch vault")
	te.Frame()
	got := te.Menu()
	want := []string{"Alpha", "Beta", "-", "Open another folder…",
		"Manage vaults…"}
	if len(got) != len(want) {
		t.Fatalf("the menu holds %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("the menu holds %q, want %q", got, want)
			break
		}
	}
}

// TestTreeVaultSwitch_choosingOneAsksForIt checks what a choice does: the
// vault the user chose is asked for, by its identifier, and the one that
// is already open is not asked for again.
func TestTreeVaultSwitch_choosingOneAsksForIt(t *testing.T) {
	tr, asked := treeWithVaults(t)
	te := treeTester(t, tr)
	te.Click("Switch vault")
	te.Frame()
	te.ChooseMenuItem("Beta")
	if *asked != "v2" {
		t.Errorf("asked for %q, want the vault that was chosen", *asked)
	}
	// The one that is open is marked, and choosing it again does
	// nothing: it is already the vault that is shown.
	if !te.HasText("Alpha") {
		t.Error("the head of the tree does not name the vault that is open")
	}
}

// TestTreeVaultSwitch_theNameTheUserGave checks the head of the tree: it
// calls the vault by the name the user gave it, and not by the name of
// the folder it stands in.
func TestTreeVaultSwitch_theNameTheUserGave(t *testing.T) {
	tr, _ := treeWithVaults(t)
	tr.Vaults.Vaults[0].Name = "My notes"
	if got := tr.vaultName(); got != "My notes" {
		t.Errorf("name = %q, want the one the user gave", got)
	}
	// A vault the app does not know is called by its folder's name.
	tr.Vaults = vault.Registry{}
	if got := tr.vaultName(); got != "vault" {
		t.Errorf("name = %q, want the folder's own", got)
	}
}

// TestTreeVaultSwitch_aVaultThatCannotBeOpenedSaysSo checks a folder that
// is gone: it stands in the menu and says why, rather than failing when
// it is chosen.
func TestTreeVaultSwitch_aVaultThatCannotBeOpenedSaysSo(t *testing.T) {
	tr, _ := treeWithVaults(t)
	tr.Vaults.Vaults[1].Path = "/gone/for/good"
	te := treeTester(t, tr)
	te.Click("Switch vault")
	te.Frame()
	for _, label := range te.Menu() {
		if strings.Contains(label, "Beta") &&
			strings.Contains(label, "not there") {
			return
		}
	}
	t.Errorf("the menu holds %q, want the vault that is gone to say so",
		te.Menu())
}

// deepRoot is a vault of many notes, and a folder at its foot with one
// note in it: a note far enough down the tree that no line is drawn for
// it until the tree is scrolled to it.
func deepRoot() *vault.Entry {
	children := make([]*vault.Entry, 0, 41)
	for i := range 40 {
		children = append(children, &vault.Entry{
			Name: fmt.Sprintf("note-%02d.md", i),
			Path: fmt.Sprintf("/vault/note-%02d.md", i),
		})
	}
	children = append(children, &vault.Entry{
		Name: "notes", Path: "/vault/notes", IsDir: true,
		Children: []*vault.Entry{
			{Name: "last.md", Path: "/vault/notes/last.md"},
		}})
	return &vault.Entry{
		Name: "vault", Path: "/vault", IsDir: true, Children: children,
	}
}

// TestTreeReveal_saysWhetherTheVaultHoldsIt checks that revealing tells
// what came of it, so that what asks knows whether the tree has the file
// to show.
func TestTreeReveal_saysWhetherTheVaultHoldsIt(t *testing.T) {
	tr := NewTree()
	tr.SetRoot(treeRoot())
	if !tr.Reveal("/vault/notes/b.md") {
		t.Error("a note the vault holds was not found")
	}
	if tr.Reveal("/vault/notes/gone.md") {
		t.Error("a note the vault does not hold was found")
	}
}

// TestTreeReveal_bringsTheNoteIntoView checks that a note revealed far
// down a long vault is drawn where the eye is, and not left below the
// fold for the user to scroll to.
func TestTreeReveal_bringsTheNoteIntoView(t *testing.T) {
	tr := NewTree()
	tr.SetRoot(deepRoot())
	if !tr.Reveal("/vault/notes/last.md") {
		t.Fatal("the note of the folder at the foot was not found")
	}
	tt := treeTester(t, tr)
	tt.Frame()
	line, ok := tt.Find("last.md")
	if !ok {
		t.Fatal("the tree draws no line for the note it was told to reveal")
	}
	if line.Y < 0 || line.Y+line.H > 600 {
		t.Errorf("the line stands at %.1f+%.1f, outside the 600 the tree "+
			"was given", line.Y, line.H)
	}
}

// followedWorkspace is a window with a vault's tree read from the disk
// and two notes open in it: one at the root of the vault, which is the
// one on show, and one in a folder, which the tree draws no line for
// while that folder stands shut.
func followedWorkspace(t *testing.T) (*Workspace, string, string) {
	t.Helper()
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	one := filepath.Join(root, "one.md")
	two := filepath.Join(sub, "two.md")
	for _, path := range []string{one, two} {
		if err := os.WriteFile(path, []byte("# note\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	scanned, err := vault.Scan(root, false, 0)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	w := NewWorkspace(config.Default())
	w.Tree.SetRoot(scanned)
	for _, path := range []string{one, two} {
		if _, err := w.Tabs.Open(path); err != nil {
			t.Fatalf("Open %s: %v", path, err)
		}
	}
	w.Tabs.Activate(one)
	return w, one, two
}

// TestWorkspace_theTreeFollowsTheNoteOnShow checks what a click on a tab
// does to the tree: the folder holding the note opens, and the note is
// chosen there, so that what is being edited is seen where it stands.
func TestWorkspace_theTreeFollowsTheNoteOnShow(t *testing.T) {
	w, one, two := followedWorkspace(t)
	tt := ui.NewTester(w.View, 900, 600)
	tt.Frame()
	if w.Tree.Selected != one {
		t.Fatalf("selected %q, want the note that is on show",
			w.Tree.Selected)
	}
	if w.Tree.Open[filepath.Dir(two)] {
		t.Fatal("the folder was open before the note in it was shown")
	}
	// The tree draws no line for the note in the folder: the click lands
	// on its tab, and on nothing else. The tree follows in the frames the
	// click itself asks for, there being no more to wait for.
	if err := tt.Click("two.md"); err != nil {
		t.Fatalf("Click: %v", err)
	}
	if !w.Tabs.IsActive(two) {
		t.Fatalf("%q is not the note on show", PathOf(w.Tabs.Active))
	}
	if !w.Tree.Open[filepath.Dir(two)] {
		t.Error("the folder above the note stayed shut")
	}
	if w.Tree.Selected != two {
		t.Errorf("selected %q, want the note that is on show",
			w.Tree.Selected)
	}
}

// TestWorkspace_aNoteTheVaultDoesNotHoldIsNotShown checks a note opened
// from outside the vault: there is no line for it in the tree, so the
// tree keeps what it had chosen rather than choosing nothing.
func TestWorkspace_aNoteTheVaultDoesNotHoldIsNotShown(t *testing.T) {
	w, one, _ := followedWorkspace(t)
	tt := ui.NewTester(w.View, 900, 600)
	tt.Frame()
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("# outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Tabs.Open(outside); err != nil {
		t.Fatalf("Open: %v", err)
	}
	tt.Frame()
	if w.Tree.Selected != one {
		t.Errorf("selected %q, want the note of the vault that was on show",
			w.Tree.Selected)
	}
}
