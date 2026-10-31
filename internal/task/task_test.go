package task

import (
	"errors"
	"strings"
	"testing"
)

func TestRetagText_theWholeTokenIsChanged(t *testing.T) {
	got := RetagText("some words #AI here\n", "AI", "人工智能")
	if got.Kind != Changed {
		t.Fatalf("kind = %v, want changed", got.Kind)
	}
	if got.Text != "some words #人工智能 here\n" {
		t.Fatalf("text = %q", got.Text)
	}
	if len(got.Preview) != 1 || got.Preview[0].Before !=
		"some words #AI here" ||
		got.Preview[0].After != "some words #人工智能 here" {
		t.Fatalf("preview = %v", got.Preview)
	}
}

func TestRetagText_aTagThatIsNotAWholeTokenIsNotTouched(t *testing.T) {
	// `#AI助手` is not `#AI`: guessing would change a tag the user
	// never named.
	got := RetagText("words #AI助手 here\n", "AI", "人工智能")
	if got.Kind != Unchanged {
		t.Fatalf("kind = %v, want unchanged", got.Kind)
	}
}

func TestRetagText_aNoteThatAlreadyHasTheNewTagIsSkipped(t *testing.T) {
	// Changing it would leave two of the same tag.
	got := RetagText("#AI and #人工智能\n", "AI", "人工智能")
	if got.Kind != Skipped {
		t.Fatalf("kind = %v, want skipped", got.Kind)
	}
	if got.Reason == "" {
		t.Fatal("a note skipped without a reason is a note lied about")
	}
}

func TestRetagText_aTagInACodeFenceIsNotATag(t *testing.T) {
	// `#define` and the like would be hurt.
	got := RetagText("```\n#define AI 1\n```\n#AI\n", "AI", "人工智能")
	if got.Kind != Changed {
		t.Fatalf("kind = %v", got.Kind)
	}
	if strings.Contains(got.Text, "#define 人工智能") {
		t.Fatalf("text = %q, want the fence left alone", got.Text)
	}
	if !strings.Contains(got.Text, "#人工智能") {
		t.Fatalf("text = %q, want the real tag changed", got.Text)
	}
}

func TestRetagText_aTagInTheFrontMatterListIsChanged(t *testing.T) {
	text := "---\ntags: [AI, other]\n---\n\nbody\n"
	got := RetagText(text, "AI", "人工智能")
	if got.Kind != Changed {
		t.Fatalf("kind = %v", got.Kind)
	}
	if !strings.Contains(got.Text, "tags: [人工智能, other]") {
		t.Fatalf("text = %q", got.Text)
	}
}

func TestRetagText_aTagUnderATagsKeyIsChanged(t *testing.T) {
	text := "---\ntags:\n  - AI\n  - other\n---\n\nbody\n"
	got := RetagText(text, "AI", "人工智能")
	if !strings.Contains(got.Text, "- 人工智能") {
		t.Fatalf("text = %q", got.Text)
	}
	// The indent it stood at is kept.
	if !strings.Contains(got.Text, "  - 人工智能") {
		t.Fatalf("text = %q, want the indent kept", got.Text)
	}
}

func TestRetagText_theQuotesOfAnEntryAreKept(t *testing.T) {
	got := RetagText("---\ntags: [\"AI\"]\n---\n", "AI", "人工智能")
	if !strings.Contains(got.Text, `"人工智能"`) {
		t.Fatalf("text = %q, want the quotes kept", got.Text)
	}
}

func TestAddTagText_itGoesIntoTheFrontMatterList(t *testing.T) {
	got := AddTagText("---\ntags: [other]\n---\n\nbody\n", "AI")
	if got.Kind != Changed {
		t.Fatalf("kind = %v", got.Kind)
	}
	if !strings.Contains(got.Text, "tags: [other, AI]") {
		t.Fatalf("text = %q", got.Text)
	}
}

func TestAddTagText_itGoesIntoAnEmptyList(t *testing.T) {
	// `tags: []` is a list that is legally empty, and it is filled in.
	got := AddTagText("---\ntags: []\n---\n", "AI")
	if !strings.Contains(got.Text, "tags: [AI]") {
		t.Fatalf("text = %q", got.Text)
	}
}

func TestAddTagText_aFrontMatterWithNoTagsGetsOne(t *testing.T) {
	got := AddTagText("---\ntitle: a\n---\n\nbody\n", "AI")
	if !strings.Contains(got.Text, "tags: [AI]") {
		t.Fatalf("text = %q, want a tags: put in", got.Text)
	}
	// What was there is not thrown away.
	if !strings.Contains(got.Text, "title: a") {
		t.Fatalf("text = %q", got.Text)
	}
}

func TestAddTagText_aNoteWithNoFrontMatterGetsALineAtTheEnd(t *testing.T) {
	got := AddTagText("# A\n\nbody\n", "AI")
	if got.Kind != Changed {
		t.Fatalf("kind = %v", got.Kind)
	}
	if !strings.HasSuffix(got.Text, "#AI\n") {
		t.Fatalf("text = %q, want the tag at the end", got.Text)
	}
	// With a blank line before it, not stuck to the last words.
	if !strings.Contains(got.Text, "body\n\n#AI\n") {
		t.Fatalf("text = %q, want a blank line before it", got.Text)
	}
}

func TestAddTagText_aNoteThatHasItAlreadyIsSkipped(t *testing.T) {
	got := AddTagText("# some #AI here\n", "AI")
	if got.Kind != Skipped {
		t.Fatalf("kind = %v, want skipped", got.Kind)
	}
}

func TestRemoveTagText_theTagIsTakenOut(t *testing.T) {
	got := RemoveTagText("some #AI here\n", "AI")
	if got.Kind != Changed {
		t.Fatalf("kind = %v", got.Kind)
	}
	// Only the token goes: the space on either side of it was not the
	// tag's, and is left where it stood.
	if got.Text != "some  here\n" {
		t.Fatalf("text = %q", got.Text)
	}
}

func TestRemoveTagText_aLineLeftWithNothingGoesAltogether(t *testing.T) {
	// The tag was all the line was: leaving an empty line behind is
	// litter.
	got := RemoveTagText("# A\n\n#AI\n\nmore\n", "AI")
	if strings.Contains(got.Text, "\n\n\n") {
		t.Fatalf("text = %q, want no blank line left behind", got.Text)
	}
	if strings.Contains(got.Text, "#AI") {
		t.Fatalf("text = %q", got.Text)
	}
}

func TestRemoveTagText_theLastEntryOfAListTakesTheLineWithIt(t *testing.T) {
	got := RemoveTagText("---\ntags: [AI]\n---\n\nbody\n", "AI")
	if strings.Contains(got.Text, "tags:") {
		t.Fatalf("text = %q, want the line gone", got.Text)
	}
	if !strings.Contains(got.Text, "body") {
		t.Fatalf("text = %q, want the rest kept", got.Text)
	}
}

func TestRemoveTagText_aNoteWithoutItIsUnchanged(t *testing.T) {
	if got := RemoveTagText("nothing here\n", "AI"); got.Kind != Unchanged {
		t.Fatalf("kind = %v, want unchanged", got.Kind)
	}
}

func TestApplyText_anEmptyTagIsNotAGuess(t *testing.T) {
	// A tag that was never named is not a thing to search the vault
	// for.
	for _, got := range []Outcome{
		RetagText("#AI\n", "", "x"),
		RetagText("#AI\n", "AI", ""),
		AddTagText("#a\n", ""),
		RemoveTagText("#a\n", ""),
	} {
		if got.Kind != Unchanged {
			t.Fatalf("kind = %v, want unchanged", got.Kind)
		}
	}
}

func TestHasTag_whereATagStands(t *testing.T) {
	if !HasTag("words #AI here", "AI") {
		t.Fatal("a tag in the body was not found")
	}
	if !HasTag("---\ntags: [AI]\n---", "AI") {
		t.Fatal("a tag in the list was not found")
	}
	if !HasTag("---\ntags:\n  - AI\n---", "AI") {
		t.Fatal("a tag under the key was not found")
	}
	if HasTag("words #AI助手 here", "AI") {
		t.Fatal("a tag that is not there was found")
	}
	if HasTag("", "AI") {
		t.Fatal("a tag was found in nothing")
	}
}

func TestTask_nextBatchSkipsWhatIsAlreadyDone(t *testing.T) {
	one := New("t", Retag, "AI", "人工智能")
	one.Items = []TaskItem{
		{Path: "a", State: ItemState{Kind: ItemDone}},
		{Path: "b"},
		{Path: "c", State: ItemState{Kind: ItemFailed, Reason: "x"}},
		{Path: "d"},
		{Path: "e"},
	}
	if got := one.Pending(); got != 3 {
		t.Fatalf("pending = %d, want 3", got)
	}
	// Taking it up again does not do a and c over.
	batch := one.NextBatch(2)
	if len(batch) != 2 || batch[0] != 1 || batch[1] != 3 {
		t.Fatalf("batch = %v, want b then d", batch)
	}
	// From where it got to on, and no further back.
	one.Cursor = 3
	if batch := one.NextBatch(5); len(batch) != 2 || batch[0] != 3 ||
		batch[1] != 4 {
		t.Fatalf("batch = %v, want d then e", batch)
	}
	one.Cursor = 4
	if batch := one.NextBatch(5); len(batch) != 1 || batch[0] != 4 {
		t.Fatalf("batch = %v, want only e", batch)
	}
}

func TestTask_theCountAndTheTitle(t *testing.T) {
	one := New("t", Retag, "AI", "人工智能")
	one.Items = []TaskItem{
		{State: ItemState{Kind: ItemDone}},
		{State: ItemState{Kind: ItemSkipped, Reason: "已含"}},
		{State: ItemState{Kind: ItemFailed, Reason: "写不了"}},
		{State: ItemState{Kind: ItemPending}},
	}
	done, skipped, failed := one.Stats()
	if done != 1 || skipped != 1 || failed != 1 {
		t.Fatalf("stats = %d %d %d", done, skipped, failed)
	}
	if one.Finished() {
		t.Fatal("a task with one left was said to be finished")
	}
	if !strings.Contains(one.Title(), "#AI → #人工智能") {
		t.Fatalf("title = %q", one.Title())
	}
	// A new name not yet said is not shown as an empty `#`.
	other := New("t", Retag, "AI", "")
	if strings.Contains(other.Title(), "→ #") {
		t.Fatalf("title = %q, want the new name left for later",
			other.Title())
	}
}

func TestTask_aTaskWithNothingLeftIsDone(t *testing.T) {
	one := New("t", RemoveTag, "AI", "")
	one.Items = []TaskItem{{State: ItemState{Kind: ItemDone}}}
	if !one.Finished() || one.Pending() != 0 {
		t.Fatal("a task with nothing left was not said to be finished")
	}
}

// TestPlan_aPlanOnlyReads checks that nothing is written by planning,
// and that a note it would do nothing to is not held at all.
func TestPlan_aPlanOnlyReads(t *testing.T) {
	notes := map[string]string{
		"a.md": "has #AI\n",
		"b.md": "has nothing\n",
		"c.md": "has #AI and #人工智能\n",
	}
	read := func(path string) (string, error) { return notes[path], nil }
	written := 0
	one := Plan("t", Retag, "AI", "人工智能",
		[]string{"a.md", "b.md", "c.md"}, read)
	if written != 0 {
		t.Fatal("planning wrote")
	}
	if len(one.Items) != 3 {
		t.Fatalf("items = %v", one.Items)
	}
	// a would change, and is left for the run to do.
	if one.Items[0].State.Finished() ||
		len(one.Items[0].Preview) != 1 {
		t.Fatalf("a = %+v, want a preview and nothing done", one.Items[0])
	}
	// b has no such tag, and c has the new one already.
	if one.Items[1].State.Kind != ItemSkipped ||
		one.Items[2].State.Kind != ItemSkipped {
		t.Fatalf("b = %+v c = %+v, want both skipped", one.Items[1],
			one.Items[2])
	}
	if one.Items[2].State.Reason == "" {
		t.Fatal("a skipped note with no reason is a note lied about")
	}
}

// TestPlan_aNoteThatCannotBeReadIsSaidSo checks that a note the plan
// cannot read is not quietly left out.
func TestPlan_aNoteThatCannotBeReadIsSaidSo(t *testing.T) {
	one := Plan("t", AddTag, "AI", "", []string{"gone.md"},
		func(string) (string, error) {
			return "", errors.New("no such file")
		})
	if len(one.Items) != 1 || one.Items[0].State.Kind != ItemFailed {
		t.Fatalf("items = %+v", one.Items)
	}
	if !strings.Contains(one.Items[0].State.Reason, "no such file") {
		t.Fatalf("reason = %q", one.Items[0].State.Reason)
	}
}

// TestRun_aBatchIsDoneAndWrittenBack checks that a run re-reads the note
// — the disk is the truth, and a plan may be old — and writes it back.
func TestRun_aBatchIsDoneAndWrittenBack(t *testing.T) {
	notes := map[string]string{
		"a.md": "has #AI\n",
		"b.md": "has #AI too\n",
		"c.md": "has #AI as well\n",
	}
	read := func(path string) (string, error) { return notes[path], nil }
	written := map[string]string{}
	write := func(path, text string) error {
		written[path] = text
		notes[path] = text
		return nil
	}
	one := Plan("t", Retag, "AI", "人工智能",
		[]string{"a.md", "b.md", "c.md"}, read)
	if did := Run(&one, 2, read, write); did != 2 {
		t.Fatalf("did %d, want 2", did)
	}
	if len(written) != 2 || written["c.md"] != "" {
		t.Fatalf("written = %v, want only the first two", written)
	}
	if !strings.Contains(written["a.md"], "#人工智能") {
		t.Fatalf("a = %q", written["a.md"])
	}
	if one.Status != Paused || one.Finished() {
		t.Fatalf("status = %v, want it paused with one left", one.Status)
	}
	// Run again: what is done is not done over.
	if did := Run(&one, 2, read, write); did != 1 {
		t.Fatalf("did %d, want the last one", did)
	}
	if !strings.Contains(written["c.md"], "#人工智能") {
		t.Fatalf("c = %q", written["c.md"])
	}
	if one.Status != Done || !one.Finished() {
		t.Fatalf("status = %v, want it done", one.Status)
	}
}

// TestRun_aNoteThatCannotBeWrittenIsSaidSo checks that a failure is
// kept on the note, and does not stop the rest of the batch.
func TestRun_aNoteThatCannotBeWrittenIsSaidSo(t *testing.T) {
	notes := map[string]string{"a.md": "#AI\n", "b.md": "#AI\n"}
	read := func(path string) (string, error) { return notes[path], nil }
	write := func(path, text string) error {
		if path == "a.md" {
			return errors.New("the disk is full")
		}
		notes[path] = text
		return nil
	}
	one := Plan("t", Retag, "AI", "x", []string{"a.md", "b.md"}, read)
	Run(&one, 0, read, write)
	if one.Items[0].State.Kind != ItemFailed ||
		!strings.Contains(one.Items[0].State.Reason, "the disk is full") {
		t.Fatalf("a = %+v", one.Items[0])
	}
	if one.Items[1].State.Kind != ItemDone {
		t.Fatalf("b = %+v, want it done all the same", one.Items[1])
	}
	// A failed one is finished: it is not tried again and again.
	if !one.Items[0].State.Finished() {
		t.Fatal("a failed note is not finished")
	}
}

// TestRun_aNoteTheTagIsNotInIsSkippedRatherThanWritten checks that the
// disk is read again at run time: a tag taken out since the plan was
// made is not written back in.
func TestRun_aNoteTheTagIsNotInIsSkippedRatherThanWritten(t *testing.T) {
	notes := map[string]string{"a.md": "#AI\n"}
	read := func(path string) (string, error) { return notes[path], nil }
	write := func(path, text string) error {
		notes[path] = text
		return nil
	}
	one := Plan("t", Retag, "AI", "x", []string{"a.md"}, read)
	// Someone took the tag out while the plan was being looked at.
	notes["a.md"] = "no tag here\n"
	Run(&one, 0, read, write)
	if one.Items[0].State.Kind != ItemSkipped {
		t.Fatalf("a = %+v, want it skipped", one.Items[0])
	}
	if notes["a.md"] != "no tag here\n" {
		t.Fatalf("a = %q, want it left alone", notes["a.md"])
	}
}

func TestKindAndStatus_theWordsTheyAreSaidIn(t *testing.T) {
	for _, kind := range []Kind{Retag, AddTag, RemoveTag} {
		if kind.ID() == "" || kind.Label() == "" {
			t.Errorf("%v is not named", kind)
		}
	}
	for _, status := range []Status{Ready, Running, Paused, Done} {
		if status.Label() == "" {
			t.Errorf("%v is not named", status)
		}
	}
}

func TestItemState_aSkippedOneSaysWhy(t *testing.T) {
	pending := ItemState{Kind: ItemPending}
	if pending.Finished() {
		t.Fatal("a note not yet done was said to be finished")
	}
	skipped := ItemState{Kind: ItemSkipped, Reason: "已含 #x"}
	if !skipped.Finished() {
		t.Fatal("a skipped note still has to be done")
	}
	if !strings.Contains(skipped.Label(), "已含 #x") {
		t.Fatalf("label = %q, want the reason in it", skipped.Label())
	}
}

func TestProgress_whatAPersonReads(t *testing.T) {
	one := New("t", AddTag, "AI", "")
	one.Items = []TaskItem{
		{Path: "a.md", State: ItemState{Kind: ItemDone},
			Preview: []Edit{{Before: "", After: "#AI"}}},
		{Path: "b.md", State: ItemState{Kind: ItemSkipped,
			Reason: "已含 #AI"}},
	}
	got := Progress(one)
	for _, want := range []string{"加标签", "a.md", "已改", "b.md",
		"跳过", "已含 #AI", "#AI"} {
		if !strings.Contains(got, want) {
			t.Errorf("progress = %q, want %q in it", got, want)
		}
	}
}

func TestRewrite_theEndOfTheNoteIsKept(t *testing.T) {
	// Whether a note ends with a newline is not a thing to change on
	// the way past.
	for _, text := range []string{"#AI\n", "#AI", "a\n#AI\nb\n", ""} {
		got := RetagText(text, "AI", "x")
		if got.Kind == Changed &&
			strings.HasSuffix(got.Text, "\n\n") {
			t.Errorf("%q became %q: a newline was doubled", text,
				got.Text)
		}
	}
	if got := RetagText("#AI\n", "AI", "x"); got.Text != "#x\n" {
		t.Fatalf("text = %q", got.Text)
	}
	if got := RetagText("#AI", "AI", "x"); got.Text != "#x" {
		t.Fatalf("text = %q", got.Text)
	}
}
