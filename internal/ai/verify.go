package ai

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/helightxu/xinkgo/internal/index"
	"github.com/helightxu/xinkgo/internal/markdown"
)

// IssueKind is what kind of thing is wrong with a note.
type IssueKind int

// The three kinds of thing that can be wrong.
const (
	// IssueWikiLink is a link in double brackets that points at
	// nothing, or at more than one note.
	IssueWikiLink IssueKind = iota
	// IssueFrontmatter is a front matter that does not read.
	IssueFrontmatter
	// IssueTag is a `tags:` that is not a list of names.
	IssueTag
)

// Name is the kind said to the model, which is the one that wrote it.
func (k IssueKind) Name() string {
	switch k {
	case IssueWikiLink:
		return "双链"
	case IssueFrontmatter:
		return "frontmatter"
	case IssueTag:
		return "标签"
	}
	return ""
}

// Issue is one thing wrong with a note.
type Issue struct {
	Kind IssueKind
	// Line is the line, as a person counts them, from one. Zero is
	// said when it is not a line's fault but the whole note's.
	Line int
	// Detail is what is wrong.
	Detail string
}

// Feedback is the issue in one line, as the model is told of it so that
// it may write it again and write it right.
func (i Issue) Feedback() string {
	if i.Line == 0 {
		return "[" + i.Kind.Name() + "] " + i.Detail
	}
	return fmt.Sprintf("[%s] 第 %d 行：%s", i.Kind.Name(), i.Line,
		i.Detail)
}

// LinkIndex is what Verify needs of the vault's index: where a link in
// double brackets points.
type LinkIndex interface {
	Resolve(target string) index.Resolution
}

// Verify is a note looked at after a model has written it: is it still a
// note that holds together?
//
// Three things are asked: that every link in double brackets can be
// resolved, that the front matter reads, and that its `tags` is a list of
// names.
//
// What it finds is only said, and never blocks: a vault may well not have
// the note a draft points at yet, and a link to a note still to be
// written is no reason to throw the draft away.
func Verify(text string, idx LinkIndex) []Issue {
	out := []Issue{}
	facts := markdown.ParseFacts(text)

	// Without a vault there is nothing to resolve a link against.
	if idx != nil {
		for _, link := range facts.WikiLinks {
			switch got := idx.Resolve(link.Target); got.Kind {
			case index.Hit:
			case index.Missing:
				out = append(out, Issue{
					Kind: IssueWikiLink,
					Line: link.Line + 1,
					Detail: fmt.Sprintf("[[%s]] 指向的笔记不存在",
						link.Target),
				})
			case index.Ambiguous:
				names := make([]string, 0, 3)
				for i, path := range got.Candidates {
					if i >= 3 {
						break
					}
					names = append(names, path)
				}
				out = append(out, Issue{
					Kind: IssueWikiLink,
					Line: link.Line + 1,
					Detail: fmt.Sprintf(
						"[[%s]] 有 %d 篇同名笔记，歧义（%s）",
						link.Target, len(got.Candidates),
						strings.Join(names, "、")),
				})
			}
		}
	}

	body, line, state := frontMatter(text)
	switch state {
	case frontUnclosed:
		// A `---` that is never closed is what a model writes when it
		// has lost its place.
		out = append(out, Issue{
			Kind:   IssueFrontmatter,
			Line:   line,
			Detail: "frontmatter 以 --- 开始但没有闭合",
		})
	case frontThere:
		if ok, what := readsAsData(body); !ok {
			out = append(out, Issue{
				Kind:   IssueFrontmatter,
				Line:   line,
				Detail: "frontmatter 不是合法 YAML：" + what,
			})
		} else {
			out = append(out, tagIssues(body, line)...)
		}
	}
	return out
}

// frontState is what a note's front matter is.
type frontState int

const (
	// frontNone is a note with no front matter, which is a note that
	// is perfectly well formed.
	frontNone frontState = iota
	// frontThere is a front matter that opens and closes.
	frontThere
	// frontUnclosed is a `---` that never closes.
	frontUnclosed
)

// frontMatter is the front matter's body and the line it starts on, as a
// person counts them, from one.
func frontMatter(text string) (body string, line int, state frontState) {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", 0, frontNone
	}
	body = ""
	for i, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			return body, 2, frontThere
		}
		body += line
		if i < len(lines)-2 {
			body += "\n"
		}
		// Fifty lines on without a close, and it is not a front
		// matter: it is a rule someone drew in the text.
		if i+1 > 50 {
			return "", 2, frontUnclosed
		}
	}
	return "", 2, frontUnclosed
}

// readsAsData is whether the front matter's body is data the app can
// read, and what is wrong with it when it is not.
//
// The app has no YAML library — it cannot be fetched — so what is asked
// is only that it is not plainly broken: a line that is neither a key,
// nor a list item, nor a blank one.
func readsAsData(body string) (bool, string) {
	for i, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "-") {
			continue
		}
		if strings.Contains(trimmed, ":") {
			continue
		}
		return false, fmt.Sprintf("第 %d 行读不懂：%s", i+2, trimmed)
	}
	return true, ""
}

// tagIssues is what is wrong with the `tags` of a front matter: it must
// be a list of names, or one name on its own.
//
// The app has no YAML library and cannot fetch one, so the front matter
// is read the way a person reads it — a `tags:` line, and either a list
// on that line or one on the lines under it — rather than parsed into a
// tree of values nobody asked for.
func tagIssues(body string, line int) []Issue {
	lines := strings.Split(body, "\n")
	for i, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if !strings.HasPrefix(trimmed, "tags:") {
			continue
		}
		rest := strings.TrimSpace(
			strings.TrimPrefix(trimmed, "tags:"))
		// `tags:` with a list under it: every entry is a line of
		// text, and a line of text is a name.
		if rest == "" {
			return nil
		}
		// `tags: [a, b]`
		if strings.HasPrefix(rest, "[") &&
			strings.HasSuffix(rest, "]") {
			bad := 0
			for _, entry := range strings.Split(
				rest[1:len(rest)-1], ",") {
				entry = unquote2(strings.TrimSpace(entry))
				if entry == "" {
					continue
				}
				if !isName(entry) {
					bad++
				}
			}
			if bad == 0 {
				return nil
			}
			return []Issue{{
				Kind: IssueTag,
				Line: line + i,
				Detail: fmt.Sprintf(
					"tags 里有 %d 项不是标签名", bad),
			}}
		}
		// One tag, and no list around it, is a thing people write.
		if isName(unquote2(rest)) {
			return nil
		}
		return []Issue{{
			Kind:   IssueTag,
			Line:   line + i,
			Detail: "tags 应是标签列表，现在是" + kindOfWord(rest),
		}}
	}
	return nil
}

// isName is whether the value is a name, and not a number or a truth
// value that was put where a name should be.
func isName(text string) bool {
	if text == "" {
		return false
	}
	switch strings.ToLower(text) {
	case "true", "false", "null", "yes", "no":
		return false
	}
	if _, err := strconv.ParseFloat(text, 64); err == nil {
		return false
	}
	return true
}

// kindOfWord is what kind of value a word is, said plainly.
func kindOfWord(text string) string {
	switch strings.ToLower(text) {
	case "true", "false", "yes", "no":
		return "布尔"
	case "null":
		return "空"
	}
	if _, err := strconv.ParseFloat(text, 64); err == nil {
		return "数字"
	}
	return "字符串"
}

// unquote2 is a value with its quotes taken off.
func unquote2(text string) string {
	text = strings.TrimSpace(text)
	runes := []rune(text)
	if len(runes) >= 2 {
		if runes[0] == '"' && runes[len(runes)-1] == '"' {
			return string(runes[1 : len(runes)-1])
		}
		if runes[0] == '\'' && runes[len(runes)-1] == '\'' {
			return string(runes[1 : len(runes)-1])
		}
	}
	return text
}
