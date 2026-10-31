package view

import (
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// reviewState is a revision being looked over: what the note was, what
// the model made of it, and which of the changes have been taken.
//
// A revision is not put into the note on the model's word: it is put
// before the user a change at a time, and each is theirs to take or to
// send back.
type reviewState struct {
	open bool
	// path is the note the revision is of.
	path string
	// before is the note as it was, kept so that the whole revision can
	// be sent back at once. It is the piece the revision is of, when
	// the revision is of a piece and not of the whole.
	before string
	// part is whether the revision is of a piece of the note and not
	// of the whole of it, and from and to where that piece stands: an
	// action done to a selection revises the selection and leaves the
	// rest of the note as it is.
	part     bool
	from, to int
	diff     *editor.DiffState
}

// StartReview puts a revision of the note before the user: what the note
// is on one side, and what the model made of it on the other, to be taken
// a change at a time.
func (w *Workspace) StartReview(text string) {
	n := w.Tabs.ActiveNote()
	if n == nil {
		return
	}
	before := n.Editor.Buffer.String()
	if text == before {
		w.status = "the model said what the note says"
		return
	}
	w.review = reviewState{
		open: true, path: n.Path, before: before,
		diff: pieceDiff(before, text),
	}
}

// StartPieceReview puts a revision of one piece of a note before the
// user: what the piece says on one side, and what the model made of it
// on the other. The rest of the note is left as it stands, and what is
// taken goes back where the piece was taken from.
func (w *Workspace) StartPieceReview(from, to int, text string) {
	n := w.Tabs.ActiveNote()
	if n == nil {
		return
	}
	whole := n.Editor.Buffer.String()
	if from < 0 || to > len(whole) || from > to {
		// The note is not as it was when the piece was taken out of
		// it: where it went is no longer known.
		w.status = "the note has changed: the piece cannot be put back"
		return
	}
	before := whole[from:to]
	if text == before {
		w.status = "the model said what the piece says"
		return
	}
	w.review = reviewState{
		open: true, path: n.Path, before: before,
		part: true, from: from, to: to,
		diff: pieceDiff(before, text),
	}
}

// pieceDiff is a revision held as a diff: the one side what it was, the
// other what was made of it, and the changes taken where they show.
func pieceDiff(before, after string) *editor.DiffState {
	return &editor.DiffState{
		Old:      editor.NewBuffer(before),
		New:      editor.NewBuffer(after),
		Wrap:     true,
		Editable: true,
		// The changes are taken or sent back where they show,
		// one at a time, as an agent's are.
		HunkActions: true,
	}
}

// reviewPane is the note's revision, where the note itself is: it takes
// the whole of the middle, for a change is not read in a corner.
func (w *Workspace) reviewPane(c *ui.Context) {
	if !w.review.open {
		return
	}
	ui.Column(c).Grow(1).FillHeight().Children(func() {
		w.reviewBar(c)
		editor.DiffView(c, w.review.diff).Grow(1).FillWidth()
	})
}

// reviewBar is what the bar above the revision says and does: how many
// changes there are, and the way to take them all, to send them all back,
// or to write what is left into the note.
func (w *Workspace) reviewBar(c *ui.Context) {
	diff := w.review.diff
	hunks := diff.Hunks()
	added, removed := diff.Stats()
	ui.Row(c).Gap(8).Padding(6).Children(func() {
		ui.Text(c, "AI 修订 · "+strconv.Itoa(len(hunks))+" 处（+"+
			strconv.Itoa(added)+" −"+strconv.Itoa(removed)+"）").Bold()
		if ui.Button(c.Key("rev-take"), "Take all").
			Label("Take every change").Clicked() {
			w.reviewEach(true)
		}
		if ui.Button(c.Key("rev-send"), "Send all back").
			Label("Send every change back").Clicked() {
			w.reviewEach(false)
		}
		if ui.Button(c.Key("rev-write"), "Write it in").
			Label("Write what is left into the note").Clicked() {
			w.writeReview()
		}
		if ui.Button(c.Key("rev-away"), "Away").
			Label("Put the revision away").Clicked() {
			w.review.open = false
		}
	})
}

// reviewEach is every change taken, or every one sent back. It goes from
// the last change to the first: taking one out moves the ones after it,
// and not the ones before.
func (w *Workspace) reviewEach(take bool) {
	diff := w.review.diff
	for i := len(diff.Hunks()) - 1; i >= 0; i-- {
		if take {
			diff.Accept(i)
		} else {
			diff.Revert(i)
		}
	}
}

// writeReview is what is left of the revision written into the note, and
// the revision put away.
func (w *Workspace) writeReview() {
	diff := w.review.diff
	n, err := w.Tabs.Open(w.review.path)
	if err != nil {
		w.status = "could not open " + PathOf(w.review.path)
		return
	}
	text := diff.New.String()
	whole := n.Editor.Buffer.String()
	if w.review.part {
		from, to := w.review.from, w.review.to
		if from > len(whole) || to > len(whole) || from > to {
			w.status = "the note has changed: the piece cannot be " +
				"put back"
			return
		}
		// The piece goes back where it was taken from, and the rest
		// of the note is left as it stands.
		text = whole[:from] + text + whole[to:]
	}
	if text == whole {
		// Nothing was taken: the note is left as it was, and saying
		// otherwise would be a lie.
		w.review.open = false
		w.status = "nothing was taken"
		return
	}
	// What the note said before is kept: a save is on the disk at once,
	// and there is no stack of undone things to come back to. What is
	// kept is the whole note, and not only the piece, for it is the
	// whole note that is written over.
	if w.OnSnapshot != nil {
		w.OnSnapshot(w.review.path, whole)
	}
	n.Editor.Buffer.Set(text)
	w.Tabs.Edited(n)
	w.review.open = false
	w.status = "the revision is in the note"
	// And whether what was written still holds together. What is found
	// is said, and does not undo the writing: a vault may well not have
	// the note a draft points at yet.
	if w.OnVerify == nil {
		return
	}
	if issues := w.OnVerify(text); len(issues) > 0 {
		w.status = "written, but " + strconv.Itoa(len(issues)) +
			" 处要再看一眼：" + issues[0].Feedback()
	}
}
