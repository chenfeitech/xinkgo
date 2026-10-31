package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// said is one thing said in a conversation.
func said(role, text string) AiBlock {
	return AiBlock{Role: role, Text: text}
}

// TestLoadAiSessions_missingFileSaysNothing checks a first start: there
// is no conversation to read, and that is not a failure.
func TestLoadAiSessions_missingFileSaysNothing(t *testing.T) {
	all, err := LoadAiSessions(t.TempDir())
	if err != nil {
		t.Fatalf("LoadAiSessions: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("got %v, want nothing", all)
	}
}

// TestLoadAiSessions_brokenFileSaysNothing checks that a file the app
// cannot read does not keep it from starting: a conversation is a thing
// that can be lost.
func TestLoadAiSessions_brokenFileSaysNothing(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, AiSessionsFile)
	for _, broken := range []string{"{", "[]", "null", ""} {
		if err := os.WriteFile(name, []byte(broken), 0o644); err != nil {
			t.Fatal(err)
		}
		all, err := LoadAiSessions(dir)
		if err == nil && broken != "null" && broken != "" {
			t.Errorf("%q: want it said that the file cannot be read",
				broken)
		}
		if all == nil {
			t.Errorf("%q: got no table at all, want an empty one",
				broken)
		}
	}
}

// TestSaveAiSessions_whatIsWrittenIsReadBack checks that a conversation
// comes back as it was said.
func TestSaveAiSessions_whatIsWrittenIsReadBack(t *testing.T) {
	dir := t.TempDir()
	all := AiSessions{
		"/vault/a.md": []AiBlock{
			said("user", "what is this"),
			said("assistant", "a note"),
		},
	}
	if err := SaveAiSessions(dir, all); err != nil {
		t.Fatalf("SaveAiSessions: %v", err)
	}
	back, err := LoadAiSessions(dir)
	if err != nil {
		t.Fatalf("LoadAiSessions: %v", err)
	}
	blocks := back["/vault/a.md"]
	if len(blocks) != 2 || blocks[0].Text != "what is this" ||
		blocks[1].Text != "a note" {
		t.Fatalf("blocks = %v, want what was said", blocks)
	}
	if blocks[1].Role != "assistant" {
		t.Fatalf("role = %q", blocks[1].Role)
	}
}

// TestSaveAiSessions_anEmptyOneIsForgotten checks that a conversation
// with nothing in it is taken out of the book, rather than kept as an
// empty list.
func TestSaveAiSessions_anEmptyOneIsForgotten(t *testing.T) {
	dir := t.TempDir()
	all := AiSessions{"/vault/a.md": []AiBlock{said("user", "hi")}}
	ApplyAiSession(all, "/vault/a.md", nil)
	if err := SaveAiSessions(dir, all); err != nil {
		t.Fatalf("SaveAiSessions: %v", err)
	}
	back, err := LoadAiSessions(dir)
	if err != nil {
		t.Fatalf("LoadAiSessions: %v", err)
	}
	if _, there := back["/vault/a.md"]; there {
		t.Fatal("a forgotten conversation came back")
	}
}

// TestApplyAiSession_oneNoteDoesNotTouchAnother checks that writing one
// note's conversation leaves the others alone.
func TestApplyAiSession_oneNoteDoesNotTouchAnother(t *testing.T) {
	all := AiSessions{
		"/vault/a.md": []AiBlock{said("user", "one")},
		"/vault/b.md": []AiBlock{said("user", "two")},
	}
	ApplyAiSession(all, "/vault/a.md", []AiBlock{said("user", "changed")})
	if all["/vault/b.md"][0].Text != "two" {
		t.Fatalf("b = %v, want it left alone", all["/vault/b.md"])
	}
	if all["/vault/a.md"][0].Text != "changed" {
		t.Fatalf("a = %v", all["/vault/a.md"])
	}
	ApplyAiSession(all, "/vault/b.md", nil)
	if _, there := all["/vault/b.md"]; there {
		t.Fatal("b was not forgotten")
	}
}

// TestTrimAiSessions_aLongConversationKeepsTheNewest checks that what is
// kept of a long conversation is the end of it, and not the beginning.
func TestTrimAiSessions_aLongConversationKeepsTheNewest(t *testing.T) {
	blocks := make([]AiBlock, 0, MaxAiBlocks+10)
	for i := range MaxAiBlocks + 10 {
		blocks = append(blocks, said("user", strconv.Itoa(i)))
	}
	all := AiSessions{"/vault/a.md": blocks}
	TrimAiSessions(all)
	kept := all["/vault/a.md"]
	if len(kept) != MaxAiBlocks {
		t.Fatalf("kept %d, want %d", len(kept), MaxAiBlocks)
	}
	if kept[0].Text != "10" {
		t.Fatalf("first kept = %q, want the 11th thing said",
			kept[0].Text)
	}
	if kept[len(kept)-1].Text != strconv.Itoa(MaxAiBlocks+9) {
		t.Fatalf("last kept = %q, want the last thing said",
			kept[len(kept)-1].Text)
	}
}

// TestTrimAiSessions_tooManyNotesChecksTheBook checks that the book keeps
// only so many conversations, and that it is the same ones every time: a
// map has no order to it, so the choice is made by the note's path.
func TestTrimAiSessions_tooManyNotesChecksTheBook(t *testing.T) {
	all := AiSessions{}
	for i := range MaxAiSessions + 12 {
		all["/vault/"+strconv.Itoa(i)+".md"] = []AiBlock{
			said("user", "hi"),
		}
	}
	TrimAiSessions(all)
	if len(all) != MaxAiSessions {
		t.Fatalf("kept %d conversations, want %d", len(all),
			MaxAiSessions)
	}
	// The same choice twice over: another run must not drop a different
	// thirty.
	again := AiSessions{}
	for i := range MaxAiSessions + 12 {
		again["/vault/"+strconv.Itoa(i)+".md"] = []AiBlock{
			said("user", "hi"),
		}
	}
	TrimAiSessions(again)
	for path := range all {
		if _, there := again[path]; !there {
			t.Fatalf("%s: kept one time and not the next", path)
		}
	}
}

// TestTrimAiSessions_whatIsKeptIsEnough checks that a book within its
// limits is left exactly as it is.
func TestTrimAiSessions_whatIsKeptIsEnough(t *testing.T) {
	all := AiSessions{
		"/vault/a.md": []AiBlock{said("user", "one")},
		"/vault/b.md": []AiBlock{said("user", "two")},
	}
	TrimAiSessions(all)
	if len(all) != 2 {
		t.Fatalf("kept %d conversations, want 2", len(all))
	}
}

// TestSaveAiSessions_theOldFileIsReplacedOnlyOnceTheNewOneIsThere checks
// that a conversation is not lost to a half-written file.
func TestSaveAiSessions_theOldFileIsReplacedOnlyOnceTheNewOneIsThere(
	t *testing.T,
) {
	dir := t.TempDir()
	if err := SaveAiSessions(dir, AiSessions{
		"/vault/a.md": []AiBlock{said("user", "first")},
	}); err != nil {
		t.Fatal(err)
	}
	if err := SaveAiSessions(dir, AiSessions{
		"/vault/a.md": []AiBlock{said("user", "second")},
	}); err != nil {
		t.Fatal(err)
	}
	back, err := LoadAiSessions(dir)
	if err != nil {
		t.Fatalf("LoadAiSessions: %v", err)
	}
	if got := back["/vault/a.md"][0].Text; got != "second" {
		t.Fatalf("text = %q, want second", got)
	}
	if _, err := os.Stat(filepath.Join(dir,
		AiSessionsFile+".tmp")); err == nil {
		t.Fatal("the temporary file was left behind")
	}
}

// TestAiBlock_whatIsKeptIsWhatWasSaid checks the shape that is written:
// the thinking and the cut are kept, and nothing else is.
func TestAiBlock_whatIsKeptIsWhatWasSaid(t *testing.T) {
	data, err := json.Marshal(AiBlock{
		Role: "assistant", Text: "hi", Thinking: "hmm", Cut: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"role":"assistant","text":"hi","thinking":"hmm","cut":true}`
	if string(data) != want {
		t.Fatalf("wrote %s, want %s", data, want)
	}
	var back AiBlock
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if back != (AiBlock{
		Role: "assistant", Text: "hi", Thinking: "hmm", Cut: true,
	}) {
		t.Fatalf("read back %+v", back)
	}
}
