package view

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/vault"
)

// dragTree presses at (fromX, fromY), moves in steps to (toX, toY), and
// releases there, as a pointer dragging a row across the tree does.
func dragTree(tt *ui.Tester, fromX, fromY, toX, toY float32) {
	tt.Press(fromX, fromY)
	for k := 1; k <= 4; k++ {
		f := float32(k) / 4
		tt.Move(fromX+(toX-fromX)*f, fromY+(toY-fromY)*f)
	}
	tt.Release(toX, toY)
	tt.Frame()
}

// rowCenter is the middle of the row showing name, where a press lands.
func rowCenter(t *testing.T, tt *ui.Tester, name string) (float32, float32) {
	t.Helper()
	r, ok := tt.Find(name)
	if !ok {
		t.Fatalf("the tree draws no row for %q", name)
	}
	return r.X + r.W/2, r.Y + r.H/2
}

// nestedRoot is a vault with a folder inside a folder, for the moves a
// folder cannot make into itself or one of its own children.
func nestedRoot() *vault.Entry {
	return &vault.Entry{
		Name: "vault", Path: "/vault", IsDir: true,
		Children: []*vault.Entry{
			{Name: "plain.md", Path: "/vault/plain.md"},
			{Name: "outer", Path: "/vault/outer", IsDir: true,
				Children: []*vault.Entry{
					{Name: "inner", Path: "/vault/outer/inner",
						IsDir: true, Children: []*vault.Entry{
							{Name: "deep.md",
								Path: "/vault/outer/inner/deep.md"},
						}},
				}},
		},
	}
}

// TestTreeCanMoveTo_onlyLegitimateMoves checks the guard the tree puts on
// a drag before it asks the app to move anything.
func TestTreeCanMoveTo_onlyLegitimateMoves(t *testing.T) {
	tr := NewTree()
	tr.SetRoot(nestedRoot())
	cases := []struct {
		name string
		src  string
		dir  string
		want bool
	}{
		{"a file into another folder", "/vault/plain.md",
			"/vault/outer", true},
		{"a nested file back to the root", "/vault/outer/inner/deep.md",
			"/vault", true},
		{"the vault itself", "/vault", "/vault/outer", false},
		{"a file into the folder it is in", "/vault/plain.md",
			"/vault", false},
		{"a folder into itself", "/vault/outer", "/vault/outer", false},
		{"a folder into its own child", "/vault/outer",
			"/vault/outer/inner", false},
		{"a file into a sibling folder", "/vault/plain.md",
			"/vault/outer/inner", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := tr.canMoveTo(c.src, c.dir); got != c.want {
				t.Errorf("canMoveTo(%q, %q) = %v, want %v",
					c.src, c.dir, got, c.want)
			}
		})
	}
}

// recorder is an OnMoveTo that keeps what it was asked to move, and where.
type recorder struct {
	path, dir string
	calls     int
}

func (r *recorder) move(path, dir string) {
	r.path, r.dir, r.calls = path, dir, r.calls+1
}

// TestTreeDrag_aFileIntoAFolder checks the heart of the feature: dragging a
// note onto a folder asks the app to move it into that folder.
func TestTreeDrag_aFileIntoAFolder(t *testing.T) {
	tr := NewTree()
	tr.SetRoot(treeRoot())
	rec := &recorder{}
	tr.OnMoveTo = rec.move
	tt := treeTester(t, tr)
	tt.SetFocused(true)
	tt.Frame()
	fx, fy := rowCenter(t, tt, "a.md")
	tx, ty := rowCenter(t, tt, "notes")
	dragTree(tt, fx, fy, tx, ty)
	if rec.calls != 1 || rec.path != "/vault/a.md" ||
		rec.dir != "/vault/notes" {
		t.Fatalf("asked to move %q into %q (%d times), want a.md "+
			"into notes once", rec.path, rec.dir, rec.calls)
	}
}

// TestTreeDrag_aFileDroppedOnAnotherGoesToItsFolder checks that a drop on a
// file row, not a folder, moves the dragged file into the folder holding it.
func TestTreeDrag_aFileDroppedOnAnotherGoesToItsFolder(t *testing.T) {
	tr := NewTree()
	tr.SetRoot(treeRoot())
	tr.Toggle("/vault/notes") // so the note inside is drawn
	rec := &recorder{}
	tr.OnMoveTo = rec.move
	tt := treeTester(t, tr)
	tt.SetFocused(true)
	tt.Frame()
	fx, fy := rowCenter(t, tt, "b.md")
	tx, ty := rowCenter(t, tt, "a.md")
	dragTree(tt, fx, fy, tx, ty)
	if rec.calls != 1 || rec.path != "/vault/notes/b.md" ||
		rec.dir != "/vault" {
		t.Fatalf("asked to move %q into %q (%d times), want b.md "+
			"into the vault's own folder", rec.path, rec.dir, rec.calls)
	}
}

// TestTreeDrag_droppedOnEmptyGroundGoesToTheRoot checks that a file dropped
// where no row stands is moved to the vault's own folder.
func TestTreeDrag_droppedOnEmptyGroundGoesToTheRoot(t *testing.T) {
	tr := NewTree()
	tr.SetRoot(treeRoot())
	tr.Toggle("/vault/notes")
	rec := &recorder{}
	tr.OnMoveTo = rec.move
	tt := treeTester(t, tr)
	tt.SetFocused(true)
	tt.Frame()
	fx, fy := rowCenter(t, tt, "b.md")
	// Well below every row, above the foot: the tree's own ground.
	dragTree(tt, fx, fy, 130, 320)
	if rec.calls != 1 || rec.path != "/vault/notes/b.md" ||
		rec.dir != "/vault" {
		t.Fatalf("asked to move %q into %q (%d times), want b.md "+
			"into the vault's own folder", rec.path, rec.dir, rec.calls)
	}
}

// TestTreeDrag_aFolderCannotMoveIntoItself checks that a slip of the hand,
// dropping a folder on its own row, leaves the disk alone.
func TestTreeDrag_aFolderCannotMoveIntoItself(t *testing.T) {
	tr := NewTree()
	tr.SetRoot(treeRoot())
	rec := &recorder{}
	tr.OnMoveTo = rec.move
	tt := treeTester(t, tr)
	tt.SetFocused(true)
	tt.Frame()
	fx, fy := rowCenter(t, tt, "notes")
	dragTree(tt, fx, fy, fx, fy+treeRowHeight)
	if rec.calls != 0 {
		t.Fatalf("a folder dropped on itself moved %q into %q",
			rec.path, rec.dir)
	}
}

// TestTreeDrag_droppedInTheSameFolderDoesNothing checks that a file let go
// on a neighbour in the folder it already stands in is not moved.
func TestTreeDrag_droppedInTheSameFolderDoesNothing(t *testing.T) {
	tr := NewTree()
	tr.SetRoot(treeRoot())
	rec := &recorder{}
	tr.OnMoveTo = rec.move
	tt := treeTester(t, tr)
	tt.SetFocused(true)
	tt.Frame()
	fx, fy := rowCenter(t, tt, "a.md")
	tx, ty := rowCenter(t, tt, "pic.png")
	dragTree(tt, fx, fy, tx, ty)
	if rec.calls != 0 {
		t.Fatalf("a file dropped on its neighbour moved %q into %q",
			rec.path, rec.dir)
	}
}

// TestTreeDrag_aFolderCannotMoveIntoItsOwnChild checks the drag path as
// well as the guard: dropping a folder onto a folder inside it is refused.
func TestTreeDrag_aFolderCannotMoveIntoItsOwnChild(t *testing.T) {
	tr := NewTree()
	tr.SetRoot(nestedRoot())
	tr.Toggle("/vault/outer")
	tr.Toggle("/vault/outer/inner")
	rec := &recorder{}
	tr.OnMoveTo = rec.move
	tt := treeTester(t, tr)
	tt.SetFocused(true)
	tt.Frame()
	fx, fy := rowCenter(t, tt, "outer")
	tx, ty := rowCenter(t, tt, "inner")
	dragTree(tt, fx, fy, tx, ty)
	if rec.calls != 0 {
		t.Fatalf("a folder dropped on its child moved %q into %q",
			rec.path, rec.dir)
	}
}

// renameRecorder keeps the name a rename settled on.
type renameRecorder struct {
	path, name string
	calls      int
}

func (r *renameRecorder) rename(path, name string) {
	r.path, r.name, r.calls = path, name, r.calls+1
}

// TestTreeQuickRename_BeginFocusesAndKeepsTheExtension checks that a rename
// begun through the shortcut path takes the focus with the name before its
// extension selected, so typing replaces it and the ".md" is kept.
func TestTreeQuickRename_BeginFocusesAndKeepsTheExtension(t *testing.T) {
	tr := NewTree()
	tr.SetRoot(treeRoot())
	rec := &renameRecorder{}
	tr.OnRename = rec.rename
	tr.BeginRename("/vault/a.md", "a.md")
	tt := treeTester(t, tr)
	tt.SetFocused(true)
	tt.Frame() // the field appears, focused, with "a" chosen
	tt.Type("renamed")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if rec.calls != 1 || rec.path != "/vault/a.md" ||
		rec.name != "renamed.md" {
		t.Fatalf("renamed %q to %q (%d times), want a.md to "+
			"renamed.md once", rec.path, rec.name, rec.calls)
	}
	if tr.Renaming != "" {
		t.Error("the tree still asks for a name after Enter")
	}
}

// TestTreeQuickRename_escapeLeavesTheFileAlone checks that Escape gives up
// a rename without asking the app to touch the disk.
func TestTreeQuickRename_escapeLeavesTheFileAlone(t *testing.T) {
	tr := NewTree()
	tr.SetRoot(treeRoot())
	rec := &renameRecorder{}
	tr.OnRename = rec.rename
	tr.BeginRename("/vault/a.md", "a.md")
	tt := treeTester(t, tr)
	tt.SetFocused(true)
	tt.Frame()
	tt.Type("renamed")
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if rec.calls != 0 {
		t.Fatalf("Escape renamed %q to %q", rec.path, rec.name)
	}
	if tr.Renaming != "" {
		t.Error("the tree still asks for a name after Escape")
	}
}

// TestTreeQuickRename_doubleClickBeginsOne checks the tree's own quick way
// in: a double click on a row starts asking for its name.
func TestTreeQuickRename_doubleClickBeginsOne(t *testing.T) {
	tr := NewTree()
	tr.SetRoot(treeRoot())
	tt := treeTester(t, tr)
	tt.SetFocused(true)
	tt.Frame()
	x, y := rowCenter(t, tt, "a.md")
	tt.ClickAt(x, y)
	tt.ClickAt(x, y)
	tt.Frame()
	if tr.Renaming != "/vault/a.md" {
		t.Fatalf("a double click began renaming %q, want a.md",
			tr.Renaming)
	}
	if tr.Name != "a.md" {
		t.Errorf("the field holds %q, want the name it stands under",
			tr.Name)
	}
}

// TestTreeQuickRename_clickingAwayKeepsTheName checks that moving on to
// another row commits the rename that was under way, rather than leaving
// the field open behind the user.
func TestTreeQuickRename_clickingAwayKeepsTheName(t *testing.T) {
	tr := NewTree()
	tr.SetRoot(treeRoot())
	rec := &renameRecorder{}
	tr.OnRename = rec.rename
	tr.BeginRename("/vault/a.md", "a.md")
	tt := treeTester(t, tr)
	tt.SetFocused(true)
	tt.Frame()
	tr.Name = "changed.md" // as though typed into the field
	if err := tt.Click("pic.png"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if rec.calls != 1 || rec.path != "/vault/a.md" ||
		rec.name != "changed.md" {
		t.Fatalf("clicking away renamed %q to %q (%d times), want "+
			"a.md to changed.md once", rec.path, rec.name, rec.calls)
	}
}

// TestTreeQuickRename_theContextMenuStillStartsOne checks that the older
// way in, the context menu's Rename, shares the new field and its focus.
func TestTreeQuickRename_theContextMenuStillStartsOne(t *testing.T) {
	tr := NewTree()
	tr.SetRoot(treeRoot())
	rec := &renameRecorder{}
	tr.OnRename = rec.rename
	tr.OnNewNoteIn = func(string) {}
	tr.OnNewFolderIn = func(string) {}
	tt := treeTester(t, tr)
	tt.SetFocused(true)
	tt.Frame()
	x, y := rowCenter(t, tt, "a.md")
	tt.RightClickAt(x, y)
	tt.Frame()
	if !hasMenuItem(tt.Menu(), locale.TreeRename()) {
		t.Fatalf("the menu holds %q, want %q", tt.Menu(),
			locale.TreeRename())
	}
	if err := tt.ChooseMenuItem(locale.TreeRename()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tr.Renaming != "/vault/a.md" {
		t.Fatalf("the menu began renaming %q, want a.md", tr.Renaming)
	}
	tt.Type("again")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if rec.calls != 1 || rec.name != "again.md" {
		t.Errorf("renamed to %q (%d times), want again.md once",
			rec.name, rec.calls)
	}
}

// TestTreeDrag_usesOSPaths checks the guard against paths as the platform
// writes them, not only the slash-separated ones the fixtures above use.
func TestTreeDrag_usesOSPaths(t *testing.T) {
	tr := NewTree()
	root := nestedRoot()
	tr.SetRoot(root)
	src := filepath.Join("/vault", "outer")
	child := filepath.Join(src, "inner")
	if tr.canMoveTo(src, child) {
		t.Errorf("a folder could move into its own child %q", child)
	}
	if !tr.canMoveTo(filepath.Join(child, "deep.md"), "/vault") {
		t.Error("a nested file could not move back to the root")
	}
}

// hasMenuItem reports whether a menu's labels include s.
func hasMenuItem(items []string, s string) bool {
	for _, one := range items {
		if one == s {
			return true
		}
	}
	return false
}
