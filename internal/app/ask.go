package app

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/skill"
	"github.com/helightxu/xinkgo/internal/snapshot"
	"github.com/helightxu/xinkgo/internal/state"
)

// ask is the app asking a model: the model the settings name as the one to
// ask, and the events of its answer as they come.
//
// Which model that is, is not for the window to know: it reads the
// settings, and is told when they change, so that a model chosen in the
// settings is the one asked next.
func (s *appState) ask(
	ctx context.Context, messages []ai.Message, model string,
) (<-chan ai.Event, error) {
	cfg := s.config.Ai
	if !cfg.Enabled {
		return nil, ai.ErrNotConfigured
	}
	entry, ok := s.modelFor(cfg, model)
	if !ok {
		return nil, ai.ErrNoModel
	}
	s.noteRecentModel(entry.ID)
	endpoint := ai.NewEndpoint(entry, cfg)
	req := ai.Request{
		Model:    entry.ID,
		Messages: messages,
		Params: ai.Params{
			Temperature: entry.Params.Temperature,
			NumPredict:  entry.Params.NumPredict,
			NumCtx:      entry.Params.NumCtx,
			Think:       entry.Params.Think,
		},
	}
	// Tools, when the model is one that has been found to call them,
	// and the user has said it may look through the vault first.
	if cfg.Agentic && entry.ToolsEnabled {
		req.Tools = ai.Specs(cfg.SkillsEnabled)
	}
	if len(req.Tools) > 0 {
		// With tools it may reach for, one asking is not the whole of
		// it: what it finds is given back, and it is asked again.
		steps := int(cfg.MaxSteps)
		if steps <= 0 {
			steps = 1
		}
		return ai.Loop(ctx, cfg, endpoint, s.host(), req, steps), nil
	}
	pieces, err := ai.Ask(ctx, cfg, endpoint, req)
	if err != nil {
		return nil, err
	}
	return ai.Events(endpoint, pieces), nil
}

// noteRecentModel remembers which model was last asked. The list is kept
// on the workspace for the picker and written to disk off the frame, so
// asking a question never waits on a file.
func (s *appState) noteRecentModel(id string) {
	if id == "" || s.ws == nil {
		return
	}
	list := []string{id}
	for _, one := range s.ws.RecentModels {
		if one != id && len(list) < 8 {
			list = append(list, one)
		}
	}
	s.ws.RecentModels = list
	go func() {
		if err := state.SaveRecentModels(s.dir, list); err != nil {
			slog.Warn("the recent models could not be kept", "err", err)
		}
	}()
}

// discoverSkills looks for skills in the vault and in the application's
// own folder, and gives the window the ones that may be used.
func (s *appState) discoverSkills() {
	s.skills, s.skillIssues = skill.Discover(s.root, skill.UserDir(s.dir))
	s.publishSkills()
}

// publishSkills is what the window is given: the skills that have not
// been turned off, and none at all when skills are turned off whole.
func (s *appState) publishSkills() {
	if s.ws == nil {
		return
	}
	if !s.config.Ai.SkillsEnabled {
		s.ws.Skills = nil
		return
	}
	s.ws.Skills = skill.Enabled(s.skills, s.config.Ai.DisabledSkills)
}

// publishActions is what the window is given: the app's own actions over
// a note, the ones that are turned on. They are the settings', and are
// given again whenever the settings change.
func (s *appState) publishActions() {
	if s.ws == nil {
		return
	}
	s.ws.Actions = config.EnabledActions(s.config.Ai.Actions)
}

// snapshotNote keeps what a note said before a revision was written into
// it: a save is on the disk at once, and there is no stack of undone
// things to come back to.
//
// A note that cannot be kept is not a reason to stop the revision: it is
// said in the log, and the user is not told their work failed.
func (s *appState) snapshotNote(path, text string) string {
	dir := snapshot.Dir(s.dir)
	var (
		where string
		err   error
	)
	// A snapshot remembers both the vault and the note's path inside
	// it, so two notes of the same name are no longer confused and a
	// snapshot from another vault is not put back here. Without a vault
	// yet, fall back to the old file-name-only form.
	if vault := s.vaultID(); vault != "" {
		if rel, ok := s.relativeToVault(path); ok {
			where, err = snapshot.WriteFor(dir, vault, rel, text)
		} else {
			where, err = snapshot.Write(dir, filepath.Base(path), text)
		}
	} else {
		where, err = snapshot.Write(dir, filepath.Base(path), text)
	}
	if err != nil {
		slog.Warn("the note could not be kept before it was changed",
			"err", err)
		return ""
	}
	return where
}

// verifyNote is whether a note still holds together after a model has
// written it: its links, its front matter, and its tags.
func (s *appState) verifyNote(text string) []ai.Issue {
	if s.idx == nil {
		// No vault open: there is nothing to resolve a link against.
		return ai.Verify(text, nil)
	}
	return ai.Verify(text, s.idx)
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

// modelFor is the model to ask: the one named, when the settings hold it
// and it is turned on; otherwise the one the settings name as the default;
// otherwise the first one that is turned on.
//
// A name the settings do not hold is not asked: it was picked in the bar
// out of a list that has since changed, and asking nothing at all is not
// an answer either.
func (s *appState) modelFor(
	cfg config.AiConfig, named string,
) (config.ModelEntry, bool) {
	if named != "" {
		for _, m := range cfg.Models {
			if m.ID == named && m.Enabled {
				return m, true
			}
		}
	}
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
