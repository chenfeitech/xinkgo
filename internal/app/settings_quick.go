package app

import (
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
)

// aiQuickWords is the words the user types in the place to ask under a
// note: what each is called, what it asks of the model at length, and
// what is done with the answer.
//
// They are the app's own actions over a note, and have been since before
// this window could say so: what is new is that they may be written,
// changed and forgotten here, rather than in `config.json` by hand.
func (s *appState) aiQuickWords(c *ui.Context) {
	th := c.Theme()
	group(c, locale.AiQuickGroup(), func() {
		ui.Text(c, locale.AiQuickGroupNote()).FontSize(th.FontSize - 1).
			TextColor(th.TextMuted)
		list := s.draft.cfg.Ai.Actions
		if len(list) == 0 {
			ui.Text(c, locale.AiQuickNone()).FontSize(th.FontSize - 1).
				TextColor(th.TextMuted)
		}
		for i := range list {
			s.aiQuickWord(c, i, i == 0, i == len(list)-1)
		}
		ui.Row(c).Gap(th.Space(2)).Children(func() {
			if ui.Button(c.Key("quick-add"), locale.AiQuickAdd()).
				Clicked() {
				s.draft.cfg.Ai.Actions = append(s.draft.cfg.Ai.Actions,
					s.draft.newQuickWord())
			}
			if ui.Button(c.Key("quick-builtins"),
				locale.AiQuickBuiltins()).Clicked() {
				s.draft.putBuiltinQuickWords()
			}
		})
	})
	// Forgetting and moving are done after the rows are drawn, as with
	// the models: a row is not taken out from under the one that
	// follows it.
	if s.draft.quickDrop != "" {
		s.draft.forgetQuickWord(s.draft.quickDrop)
		s.draft.quickDrop = ""
	}
	if s.draft.quickMoveID != "" {
		s.draft.moveQuickWord(s.draft.quickMoveID, s.draft.quickMoveBy)
		s.draft.quickMoveID = ""
		s.draft.quickMoveBy = 0
	}
}

// aiQuickWord is one word: what it is called, what it is typed as, what
// it asks of the model, what of the note it is done to, and what is done
// with the answer. Only the row that is open says the last of these.
func (s *appState) aiQuickWord(c *ui.Context, at int, first, last bool) {
	th := c.Theme()
	a := &s.draft.cfg.Ai.Actions[at]
	open := s.draft.quickOpen == a.ID
	aiRow(c, func() {
		ui.Row(c).FillWidth().Gap(th.Space(3)).AlignItems(ui.Center).
			Children(func() {
				ui.Checkbox(c, &a.Enable, locale.AiQuickEnabled())
				ui.Text(c, a.Name).Bold()
				if a.Pinned {
					ui.Badge(c, locale.AiQuickPinnedBadge()).
						FontSize(11)
				}
				ui.Text(c, quickTyped(*a)).TextColor(th.TextMuted).
					FontSize(11).Grow(1)
				if ui.Button(c.Key("quick-up#"+a.ID),
					locale.AiModelEarlier()).Label(
					locale.AiModelEarlierLabel()).Disabled(
					first).Clicked() {
					s.draft.quickMoveID = a.ID
					s.draft.quickMoveBy = -1
				}
				if ui.Button(c.Key("quick-down#"+a.ID),
					locale.AiModelLater()).Label(
					locale.AiModelLaterLabel()).Disabled(
					last).Clicked() {
					s.draft.quickMoveID = a.ID
					s.draft.quickMoveBy = 1
				}
				label := locale.AiQuickOpen()
				if open {
					label = locale.AiQuickShut()
				}
				if ui.Button(c.Key("quick-open#"+a.ID), label).Clicked() {
					if open {
						s.draft.quickOpen = ""
					} else {
						s.draft.quickOpen = a.ID
					}
				}
				if ui.Button(c.Key("quick-drop#"+a.ID),
					locale.AiQuickForget()).Clicked() {
					s.draft.quickDrop = a.ID
				}
			})
		if !open {
			return
		}
		ui.Column(c).FillWidth().Children(func() {
			text(c, locale.AiQuickWord(), locale.AiQuickWordNote(),
				&a.Name, "")
			text(c, locale.AiQuickTyped(), locale.AiQuickTypedNote(),
				&a.Trigger, a.Word())
			s.aiQuickPrompt(c, a)
			toggle(c, locale.AiQuickPinned(),
				locale.AiQuickPinnedNote(), &a.Pinned)
			text(c, locale.AiQuickCategory(),
				locale.AiQuickCategoryNote(), &a.Category, "Writing")
			text(c, locale.AiQuickScene(),
				locale.AiQuickSceneNote(), &a.Scene, "Selection")
			choiceNamed(c, locale.AiQuickScope(),
				locale.AiQuickScopeNote(), &a.Scope,
				[]string{
					config.ActionScopeSelection, config.ActionScopeWhole,
				}, quickScopeName)
			choiceNamed(c, locale.AiQuickApply(),
				locale.AiQuickApplyNote(), &a.Apply,
				[]string{
					config.ActionApplyReplace, config.ActionApplyInsert,
				}, quickApplyName)
			s.aiQuickModel(c, a)
		})
	})
}

// aiQuickPrompt is what a word asks, written in a field of its own the
// width of the window: what is written there is longer than a line, and
// is the whole of what the word is for.
func (s *appState) aiQuickPrompt(c *ui.Context, a *config.AiAction) {
	th := c.Theme()
	ui.Column(c).FillWidth().Gap(th.Space(1)).
		Padding(th.Space(2.5), settingsRowPad).Children(func() {
		ui.Text(c, locale.AiQuickPrompt())
		ui.Text(c, locale.AiQuickPromptNote()).
			FontSize(th.FontSize - 1).TextColor(th.TextMuted)
		ui.TextArea(c, &a.Prompt).FillWidth().
			MinHeight(th.Space(16))
	})
	ui.Box(c).FillWidth().Height(1).Background(th.Border)
}

// aiQuickModel is which model a word is asked of: the one the place to
// ask names, unless the word names one of its own, as a word that asks
// for a title may well be given to a smaller model than one that
// rewrites the whole of a note.
func (s *appState) aiQuickModel(c *ui.Context, a *config.AiAction) {
	names := []string{""}
	for i := range s.draft.cfg.Ai.Models {
		if s.draft.cfg.Ai.Models[i].Enabled {
			names = append(names, s.draft.cfg.Ai.Models[i].ID)
		}
	}
	model := ""
	if a.Model != nil {
		model = *a.Model
	}
	// A model taken out of the settings since is not one to ask: the
	// word falls back to the one the place to ask names.
	if !slices.Contains(names, model) {
		model = ""
	}
	choiceNamed(c, locale.AiQuickModel(), locale.AiQuickModelNote(),
		&model, names, quickModelName)
	if model == "" {
		a.Model = nil
		return
	}
	named := model
	a.Model = &named
}

// newQuickWord is a word of one's own, turned on and doing to the lines
// chosen what any word does, with nothing written in it yet.
func (st *settings) newQuickWord() config.AiAction {
	for n := 1; ; n++ {
		id := "word-" + strconv.Itoa(n)
		if !quickNamed(st.cfg.Ai.Actions, id) {
			return config.AiAction{
				ID:     id,
				Scope:  config.ActionScopeSelection,
				Apply:  config.ActionApplyReplace,
				Enable: true,
			}
		}
	}
}

// putBuiltinQuickWords puts the words the app ships with back among the
// user's own, the ones that are not there already: what the user wrote
// is theirs, and is not taken away to make room for them.
func (st *settings) putBuiltinQuickWords() {
	for _, one := range config.DefaultActions() {
		if quickNamed(st.cfg.Ai.Actions, one.ID) {
			continue
		}
		st.cfg.Ai.Actions = append(st.cfg.Ai.Actions, one)
	}
}

// forgetQuickWord takes a word out of the settings for good.
func (st *settings) forgetQuickWord(id string) {
	st.cfg.Ai.Actions = slices.DeleteFunc(st.cfg.Ai.Actions,
		func(one config.AiAction) bool { return one.ID == id })
	if st.quickOpen == id {
		st.quickOpen = ""
	}
}

// moveQuickWord moves a word earlier or later among them, by one place:
// the order is the order the picker offers them in.
func (st *settings) moveQuickWord(id string, by int) {
	list := st.cfg.Ai.Actions
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

// quickNamed is whether a word of that name is one of them.
func quickNamed(list []config.AiAction, id string) bool {
	for i := range list {
		if list[i].ID == id {
			return true
		}
	}
	return false
}

// quickTyped is a word as the user types it to run it: "/" and then the
// word itself, as it is typed into the place to ask.
func quickTyped(a config.AiAction) string {
	return "/" + strings.TrimSpace(a.Word())
}

// What each of the choices of a word is called in the language the app
// shows. They are named after what the configuration keeps, and not
// after the words of any one language.
func quickScopeName(v string) string {
	if v == config.ActionScopeWhole {
		return locale.AiQuickScopeWhole()
	}
	return locale.AiQuickScopeSelection()
}

func quickApplyName(v string) string {
	if v == config.ActionApplyInsert {
		return locale.AiQuickApplyInsert()
	}
	return locale.AiQuickApplyReplace()
}

func quickModelName(v string) string {
	if v == "" {
		return locale.AiQuickModelDefault()
	}
	return v
}
