package app

import (
	"context"
	"log/slog"

	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/state"
)

// ask is the app asking a model: the model the settings name as the one to
// ask, and the events of its answer as they come.
//
// Which model that is, is not for the window to know: it reads the
// settings, and is told when they change, so that a model chosen in the
// settings is the one asked next.
func (s *appState) ask(
	ctx context.Context, messages []ai.Message,
) (<-chan ai.Event, error) {
	cfg := s.config.Ai
	if !cfg.Enabled {
		return nil, ai.ErrNotConfigured
	}
	model, ok := s.modelFor(cfg)
	if !ok {
		return nil, ai.ErrNoModel
	}
	endpoint := ai.NewEndpoint(model, cfg)
	pieces, err := ai.Ask(ctx, cfg, endpoint, ai.Request{
		Model:    model.ID,
		Messages: messages,
		Params: ai.Params{
			Temperature: model.Params.Temperature,
			NumPredict:  model.Params.NumPredict,
			NumCtx:      model.Params.NumCtx,
			Think:       model.Params.Think,
		},
	})
	if err != nil {
		return nil, err
	}
	return ai.Events(endpoint, pieces), nil
}

// aiLoad is what was said about a note before, read back when the bar is
// opened for it. A file that cannot be read says nothing and keeps the app
// running: a conversation is a thing that can be lost.
func (s *appState) aiLoad(path string) []state.AiBlock {
	return s.aiTalksOf()[path]
}

// aiSave is what has been said about a note, written down. A failure to
// write is said in the log and nowhere else: it must not break the
// conversation the user is having.
func (s *appState) aiSave(path string, blocks []state.AiBlock) {
	all := s.aiTalksOf()
	state.ApplyAiSession(all, path, blocks)
	if err := state.SaveAiSessions(s.dir, all); err != nil {
		slog.Warn("the conversation could not be kept", "err", err)
	}
}

// aiTalksOf is the book of conversations, read off the disk the first
// time it is asked for and kept in hand from then on.
func (s *appState) aiTalksOf() state.AiSessions {
	if s.aiTalks == nil {
		all, err := state.LoadAiSessions(s.dir)
		if err != nil {
			slog.Warn("the conversations could not be read", "err", err)
		}
		if all == nil {
			all = state.AiSessions{}
		}
		s.aiTalks = all
	}
	return s.aiTalks
}

// modelFor is the model to ask: the one the settings name as the default,
// or the first one that is turned on when none is named.
func (s *appState) modelFor(cfg config.AiConfig) (config.ModelEntry, bool) {
	if cfg.DefaultModel != "" {
		for _, m := range cfg.Models {
			if m.ID == cfg.DefaultModel && m.Enabled {
				return m, true
			}
		}
	}
	for _, m := range cfg.Models {
		if m.Enabled {
			return m, true
		}
	}
	return config.ModelEntry{}, false
}
