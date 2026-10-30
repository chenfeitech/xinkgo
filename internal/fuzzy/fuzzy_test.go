package fuzzy

import (
	"strings"
	"testing"
)

// TestMatch_lettersInOrder checks the one thing that makes a match: the
// letters were typed in the order they stand.
func TestMatch_lettersInOrder(t *testing.T) {
	if got := Match("notes/b.md", "nb"); !got.Found() {
		t.Error("nb was not found in notes/b.md")
	}
	if got := Match("notes/b.md", "bn"); got.Found() {
		t.Error("bn was found in notes/b.md, and it is not there in order")
	}
	if got := Match("notes/b.md", "zz"); got.Found() {
		t.Error("zz was found in notes/b.md")
	}
}

// TestMatch_theCaseOfLettersMattersNot checks that what was typed finds
// the name whether it was typed in capitals or not.
func TestMatch_theCaseOfLettersMattersNot(t *testing.T) {
	if got := Match("Notes/Readme.md", "readme"); !got.Found() {
		t.Error("readme was not found in Notes/Readme.md")
	}
}

// TestMatch_lettersTogetherAreWorthMore checks that what stands together
// in the name is a better answer than what is spread through it.
func TestMatch_lettersTogetherAreWorthMore(t *testing.T) {
	together := Match("abc.md", "abc").Score
	spread := Match("a_x_b_x_c.md", "abc").Score
	if together <= spread {
		t.Errorf("abc in abc.md scored %d, in a_x_b_x_c.md %d: want the "+
			"letters standing together to be worth more", together, spread)
	}
}

// TestMatch_theStartOfAWordIsWorthMore checks that a letter where a word
// begins is worth more than one in the middle of it.
func TestMatch_theStartOfAWordIsWorthMore(t *testing.T) {
	word := Match("notes/b.md", "b").Score
	middle := Match("notes/abc.md", "b").Score
	if word <= middle {
		t.Errorf("b after a slash scored %d, in a word %d: want the one "+
			"at the start of a word to be worth more", word, middle)
	}
}

// TestMatch_theWholeOfItAtTheFront checks a name that begins with what
// was typed.
func TestMatch_theWholeOfItAtTheFront(t *testing.T) {
	front := Match("readme.md", "read").Score
	elsewhere := Match("my-readme.md", "read").Score
	if front <= elsewhere {
		t.Errorf("read at the front scored %d, further in %d: want the "+
			"front", front, elsewhere)
	}
}

// TestMatch_whereTheLettersWereFound checks what the window lights up.
func TestMatch_whereTheLettersWereFound(t *testing.T) {
	got := Match("abc", "ac")
	want := []int{0, 2}
	if len(got.Positions) != len(want) {
		t.Fatalf("positions %v, want %v", got.Positions, want)
	}
	for i := range want {
		if got.Positions[i] != want[i] {
			t.Errorf("position %d is %d, want %d", i, got.Positions[i],
				want[i])
		}
	}
}

// TestSearch_theBestFirst checks the order the answers come in.
func TestSearch_theBestFirst(t *testing.T) {
	names := []string{"zebra.md", "alpha.md", "alphabet.md"}
	got := Search(names, "alpha", 0)
	if len(got) != 2 {
		t.Fatalf("%d answers, want 2", len(got))
	}
	if got[0].Text != "alpha.md" {
		t.Errorf("the first answer is %q, want alpha.md", got[0].Text)
	}
}

// TestSearch_noMoreThanAskedFor checks the limit: the command palette
// shows the first fifty, and no more.
func TestSearch_noMoreThanAskedFor(t *testing.T) {
	names := []string{"a.md", "b.md", "c.md", "d.md"}
	if got := Search(names, "", 2); len(got) != 2 {
		t.Errorf("%d answers, want 2", len(got))
	}
}

// TestSearch_nothingTypedGivesEverything checks a query of nothing.
func TestSearch_nothingTypedGivesEverything(t *testing.T) {
	names := []string{"b.md", "a.md"}
	got := Search(names, "", 0)
	if len(got) != 2 {
		t.Fatalf("%d answers, want 2", len(got))
	}
	if got[0].Text != "a.md" || got[1].Text != "b.md" {
		t.Errorf("answers %q and %q, want them in order", got[1].Text,
			got[0].Text)
	}
}

// TestSegments_whatWasFoundAndWhatWasNot checks how a name is broken up
// for the window to light up.
func TestSegments_whatWasFoundAndWhatWasNot(t *testing.T) {
	segments := Segments("abc", Match("abc", "ac").Positions)
	var marked, plain strings.Builder
	for _, s := range segments {
		if s.Found {
			marked.WriteString(s.Text)
		} else {
			plain.WriteString(s.Text)
		}
	}
	if marked.String() != "ac" {
		t.Errorf("the letters found are %q, want ac", marked.String())
	}
	if plain.String() != "b" {
		t.Errorf("the letters not found are %q, want b", plain.String())
	}
}

// TestSegments_nothingFoundIsOnePiece checks a name that was not
// matched.
func TestSegments_nothingFoundIsOnePiece(t *testing.T) {
	segments := Segments("abc", nil)
	if len(segments) != 1 || segments[0].Text != "abc" ||
		segments[0].Found {
		t.Errorf("segments %v, want the name as it is", segments)
	}
}
