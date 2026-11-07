package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/helightxu/xinkgo/internal/index"
	"github.com/helightxu/xinkgo/internal/search"
	"github.com/helightxu/xinkgo/internal/skill"
)

// The tools a model may ask for. Every one of them reads: there is no
// way in this layer to write, and that is on purpose.
const (
	// ToolReadNote is a note's text.
	ToolReadNote = "read_note"
	// ToolSearchNotes is the lines the words asked for stand in.
	ToolSearchNotes = "search_notes"
	// ToolListTags is every tag in the vault, and how many notes use it.
	ToolListTags = "list_tags"
	// ToolListBacklinks is where a note is pointed at from.
	ToolListBacklinks = "list_backlinks"
	// ToolOutline is the headings of a note.
	ToolOutline = "outline"
	// ToolListNotes is the notes of the vault.
	ToolListNotes = "list_notes"
	// ToolReadSkillResource is a file a skill brings with it.
	ToolReadSkillResource = "read_skill_resource"
	// ToolAskUser is the model asking the user to decide. It is the one
	// tool that does not read the vault: the host shows its card and
	// ends the turn instead of executing it.
	ToolAskUser = "ask_user"
)

// Tools are the tools there are, in the order they are offered.
var Tools = []string{
	ToolReadNote, ToolSearchNotes, ToolListTags, ToolListBacklinks,
	ToolOutline, ToolListNotes, ToolReadSkillResource, ToolAskUser,
}

// How many of a thing a tool gives back when it was not told how many.
const (
	defaultSearchLimit = 10
	defaultListLimit   = 30
	// maxLimit is the most a tool gives back, however many were asked
	// for: what comes back goes into what the model is told, and that
	// is not endless.
	maxLimit = 50
)

// Host is what a tool can see of the vault.
//
// All of it reads. A tool that would write is not one of these, and
// this layer gives no way to write one.
type Host interface {
	// ReadNote is a note's text, and whether it could be read: a note
	// that is gone, or that is not text, is one that could not.
	ReadNote(path string) (string, bool)
	// Search is the lines the words asked for stand in.
	Search(query string, limit int) []search.Hit
	// Tags is every tag in the vault, and how many notes use it.
	Tags() []index.TagCount
	// Backlinks is where a note is pointed at from.
	Backlinks(path string) []index.LinkRef
	// Outline is the headings of a note.
	Outline(path string) []string
	// ListNotes is the notes of the vault, with what they are called.
	ListNotes(limit int) []NoteRef
	// Root is the vault's folder, which paths are shown against: a
	// path shown from the root is a shorter thing to say.
	Root() string
	// Skill is the skill of that name, when there is one.
	Skill(name string) (*skill.Skill, bool)
}

// NoteRef is a note a tool names: where it is, and what it is called.
type NoteRef struct {
	Path  string
	Title string
}

// Result is what a tool came back with, to be handed back to the model.
type Result struct {
	Name   string
	Output string
}

// Specs is what a model is told the tools are, as JSON: one shape, for
// both the local server and the clouds speak it.
//
// With skills turned off, the skill resource is left out: a tool the
// model has nothing to use on is only another chance for it to guess.
func Specs(skills bool) []byte {
	specs := make([]map[string]any, 0, len(Tools))
	for _, name := range Tools {
		if name == ToolReadSkillResource && !skills {
			continue
		}
		specs = append(specs, schemaOf(name))
	}
	data, err := json.Marshal(specs)
	if err != nil {
		return nil
	}
	return data
}

// schemaOf is what a model is told one tool is.
func schemaOf(name string) map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        name,
			"description": descriptionOf(name),
			"parameters":  parametersOf(name),
		},
	}
}

// descriptionOf is what a tool is for, as the model is told it.
func descriptionOf(name string) string {
	switch name {
	case ToolReadNote:
		return "读取指定笔记的正文（给出完整路径）"
	case ToolSearchNotes:
		return "按关键词全文检索笔记，返回命中行"
	case ToolListTags:
		return "列出库内标签及其使用次数"
	case ToolListBacklinks:
		return "列出引用了指定笔记的位置（反向链接）"
	case ToolOutline:
		return "列出指定笔记的标题大纲"
	case ToolListNotes:
		return "列出库内笔记（路径与标题）"
	case ToolReadSkillResource:
		return "读取技能自带的资源文件（如 references/），只能读该技能" +
			"目录内"
	case ToolAskUser:
		return "需要用户做决定时提问：给出问题与选项，等待用户回答后再继续。" +
			"一次只问一个问题。"
	}
	return ""
}

// parametersOf is what a tool is given, as the model is told it.
func parametersOf(name string) map[string]any {
	path := func() map[string]any {
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "笔记路径（相对库根或绝对路径）",
				},
			},
			"required": []string{"path"},
		}
	}
	limit := func(what string) map[string]any {
		return map[string]any{
			"type":        "integer",
			"description": what,
		}
	}
	switch name {
	case ToolReadNote, ToolOutline, ToolListBacklinks:
		return path()
	case ToolSearchNotes:
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "检索关键词",
				},
				"limit": limit("最多返回几条，默认 10"),
			},
			"required": []string{"query"},
		}
	case ToolListTags, ToolListNotes:
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"limit": limit("最多返回几条，默认 30"),
			},
		}
	case ToolReadSkillResource:
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"skill": map[string]any{
					"type":        "string",
					"description": "技能名（SKILL.md 里的 name）",
				},
				"path": map[string]any{
					"type": "string",
					"description": "技能目录内的相对路径，如 " +
						"references/a.md",
				},
			},
			"required": []string{"skill", "path"},
		}
	case ToolAskUser:
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"question": map[string]any{
					"type":        "string",
					"description": "要问用户的问题",
				},
				"header": map[string]any{
					"type":        "string",
					"description": "卡片标题",
				},
				"options": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "可选项的短标签，最多 8 个",
				},
				"allow_other": map[string]any{
					"type":        "boolean",
					"description": "是否允许用户自己填写",
				},
				"secret": map[string]any{
					"type":        "boolean",
					"description": "回答是否应脱敏、不写入会话",
				},
			},
			"required": []string{"question"},
		}
	}
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

// Execute runs a tool a model asked for.
//
// It does not fail. A tool the app has never heard of, arguments that
// make no sense, a note that is not there: none of these is a failure
// of the app, and all of them are things the model needs to be told so
// that it may ask better. So every one of them comes back as words.
func Execute(call ToolCall, host Host, maxChars int) Result {
	output := ""
	if !known(call.Name) {
		output = fmt.Sprintf("没有这个工具：%s。可用的工具：%s",
			call.Name, strings.Join(Tools, ", "))
	} else {
		output = run(call.Name, givenOf(call.Arguments), host)
	}
	return Result{Name: call.Name, Output: clip(output, maxChars)}
}

// known is whether the app has a tool of that name.
func known(name string) bool {
	for _, one := range Tools {
		if one == name {
			return true
		}
	}
	return false
}

// givenOf is what a model gave a tool. Arguments that are not JSON, or
// that are no object at all, are taken for none given: the tool says
// what it was not told, rather than the app complaining of parsing.
func givenOf(text string) map[string]any {
	if strings.TrimSpace(text) == "" {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return map[string]any{}
	}
	return out
}

// run is one tool, given what it was told.
func run(name string, args map[string]any, host Host) string {
	word := func(key string) string {
		value, ok := args[key].(string)
		if !ok {
			return ""
		}
		return value
	}
	count := func(fallback int) int {
		switch value := args["limit"].(type) {
		case float64:
			n := int(value)
			if n < 1 {
				return fallback
			}
			if n > maxLimit {
				return maxLimit
			}
			return n
		case string:
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 {
				return fallback
			}
			if n > maxLimit {
				return maxLimit
			}
			return n
		}
		return fallback
	}

	switch name {
	case ToolReadNote:
		path := word("path")
		if path == "" {
			return "缺少参数 path"
		}
		text, ok := host.ReadNote(path)
		if !ok {
			return "读不到这篇笔记：" + path
		}
		return text
	case ToolSearchNotes:
		query := word("query")
		if query == "" {
			return "缺少参数 query"
		}
		hits := host.Search(query, count(defaultSearchLimit))
		if len(hits) == 0 {
			return "没有命中“" + query + "”"
		}
		lines := make([]string, 0, len(hits))
		for _, hit := range hits {
			lines = append(lines, fmt.Sprintf("%s:%d: %s",
				show(host, hit.Path), hit.Line+1,
				strings.TrimSpace(hit.Text)))
		}
		return strings.Join(lines, "\n")
	case ToolListTags:
		tags := host.Tags()
		if len(tags) == 0 {
			return "库里还没有标签"
		}
		// The most used first: that is the order they are shown in.
		sorted := make([]index.TagCount, len(tags))
		copy(sorted, tags)
		sort.SliceStable(sorted, func(i, j int) bool {
			return sorted[i].Count > sorted[j].Count
		})
		at := count(defaultListLimit)
		if at > len(sorted) {
			at = len(sorted)
		}
		lines := make([]string, 0, at)
		for _, tag := range sorted[:at] {
			lines = append(lines, fmt.Sprintf("#%s (%d)",
				tag.Name, tag.Count))
		}
		return strings.Join(lines, "\n")
	case ToolListBacklinks:
		path := word("path")
		if path == "" {
			return "缺少参数 path"
		}
		links := host.Backlinks(path)
		if len(links) == 0 {
			return "没有笔记引用“" + path + "”"
		}
		lines := make([]string, 0, len(links))
		for _, link := range links {
			lines = append(lines, fmt.Sprintf("%s:%d: [[%s]]",
				show(host, link.From), link.Line+1, link.Target))
		}
		return strings.Join(lines, "\n")
	case ToolOutline:
		path := word("path")
		if path == "" {
			return "缺少参数 path"
		}
		headings := host.Outline(path)
		if len(headings) == 0 {
			return "“" + path + "”没有标题，或读不到这篇笔记"
		}
		return strings.Join(headings, "\n")
	case ToolListNotes:
		notes := host.ListNotes(count(defaultListLimit))
		if len(notes) == 0 {
			return "库里还没有笔记"
		}
		lines := make([]string, 0, len(notes))
		for _, note := range notes {
			lines = append(lines, show(host, note.Path)+" — "+
				note.Title)
		}
		return strings.Join(lines, "\n")
	case ToolReadSkillResource:
		name := word("skill")
		rel := word("path")
		if name == "" || rel == "" {
			return "缺少参数 skill 或 path"
		}
		one, ok := host.Skill(name)
		if !ok {
			return "没有这个技能：" + name
		}
		// A skill is not to reach out of its own folder: what it
		// brings with it is all it may be given.
		path, ok := skill.Resource(*one, rel)
		if !ok {
			return "不允许的资源路径：" + rel +
				"（只能读技能目录内的文件）"
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "读不到技能资源：" + rel
		}
		return string(data)
	case ToolAskUser:
		// The loop intercepts this call before Execute ever runs; the
		// words here are only a guard for a direct caller.
		return "已把问题交给用户，等待回答。"
	}
	return "没有这个工具：" + name
}

// show is a path as the vault knows it: from the root, which is the
// shorter thing to say. One that is not under the root at all is said
// whole.
func show(host Host, path string) string {
	root := host.Root()
	if root == "" {
		return path
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return path
	}
	return rel
}

// clip is the first of a tool's answer, as many characters as count.
// What a tool found may be longer than what a model can be told: it is
// cut short, and told that it was.
func clip(text string, count int) string {
	if count <= 0 {
		return text
	}
	runes := []rune(text)
	if len(runes) <= count {
		return text
	}
	return string(runes[:count]) + fmt.Sprintf(
		"\n…（已截断：共 %d 字符，只回传前 %d 个）", len(runes), count)
}
