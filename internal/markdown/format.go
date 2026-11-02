// Package markdown is what the app knows about Markdown beyond drawing
// it: the edits its toolbar makes to a note, and later the facts it takes
// out of one.
package markdown

import (
	"strings"

	"github.com/helightxu/xinkgo/internal/locale"
)

// Format is one button of the Markdown toolbar.
type Format int

// The buttons, in the order they are shown. Two of them are named for
// what they write, which is also what a note holds — a heading, a link —
// so the Format in front of their names keeps the two apart.
const (
	Bold Format = iota
	Italic
	Strikethrough
	InlineCode
	FormatHeading
	BulletList
	NumberedList
	TaskList
	Quote
	CodeBlock
	FormatLink
	Image
	Table
	HorizontalRule
)

// Groups are the buttons as they stand in the toolbar: what goes around
// what is selected, what a line begins with, and what is put in whole.
var Groups = [][]Format{
	{Bold, Italic, Strikethrough, InlineCode},
	{FormatHeading, BulletList, NumberedList, TaskList, Quote, CodeBlock},
	{FormatLink, Image, Table, HorizontalRule},
}

// Formats are the buttons, in the order they are shown.
var Formats = everyFormat()

// everyFormat is every button of every group, in the order they are
// shown.
func everyFormat() []Format {
	var all []Format
	for _, group := range Groups {
		all = append(all, group...)
	}
	return all
}

// Label is what the button says.
func (f Format) Label() string {
	switch f {
	case Bold:
		return locale.FormatBold()
	case Italic:
		return locale.FormatItalic()
	case Strikethrough:
		return locale.FormatStrikethrough()
	case InlineCode:
		return locale.FormatInlineCode()
	case FormatHeading:
		return locale.FormatHeading()
	case BulletList:
		return locale.FormatBulletList()
	case NumberedList:
		return locale.FormatNumberedList()
	case TaskList:
		return locale.FormatTaskList()
	case Quote:
		return locale.FormatQuote()
	case CodeBlock:
		return locale.FormatCodeBlock()
	case FormatLink:
		return locale.FormatLink()
	case Image:
		return locale.FormatImage()
	case Table:
		return locale.FormatTable()
	default:
		return locale.FormatRule()
	}
}

// Desc is what the button means, for the tooltip.
func (f Format) Desc() string {
	switch f {
	case Bold:
		return locale.FormatBoldDesc()
	case Italic:
		return locale.FormatItalicDesc()
	case Strikethrough:
		return locale.FormatStrikethroughDesc()
	case InlineCode:
		return locale.FormatInlineCodeDesc()
	case FormatHeading:
		return locale.FormatHeadingDesc()
	case BulletList:
		return locale.FormatBulletListDesc()
	case NumberedList:
		return locale.FormatNumberedListDesc()
	case TaskList:
		return locale.FormatTaskListDesc()
	case Quote:
		return locale.FormatQuoteDesc()
	case CodeBlock:
		return locale.FormatCodeBlockDesc()
	case FormatLink:
		return locale.FormatLinkDesc()
	case Image:
		return locale.FormatImageDesc()
	case Table:
		return locale.FormatTableDesc()
	default:
		return locale.FormatRuleDesc()
	}
}

// How a button writes: surround wraps what is selected, block writes a
// line's beginning, template writes two parts with the caret between
// them, as a code block does.
const (
	surround = iota
	block
	template
)

// syntax is what the button writes, and how.
func (f Format) syntax() (before, after string, kind int) {
	switch f {
	case Bold:
		return "**", "**", surround
	case Italic:
		return "*", "*", surround
	case Strikethrough:
		return "~~", "~~", surround
	case InlineCode:
		return "`", "`", surround
	case FormatLink:
		return "[", "](url)", surround
	case Image:
		return "![", "](url)", surround
	case FormatHeading:
		return "# ", "", block
	case BulletList:
		return "- ", "", block
	case NumberedList:
		return "1. ", "", block
	case TaskList:
		return "- [ ] ", "", block
	case Quote:
		return "> ", "", block
	case HorizontalRule:
		return "---\n", "", block
	case CodeBlock:
		return "```\n", "\n```\n", template
	default:
		return "| 列1 | 列2 |\n| --- | --- |\n| ", " |     |\n", template
	}
}

// Plan is what a button does: replace the text between Start and End with
// Text, and leave the caret at Caret. Offsets are bytes into the note.
type Plan struct {
	Start int
	End   int
	Text  string
	Caret int
}

// Plan is what the button does to a note with the text between selStart
// and selEnd selected: the same plan whether text is selected or only a
// caret stands in it, which is what makes one button serve both.
func (f Format) Plan(text string, selStart, selEnd int) Plan {
	before, after, kind := f.syntax()
	if selStart > selEnd {
		selStart, selEnd = selEnd, selStart
	}
	if selStart < 0 {
		selStart = 0
	}
	if selEnd > len(text) {
		selEnd = len(text)
	}
	switch kind {
	case surround:
		if selStart == selEnd {
			// Nothing selected: a pair of marks with the caret in it.
			return Plan{
				Start: selStart, End: selStart,
				Text:  before + after,
				Caret: selStart + len(before),
			}
		}
		inner := text[selStart:selEnd]
		inserted := before + inner + after
		return Plan{
			Start: selStart, End: selEnd,
			Text:  inserted,
			Caret: selStart + len(inserted),
		}
	case block:
		// A line's beginning: a note written in the middle of a line
		// gets a break before it, so that the mark means what it says.
		lead := lineBreakBefore(text, selStart)
		inserted := lead + before
		return Plan{
			Start: selStart, End: selStart,
			Text:  inserted,
			Caret: selStart + len(inserted),
		}
	default:
		lead := lineBreakBefore(text, selStart)
		inserted := lead + before + after
		return Plan{
			Start: selStart, End: selStart,
			Text:  inserted,
			Caret: selStart + len(lead) + len(before),
		}
	}
}

// Apply is the note the plan leaves.
func (p Plan) Apply(text string) string {
	if p.Start > len(text) || p.End > len(text) {
		return text
	}
	return text[:p.Start] + p.Text + text[p.End:]
}

// lineBreakBefore is "\n" when a caret before start is in the middle of a
// line, and "" when it already stands at a line's beginning.
func lineBreakBefore(text string, start int) string {
	if start <= 0 || start > len(text) {
		return ""
	}
	if strings.HasSuffix(text[:start], "\n") {
		return ""
	}
	return "\n"
}
