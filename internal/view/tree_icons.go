package view

import (
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/media"
	"github.com/helightxu/xinkgo/internal/vault"
)

// The tree's icons, parsed once: a folder open or closed, a kind of file
// each looking like its own, and the few the header's buttons use. They
// are drawn in the color of the text beside them, so that they follow the
// size and the appearance the window has.
var (
	iconChevron = svgIcon(`<path d="m9 18 6-6-6-6"/>`)

	iconFolder     = svgIcon(`<path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"/>`)
	iconFolderOpen = svgIcon(`<path d="m6 14 1.45-2.9A2 2 0 0 1 9.24 10H20a2 2 0 0 1 1.94 2.5l-1.55 6a2 2 0 0 1-1.95 1.5H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h3.9a2 2 0 0 1 1.69.9l.81 1.2a2 2 0 0 0 1.67.9H18a2 2 0 0 1 2 2v2"/>`)

	iconFileText = svgIcon(`<path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2Z"/><path d="M14 2v5h5"/><path d="M16 13H8"/><path d="M16 17H8"/><path d="M10 9H8"/>`)
	iconFile     = svgIcon(`<path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2Z"/><path d="M14 2v5h5"/>`)
	iconImage    = svgIcon(`<rect width="18" height="18" x="3" y="3" rx="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.1-3.1a2 2 0 0 0-2.8 0L6 21"/>`)
	iconCode     = svgIcon(`<path d="m16 18 6-6-6-6"/><path d="m8 6-6 6 6 6"/>`)
	iconDatabase = svgIcon(`<ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v14a9 3 0 0 0 18 0V5"/><path d="M3 12a9 3 0 0 0 18 0"/>`)
	iconPackage  = svgIcon(`<path d="m7.5 4.3 9 5.1"/><path d="M21 8a2 2 0 0 0-1-1.7l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.7l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16Z"/><path d="m3.3 7 8.7 5 8.7-5"/><path d="M12 22V12"/>`)
	iconTable    = svgIcon(`<rect width="18" height="18" x="3" y="3" rx="2"/><path d="M3 9h18"/><path d="M3 15h18"/><path d="M9 3v18"/>`)
	iconMusic    = svgIcon(`<path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/>`)
	iconFilm     = svgIcon(`<rect width="18" height="18" x="3" y="3" rx="2"/><path d="M7 3v18"/><path d="M17 3v18"/><path d="M3 7.5h4"/><path d="M3 12h18"/><path d="M3 16.5h4"/><path d="M17 7.5h4"/><path d="M17 16.5h4"/>`)

	iconFilePlus   = svgIcon(`<path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2Z"/><path d="M14 2v5h5"/><path d="M12 11v6"/><path d="M9 14h6"/>`)
	iconFolderPlus = svgIcon(`<path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"/><path d="M12 11v6"/><path d="M9 14h6"/>`)
	iconRefresh    = svgIcon(`<path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/><path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16"/><path d="M8 16H3v5"/>`)
	iconFoldAll    = svgIcon(`<path d="m7 20 5-5 5 5"/><path d="m7 4 5 5 5-5"/>`)
	iconUnfoldAll  = svgIcon(`<path d="m7 15 5 5 5-5"/><path d="m7 9 5-5 5 5"/>`)
)

// svgIcon is one icon: paths and shapes of a 24 by 24 box, drawn as a
// stroke in the color of the text beside it.
func svgIcon(body string) *ui.SVG {
	const head = `<svg xmlns="http://www.w3.org/2000/svg" ` +
		`viewBox="0 0 24 24" fill="none" stroke="currentColor" ` +
		`stroke-width="2" stroke-linecap="round" ` +
		`stroke-linejoin="round">`
	return ui.MustParseSVG([]byte(head + body + `</svg>`))
}

// iconOf is the icon an entry is drawn by: a folder open or closed, or a
// file by what it is.
func iconOf(e *vault.Entry, open bool) *ui.SVG {
	if e.IsDir {
		if open {
			return iconFolderOpen
		}
		return iconFolder
	}
	switch media.KindOf(e.Path) {
	case media.KindMarkdown:
		return iconFileText
	case media.KindImage:
		return iconImage
	case media.KindCode:
		return iconCode
	case media.KindData:
		return iconDatabase
	case media.KindArchive:
		return iconPackage
	case media.KindSpreadsheet:
		return iconTable
	case media.KindAudio:
		return iconMusic
	case media.KindVideo:
		return iconFilm
	default:
		return iconFile
	}
}
