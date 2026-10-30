package markdown

import (
	"strings"
	"testing"
)

// tagNames is what a note is tagged with, and nothing else about it.
func tagNames(text string) []string {
	facts := ParseFacts(text)
	out := make([]string, 0, len(facts.Tags))
	for _, tag := range facts.Tags {
		out = append(out, tag.Name)
	}
	return out
}

// TestParseFacts_headingsWithLevelsAndLines checks the headings of a
// note: how deep each is, what it says, and where it stands.
func TestParseFacts_headingsWithLevelsAndLines(t *testing.T) {
	facts := ParseFacts("# one\ntext\n### three\n")
	if len(facts.Headings) != 2 {
		t.Fatalf("%d headings, want 2", len(facts.Headings))
	}
	if facts.Headings[0].Level != 1 || facts.Headings[0].Text != "one" ||
		facts.Headings[0].Line != 0 {
		t.Errorf("first heading is %+v, want level 1 one at line 0",
			facts.Headings[0])
	}
	if facts.Headings[1].Level != 3 || facts.Headings[1].Line != 2 {
		t.Errorf("second heading is %+v, want level 3 at line 2",
			facts.Headings[1])
	}
}

// TestParseFacts_closingHashesAndHashWithoutSpace checks that a heading
// keeps its text and not its closing marks, and that `#tag` is a tag
// rather than a heading.
func TestParseFacts_closingHashesAndHashWithoutSpace(t *testing.T) {
	facts := ParseFacts("## title ##\n#tag\n")
	if len(facts.Headings) != 1 {
		t.Fatalf("%d headings, want 1", len(facts.Headings))
	}
	if facts.Headings[0].Text != "title" {
		t.Errorf("text %q, want title", facts.Headings[0].Text)
	}
	if got := tagNames("#tag"); len(got) != 1 || got[0] != "tag" {
		t.Errorf("tags %v, want just tag", got)
	}
}

// TestParseFacts_aFencedBlockIsNoFact checks that what stands in a block
// of code is text the note shows, not something it says.
func TestParseFacts_aFencedBlockIsNoFact(t *testing.T) {
	facts := ParseFacts("```\n# no heading\n[[no link]]\n#notag\n```\n" +
		"# a heading\n")
	if len(facts.Headings) != 1 || facts.Headings[0].Text != "a heading" {
		t.Errorf("headings %v, want only the one after the block",
			facts.Headings)
	}
	if len(facts.WikiLinks) != 0 {
		t.Errorf("wiki links %v, want none from inside the block",
			facts.WikiLinks)
	}
	if len(facts.Tags) != 0 {
		t.Errorf("tags %v, want none from inside the block", facts.Tags)
	}
}

// TestParseFacts_aFenceNeverClosed checks that a fence that is never
// closed takes the rest of the note with it.
func TestParseFacts_aFenceNeverClosed(t *testing.T) {
	facts := ParseFacts("text\n```\n# heading\n")
	if len(facts.Headings) != 0 {
		t.Errorf("headings %v, want none after an open fence",
			facts.Headings)
	}
}

// TestParseFacts_inlineCodeIsNoFact checks what stands between
// backticks.
func TestParseFacts_inlineCodeIsNoFact(t *testing.T) {
	facts := ParseFacts("like `#tag` and `[no link](x)`\n")
	if len(facts.Tags) != 0 {
		t.Errorf("tags %v, want none from inside code", facts.Tags)
	}
	if len(facts.Links) != 0 {
		t.Errorf("links %v, want none from inside code", facts.Links)
	}
}

// TestParseFacts_linksImagesAndTitles checks that a link is read with
// where it points and what it says, an image among them, and that a
// title in quotes is not part of the target.
func TestParseFacts_linksImagesAndTitles(t *testing.T) {
	facts := ParseFacts("[doc](notes/a.md) ![pic](img/a.png) " +
		"[titled](b.md \"title\")\n")
	if len(facts.Links) != 3 {
		t.Fatalf("%d links, want 3", len(facts.Links))
	}
	want := []string{"notes/a.md", "img/a.png", "b.md"}
	for i, link := range facts.Links {
		if link.Target != want[i] {
			t.Errorf("link %d points at %q, want %q", i, link.Target,
				want[i])
		}
	}
	if facts.Links[0].Text != "doc" || facts.Links[2].Text != "titled" {
		t.Errorf("texts %q and %q, want doc and titled",
			facts.Links[0].Text, facts.Links[2].Text)
	}
}

// TestParseFacts_wikilinksWithAliasAndAnchor checks the three parts a
// link in double brackets may have.
func TestParseFacts_wikilinksWithAliasAndAnchor(t *testing.T) {
	facts := ParseFacts("[[a]] and [[b|alias]] and [[c#part|shown]]\n")
	if len(facts.WikiLinks) != 3 {
		t.Fatalf("%d wiki links, want 3", len(facts.WikiLinks))
	}
	if facts.WikiLinks[0].Target != "a" || facts.WikiLinks[0].Alias != "" {
		t.Errorf("first is %+v, want a with no alias", facts.WikiLinks[0])
	}
	if facts.WikiLinks[1].Alias != "alias" {
		t.Errorf("alias %q, want alias", facts.WikiLinks[1].Alias)
	}
	third := facts.WikiLinks[2]
	if third.Target != "c" || third.Anchor != "part" ||
		third.Alias != "shown" {
		t.Errorf("third is %+v, want c, part and shown", third)
	}
}

// TestParseFacts_emptyWikilinkIsDropped checks that brackets with
// nothing in them are no link.
func TestParseFacts_emptyWikilinkIsDropped(t *testing.T) {
	facts := ParseFacts("[[]] and [[ ]] and [[good]]\n")
	if len(facts.WikiLinks) != 1 || facts.WikiLinks[0].Target != "good" {
		t.Errorf("wiki links %v, want only good", facts.WikiLinks)
	}
}

// TestParseFacts_whatATagNeeds checks that a tag must be parted from
// what stands before it, and must not be made of digits alone.
func TestParseFacts_whatATagNeeds(t *testing.T) {
	if got := tagNames("#idea text #todo"); len(got) != 2 ||
		got[0] != "idea" || got[1] != "todo" {
		t.Errorf("tags %v, want idea and todo", got)
	}
	if got := tagNames("#123"); len(got) != 0 {
		t.Errorf("tags %v, want none: digits alone are no tag", got)
	}
	if got := tagNames("see [doc](a.md#part)"); len(got) != 0 {
		t.Errorf("tags %v, want none: an anchor of a link is no tag", got)
	}
	// A tag after a bracket is taken. The closing bracket goes with it:
	// every rune that is not ASCII may stand in a tag, as xink keeps it.
	if got := tagNames("（#中文）"); len(got) != 1 || got[0] != "中文）" {
		t.Errorf("tags %v, want the one after the bracket", got)
	}
}

// TestParseFacts_tagsInChineseNestedAndHyphen checks what a tag may
// hold: words of any language, one tag inside another, and a hyphen.
func TestParseFacts_tagsInChineseNestedAndHyphen(t *testing.T) {
	got := tagNames("#notes/idea #rust-lang #2026plan")
	want := []string{"notes/idea", "rust-lang", "2026plan"}
	if len(got) != len(want) {
		t.Fatalf("tags %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("tag %d is %q, want %q", i, got[i], want[i])
		}
	}
}

// TestParseFacts_frontmatterTags checks the tags written at the top of a
// note, and that what else is written there is no tag.
func TestParseFacts_frontmatterTags(t *testing.T) {
	facts := ParseFacts("---\ntitle: x\ntags: [idea, todo]\n---\n" +
		"text #more\n")
	got := make([]string, 0, len(facts.Tags))
	for _, tag := range facts.Tags {
		got = append(got, tag.Name)
	}
	want := []string{"idea", "todo", "more"}
	if len(got) != len(want) {
		t.Fatalf("tags %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("tag %d is %q, want %q", i, got[i], want[i])
		}
	}
	// The body starts at the fifth line, and so does its tag.
	if facts.Tags[2].Line != 4 {
		t.Errorf("the last tag is at line %d, want 4", facts.Tags[2].Line)
	}
}

// TestParseFacts_nothingYieldsNoFacts checks a note with nothing in it.
func TestParseFacts_nothingYieldsNoFacts(t *testing.T) {
	facts := ParseFacts("")
	if len(facts.Headings) != 0 || len(facts.Links) != 0 ||
		len(facts.WikiLinks) != 0 || len(facts.Tags) != 0 {
		t.Errorf("facts %+v, want nothing", facts)
	}
}

// TestParseFacts_aTildeFenceClosesItself checks that the two kinds of
// fence are told apart: one of tildes is closed by tildes and not by
// backticks.
func TestParseFacts_aTildeFenceClosesItself(t *testing.T) {
	facts := ParseFacts("~~~\n# in code\n```\n# still in code\n~~~\n" +
		"# heading\n")
	if len(facts.Headings) != 1 || facts.Headings[0].Text != "heading" {
		t.Errorf("headings %v, want only the one after the block",
			facts.Headings)
	}
}

// TestMaskInlineCode_keepsThePlace checks that what is masked out is
// made of spaces of the same length, so that what follows keeps its
// place.
func TestMaskInlineCode_keepsThePlace(t *testing.T) {
	got := maskInlineCode("a `code` #tag")
	if strings.Contains(got, "code") {
		t.Errorf("masked line is %q, want the code gone", got)
	}
	if len([]rune(got)) != len([]rune("a `code` #tag")) {
		t.Errorf("masked line is %d runes long, want as long as the line",
			len([]rune(got)))
	}
	if !strings.Contains(got, "#tag") {
		t.Errorf("masked line is %q, want the tag kept", got)
	}
}
