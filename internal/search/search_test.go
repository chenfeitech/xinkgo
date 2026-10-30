package search

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// vaultOf writes the notes given, and gives back their paths.
func vaultOf(t *testing.T, files map[string]string) []string {
	t.Helper()
	root := t.TempDir()
	paths := make([]string, 0, len(files))
	for name, text := range files {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	return paths
}

// TestFiles_findsTheWordsInEveryNote checks the one thing a search does.
func TestFiles_findsTheWordsInEveryNote(t *testing.T) {
	paths := vaultOf(t, map[string]string{
		"a.md": "# A\nthe words are here\n",
		"b.md": "nothing of the kind\n",
		"c.md": "the WORDS again\n",
	})
	hits := Sort(paths, "the words")
	got := make([]string, 0, len(hits))
	for _, h := range hits {
		got = append(got, filepath.Base(h.Path)+":"+
			strconv.Itoa(h.Line))
	}
	want := []string{"a.md:1", "c.md:0"}
	if len(got) != len(want) {
		t.Fatalf("hits %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("hit %d is %s, want %s", i, got[i], want[i])
		}
	}
}

// Sort is Sortable, as one line reads.
func Sort(paths []string, query string) []Hit {
	hits := Files(paths, query, 0)
	Sortable(hits)
	return hits
}

// TestFiles_theCaseOfLettersMattersNot checks that what is looked for is
// found however it was written.
func TestFiles_theCaseOfLettersMattersNot(t *testing.T) {
	paths := vaultOf(t, map[string]string{"a.md": "Rust and rust\n"})
	if hits := Sort(paths, "RUST"); len(hits) != 1 {
		t.Errorf("%d hits, want 1", len(hits))
	}
}

// TestFiles_nothingTypedFindsNothing checks a search of no words at all.
func TestFiles_nothingTypedFindsNothing(t *testing.T) {
	paths := vaultOf(t, map[string]string{"a.md": "text\n"})
	if hits := Sort(paths, "   "); len(hits) != 0 {
		t.Errorf("%d hits, want none", len(hits))
	}
}

// TestFiles_noMoreThanAskedFor checks the limit: a hundred is as many as
// are worth showing.
func TestFiles_noMoreThanAskedFor(t *testing.T) {
	text := strings.Repeat("word\n", 300)
	paths := vaultOf(t, map[string]string{"a.md": text})
	if hits := Sort(paths, "word"); len(hits) != MaxHits {
		t.Errorf("%d hits, want %d", len(hits), MaxHits)
	}
}

// TestFiles_aFileOfGreatSizeIsLeftOut checks that a note too large to
// read line by line is passed over rather than slowing the search.
func TestFiles_aFileOfGreatSizeIsLeftOut(t *testing.T) {
	paths := vaultOf(t, map[string]string{"a.md": "small\n"})
	big := filepath.Join(filepath.Dir(paths[0]), "big.md")
	if err := os.WriteFile(big, make([]byte, MaxFileSize+1), 0o644); err != nil {
		t.Fatal(err)
	}
	hits := Sort(append(paths, big), "small")
	if len(hits) != 1 || hits[0].Path != paths[0] {
		t.Errorf("hits %v, want only the small note", hits)
	}
}

// TestFiles_aFileThatCannotBeReadIsPassedOver checks a file that is gone
// between the looking and the reading.
func TestFiles_aFileThatCannotBeReadIsPassedOver(t *testing.T) {
	paths := vaultOf(t, map[string]string{"a.md": "text\n"})
	gone := filepath.Join(filepath.Dir(paths[0]), "gone.md")
	hits := Sort(append(paths, gone), "text")
	if len(hits) != 1 {
		t.Errorf("%d hits, want the one note that is there", len(hits))
	}
}

// TestTrimContext_aShortLineAsItIs checks that a line that fits is shown
// whole.
func TestTrimContext_aShortLineAsItIs(t *testing.T) {
	if got := TrimContext("  a short line  "); got != "a short line" {
		t.Errorf("the line is %q, want it trimmed", got)
	}
}

// TestTrimContext_aLongLineIsCutToBothEnds checks that a long line keeps
// a beginning and an end, and says that it was cut.
func TestTrimContext_aLongLineIsCutToBothEnds(t *testing.T) {
	long := strings.Repeat("a", 300) + "middle" + strings.Repeat("b", 300)
	got := TrimContext(long)
	if !strings.Contains(got, "…") {
		t.Error("the line was cut without saying so")
	}
	if len([]rune(got)) >= len([]rune(long)) {
		t.Error("the long line was not cut")
	}
	if !strings.HasSuffix(got, strings.Repeat("b", 60)) {
		t.Error("the end of the line was not kept")
	}
}

// TestFiles_theWordsAreFoundInChinese checks that what is looked for is
// found in a language with no case of letters at all.
func TestFiles_theWordsAreFoundInChinese(t *testing.T) {
	paths := vaultOf(t, map[string]string{"a.md": "一些文字在这里\n"})
	if hits := Sort(paths, "文字"); len(hits) != 1 {
		t.Errorf("%d hits, want 1", len(hits))
	}
}
