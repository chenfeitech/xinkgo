// Package keymap is the table of the actions a key can be bound to, and
// how the user's own bindings are read on top of it.
//
// It is data and string parsing, and nothing else: what an action does is
// decided where it is wired, so that the table can be tested without an
// application running.
package keymap

import (
	"log/slog"
	"strings"
	"unicode"
)

// Binding is one action a key may be bound to. Action is the name the
// configuration's keymap overrides by; ID is the name the settings show
// it under.
type Binding struct {
	Action string
	ID     string
	// Label and Desc are what the settings window says.
	Label string
	Desc  string
	// Default is the key of a first start; empty means the action has
	// none until the user gives it one.
	Default string
	// Context is where the key works, such as the file tree; empty
	// means it works everywhere.
	Context string
}

// Bindings is the table of bindable actions, in the order xink has them:
// when two ask for one key, the earlier one keeps it.
var Bindings = []Binding{
	{"OpenFolder", "editor.keymap.OpenFolder", "打开笔记库",
		"选一个文件夹作为笔记库", "cmd-o", ""},
	{"NewNote", "editor.keymap.NewNote", "新建笔记",
		"在当前目录下新建笔记", "cmd-n", ""},
	{"NewFolder", "editor.keymap.NewFolder", "新建文件夹",
		"在当前目录下新建文件夹", "", ""},
	{"DeleteItem", "editor.keymap.DeleteItem", "删除选中项",
		"删除文件树里选中的项", "", ""},
	{"Save", "editor.keymap.Save", "保存当前笔记",
		"立即把当前笔记写盘", "cmd-s", ""},
	{"ClosePanel", "editor.keymap.ClosePanel", "关闭当前标签",
		"关闭当前笔记标签页", "ctrl-w", ""},
	{"RenameItem", "editor.keymap.RenameItem", "重命名选中项",
		"重命名文件树里的选中项", "f2", "FilesPanel"},
	{"OpenSettings", "editor.keymap.OpenSettings", "打开设置",
		"打开设置窗口", "cmd-,", ""},
	{"FormatBold", "editor.keymap.FormatBold", "加粗",
		"包裹选中文本为粗体", "cmd-b", ""},
	{"FormatItalic", "editor.keymap.FormatItalic", "斜体",
		"包裹选中文本为斜体", "cmd-i", ""},
	{"FormatStrikethrough", "editor.keymap.FormatStrikethrough", "删除线",
		"包裹选中文本为删除线", "cmd-shift-x", ""},
	{"FormatInlineCode", "editor.keymap.FormatInlineCode", "行内代码",
		"包裹选中文本为行内代码", "cmd-e", ""},
	{"FormatLink", "editor.keymap.FormatLink", "链接",
		"把选中文本变成链接", "cmd-k", ""},
	{"ToggleTheme", "editor.keymap.ToggleTheme", "切换明暗主题",
		"在浅色与深色之间切换", "cmd-shift-l", ""},
	{"OpenCommandPalette", "editor.keymap.OpenCommandPalette", "命令面板",
		"按文件名搜索并打开笔记", "cmd-p", ""},
	{"OpenSearch", "editor.keymap.OpenSearch", "全文搜索",
		"按内容检索整个库", "cmd-shift-f", ""},
	{"ToggleAiChat", "editor.keymap.ToggleAiChat", "AI 对话条",
		"打开或关闭 AI 对话条", "cmd-shift-a", ""},
}

// Resolved is one action and the key that finally does it.
type Resolved struct {
	Action    string
	Keystroke string
	Context   string
}

// modifiers are the keys that only change another key.
var modifiers = map[string]bool{
	"cmd": true, "ctrl": true, "alt": true, "shift": true,
}

// namedKeys are the keys that are written as a word.
var namedKeys = map[string]bool{
	"space": true, "enter": true, "return": true, "tab": true,
	"esc": true, "escape": true, "backspace": true, "delete": true,
	"del": true, "ins": true, "up": true, "down": true,
	"left": true, "right": true, "home": true, "end": true,
	"pageup": true, "pagedown": true,
}

// Normalize is what a key written by hand becomes: trimmed, its spaces
// squeezed into one, and lower-cased, so that "CMD-S" is "cmd-s".
func Normalize(raw string) string {
	parts := strings.Fields(raw)
	for i, p := range parts {
		parts[i] = strings.ToLower(p)
	}
	return strings.Join(parts, " ")
}

// IsValid reports whether the key can be bound: one or more strokes, each
// of modifiers and a key that is not one of them. "cmd-" has no key, and
// no key press could ever match it.
func IsValid(raw string) bool {
	strokes := strings.Fields(raw)
	if len(strokes) == 0 {
		return false
	}
	for _, stroke := range strokes {
		if !validStroke(stroke) {
			return false
		}
	}
	return true
}

// validStroke reports whether one stroke, such as "cmd-shift-x", is one a
// key press could match.
func validStroke(stroke string) bool {
	parts := strings.Split(stroke, "-")
	key := parts[len(parts)-1]
	if key == "" || modifiers[key] {
		return false
	}
	for _, m := range parts[:len(parts)-1] {
		if !modifiers[m] {
			return false
		}
	}
	return validKey(key)
}

// validKey reports whether the last part of a stroke is a key: a word
// such as "f2" or "space", or one printed character.
func validKey(key string) bool {
	if namedKeys[key] || functionKey(key) {
		return true
	}
	runes := []rune(key)
	if len(runes) != 1 {
		return false
	}
	return !unicode.IsSpace(runes[0])
}

// functionKey reports whether key is f1 to f12.
func functionKey(key string) bool {
	if len(key) < 2 || key[0] != 'f' {
		return false
	}
	n := 0
	for _, r := range key[1:] {
		if r < '0' || r > '9' {
			return false
		}
		n = n*10 + int(r-'0')
	}
	return n >= 1 && n <= 12
}

// OverrideOf is what the user gave the action, normalized, or empty when
// they gave nothing.
func OverrideOf(overrides map[string]string, action string) string {
	return Normalize(overrides[action])
}

// Resolve turns the user's overrides into the keys that are bound: an
// override that is empty or cannot be bound falls back to the default, and
// a key two actions ask for stays with the earlier one.
func Resolve(overrides map[string]string) []Resolved {
	used := map[string]bool{}
	out := make([]Resolved, 0, len(Bindings))
	for _, b := range Bindings {
		preferred := b.Default
		if candidate := OverrideOf(overrides, b.Action); candidate != "" &&
			IsValid(candidate) {
			preferred = candidate
		}
		// Taken by an earlier action: fall back, so that a key the user
		// moved does not leave them with nothing at all.
		keystroke := preferred
		if used[keystroke] {
			keystroke = b.Default
		}
		if keystroke == "" || used[keystroke] {
			if keystroke != "" {
				slog.Warn("two actions wanted one key, the earlier "+
					"one keeps it", "key", keystroke, "action", b.Action)
			}
			continue
		}
		used[keystroke] = true
		out = append(out, Resolved{
			Action:    b.Action,
			Keystroke: keystroke,
			Context:   b.Context,
		})
	}
	return out
}

// Sanitize drops what the configuration cannot use: unknown actions, and
// keys that are empty or cannot be bound, which is the same as asking for
// the default. Nothing is kept that the app would not honour.
func Sanitize(overrides map[string]string) {
	for action, raw := range overrides {
		if !IsKnown(action) {
			delete(overrides, action)
			continue
		}
		key := Normalize(raw)
		if !IsValid(key) {
			delete(overrides, action)
			continue
		}
		overrides[action] = key
	}
}

// Accelerator is the key as a menu item takes it, such as "CmdOrCtrl+O",
// or "" for one a menu cannot show: a chord of two strokes, or nothing a
// key press could match.
func Accelerator(keystroke string) string {
	strokes := strings.Fields(keystroke)
	if len(strokes) != 1 || !validStroke(strokes[0]) {
		return ""
	}
	parts := strings.Split(strokes[0], "-")
	var out strings.Builder
	for _, m := range parts[:len(parts)-1] {
		switch m {
		case "cmd":
			out.WriteString("CmdOrCtrl+")
		case "ctrl":
			out.WriteString("Ctrl+")
		case "alt":
			out.WriteString("Alt+")
		case "shift":
			out.WriteString("Shift+")
		}
	}
	key := parts[len(parts)-1]
	if len(key) == 1 {
		key = strings.ToUpper(key)
	} else if _, ok := namedKeys[key]; !ok {
		// f1 to f12: the menu writes them as they are said.
		key = strings.ToUpper(key)
	}
	out.WriteString(key)
	return out.String()
}

// IsKnown reports whether the action is one of the table's.
func IsKnown(action string) bool {
	for _, b := range Bindings {
		if b.Action == action {
			return true
		}
	}
	return false
}
