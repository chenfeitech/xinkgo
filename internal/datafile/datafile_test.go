package datafile

import "testing"

// TestKindOf_theNameItWasGiven checks how a file's kind is known.
func TestKindOf_theNameItWasGiven(t *testing.T) {
	cases := map[string]Kind{
		"a.json":       JSON,
		"a.JSON":       JSON,
		"a.yaml":       YAML,
		"a.yml":        YAML,
		"a.toml":       TOML,
		"a.md":         None,
		"config.json5": None,
	}
	for path, want := range cases {
		if got := KindOf(path); got != want {
			t.Errorf("KindOf(%q) = %v, want %v", path, got, want)
		}
	}
}

// TestCheck_aWellFormedFileHasNothingSaidOfIt checks a file that reads.
func TestCheck_aWellFormedFileHasNothingSaidOfIt(t *testing.T) {
	if _, bad := Check(JSON, `{"a": 1}`); bad {
		t.Error("a well formed file was said to have something wrong")
	}
}

// TestCheck_whereItIsWrong checks that what is wrong is said, and where.
func TestCheck_whereItIsWrong(t *testing.T) {
	problem, bad := Check(JSON, "{\n  \"a\": 1,\n  \"b\" 2\n}\n")
	if !bad {
		t.Fatal("the file has something wrong and nothing was said of it")
	}
	if problem.Line != 2 {
		t.Errorf("the problem is at line %d, want 2", problem.Line)
	}
	if problem.Message == "" {
		t.Error("what is wrong was not said")
	}
}

// TestCheck_theKindsTheAppDoesNotRead checks that nothing is said of a
// file the app does not read: better nothing than a wrong guess.
func TestCheck_theKindsTheAppDoesNotRead(t *testing.T) {
	for _, kind := range []Kind{YAML, TOML, None} {
		if _, bad := Check(kind, "::: not really :::"); bad {
			t.Errorf("kind %v: something was said of a file the app "+
				"does not read", kind)
		}
	}
}

// TestFormat_writtenOutEvenly checks a JSON file written out again.
func TestFormat_writtenOutEvenly(t *testing.T) {
	text := `{"a":1,"b":[2,3]}`
	got, ok := Format(JSON, text, 2)
	if !ok {
		t.Fatal("the file was not written out")
	}
	want := "{\n  \"a\": 1,\n  \"b\": [\n    2,\n    3\n  ]\n}"
	if got != want {
		t.Errorf("written out as %q, want %q", got, want)
	}
}

// TestFormat_aFileThatDoesNotReadIsLeftAlone checks that a broken file is
// not rewritten: what it holds is still what the user wrote.
func TestFormat_aFileThatDoesNotReadIsLeftAlone(t *testing.T) {
	text := "{\"a\": }"
	got, ok := Format(JSON, text, 2)
	if ok {
		t.Error("a file that does not read was written out")
	}
	if got != text {
		t.Errorf("the file was changed to %q", got)
	}
}

// TestFormat_theKindsTheAppDoesNotRead checks that they are left as they
// are.
func TestFormat_theKindsTheAppDoesNotRead(t *testing.T) {
	text := "a: 1\n"
	if got, ok := Format(YAML, text, 2); ok || got != text {
		t.Errorf("a YAML file was written out as %q", got)
	}
}

// TestTooLarge_aFileOfMegabytes checks when a file is left as it stands.
func TestTooLarge_aFileOfMegabytes(t *testing.T) {
	if !TooLarge(5_000_000, 1_000_000) {
		t.Error("a file of five megabytes was said to be small enough")
	}
	if TooLarge(500, 1_000_000) {
		t.Error("a file of 500 bytes was said to be too large")
	}
	// Nothing chosen is nothing skipped.
	if TooLarge(5_000_000, 0) {
		t.Error("a file was skipped though no size was chosen")
	}
}

// TestKind_theNameAsTheWindowSaysIt checks what each kind is called.
func TestKind_theNameAsTheWindowSaysIt(t *testing.T) {
	if JSON.Name() != "JSON" || YAML.Name() != "YAML" ||
		TOML.Name() != "TOML" || None.Name() != "" {
		t.Error("a kind is not called what it is")
	}
}
