package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/secret"
)

// modelState is what one model's row keeps between two frames: whether
// it is open, the last thing trying it out said, and the last list of
// models its address gave back.
type modelState struct {
	open    bool
	trying  bool
	probe   *ai.Probe
	listing bool
	held    []string
	listErr error
	// idField is what the field of the model's name holds while it is
	// being written into, and idSet whether it has been filled in: the
	// name is what the row is known by, so it is not taken into the
	// settings one letter at a time but when the writing is done.
	idField string
	idSet   bool
	// keyNote is what keeping the key in the keyring came to, said
	// under the row when it was said at all.
	keyNote string
}

// ollamaState is what the window knows of the models the machine itself
// holds: the last look at them, when it was made, and what came of it.
type ollamaState struct {
	looking bool
	at      time.Time
	models  []ai.OllamaModel
	err     error
}

// stateOf is the row state of the model of that name, made the first time
// it is asked for.
func (st *settings) stateOf(id string) *modelState {
	if st.models == nil {
		st.models = map[string]*modelState{}
	}
	if _, ok := st.models[id]; !ok {
		st.models[id] = &modelState{}
	}
	return st.models[id]
}

// ai is the settings of the AI: whether it is used at all, the models the
// machine holds, the models the user wrote in, and how a question is
// asked.
func (s *appState) ai(c *ui.Context) {
	toggle(c, locale.AiSettingAskAtAll(),
		locale.AiSettingAskAtAllNote(), &s.draft.cfg.Ai.Enabled)
	if !s.draft.cfg.Ai.Enabled {
		return
	}
	s.aiOllama(c)
	s.aiOther(c)
	s.aiWhich(c)
	s.aiAsking(c)
	s.aiQuickWords(c)
	s.aiSkills(c)
}

// aiSkills is the skills found in the vault and in the application's own
// folder: each may be turned off, and one that was found but cannot be
// used is said so rather than left out.
func (s *appState) aiSkills(c *ui.Context) {
	group(c, "Skills", func() {
		if len(s.skills) == 0 && len(s.skillIssues) == 0 {
			ui.Text(c, locale.AiSkillsNone()).TextColor(
				c.Theme().TextMuted).FontSize(11)
		}
		for _, one := range s.skills {
			one := one
			off := holds(s.draft.cfg.Ai.DisabledSkills, one.Name)
			use := !off
			toggle(c, one.Name,
				one.Description+" · "+one.From.Label(), &use)
			if use == off {
				// It was turned, one way or the other.
				s.draft.cfg.Ai.DisabledSkills =
					without(s.draft.cfg.Ai.DisabledSkills, one.Name)
				if !use {
					s.draft.cfg.Ai.DisabledSkills = append(
						s.draft.cfg.Ai.DisabledSkills, one.Name)
				}
			}
		}
		for _, issue := range s.skillIssues {
			ui.Text(c, locale.AiSkillIssue(issue.Path, issue.Reason)).
				TextColor(c.Theme().Danger).FontSize(11)
		}
	})
}

// aiOllama is the models the machine itself holds: where the local Ollama
// is, and, once it has been asked, what it answered.
//
// Nothing of a model here is written in by hand: its name and what it can
// do are both given by the machine, and its address is the one above and
// not one of its own.
func (s *appState) aiOllama(c *ui.Context) {
	group(c, locale.AiGroupOllama(), func() {
		text(c, locale.AiOllamaAddress(), locale.AiOllamaAddressNote(),
			&s.draft.cfg.Ai.Ollama.BaseURL, "http://localhost:11434")
		text(c, locale.AiSettingKeepAlive(),
			locale.AiSettingKeepAliveNote(),
			&s.draft.cfg.Ai.Ollama.KeepAlive, "5m")
		buttons(c, locale.AiOllamaModels(), locale.AiOllamaModelsNote(),
			func() {
				if ui.Button(c.Key("ollama-refresh"),
					locale.AiOllamaRefresh()).
					Disabled(s.draft.ollama.looking).Clicked() {
					s.lookForOllamaModels()
				}
			})
		s.aiOllamaFound(c)
	})
}

// aiOllamaFound is what the last look at the machine found: one row a
// model, each saying what it can do, whether it is asked, and whether it
// is the one asked when a note names none.
func (s *appState) aiOllamaFound(c *ui.Context) {
	th := c.Theme()
	found := s.draft.ollama
	switch {
	case found.looking:
		ui.Text(c, locale.AiOllamaLooking()).TextColor(
			th.TextMuted).FontSize(11)
		return
	case found.err != nil:
		ui.Text(c, locale.AiOllamaNotAsked(found.err.Error())).
			TextColor(th.Danger).FontSize(11)
		return
	case len(found.models) == 0:
		if found.at.IsZero() {
			ui.Text(c, locale.AiOllamaNotLooked()).TextColor(
				th.TextMuted).FontSize(11)
			return
		}
		ui.Text(c, locale.AiOllamaNone()).TextColor(
			th.TextMuted).FontSize(11)
		return
	}
	ui.Text(c, locale.AiOllamaCount(strconv.Itoa(len(found.models)),
		found.at.Format("15:04:05"))).TextColor(
		th.TextMuted).FontSize(11)
	for _, one := range found.models {
		s.aiOllamaModel(c, one)
	}
}

// aiOllamaModel is one model the machine holds: whether the app may ask
// it, how big it is and what it is made of, what it can do, and what
// trying it out came to.
func (s *appState) aiOllamaModel(c *ui.Context, m ai.OllamaModel) {
	th := c.Theme()
	was, there := s.draft.ollamaOn(m.Name)
	on := was
	st := s.draft.stateOf(m.Name)
	asked := s.draft.cfg.Ai.DefaultModel == m.Name
	aiRow(c, func() {
		ui.Row(c).FillWidth().Gap(th.Space(3)).AlignItems(ui.Center).
			Children(func() {
				ui.Checkbox(c, &on, m.Name).Grow(1)
				if size := sizeOf(m.Size); size != "" {
					ui.Text(c, size).TextColor(th.TextMuted).
						FontSize(11)
				}
				if made := madeOf(m); made != "" {
					ui.Text(c, made).TextColor(th.TextMuted).
						FontSize(11)
				}
				if asked {
					ui.Badge(c, locale.AiModelDefault()).FontSize(11)
				}
				if there && ui.Button(c.Key("ollama-try#"+m.Name),
					locale.AiModelTry()).Clicked() && !st.trying {
					s.tryModel(m.Name)
				}
				if ui.Button(c.Key("ollama-default#"+m.Name),
					locale.AiModelMakeDefault()).Clicked() {
					s.draft.addOllamaModel(m)
					s.draft.cfg.Ai.DefaultModel = m.Name
				}
			})
		capabilityChips(c, m.Capabilities, m.Guessed)
		if st.probe != nil {
			ui.Text(c, s.saidOf(st.probe)).TextColor(
				th.TextMuted).FontSize(11)
		} else if st.trying {
			ui.Text(c, locale.AiModelAsking()).TextColor(
				th.TextMuted).FontSize(11)
		}
	})
	// Ticking one takes it into the settings, and unticking takes it
	// out: the machine's own list is not the settings, and only what is
	// in the settings is asked.
	if on != was {
		if on {
			s.draft.addOllamaModel(m)
		} else {
			s.draft.forgetModel(m.Name)
		}
	}
}

// aiOther is the models the user wrote in: an address, a key, and the
// name the address knows the model by.
func (s *appState) aiOther(c *ui.Context) {
	// The rows are drawn from the list of the models the user wrote in,
	// and not from every model: the machine's own are above, and are
	// not moved in among these.
	rows := []int{}
	for i := range s.draft.cfg.Ai.Models {
		if !s.draft.cfg.Ai.Models[i].FromOllama() {
			rows = append(rows, i)
		}
	}
	ui.Column(c).Gap(6).Children(func() {
		for n, i := range rows {
			s.aiOtherModel(c, i, n == 0, n == len(rows)-1)
		}
		if ui.Button(c.Key("add-model"), locale.AiModelsAdd()).Clicked() {
			s.draft.cfg.Ai.Models = append(s.draft.cfg.Ai.Models,
				s.newModel())
		}
	})
	// Forgetting, copying and moving are done after the rows are
	// drawn: a row is not taken out, nor made into two, nor moved, from
	// under the one that follows it.
	if s.draft.drop != "" {
		s.draft.forgetModel(s.draft.drop)
		s.draft.drop = ""
	}
	if s.draft.clone != "" {
		s.cloneModel(s.draft.clone)
		s.draft.clone = ""
	}
	if s.draft.moveID != "" {
		s.moveModel(s.draft.moveID, s.draft.moveBy)
		s.draft.moveID = ""
		s.draft.moveBy = 0
	}
}

// aiOtherModel is one model written in by hand: what it is called, where
// it is asked, what it is asked with, what it can do, and what trying it
// out and asking what it holds came to.
func (s *appState) aiOtherModel(c *ui.Context, at int, first, last bool) {
	th := c.Theme()
	m := &s.draft.cfg.Ai.Models[at]
	st := s.draft.stateOf(m.ID)
	aiRow(c, func() {
		ui.Row(c).FillWidth().Gap(th.Space(3)).AlignItems(ui.Center).
			Children(func() {
				ui.Checkbox(c, &m.Enabled, locale.AiModelEnabled())
				ui.Text(c, m.ID).Bold().Grow(1)
				if s.draft.cfg.Ai.DefaultModel == m.ID {
					ui.Badge(c, "Default").FontSize(11)
				}
				if ui.Button(c.Key("model-up#"+m.ID),
					locale.AiModelEarlier()).Label(
					locale.AiModelEarlierLabel()).Disabled(
					first).Clicked() {
					s.draft.moveID = m.ID
					s.draft.moveBy = -1
				}
				if ui.Button(c.Key("model-down#"+m.ID),
					locale.AiModelLater()).Label(
					locale.AiModelLaterLabel()).Disabled(
					last).Clicked() {
					s.draft.moveID = m.ID
					s.draft.moveBy = 1
				}
				open := st.open
				label := locale.AiModelOpen()
				if open {
					label = locale.AiModelShut()
				}
				if ui.Button(c.Key("model-open#"+m.ID), label).Clicked() {
					st.open = !st.open
				}
				if ui.Button(c.Key("model-try#"+m.ID),
					locale.AiModelTest()).Label(
					locale.AiModelTestLabel()).Clicked() && !st.trying {
					s.tryModel(m.ID)
				}
				if ui.Button(c.Key("model-clone#"+m.ID),
					locale.AiModelClone()).Label(
					locale.AiModelCloneLabel()).Clicked() {
					s.draft.clone = m.ID
				}
				if ui.Button(c.Key("model-drop#"+m.ID),
					locale.AiModelForget()).Clicked() {
					s.draft.drop = m.ID
				}
			})
		if st.probe != nil {
			ui.Text(c, s.saidOf(st.probe)).TextColor(
				c.Theme().TextMuted).FontSize(11)
			if st.probe.OK {
				capabilityChips(c,
					ai.Measured(*st.probe, ai.Effective(*m)), false)
			}
		} else if st.trying {
			ui.Text(c, locale.AiModelAsking()).TextColor(
				c.Theme().TextMuted).FontSize(11)
		}
		if !st.open {
			return
		}
		ui.Column(c).FillWidth().Children(func() {
			text(c, locale.AiModelName(), locale.AiModelNameNote(),
				&m.Label, m.ID)
			// The name the address knows it by is the name the app
			// asks by: it is written out, or taken from the list
			// the address gives back.
			if !st.idSet {
				st.idField = m.ID
				st.idSet = true
			}
			field(c, locale.AiModelId(), locale.AiModelIdNote(), func() {
				if ui.TextInput(c, &st.idField).
					Placeholder(locale.AiModelIdPlaceholder()).
					Submitted() &&
					st.idField != m.ID &&
					!s.draft.renameModel(m.ID, st.idField) {
					st.idField = m.ID
				}
			})
			choice(c, locale.AiModelSpeaks(),
				locale.AiModelSpeaksNote(), &m.Kind,
				[]string{
					config.ModelOllama, config.ModelOpenAICompatible,
				})
			text(c, locale.AiModelAddress(),
				locale.AiModelAddressNote(), &m.BaseURL,
				"https://api.openai.com/v1")
			choice(c, locale.AiModelKeyFrom(),
				locale.AiModelKeyFromNote(), &m.ApiKeySource,
				[]string{config.KeyPlain, config.KeyEnv})
			if m.ApiKeySource == config.KeyEnv {
				text(c, locale.AiModelKeyName(),
					locale.AiModelKeyNameNote(), &m.PlainKey,
					locale.AiModelKeyNamePlaceholder())
			} else {
				text(c, locale.AiModelKey(),
					locale.AiModelKeyNote(), &m.PlainKey,
					locale.AiModelKeyPlaceholder())
			}
			if secret.Available() {
				s.keyRingOf(c, m)
			}
			if m.Kind == config.ModelOpenAICompatible {
				buttons(c, locale.AiModelHolds(),
					locale.AiModelHoldsNote(), func() {
						if ui.Button(c.Key("model-holds#"+m.ID),
							locale.AiModelList()).
							Disabled(st.listing).Clicked() {
							s.listWhatItHolds(m.ID)
						}
					})
				s.aiWhatItHolds(c, m.ID)
			}
			if ui.Button(c.Key("model-default#"+m.ID),
				locale.AiModelMakeDefault()).Clicked() {
				m.Enabled = true
				s.draft.cfg.Ai.DefaultModel = m.ID
			}
			s.capabilitiesOf(c, m)
		})
	})
}

// aiWhatItHolds is what the address said it holds: each of them may be
// made the model asked by, which is the whole point of asking.
func (s *appState) aiWhatItHolds(c *ui.Context, id string) {
	th := c.Theme()
	st := s.draft.stateOf(id)
	switch {
	case st.listing:
		ui.Text(c, locale.AiModelAsking()).TextColor(
			th.TextMuted).FontSize(11)
	case st.listErr != nil:
		ui.Text(c, locale.AiModelNotSaid(st.listErr.Error())).
			TextColor(th.Danger).FontSize(11)
	case len(st.held) == 0:
		return
	}
	ui.Row(c).Gap(th.Space(2)).Children(func() {
		for _, one := range st.held {
			if ui.Button(c.Key("held#"+id+"#"+one), one).
				FontSize(11).Clicked() &&
				s.draft.renameModel(id, one) {
				st.idField = one
			}
		}
	})
}

// aiWhich is which model is asked, and how much it may do before it
// answers.
func (s *appState) aiWhich(c *ui.Context) {
	names := make([]string, 0, len(s.draft.cfg.Ai.Models)+1)
	names = append(names, "")
	for i := range s.draft.cfg.Ai.Models {
		if !s.draft.cfg.Ai.Models[i].Enabled {
			continue
		}
		names = append(names, s.draft.cfg.Ai.Models[i].ID)
	}
	group(c, locale.AiGroupWhich(), func() {
		choice(c, locale.AiSettingDefaultModel(),
			locale.AiSettingDefaultModelNote(),
			&s.draft.cfg.Ai.DefaultModel, names)
		choice(c, locale.AiSettingFallbackModel(),
			locale.AiSettingFallbackModelNote(),
			&s.draft.cfg.Ai.FallbackModel, names)
		toggle(c, locale.AiSettingAgentic(),
			locale.AiSettingAgenticNote(), &s.draft.cfg.Ai.Agentic)
		toggle(c, locale.AiSettingSkills(),
			locale.AiSettingSkillsNote(),
			&s.draft.cfg.Ai.SkillsEnabled)
	})
}

// aiAsking is how a question is asked: which tongue the answer comes
// in, how long it may take, how often it is tried again when it fails,
// how far the model may go looking through the vault, and how much of a
// note and of a tool's answer it is shown.
//
// Every one of these takes effect when a model is asked, and until now
// none of them could be reached from this window at all: they lived in
// `config.json` and nowhere else, which is another way of saying the
// app had settings the user was not given.
//
// The two ends of each are the two ends the settings themselves hold
// (`AiConfig.Sanitize` clamps to these): a window that offers a number
// the settings then refuse is a window that lies.
func (s *appState) aiAsking(c *ui.Context) {
	group(c, locale.AiGroupAsking(), func() {
		choice(c, locale.AiSettingOutputLanguage(),
			locale.AiSettingOutputLanguageNote(),
			&s.draft.cfg.Ai.OutputLanguage, outputLanguages)
		number(c, locale.AiSettingTimeout(),
			locale.AiSettingTimeoutNote(), &s.draft.timeout, 5, 600)
		number(c, locale.AiSettingFirstByte(),
			locale.AiSettingFirstByteNote(), &s.draft.firstByte, 5,
			600)
		number(c, locale.AiSettingTries(),
			locale.AiSettingTriesNote(), &s.draft.tries, 1, 5)
		number(c, locale.AiSettingBackoff(),
			locale.AiSettingBackoffNote(), &s.draft.backoff, 100,
			10_000)
		number(c, locale.AiSettingSteps(),
			locale.AiSettingStepsNote(), &s.draft.steps, 1, 64)
		number(c, locale.AiSettingNoteChars(),
			locale.AiSettingNoteCharsNote(), &s.draft.ctxChars,
			1_000, 200_000)
		number(c, locale.AiSettingToolChars(),
			locale.AiSettingToolCharsNote(), &s.draft.toolChars,
			500, 20_000)
	})
}

// cloneModel makes a model into two: the copy is the same model under a
// name of its own, and stands next to the one it was taken of.
//
// What is worth copying is worth copying whole: see
// `config.ModelEntry.Clone` for what is shared and what is not.
func (s *appState) cloneModel(id string) {
	list := s.draft.cfg.Ai.Models
	at := -1
	for i := range list {
		if list[i].ID == id {
			at = i
			break
		}
	}
	if at < 0 {
		return
	}
	name := id + "-copy"
	for n := 2; named(list, name); n++ {
		name = id + "-copy" + strconv.Itoa(n)
	}
	out := make([]config.ModelEntry, 0, len(list)+1)
	out = append(out, list[:at+1]...)
	out = append(out, list[at].Clone(name))
	out = append(out, list[at+1:]...)
	s.draft.cfg.Ai.Models = out
}

// moveModel moves a model earlier or later in the list, by one place.
//
// The order is not only the order they are shown in: when no model has
// been named as the one to ask, the first one that is turned on is the
// one that is asked.
func (s *appState) moveModel(id string, by int) {
	list := s.draft.cfg.Ai.Models
	for i := range list {
		if list[i].ID != id {
			continue
		}
		to := i + by
		if to < 0 || to >= len(list) {
			return
		}
		list[i], list[to] = list[to], list[i]
		return
	}
}

// renameModel gives a model another name, and follows it everywhere it
// has been named: a default, or a way out, or a route, that points at the
// old name is one that points at nothing at all.
//
// A name another model already answers to is not taken: two models of one
// name is one model the app cannot tell from the other.
func (st *settings) renameModel(from, to string) bool {
	to = strings.TrimSpace(to)
	if to == "" || to == from {
		return false
	}
	if named(st.cfg.Ai.Models, to) {
		return false
	}
	for i := range st.cfg.Ai.Models {
		if st.cfg.Ai.Models[i].ID != from {
			continue
		}
		if st.cfg.Ai.Models[i].Label == from {
			st.cfg.Ai.Models[i].Label = to
		}
		st.cfg.Ai.Models[i].ID = to
	}
	if st.cfg.Ai.DefaultModel == from {
		st.cfg.Ai.DefaultModel = to
	}
	if st.cfg.Ai.FallbackModel == from {
		st.cfg.Ai.FallbackModel = to
	}
	for _, one := range []**string{
		&st.cfg.Ai.Route.Chat, &st.cfg.Ai.Route.Action,
		&st.cfg.Ai.Route.Skill, &st.cfg.Ai.Route.Agent,
	} {
		if *one != nil && **one == from {
			name := to
			*one = &name
		}
	}
	for _, act := range st.cfg.Ai.Actions {
		if act.Model != nil && *act.Model == from {
			name := to
			act.Model = &name
		}
	}
	if st.models != nil {
		if kept, there := st.models[from]; there {
			delete(st.models, from)
			st.models[to] = kept
		}
	}
	return true
}

// lookForOllamaModels asks the local Ollama what it holds. It is asked
// away from the window: a machine being started up takes a while to
// answer, and the window is not to wait for it.
func (s *appState) lookForOllamaModels() {
	found := &s.draft.ollama
	found.looking = true
	found.err = nil
	url := s.draft.cfg.Ai.Ollama.BaseURL
	go func() {
		models, err := ai.OllamaModels(context.Background(), url)
		mygo.RunOnMain(func() {
			found.looking = false
			found.models = models
			found.err = err
			found.at = time.Now()
		})
	}()
}

// ollamaOn is whether a model the machine holds is one the settings ask,
// and whether it is one of the settings at all. A model of the same name
// that the user wrote in by hand is not the machine's, and is left out.
func (st *settings) ollamaOn(name string) (on, there bool) {
	for i := range st.cfg.Ai.Models {
		m := &st.cfg.Ai.Models[i]
		if m.ID != name || !m.FromOllama() {
			continue
		}
		return m.Enabled, true
	}
	return false, false
}

// addOllamaModel takes a model the machine holds into the settings: what
// it can do comes from the machine, and its address is left unsaid, so
// that it follows the one written above and not one of its own.
func (st *settings) addOllamaModel(m ai.OllamaModel) {
	for i := range st.cfg.Ai.Models {
		one := &st.cfg.Ai.Models[i]
		if one.ID != m.Name {
			continue
		}
		one.Enabled = true
		one.Source = config.ModelSourceOllama
		if one.CapabilitiesPinned {
			return
		}
		one.Capabilities = append([]string(nil), m.Capabilities...)
		one.ToolsEnabled = holds(m.Capabilities, ai.CapTools)
		if m.ContextLimit > 0 {
			limit := m.ContextLimit
			one.ContextLimit = &limit
		}
		return
	}
	entry := config.ModelEntry{
		ID:            m.Name,
		Source:        config.ModelSourceOllama,
		Kind:          config.ModelOllama,
		Label:         m.Name,
		Enabled:       true,
		Params:        config.DefaultModelParams(),
		Capabilities:  append([]string(nil), m.Capabilities...),
		ToolsEnabled:  holds(m.Capabilities, ai.CapTools),
		LastCheckedMs: uint64(time.Now().UnixMilli()),
	}
	if m.ContextLimit > 0 {
		limit := m.ContextLimit
		entry.ContextLimit = &limit
	}
	st.cfg.Ai.Models = append(st.cfg.Ai.Models, entry)
}

// named is whether a model of that name is one of them.
func named(list []config.ModelEntry, id string) bool {
	for i := range list {
		if list[i].ID == id {
			return true
		}
	}
	return false
}

// forgetModel takes a model out of the settings, and out of the two
// places where it may have been named as the one to ask: a default that
// points at nothing is worse than none at all.
func (st *settings) forgetModel(id string) {
	kept := make([]config.ModelEntry, 0, len(st.cfg.Ai.Models))
	for i := range st.cfg.Ai.Models {
		if st.cfg.Ai.Models[i].ID != id {
			kept = append(kept, st.cfg.Ai.Models[i])
		}
	}
	st.cfg.Ai.Models = kept
	if st.cfg.Ai.DefaultModel == id {
		st.cfg.Ai.DefaultModel = ""
	}
	if st.cfg.Ai.FallbackModel == id {
		st.cfg.Ai.FallbackModel = ""
	}
	delete(st.models, id)
}

// newModel is a model that is not one yet: a name it is known by, and
// the rest left to be filled in. It is one of the user's own, and not one
// the machine gave: those are found by asking the machine.
func (s *appState) newModel() config.ModelEntry {
	id := "model"
	for n := 2; named(s.draft.cfg.Ai.Models, id); n++ {
		id = "model" + strconv.Itoa(n)
	}
	return config.ModelEntry{
		ID: id, Label: id, Source: config.ModelSourceCustom,
		Kind: config.ModelOpenAICompatible, Enabled: true,
		ApiKeySource: config.KeyPlain,
		Params:       config.DefaultModelParams(),
	}
}

// capabilitiesOf is what a model is thought to be able to do. Checking
// one of them is saying it by hand, and from then on what the user said
// is what is believed: the guess is no longer asked.
func (s *appState) capabilitiesOf(c *ui.Context, m *config.ModelEntry) {
	group(c, locale.AiGroupAbilities(), func() {
		if m.CapabilitiesPinned {
			ui.Text(c, locale.AiCapabilityPinned()).TextColor(
				c.Theme().TextMuted).FontSize(11)
		} else {
			ui.Text(c, locale.AiCapabilityTested()).TextColor(
				c.Theme().TextMuted).FontSize(11)
		}
		for _, id := range ai.Known {
			info := ai.Info(id)
			has := holds(ai.Effective(*m), id)
			use := has
			toggle(c, info.Label, info.Meaning+" · "+info.Usage, &use)
			if use == has {
				continue
			}
			// The hand has been laid on it.
			now := ai.Effective(*m)
			if use {
				now = append(now, id)
			} else {
				kept := make([]string, 0, len(now))
				for _, one := range now {
					if one != id {
						kept = append(kept, one)
					}
				}
				now = kept
			}
			m.Capabilities = now
			m.CapabilitiesPinned = true
		}
		// A capability the app has no name for — and the vendors do
		// make them up — is shown as it stands, and may be dropped.
		unknown := []string{}
		for _, one := range m.Capabilities {
			if !holds(ai.Known, one) {
				unknown = append(unknown, one)
			}
		}
		for _, one := range unknown {
			ui.Row(c).Children(func() {
				ui.Text(c, locale.AiModelCapabilityUnknown(one,
					ai.Info(one).Label)).Grow(1)
				if ui.Button(c.Key("drop-cap#"+m.ID+"#"+one),
					locale.AiModelForget()).Clicked() {
					kept := make([]string, 0, len(m.Capabilities))
					for _, other := range m.Capabilities {
						if other != one {
							kept = append(kept, other)
						}
					}
					m.Capabilities = kept
					m.CapabilitiesPinned = true
				}
			})
		}
	})
}

// saidOf is what trying a model out came to, said in a line.
func (s *appState) saidOf(p *ai.Probe) string {
	took := p.Took.Round(time.Millisecond).String()
	if !p.OK {
		if p.Err != nil {
			return locale.AiProbeUnreachableWhy(p.Err.Error())
		}
		return locale.AiProbeUnreachable()
	}
	if p.Tools {
		return locale.AiProbeAnswersTools(took)
	}
	return locale.AiProbeAnswersNoTools(took)
}

// tryModel asks a model one small question, and one tool it may call, to
// find out what it can do. It is asked away from the window: a model is
// given a while to answer, and the window is not to wait for it.
func (s *appState) tryModel(id string) {
	st := s.draft.stateOf(id)
	st.trying = true
	st.probe = nil
	cfg := s.draft.cfg.Ai
	var entry config.ModelEntry
	for i := range cfg.Models {
		if cfg.Models[i].ID == id {
			entry = cfg.Models[i]
		}
	}
	go func() {
		got := ai.Try(context.Background(), cfg, entry, true)
		mygo.RunOnMain(func() {
			st.trying = false
			st.probe = &got
			if !got.OK {
				return
			}
			// What trying found out is the one thing worth keeping:
			// a model that declares tools and then answers 500 when
			// they are sent is not one that has them.
			for i := range s.draft.cfg.Ai.Models {
				if s.draft.cfg.Ai.Models[i].ID != id {
					continue
				}
				m := &s.draft.cfg.Ai.Models[i]
				m.ToolsEnabled = got.Tools
				m.LastCheckedMs = uint64(time.Now().UnixMilli())
				if !m.CapabilitiesPinned {
					m.Capabilities = ai.Measured(got,
						ai.Effective(*m))
				}
			}
		})
	}()
}

// listWhatItHolds asks a model's address which models it holds, so that
// one of them may be asked by its right name. It is asked away from the
// window, as trying a model out is.
func (s *appState) listWhatItHolds(id string) {
	st := s.draft.stateOf(id)
	st.listing = true
	st.held = nil
	st.listErr = nil
	cfg := s.draft.cfg.Ai
	var entry config.ModelEntry
	for i := range cfg.Models {
		if cfg.Models[i].ID == id {
			entry = cfg.Models[i]
		}
	}
	go func() {
		names, err := ai.OpenAIModels(context.Background(), entry, cfg)
		mygo.RunOnMain(func() {
			st.listing = false
			st.held = names
			st.listErr = err
		})
	}()
}

// aiRow is one row of the AI's settings: what it is on the near side,
// what may be done with it on the far side, and a line under it setting
// it apart from the row below, as the other settings rows have.
func aiRow(c *ui.Context, fn func()) {
	th := c.Theme()
	ui.Column(c).FillWidth().Children(func() {
		ui.Column(c).FillWidth().Gap(th.Space(1)).
			Padding(th.Space(2.5), settingsRowPad).Children(fn)
		ui.Box(c).FillWidth().Height(1).Background(th.Border)
	})
}

// capabilityChips is what a model can do, one small label a piece. One
// that was guessed at rather than read off the server is marked with a
// question: an older server does not say.
func capabilityChips(c *ui.Context, caps []string, guessed bool) {
	th := c.Theme()
	if len(caps) == 0 {
		ui.Text(c, locale.AiCapabilityNone()).TextColor(
			th.TextMuted).FontSize(11)
		return
	}
	ui.Row(c).Gap(th.Space(1)).Children(func() {
		for _, one := range caps {
			label := ai.Info(one).Label
			if guessed {
				label = locale.AiCapabilityGuessed(label)
			}
			ui.Badge(c, label).FontSize(11)
		}
	})
}

// madeOf is what a model is made of, said in a few words: its family,
// how many knobs it has, and how finely they are set.
func madeOf(m ai.OllamaModel) string {
	out := []string{}
	if m.Family != "" {
		out = append(out, m.Family)
	}
	if m.ParameterSize != "" {
		out = append(out, m.ParameterSize)
	}
	if m.Quantization != "" {
		out = append(out, m.Quantization)
	}
	return strings.Join(out, " · ")
}

// sizeOf is how much room a model takes, said in words a person can
// read: a model is hundreds of millions of bytes, and that number says
// nothing to anybody.
func sizeOf(n uint64) string {
	switch {
	case n == 0:
		return ""
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.0f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	default:
		return strconv.FormatUint(n, 10) + " B"
	}
}

// keyRingOf is the key of a model and the keyring of the system: it may
// be kept there instead of being written in the settings, and a key kept
// there is written to disk nowhere at all.
//
// Putting it there is not done in the window: the system is asked, and
// may ask the user besides, and a window is not to wait for that.
func (s *appState) keyRingOf(c *ui.Context, m *config.ModelEntry) {
	st := s.draft.stateOf(m.ID)
	kept := m.ApiKeySource == config.KeyKeychain &&
		strings.TrimSpace(m.PlainKey) == ""
	ui.Row(c).Gap(6).Children(func() {
		if kept {
			ui.Text(c, locale.AiModelKeyKept()).TextColor(
				c.Theme().TextMuted).FontSize(11)
			if ui.Button(c.Key("key-forget#"+m.ID),
				locale.AiModelKeyTakeOut()).Label(
				locale.AiModelKeyTakeOutLabel()).Clicked() {
				s.forgetKey(m.ID)
			}
			return
		}
		if ui.Button(c.Key("key-keep#"+m.ID),
			locale.AiModelKeyKeep()).Label(
			locale.AiModelKeyKeepLabel()).
			Disabled(strings.TrimSpace(m.PlainKey) == "").Clicked() {
			s.keepKey(m.ID)
		}
	})
	if st.keyNote != "" {
		ui.Text(c, st.keyNote).TextColor(c.Theme().Danger).FontSize(11)
	}
}

// keepKey puts the key of that model into the keyring of the system, and
// out of the settings.
//
// A key the system would not take is left where it was: a key half moved
// is worse than one written in the settings.
func (s *appState) keepKey(id string) {
	st := s.draft.stateOf(id)
	st.keyNote = ""
	key := ""
	for i := range s.draft.cfg.Ai.Models {
		m := &s.draft.cfg.Ai.Models[i]
		if m.ID != id {
			continue
		}
		key = strings.TrimSpace(m.PlainKey)
		if key == "" {
			return
		}
		if m.KeyRef == "" {
			m.KeyRef = newKeyRef()
		}
	}
	if key == "" {
		return
	}
	ref := s.draft.keyRefOf(id)
	go func() {
		err := secret.Put(ref, key)
		mygo.RunOnMain(func() {
			if err != nil {
				slog.Warn("the key could not be kept in the keyring",
					"model", id, "err", err)
				st.keyNote = locale.AiModelKeyringRefused(
					err.Error())
				return
			}
			for i := range s.draft.cfg.Ai.Models {
				m := &s.draft.cfg.Ai.Models[i]
				if m.ID != id {
					continue
				}
				m.ApiKeySource = config.KeyKeychain
				m.PlainKey = ""
			}
		})
	}()
}

// forgetKey takes the key of that model out of the keyring: what is
// asked with from then on is nothing, until a key is written again.
func (s *appState) forgetKey(id string) {
	ref := s.draft.keyRefOf(id)
	st := s.draft.stateOf(id)
	st.keyNote = ""
	if ref == "" {
		return
	}
	go func() {
		err := secret.Drop(ref)
		mygo.RunOnMain(func() {
			if err != nil {
				slog.Warn("the key could not be taken out of the "+
					"keyring", "model", id, "err", err)
				st.keyNote = locale.AiModelKeyringKeeps(
					err.Error())
				return
			}
			for i := range s.draft.cfg.Ai.Models {
				m := &s.draft.cfg.Ai.Models[i]
				if m.ID != id {
					continue
				}
				m.ApiKeySource = config.KeyPlain
				m.PlainKey = ""
				m.KeyRef = ""
			}
		})
	}()
}

// keyRefOf is the name the key of that model is kept under.
func (st *settings) keyRefOf(id string) string {
	for i := range st.cfg.Ai.Models {
		if st.cfg.Ai.Models[i].ID == id {
			return st.cfg.Ai.Models[i].KeyRef
		}
	}
	return ""
}

// newKeyRef is a name for a key to be kept under: one made up, so that a
// model called by a new name keeps the key it had, and so that the name
// says nothing of the key itself.
func newKeyRef() string {
	var made [8]byte
	if _, err := rand.Read(made[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(made[:])
}

// without is the list with that name taken out of it.
func without(list []string, name string) []string {
	kept := make([]string, 0, len(list))
	for _, one := range list {
		if one != name {
			kept = append(kept, one)
		}
	}
	return kept
}

// holds is whether the list holds it.
func holds(list []string, want string) bool {
	for _, one := range list {
		if one == want {
			return true
		}
	}
	return false
}
