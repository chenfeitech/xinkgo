// This file is the shapes the window may be put in: a pane given the whole
// window, the note stretched over the preview, the preview spread over the
// note. One mode at a time, and the one button that gives it is the one
// that takes it away.

package view

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// zoom is which pane has been given the whole window: the note's or the
// preview's. zoomNone is the usual, in which every panel keeps the room
// the user gave it.
type zoom int

const (
	zoomNone zoom = iota
	// zoomEditor is the note's: the tree and the preview are not drawn,
	// and the note is as wide as the window.
	zoomEditor
	// zoomPreview is the preview's, for reading: the note's own pane is
	// not drawn, and the preview is as wide as the window.
	zoomPreview
	// zoomPreviewSpread is the preview's over the note alone: the
	// preview reaches left as far as the tree, and the tree keeps the
	// room it was given, so that another note may be chosen in it while
	// one is read. It is what the button in the preview's own corner
	// gives, and not the whole window, which is the rail's button.
	zoomPreviewSpread
)

// setZoom gives pane the whole window, or gives the room back when it has
// it already: one button, and the same one out again.
func (w *Workspace) setZoom(pane zoom) {
	if w.zoom == pane {
		w.zoom = zoomNone
		return
	}
	w.zoom = pane
}

// widenAnim is how the note goes over the preview's room, and back: a
// curve rather than a spring, so that it neither overshoots the window's
// edge nor keeps going after the pointer has left the button.
var widenAnim = ui.Animation{
	Duration: 180 * time.Millisecond,
	Ease:     ui.EaseInOut,
}
