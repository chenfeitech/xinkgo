// This file is the conversation itself: what has been said, in what
// order, and said how.

package view

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/skill"
)

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
