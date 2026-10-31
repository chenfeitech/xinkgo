package ai

import (
	"strings"
	"testing"

	"github.com/helightxu/xinkgo/internal/index"
)

// vaultKnown is an index of two notes, for the verifier to resolve against.
type vaultKnown struct {
	notes map[string]string
	many  map[string][]string
}

func (k vaultKnown) Resolve(target string) index.Resolution {
	if paths, ok := k.many[target]; ok {
		return index.Resolution{Kind: index.Ambiguous,
			Candidates: paths}
	}
	if path, ok := k.notes[target]; ok {
		return index.Resolution{Kind: index.Hit, Path: path}
	}
	return index.Resolution{Kind: index.Missing}
}

// TestVerify_aLinkThatPointsNowhereIsSaid checks the first of the three:
// a link the vault cannot resolve.
func TestVerify_aLinkThatPointsNowhereIsSaid(t *testing.T) {
	got := Verify("see [[nowhere]]\n", vaultKnown{})
	if len(got) != 1 || got[0].Kind != IssueWikiLink {
		t.Fatalf("issues = %v, want one about a link", got)
	}
	// The line is said as a person counts it, from one.
	if got[0].Line != 1 {
		t.Fatalf("line = %d, want 1", got[0].Line)
	}
	if !strings.Contains(got[0].Feedback(), "第 1 行") {
		t.Fatalf("feedback = %q, want the line in it", got[0].Feedback())
	}
}

// TestVerify_aLinkOfTwoMeaningsIsSaid checks that a link which may mean
// more than one note is said, and not quietly taken for one of them.
func TestVerify_aLinkOfTwoMeaningsIsSaid(t *testing.T) {
	got := Verify("see [[a]]\n", vaultKnown{
		many: map[string][]string{
			"a": {"/vault/x/a.md", "/vault/y/a.md"},
		},
	})
	if len(got) != 1 || got[0].Kind != IssueWikiLink ||
		!strings.Contains(got[0].Detail, "歧义") {
		t.Fatalf("issues = %v, want it said to be ambiguous", got)
	}
}

// TestVerify_aLinkThatResolvesIsNotSaid checks that a good note is left
// alone: a verifier that finds fault with everything is of no use.
func TestVerify_aLinkThatResolvesIsNotSaid(t *testing.T) {
	got := Verify("see [[a]]\n", vaultKnown{
		notes: map[string]string{"a": "/vault/a.md"},
	})
	if len(got) != 0 {
		t.Fatalf("issues = %v, want none", got)
	}
}

// TestVerify_withoutAVaultNoLinkIsFaulted checks that a link is not said
// to point nowhere when there is nowhere to look: no vault, no answer.
func TestVerify_withoutAVaultNoLinkIsFaulted(t *testing.T) {
	if got := Verify("see [[a]]\n", nil); len(got) != 0 {
		t.Fatalf("issues = %v, want none", got)
	}
}

// TestVerify_aFrontMatterThatNeverClosesIsSaid checks the second of the
// three: a `---` that is never closed is what a model writes when it has
// lost its place.
func TestVerify_aFrontMatterThatNeverClosesIsSaid(t *testing.T) {
	got := Verify("---\ntitle: a\n\nbody\n", vaultKnown{})
	if len(got) != 1 || got[0].Kind != IssueFrontmatter ||
		!strings.Contains(got[0].Detail, "没有闭合") {
		t.Fatalf("issues = %v", got)
	}
}

// TestVerify_aNoteWithNoFrontMatterIsWellFormed checks that a note with
// none is not one that is wrong: most notes have none.
func TestVerify_aNoteWithNoFrontMatterIsWellFormed(t *testing.T) {
	if got := Verify("# A\n\nbody\n", vaultKnown{}); len(got) != 0 {
		t.Fatalf("issues = %v, want none", got)
	}
	// And a rule drawn in the text is not a front matter either.
	if got := Verify("# A\n\n---\n\nbody\n", vaultKnown{}); len(got) != 0 {
		t.Fatalf("issues = %v, want none", got)
	}
}

// TestVerify_aFrontMatterThatCannotBeReadIsSaid checks a line that is
// neither a key, nor an entry of a list, nor blank.
func TestVerify_aFrontMatterThatCannotBeReadIsSaid(t *testing.T) {
	got := Verify("---\ntitle: a\nthis is not a key\n---\n", vaultKnown{})
	if len(got) != 1 || got[0].Kind != IssueFrontmatter ||
		!strings.Contains(got[0].Detail, "读不懂") {
		t.Fatalf("issues = %v", got)
	}
}

// TestVerify_aListUnderATagsKeyIsWellFormed checks the shape people
// write most often.
func TestVerify_aListUnderATagsKeyIsWellFormed(t *testing.T) {
	got := Verify("---\ntags:\n  - a\n  - b\n---\n", vaultKnown{})
	if len(got) != 0 {
		t.Fatalf("issues = %v, want none", got)
	}
	// A `tags:` with nothing after it is not wrong either.
	if got := Verify("---\ntags:\n---\n", vaultKnown{}); len(got) != 0 {
		t.Fatalf("issues = %v, want none", got)
	}
}

// TestVerify_aTagsListOfNamesIsWellFormed checks the third of the three.
func TestVerify_aTagsListOfNamesIsWellFormed(t *testing.T) {
	for _, text := range []string{
		"---\ntags: [a, b]\n---\n",
		`---\ntags: ["a", "b"]\n---\n`,
		"---\ntags: a\n---\n",
	} {
		if got := Verify(text, vaultKnown{}); len(got) != 0 {
			t.Errorf("%q: issues = %v, want none", text, got)
		}
	}
}

// TestVerify_aTagsThatIsNotNamesIsSaid checks that a number or a truth
// value where a tag should be is said, rather than written into the
// index as a tag.
func TestVerify_aTagsThatIsNotNamesIsSaid(t *testing.T) {
	for _, text := range []string{
		"---\ntags: [1, 2]\n---\n",
		"---\ntags: [a, true]\n---\n",
		"---\ntags: 42\n---\n",
	} {
		got := Verify(text, vaultKnown{})
		if len(got) != 1 || got[0].Kind != IssueTag {
			t.Errorf("%q: issues = %v, want one about tags", text, got)
		}
	}
}

// TestVerify_aNoteWithSeveralThingsWrongSaysThemAll checks that one
// thing being wrong does not hide the others.
func TestVerify_aNoteWithSeveralThingsWrongSaysThemAll(t *testing.T) {
	got := Verify("---\ntags: [1]\n---\n\nsee [[nowhere]]\n", vaultKnown{})
	if len(got) != 2 {
		t.Fatalf("issues = %v, want both said", got)
	}
	kinds := map[IssueKind]bool{}
	for _, issue := range got {
		kinds[issue.Kind] = true
	}
	if !kinds[IssueTag] || !kinds[IssueWikiLink] {
		t.Fatalf("issues = %v, want a tag and a link", got)
	}
}

// TestFeedback_aThingThatIsNoLinesFault checks the line a whole-note
// issue is said on: zero, which is not a line anyone counts.
func TestFeedback_aThingThatIsNoLinesFault(t *testing.T) {
	issue := Issue{Kind: IssueFrontmatter, Line: 0, Detail: "broken"}
	if strings.Contains(issue.Feedback(), "第") {
		t.Fatalf("feedback = %q, want no line in it", issue.Feedback())
	}
	if !strings.Contains(issue.Feedback(), "frontmatter") {
		t.Fatalf("feedback = %q", issue.Feedback())
	}
}

// TestIssueKind_theKindNamedToTheModel checks that what the model is
// told is a name it wrote in.
func TestIssueKind_theKindNamedToTheModel(t *testing.T) {
	for _, kind := range []IssueKind{
		IssueWikiLink, IssueFrontmatter, IssueTag,
	} {
		if kind.Name() == "" {
			t.Errorf("%v is not named", kind)
		}
	}
}
