package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWritePage_thePageIsPutBesideThePdfItBecomes checks where the page to
// be printed is written: next to the PDF it becomes, and not in the
// machine's own folder of temporary files. A note whose pictures stand
// beside the page names them by a path relative to it, and a webview reads
// them only from where the page is.
func TestWritePage_thePageIsPutBesideThePdfItBecomes(t *testing.T) {
	dir := t.TempDir()
	name, err := writePage(dir, []byte("<p>a note</p>"))
	if err != nil {
		t.Fatalf("writePage: %v", err)
	}
	if filepath.Dir(name) != dir {
		t.Fatalf("the page is at %q, want it in %q", name, dir)
	}
	if !strings.HasSuffix(name, ".html") {
		t.Errorf("the page is named %q, want it to end in .html", name)
	}
	got, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read the page: %v", err)
	}
	if string(got) != "<p>a note</p>" {
		t.Errorf("the page holds %q, want %q", got, "<p>a note</p>")
	}
}

// TestWritePage_twoPagesDoNotTakeEachOthersPlace checks that a second page
// printed into the same folder stands on its own: the window reads the
// first while the second is being written.
func TestWritePage_twoPagesDoNotTakeEachOthersPlace(t *testing.T) {
	dir := t.TempDir()
	first, err := writePage(dir, []byte("one"))
	if err != nil {
		t.Fatalf("writePage: %v", err)
	}
	second, err := writePage(dir, []byte("two"))
	if err != nil {
		t.Fatalf("writePage: %v", err)
	}
	if first == second {
		t.Fatalf("both pages are at %q, want two of them", first)
	}
	one, err := os.ReadFile(first)
	if err != nil {
		t.Fatalf("read the first page: %v", err)
	}
	if string(one) != "one" {
		t.Errorf("the first page holds %q, want %q", one, "one")
	}
}

// TestWritePage_aFolderThereIsNoWritingInIsSaid checks the way out of a
// folder the page cannot be put in: it is said, rather than a printing
// that would wait for a page that was never written.
func TestWritePage_aFolderThereIsNoWritingInIsSaid(t *testing.T) {
	_, err := writePage(filepath.Join(t.TempDir(), "gone"), []byte("a note"))
	if err == nil {
		t.Fatal("writePage wrote into a folder that is not there")
	}
}
