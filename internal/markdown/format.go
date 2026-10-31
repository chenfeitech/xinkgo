// Package markdown is what the app knows about Markdown beyond drawing
// it: the edits its toolbar makes to a note, and later the facts it takes
// out of one.
package markdown

import "strings"

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
		return "粗"
	case Italic:
		return "斜"
	case Strikethrough:
		return "删"
	case InlineCode:
		return "码"
	case FormatHeading:
		return "标题"
	case BulletList:
		return "无序"
	case NumberedList:
		return "有序"
	case TaskList:
		return "任务"
	case Quote:
		return "引用"
	case CodeBlock:
		return "代码块"
	case FormatLink:
		return "链接"
	case Image:
		return "图片"
	case Table:
		return "表格"
	default:
		return "横线"
	}
}

// Desc is what the button means, for the tooltip.
func (f Format) Desc() string {
	switch f {
	case Bold:
		return "加粗：包裹选中文本，或插入 ** 标记"
	case Italic:
		return "斜体：包裹选中文本，或插入 * 标记"
	case Strikethrough:
		return "删除线：包裹选中文本，或插入 ~~ 标记"
	case InlineCode:
		return "行内代码：包裹选中文本，或插入 ` 标记"
	case FormatHeading:
		return "标题：插入 # 标题行"
	case BulletList:
		return "无序列表：插入 - 列表项"
	case NumberedList:
		return "有序列表：插入 1. 列表项"
	case TaskList:
		return "任务列表：插入待办项 - [ ]"
	case Quote:
		return "引用：插入 > 引用块"
	case CodeBlock:
		return "代码块：插入 ``` 代码块"
	case FormatLink:
		return "链接：插入 [文本](地址)，无选中则插入占位"
	case Image:
		return "图片：插入 ![描述](地址)，无选中则插入占位"
	case Table:
		return "表格：插入 Markdown 表格模板"
	default:
		return "分割线：插入 --- 水平分隔线"
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
