package app

import (
	"reflect"
	"testing"
)

// A false in plugins.json must not start the plugin. Disabling normally
// removes the key, but a hand-edited or synced file can leave it behind.
func TestEnabledIDs_onlyOnesLeftOn(t *testing.T) {
	got := enabledIDs(map[string]bool{
		"ollama-chat": false,
		"z-plugin":    true,
		"a-plugin":    true,
	})
	want := []string{"a-plugin", "z-plugin"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("enabledIDs = %v, want %v", got, want)
	}
}

func TestEnabledIDs_emptyWhenNoneOn(t *testing.T) {
	if got := enabledIDs(map[string]bool{
		"one": false, "two": false,
	}); len(got) != 0 {
		t.Fatalf("enabledIDs = %v, want none", got)
	}
}
