package config

import "testing"

// TestQuickWord_oneWithNoWordOfItsOwnIsRunByItsName checks that a word is
// run by what it is called: a trigger the user wrote none of is not a
// word that cannot be run at all.
func TestQuickWord_oneWithNoWordOfItsOwnIsRunByItsName(t *testing.T) {
	cfg := DefaultAi()
	cfg.Actions = []AiAction{
		{ID: "polish", Name: "Make it read well", Enable: true},
	}
	cfg.Sanitize()
	if got := cfg.Actions[0].Word(); got != "makeitreadwell" {
		t.Fatalf("Word() = %q, want what it is called, made into one word",
			got)
	}
}

// TestQuickWord_twoAreNotRunByTheSameWord checks that a word two of them
// were given falls back to the name of the second: two words run by one
// word is one of them that cannot be told from the other.
func TestQuickWord_twoAreNotRunByTheSameWord(t *testing.T) {
	cfg := DefaultAi()
	cfg.Actions = []AiAction{
		{ID: "one", Name: "Polish", Trigger: "shine", Enable: true},
		{ID: "two", Name: "Shine", Trigger: "shine", Enable: true},
	}
	cfg.Sanitize()
	if got := cfg.Actions[0].Word(); got != "shine" {
		t.Fatalf("the first is run by %q, want shine", got)
	}
	// The word was taken, and so was the name it is called by: what is
	// left is its own name, which no other answers to.
	if got := cfg.Actions[1].Word(); got != "two" {
		t.Fatalf("the second is run by %q, want the name it is known by",
			got)
	}
}

// TestQuickWord_theTriggerIsMadeIntoOneWord checks that the slash that
// runs a word and the spaces in it are no part of the word: it is typed
// after a slash, and a space is where what the user adds to it begins.
func TestQuickWord_theTriggerIsMadeIntoOneWord(t *testing.T) {
	cfg := DefaultAi()
	cfg.Actions = []AiAction{
		{ID: "one", Name: "Polish", Trigger: " /Make It Shine ",
			Enable: true},
	}
	cfg.Sanitize()
	if got := cfg.Actions[0].Trigger; got != "makeitshine" {
		t.Fatalf("trigger = %q, want makeitshine", got)
	}
}

// TestQuickWord_theBuiltInOnesAreRunByTheirOwnNames checks the words the
// app ships with: each is run by the word it is called by, in small
// letters, and none of them by a word another answers to.
func TestQuickWord_theBuiltInOnesAreRunByTheirOwnNames(t *testing.T) {
	cfg := DefaultAi()
	cfg.Sanitize()
	seen := map[string]bool{}
	for _, one := range cfg.Actions {
		word := one.Word()
		if seen[word] {
			t.Errorf("%q and another are both run by %q", one.ID, word)
		}
		seen[word] = true
		if word == "" {
			t.Errorf("%q is run by nothing at all", one.ID)
		}
	}
	if !seen["polish"] || !seen["condense"] {
		t.Fatalf("the built-in words are %v", seen)
	}
}

// TestQuickWord_aLongWordIsCutShort checks that a word longer than the
// longest there is room for is cut, and cut by letters and not by bytes,
// so that a letter of a tongue written in more than one byte is left
// whole.
func TestQuickWord_aLongWordIsCutShort(t *testing.T) {
	cfg := DefaultAi()
	cfg.Actions = []AiAction{
		{ID: "long", Name: "词", Trigger: "词词词词词词词词词词词词词词词词词词词词词词词词词词词词词词",
			Enable: true},
	}
	cfg.Sanitize()
	got := cfg.Actions[0].Trigger
	if len([]rune(got)) != maxTriggerLen {
		t.Fatalf("trigger = %q, want %d letters", got, maxTriggerLen)
	}
}

// TestEnabledActions_oneTurnedOffIsNotOffered checks that a word turned
// off is not one the user is offered, and that one with no name at all
// is not either.
func TestEnabledActions_oneTurnedOffIsNotOffered(t *testing.T) {
	cfg := DefaultAi()
	cfg.Sanitize()
	cfg.Actions[0].Enable = false
	cfg.Actions[1].ID = " "
	cfg.Sanitize()
	got := EnabledActions(cfg.Actions)
	if len(got) != len(cfg.Actions)-1 {
		t.Fatalf("%d offered, want %d", len(got), len(cfg.Actions)-1)
	}
}
