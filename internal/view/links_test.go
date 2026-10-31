package view

import (
	"strings"
	"testing"

	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/index"
)

// windowWithTwoNotes is a window whose vault holds the two notes of
// `vaultWithNotes`, and knows what they say.
func windowWithTwoNotes(t *testing.T) (*Workspace, string, string) {
	t.Helper()
	root, a, b := vaultWithNotes(t)
	w := NewWorkspace(config.Default())
	idx := index.New()
	idx.Build(root, []string{a, b})
	w.Index = idx
	if _, err := w.Tabs.Open(a); err != nil {
		t.Fatalf("Open: %v", err)
	}
	return w, a, b
}

// TestPreviewText_aLinkToANoteCanBeClicked checks what the preview is
// given: `[[b]]` is written as a link, and not left as brackets the
// reader cannot follow.
func TestPreviewText_aLinkToANoteCanBeClicked(t *testing.T) {
	w, _, _ := windowWithTwoNotes(t)
	n := w.Tabs.ActiveNote()
	n.Note.Text = "see [[b]] and [[nothing]]"
	got := w.previewText(n)
	if !strings.Contains(got, "[b](xinkgo:b)") {
		t.Fatalf("preview = %q, want a link to the note", got)
	}
	// What the vault has not is left as it was written.
	if !strings.Contains(got, "[[nothing]]") {
		t.Fatalf("preview = %q, want the note that is not there kept",
			got)
	}
}

// TestOpenLink_aLinkToANoteOpensIt checks what a click does: the note the
// link names is opened.
func TestOpenLink_aLinkToANoteOpensIt(t *testing.T) {
	w, _, b := windowWithTwoNotes(t)
	w.openLink("xinkgo:b", w.Tabs.Active)
	if _, ok := w.Tabs.Notes[b]; !ok && w.Tabs.Active != b {
		t.Fatalf("open = %v, want the note the link names",
			w.Tabs.Paths())
	}
}

// TestOpenLink_aNoteTheVaultHasNotIsSaid checks that a link whose note is
// not there says so, rather than opening nothing.
func TestOpenLink_aNoteTheVaultHasNotIsSaid(t *testing.T) {
	w, _, _ := windowWithTwoNotes(t)
	w.openLink("xinkgo:nothing", w.Tabs.Active)
	if !strings.Contains(w.status, "nothing") {
		t.Fatalf("status = %q, want it said", w.status)
	}
	if len(w.Tabs.Paths()) != 1 {
		t.Fatalf("open = %v, want only the one note", w.Tabs.Paths())
	}
}

// TestOpenLink_aPageOnTheWebGoesOutside checks that a link to a page is
// handed to the app, and is not looked for among the notes.
func TestOpenLink_aPageOnTheWebGoesOutside(t *testing.T) {
	w, _, _ := windowWithTwoNotes(t)
	got := ""
	w.OnOpenOutside = func(url string) { got = url }
	w.openLink("https://go.dev", w.Tabs.Active)
	if got != "https://go.dev" {
		t.Fatalf("outside = %q, want the page handed over", got)
	}
	if len(w.Tabs.Paths()) != 1 {
		t.Fatalf("open = %v, want the page not taken for a note",
			w.Tabs.Paths())
	}
}
