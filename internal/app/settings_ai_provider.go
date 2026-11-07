package app

import (
	"context"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
)

// aiProviderUI is the state of the service list on the AI settings page.
// None of it is written down except the services themselves: a fetched
// model list is a thing of the moment, not a setting.
type aiProviderUI struct {
	draft aiProviderDraft
	lists map[string]*aiProviderList
	open  map[string]bool
	drop  string
}

type aiProviderDraft struct {
	open                            bool
	preset, name, url, key, keyFrom string
}

type aiProviderList struct {
	held []string
	err  string
	busy bool
}

func (u *aiProviderUI) stateOf(id string) *aiProviderList {
	if u.lists == nil {
		u.lists = map[string]*aiProviderList{}
	}
	if u.lists[id] == nil {
		u.lists[id] = &aiProviderList{}
	}
	return u.lists[id]
}

// providerGroup is one service and the positions of its models in the
// flat list, or the models that belong to no service yet.
type providerGroup struct {
	id, name string
	rows     []int
	custom   bool
}

// aiOther is the models the user wrote in, grouped by the service they
// came from. Ollama keeps its own group above; here a service holds the
// address and key once, and its models are fetched in one go.
func (s *appState) aiOther(c *ui.Context) {
	cfg := &s.draft.cfg.Ai
	groups := []providerGroup{}
	for _, p := range cfg.Providers {
		groups = append(groups, providerGroup{id: p.ID, name: p.Name})
	}
	ungrouped := providerGroup{id: "", name: locale.AiProviderUngrouped(),
		custom: true}
	for i := range cfg.Models {
		m := &cfg.Models[i]
		if m.FromOllama() {
			continue
		}
		placed := false
		for g := range groups {
			if groups[g].id != "" && groups[g].id == m.ProviderID {
				groups[g].rows = append(groups[g].rows, i)
				placed = true
				break
			}
		}
		if !placed {
			ungrouped.rows = append(ungrouped.rows, i)
		}
	}
	if len(ungrouped.rows) > 0 {
		groups = append(groups, ungrouped)
	}
	ui.Column(c).Gap(6).Children(func() {
		if len(groups) == 0 {
			ui.Text(c, locale.AiProviderNone()).
				FontSize(c.Theme().FontSize - 1).
				TextColor(c.Theme().TextMuted)
		}
		for _, one := range groups {
			s.aiProviderGroup(c, one)
		}
		if ui.Button(c.Key("add-provider"),
			locale.AiProviderAdd()).Clicked() {
			s.draft.aiProviders.draft = aiProviderDraft{
				open: true, preset: "custom", keyFrom: config.KeyPlain,
			}
		}
		// A model may still be written in entirely by hand.
		if ui.Button(c.Key("add-model"), locale.AiModelsAdd()).Clicked() {
			s.draft.cfg.Ai.Models = append(s.draft.cfg.Ai.Models,
				s.newModel())
		}
	})
	s.aiProviderModal(c)
	// Forgetting, copying and moving are done after the rows are drawn:
	// a row must not be taken out from under the one that follows it.
	if s.draft.aiProviders.drop != "" {
		s.forgetProvider(s.draft.aiProviders.drop)
		s.draft.aiProviders.drop = ""
	}
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

// aiProviderGroup is one service's row: its name, how many models it
// holds, the fetched models waiting to be added, and its model rows.
func (s *appState) aiProviderGroup(c *ui.Context, g providerGroup) {
	th := c.Theme()
	state := &s.draft.aiProviders
	// A group starts open: a model that was on show before services
	// existed must stay on show.
	expanded := true
	if v, ok := state.open[g.id]; ok {
		expanded = v
	}
	aiRow(c, func() {
		ui.Row(c).FillWidth().Gap(th.Space(3)).AlignItems(ui.Center).
			Children(func() {
				ui.Text(c, g.name).Bold().Grow(1)
				ui.Badge(c, providerModelCount(len(g.rows))).
					FontSize(11)
				if !g.custom {
					if ui.Button(c.Key("provider-refresh-"+g.id),
						locale.AiProviderRefresh()).Clicked() {
						s.listProviderModels(g.id)
					}
					if ui.Button(c.Key("provider-drop-"+g.id),
						locale.AiProviderRemove()).Clicked() {
						state.drop = g.id
					}
				}
				// The label is its own, so it cannot be mistaken for
				// a model row's Open button.
				open := expanded
				label := locale.AiProviderShow()
				if open {
					label = locale.AiProviderHide()
				}
				if ui.Button(c.Key("provider-open-"+g.id), label).
					Clicked() {
					if state.open == nil {
						state.open = map[string]bool{}
					}
					state.open[g.id] = !open
				}
			})
		if !expanded {
			return
		}
		if !g.custom {
			st := state.stateOf(g.id)
			switch {
			case st.busy:
				ui.Text(c, locale.AiProviderBusy()).
					FontSize(11).TextColor(th.TextMuted)
			case st.err != "":
				ui.Text(c, locale.AiProviderFailed(st.err)).
					FontSize(11).TextColor(th.Danger)
			case len(st.held) > 0:
				ui.Text(c, locale.AiProviderHeld()).
					FontSize(11).TextColor(th.TextMuted)
				ui.Row(c).FillWidth().Gap(th.Space(1)).Wrap().
					Children(func() {
						for _, name := range st.held {
							name := name
							if ui.Button(c.Key("provider-held-"+
								g.id+"-"+name), name).
								FontSize(11).Clicked() {
								s.addProviderModel(g.id, name)
							}
						}
					})
			}
		}
		for n, at := range g.rows {
			s.aiOtherModel(c, at, n == 0, n == len(g.rows)-1)
		}
	})
}

// forgetProvider takes a service away but keeps its models: they become
// ungrouped hand-written entries rather than disappearing.
func (s *appState) forgetProvider(id string) {
	cfg := &s.draft.cfg.Ai
	kept := make([]config.AiProvider, 0, len(cfg.Providers))
	for _, p := range cfg.Providers {
		if p.ID != id {
			kept = append(kept, p)
		}
	}
	cfg.Providers = kept
	for i := range cfg.Models {
		if cfg.Models[i].ProviderID == id {
			cfg.Models[i].ProviderID = ""
		}
	}
	delete(s.draft.aiProviders.lists, id)
	delete(s.draft.aiProviders.open, id)
}

// providerPresetIDs are the choices of the add-service modal.
func providerPresetIDs() []string {
	ids := []string{"custom"}
	for _, p := range config.ProviderPresets {
		ids = append(ids, p.ID)
	}
	return ids
}

func providerPresetLabel(id string) string {
	if id == "custom" {
		return locale.AiProviderCustom()
	}
	for _, p := range config.ProviderPresets {
		if p.ID == id {
			return p.Name
		}
	}
	return id
}

// aiProviderModal is the one-screen form that adds a service. It asks
// for as little as a preset can supply: the rest is filled in for the
// user, and may still be changed.
func (s *appState) aiProviderModal(c *ui.Context) {
	d := &s.draft.aiProviders.draft
	if !d.open {
		return
	}
	th := c.Theme()
	ui.Modal(c, &d.open, func() {
		ui.Column(c).Width(420).Gap(th.Space(2)).
			Padding(th.Space(3)).Children(func() {
			ui.Text(c, locale.AiProviderAddTitle()).Bold()
			ui.Text(c, locale.AiProviderAddNote()).
				FontSize(th.FontSize - 1).TextColor(th.TextMuted)
			choiceNamed(c, locale.AiProviderPreset(),
				locale.AiProviderPresetNote(), &d.preset,
				providerPresetIDs(), providerPresetLabel)
			// A preset fills the two fields it knows, unless the user
			// has already written something of their own.
			if d.preset != "custom" && strings.TrimSpace(d.url) == "" {
				for _, p := range config.ProviderPresets {
					if p.ID == d.preset {
						d.url = p.BaseURL
						if strings.TrimSpace(d.name) == "" {
							d.name = p.Name
						}
						break
					}
				}
			}
			text(c, locale.AiProviderName(),
				locale.AiProviderNameNote(), &d.name, "OpenAI")
			textField(c, locale.AiModelAddress(), &d.url,
				"https://api.openai.com/v1")
			choice(c, locale.AiModelKeyFrom(),
				locale.AiModelKeyFromNote(), &d.keyFrom,
				[]string{config.KeyPlain, config.KeyEnv})
			text(c, locale.AiProviderKey(),
				locale.AiProviderKeyNote(), &d.key, "")
			ui.Row(c).Gap(th.Space(2)).Children(func() {
				if ui.Button(c.Key("provider-add-save"),
					locale.AiProviderSave()).Clicked() {
					s.commitProvider()
				}
				if ui.Button(c.Key("provider-add-cancel"),
					locale.AiProviderCancel()).Clicked() {
					d.open = false
				}
			})
		})
	})
}

// textField is a plain text row without a note line, for the modal.
func textField(c *ui.Context, name string, value *string,
	placeholder string) {
	text(c, name, "", value, placeholder)
}

// commitProvider adds the drafted service and clears the form.
func (s *appState) commitProvider() {
	d := &s.draft.aiProviders.draft
	name := strings.TrimSpace(d.name)
	address := strings.TrimSpace(d.url)
	if d.preset != "custom" && address == "" {
		for _, p := range config.ProviderPresets {
			if p.ID == d.preset {
				address = p.BaseURL
				if name == "" {
					name = p.Name
				}
				break
			}
		}
	}
	if name == "" || address == "" {
		return
	}
	cfg := &s.draft.cfg.Ai
	id := config.ProviderIDFor(address)
	for _, p := range cfg.Providers {
		if p.ID == id {
			d.open = false
			return
		}
	}
	cfg.Providers = append(cfg.Providers, config.AiProvider{
		ID: id, Name: name, BaseURL: address,
		ApiKeySource: d.keyFrom, PlainKey: d.key,
		ShowInPicker: true,
	})
	*d = aiProviderDraft{}
}

// listProviderModels asks a service which models it holds. The work is
// done off the frame, as every other network ask in the settings is.
func (s *appState) listProviderModels(id string) {
	cfg := s.draft.cfg.Ai
	var p *config.AiProvider
	for i := range cfg.Providers {
		if cfg.Providers[i].ID == id {
			p = &cfg.Providers[i]
			break
		}
	}
	if p == nil {
		return
	}
	st := s.draft.aiProviders.stateOf(id)
	st.busy, st.err, st.held = true, "", nil
	entry := config.ModelEntry{
		BaseURL: p.BaseURL, ApiKeySource: p.ApiKeySource,
		PlainKey: p.PlainKey, KeyRef: p.KeyRef,
		Kind: config.ModelOpenAICompatible,
	}
	go func() {
		names, err := ai.OpenAIModels(context.Background(), entry, cfg)
		mygo.RunOnMain(func() {
			st.busy = false
			if err != nil {
				st.err = err.Error()
			} else {
				st.held = names
			}
		})
	}()
}

// addProviderModel turns one fetched model name into a callable model
// of that service, inheriting its address and key.
func (s *appState) addProviderModel(id, name string) {
	cfg := &s.draft.cfg.Ai
	name = strings.TrimSpace(name)
	if name == "" || named(cfg.Models, name) {
		return
	}
	var p *config.AiProvider
	for i := range cfg.Providers {
		if cfg.Providers[i].ID == id {
			p = &cfg.Providers[i]
			break
		}
	}
	if p == nil {
		return
	}
	cfg.Models = append(cfg.Models, config.ModelEntry{
		ID: name, ProviderID: id, Source: config.ModelSourceCustom,
		Kind: config.ModelOpenAICompatible, Label: name,
		BaseURL: p.BaseURL, ApiKeySource: p.ApiKeySource,
		PlainKey: p.PlainKey, KeyRef: p.KeyRef,
		Enabled: true, Params: config.DefaultModelParams(),
	})
	st := s.draft.aiProviders.stateOf(id)
	if len(st.held) > 0 {
		kept := st.held[:0:0]
		for _, one := range st.held {
			if one != name {
				kept = append(kept, one)
			}
		}
		st.held = kept
	}
}

// providerModelCount is the badge on a service row.
func providerModelCount(n int) string {
	return locale.AiProviderModels(strconv.Itoa(n))
}
