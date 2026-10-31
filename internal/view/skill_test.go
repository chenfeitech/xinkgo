package view

import (
	"strings"
	"testing"

	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/skill"
)

// workspaceWithSkills is a window with one note open in it and two skills
// it may be told to use.
func workspaceWithSkills(t *testing.T) (*Workspace, *aiSession) {
	t.Helper()
	_, a, _ := vaultWithNotes(t)
	w := NewWorkspace(config.Default())
	if _, err := w.Tabs.Open(a); err != nil {
		t.Fatalf("Open: %v", err)
	}
	w.Skills = []skill.Skill{
		{Name: "polish", Description: "makes it read well",
			Body: "do it thus"},
		{Name: "tag", Description: "suggests tags", Body: "tag it thus"},
	}
	return w, w.aiOf(a)
}

// TestSkillItems_theSkillsComeUpInThePalette checks that a skill is found
// by its name, and shown as what it is for.
func TestSkillItems_theSkillsComeUpInThePalette(t *testing.T) {
	w, _ := workspaceWithSkills(t)
	w.paletteQuery = "pol"
	items := w.skillItems()
	if len(items) != 1 {
		t.Fatalf("items = %v, want the one", items)
	}
	if items[0].Name != "Skill · polish" {
		t.Fatalf("name = %q, want it said to be a skill", items[0].Name)
	}
	if items[0].Title != "makes it read well" {
		t.Fatalf("title = %q, want what it is for", items[0].Title)
	}
	if items[0].Skill == nil || items[0].Skill.Name != "polish" {
		t.Fatalf("skill = %v, want the one that was found", items[0].Skill)
	}
	// The letters that were typed are lit up where they stand, and not
	// in the prefix: the prefix is not part of the name.
	if len(items[0].Positions) == 0 {
		t.Fatal("no letters were lit up")
	}
	if items[0].Positions[0] < len("Skill · ") {
		t.Fatalf("positions = %v, want them after the prefix",
			items[0].Positions)
	}
}

// TestSkillItems_aVaultWithNoSkillsAnswersWithNotesOnly checks that the
// palette is still the notes' when there is no skill to show.
func TestSkillItems_aVaultWithNoSkillsAnswersWithNotesOnly(t *testing.T) {
	w := NewWorkspace(config.Default())
	if items := w.skillItems(); len(items) != 0 {
		t.Fatalf("items = %v, want none", items)
	}
}

// TestAiTold_theSkillsThereAreAreTold checks that a model is told what
// skills it may ask for, even when none has been taken up.
func TestAiTold_theSkillsThereAreAreTold(t *testing.T) {
	w, s := workspaceWithSkills(t)
	messages := w.aiTold(s, w.Tabs.ActiveNote())
	said := false
	for _, m := range messages {
		if m.Role == ai.System && strings.Contains(m.Content, "可用技能") {
			said = true
		}
	}
	if !said {
		t.Fatalf("messages = %v, want the skills named", messages)
	}
	// Naming them is not the same as telling what one of them says.
	for _, m := range messages {
		if strings.Contains(m.Content, "do it thus") {
			t.Fatal("a skill was told whole although none was taken up")
		}
	}
}

// TestUseSkill_whatTheSkillSaysIsTold checks that a skill taken up is
// told to the model whole, and that putting it down stops it being told.
func TestUseSkill_whatTheSkillSaysIsTold(t *testing.T) {
	w, s := workspaceWithSkills(t)
	w.useSkill(w.Skills[0])
	if !s.Open {
		t.Fatal("the bar did not open for it")
	}
	if s.Skill == nil || s.Skill.Name != "polish" {
		t.Fatalf("skill = %v, want the one taken up", s.Skill)
	}
	messages := w.aiTold(s, w.Tabs.ActiveNote())
	said := false
	for _, m := range messages {
		if m.Role == ai.System &&
			strings.Contains(m.Content, "do it thus") {
			said = true
		}
	}
	if !said {
		t.Fatalf("messages = %v, want the skill's words", messages)
	}

	s.Skill = nil
	messages = w.aiTold(s, w.Tabs.ActiveNote())
	for _, m := range messages {
		if strings.Contains(m.Content, "do it thus") {
			t.Fatal("the skill is still being told after being " +
				"put down")
		}
	}
}

// TestUseSkill_aWindowWithNoNoteHasNowhereToUseOne checks that taking a
// skill up with nothing open does not open a conversation out of nowhere.
func TestUseSkill_aWindowWithNoNoteHasNowhereToUseOne(t *testing.T) {
	w := NewWorkspace(config.Default())
	w.Skills = []skill.Skill{{Name: "polish"}}
	w.useSkill(w.Skills[0])
	if len(w.ai) != 0 {
		t.Fatalf("a conversation was made for no note at all: %v", w.ai)
	}
}

// TestChoose_aSkillChosenIsUsedAndANoteChosenIsOpened checks that one
// palette answers two kinds of thing, and each is done with as it should
// be.
func TestChoose_aSkillChosenIsUsedAndANoteChosenIsOpened(t *testing.T) {
	w, s := workspaceWithSkills(t)
	w.paletteOpen = true
	w.paletteQuery = "pol"
	items := w.paletteItems()
	if len(items) != 1 || items[0].Skill == nil {
		t.Fatalf("items = %v, want the skill", items)
	}
	w.choose(0)
	if w.paletteOpen {
		t.Fatal("the palette stayed open")
	}
	if s.Skill == nil || s.Skill.Name != "polish" {
		t.Fatalf("skill = %v, want the one chosen", s.Skill)
	}
}
