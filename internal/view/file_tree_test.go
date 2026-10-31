package view

import (
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
