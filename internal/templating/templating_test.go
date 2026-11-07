package templating

import (
	"strings"
	"testing"
	"time"
)

func at() time.Time {
	return time.Date(2026, 11, 6, 15, 4, 5, 0, time.Local)
}

func TestFormat_knownTokens(t *testing.T) {
	for _, one := range []struct {
		format, want string
	}{
		{"YYYY-MM-DD", "2026-11-06"},
		{"YYYY/MM/DD", "2026/11/06"},
		{"YY-M-D", "26-11-6"},
		{"MMMM D, YYYY", "November 6, 2026"},
		{"ddd, MMM DD", "Fri, Nov 06"},
		{"HH:mm:ss", "15:04:05"},
		{"h:mm A", "3:04 PM"},
		{"h:mm a", "3:04 pm"},
		{"[note-]YYYY", "note-2026"},
		{"YYYY[年]MM[月]DD[日]", "2026年11月06日"},
	} {
		got, err := Format(at(), one.format)
		if err != nil {
			t.Errorf("Format(%q): %v", one.format, err)
			continue
		}
		if got != one.want {
			t.Errorf("Format(%q) = %q, want %q", one.format, got,
				one.want)
		}
	}
}

func TestValidate_rejectsUnknownTokens(t *testing.T) {
	for _, one := range []string{"", "YYYY-Do-MM", "gggg-MM", "[unclosed"} {
		if err := Validate(one); err == nil {
			t.Errorf("Validate(%q) accepted a bad format", one)
		}
	}
	if err := Validate("YYYY-MM-DD"); err != nil {
		t.Errorf("a good format was refused: %v", err)
	}
}

func TestParse_roundTrip(t *testing.T) {
	for _, format := range []string{
		"YYYY-MM-DD", "YYYY/MM/DD", "YYYYMMDD", "MMMM D, YYYY",
		"YYYY-MM-DD-HH-mm",
	} {
		name, err := Format(at(), format)
		if err != nil {
			t.Fatal(err)
		}
		got, err := Parse(name, format)
		if err != nil {
			t.Errorf("Parse(%q, %q): %v", name, format, err)
			continue
		}
		if got.Year() != 2026 || got.Month() != 11 || got.Day() != 6 {
			t.Errorf("Parse(%q, %q) = %v", name, format, got)
		}
	}
}

func TestParse_refusesImpossibleAndIncomplete(t *testing.T) {
	if _, err := Parse("2026-02-31", "YYYY-MM-DD"); err == nil {
		t.Error("31 February was accepted")
	}
	if _, err := Parse("2026-11-06", "MM-DD"); err == nil {
		t.Error("a format without a year was accepted")
	}
	if _, err := Parse("not a date", "YYYY-MM-DD"); err == nil {
		t.Error("a name that does not match was accepted")
	}
}

func TestExpand_variables(t *testing.T) {
	v := Vars{Title: "Daily", Now: at(),
		DateFormat: "YYYY-MM-DD", TimeFormat: "HH:mm"}
	in := "# {{title}}\n{{date}} {{time}}\n{{date:YYYY/MM/DD}} " +
		"{{time:HH-mm}}\n{{cursor}} {{unknown}}"
	got, err := Expand(in, v)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"# Daily", "2026-11-06 15:04", "2026/11/06 15-04",
		"{{cursor}}", "{{unknown}}",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expansion is missing %q:\n%s", want, got)
		}
	}
}

func TestExpand_badInlineFormatIsAnError(t *testing.T) {
	_, err := Expand("{{date:YYYY-Do}}", Vars{Now: at()})
	if err == nil {
		t.Error("a bad inline format was expanded")
	}
}

func TestExpand_defaults(t *testing.T) {
	got, err := Expand("{{date}} {{time}}", Vars{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "{{") {
		t.Errorf("defaults did not expand: %q", got)
	}
}

// A caller may supply variables of its own, such as the tags of the
// vault, without this package having to know their names. Unknown ones
// are still left alone.
func TestExpand_extras(t *testing.T) {
	v := Vars{
		Title: "Daily",
		Extras: map[string]string{
			"known_tags": "work, ideas",
			"selection":  "the selected words",
		},
	}
	got, err := Expand(
		"# {{title}}\nTags: {{known_tags}}\n{{selection}}\n{{unknown}}", v)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"# Daily", "Tags: work, ideas", "the selected words",
		"{{unknown}}",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expansion is missing %q:\n%s", want, got)
		}
	}
}
