package view

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/fuzzy"
)

// How many quick words stand above the place to ask at once: enough for
// the words a user keeps, and few enough to take in at a glance.
const quickLimit = 8

// quickStrip is the quick words, in a row above the place to ask: one
// press on a word asks the model what that word asks, and no slash has
// to be typed first.
//
// What is typed into the place to ask picks among them: after a slash,
// the row shows the words those letters were found in, and none of the
// others. typing is whether a word is being typed, and extra what was
// typed after it, which goes with the asking.
func (w *Workspace) quickStrip(
	c *ui.Context, s *aiSession, n *OpenNote, items []config.AiAction,
	extra string, typing bool,
) {
	if len(items) == 0 {
		return
	}
	th := c.Theme()
	// A word is not asked while an answer is coming: what is asked is
	// asked of a model that is already speaking.
	busy := s.isBusy()
	ui.Row(c).Gap(th.Space(0.5)).Wrap().FillWidth().Children(func() {
		for i := range items {
			i, one := i, items[i]
			chip := ui.Button(c.Key("quick-"+one.ID), one.Name).
				FontSize(12).Disabled(busy).
				// The one the keys are on is lit while a word is
				// being typed, and not while a question is.
				Selected(!busy && typing && i == w.quickChosen)
			// What the word asks, for the pointer resting on it: the
			// word alone says little of what it will do.
			if tip := oneLine(one.Prompt); tip != "" {
				chip.Tooltip(tip)
			}
			if chip.Clicked() {
				w.quickRun(n, s, one, extra)
			}
		}
	})
}

// quickKeys is the keys of the quick words, taken from the place to ask
// while a word is being typed in it: the arrows move among the words,
// and Enter asks what the one they are on asks.
//
// It gives back whether a word was asked: when it was, what was typed is
// not asked as it stands, for it is a word and not a question.
func (w *Workspace) quickKeys(
	c *ui.Context, s *aiSession, n *OpenNote, field ui.Element,
	items []config.AiAction, extra string, typing bool,
) bool {
	if !typing || len(items) == 0 || s.isBusy() {
		return false
	}
	if field.Shortcut(0, ui.KeyDown) && w.quickChosen < len(items)-1 {
		w.quickChosen++
	}
	if field.Shortcut(0, ui.KeyUp) && w.quickChosen > 0 {
		w.quickChosen--
	}
	if !field.Submitted() || w.quickChosen >= len(items) {
		return false
	}
	w.quickRun(n, s, items[w.quickChosen], extra)
	return true
}

// quickRun is a quick word run on the note: what the word asks is asked
// of the model, and what comes back is put before the user as a
// revision, as any other answer to an action is.
//
// extra is what was typed after the word: it goes with the asking, so
// that a word may be made to ask for a little more this once.
func (w *Workspace) quickRun(
	n *OpenNote, s *aiSession, a config.AiAction, extra string,
) {
	w.quickChosen = 0
	s.Input = ""
	w.runAction(askingMore(a, extra))
}

// quickWords is the words standing above the place to ask: all of them
// as the settings hold them, and, when letters were typed after a slash,
// those the letters were found in — the word it is run by first, and
// then the name it is called by, for a word is what is typed and a name
// only what is read.
func (w *Workspace) quickWords(query string) []config.AiAction {
	if len(w.Actions) == 0 {
		return nil
	}
	if query == "" {
		// Nothing was typed: the words as they were set, and not as
		// an alphabet would have them.
		out := make([]config.AiAction, 0, len(w.Actions))
		for i := range w.Actions {
			out = append(out, w.Actions[i])
			if len(out) == quickLimit {
				break
			}
		}
		return out
	}
	out := make([]config.AiAction, 0, len(w.Actions))
	seen := map[string]bool{}
	for _, names := range [][]string{
		wordsOf(w.Actions), namesOf(w.Actions),
	} {
		for _, r := range fuzzy.Search(names, query, quickLimit) {
			one := w.actionByWord(r.Text)
			if one == nil || seen[one.ID] {
				continue
			}
			seen[one.ID] = true
			out = append(out, *one)
			if len(out) == quickLimit {
				return out
			}
		}
	}
	return out
}

// actionByWord is the action run by that word, or called by that name,
// when there is one.
func (w *Workspace) actionByWord(word string) *config.AiAction {
	for i := range w.Actions {
		one := &w.Actions[i]
		if strings.EqualFold(one.Word(), word) ||
			strings.EqualFold(one.Name, word) {
			return one
		}
	}
	return nil
}

// askingMore is a quick word asked with more than it says: what was
// typed after it goes with it, as its own words at the end.
func askingMore(a config.AiAction, extra string) config.AiAction {
	extra = strings.TrimSpace(extra)
	if extra == "" {
		return a
	}
	out := a
	out.Prompt = a.Prompt + "\n\n" + extra
	return out
}

// wordBeingTyped is the quick word being typed in the place to ask: the
// letters after the slash it begins with, up to the first space, and
// after them the words that go with the asking.
func wordBeingTyped(input string) (query, extra string, typing bool) {
	if !strings.HasPrefix(input, "/") ||
		strings.ContainsAny(input, "\n") {
		return "", "", false
	}
	rest := input[1:]
	at := strings.Index(rest, " ")
	if at < 0 {
		return rest, "", true
	}
	return rest[:at], strings.TrimSpace(rest[at+1:]), true
}

// wordsOf is what the actions are run by, and namesOf what they are
// called: what was typed is looked for in both.
func wordsOf(list []config.AiAction) []string {
	out := make([]string, 0, len(list))
	for _, one := range list {
		out = append(out, one.Word())
	}
	return out
}

func namesOf(list []config.AiAction) []string {
	out := make([]string, 0, len(list))
	for _, one := range list {
		out = append(out, one.Name)
	}
	return out
}
