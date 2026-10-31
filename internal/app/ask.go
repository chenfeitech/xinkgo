package app

import (
	"context"

	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
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
