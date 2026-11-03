// This file is what is said about a note: one conversation to a note,
// the turns of it, and how they are kept between two runs of the app.

package view

import (
	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/state"
)

// blocks is what has been said, as it stands now.
func (s *aiSession) blocks() []aiBlock {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]aiBlock{}, s.Blocks...)
}

// isBusy is whether the model is still answering.
func (s *aiSession) isBusy() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.busy
}

// aiStop is the answer stopped: the model is let go, and what it had said
// by then is kept.
func (w *Workspace) aiStop(s *aiSession) {
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// aiForget is the conversation begun again.
func (w *Workspace) aiForget(s *aiSession) {
	w.aiStop(s)
	s.mu.Lock()
	s.Blocks = nil
	s.busy = false
	s.cancel = nil
	// Nothing left to keep is the same as never having kept it: the
	// note's conversation is taken out of the book.
	s.dirty = true
	s.mu.Unlock()
	s.Input = ""
}

// aiSave is the conversation written down, when it has changed since it
// last was.
//
// It is the frame that writes, and not the goroutine that hears the
// answer: the book the conversation goes into is the app's own, and is
// not to be touched from another goroutine.
func (w *Workspace) aiSave(path string, s *aiSession) {
	if w.OnAiSave == nil {
		return
	}
	s.mu.Lock()
	if !s.dirty {
		s.mu.Unlock()
		return
	}
	s.dirty = false
	blocks := append([]aiBlock{}, s.Blocks...)
	s.mu.Unlock()
	w.OnAiSave(path, w.aiStored(blocks))
}

// aiStored is what is kept of a conversation: what was said, and none of
// what is still being said, or of what went wrong instead of it.
func (w *Workspace) aiStored(blocks []aiBlock) []state.AiBlock {
	out := make([]state.AiBlock, 0, len(blocks))
	for _, b := range blocks {
		if b.Text == "" || b.Failed != "" {
			continue
		}
		if b.Role != ai.User && b.Role != ai.Assistant {
			continue
		}
		out = append(out, state.AiBlock{
			Role:     string(b.Role),
			Text:     b.Text,
			Thinking: b.Thinking,
			Cut:      b.Cut,
		})
	}
	return out
}

// aiRestored is a conversation read back: everything in it was said, and
// is over.
func (w *Workspace) aiRestored(blocks []state.AiBlock) []aiBlock {
	out := make([]aiBlock, 0, len(blocks))
	for _, b := range blocks {
		role := ai.Role(b.Role)
		if role != ai.User && role != ai.Assistant {
			continue
		}
		out = append(out, aiBlock{
			Role:     role,
			Text:     b.Text,
			Thinking: b.Thinking,
			Cut:      b.Cut,
			Over:     true,
		})
	}
	return out
}
