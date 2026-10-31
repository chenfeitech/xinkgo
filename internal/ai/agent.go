package ai

import (
	"context"
	"strings"

	"github.com/helightxu/xinkgo/internal/config"
)

// Loop is a question asked with tools at hand: the model may reach for
// one, is told what it found, and is asked again — until it answers
// without reaching, or until it has been asked as often as it may.
//
// A model given no tools, or that reaches for none, is asked once: the
// loop is then the same as a single asking, and no worse for it.
//
// What the model said on the way is handed on as it comes; what a tool
// found is handed on too, for the same words are given back to the
// model and the user may as well see them.
func Loop(
	ctx context.Context, cfg config.AiConfig, e Endpoint, host Host,
	req Request, steps int,
) <-chan Event {
	out := make(chan Event)
	go func() {
		defer close(out)
		if steps < 1 {
			steps = 1
		}
		messages := req.Messages
		for step := 0; step < steps; step++ {
			pieces, err := Ask(ctx, cfg, e, Request{
				Model:    req.Model,
				Messages: messages,
				Params:   req.Params,
				Tools:    req.Tools,
			})
			if err != nil {
				out <- Failed(err)
				return
			}
			parser := NewParser(e)
			var said strings.Builder
			var calls []ToolCall
			var used Usage
			broken := false
			for piece := range pieces {
				for _, event := range parser.Piece(piece) {
					switch event.Kind {
					case EventDelta:
						said.WriteString(event.Text)
						out <- event
					case EventThinking:
						// The thinking is shown, and not given
						// back to the model: it is not a thing
						// that was said.
						out <- event
					case EventToolCall:
						calls = append(calls, event.Call)
					case EventUsage:
						used = event.Used
					case EventFailed:
						out <- event
						broken = true
					}
				}
			}
			// An answer that stopped without saying it was over is
			// one that broke off: there is no asking again, for the
			// words would come twice.
			for _, event := range parser.End() {
				if event.Kind == EventFailed {
					out <- event
					broken = true
				}
			}
			if broken {
				return
			}
			if len(calls) == 0 {
				// Nothing was reached for: this is the answer.
				out <- Used(used)
				out <- Done()
				return
			}
			if text := said.String(); strings.TrimSpace(text) != "" {
				messages = append(messages, Message{
					Role:    Assistant,
					Content: text,
				})
			}
			for _, call := range calls {
				out <- Called(call)
				found := Execute(call, host, int(cfg.MaxToolChars))
				out <- Found(found)
				messages = append(messages, Message{
					Role:    Tool,
					Content: found.Name + ": " + found.Output,
				})
			}
		}
		// It kept reaching and never came to an answer: the user is
		// told, rather than being shown nothing at all.
		out <- Failed(ErrSteps)
	}()
	return out
}
