package view

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/skill"
	"github.com/helightxu/xinkgo/internal/state"
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

// blocks is what has been said, as it stands now.
func (s *aiSession) blocks() []aiBlock {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]aiBlock{}, s.Blocks...)
}

// isBusy is whether the model is still answering.
func (s *aiSession) isBusy() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.busy
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

// aiTalk is what has been said, the newest at the bottom.
func (w *Workspace) aiTalk(c *ui.Context, s *aiSession, n *OpenNote) {
	blocks := s.blocks()
	busy := s.isBusy()
	opts := w.markdownOptions(c, n.Path)
	ui.Scroll(c).TrackScroll(&s.scroll).MaxHeight(aiTalkHeight).
		FillWidth().Children(func() {
		ui.Column(c).Gap(10).Children(func() {
			for i := range blocks {
				// The last of them is still being written, and
				// is shown as it comes.
				coming := busy && i == len(blocks)-1
				w.aiSaid(c, i, blocks[i], opts, coming)
			}
		})
	})
}

// aiSaid is one thing said: the user's words plain, and the model's as
// Markdown, for that is how a model answers.
func (w *Workspace) aiSaid(
	c *ui.Context, at int, b aiBlock, opts *markdown.Options,
	coming bool,
) {
	ui.Column(c).Gap(2).Children(func() {
		if b.Role == ai.User {
			ui.Text(c, b.Text).TextColor(c.Theme().TextMuted)
			return
		}
		if b.Thinking != "" {
			ui.Text(c, b.Thinking).FontSize(12).
				TextColor(c.Theme().TextMuted)
		}
		// What the tools came back with on the way to this answer:
		// the same words the model was given, so that what it read
		// before answering can be seen.
		for _, got := range b.Tool {
			ui.Text(c, got.Name+" → "+oneLine(got.Output)).
				FontSize(11).TextColor(c.Theme().TextMuted)
		}
		if b.Text != "" {
			if coming || !b.Over {
				markdown.Stream(c, b.Text, opts)
			} else {
				markdown.View(c, b.Text, opts)
			}
		}
		if b.Cut {
			ui.Text(c, locale.AiOutOfRoom()).FontSize(12).
				TextColor(c.Theme().TextMuted)
		}
		if b.Failed != "" {
			ui.Text(c, b.Failed).FontSize(12).
				TextColor(c.Theme().Danger)
		}
		// An answer that is over and has words in it can be looked over
		// as a revision of the note: what the model made of it, a
		// change at a time.
		if b.Over && b.Text != "" {
			if ui.Button(c.Key("ai-review#"+strconv.Itoa(at)),
				locale.AiLookOver()).
				Label(locale.AiLookOverLabel()).
				Clicked() {
				w.StartReview(b.Text)
			}
		}
	})
}

// oneLine is a tool's answer on one line: what a tool finds may be many
// lines long, and it is only being pointed at here, not read.
func oneLine(text string) string {
	flat := strings.TrimSpace(strings.Join(strings.Fields(text), " "))
	runes := []rune(flat)
	if len(runes) > 120 {
		return string(runes[:120]) + "…"
	}
	return flat
}

// aiCompose is the place to ask: which model is asked, the words to be
// asked, and the way to ask them — or to stop the answer that is coming.
func (w *Workspace) aiCompose(c *ui.Context, s *aiSession, n *OpenNote) {
	ui.Row(c).Gap(6).Children(func() {
		w.aiModelPick(c, s)
		field := ui.TextInput(c.Key("ai-input"), &s.Input).Grow(1)
		if s.focus {
			field.Focus()
			s.focus = false
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

// runAction is one of the app's own actions run on the note: what the
// action says is asked of the model, and what comes back is put before
// the user as a revision — as any other revision is, and not written
// into the note on the model's word.
func (w *Workspace) runAction(a config.AiAction) {
	n := w.Tabs.ActiveNote()
	if n == nil {
		w.status = locale.StatusNoNoteForAction()
		return
	}
	s := w.aiOf(n.Path)
	if s.isBusy() {
		w.status = locale.StatusStillAnswering()
		return
	}
	piece, from, to, part := pieceOf(n, a)
	s.Open = true
	w.aiSend(n, s, askingOf(a, piece, part), modelOf(a, w.AiModel),
		&waitReview{part: part, from: from, to: to})
	w.status = locale.StatusActionAsked(a.Name)
}

// pieceOf is what an action is done to: the selection, when the note has
// one and the action is one that goes by what is selected; and the whole
// note otherwise. It gives back the words, and where they stand in the
// note — 0 and 0 when they are the whole of it.
func pieceOf(
	n *OpenNote, a config.AiAction,
) (text string, from, to int, part bool) {
	whole := n.Editor.Buffer.String()
	sel := n.Editor.Selection()
	if a.Scope == config.ActionScopeWhole || sel.Empty() {
		return whole, 0, 0, false
	}
	from, to = sel.Start(), sel.End()
	if from < 0 {
		from = 0
	}
	if to > len(whole) {
		to = len(whole)
	}
	if from > to {
		from = to
	}
	return whole[from:to], from, to, true
}

// askingOf is what the model is asked when an action is run: what the
// action says. When the action is done to a piece of the note and not to
// the whole of it, the piece goes with it — the note as a whole the
// model has already been told — and it is told to answer with the piece
// and nothing else, for what comes back goes back where the piece was
// taken from.
func askingOf(a config.AiAction, piece string, part bool) string {
	if !part {
		return a.Prompt
	}
	return a.Prompt + "\n\nThis is the piece to do it to. Answer with " +
		"the piece as it should be, and with nothing else:\n\n" + piece
}

// aiReviewDone is an answer that an action was waiting for, put before
// the user as a revision now that it is over.
//
// It is done here, by the frame, and not where the answer is heard: a
// revision is made out of the note's own editor, and only this thread
// may touch that.
func (w *Workspace) aiReviewDone() {
	n := w.Tabs.ActiveNote()
	if n == nil || w.ai == nil {
		return
	}
	s := w.ai[n.Path]
	if s == nil {
		return
	}
	s.mu.Lock()
	wait := s.waiting
	text, failed := "", ""
	if wait != nil && !s.busy && len(s.Blocks) > 0 {
		last := s.Blocks[len(s.Blocks)-1]
		if last.Over {
			text, failed = last.Text, last.Failed
			s.waiting = nil
		}
	}
	s.mu.Unlock()
	if text == "" || failed != "" {
		return
	}
	if wait.part {
		w.StartPieceReview(wait.from, wait.to, text)
		return
	}
	w.StartReview(text)
}

// aiAsk is the question asked: what was typed into the place to ask.
func (w *Workspace) aiAsk(n *OpenNote, s *aiSession) {
	w.aiSend(n, s, strings.TrimSpace(s.Input), w.AiModel, nil)
	s.Input = ""
	s.focus = true
}

// aiSend is a question sent: it is written into the conversation before
// the model is asked, and the answer grows into the block after it as it
// comes.
//
// model is which model is asked of the ones the settings hold, empty for
// the one the settings name. wait, when there is one, is what the answer
// is to be put before the user as when it is over, and which piece of the
// note it is about.
func (w *Workspace) aiSend(
	n *OpenNote, s *aiSession, asking, model string, wait *waitReview,
) {
	text := strings.TrimSpace(asking)
	if text == "" {
		return
	}
	s.mu.Lock()
	s.Blocks = append(s.Blocks,
		aiBlock{Role: ai.User, Text: text, Over: true},
		aiBlock{Role: ai.Assistant},
	)
	at := len(s.Blocks) - 1
	s.busy = true
	s.dirty = true
	s.waiting = wait
	s.mu.Unlock()

	if w.OnAsk == nil {
		w.aiDone(s, at, ai.ErrNotConfigured)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.cancel = cancel
	s.mu.Unlock()
	events, err := w.OnAsk(ctx, w.aiTold(s, n), model)
	if err != nil {
		cancel()
		w.aiDone(s, at, err)
		return
	}
	go w.aiHear(s, at, events, cancel)
}

// aiHear is the answer being heard: each piece of it is written into the
// block that waits for it, until the model is done or something goes
// wrong.
func (w *Workspace) aiHear(
	s *aiSession, at int, events <-chan ai.Event, cancel context.CancelFunc,
) {
	defer cancel()
	for event := range events {
		s.mu.Lock()
		if at < len(s.Blocks) {
			b := &s.Blocks[at]
			switch event.Kind {
			case ai.EventDelta:
				b.Text += event.Text
			case ai.EventThinking:
				b.Thinking += event.Text
			case ai.EventTruncated:
				b.Cut = true
			case ai.EventToolResult:
				b.Tool = append(b.Tool, event.Result)
			case ai.EventFailed:
				b.Failed = event.Err.Error()
				b.Over = true
			case ai.EventDone:
				b.Over = true
			}
		}
		s.mu.Unlock()
	}
	// An answer that stopped without saying it was over is over anyway:
	// what was said of it was said as it came.
	s.mu.Lock()
	if at < len(s.Blocks) {
		s.Blocks[at].Over = true
	}
	s.busy = false
	s.cancel = nil
	s.dirty = true
	s.mu.Unlock()
}

// aiDone is an answer that never began.
func (w *Workspace) aiDone(s *aiSession, at int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at < len(s.Blocks) {
		s.Blocks[at].Failed = err.Error()
		s.Blocks[at].Over = true
	}
	s.busy = false
	s.cancel = nil
	s.dirty = true
}

// aiStop is the answer stopped: the model is let go, and what it had said
// by then is kept.
func (w *Workspace) aiStop(s *aiSession) {
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// aiForget is the conversation begun again.
func (w *Workspace) aiForget(s *aiSession) {
	w.aiStop(s)
	s.mu.Lock()
	s.Blocks = nil
	s.busy = false
	s.cancel = nil
	// Nothing left to keep is the same as never having kept it: the
	// note's conversation is taken out of the book.
	s.dirty = true
	s.mu.Unlock()
	s.Input = ""
}

// aiSave is the conversation written down, when it has changed since it
// last was.
//
// It is the frame that writes, and not the goroutine that hears the
// answer: the book the conversation goes into is the app's own, and is
// not to be touched from another goroutine.
func (w *Workspace) aiSave(path string, s *aiSession) {
	if w.OnAiSave == nil {
		return
	}
	s.mu.Lock()
	if !s.dirty {
		s.mu.Unlock()
		return
	}
	s.dirty = false
	blocks := append([]aiBlock{}, s.Blocks...)
	s.mu.Unlock()
	w.OnAiSave(path, w.aiStored(blocks))
}

// aiStored is what is kept of a conversation: what was said, and none of
// what is still being said, or of what went wrong instead of it.
func (w *Workspace) aiStored(blocks []aiBlock) []state.AiBlock {
	out := make([]state.AiBlock, 0, len(blocks))
	for _, b := range blocks {
		if b.Text == "" || b.Failed != "" {
			continue
		}
		if b.Role != ai.User && b.Role != ai.Assistant {
			continue
		}
		out = append(out, state.AiBlock{
			Role:     string(b.Role),
			Text:     b.Text,
			Thinking: b.Thinking,
			Cut:      b.Cut,
		})
	}
	return out
}

// aiRestored is a conversation read back: everything in it was said, and
// is over.
func (w *Workspace) aiRestored(blocks []state.AiBlock) []aiBlock {
	out := make([]aiBlock, 0, len(blocks))
	for _, b := range blocks {
		role := ai.Role(b.Role)
		if role != ai.User && role != ai.Assistant {
			continue
		}
		out = append(out, aiBlock{
			Role:     role,
			Text:     b.Text,
			Thinking: b.Thinking,
			Cut:      b.Cut,
			Over:     true,
		})
	}
	return out
}

// aiTold is what the model is told: what the note is, and all that has
// been said in this conversation so far.
func (w *Workspace) aiTold(s *aiSession, n *OpenNote) []ai.Message {
	out := []ai.Message{{
		Role: ai.System, Content: w.aiBrief(n),
	}}
	// What a model is told of the skills there are; and, when one has
	// been taken up, what that one says, whole.
	if s.Skill != nil {
		out = append(out, ai.Message{
			Role: ai.System,
			Content: "Using the skill " + s.Skill.Name + ":\n\n" +
				s.Skill.Body,
		})
	} else if catalog := skill.Catalog(w.Skills); catalog != "" {
		out = append(out, ai.Message{Role: ai.System, Content: catalog})
	}
	for _, b := range s.blocks() {
		// What went wrong is not a thing that was said, and is not
		// sent back.
		if b.Text == "" || b.Failed != "" {
			continue
		}
		if b.Role == ai.User || b.Role == ai.Assistant {
			out = append(out, ai.Message{Role: b.Role, Content: b.Text})
		}
	}
	return out
}

// aiBrief is what the model is told before it is asked: which note this
// is about, so that it need not be told twice, and the tongue it is to
// answer in.
func (w *Workspace) aiBrief(n *OpenNote) string {
	var said strings.Builder
	said.WriteString("You help the user write notes in Markdown, in the " +
		"app xinkgo. Answer in the note's own tongue unless told " +
		"otherwise.\n")
	switch w.Config.Ai.OutputLanguage {
	case config.OutputZh:
		said.WriteString("Answer in Chinese.\n")
	case config.OutputEn:
		said.WriteString("Answer in English.\n")
	}
	if n == nil {
		return said.String()
	}
	said.WriteString("\nThe note being edited is " + PathOf(n.Path) + ":\n\n")
	text := n.Editor.Buffer.String()
	// A note of a hundred thousand words is not sent whole: the model
	// is given as much of it as the settings allow.
	if at := int(w.Config.Ai.MaxContextChars); at > 0 && len(text) > at {
		text = text[:at]
	}
	said.WriteString(text)
	return said.String()
}
