package view

import (
	"context"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/skill"
)

// How much room the conversation takes at the bottom of the note: the
// talk itself scrolls within it, and the place to ask stays below.
const aiTalkHeight float32 = 260

// How wide the list of the models is, in the place to ask: wide enough
// for the names the machine gives them, and no wider.
const aiModelWidth float32 = 150

// aiBlock is one thing said in a conversation: what the user asked, or
// what the model answered.
type aiBlock struct {
	Role ai.Role
	Text string
	// Thinking is the model's thinking, kept apart from the answer so
	// that it can be shown as thinking and not as words spoken to the
	// user.
	Thinking string
	// Over is whether what was said has come to an end.
	Over bool
	// Cut is an answer the model broke off because it had said as much
	// as it was allowed to: half of what it would have said.
	Cut bool
	// Failed is what went wrong, when something did.
	Failed string
	// Tool is what the tools came back with on the way to this answer:
	// the same words the model was given, so that what it read before
	// answering can be seen.
	Tool []ai.Result
}

// aiSession is one note's conversation with a model. It belongs to the
// note and not to the window: closing the bar hides it and nothing more,
// and what was said comes back with the note.
type aiSession struct {
	Open  bool
	Input string
	focus bool
	// scroll is where the conversation is scrolled to, kept between
	// frames as any scroll is.
	scroll ui.ScrollState

	// mu keeps what was said: the words come from the model on a
	// goroutine of its own, and are read by the frame being drawn.
	mu     sync.Mutex
	Blocks []aiBlock
	busy   bool
	cancel context.CancelFunc
	// dirty is whether what was said has changed since it was last
	// written down.
	dirty bool
	// Skill is the skill the model is being told to use, when one has
	// been taken up. It is told with every question until it is put
	// down, and it is not written down: it is a thing of the moment,
	// and the skill itself is on the disk.
	Skill *skill.Skill
	// waiting is the revision an answer is to be put before the user as
	// when it is over: what an action asked for, and which piece of the
	// note it was about. It is looked at by the frame and not by the
	// goroutine that hears the answer, for a revision is made out of
	// the note's own editor.
	waiting *waitReview
}

// waitReview is what an action is waiting for: whether what it asked
// about was a piece of the note and not the whole of it, and where that
// piece stands.
type waitReview struct {
	part     bool
	from, to int
}

// ToggleAI is the bar opened under the note, or hidden: what was said in
// it is kept either way.
func (w *Workspace) ToggleAI() {
	n := w.Tabs.ActiveNote()
	if n == nil {
		return
	}
	s := w.aiOf(n.Path)
	s.Open = !s.Open
	if s.Open {
		s.focus = true
	}
}

// aiBand is the room the way to the model takes in the toolbar's row: its
// own button, the line before it and the room between. The row of the
// Markdown buttons is made that much narrower, so that the button stands
// beside it and not under the rail.
const aiBand float32 = 40

// aiOpen is whether the note's conversation stands open, and nothing
// more: the button that opens it asks this every frame, and so must not
// make a conversation that is not there yet.
func (w *Workspace) aiOpen(path string) bool {
	s, ok := w.ai[path]
	return ok && s.Open
}

// aiOf is the note's conversation, made the first time it is asked for:
// what was said about it before is read back then, and not every frame.
func (w *Workspace) aiOf(path string) *aiSession {
	if w.ai == nil {
		w.ai = map[string]*aiSession{}
	}
	if _, ok := w.ai[path]; !ok {
		// Whether the bar is open is kept as the user left it: a note
		// they were asking about is one they ask about again.
		s := &aiSession{}
		if w.OnAiLoad != nil {
			s.Blocks = w.aiRestored(w.OnAiLoad(path))
		}
		w.ai[path] = s
	}
	return w.ai[path]
}

// aiBar is the conversation with a model, under the note: what was asked,
// what was answered, and the place to ask again.
func (w *Workspace) aiBar(c *ui.Context) {
	n := w.Tabs.ActiveNote()
	if n == nil {
		return
	}
	s := w.aiOf(n.Path)
	if !s.Open {
		return
	}
	ui.Column(c).Gap(6).Padding(8).Children(func() {
		w.aiHead(c, s)
		w.aiTalk(c, s, n)
		w.aiCompose(c, s, n)
	})
	w.aiSave(n.Path, s)
}

// aiHead is the bar's own line: what it is, and the way to forget or to
// hide what was said.
func (w *Workspace) aiHead(c *ui.Context, s *aiSession) {
	ui.Row(c).Gap(8).Children(func() {
		ui.Text(c, "AI").Bold()
		if s.Skill != nil {
			ui.Text(c, locale.AiUsingSkill(s.Skill.Name)).FontSize(11).
				TextColor(c.Theme().TextMuted)
			if ui.Button(c.Key("ai-skill-away"), locale.AiPutSkillDown()).
				Label(locale.AiPutSkillDownLabel()).Clicked() {
				s.Skill = nil
			}
		}
		if s.isBusy() {
			ui.Text(c, locale.AiAnswering()).FontSize(12).
				TextColor(c.Theme().TextMuted)
		}
		if ui.Button(c.Key("ai-forget"), locale.AiForget()).
			Label(locale.AiForgetLabel()).Clicked() {
			w.aiForget(s)
		}
		if ui.Button(c.Key("ai-hide"), "×").
			Label(locale.AiHideLabel()).Clicked() {
			s.Open = false
		}
	})
}

// aiCompose is the place to ask: which model is asked, the words to be
// asked, and the way to ask them — or to stop the answer that is coming.
//
// The quick words stand in a row of their own above it, one press away;
// a slash typed into the place to ask picks among them as it is typed.
func (w *Workspace) aiCompose(c *ui.Context, s *aiSession, n *OpenNote) {
	word, extra, typing := wordBeingTyped(s.Input)
	items := w.quickWords(word)
	// A word typed anew is a choice anew: what the keys were on is let
	// go of as soon as the letters change.
	if word != w.quickQuery {
		w.quickQuery, w.quickChosen = word, 0
	}
	if w.quickChosen >= len(items) {
		w.quickChosen = 0
	}
	ui.Column(c).Gap(6).Children(func() {
		w.quickStrip(c, s, n, items, extra, typing)
		ui.Row(c).Gap(6).Children(func() {
			w.aiModelPick(c, s)
			field := ui.TextInput(c.Key("ai-input"), &s.Input).Grow(1)
			if s.focus {
				field.Focus()
				s.focus = false
			}
			// A word being typed is a word, and not a question:
			// Enter asks what that word asks, and not the letters.
			if w.quickKeys(c, s, n, field, items, extra, typing) {
				return
			}
			if s.isBusy() {
				if ui.Button(c.Key("ai-stop"), locale.AiStop()).
					Label(locale.AiStopLabel()).Clicked() {
					w.aiStop(s)
				}
				return
			}
			if field.Submitted() {
				w.aiAsk(n, s)
				return
			}
			if ui.Button(c.Key("ai-send"), locale.AiAsk()).
				Label(locale.AiAskLabel()).Clicked() {
				w.aiAsk(n, s)
			}
		})
	})
}

// aiModelPick is which model is asked: the ones the settings hold and are
// turned on, of which one is picked here.
//
// What is offered is only what can be asked. A model turned off is not on
// the list, and neither is one the settings no longer hold; a picked one
// that has gone since is let go of, and the one the settings name is
// asked again.
func (w *Workspace) aiModelPick(c *ui.Context, s *aiSession) {
	// A model is offered where it may be asked: with the asking turned
	// off, or with no model to ask, there is nothing to choose from.
	if !w.Config.Ai.Enabled {
		return
	}
	names := w.aiModelNames()
	if len(names) == 0 {
		return
	}
	w.AiModel = w.aiModelNamed(names)
	was := w.AiModel
	ui.Select(c, &w.AiModel, names).Width(aiModelWidth).
		Label("Which model is asked").Disabled(s.isBusy())
	if w.AiModel != was {
		// The hand has been laid on it: what is picked here is asked
		// from now on, and not what the settings name.
		w.aiPicked = true
	}
}

// aiModelNamed is the model the bar asks as things stand: the one picked
// in it while it is still one the settings hold, and the one the settings
// name when it is not. A model taken out of the settings since is let go
// of, rather than asked and found wanting.
func (w *Workspace) aiModelNamed(names []string) string {
	if !w.aiPicked || !hasName(names, w.AiModel) {
		w.aiPicked = false
		return namedOf(names, w.Config.Ai.DefaultModel)
	}
	return w.AiModel
}

// aiModelNames is the models the bar may ask, in the order the settings
// hold them: one that is turned off is not one of them.
func (w *Workspace) aiModelNames() []string {
	out := make([]string, 0, len(w.Config.Ai.Models))
	for _, m := range w.Config.Ai.Models {
		if m.Enabled && strings.TrimSpace(m.ID) != "" {
			out = append(out, m.ID)
		}
	}
	return out
}

// namedOf is the name in the list that the settings named, or the first of
// them when they named none: a bar that asks a model the settings no
// longer hold asks nothing at all.
func namedOf(names []string, want string) string {
	if hasName(names, want) {
		return want
	}
	return names[0]
}

// modelOf is the model an action is asked of: the one the action names of
// its own, when it names one, and otherwise the one the bar asks.
func modelOf(a config.AiAction, fallback string) string {
	if a.Model != nil && strings.TrimSpace(*a.Model) != "" {
		return strings.TrimSpace(*a.Model)
	}
	return fallback
}

// hasName is whether the list holds that name.
func hasName(list []string, want string) bool {
	for _, one := range list {
		if one == want {
			return true
		}
	}
	return false
}
