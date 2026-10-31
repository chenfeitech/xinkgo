package ai

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/helightxu/xinkgo/internal/index"
	"github.com/helightxu/xinkgo/internal/search"
	"github.com/helightxu/xinkgo/internal/skill"
)

// vaultHost is a vault a tool can look through, held in memory: two
// notes, one pointing at the other, one tag between them, and one skill
// with a file of its own.
type vaultHost struct {
	root  string
	notes map[string]string
}

func newVaultHost(t *testing.T) *vaultHost {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, ".xink", "skills", "demo")
	if err := os.MkdirAll(filepath.Join(dir, "references"),
		0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "references", "api.md"),
		[]byte("the api"), 0o644); err != nil {
		t.Fatal(err)
	}
	return &vaultHost{
		root: root,
		notes: map[string]string{
			filepath.Join(root, "a.md"): "# A\n\nsome words here\n",
			filepath.Join(root, "b.md"): "# B\n\nsee [[a]]\n",
		},
	}
}

func (h *vaultHost) ReadNote(path string) (string, bool) {
	text, ok := h.notes[path]
	return text, ok
}

func (h *vaultHost) Search(query string, limit int) []search.Hit {
	return []search.Hit{
		{Path: filepath.Join(h.root, "a.md"), Line: 2,
			Text: "some words here"},
	}
}

func (h *vaultHost) Tags() []index.TagCount {
	return []index.TagCount{{Name: "note", Count: 3}, {Name: "x", Count: 1}}
}

func (h *vaultHost) Backlinks(path string) []index.LinkRef {
	return []index.LinkRef{
		{From: filepath.Join(h.root, "b.md"), Line: 2, Target: "a"},
	}
}

func (h *vaultHost) Outline(path string) []string {
	return []string{"A"}
}

func (h *vaultHost) ListNotes(limit int) []NoteRef {
	return []NoteRef{
		{Path: filepath.Join(h.root, "a.md"), Title: "A"},
		{Path: filepath.Join(h.root, "b.md"), Title: "B"},
	}
}

func (h *vaultHost) Root() string { return h.root }

func (h *vaultHost) Skill(name string) (*skill.Skill, bool) {
	if name != "demo" {
		return nil, false
	}
	return &skill.Skill{
		Name: "demo", Description: "a demo",
		Dir: filepath.Join(h.root, ".xink", "skills", "demo"),
	}, true
}

// said is what a tool came back with, run against the memory vault.
func said(t *testing.T, name, arguments string) string {
	t.Helper()
	return Execute(ToolCall{Name: name, Arguments: arguments},
		newVaultHost(t), 0).Output
}

func TestExecute_aNoteIsRead(t *testing.T) {
	host := newVaultHost(t)
	got := Execute(ToolCall{Name: ToolReadNote, Arguments: `{"path":"` +
		filepath.Join(host.root, "a.md") + `"}`}, host, 0)
	if !strings.Contains(got.Output, "some words here") {
		t.Fatalf("output = %q, want the note's words", got.Output)
	}
	if got.Name != ToolReadNote {
		t.Fatalf("name = %q", got.Name)
	}
}

func TestExecute_aNoteThatIsNotThereIsSaidSo(t *testing.T) {
	got := said(t, ToolReadNote, `{"path":"/nowhere/x.md"}`)
	if !strings.Contains(got, "读不到") {
		t.Fatalf("output = %q, want it said that it could not be read",
			got)
	}
}

func TestExecute_aToolWithoutWhatItNeedsSaysWhatItWasNotTold(t *testing.T) {
	for _, name := range []string{
		ToolReadNote, ToolOutline, ToolListBacklinks,
	} {
		if got := said(t, name, `{}`); !strings.Contains(got, "path") {
			t.Errorf("%s: output = %q, want it said what was missing",
				name, got)
		}
	}
	if got := said(t, ToolSearchNotes, `{}`); !strings.Contains(got,
		"query") {
		t.Errorf("output = %q, want it said what was missing", got)
	}
}

func TestExecute_aToolTheAppHasNeverHeardOfIsSaidSo(t *testing.T) {
	got := said(t, "fly_away", `{}`)
	if !strings.Contains(got, "没有这个工具") {
		t.Fatalf("output = %q", got)
	}
	// And it is told what there is, so that it may ask better.
	for _, name := range Tools {
		if !strings.Contains(got, name) {
			t.Errorf("output = %q, want %q offered", got, name)
		}
	}
}

func TestExecute_argumentsThatMakeNoSenseAreTakenForNoneGiven(t *testing.T) {
	// Not JSON at all, and JSON that is not an object: neither is a
	// reason to stop, and both are the same as being told nothing.
	for _, arguments := range []string{"", "not json", `[]`, `null`} {
		if got := said(t, ToolReadNote, arguments); !strings.Contains(
			got, "path") {
			t.Errorf("%q: output = %q", arguments, got)
		}
	}
}

func TestExecute_theSearchHitsAreShownFromTheRoot(t *testing.T) {
	got := said(t, ToolSearchNotes, `{"query":"words"}`)
	for _, want := range []string{"a.md", "3", "some words here"} {
		if !strings.Contains(got, want) {
			t.Errorf("output = %q, want %q in it", got, want)
		}
	}
	// The whole path would be longer than it needs to be.
	if strings.Contains(got, newVaultHost(t).root) {
		t.Error("the path was said whole, and not from the root")
	}
}

func TestExecute_theTagsAreSaidWithHowManyNotesUseThem(t *testing.T) {
	got := said(t, ToolListTags, `{}`)
	// The most used comes first.
	if !strings.HasPrefix(got, "#note (3)") {
		t.Fatalf("output = %q, want the most used first", got)
	}
	if !strings.Contains(got, "#x (1)") {
		t.Fatalf("output = %q", got)
	}
}

func TestExecute_theBacklinksSayWhereFrom(t *testing.T) {
	got := said(t, ToolListBacklinks, `{"path":"a.md"}`)
	for _, want := range []string{"b.md", "3", "[[a]]"} {
		if !strings.Contains(got, want) {
			t.Errorf("output = %q, want %q in it", got, want)
		}
	}
}

func TestExecute_theOutlineAndTheNotesAreSaid(t *testing.T) {
	if got := said(t, ToolOutline, `{"path":"a.md"}`); got != "A" {
		t.Errorf("outline = %q", got)
	}
	got := said(t, ToolListNotes, `{}`)
	for _, want := range []string{"a.md", "b.md", "—"} {
		if !strings.Contains(got, want) {
			t.Errorf("notes = %q, want %q in it", got, want)
		}
	}
}

func TestExecute_aSkillMayOnlyReadWhatIsInItsOwnFolder(t *testing.T) {
	got := said(t, ToolReadSkillResource,
		`{"skill":"demo","path":"references/api.md"}`)
	if got != "the api" {
		t.Fatalf("output = %q, want the resource", got)
	}
	// Reaching out of the folder is refused.
	for _, rel := range []string{"../../a.md", "/etc/passwd"} {
		got := said(t, ToolReadSkillResource,
			`{"skill":"demo","path":"`+rel+`"}`)
		if !strings.Contains(got, "不允许") {
			t.Errorf("%q: output = %q, want it refused", rel, got)
		}
	}
	if got := said(t, ToolReadSkillResource,
		`{"skill":"nope","path":"a.md"}`); !strings.Contains(got,
		"没有这个技能") {
		t.Errorf("output = %q", got)
	}
}

func TestExecute_aLongAnswerIsCutShortAndSaidToBe(t *testing.T) {
	host := newVaultHost(t)
	got := Execute(ToolCall{Name: ToolReadNote, Arguments: `{"path":"` +
		filepath.Join(host.root, "a.md") + `"}`}, host, 5)
	if !strings.Contains(got.Output, "已截断") {
		t.Fatalf("output = %q, want it said that it was cut short",
			got.Output)
	}
}

func TestExecute_howManyIsClamped(t *testing.T) {
	// A model that asks for more than there is room for is given the
	// most there is room for, and no more.
	for _, arguments := range []string{
		`{"query":"x","limit":10000}`, `{"query":"x","limit":0}`,
		`{"query":"x","limit":"3"}`,
	} {
		if got := said(t, ToolSearchNotes, arguments); !strings.Contains(
			got, "some words here") {
			t.Errorf("%s: output = %q", arguments, got)
		}
	}
}

func TestSpecs_everyToolIsDescribed(t *testing.T) {
	data := Specs(true)
	var specs []map[string]any
	if err := json.Unmarshal(data, &specs); err != nil {
		t.Fatalf("specs = %s: %v", data, err)
	}
	if len(specs) != len(Tools) {
		t.Fatalf("got %d specs, want %d", len(specs), len(Tools))
	}
	for i, spec := range specs {
		fn, ok := spec["function"].(map[string]any)
		if !ok {
			t.Fatalf("spec %d = %v", i, spec)
		}
		if fn["name"] != Tools[i] {
			t.Errorf("spec %d is %v, want %q", i, fn["name"], Tools[i])
		}
		if fn["description"] == "" {
			t.Errorf("%s is not described", Tools[i])
		}
		if fn["parameters"] == nil {
			t.Errorf("%s has no parameters", Tools[i])
		}
	}
}

func TestSpecs_theSkillResourceIsLeftOutWhenSkillsAreOff(t *testing.T) {
	data := Specs(false)
	if strings.Contains(string(data), ToolReadSkillResource) {
		t.Fatalf("specs = %s, want the skill resource left out", data)
	}
	var specs []map[string]any
	if err := json.Unmarshal(data, &specs); err != nil {
		t.Fatal(err)
	}
	if len(specs) != len(Tools)-1 {
		t.Fatalf("got %d specs, want %d", len(specs), len(Tools)-1)
	}
}

func TestClip_aShortAnswerIsLeftAlone(t *testing.T) {
	if got := clip("short", 100); got != "short" {
		t.Fatalf("clip = %q", got)
	}
	if got := clip("anything", 0); got != "anything" {
		t.Fatalf("clip = %q", got)
	}
}

func TestClip_aCutIsNotMadeMidCharacter(t *testing.T) {
	// Cutting by characters, and not by bytes: a Chinese character is
	// three bytes long, and half of one is not a thing to hand to a
	// model.
	got := clip(strings.Repeat("字", 100), 10)
	runes := []rune(got)
	if len(runes) < 10 {
		t.Fatalf("clip = %q, want ten characters at least", got)
	}
	if !strings.Contains(got, strings.Repeat("字", 10)) {
		t.Fatalf("clip = %q", got)
	}
}
