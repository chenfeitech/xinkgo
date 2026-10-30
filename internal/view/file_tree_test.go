package view

import (
	"testing"

	"github.com/egoist/mygo/ui"
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
