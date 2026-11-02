package view

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/helightxu/xinkgo/internal/config"
)

// What the weights of a font are called, and what they weigh.
var weights = map[string]int{
	"thin":     100,
	"light":    300,
	"regular":  400,
	"normal":   400,
	"medium":   500,
	"semibold": 600,
	"bold":     700,
	"heavy":    900,
	"black":    900,
}

// applyEditor is what the user chose for the editor, put on the editor
// itself: the font it draws with, whether long lines are wrapped, whether
// the lines are numbered, and what indenting puts in.
//
// It is put on every time the window is drawn, so that a choice made in
// the settings is seen at once, with no note to reopen.
func (w *Workspace) applyEditor(n *OpenNote) {
	ed := n.Editor
	font := w.Config.Editor.Font
	ed.Font = ui.Font{
		Family: font.Family,
		Size:   font.Size,
		Weight: weightOf(font.Weight),
	}
	ed.LineHeight = font.LineHeight
	ed.SoftWrap = w.Config.Editor.SoftWrap
	if w.Config.Editor.ShowLineNumbers {
		ed.LineNumbers = editor.LineNumbersOn
	} else {
		ed.LineNumbers = editor.LineNumbersOff
	}
	ed.TabSize = int(w.Config.Editor.Indent.TabSize)
	// Nothing here folds a note: the marks of folding stand in a gutter
	// of their own beside the lines, which takes the room the note's
	// padding gives it on that side and on that side only, and left the
	// note standing nearer one edge of its pane than the other.
	ed.NoFolding = true
	if w.Config.Editor.Indent.UseTabs {
		ed.Indent = editor.IndentTabs
		return
	}
	ed.Indent = editor.IndentSpaces
}

// weightOf is how heavy a font is, from the name the settings keep.
func weightOf(name string) int {
	if weight, ok := weights[name]; ok {
		return weight
	}
	return 400
}

// previewTheme is how the preview looks: the size and family of its
// text, how far apart its blocks stand, and the code colors of the
// preset the user chose, the ones the editor paints with.
func (w *Workspace) previewTheme(c *ui.Context) *markdown.Theme {
	font := w.Config.Preview.Font
	theme := &markdown.Theme{
		CodeColors:   w.codeThemeFor(c),
		FontSize:     font.Size,
		Font:         font.Family,
		LineHeight:   font.LineHeight,
		BlockSpacing: spacing(w.Config.Preview.Density),
		ItemSpacing:  spacing(w.Config.Preview.Density) / 2,
	}
	return theme
}

// spacing is how far apart the blocks of the preview stand, in ems: the
// closer together the denser it is.
func spacing(density string) float32 {
	switch density {
	case config.DensityCompact:
		return 0.5
	case config.DensityRelaxed:
		return 1
	default:
		return 0.75
	}
}

// previewWidth is how wide the preview lets its text run: as the user
// chose, and as far as the window gives it room.
func (w *Workspace) previewWidth(available float32) float32 {
	if w.Config.Preview.MaxWidth <= 0 {
		return available
	}
	if float32(w.Config.Preview.MaxWidth) < available {
		return float32(w.Config.Preview.MaxWidth)
	}
	return available
}
