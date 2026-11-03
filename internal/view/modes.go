// This file is the shapes the window may be put in: a pane given the whole
// window, the note stretched over the preview, the preview spread over the
// note. One mode at a time, and the one button that gives it is the one
// that takes it away.

package view

import (
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/helightxu/xinkgo/internal/view/layout"
)

// setZoom gives pane the whole window, or gives the room back when it has
// it already: one button, and the same one out again.
func (w *Workspace) setZoom(pane layout.Zoom) {
	if w.zoom == pane {
		w.zoom = layout.ZoomNone
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
