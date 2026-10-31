package markdown

import (
	"strings"
	"testing"
)

// resolver is a vault that has the notes named here, and no others.
func resolver(notes ...string) func(string) (string, bool) {
	return func(target string) (string, bool) {
		for _, note := range notes {
			if note == target {
				return "/vault/" + note + ".md", true
			}
		}
		return "", false
	}
}

// TestLinks_aLinkToANoteIsMadeClickable checks the one thing this is for:
// what was written as [[a note]] is shown as a link that can be clicked.
func TestLinks_aLinkToANoteIsMadeClickable(t *testing.T) {
	got := Links("see [[a note]] here", resolver("a note"))
	if !strings.Contains(got, "[a note](xinkgo:a%20note)") {
		t.Fatalf("got %q, want a link to the note", got)
	}
	if strings.Contains(got, "[[") {
		t.Fatalf("got %q, want the brackets gone", got)
	}
}

// TestLinks_oneThatNamesNothingIsLeftAsItWas checks that a link to a note
// the vault has not is not made into one that goes nowhere: a draft may
// well point at what is still to be written.
func TestLinks_oneThatNamesNothingIsLeftAsItWas(t *testing.T) {
	got := Links("see [[not yet]] here", resolver("a note"))
	if got != "see [[not yet]] here" {
		t.Fatalf("got %q, want it left as it was written", got)
	}
}

// TestLinks_theAliasIsWhatTheReaderSees checks that what the user wrote
// after the bar is what is shown, and not what the link names.
func TestLinks_theAliasIsWhatTheReaderSees(t *testing.T) {
	got := Links("see [[a note|this one]]", resolver("a note"))
	if !strings.Contains(got, "[this one](xinkgo:a%20note)") {
		t.Fatalf("got %q, want the alias shown", got)
	}
}

// TestLinks_whatStandsInCodeIsLeftAlone checks that `[[x]]` in code is
// code: a fence of code, and backticks on a line.
func TestLinks_whatStandsInCodeIsLeftAlone(t *testing.T) {
	text := "```\n[[a note]]\n```\n\nand `[[a note]]` inline\n"
	got := Links(text, resolver("a note"))
	if got != text {
		t.Fatalf("got %q, want the code as it was", got)
	}
}

// TestLinks_moreThanOneOnALineAndMoreThanOneLine checks that every link
// of a line is written out, and that the lines keep their places.
func TestLinks_moreThanOneOnALineAndMoreThanOneLine(t *testing.T) {
	text := "one [[a]] two [[b]]\n\nthree\n"
	got := Links(text, resolver("a", "b"))
	want := "one [a](xinkgo:a) two [b](xinkgo:b)\n\nthree\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// TestWikiTarget_whatADestinationNames checks the way back: what was
// written into a link is read out of it again, spaces and all.
func TestWikiTarget_whatADestinationNames(t *testing.T) {
	got, ok := WikiTarget(WikiHref("a note"))
	if !ok || got != "a note" {
		t.Fatalf("WikiTarget = %q, %v", got, ok)
	}
	// One that is already undone reads the same.
	if got, ok := WikiTarget("xinkgo:a note"); !ok || got != "a note" {
		t.Fatalf("WikiTarget = %q, %v", got, ok)
	}
	// A link to the web is not one to a note.
	if _, ok := WikiTarget("https://go.dev"); ok {
		t.Fatal("a page on the web was read as a note")
	}
}
