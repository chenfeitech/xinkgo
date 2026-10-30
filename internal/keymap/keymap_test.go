package keymap

import (
	"strings"
	"testing"
)

// keystrokeOf is the key the action ended up with, or "" when it has none.
func keystrokeOf(resolved []Resolved, action string) string {
	for _, r := range resolved {
		if r.Action == action {
			return r.Keystroke
		}
	}
	return ""
}

// TestBindings_areConsistent checks the table itself: one id and one
// action each, and a default that can be bound.
func TestBindings_areConsistent(t *testing.T) {
	ids := map[string]bool{}
	actions := map[string]bool{}
	for _, b := range Bindings {
		if ids[b.ID] {
			t.Errorf("the id %s is in the table twice", b.ID)
		}
		ids[b.ID] = true
		if actions[b.Action] {
			t.Errorf("the action %s is in the table twice", b.Action)
		}
		actions[b.Action] = true
		if !strings.HasPrefix(b.ID, "editor.keymap.") {
			t.Errorf("%s: an id starts with editor.keymap.", b.ID)
		}
		if b.Label == "" || b.Desc == "" {
			t.Errorf("%s has no title or no description", b.ID)
		}
		if b.Default != "" && !IsValid(b.Default) {
			t.Errorf("%s: the default %q cannot be bound", b.ID,
				b.Default)
		}
	}
}

// TestResolve_withNoOverridesGivesTheDefaults checks that a first start
// binds what the table says.
func TestResolve_withNoOverridesGivesTheDefaults(t *testing.T) {
	resolved := Resolve(map[string]string{})
	want := 0
	for _, b := range Bindings {
		if b.Default != "" {
			want++
		}
	}
	if len(resolved) != want {
		t.Fatalf("%d bindings, want the %d that have a default",
			len(resolved), want)
	}
	for _, b := range Bindings {
		if b.Default == "" {
			continue
		}
		if got := keystrokeOf(resolved, b.Action); got != b.Default {
			t.Errorf("%s got %q, want %q", b.Action, got, b.Default)
		}
	}
}

// TestResolve_overrideReplacesTheDefault checks a key the user moved.
func TestResolve_overrideReplacesTheDefault(t *testing.T) {
	resolved := Resolve(map[string]string{"Save": "cmd-shift-s"})
	if got := keystrokeOf(resolved, "Save"); got != "cmd-shift-s" {
		t.Errorf("Save got %q, want cmd-shift-s", got)
	}
	for _, r := range resolved {
		if r.Keystroke == "cmd-s" {
			t.Error("cmd-s is still bound: the old key is let go")
		}
	}
}

// TestResolve_emptyOverrideKeepsTheDefault checks that an override of
// nothing is the same as no override at all.
func TestResolve_emptyOverrideKeepsTheDefault(t *testing.T) {
	resolved := Resolve(map[string]string{"Save": "   "})
	if got := keystrokeOf(resolved, "Save"); got != "cmd-s" {
		t.Errorf("Save got %q, want cmd-s", got)
	}
}

// TestResolve_keyThatCannotBeBoundKeepsTheDefault checks "cmd-", which has
// no key at all.
func TestResolve_keyThatCannotBeBoundKeepsTheDefault(t *testing.T) {
	resolved := Resolve(map[string]string{
		"Save":       "cmd-",
		"OpenSearch": "shift-",
	})
	if got := keystrokeOf(resolved, "Save"); got != "cmd-s" {
		t.Errorf("Save got %q, want cmd-s", got)
	}
	if got := keystrokeOf(resolved, "OpenSearch"); got != "cmd-shift-f" {
		t.Errorf("OpenSearch got %q, want cmd-shift-f", got)
	}
}

// TestResolve_unknownActionIsNotBound checks that an action the table does
// not have cannot be added.
func TestResolve_unknownActionIsNotBound(t *testing.T) {
	resolved := Resolve(map[string]string{"NotAnAction": "cmd-j"})
	if got := keystrokeOf(resolved, "NotAnAction"); got != "" {
		t.Errorf("NotAnAction got %q, want nothing", got)
	}
}

// TestNormalize_trimsAndLowers checks a key written by hand.
func TestNormalize_trimsAndLowers(t *testing.T) {
	cases := map[string]string{
		" cmd-S ":         "cmd-s",
		"CMD-SHIFT-X":     "cmd-shift-x",
		"cmd-k   cmd-s":   "cmd-k cmd-s",
		"cmd-shift-space": "cmd-shift-space",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestResolve_twoActionsOneKeyKeepsTheEarlier checks the conflict: NewNote
// is earlier in the table, so Save asking for its key gets its own back.
func TestResolve_twoActionsOneKeyKeepsTheEarlier(t *testing.T) {
	resolved := Resolve(map[string]string{"Save": "cmd-n"})
	if got := keystrokeOf(resolved, "Save"); got != "cmd-s" {
		t.Errorf("Save got %q, want cmd-s", got)
	}
	if got := keystrokeOf(resolved, "NewNote"); got != "cmd-n" {
		t.Errorf("NewNote got %q, want cmd-n", got)
	}
}

// TestSanitize_dropsWhatCannotBeUsed checks the clean-up of a file someone
// edited by hand.
func TestSanitize_dropsWhatCannotBeUsed(t *testing.T) {
	overrides := map[string]string{
		"Save":        " CMD-SHIFT-S ",
		"OpenSearch":  "shift-",
		"NotAnAction": "cmd-j",
		"ToggleTheme": "",
	}
	Sanitize(overrides)
	if got := overrides["Save"]; got != "cmd-shift-s" {
		t.Errorf("Save is %q, want cmd-shift-s", got)
	}
	if _, ok := overrides["OpenSearch"]; ok {
		t.Error("a key that cannot be bound was kept")
	}
	if _, ok := overrides["NotAnAction"]; ok {
		t.Error("an action the table does not have was kept")
	}
	if _, ok := overrides["ToggleTheme"]; ok {
		t.Error("an empty key was kept")
	}
}

// TestAccelerator_isWhatAMenuTakes checks the key as a menu item wants it.
func TestAccelerator_isWhatAMenuTakes(t *testing.T) {
	cases := map[string]string{
		"cmd-o":       "CmdOrCtrl+O",
		"cmd-shift-f": "CmdOrCtrl+Shift+F",
		"ctrl-w":      "Ctrl+W",
		"f2":          "F2",
		"cmd-,":       "CmdOrCtrl+,",
		"cmd-k cmd-s": "",
		"cmd-":        "",
	}
	for in, want := range cases {
		if got := Accelerator(in); got != want {
			t.Errorf("Accelerator(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestIsValid_rejectsBrokenKeys checks what a key has to look like.
func TestIsValid_rejectsBrokenKeys(t *testing.T) {
	valid := []string{"cmd-s", "cmd-,", "f2", "cmd-shift-f",
		"cmd-k cmd-s", "ctrl-w", "cmd-shift-space"}
	for _, key := range valid {
		if !IsValid(key) {
			t.Errorf("IsValid(%q) = false, want true", key)
		}
	}
	broken := []string{"", "   ", "cmd-", "shift-", "ctrl-alt", "cmd-f13"}
	for _, key := range broken {
		if IsValid(key) {
			t.Errorf("IsValid(%q) = true, want false", key)
		}
	}
}
