package view

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/fuzzy"
	"github.com/helightxu/xinkgo/internal/locale"
)

// roundChange is what one revision did to one note. It is kept only for
// the most recent revision; the snapshots on disk are the longer history.
type roundChange struct {
	Path          string
	Before, After string
}

// activityOf is one step of a model's work, named by a verb and what it
// was done to. Consecutive read-only steps fold into one line.
func (w *Workspace) activityOf(call ai.ToolCall) aiActivity {
	verb := "read"
	switch call.Name {
	case ai.ToolSearchNotes:
		verb = "search"
	case ai.ToolListTags, ai.ToolListNotes, ai.ToolListBacklinks,
		ai.ToolOutline:
		verb = "list"
	case ai.ToolAskUser:
		verb = "ask"
	}
	target := call.Arguments
	var args map[string]any
	if err := json.Unmarshal([]byte(call.Arguments), &args); err == nil {
		if one, ok := args["path"].(string); ok && one != "" {
			target = one
		} else if one, ok := args["query"].(string); ok && one != "" {
			target = one
		}
	}
	if len([]rune(target)) > 60 {
		target = string([]rune(target)[:60]) + "…"
	}
	return aiActivity{Verb: verb, Target: target}
}

// timelineLine is the folded work of an answer, and the unfolded list
// of steps when the user asks to see it.
func (w *Workspace) timelineLine(
	c *ui.Context, b aiBlock, s *aiSession,
) {
	if len(b.Activities) == 0 {
		return
	}
	th := c.Theme()
	reads, searches := 0, 0
	for _, one := range b.Activities {
		if one.Verb == "search" {
			searches++
		} else {
			reads++
		}
	}
	label := locale.AiTimeline(strconv.Itoa(reads),
		strconv.Itoa(searches))
	toggle := locale.AiTimelineShow()
	if s.TimelineOpen {
		toggle = locale.AiTimelineHide()
	}
	ui.Row(c).Gap(th.Space(1)).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(th.TextMuted)
		if ui.Button(c.Key("ai-timeline-toggle"), toggle).
			FontSize(11).Clicked() {
			s.TimelineOpen = !s.TimelineOpen
		}
	})
	if !s.TimelineOpen {
		return
	}
	for _, one := range b.Activities {
		ui.Text(c, locale.AiTimelineStep(one.Verb, one.Target)).
			FontSize(11).TextColor(th.TextMuted)
	}
	if b.Usage != nil && b.Usage.DurationMs != nil && b.Over {
		ui.Text(c, locale.AiTimelineDone(
			strconv.FormatUint(*b.Usage.DurationMs, 10))).
			FontSize(11).TextColor(th.TextMuted)
	}
}

// usageBar is how much of the model's context window this answer used,
// when the model reported it and the model declares a limit.
func (w *Workspace) usageBar(c *ui.Context, b aiBlock) {
	if b.Usage == nil || b.Usage.PromptTokens == nil || !b.Over {
		return
	}
	limit := w.contextLimitOf(w.AiModel)
	if limit == 0 {
		return
	}
	used := *b.Usage.PromptTokens
	if b.Usage.CompletionTokens != nil {
		used += *b.Usage.CompletionTokens
	}
	pct := int(used * 100 / uint64(limit))
	if pct > 100 {
		pct = 100
	}
	th := c.Theme()
	ui.Row(c).Gap(th.Space(1)).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Width(64).Height(4).Background(th.Border).
			Children(func() {
				ui.Box(c).Width(float32(pct) * 0.64).Height(4).
					Background(th.Accent)
			})
		ui.Text(c, locale.AiUsagePct(strconv.Itoa(pct))).
			FontSize(11).TextColor(th.TextMuted)
	})
}

func (w *Workspace) contextLimitOf(id string) uint32 {
	for _, m := range w.Config.Ai.Models {
		if m.ID == id && m.ContextLimit != nil {
			return *m.ContextLimit
		}
	}
	return 0
}

// undoRound puts the last revision back, but only when the note is still
// exactly as the revision left it. A note the user has edited since is
// not overwritten.
func (w *Workspace) undoRound() {
	change := w.LastChange
	if change == nil || w.OnAiUndo == nil {
		return
	}
	w.LastChange = nil
	w.status = w.OnAiUndo(change.Path, change.Before, change.After)
}

// questionCard is a decision a model asked for: lettered options, an
// optional typed answer, and a way to skip. One card at a time.
func (w *Workspace) questionCard(
	c *ui.Context, s *aiSession, n *OpenNote,
) {
	q := s.PendingQuestion
	if q == nil {
		return
	}
	th := c.Theme()
	ui.Column(c).FillWidth().Gap(th.Space(1)).
		Padding(th.Space(2)).Background(th.Surface).
		Border(1, th.Border).Radius(th.Radius).Children(func() {
		if q.Header != "" {
			ui.Text(c, q.Header).Bold().FontSize(11)
		}
		ui.Text(c, q.Text)
		for i, one := range q.Options {
			label := string(rune('A'+i)) + "  " + one
			if ui.Button(c.Key("ai-q-opt-"+strconv.Itoa(i)), label).
				Clicked() {
				w.answerQuestion(s, n, one)
			}
		}
		if q.AllowOther || len(q.Options) == 0 {
			field := ui.TextInput(c.Key("ai-q-other"),
				&s.QuestionAnswer).
				Placeholder(locale.AiQuestionOther()).Grow(1)
			if field.Submitted() {
				w.answerQuestion(s, n, s.QuestionAnswer)
			}
			if ui.Button(c.Key("ai-q-send"),
				locale.AiQuestionSend()).Clicked() {
				w.answerQuestion(s, n, s.QuestionAnswer)
			}
		}
		if ui.Button(c.Key("ai-q-skip"),
			locale.AiQuestionSkip()).Clicked() {
			w.answerQuestion(s, n, "")
		}
	})
}

// answerQuestion starts a new turn with the user's decision as the next
// thing said. A skipped question ends the conversation's turn quietly.
func (w *Workspace) answerQuestion(
	s *aiSession, n *OpenNote, answer string,
) {
	q := s.PendingQuestion
	if q == nil {
		return
	}
	s.PendingQuestion = nil
	s.QuestionAnswer = ""
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return
	}
	s.secretNext = q.Secret
	w.aiSend(n, s, answer, config.RouteKindChat, nil, nil)
}

// editUserBlock puts an earlier question back in the composer and drops
// everything said after it, so resubmitting regenerates from there.
func (w *Workspace) editUserBlock(s *aiSession, at int, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at < 0 || at > len(s.Blocks) {
		return
	}
	s.Blocks = s.Blocks[:at]
	s.Input = text
	s.focus = true
	s.dirty = true
}

// notePath is the full path of a note named in the vault's own terms.
func (w *Workspace) notePath(name string) string {
	if w.Index == nil {
		return ""
	}
	root := w.Index.Root()
	for _, path := range w.Index.Paths() {
		rel := path
		if root != "" {
			if got, err := filepath.Rel(root, path); err == nil {
				rel = got
			}
		}
		if filepath.ToSlash(rel) == filepath.ToSlash(name) {
			return path
		}
	}
	return ""
}

// atCompletions is @ followed by a note name: a popover of vault notes,
// one press attaches that note as context instead of sending the words.
func (w *Workspace) atCompletions(
	c *ui.Context, s *aiSession, field ui.Element,
) {
	at := strings.LastIndex(s.Input, "@")
	if at < 0 {
		w.aiP1.atOpen = false
		return
	}
	query := s.Input[at+1:]
	if strings.ContainsAny(query, " \n\t") {
		w.aiP1.atOpen = false
		return
	}
	names := w.noteNames()
	if len(names) == 0 {
		w.aiP1.atOpen = false
		return
	}
	items := fuzzy.Search(names, query, 8)
	if len(items) == 0 {
		w.aiP1.atOpen = false
		return
	}
	w.aiP1.atOpen = true
	ui.Popover(c, field, &w.aiP1.atOpen, func() {
		ui.Column(c).Width(320).ClipY().MaxHeight(220).Children(func() {
			for _, one := range items {
				name := one.Text
				if ui.Button(c.Key("ai-at-"+name), name).
					FontSize(12).Clicked() {
					w.attachRef(s, name)
					s.Input = s.Input[:at]
					w.aiP1.atOpen = false
				}
			}
		})
	})
}

// attachRef reads a note once and keeps it as a removable context until
// the user takes it away or the note changes.
func (w *Workspace) attachRef(s *aiSession, name string) {
	for _, one := range s.Refs {
		if one == name {
			return
		}
	}
	path := w.notePath(name)
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	s.Refs = append(s.Refs, name)
	if s.RefTexts == nil {
		s.RefTexts = map[string]string{}
	}
	s.RefTexts[name] = string(data)
	s.contextKey = ""
}

// sessionList is the conversations the app remembers, one to a note.
// Picking one opens that note and its conversation comes back with it.
func (w *Workspace) sessionList(c *ui.Context, current string) {
	if w.ai == nil {
		return
	}
	paths := []string{}
	for path, one := range w.ai {
		if path != current && len(one.Blocks) > 0 {
			paths = append(paths, path)
		}
	}
	if len(paths) == 0 {
		return
	}
	chip := ui.Button(c.Key("ai-sessions"), locale.AiSessions())
	ui.Popover(c, chip, &w.aiP1.sessionOpen, func() {
		ui.Column(c).Width(300).ClipY().MaxHeight(240).Children(func() {
			if len(paths) == 0 {
				ui.Text(c, locale.AiSessionsNone()).
					FontSize(11).TextColor(c.Theme().TextMuted)
				return
			}
			for _, path := range paths {
				path := path
				label := PathOf(path)
				if ui.Button(c.Key("ai-session-"+label), label).
					FontSize(12).Clicked() {
					w.aiP1.sessionOpen = false
					_, _ = w.Tabs.Open(path)
				}
			}
		})
	})
}
