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
				w.aiSaid(c, i, blocks[i], opts, coming, s)
			}
		})
	})
}

// aiSaid is one thing said: the user's words plain, and the model's as
// Markdown, for that is how a model answers.
func (w *Workspace) aiSaid(
	c *ui.Context, at int, b aiBlock, opts *markdown.Options,
	coming bool, s *aiSession,
) {
	ui.Column(c).Gap(2).Children(func() {
		if b.Role == ai.User {
			ui.Row(c).Gap(c.Theme().Space(1)).AlignItems(ui.Center).
				Children(func() {
					shown := b.Text
					if b.Secret {
						shown = locale.AiQuestionPrivate()
					}
					ui.Text(c, shown).TextColor(c.Theme().TextMuted)
					if !b.Secret &&
						ui.Button(c.Key("ai-edit#"+
							strconv.Itoa(at)), locale.AiEdit()).
							FontSize(11).Clicked() {
						w.editUserBlock(s, at, b.Text)
					}
				})
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
		// What the answer cost, when the model said so: a quiet line at
		// the foot of the answer, not a badge in the middle of it.
		if b.Usage != nil && b.Over {
			ui.Text(c, locale.AiUsage(usageCount(b.Usage.PromptTokens),
				usageCount(b.Usage.CompletionTokens),
				usageCount(b.Usage.DurationMs))).
				FontSize(11).TextColor(c.Theme().TextMuted)
			w.usageBar(c, b)
		}
		w.timelineLine(c, b, s)
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
			// An answer may also be copied or appended to a note.
			// These only add words; a revision still goes through
			// the diff review above.
			if w.OnAiReply != nil {
				options := []struct{ kind, label string }{
					{"copy", locale.AiReplyCopy()},
				}
				if w.Config.Ai.Permission == config.PermissionAppend {
					options = append(options,
						struct{ kind, label string }{
							"daily", locale.AiReplyDaily()},
						struct{ kind, label string }{
							"note", locale.AiReplyNote()})
				}
				ui.Row(c).Gap(c.Theme().Space(1)).Wrap().
					Children(func() {
						for _, one := range options {
							one := one
							if ui.Button(c.Key("ai-reply-"+
								one.kind+"#"+strconv.Itoa(at)),
								one.label).FontSize(11).Clicked() {
								w.OnAiReply(one.kind, b.Text)
							}
						}
					})
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

// usageCount is a cost the model reported, or a dash for one it did not.
func usageCount(n *uint64) string {
	if n == nil {
		return "-"
	}
	return strconv.FormatUint(*n, 10)
}

// aiTold is what the model is told: what the note is, and all that has
// been said in this conversation so far.
func (w *Workspace) aiTold(s *aiSession, n *OpenNote) []ai.Message {
	// The contexts must be in place even when the composer was not
	// drawn first, as when a command came from the palette.
	w.syncContexts(s, n)
	out := []ai.Message{{
		Role: ai.System, Content: w.aiBrief(s, n),
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

// aiBrief is what the model is told before it is asked: the tongue it
// is to answer in, and whatever contexts the user has left attached.
// A context removed above the composer is not sent.
func (w *Workspace) aiBrief(s *aiSession, n *OpenNote) string {
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
	// A note of a hundred thousand words is not sent whole: the model
	// is given as much of each context as the settings allow.
	at := w.contextMaxChars()
	for _, one := range s.Contexts {
		if one.Removed {
			continue
		}
		text := one.Text
		if at > 0 && len(text) > at {
			text = text[:at]
		}
		switch one.Kind {
		case "note":
			said.WriteString("\nThe note being edited is " +
				PathOf(n.Path) + ":\n\n")
		case "selection":
			said.WriteString("\nThe selected text is:\n\n")
		case "ref":
			said.WriteString("\nAn attached note (" + one.Label +
				") is:\n\n")
		default:
			continue
		}
		said.WriteString(text)
	}
	return said.String()
}
