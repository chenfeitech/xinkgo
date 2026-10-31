// Package task is the long work done over a whole vault: a tag changed
// everywhere it stands, one added to many notes, one taken out of them.
//
// It is deliberately the kind of work that needs no model: what changes
// is decided by a rule on strings, so this layer is pure — it reads the
// disk once and says what it would do, and it can be tested without a
// vault on the screen.
//
// Three things hold:
//
//  1. A plan only reads. What it would do is shown before anything is
//     written, and writing is another step entirely.
//  2. Nothing is guessed at. A whole tag token is what is hit: `#AI` is,
//     `#AI助手` is not. A code fence and a front matter are read each in
//     its own way, and a note that is not hit is skipped — with the
//     reason said.
//  3. It can be taken up again. Where it got to, and what each note came
//     to, are written down, so what is already done is not done twice.
package task

import (
	"fmt"
	"strings"
	"time"
)

// BatchSize is how many notes one run does: it stops and reports when it
// comes to that number, so that a vault of thousands is not rewritten in
// one breath, and what it has done can be looked at.
const BatchSize = 20

// Kind is what a task does.
type Kind int

// The three kinds of task there are.
const (
	// Retag is `#target` changed into `#new` everywhere it stands.
	Retag Kind = iota
	// AddTag is a tag put on every note of the set.
	AddTag
	// RemoveTag is a tag taken out of every note of the set.
	RemoveTag
)

// ID is the kind named as it is written down.
func (k Kind) ID() string {
	switch k {
	case Retag:
		return "retag"
	case AddTag:
		return "add_tag"
	case RemoveTag:
		return "remove_tag"
	}
	return ""
}

// Label is the kind said in words.
func (k Kind) Label() string {
	switch k {
	case Retag:
		return "改标签"
	case AddTag:
		return "加标签"
	case RemoveTag:
		return "删标签"
	}
	return ""
}

// Status is where a task has got to.
type Status int

// The four places a task can be.
const (
	// Ready is a plan made and not yet run.
	Ready Status = iota
	// Running is a task being run. It is never written down as this:
	// one that was left halfway is written down as Paused, so that it
	// is not taken for a task that is still going.
	Running
	// Paused is a task that came to the batch limit, or was stopped.
	Paused
	// Done is a task with nothing left to do.
	Done
)

// Label is the status said in words.
func (s Status) Label() string {
	switch s {
	case Ready:
		return "待执行"
	case Running:
		return "执行中"
	case Paused:
		return "已暂停"
	case Done:
		return "已完成"
	}
	return ""
}

// ItemKind is where one note of a task has got to.
type ItemKind int

// The four places a note can be. Skipped and Failed carry a reason: a
// note quietly lost is a note lied about.
const (
	ItemPending ItemKind = iota
	ItemDone
	ItemSkipped
	ItemFailed
)

// ItemState is where one note has got to, and why, when there is a why.
type ItemState struct {
	Kind   ItemKind `json:"kind"`
	Reason string   `json:"reason,omitempty"`
}

// Finished is whether there is nothing more to do to this one. Taking a
// task up again goes by this.
func (s ItemState) Finished() bool { return s.Kind != ItemPending }

// Label is the state said in words, with the reason when there is one.
func (s ItemState) Label() string {
	switch s.Kind {
	case ItemPending:
		return "待处理"
	case ItemDone:
		return "已改"
	case ItemSkipped:
		return "跳过（" + s.Reason + "）"
	case ItemFailed:
		return "失败（" + s.Reason + "）"
	}
	return ""
}

// Edit is one line as it was and as it would be: only the lines that
// change are kept, for a plan is looked at, not read.
type Edit struct {
	Before string `json:"before"`
	After  string `json:"after"`
}

// TaskItem is one note of a task.
type TaskItem struct {
	Path    string    `json:"path"`
	Preview []Edit    `json:"preview"`
	State   ItemState `json:"state"`
}

// Task is a piece of long work over a vault.
type Task struct {
	ID   string `json:"id"`
	Kind Kind   `json:"kind"`
	// Target is the tag to change, add or take out, without its `#`.
	Target string `json:"target"`
	// NewTag is what Target becomes; only a Retag uses it.
	NewTag string     `json:"new_tag,omitempty"`
	Status Status     `json:"status"`
	Items  []TaskItem `json:"items"`
	// Cursor is where to take it up from: what stands before it has
	// been done already.
	Cursor    int   `json:"cursor"`
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

// New is a task planned and not yet written down.
func New(id string, kind Kind, target, newTag string) Task {
	now := time.Now().Unix()
	return Task{
		ID: id, Kind: kind, Target: target, NewTag: newTag,
		Status: Ready, CreatedAt: now, UpdatedAt: now,
	}
}

// Title is the task in one line, as a panel heads it.
func (t Task) Title() string {
	switch t.Kind {
	case Retag:
		// A new name not yet said is not shown as an empty `#`.
		if t.NewTag == "" {
			return fmt.Sprintf("%s：#%s → （待填）", t.Kind.Label(),
				t.Target)
		}
		return fmt.Sprintf("%s：#%s → #%s", t.Kind.Label(), t.Target,
			t.NewTag)
	case AddTag:
		if t.Target == "" {
			return fmt.Sprintf("%s：（待填）", t.Kind.Label())
		}
		return fmt.Sprintf("%s：#%s", t.Kind.Label(), t.Target)
	}
	return fmt.Sprintf("%s：#%s", t.Kind.Label(), t.Target)
}

// Pending is how many notes there are still to do.
func (t Task) Pending() int {
	out := 0
	for i := range t.Items {
		if !t.Items[i].State.Finished() {
			out++
		}
	}
	return out
}

// Stats is how many came out changed, skipped and failed.
func (t Task) Stats() (done, skipped, failed int) {
	for i := range t.Items {
		switch t.Items[i].State.Kind {
		case ItemDone:
			done++
		case ItemSkipped:
			skipped++
		case ItemFailed:
			failed++
		}
	}
	return
}

// NextBatch is which notes to do next: as many as size, from the cursor
// on, and none that are already done.
func (t Task) NextBatch(size int) []int {
	out := []int{}
	for i := t.Cursor; i < len(t.Items); i++ {
		if t.Items[i].State.Finished() {
			continue
		}
		out = append(out, i)
		if len(out) >= size {
			break
		}
	}
	return out
}

// Finished is whether there is nothing left to do.
func (t Task) Finished() bool { return t.Pending() == 0 }

// OutcomeKind is what came of rewriting one note.
type OutcomeKind int

// The three things that can come of it.
const (
	// Changed is the whole note as it would be.
	Changed OutcomeKind = iota
	// Unchanged is a note the tag does not stand in. An AddTag never
	// comes to this.
	Unchanged
	// Skipped is a note left alone, with the reason said.
	Skipped
)

// Outcome is what came of rewriting one note.
type Outcome struct {
	Kind    OutcomeKind
	Text    string
	Reason  string
	Preview []Edit
}

// orSkip is a note the tag does not stand in, told as one skipped: a
// plan does not hold notes it would do nothing to.
func (o Outcome) orSkip(reason string) Outcome {
	if o.Kind == Unchanged {
		return Outcome{Kind: Skipped, Reason: reason}
	}
	return o
}

// ApplyText is one note rewritten, as the task asks.
func ApplyText(kind Kind, text, target, newTag string) Outcome {
	switch kind {
	case Retag:
		return RetagText(text, target, newTag)
	case AddTag:
		return AddTagText(text, target)
	}
	return RemoveTagText(text, target)
}

// RetagText is `#target` changed into `#new`.
func RetagText(text, target, newTag string) Outcome {
	if target == "" || newTag == "" {
		return Outcome{Kind: Unchanged}
	}
	// A note that already has the new tag is left alone: changing it
	// would leave two of the same.
	if HasTag(text, newTag) {
		return Outcome{Kind: Skipped,
			Reason: "已含 #" + newTag}
	}
	out, preview, found := rewrite(text, target, &newTag)
	if !found {
		return Outcome{Kind: Unchanged}
	}
	return Outcome{Kind: Changed, Text: out, Preview: preview}
}

// AddTagText is a tag put on a note.
//
// A note with a front matter `tags:` gets it in the list; one without
// gets a line at the end. A note that has it already is skipped.
func AddTagText(text, tag string) Outcome {
	if tag == "" {
		return Outcome{Kind: Unchanged}
	}
	if HasTag(text, tag) {
		return Outcome{Kind: Skipped, Reason: "已含 #" + tag}
	}
	out, preview, _ := rewrite(text, tag, nil)
	return Outcome{Kind: Changed, Text: out, Preview: preview}
}

// RemoveTagText is a tag taken out of a note.
func RemoveTagText(text, tag string) Outcome {
	if tag == "" {
		return Outcome{Kind: Unchanged}
	}
	empty := ""
	out, preview, found := rewrite(text, tag, &empty)
	if !found {
		return Outcome{Kind: Unchanged}
	}
	return Outcome{Kind: Changed, Text: out, Preview: preview}
}

// HasTag is whether the tag stands in the note, as a token in the text
// or as an entry in the front matter's list.
func HasTag(text, tag string) bool {
	_, _, found := rewrite(text, tag, nil)
	return found
}

// rewrite reads a note once and does three things at once: it writes the
// note as it would be, gathers the lines that would change, and says
// whether the tag stands in it at all.
//
// replace is what `#target` becomes: nil looks without changing, a
// pointer to "" takes it out, and a pointer to a name changes it into
// that.
func rewrite(text, target string, replace *string) (string, []Edit, bool) {
	adding := replace == nil
	out := []string{}
	preview := []Edit{}
	found := false
	// Only an AddTag that found nothing needs to write into the front
	// matter or at the end.
	needAdd := adding
	inFront := false
	frontClosed := false
	inFence := false
	inList := false
	inserted := false
	// A line taken out at the top takes the blank line after it with
	// it.
	skipBlank := false

	lines := strings.Split(text, "\n")
	// A note that ends with a newline does not have a last empty line:
	// whether it ended with one is put back at the end.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for i, line := range lines {
		if skipBlank {
			skipBlank = false
			if strings.TrimSpace(line) == "" {
				continue
			}
		}

		// The front matter opens and closes at the top of the file,
		// and nowhere else.
		if !frontClosed && i == 0 && strings.TrimSpace(line) == "---" {
			inFront = true
			out = append(out, line)
			continue
		}
		if inFront && strings.TrimSpace(line) == "---" {
			// Not put in before the closing line: put it in now.
			if needAdd && !inserted {
				added := "tags: [" + target + "]"
				out = append(out, added)
				preview = append(preview, Edit{Before: "", After: added})
				inserted = true
			}
			inFront = false
			frontClosed = true
			inList = false
			out = append(out, line)
			continue
		}

		// A `#xxx` inside a code fence is not a tag: `#define` and the
		// like would be hurt.
		if isFence(line) {
			inFence = !inFence
			inList = false
			out = append(out, line)
			continue
		}
		if inFence {
			out = append(out, line)
			continue
		}

		if inFront {
			if value, ok := listValue(line); ok {
				// `tags: [a, b]`
				next, hit := editList(value, target, replace)
				found = found || hit
				if next == nil {
					// Taken out and nothing left: the line goes.
					preview = append(preview,
						Edit{Before: line, After: ""})
					needAdd = false
					continue
				}
				newLine := withValue(line, *next)
				if newLine != line {
					preview = append(preview,
						Edit{Before: line, After: newLine})
				}
				needAdd = false
				out = append(out, newLine)
				continue
			}
			if name, ok := listItem(line); ok {
				// `- a` under a `tags:` — it is found whether it
				// is being looked for only, or changed too.
				if inList && name == target {
					found = true
					if replace == nil {
						out = append(out, line)
						continue
					}
					if *replace == "" {
						preview = append(preview,
							Edit{Before: line, After: ""})
						continue
					}
					newLine := indentOf(line) + "- " + *replace
					preview = append(preview,
						Edit{Before: line, After: newLine})
					needAdd = false
					out = append(out, newLine)
					continue
				}
				out = append(out, line)
				continue
			}
			// A `tags:` with a list under it: only a `- x` after it
			// is an entry of it.
			inList = isTagsKey(line)
			out = append(out, line)
			continue
		}

		// The body: only a whole `#tag` token is changed.
		newLine, hit := rewriteInline(line, target, replace)
		found = found || hit
		if hit && replace != nil && *replace == "" &&
			strings.TrimSpace(newLine) == "" {
			// Nothing left of the line but space: the line goes,
			// and so does the blank one it leaves behind.
			preview = append(preview, Edit{Before: line, After: ""})
			nextBlank := true
			if i+1 < len(lines) {
				nextBlank = strings.TrimSpace(lines[i+1]) == ""
			}
			if nextBlank && len(out) > 0 &&
				strings.TrimSpace(out[len(out)-1]) == "" {
				out = out[:len(out)-1]
			}
			if len(out) == 0 {
				skipBlank = true
			}
			continue
		}
		if newLine != line {
			preview = append(preview, Edit{Before: line, After: newLine})
		}
		out = append(out, newLine)
	}

	// An AddTag with no front matter to hang on goes at the end.
	if needAdd && !inserted {
		tail := strings.Join(out, "\n")
		if tail != "" && !strings.HasSuffix(tail, "\n") {
			tail += "\n"
		}
		if tail != "" && !strings.HasSuffix(tail, "\n\n") {
			tail += "\n"
		}
		preview = append(preview, Edit{Before: "", After: "#" + target})
		tail += "#" + target + "\n"
		return tail, preview, found
	}

	// Splitting on "\n" loses whether the note ended with one.
	joined := strings.Join(out, "\n")
	if strings.HasSuffix(text, "\n") && joined != "" {
		joined += "\n"
	}
	return joined, preview, found
}

// isFence is whether the line is a code fence.
func isFence(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	return strings.HasPrefix(trimmed, "```") ||
		strings.HasPrefix(trimmed, "~~~")
}

// indentOf is the space the line starts with.
func indentOf(line string) string {
	out := []rune{}
	for _, c := range line {
		if c != ' ' && c != '\t' {
			break
		}
		out = append(out, c)
	}
	return string(out)
}

// isTagsKey is whether the line is a `tags:` key whose list comes under
// it.
func isTagsKey(line string) bool {
	switch strings.TrimSpace(line) {
	case "tags:", "tags: []", "tags: [ ]":
		return true
	}
	return false
}

// listValue is the `[a, b]` of a `tags: [a, b]`, brackets and all.
//
// A `tags: []` counts: it is a list that is legally empty, and a tag
// being added goes into it.
func listValue(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	rest, ok := strings.CutPrefix(trimmed, "tags:")
	if !ok {
		return "", false
	}
	rest = strings.TrimSpace(rest)
	if !strings.HasPrefix(rest, "[") || !strings.HasSuffix(rest, "]") {
		return "", false
	}
	return rest, true
}

// withValue is the `tags:` line with another value, its indent kept.
func withValue(line, value string) string {
	return indentOf(line) + "tags: " + value
}

// listItem is the name of a front matter list entry `- x`, its quotes
// taken off.
func listItem(line string) (string, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(line), "-")
	if !ok {
		return "", false
	}
	return unquote(rest), true
}

// unquote is a name with its quotes taken off.
func unquote(text string) string {
	text = strings.TrimSpace(text)
	if len(text) >= 2 {
		if strings.HasPrefix(text, `"`) && strings.HasSuffix(text, `"`) {
			return text[1 : len(text)-1]
		}
		if strings.HasPrefix(text, "'") && strings.HasSuffix(text, "'") {
			return text[1 : len(text)-1]
		}
	}
	return text
}

// editList is one entry of a `[a, b]` changed.
//
// It gives back the new value, and whether the tag was in it at all;
// no value at all means the list came out empty and the line goes.
func editList(value, target string, replace *string) (*string, bool) {
	inner := value[1 : len(value)-1]
	out := []string{}
	hit := false
	for _, entry := range strings.Split(inner, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if unquote(entry) == target {
			hit = true
			switch {
			case replace != nil && *replace == "":
				continue
			case replace != nil:
				out = append(out, quoteLike(entry, *replace))
			default:
				out = append(out, entry)
			}
			continue
		}
		out = append(out, entry)
	}
	// An AddTag puts it in when it is not there.
	if replace == nil && !hit {
		out = append(out, target)
	}
	if len(out) == 0 {
		return nil, hit
	}
	joined := "[" + strings.Join(out, ", ") + "]"
	return &joined, hit
}

// quoteLike is a name in the same quotes the entry it replaces had.
func quoteLike(entry, name string) string {
	if strings.HasPrefix(entry, `"`) {
		return `"` + name + `"`
	}
	if strings.HasPrefix(entry, "'") {
		return "'" + name + "'"
	}
	return name
}

// rewriteInline is one line's whole `#tag` tokens changed.
func rewriteInline(line, target string, replace *string) (string, bool) {
	chars := []rune(line)
	out := []rune{}
	found := false
	atStart := true
	i := 0
	for i < len(chars) {
		c := chars[i]
		afterSpace := atStart || (i > 0 && isSpace(chars[i-1]))
		if c == '#' && afterSpace {
			end := i + 1
			for end < len(chars) && !isSpace(chars[end]) &&
				chars[end] != '#' {
				end++
			}
			token := string(chars[i+1 : end])
			if token != "" && token == target {
				found = true
				switch {
				case replace != nil && *replace == "":
				case replace != nil:
					out = append(out, '#')
					out = append(out, []rune(*replace)...)
				default:
					out = append(out, '#')
					out = append(out, []rune(token)...)
				}
			} else {
				out = append(out, '#')
				out = append(out, []rune(token)...)
			}
			i = end
			atStart = false
			continue
		}
		out = append(out, c)
		if !isSpace(c) {
			atStart = false
		}
		i++
	}
	return string(out), found
}

// isSpace is whether the character parts words.
func isSpace(c rune) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// Plan is a task made of the notes given. Each is read once, what would
// change in it is worked out, and one that would not change is not held
// at all: it is skipped, with the reason said.
//
// It writes nothing. A plan is looked at before it is believed.
func Plan(
	id string, kind Kind, target, newTag string, paths []string,
	read func(path string) (string, error),
) Task {
	one := New(id, kind, target, newTag)
	for _, path := range paths {
		text, err := read(path)
		if err != nil {
			one.Items = append(one.Items, TaskItem{
				Path: path,
				State: ItemState{Kind: ItemFailed,
					Reason: "读不到：" + err.Error()},
			})
			continue
		}
		got := ApplyText(kind, text, target, newTag).orSkip(
			"没有 #" + target)
		item := TaskItem{Path: path}
		switch got.Kind {
		case Changed:
			item.Preview = got.Preview
		case Skipped:
			item.State = ItemState{Kind: ItemSkipped, Reason: got.Reason}
		default:
			item.State = ItemState{Kind: ItemSkipped,
				Reason: "没有 #" + target}
		}
		one.Items = append(one.Items, item)
	}
	return one
}

// Run is the next batch of a task done: each note of it is read again —
// the disk is the truth of a note, and a plan may be old — rewritten,
// and written back, and what came of each is kept on the task.
//
// It gives back how many it did. A task that was stopped halfway can be
// run again: what is already finished is not done over.
func Run(
	t *Task, size int, read func(path string) (string, error),
	write func(path, text string) error,
) int {
	if size <= 0 {
		size = BatchSize
	}
	batch := t.NextBatch(size)
	for _, at := range batch {
		item := &t.Items[at]
		if item.State.Finished() {
			continue
		}
		text, err := read(item.Path)
		if err != nil {
			item.State = ItemState{Kind: ItemFailed,
				Reason: "读不到：" + err.Error()}
			continue
		}
		got := ApplyText(t.Kind, text, t.Target, t.NewTag)
		switch got.Kind {
		case Changed:
			if err := write(item.Path, got.Text); err != nil {
				item.State = ItemState{Kind: ItemFailed,
					Reason: "写不了：" + err.Error()}
				continue
			}
			item.Preview = got.Preview
			item.State = ItemState{Kind: ItemDone}
		case Skipped:
			item.State = ItemState{Kind: ItemSkipped, Reason: got.Reason}
		default:
			item.State = ItemState{Kind: ItemSkipped,
				Reason: "没有 #" + t.Target}
		}
	}
	// The cursor moves to just past the last one that was done, and no
	// further: what stands after it is for the next run.
	for _, at := range batch {
		if at+1 > t.Cursor {
			t.Cursor = at + 1
		}
	}
	t.Status = Paused
	if t.Finished() {
		t.Status = Done
	} else if len(batch) == 0 {
		t.Status = Ready
	}
	t.UpdatedAt = time.Now().Unix()
	return len(batch)
}

// Progress is the task as a person reads it: what it set out to do, and
// what came of each note. It is written next to the task so that it can
// be opened and read when something has gone wrong.
func Progress(t Task) string {
	out := strings.Builder{}
	out.WriteString("# " + t.Title() + "\n\n")
	done, skipped, failed := t.Stats()
	out.WriteString(fmt.Sprintf("状态：%s · 已改 %d · 跳过 %d · "+
		"失败 %d · 待处理 %d\n\n", t.Status.Label(), done, skipped,
		failed, t.Pending()))
	for i := range t.Items {
		item := t.Items[i]
		out.WriteString(fmt.Sprintf("- %s：%s\n", item.Path,
			item.State.Label()))
		for _, edit := range item.Preview {
			out.WriteString(fmt.Sprintf("    - `%s` → `%s`\n",
				edit.Before, edit.After))
		}
	}
	return out.String()
}
