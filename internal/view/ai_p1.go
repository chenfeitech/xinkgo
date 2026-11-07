package view

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
)

// aiP1State is the composer state added by the AI upgrade's P1: the
// model picker, the form that fills a prompt's blanks, and the answers
// it gave last. None of it is written down.
type aiP1State struct {
	pickerOpen   bool
	pickerQuery  string
	pickerChosen int

	blankOpen    bool
	blankAction  config.AiAction
	blankExtra   string
	blankNames   []string
	blankValues  map[string]string
	blankAnswers map[string]string
	atOpen       bool
	sessionOpen  bool
}

// aiModelPick is the model chip and its popover. One picker serves every
// source: recent models first, then Ollama and each service, with a
// search box that also accepts a model ID typed by hand.
func (w *Workspace) aiModelPick(c *ui.Context, s *aiSession) {
	if !w.Config.Ai.Enabled {
		return
	}
	names := w.aiModelNames()
	if len(names) == 0 {
		return
	}
	w.AiModel = w.aiModelNamed(names)
	chip := ui.Button(c.Key("ai-model-chip"), w.AiModel).
		Width(aiModelWidth).Tooltip(w.AiModel).Disabled(s.isBusy())
	if chip.Clicked() {
		w.aiP1.pickerOpen = !w.aiP1.pickerOpen
		w.aiP1.pickerQuery = ""
		w.aiP1.pickerChosen = 0
	}
	if !w.aiP1.pickerOpen {
		return
	}
	ui.Popover(c, chip, &w.aiP1.pickerOpen, func() {
		th := c.Theme()
		ui.Column(c).Width(320).ClipY().MaxHeight(300).
			Padding(th.Space(1)).Gap(th.Space(1)).Children(func() {
			field := ui.TextInput(c.Key("ai-model-search"),
				&w.aiP1.pickerQuery).
				Placeholder(locale.AiModelSearch()).Grow(1)
			if field.Submitted() {
				lines := w.modelLines(w.aiP1.pickerQuery)
				if len(lines) > 0 {
					w.pickModel(lines[0].ID)
				}
			}
			lines := w.modelLines(w.aiP1.pickerQuery)
			if len(lines) == 0 {
				ui.Text(c, locale.PaletteNothing()).
					FontSize(11).TextColor(th.TextMuted)
				return
			}
			if w.aiP1.pickerChosen >= len(lines) {
				w.aiP1.pickerChosen = 0
			}
			lastGroup := ""
			for i, one := range lines {
				if one.Group != lastGroup {
					lastGroup = one.Group
					ui.Text(c, lastGroup).Bold().FontSize(11).
						TextColor(th.TextMuted)
				}
				one := one
				row := ui.Button(c.Key("ai-model-choice-"+one.ID),
					one.ID).FontSize(12)
				if i == w.aiP1.pickerChosen {
					row.Selected(true)
				}
				if row.Clicked() {
					w.pickModel(one.ID)
				}
			}
		})
	})
}

// modelLine is one model in the picker, named by the source it came
// from so the list can be grouped.
type modelLine struct {
	ID, Group string
}

// modelLines is the picker's list: recent models first when nothing is
// typed, then Ollama and each service. A query filters by ID across
// every source, and an unknown ID may still be picked by hand.
func (w *Workspace) modelLines(query string) []modelLine {
	groupOf := func(m config.ModelEntry) string {
		if m.FromOllama() {
			return locale.AiModelSourceOllama()
		}
		if m.ProviderID != "" {
			for _, p := range w.Config.Ai.Providers {
				if p.ID == m.ProviderID && p.Name != "" {
					return p.Name
				}
			}
		}
		return locale.AiModelSourceOther()
	}
	all := make([]modelLine, 0, len(w.Config.Ai.Models))
	for _, m := range w.Config.Ai.Models {
		if m.Enabled {
			all = append(all, modelLine{m.ID, groupOf(m)})
		}
	}
	q := strings.ToLower(strings.TrimSpace(query))
	if q != "" {
		out := make([]modelLine, 0, len(all))
		for _, one := range all {
			if strings.Contains(strings.ToLower(one.ID), q) {
				out = append(out, one)
			}
		}
		return out
	}
	// Recent first, then the rest grouped by source in settings order.
	byID := map[string]modelLine{}
	for _, one := range all {
		byID[one.ID] = one
	}
	seen := map[string]bool{}
	out := []modelLine{}
	for _, id := range w.RecentModels {
		if one, ok := byID[id]; ok && !seen[id] {
			seen[id] = true
			out = append(out,
				modelLine{one.ID, locale.AiModelRecent()})
		}
	}
	groups := []string{}
	byGroup := map[string][]modelLine{}
	for _, one := range all {
		if _, ok := byGroup[one.Group]; !ok {
			groups = append(groups, one.Group)
		}
		byGroup[one.Group] = append(byGroup[one.Group], one)
	}
	for _, g := range groups {
		for _, one := range byGroup[g] {
			if !seen[one.ID] {
				seen[one.ID] = true
				out = append(out, one)
			}
		}
	}
	return out
}

func (w *Workspace) pickModel(id string) {
	if !hasName(w.aiModelNames(), id) {
		return
	}
	w.AiModel = id
	w.aiPicked = true
	w.aiP1.pickerOpen = false
}

// syncContexts rebuilds the removable chips when the note or the
// selection changes. A dismissal lasts until then, so a chip the user
// removed does not come back on the next frame.
func (w *Workspace) syncContexts(s *aiSession, n *OpenNote) {
	if n == nil {
		s.Contexts, s.contextKey = nil, ""
		return
	}
	sel := n.Editor.Selection()
	key := n.Path + "|" + strconv.Itoa(sel.Start()) + "|" +
		strconv.Itoa(sel.End())
	if s.contextKey == key {
		return
	}
	s.contextKey = key
	whole := n.Editor.Buffer.String()
	s.Contexts = []aiContext{{
		ID: "note", Kind: "note",
		Label: locale.AiContextNote(PathOf(n.Path)), Text: whole,
	}}
	if !sel.Empty() {
		from, to := sel.Start(), sel.End()
		if from >= 0 && to <= len(whole) && from < to {
			piece := whole[from:to]
			s.Contexts = append(s.Contexts, aiContext{
				ID: "selection", Kind: "selection",
				Label: locale.AiContextSelection(
					strconv.Itoa(len([]rune(piece)))),
				Text: piece,
			})
		}
	}
	// Notes the user attached with @ stay until they are removed or the
	// note changes; they are rebuilt here with the other contexts.
	for _, name := range s.Refs {
		s.Contexts = append(s.Contexts, aiContext{
			ID: "ref-" + name, Kind: "ref",
			Label: locale.AiAtAttached(name),
			Text:  s.RefTexts[name],
		})
	}
}

// contextStrip is the removable chips above the composer.
func (w *Workspace) contextStrip(c *ui.Context, s *aiSession) {
	if len(s.Contexts) == 0 {
		return
	}
	th := c.Theme()
	ui.Row(c).FillWidth().Gap(th.Space(1)).Wrap().Children(func() {
		for i := range s.Contexts {
			one := &s.Contexts[i]
			if one.Removed {
				continue
			}
			ui.Badge(c, one.Label).FontSize(11)
			if ui.Button(c.Key("ctx-remove-"+one.ID), "x").
				FontSize(11).Tooltip(locale.AiContextRemove()).
				Clicked() {
				one.Removed = true
			}
		}
	})
}

// starterStrip is the empty conversation's first screen. Its heading and
// suggestions follow what is on screen, so nothing is random between
// frames.
func (w *Workspace) starterStrip(
	c *ui.Context, s *aiSession, n *OpenNote,
) {
	// The first screen stands only while the conversation is empty. Its
	// shape must not change while a slash word is typed, or the input
	// would lose its focus mid-word.
	if len(s.blocks()) > 0 {
		return
	}
	th := c.Theme()
	title := locale.AiStarterTitle()
	if n == nil {
		title = locale.AiStarterEmpty()
	} else if !n.Editor.Selection().Empty() {
		title = locale.AiStarterSelection()
	}
	ui.Column(c).FillWidth().Gap(th.Space(1)).Children(func() {
		ui.Text(c, title).Bold()
		ui.Text(c, locale.AiStarterHint()).
			FontSize(11).TextColor(th.TextMuted)
		// The suggestions are the quick strip itself: drawing a second
		// row of the same actions would compete with it for the keys,
		// and a slash typed to pick among them would be lost.
	})
}

// blankForm is the small form that fills a prompt's {{blanks}} before
// it runs. Running is separate from editing: a chip sends at once, and
// only a prompt with blanks waits for these fields.
func (w *Workspace) blankForm(
	c *ui.Context, s *aiSession, n *OpenNote,
) {
	if !w.aiP1.blankOpen {
		return
	}
	th := c.Theme()
	ui.Modal(c, &w.aiP1.blankOpen, func() {
		ui.Column(c).Width(360).Gap(th.Space(2)).
			Padding(th.Space(3)).Children(func() {
			ui.Text(c, locale.AiBlankTitle()).Bold()
			for _, name := range w.aiP1.blankNames {
				name := name
				value := w.aiP1.blankValues[name]
				field := ui.TextInput(c.Key("ai-blank-"+name),
					&value).Placeholder(name).Grow(1)
				w.aiP1.blankValues[name] = value
				if field.Submitted() {
					w.runBlankAction(n, s)
				}
			}
			ui.Row(c).Gap(th.Space(2)).Children(func() {
				if ui.Button(c.Key("ai-blank-send"),
					locale.AiBlankSend()).Clicked() {
					w.runBlankAction(n, s)
				}
				if ui.Button(c.Key("ai-blank-cancel"),
					locale.AiBlankCancel()).Clicked() {
					w.aiP1.blankOpen = false
				}
			})
		})
	})
}

// runBlankAction sends the held action with the answers as extra
// variables for the prompt.
func (w *Workspace) runBlankAction(n *OpenNote, s *aiSession) {
	answers := map[string]string{}
	for k, v := range w.aiP1.blankValues {
		answers[k] = v
	}
	w.aiP1.blankAnswers = answers
	a := w.aiP1.blankAction
	w.aiP1.blankOpen = false
	s.Input = ""
	w.runAction(askingMore(a, w.aiP1.blankExtra))
}

// openBlankForm holds an action until its blanks are filled.
func (w *Workspace) openBlankForm(a config.AiAction, extra string) {
	names := config.ActionBlanks(a.Prompt)
	if len(names) == 0 {
		return
	}
	w.aiP1.blankOpen = true
	w.aiP1.blankAction = a
	w.aiP1.blankExtra = extra
	w.aiP1.blankNames = names
	w.aiP1.blankValues = map[string]string{}
	w.aiP1.blankAnswers = nil
}

// blankExtras are the last answers, merged into an action's prompt by
// expandActionPrompt.
func (w *Workspace) blankExtras() map[string]string {
	return w.aiP1.blankAnswers
}
