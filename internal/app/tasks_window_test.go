package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/helightxu/xinkgo/internal/index"
	"github.com/helightxu/xinkgo/internal/task"
)

// appOn is an app with a vault of three notes open, two of them carrying
// the tag AI.
func appOn(t *testing.T) (*appState, string) {
	t.Helper()
	root := t.TempDir()
	paths := []string{}
	texts := map[string]string{
		"a.md": "# A\n\nhas #AI\n",
		"b.md": "# B\n\nhas #AI too\n",
		"c.md": "# C\n\nnothing here\n",
	}
	for name, text := range texts {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	idx := index.New()
	idx.Build(root, paths)
	return &appState{root: root, idx: idx}, root
}

// TestPlanTask_whatItWouldDoIsSaidBeforeAnythingIsWritten checks the one
// thing a plan is for: the vault is read, and nothing is changed.
func TestPlanTask_whatItWouldDoIsSaidBeforeAnythingIsWritten(t *testing.T) {
	s, root := appOn(t)
	s.taskDraft.Kind = task.Retag.ID()
	s.taskDraft.Target = "AI"
	s.taskDraft.NewTag = "人工智能"
	s.planTask()
	if s.work == nil {
		t.Fatal("the task was not planned")
	}
	if len(s.work.Items) != 3 {
		t.Fatalf("items = %v, want all three notes", s.work.Items)
	}
	for _, path := range []string{"a.md", "b.md", "c.md"} {
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "人工智能") {
			t.Fatalf("%s was written by planning", path)
		}
	}
	// The two that have it would change; the one that has it not is
	// skipped, with the reason said.
	if s.work.Items[0].State.Finished() {
		t.Fatal("a note that would change was not left to be done")
	}
	skipped := 0
	for i := range s.work.Items {
		if s.work.Items[i].State.Kind == task.ItemSkipped &&
			s.work.Items[i].State.Reason != "" {
			skipped++
		}
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d, want the one without the tag", skipped)
	}
}

// TestPlanTask_aTagThatWasNotNamedIsNotPlanned checks that a task with no
// tag to work on is not set going over the whole vault.
func TestPlanTask_aTagThatWasNotNamedIsNotPlanned(t *testing.T) {
	s, _ := appOn(t)
	s.taskDraft.Kind = task.Retag.ID()
	s.taskDraft.Target = ""
	s.taskDraft.NewTag = "x"
	s.planTask()
	if s.work != nil {
		t.Fatal("a task with no tag was planned")
	}
	// A tag changed into nothing is not a thing either.
	s.taskDraft.Target = "AI"
	s.taskDraft.NewTag = ""
	s.planTask()
	if s.work != nil {
		t.Fatal("a tag changed into nothing was planned")
	}
}

// TestRunTask_theBatchIsWrittenAndTheVaultKnownAgain checks that running
// writes the notes and puts what changed back into the index.
func TestRunTask_theBatchIsWrittenAndTheVaultKnownAgain(t *testing.T) {
	s, root := appOn(t)
	s.taskDraft.Kind = task.Retag.ID()
	s.taskDraft.Target = "AI"
	s.taskDraft.NewTag = "人工智能"
	s.planTask()
	s.runTask()
	if !s.work.Finished() {
		t.Fatalf("status = %v, want it finished", s.work.Status)
	}
	for _, name := range []string{"a.md", "b.md"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "#人工智能") {
			t.Fatalf("%s = %q, want the tag changed", name, data)
		}
	}
	// What stands on the disk is not what the index held: it is told
	// again.
	if got := s.idx.Tags(); len(got) != 1 || got[0].Name != "人工智能" ||
		got[0].Count != 2 {
		t.Fatalf("tags = %v, want 人工智能 twice", got)
	}
	// The note without the tag was left alone.
	data, _ := os.ReadFile(filepath.Join(root, "c.md"))
	if strings.Contains(string(data), "人工智能") {
		t.Fatal("a note without the tag was written to")
	}
}

// TestRunTask_aNoteThatCouldNotBeWrittenIsSaidSo checks that one failure
// does not stop the rest, and is not hidden.
func TestRunTask_aNoteThatCouldNotBeWrittenIsSaidSo(t *testing.T) {
	s, root := appOn(t)
	// One of them is put where it can be read but not written: a save
	// makes its temporary file in the same folder, so a folder that
	// cannot be written to is a note that cannot be saved.
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "d.md"),
		[]byte("# D\n\nhas #AI\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(sub, 0o555); err != nil {
		t.Fatal(err)
	}
	// So that the vault can be cleaned up after the test.
	t.Cleanup(func() { _ = os.Chmod(sub, 0o755) })

	s.idx.Build(root, []string{
		filepath.Join(root, "a.md"), filepath.Join(root, "b.md"),
		filepath.Join(root, "c.md"), filepath.Join(sub, "d.md"),
	})
	s.taskDraft.Kind = task.Retag.ID()
	s.taskDraft.Target = "AI"
	s.taskDraft.NewTag = "x"
	s.planTask()
	s.runTask()

	failed := 0
	done := 0
	for i := range s.work.Items {
		switch s.work.Items[i].State.Kind {
		case task.ItemFailed:
			if s.work.Items[i].State.Reason != "" {
				failed++
			}
		case task.ItemDone:
			done++
		}
	}
	if failed != 1 {
		t.Fatalf("failed = %d (%v), want the one that could not be"+
			" written", failed, s.work.Items)
	}
	// The rest went on all the same.
	if done != 2 {
		t.Fatalf("done = %d, want the other two", done)
	}
}

// TestWriteNote_aTaskWorksOnTheVaultAndNothingElse checks that a note
// outside the vault is not written, however it came to be named.
func TestWriteNote_aTaskWorksOnTheVaultAndNothingElse(t *testing.T) {
	s, root := appOn(t)
	outside := filepath.Join(filepath.Dir(root), "outside.md")
	if err := os.WriteFile(outside, []byte("untouched"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.writeNote(outside, "changed"); err == nil {
		t.Fatal("a note outside the vault was written")
	}
	if data, _ := os.ReadFile(outside); string(data) != "untouched" {
		t.Fatal("the note outside the vault was changed")
	}
	if _, err := s.readNote(outside); err == nil {
		t.Fatal("a note outside the vault was read")
	}
}

// TestWithinRoot_whatStandsInsideTheVault checks the test a task's every
// read and write goes through.
func TestWithinRoot_whatStandsInsideTheVault(t *testing.T) {
	root := t.TempDir()
	if !withinRoot(root, filepath.Join(root, "a.md")) {
		t.Fatal("a note of the vault was said to be outside it")
	}
	for _, path := range []string{
		filepath.Join(filepath.Dir(root), "outside.md"),
		root + "/../outside.md",
		"/etc/passwd",
		"",
	} {
		if withinRoot(root, path) {
			t.Errorf("%q was said to be inside the vault", path)
		}
	}
	// A vault that has not been opened has no inside.
	if withinRoot("", filepath.Join(root, "a.md")) {
		t.Error("a vault that is not open was said to hold a note")
	}
}

// TestKindOf_theKindNamedByThatWord checks the word the window shows and
// the kind the task is, both ways.
func TestKindOf_theKindNamedByThatWord(t *testing.T) {
	cases := map[string]task.Kind{
		task.Retag.ID():     task.Retag,
		task.AddTag.ID():    task.AddTag,
		task.RemoveTag.ID(): task.RemoveTag,
		"something else":    task.Retag,
	}
	for id, want := range cases {
		if got := kindOf(id); got != want {
			t.Errorf("kindOf(%q) = %v, want %v", id, got, want)
		}
	}
}

// TestShortPath_aPathAsTheVaultKnowsIt checks that the root is not said
// over and over in a list of notes.
func TestShortPath_aPathAsTheVaultKnowsIt(t *testing.T) {
	root := t.TempDir()
	got := shortPath(root, filepath.Join(root, "sub", "a.md"))
	if got != filepath.Join("sub", "a.md") {
		t.Fatalf("path = %q, want it from the root", got)
	}
	if got := shortPath("", filepath.Join(root, "a.md")); got !=
		filepath.Join(root, "a.md") {
		t.Fatalf("path = %q, want it whole", got)
	}
}
