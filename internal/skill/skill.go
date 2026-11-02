// Package skill is the skills a model may be told to use: what they are
// called, where they were found, and what they say.
//
// A skill is a folder holding a `SKILL.md`, with the skill's name and
// what it is for written at the top of it between two lines of dashes,
// and the rest of the file being what the model is told. They are looked
// for in two places: in the vault, under the AI's own directory, and in
// the application's own. One in the vault is the one that is used when
// two of them are called the same.
package skill

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/media"
)

// File is the name a skill's file has.
const File = "SKILL.md"

// MaxBodyChars is the longest a skill's body may be read: a longer one
// would eat the whole of what the model can be told at once.
const MaxBodyChars = 32_000

// Source is where a skill was found.
type Source int

// The two places a skill may come from.
const (
	// Vault is the vault's own: `<vault>/.xink/skills/<name>/`.
	Vault Source = iota
	// User is the application's own: `<app data>/skills/<name>/`.
	User
)

// Label is where it came from, said plainly, in the language the app
// shows.
func (s Source) Label() string {
	if s == Vault {
		return locale.SkillFromVault()
	}
	return locale.SkillFromUser()
}

// Skill is one that can be used.
type Skill struct {
	// Name and Description are what is written at the top of the file.
	Name        string
	Description string
	// Dir is the folder the file stands in: what a skill brings with
	// it — its references and the like — is looked for in there.
	Dir string
	// Body is the rest of the file: what the model is told.
	Body string
	// From is where it was found.
	From Source
}

// Issue is a skill that was found but cannot be used, and why. One that
// cannot be used is said so rather than quietly left out: a skill the
// user wrote and cannot see is worse than an error on the screen.
type Issue struct {
	Path   string
	Reason string
}

// VaultDir is where the vault's own skills are.
func VaultDir(root string) string {
	return filepath.Join(root, media.AIDirName, "skills")
}

// UserDir is where the application's own skills are.
func UserDir(appData string) string {
	return filepath.Join(appData, "skills")
}

// Discover looks for skills in both places. One in the vault is the one
// that is used when two of them are called the same.
//
// A file that cannot be read, or that has no name or no description
// written in it, or whose front matter is never closed, is not a skill:
// it is an issue, and is given back as one.
func Discover(vaultRoot, userDir string) ([]Skill, []Issue) {
	skills := []Skill{}
	issues := []Issue{}
	// The user's own are looked at first, so that one in the vault
	// takes its place when they are called the same.
	places := []struct {
		dir    string
		source Source
	}{{userDir, User}}
	if vaultRoot != "" {
		places = append(places,
			struct {
				dir    string
				source Source
			}{VaultDir(vaultRoot), Vault})
	}
	for _, place := range places {
		for _, dir := range childDirs(place.dir) {
			path := filepath.Join(dir, File)
			text, err := os.ReadFile(path)
			if err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					issues = append(issues, Issue{
						Path:   path,
						Reason: locale.SkillIssueNotRead(err.Error()),
					})
				}
				continue
			}
			one, reason := parse(string(text), place.source)
			if reason != "" {
				issues = append(issues, Issue{
					Path: path, Reason: reason,
				})
				continue
			}
			one.Dir = dir
			// One of the same name, found before, gives way.
			kept := skills[:0:0]
			for _, other := range skills {
				if other.Name != one.Name {
					kept = append(kept, other)
				}
			}
			skills = append(kept, one)
		}
	}
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})
	return skills, issues
}

// parse is a skill's file read: the name and what it is for from between
// the two lines of dashes, and the rest as the body.
func parse(text string, source Source) (Skill, string) {
	text = strings.TrimPrefix(text, "\ufeff")
	if !strings.HasPrefix(text, "---") {
		return Skill{}, locale.SkillIssueNoFrontMatter()
	}
	rest := strings.TrimPrefix(text[3:], "\n")
	end, closed := frontMatterEnd(rest)
	if !closed {
		return Skill{}, locale.SkillIssueFrontMatterUnclosed()
	}
	body := ""
	if at := strings.Index(rest[end:], "\n"); at >= 0 {
		body = strings.TrimLeft(rest[end+at+1:], " \t\r\n")
	}
	one := Skill{From: source}
	for _, line := range strings.Split(rest[:end], "\n") {
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		value = strings.Trim(strings.Trim(strings.TrimSpace(value),
			`"`), `'`)
		switch strings.TrimSpace(key) {
		case "name":
			one.Name = value
		case "description":
			one.Description = value
		}
	}
	if strings.TrimSpace(one.Name) == "" {
		return Skill{}, locale.SkillIssueNoName()
	}
	if strings.TrimSpace(one.Description) == "" {
		return Skill{}, locale.SkillIssueNoDescription()
	}
	one.Body = truncateChars(body, MaxBodyChars)
	return one, ""
}

// frontMatterEnd is where the closing line of dashes stands.
func frontMatterEnd(rest string) (at int, closed bool) {
	offset := 0
	for _, line := range strings.Split(rest, "\n") {
		if strings.TrimSpace(line) == "---" {
			return offset, true
		}
		offset += len(line) + 1
	}
	return 0, false
}

// childDirs is the folders standing in dir, in the order they are named.
func childDirs(dir string) []string {
	names, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := []string{}
	for _, name := range names {
		if !name.IsDir() {
			continue
		}
		out = append(out, filepath.Join(dir, name.Name()))
	}
	sort.Strings(out)
	return out
}

// truncateChars is the first of a text, as many characters as count.
func truncateChars(text string, count int) string {
	runes := []rune(text)
	if len(runes) <= count {
		return text
	}
	return string(runes[:count])
}

// Catalog is the skills as a model is told of them: the names, and what
// each is for.
func Catalog(skills []Skill) string {
	if len(skills) == 0 {
		return ""
	}
	out := strings.Builder{}
	out.WriteString("可用技能：\n")
	for _, one := range skills {
		out.WriteString("- " + one.Name + "：" + one.Description + "\n")
	}
	return out.String()
}

// Find is the skill of that name, when there is one.
func Find(skills []Skill, name string) *Skill {
	for i := range skills {
		if skills[i].Name == name {
			return &skills[i]
		}
	}
	return nil
}

// Enabled is the skills that have not been turned off.
func Enabled(skills []Skill, disabled []string) []Skill {
	out := make([]Skill, 0, len(skills))
	for _, one := range skills {
		if turnedOff(disabled, one.Name) {
			continue
		}
		out = append(out, one)
	}
	return out
}

// turnedOff is whether the name is among those turned off.
func turnedOff(disabled []string, name string) bool {
	for _, one := range disabled {
		if one == name {
			return true
		}
	}
	return false
}

// Resource is where a file a skill brings with it stands. Only what is
// inside the skill's own folder can be asked for: a skill is not to
// reach out of it, and take a file of the vault's, or of the machine's.
func Resource(one Skill, rel string) (string, bool) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", false
	}
	if filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" {
		return "", false
	}
	clean := filepath.Clean(rel)
	for _, part := range strings.Split(clean, string(filepath.Separator)) {
		if part == ".." {
			return "", false
		}
	}
	if clean == ".." || strings.HasPrefix(clean, "..") {
		return "", false
	}
	return filepath.Join(one.Dir, clean), true
}
