package ai

import "testing"

func TestReplacement_onlyTheTaggedText(t *testing.T) {
	got, ok := Replacement(
		"Here is why.\n<replacement>the new words</replacement>\nDone.")
	if !ok || got != "the new words" {
		t.Fatalf("replacement = %q, %v", got, ok)
	}
}

func TestReplacement_untaggedOrUnclosed(t *testing.T) {
	for _, in := range []string{
		"just an answer", "<replacement>never closed",
	} {
		if _, ok := Replacement(in); ok {
			t.Errorf("%q was taken for a replacement", in)
		}
	}
}
