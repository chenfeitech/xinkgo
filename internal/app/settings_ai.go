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
	})
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
		choice(c, "Ask this one", &s.draft.cfg.Ai.DefaultModel, names)
		choice(c, "And if it cannot answer",
			&s.draft.cfg.Ai.FallbackModel, names)
		ui.Checkbox(c, &s.draft.cfg.Ai.Agentic,
			"Let it look through the vault before it answers")
		ui.Checkbox(c, &s.draft.cfg.Ai.SkillsEnabled,
			"Let it use the skills written in the vault")
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
	// Forgetting is done after the rows are drawn: a row is not taken
	// out from under the one that follows it.
	if s.draft.drop != "" {
		s.draft.forgetModel(s.draft.drop)
		s.draft.drop = ""
	}
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
	for n := 2; ; n++ {
		taken := false
		for i := range s.draft.cfg.Ai.Models {
			if s.draft.cfg.Ai.Models[i].ID == id {
				taken = true
				break
			}
		}
		if !taken {
			break
		}
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
	ui.Column(c).Gap(4).Children(func() {
		ui.Row(c).Gap(6).Children(func() {
			ui.Checkbox(c, &m.Enabled, "Turned on")
			ui.Text(c, m.ID).Bold().Grow(1)
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
