package app

import (
	"context"
	"strconv"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
)

// modelState is what one model's row keeps between two frames: whether
// it is open, and the last thing trying it out said.
type modelState struct {
	open   bool
	trying bool
	probe  *ai.Probe
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

// ai is the settings of the AI: whether it is used at all, and the models
// it may be asked.
func (s *appState) ai(c *ui.Context) {
	group(c, "AI", func() {
		ui.Checkbox(c, &s.draft.cfg.Ai.Enabled, "Ask a model at all")
		if !s.draft.cfg.Ai.Enabled {
			ui.Text(c, "Turning this on is what lets a note be asked "+
				"about.").TextColor(c.Theme().TextMuted)
			return
		}
		s.aiServer(c)
		s.aiModels(c)
		s.aiAsking(c)
		s.aiSkills(c)
	})
}

// aiSkills is the skills found in the vault and in the application's own
// folder: each may be turned off, and one that was found but cannot be
// used is said so rather than left out.
func (s *appState) aiSkills(c *ui.Context) {
	ui.Column(c).Gap(4).Children(func() {
		ui.Text(c, "Skills").Bold()
		if len(s.skills) == 0 && len(s.skillIssues) == 0 {
			ui.Text(c, "No skills yet：write one at <vault>/.xink/"+
				"skills/<name>/SKILL.md").TextColor(
				c.Theme().TextMuted).FontSize(11)
		}
		for _, one := range s.skills {
			one := one
			off := holds(s.draft.cfg.Ai.DisabledSkills, one.Name)
			use := !off
			ui.Row(c).Children(func() {
				ui.Checkbox(c, &use, one.Name).Label(one.Description)
				ui.Text(c, one.From.Label()).TextColor(
					c.Theme().TextMuted).FontSize(11)
			})
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
			ui.Text(c, issue.Path+"："+issue.Reason).TextColor(
				c.Theme().Danger).FontSize(11)
		}
	})
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

// aiServer is where a local Ollama is, and which model is asked when
// nothing else is said, and which one is asked when that one cannot
// answer.
func (s *appState) aiServer(c *ui.Context) {
	names := make([]string, 0, len(s.draft.cfg.Ai.Models)+1)
	names = append(names, "")
	for i := range s.draft.cfg.Ai.Models {
		names = append(names, s.draft.cfg.Ai.Models[i].ID)
	}
	ui.Column(c).Gap(4).Children(func() {
		ui.Row(c).Children(func() {
			ui.Text(c, "Local Ollama").Grow(1)
			ui.TextInput(c, &s.draft.cfg.Ai.Ollama.BaseURL).
				Placeholder("http://localhost:11434")
		})
		ui.Row(c).Children(func() {
			ui.Text(c, "Keep it loaded for").Grow(1)
			ui.TextInput(c, &s.draft.cfg.Ai.Ollama.KeepAlive).
				Placeholder("5m")
		})
		choice(c, "Ask this one", &s.draft.cfg.Ai.DefaultModel, names)
		choice(c, "And if it cannot answer",
			&s.draft.cfg.Ai.FallbackModel, names)
		ui.Checkbox(c, &s.draft.cfg.Ai.Agentic,
			"Let it look through the vault before it answers")
		ui.Checkbox(c, &s.draft.cfg.Ai.SkillsEnabled,
			"Let it use the skills written in the vault")
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
	group(c, "Asking", func() {
		choice(c, "Answer in", &s.draft.cfg.Ai.OutputLanguage,
			outputLanguages)
		number(c, "How long an answer may take (s)", &s.draft.timeout,
			5, 600)
		number(c, "How long before the first of it (s)",
			&s.draft.firstByte, 5, 600)
		number(c, "How often it is tried again", &s.draft.tries, 1, 5)
		number(c, "How long between two tries (ms)", &s.draft.backoff,
			100, 10_000)
		number(c, "How far it may go through the vault", &s.draft.steps,
			1, 64)
		number(c, "How much of a note it is shown (letters)",
			&s.draft.ctxChars, 1_000, 200_000)
		number(c, "How much a tool may answer (letters)",
			&s.draft.toolChars, 500, 20_000)
	})
}

// aiModels is the models the app knows, and the way to add one.
func (s *appState) aiModels(c *ui.Context) {
	ui.Column(c).Gap(6).Children(func() {
		for i := range s.draft.cfg.Ai.Models {
			s.aiModel(c, i)
		}
		if ui.Button(c.Key("add-model"), "Add a model").Clicked() {
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
// the rest left to be filled in.
func (s *appState) newModel() config.ModelEntry {
	id := "model"
	for n := 2; named(s.draft.cfg.Ai.Models, id); n++ {
		id = "model" + strconv.Itoa(n)
	}
	return config.ModelEntry{
		ID: id, Label: id, Kind: config.ModelOllama, Enabled: true,
		Params: config.DefaultModelParams(),
	}
}

// aiModel is one model: its name, where it is asked, and what it is
// thought to be able to do.
func (s *appState) aiModel(c *ui.Context, at int) {
	m := &s.draft.cfg.Ai.Models[at]
	st := s.draft.stateOf(m.ID)
	last := at == len(s.draft.cfg.Ai.Models)-1
	ui.Column(c).Gap(4).Children(func() {
		ui.Row(c).Gap(6).Children(func() {
			ui.Checkbox(c, &m.Enabled, "Turned on")
			ui.Text(c, m.ID).Bold().Grow(1)
			// The order is the order they are asked in: the
			// first one turned on is the one asked when none
			// has been named.
			if ui.Button(c.Key("model-up#"+m.ID), "↑").
				Label("Ask it earlier").Disabled(at == 0).Clicked() {
				s.draft.moveID = m.ID
				s.draft.moveBy = -1
			}
			if ui.Button(c.Key("model-down#"+m.ID), "↓").
				Label("Ask it later").Disabled(last).Clicked() {
				s.draft.moveID = m.ID
				s.draft.moveBy = 1
			}
			open := st.open
			label := "Open"
			if open {
				label = "Shut"
			}
			if ui.Button(c.Key("model-open#"+m.ID), label).Clicked() {
				st.open = !st.open
			}
			if ui.Button(c.Key("model-try#"+m.ID), "Try it").
				Clicked() && !st.trying {
				s.tryModel(m.ID)
			}
			if ui.Button(c.Key("model-clone#"+m.ID), "Clone").
				Label("The same model under another name").Clicked() {
				s.draft.clone = m.ID
			}
			if ui.Button(c.Key("model-drop#"+m.ID), "Forget").
				Clicked() {
				s.draft.drop = m.ID
			}
		})
		if st.probe != nil {
			ui.Text(c, s.saidOf(st.probe)).TextColor(
				c.Theme().TextMuted).FontSize(11)
		} else if st.trying {
			ui.Text(c, "Asking…").TextColor(
				c.Theme().TextMuted).FontSize(11)
		}
		if !st.open {
			return
		}
		ui.Column(c).Gap(4).Children(func() {
			ui.Row(c).Children(func() {
				ui.Text(c, "Name").Grow(1)
				ui.TextInput(c, &m.Label).Placeholder(m.ID)
			})
			ui.Row(c).Children(func() {
				ui.Text(c, "Address").Grow(1)
				ui.TextInput(c, &m.BaseURL).Placeholder(
					"http://localhost:11434")
			})
			ui.Row(c).Children(func() {
				ui.Text(c, "Key").Grow(1)
				ui.TextInput(c, &m.PlainKey).Placeholder("none needed")
			})
			choice(c, "Speaks", &m.Kind, []string{
				config.ModelOllama, config.ModelOpenAICompatible,
			})
			s.capabilitiesOf(c, m)
		})
	})
}

// capabilitiesOf is what a model is thought to be able to do. Checking
// one of them is saying it by hand, and from then on what the user said
// is what is believed: the guess is no longer asked.
func (s *appState) capabilitiesOf(c *ui.Context, m *config.ModelEntry) {
	ui.Column(c).Gap(2).Children(func() {
		ui.Text(c, "What it can do").Bold()
		for _, id := range ai.Known {
			info := ai.Info(id)
			has := holds(ai.Effective(*m), id)
			use := has
			ui.Row(c).Children(func() {
				ui.Checkbox(c, &use, info.Label).Label(info.Meaning)
				ui.Text(c, info.Usage).TextColor(
					c.Theme().TextMuted).FontSize(11)
			})
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
				ui.Text(c, one+"（"+ai.Info(one).Label+"）").Grow(1)
				if ui.Button(c.Key("drop-cap#"+m.ID+"#"+one), "Forget").
					Clicked() {
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
	if !p.OK {
		if p.Err != nil {
			return "Could not be reached：" + p.Err.Error()
		}
		return "Could not be reached"
	}
	out := "It answers（" + p.Took.Round(time.Millisecond).String() + "）"
	if p.Tools {
		out += "，and it calls tools"
	} else {
		out += "，but it calls no tools"
	}
	return out
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
			// What trying found out about tools is the one thing
			// worth keeping: a model that declares them and then
			// answers 500 when they are sent is not one that has
			// them.
			for i := range s.draft.cfg.Ai.Models {
				if s.draft.cfg.Ai.Models[i].ID != id {
					continue
				}
				s.draft.cfg.Ai.Models[i].ToolsEnabled = got.Tools
				s.draft.cfg.Ai.Models[i].LastCheckedMs =
					uint64(time.Now().UnixMilli())
			}
		})
	}()
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
