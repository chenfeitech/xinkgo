// Package highlight is the three presets of colors code is painted in.
//
// The editor and the preview are given the same theme, so that a piece of
// code is one color wherever the user reads it, and the preset follows
// the appearance: each of them has a light and a dark set of colors, as
// xink's have.
package highlight

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// The presets, as the configuration names them.
const (
	Xink      = "xink"
	Ocean     = "ocean"
	Solarized = "solarized"
)

// Presets are the three, in the order they are shown.
var Presets = []string{Xink, Ocean, Solarized}

// palette is the colors of one preset in one appearance: the ones xink
// paints the kinds of tokens in, taken from its own theme files.
type palette struct {
	keyword     string
	function    string
	typeName    string
	str         string
	escape      string
	number      string
	constant    string
	attribute   string
	property    string
	variable    string
	comment     string
	operator    string
	punctuation string
	tag         string
}

// palettes are the presets' colors, light first and dark second.
var palettes = map[string][2]palette{
	Xink: {
		{keyword: "#0433ff", function: "#0000a2", typeName: "#6f42c1",
			str: "#036a07", escape: "#036a07", number: "#0433ff",
			constant: "#c5060b", attribute: "#957931",
			property: "#333333", variable: "#333333",
			comment: "#007fff", tag: "#0433ff"},
		{keyword: "#87b1f6", function: "#b3c5f3", typeName: "#cba6f7",
			str: "#a3e09f", escape: "#68dc7c", number: "#cc9e00",
			constant: "#cc9e00", attribute: "#7faef9",
			property: "#bcc4e0", comment: "#9d9d9d", tag: "#419cff"},
	},
	Ocean: {
		{keyword: "#b48ead", function: "#8fa1b3", str: "#a3be8c",
			escape: "#96b5b4", number: "#d08770", constant: "#d08770",
			attribute: "#d08770", variable: "#bf616a",
			comment: "#a7adba", operator: "#4f5b66", tag: "#bf616a"},
		{keyword: "#b48ead", function: "#8fa1b3", str: "#a3be8c",
			escape: "#96b5b4", number: "#d08770", constant: "#d08770",
			attribute: "#d08770", variable: "#bf616a",
			comment: "#65737e", operator: "#c0c5ce", tag: "#bf616a"},
	},
	Solarized: {
		{keyword: "#859900", function: "#b58900", typeName: "#b58900",
			str: "#2aa198", escape: "#dc322f", number: "#6c71c4",
			constant: "#b58900", attribute: "#b58900",
			property: "#859900", variable: "#268bd2",
			comment: "#93a1a1", operator: "#859900", tag: "#268bd2"},
		{keyword: "#859900", function: "#b58900", typeName: "#b58900",
			str: "#2aa198", escape: "#dc322f", number: "#6c71c4",
			constant: "#b58900", attribute: "#b58900",
			property: "#859900", variable: "#268bd2",
			comment: "#586e75", operator: "#859900", tag: "#268bd2"},
	},
}

// Normalize is the preset a name stands for: one that is not one of the
// three is the default, so that a name edited by hand never leaves code
// unpainted.
func Normalize(name string) string {
	switch name {
	case Xink, Ocean, Solarized:
		return name
	default:
		return Xink
	}
}

// EditorTheme is the preset's colors in the appearance the window has:
// what the editor paints code with, and what the preview paints its code
// blocks with, are one and the same.
func EditorTheme(name string, base *ui.Theme) *editor.Theme {
	if base == nil {
		base = ui.LightTheme()
	}
	t := editor.DefaultTheme(base)
	both := palettes[Normalize(name)]
	p := both[0]
	if base.Dark {
		p = both[1]
	}
	tok := func(k editor.TokenKind, color string) {
		if color == "" {
			return
		}
		t.Tokens[k].Color = ui.Hex(color)
	}
	tok(editor.TokenKeyword, p.keyword)
	tok(editor.TokenFunction, p.function)
	tok(editor.TokenType, p.typeName)
	tok(editor.TokenString, p.str)
	tok(editor.TokenEscape, p.escape)
	tok(editor.TokenNumber, p.number)
	tok(editor.TokenConstant, p.constant)
	tok(editor.TokenAttribute, p.attribute)
	tok(editor.TokenProperty, p.property)
	tok(editor.TokenVariable, p.variable)
	tok(editor.TokenComment, p.comment)
	tok(editor.TokenOperator, p.operator)
	tok(editor.TokenPunctuation, p.punctuation)
	tok(editor.TokenTag, p.tag)
	return t
}
