package view

import (
	"github.com/egoist/mygo/ui"
	xmd "github.com/helightxu/xinkgo/internal/markdown"
)

// How large an icon of the toolbar is drawn, and how high the line
// between two of its groups stands.
const (
	iconSize    float32 = 16
	iconDivider float32 = 16
)

// The pictures of the Markdown toolbar's buttons, made once as the tree's
// are. A picture, a table and a piece of code are drawn by the tree's own
// icons, which say the same thing; the shapes of the rest are Lucide's
// (ISC licence).
var (
	iconBold      = svgIcon(`<path d="M6 12h9a4 4 0 0 1 0 8H7a1 1 0 0 1-1-1V5a1 1 0 0 1 1-1h7a4 4 0 0 1 0 8"/>`)
	iconItalic    = svgIcon(`<line x1="19" x2="10" y1="4" y2="4"/><line x1="14" x2="5" y1="20" y2="20"/><line x1="15" x2="9" y1="4" y2="20"/>`)
	iconStrike    = svgIcon(`<path d="M16 4H9a3 3 0 0 0-2.83 4"/><path d="M14 12a4 4 0 0 1 0 8H6"/><line x1="4" x2="20" y1="12" y2="12"/>`)
	iconHeading   = svgIcon(`<path d="M6 12h12"/><path d="M6 20V4"/><path d="M18 20V4"/>`)
	iconBullets   = svgIcon(`<path d="M3 5h.01"/><path d="M3 12h.01"/><path d="M3 19h.01"/><path d="M8 5h13"/><path d="M8 12h13"/><path d="M8 19h13"/>`)
	iconNumbers   = svgIcon(`<path d="M11 5h10"/><path d="M11 12h10"/><path d="M11 19h10"/><path d="M4 4h1v5"/><path d="M4 9h2"/><path d="M6.5 20H3.4c0-1 2.6-1.925 2.6-3.5a1.5 1.5 0 0 0-2.6-1.02"/>`)
	iconTasks     = svgIcon(`<path d="M13 5h8"/><path d="M13 12h8"/><path d="M13 19h8"/><path d="m3 17 2 2 4-4"/><rect x="3" y="4" width="6" height="6" rx="1"/>`)
	iconQuote     = svgIcon(`<path d="M17 5H3"/><path d="M21 12H8"/><path d="M21 19H8"/><path d="M3 12v7"/>`)
	iconCodeBlock = svgIcon(`<path d="m10 9-3 3 3 3"/><path d="m14 15 3-3-3-3"/><rect x="3" y="3" width="18" height="18" rx="2"/>`)
	iconLink      = svgIcon(`<path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>`)
	iconRule      = svgIcon(`<path d="M5 12h14"/>`)
	// The two panes given the whole window: a frame with the pane that
	// takes it, and an arrow saying which way it grows. Their shapes are
	// Lucide's "maximize" and "minimize", turned to say which pane.
	iconZoomEditor  = svgIcon(`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M13 4v16"/><path d="m10 9-4 3 4 3"/>`)
	iconZoomPreview = svgIcon(`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M11 4v16"/><path d="m14 9 4 3-4 3"/>`)
	// The preview spread over the note: a frame whose right pane is the
	// preview, and an arrow from it pointing left, into the room the
	// note's pane gives up.
	iconPreviewSpread = svgIcon(`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M15 4v16"/><path d="m12 8-4 4 4 4"/>`)
	// The way back: a pane given the whole window gives it up again,
	// which is the same picture as Lucide's "minimize".
	iconZoomBack = svgIcon(`<path d="M4 14h6v6"/><path d="M20 10h-6V4"/><path d="M14 10l7-7"/><path d="M3 21l7-7"/>`)

	// A spark, as the others are: the picture of what a model is asked
	// for, which is not a talk but a favour. Lucide's "sparkles".
	iconAi = svgIcon(`<path d="M11.017 2.814a1 1 0 0 1 1.966 0l1.051 5.558a2 2 0 0 0 1.594 1.594l5.558 1.051a1 1 0 0 1 0 1.966l-5.558 1.051a2 2 0 0 0-1.594 1.594l-1.051 5.558a1 1 0 0 1-1.966 0l-1.051-5.558a2 2 0 0 0-1.594-1.594l-5.558-1.051a1 1 0 0 1 0-1.966l5.558-1.051a2 2 0 0 0 1.594-1.594z"/><path d="M20 2v4"/><path d="M22 4h-4"/><circle cx="4" cy="20" r="2"/>`)
)

// formatIcon is the picture a button of the Markdown toolbar shows.
func formatIcon(f xmd.Format) *ui.SVG {
	switch f {
	case xmd.Bold:
		return iconBold
	case xmd.Italic:
		return iconItalic
	case xmd.Strikethrough:
		return iconStrike
	case xmd.InlineCode:
		return iconCode
	case xmd.FormatHeading:
		return iconHeading
	case xmd.BulletList:
		return iconBullets
	case xmd.NumberedList:
		return iconNumbers
	case xmd.TaskList:
		return iconTasks
	case xmd.Quote:
		return iconQuote
	case xmd.CodeBlock:
		return iconCodeBlock
	case xmd.FormatLink:
		return iconLink
	case xmd.Image:
		return iconImage
	case xmd.Table:
		return iconTable
	default:
		return iconRule
	}
}
