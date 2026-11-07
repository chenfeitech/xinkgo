package state

import "testing"

func TestRecentModels_roundTrip(t *testing.T) {
	dir := t.TempDir()
	if got := LoadRecentModels(dir); len(got) != 0 {
		t.Fatalf("a missing file gave %v", got)
	}
	in := []string{"a", "b", "a", "c"}
	if err := SaveRecentModels(dir, in); err != nil {
		t.Fatal(err)
	}
	got := LoadRecentModels(dir)
	if len(got) != 3 || got[0] != "a" || got[1] != "b" ||
		got[2] != "c" {
		t.Fatalf("recent = %v, want a b c without the duplicate", got)
	}
}

func TestRecentModels_bounded(t *testing.T) {
	dir := t.TempDir()
	in := []string{}
	for i := 0; i < 20; i++ {
		in = append(in, string(rune('a'+i)))
	}
	if err := SaveRecentModels(dir, in); err != nil {
		t.Fatal(err)
	}
	if got := LoadRecentModels(dir); len(got) != maxRecentModels {
		t.Fatalf("recent = %d, want %d", len(got), maxRecentModels)
	}
}
