package markdown

import "testing"

// TestEveryFormatHasALabel checks that no button is shown without a name.
func TestEveryFormatHasALabel(t *testing.T) {
	for _, f := range Formats {
		if f.Label() == "" || f.Desc() == "" {
			t.Errorf("%d has no label or no description", int(f))
		}
	}
}

// TestPlan_surroundsTheSelection checks wrapping what is selected.
func TestPlan_surroundsTheSelection(t *testing.T) {
	got := Bold.Plan("hello world", 6, 11).Apply("hello world")
	if got != "hello **world**" {
		t.Errorf("got %q, want %q", got, "hello **world**")
	}
}

// TestPlan_insertsAPairWhenNothingIsSelected checks the other half: a
// caret in the text, and no selection.
func TestPlan_insertsAPairWhenNothingIsSelected(t *testing.T) {
	p := Bold.Plan("hello", 5, 5)
	if got := p.Apply("hello"); got != "hello****" {
		t.Errorf("got %q, want %q", got, "hello****")
	}
	if p.Caret != 7 {
		t.Errorf("caret %d, want 7, between the marks", p.Caret)
	}
}

// TestPlan_linkAndImagePlaceholders checks the two buttons whose second
// half is a placeholder to fill in.
func TestPlan_linkAndImagePlaceholders(t *testing.T) {
	if got := FormatLink.Plan("see", 0, 3).Apply("see"); got != "[see](url)" {
		t.Errorf("link: got %q", got)
	}
	if got := Image.Plan("alt", 0, 3).Apply("alt"); got != "![alt](url)" {
		t.Errorf("image: got %q", got)
	}
}

// TestPlan_blockAtTheStartOfALine checks a line's beginning: no break is
// added when the caret is already there.
func TestPlan_blockAtTheStartOfALine(t *testing.T) {
	p := FormatHeading.Plan("# old\n", 6, 6)
	if got := p.Apply("# old\n"); got != "# old\n# " {
		t.Errorf("got %q", got)
	}
	if p.Caret != 8 {
		t.Errorf("caret %d, want 8", p.Caret)
	}
}

// TestPlan_blockInTheMiddleOfALine checks that a mark that means a line
// gets a line of its own.
func TestPlan_blockInTheMiddleOfALine(t *testing.T) {
	got := Quote.Plan("text", 4, 4).Apply("text")
	if got != "text\n> " {
		t.Errorf("got %q, want %q", got, "text\n> ")
	}
}

// TestPlan_taskListAndRule checks the block buttons with more to them.
func TestPlan_taskListAndRule(t *testing.T) {
	if got := TaskList.Plan("", 0, 0).Apply(""); got != "- [ ] " {
		t.Errorf("task: got %q", got)
	}
	if got := HorizontalRule.Plan("a", 1, 1).Apply("a"); got != "a\n---\n" {
		t.Errorf("rule: got %q", got)
	}
}

// TestPlan_templateLeavesTheCaretBetween checks a code block: the caret
// goes where the code is to be typed, not after the closing fence.
func TestPlan_templateLeavesTheCaretBetween(t *testing.T) {
	p := CodeBlock.Plan("note", 4, 4)
	got := p.Apply("note")
	if got != "note\n```\n\n```\n" {
		t.Errorf("got %q", got)
	}
	if p.Caret != len("note\n```\n") {
		t.Errorf("caret %d, want %d, between the fences", p.Caret,
			len("note\n```\n"))
	}
}

// TestPlan_tableTemplate checks the table's shape.
func TestPlan_tableTemplate(t *testing.T) {
	got := Table.Plan("", 0, 0).Apply("")
	want := "| 列1 | 列2 |\n| --- | --- |\n|  |     |\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestPlan_reversedSelectionIsWrappedToo checks a selection dragged
// backwards: it wraps the same way.
func TestPlan_reversedSelectionIsWrappedToo(t *testing.T) {
	got := Italic.Plan("hello world", 11, 6).Apply("hello world")
	if got != "hello *world*" {
		t.Errorf("got %q, want %q", got, "hello *world*")
	}
}
