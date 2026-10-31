package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// place is a skill written to disk: its folder, named, holding a
// SKILL.md that says what is given.
func place(t *testing.T, dir, name, text string) string {
	t.Helper()
	folder := filepath.Join(dir, name)
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, File), []byte(text),
		0o644); err != nil {
		t.Fatal(err)
	}
	return folder
}

// front is a SKILL.md of that name and description, with the body given.
func front(name, description, body string) string {
	return "---\nname: " + name + "\ndescription: " + description +
		"\n---\n\n" + body
}

func TestDiscover_bothPlacesAreLookedIn(t *testing.T) {
	user := t.TempDir()
	vault := t.TempDir()
	place(t, filepath.Join(user, "skills"), "one", front("one", "does one",
		"body one"))
	place(t, filepath.Join(vault, ".xink", "skills"), "two",
		front("two", "does two", "body two"))

	skills, issues := Discover(vault, filepath.Join(user, "skills"))
	if len(issues) != 0 {
		t.Fatalf("issues = %v", issues)
	}
	if len(skills) != 2 {
		t.Fatalf("skills = %v, want two", skills)
	}
	if skills[0].Name != "one" || skills[0].From != User {
		t.Fatalf("first = %+v, want the user's one", skills[0])
	}
	if skills[1].Name != "two" || skills[1].From != Vault {
		t.Fatalf("second = %+v, want the vault's two", skills[1])
	}
	if skills[1].Body != "body two" {
		t.Fatalf("body = %q", skills[1].Body)
	}
	// The folder it stands in is kept: what it brings with it is in
	// there.
	if !strings.HasSuffix(skills[1].Dir, "two") {
		t.Fatalf("dir = %q", skills[1].Dir)
	}
}

func TestDiscover_theVaultsOwnWinsWhenTwoAreCalledTheSame(t *testing.T) {
	user := t.TempDir()
	vault := t.TempDir()
	place(t, filepath.Join(user, "skills"), "same",
		front("same", "the user's", "user body"))
	place(t, filepath.Join(vault, ".xink", "skills"), "same",
		front("same", "the vault's", "vault body"))

	skills, _ := Discover(vault, filepath.Join(user, "skills"))
	if len(skills) != 1 {
		t.Fatalf("skills = %v, want one", skills)
	}
	if skills[0].Body != "vault body" {
		t.Fatalf("body = %q, want the vault's", skills[0].Body)
	}
	if skills[0].From != Vault {
		t.Fatalf("from = %v, want the vault's", skills[0].From)
	}
}

func TestDiscover_aFolderWithNoSkillFileInItIsNotASkill(t *testing.T) {
	dir := t.TempDir()
	folder := filepath.Join(dir, "empty")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "README.md"),
		[]byte("nothing here"), 0o644); err != nil {
		t.Fatal(err)
	}
	skills, issues := Discover("", dir)
	if len(skills) != 0 || len(issues) != 0 {
		t.Fatalf("skills = %v issues = %v, want neither", skills, issues)
	}
}

func TestDiscover_whatCannotBeUsedIsSaidAndNotDropped(t *testing.T) {
	dir := t.TempDir()
	place(t, dir, "noname", "---\ndescription: no name here\n---\n\nbody")
	place(t, dir, "nodesc", "---\nname: nodesc\n---\n\nbody")
	place(t, dir, "unclosed", "---\nname: unclosed\n\nbody")
	place(t, dir, "nofront", "just words")

	skills, issues := Discover("", dir)
	if len(skills) != 0 {
		t.Fatalf("skills = %v, want none", skills)
	}
	if len(issues) != 4 {
		t.Fatalf("issues = %v, want four", issues)
	}
	reasons := map[string]bool{}
	for _, one := range issues {
		reasons[one.Reason] = true
	}
	// Each is given a reason the user can act on.
	for _, want := range []string{"缺少 name", "缺少 description",
		"front-matter 没有闭合", "缺少 front-matter"} {
		said := false
		for reason := range reasons {
			if strings.Contains(reason, want) {
				said = true
			}
		}
		if !said {
			t.Errorf("no issue says %q (%v)", want, reasons)
		}
	}
}

func TestDiscover_aFolderThatCannotBeReadIsNotAnError(t *testing.T) {
	// A machine with no skills at all is not a broken machine.
	skills, issues := Discover(filepath.Join(t.TempDir(), "nowhere"),
		filepath.Join(t.TempDir(), "nowhere"))
	if len(skills) != 0 || len(issues) != 0 {
		t.Fatalf("skills = %v issues = %v", skills, issues)
	}
}

func TestParse_theQuotesAroundAValueAreTakenOff(t *testing.T) {
	dir := t.TempDir()
	place(t, dir, "quoted",
		"---\nname: \"quoted\"\ndescription: 'what it does'\n---\n\nbody")
	skills, _ := Discover("", dir)
	if len(skills) != 1 {
		t.Fatalf("skills = %v", skills)
	}
	if skills[0].Name != "quoted" {
		t.Fatalf("name = %q", skills[0].Name)
	}
	if skills[0].Description != "what it does" {
		t.Fatalf("description = %q", skills[0].Description)
	}
}

func TestParse_aLongBodyIsCutShort(t *testing.T) {
	dir := t.TempDir()
	place(t, dir, "long", front("long", "is long",
		strings.Repeat("a", MaxBodyChars+100)))
	skills, _ := Discover("", dir)
	if len(skills) != 1 {
		t.Fatalf("skills = %v", skills)
	}
	if n := len([]rune(skills[0].Body)); n != MaxBodyChars {
		t.Fatalf("body is %d characters, want %d", n, MaxBodyChars)
	}
}

func TestCatalog_theSkillsAsAModelIsToldOfThem(t *testing.T) {
	skills := []Skill{
		{Name: "one", Description: "does one"},
		{Name: "two", Description: "does two"},
	}
	got := Catalog(skills)
	for _, want := range []string{"可用技能", "one", "does one",
		"two", "does two"} {
		if !strings.Contains(got, want) {
			t.Errorf("catalog = %q, want %q in it", got, want)
		}
	}
	if got := Catalog(nil); got != "" {
		t.Fatalf("catalog = %q, want nothing", got)
	}
}

func TestFind_theSkillOfThatName(t *testing.T) {
	skills := []Skill{{Name: "one"}, {Name: "two"}}
	if Find(skills, "two") == nil {
		t.Fatal("two was not found")
	}
	if Find(skills, "three") != nil {
		t.Fatal("three was found although there is no such skill")
	}
}

func TestEnabled_theOnesThatWereTurnedOffAreNotUsed(t *testing.T) {
	skills := []Skill{{Name: "one"}, {Name: "two"}, {Name: "three"}}
	got := Enabled(skills, []string{"two"})
	if len(got) != 2 || got[0].Name != "one" || got[1].Name != "three" {
		t.Fatalf("enabled = %v", got)
	}
	if got := Enabled(skills, []string{"one", "two", "three"}); len(got) != 0 {
		t.Fatalf("enabled = %v, want none", got)
	}
}

func TestResource_whatASkillBringsWithIt(t *testing.T) {
	one := Skill{Dir: filepath.Join("/vault", ".xink", "skills", "one")}
	got, ok := Resource(one, "references/api.md")
	if !ok {
		t.Fatal("a file of the skill's own was refused")
	}
	if got != filepath.Join(one.Dir, "references", "api.md") {
		t.Fatalf("path = %q", got)
	}
}

func TestResource_aSkillDoesNotReachOutOfItsFolder(t *testing.T) {
	one := Skill{Dir: filepath.Join("/vault", ".xink", "skills", "one")}
	for _, rel := range []string{
		"../other/ref.md",
		"a/../../out.md",
		"../../a.md",
		"/etc/passwd",
		"",
		"   ",
	} {
		if _, ok := Resource(one, rel); ok {
			t.Errorf("%q was allowed out of the folder", rel)
		}
	}
}

func TestSource_whereItCameFromIsSaid(t *testing.T) {
	if Vault.Label() == User.Label() {
		t.Fatal("the two places read the same")
	}
	if Vault.Label() == "" || User.Label() == "" {
		t.Fatal("a place is not named")
	}
}

func TestDirs_whereTheSkillsAreLookedFor(t *testing.T) {
	if got := VaultDir("/vault"); got != filepath.Join("/vault",
		".xink", "skills") {
		t.Fatalf("vault dir = %q", got)
	}
	if got := UserDir("/app"); got != filepath.Join("/app", "skills") {
		t.Fatalf("user dir = %q", got)
	}
}
