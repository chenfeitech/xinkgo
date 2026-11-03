// This file is the asking: what is sent to a model, what comes back a
// piece at a time, and what is made of it when it is over.

package view

import (
	"context"
	"strings"

	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
)

// runAction is one of the app's own actions run on the note: what the
// action says is asked of the model, and what comes back is put before
// the user as a revision — as any other revision is, and not written
// into the note on the model's word.
func (w *Workspace) runAction(a config.AiAction) {
	n := w.Tabs.ActiveNote()
	if n == nil {
		w.status = locale.StatusNoNoteForAction()
		return
	}
	s := w.aiOf(n.Path)
	if s.isBusy() {
		w.status = locale.StatusStillAnswering()
		return
	}
	// A word that was given nothing to ask asks nothing at all: the
	// note is left as it is, and the user is told where to write it.
	if strings.TrimSpace(a.Prompt) == "" {
		w.status = locale.StatusQuickNoPrompt(a.Name)
		return
	}
	piece, from, to, part := pieceOf(n, a)
	s.Open = true
	w.aiSend(n, s, askingOf(a, piece, part), modelOf(a, w.AiModel),
		&waitReview{part: part, from: from, to: to})
	w.status = locale.StatusActionAsked(a.Name)
}

// pieceOf is what an action is done to: the selection, when the note has
// one and the action is one that goes by what is selected; and the whole
// note otherwise. It gives back the words, and where they stand in the
// note — 0 and 0 when they are the whole of it.
func pieceOf(
	n *OpenNote, a config.AiAction,
) (text string, from, to int, part bool) {
	whole := n.Editor.Buffer.String()
	sel := n.Editor.Selection()
	if a.Scope == config.ActionScopeWhole || sel.Empty() {
		return whole, 0, 0, false
	}
	from, to = sel.Start(), sel.End()
	if from < 0 {
		from = 0
	}
	if to > len(whole) {
		to = len(whole)
	}
	if from > to {
		from = to
	}
	return whole[from:to], from, to, true
}

// askingOf is what the model is asked when an action is run: what the
// action says. When the action is done to a piece of the note and not to
// the whole of it, the piece goes with it — the note as a whole the
// model has already been told — and it is told to answer with the piece
// and nothing else, for what comes back goes back where the piece was
// taken from.
func askingOf(a config.AiAction, piece string, part bool) string {
	if !part {
		return a.Prompt
	}
	return a.Prompt + "\n\nThis is the piece to do it to. Answer with " +
		"the piece as it should be, and with nothing else:\n\n" + piece
}

// aiReviewDone is an answer that an action was waiting for, put before
// the user as a revision now that it is over.
//
// It is done here, by the frame, and not where the answer is heard: a
// revision is made out of the note's own editor, and only this thread
// may touch that.
func (w *Workspace) aiReviewDone() {
	n := w.Tabs.ActiveNote()
	if n == nil || w.ai == nil {
		return
	}
	s := w.ai[n.Path]
	if s == nil {
		return
	}
	s.mu.Lock()
	wait := s.waiting
	text, failed := "", ""
	if wait != nil && !s.busy && len(s.Blocks) > 0 {
		last := s.Blocks[len(s.Blocks)-1]
		if last.Over {
			text, failed = last.Text, last.Failed
			s.waiting = nil
		}
	}
	s.mu.Unlock()
	if text == "" || failed != "" {
		return
	}
	if wait.part {
		w.StartPieceReview(wait.from, wait.to, text)
		return
	}
	w.StartReview(text)
}

// aiAsk is the question asked: what was typed into the place to ask.
func (w *Workspace) aiAsk(n *OpenNote, s *aiSession) {
	w.aiSend(n, s, strings.TrimSpace(s.Input), w.AiModel, nil)
	s.Input = ""
	s.focus = true
}

// aiSend is a question sent: it is written into the conversation before
// the model is asked, and the answer grows into the block after it as it
// comes.
//
// model is which model is asked of the ones the settings hold, empty for
// the one the settings name. wait, when there is one, is what the answer
// is to be put before the user as when it is over, and which piece of the
// note it is about.
func (w *Workspace) aiSend(
	n *OpenNote, s *aiSession, asking, model string, wait *waitReview,
) {
	text := strings.TrimSpace(asking)
	if text == "" {
		return
	}
	s.mu.Lock()
	s.Blocks = append(s.Blocks,
		aiBlock{Role: ai.User, Text: text, Over: true},
		aiBlock{Role: ai.Assistant},
	)
	at := len(s.Blocks) - 1
	s.busy = true
	s.dirty = true
	s.waiting = wait
	s.mu.Unlock()

	if w.OnAsk == nil {
		w.aiDone(s, at, ai.ErrNotConfigured)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.cancel = cancel
	s.mu.Unlock()
	events, err := w.OnAsk(ctx, w.aiTold(s, n), model)
	if err != nil {
		cancel()
		w.aiDone(s, at, err)
		return
	}
	go w.aiHear(s, at, events, cancel)
}

// aiHear is the answer being heard: each piece of it is written into the
// block that waits for it, until the model is done or something goes
// wrong.
func (w *Workspace) aiHear(
	s *aiSession, at int, events <-chan ai.Event, cancel context.CancelFunc,
) {
	defer cancel()
	for event := range events {
		s.mu.Lock()
		if at < len(s.Blocks) {
			b := &s.Blocks[at]
			switch event.Kind {
			case ai.EventDelta:
				b.Text += event.Text
			case ai.EventThinking:
				b.Thinking += event.Text
			case ai.EventTruncated:
				b.Cut = true
			case ai.EventToolResult:
				b.Tool = append(b.Tool, event.Result)
			case ai.EventFailed:
				b.Failed = event.Err.Error()
				b.Over = true
			case ai.EventDone:
				b.Over = true
			}
		}
		s.mu.Unlock()
	}
	// An answer that stopped without saying it was over is over anyway:
	// what was said of it was said as it came.
	s.mu.Lock()
	if at < len(s.Blocks) {
		s.Blocks[at].Over = true
	}
	s.busy = false
	s.cancel = nil
	s.dirty = true
	s.mu.Unlock()
}

// aiDone is an answer that never began.
func (w *Workspace) aiDone(s *aiSession, at int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at < len(s.Blocks) {
		s.Blocks[at].Failed = err.Error()
		s.Blocks[at].Over = true
	}
	s.busy = false
	s.cancel = nil
	s.dirty = true
}
