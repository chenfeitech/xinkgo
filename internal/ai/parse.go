package ai

import (
	"encoding/json"
	"strings"

	"github.com/helightxu/xinkgo/internal/config"
)

// Parser is one answer being read. It is given the pieces as they come and
// says what each of them means; it remembers what it needs to — the calls
// being built up out of fragments, what the answer has cost so far —
// between one piece and the next.
//
// A Parser is one answer's: it is not shared, and it is not used twice.
type Parser struct {
	kind   string
	model  string
	drafts []callDraft
	used   Usage
	over   bool
}

// callDraft is a tool call being built up out of fragments. Only the
// OpenAI-shaped endpoints send them in pieces: the name comes on one
// fragment and the arguments a few characters at a time, and the same
// `index` is what says two fragments are of one call. Ollama gives a call
// whole or not at all.
type callDraft struct {
	name      string
	arguments string
}

// NewParser is a parser for the answer of the model at e.
func NewParser(e Endpoint) *Parser {
	return &Parser{kind: e.Kind, model: e.Model}
}

// Piece says what one piece of an answer means. A piece that means nothing
// — an empty line, a field that is of no interest — gives back nothing.
func (p *Parser) Piece(piece Piece) []Event {
	if p.over {
		// The answer is over: what comes after it has nowhere to land.
		return nil
	}
	if piece.Err != nil {
		return p.fail(failure(piece.Err, p.model))
	}
	line := strings.TrimSpace(string(piece.Data))
	if line == "" {
		return nil
	}
	if p.kind == config.ModelOpenAICompatible {
		if line == "[DONE]" {
			return p.finish(false)
		}
		return p.openAI(line)
	}
	return p.ollama(line)
}

// End is the answer stopping. An answer that stopped without ever having
// been said to be over is one that broke off: the words may be half of
// what they should be, and the one who listens is told so.
func (p *Parser) End() []Event {
	if p.over {
		return nil
	}
	return p.fail(ErrTruncated)
}

// finish is the end of an answer: what it cost, whether it was cut short,
// and that it is over.
func (p *Parser) finish(cut bool) []Event {
	out := []Event{Used(p.used)}
	if cut {
		out = append(out, Truncated())
	}
	out = append(out, Done())
	p.over = true
	return out
}

// fail is the answer going wrong.
func (p *Parser) fail(err error) []Event {
	p.over = true
	return []Event{Failed(err)}
}

// ollama says what one line of an Ollama answer means.
func (p *Parser) ollama(line string) []Event {
	var frame struct {
		Error  any `json:"error"`
		Done   bool
		Reason string  `json:"done_reason"`
		Prompt *uint64 `json:"prompt_eval_count"`
		Eval   *uint64 `json:"eval_count"`
		// The durations are nanoseconds, and there are two of them:
		// how long the model spent answering, and how long the whole
		// thing took. The first is the one that is of interest, when
		// it is there at all.
		EvalNS  *uint64 `json:"eval_duration"`
		TotalNS *uint64 `json:"total_duration"`
		Message struct {
			Content  string `json:"content"`
			Thinking string `json:"thinking"`
			Calls    []struct {
				Function struct {
					Name      string          `json:"name"`
					Arguments json.RawMessage `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
	}
	if err := json.Unmarshal([]byte(line), &frame); err != nil {
		// A line that is not JSON says nothing.
		return nil
	}
	if message := errMessage(frame.Error); message != "" {
		return p.fail(Protocol(message))
	}
	out := []Event{}
	if frame.Message.Content != "" {
		out = append(out, Delta(frame.Message.Content))
	}
	if frame.Message.Thinking != "" {
		out = append(out, Thinking(frame.Message.Thinking))
	}
	for _, call := range frame.Message.Calls {
		if call.Function.Name == "" {
			continue
		}
		out = append(out, Called(ToolCall{
			Name:      call.Function.Name,
			Arguments: argumentsOf(call.Function.Arguments),
		}))
	}
	if !frame.Done {
		return out
	}
	p.used.PromptTokens = frame.Prompt
	p.used.CompletionTokens = frame.Eval
	p.used.DurationMs = millisOf(frame.EvalNS, frame.TotalNS)
	// A model that ran out of budget and one that came to a natural end
	// both say `done: true`; only `done_reason` tells them apart, and
	// not reading it is the same as taking half an answer for a whole
	// one. Only the one literal "length" means it; every other value —
	// "stop", "unload", or the field not being there at all — is a
	// whole answer.
	return append(out, p.finish(frame.Reason == "length")...)
}

// openAI says what one event of an OpenAI-shaped answer means.
func (p *Parser) openAI(payload string) []Event {
	var frame struct {
		Error any `json:"error"`
		Used  *struct {
			Prompt     *uint64 `json:"prompt_tokens"`
			Completion *uint64 `json:"completion_tokens"`
		} `json:"usage"`
		Choices []struct {
			Delta struct {
				Content  string `json:"content"`
				Thinking string `json:"reasoning"`
				// Which field the thinking comes in is not agreed
				// on: Ollama says `reasoning`, most of the clouds
				// say `reasoning_content`. Both are read.
				ThinkingToo string `json:"reasoning_content"`
				Calls       []struct {
					Index    *uint64 `json:"index"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"delta"`
			// A frame that is not the last says `null` here, so this
			// is a pointer: null and "" are not the same thing, and
			// only a string means the answer is over.
			Reason *string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal([]byte(payload), &frame); err != nil {
		return nil
	}
	if message := errMessage(frame.Error); message != "" {
		return p.fail(Protocol(message))
	}
	if frame.Used != nil {
		p.used.PromptTokens = frame.Used.Prompt
		p.used.CompletionTokens = frame.Used.Completion
	}
	// The first frame may carry nothing but a role, and a frame that
	// tells the usage may carry no choices at all: the first choice is
	// not there for the taking.
	if len(frame.Choices) == 0 {
		return nil
	}
	choice := frame.Choices[0]
	out := []Event{}
	if choice.Delta.Content != "" {
		out = append(out, Delta(choice.Delta.Content))
	}
	for _, thinking := range []string{
		choice.Delta.Thinking, choice.Delta.ThinkingToo,
	} {
		if thinking != "" {
			out = append(out, Thinking(thinking))
		}
	}
	for _, call := range choice.Delta.Calls {
		p.draft(call.Index, call.Function.Name, call.Function.Arguments)
	}
	if choice.Reason == nil {
		return out
	}
	// The arguments are only whole now that the answer is over: this is
	// when the calls are handed on.
	for _, draft := range p.drafts {
		if draft.name == "" {
			continue
		}
		out = append(out, Called(ToolCall{
			Name: draft.name, Arguments: draft.arguments,
		}))
	}
	p.drafts = nil
	return append(out, p.finish(*choice.Reason == "length")...)
}

// draft is one fragment of a tool call being added to the call it belongs
// to. Later fragments carry arguments only, and their name is empty: it
// must not wash away the name that came before it.
func (p *Parser) draft(index *uint64, name, arguments string) {
	at := 0
	if index != nil {
		at = int(*index)
	}
	for len(p.drafts) <= at {
		p.drafts = append(p.drafts, callDraft{})
	}
	if name != "" {
		p.drafts[at].name = name
	}
	p.drafts[at].arguments += arguments
}

// errMessage is the complaint of a frame that carries one: Ollama writes
// it as a string, the OpenAI-shaped endpoints as an object with a
// message inside it.
func errMessage(value any) string {
	switch said := value.(type) {
	case string:
		return said
	case map[string]any:
		if message, ok := said["message"].(string); ok {
			return message
		}
	}
	return ""
}

// argumentsOf is a tool call's arguments as text, whichever shape they
// came in: the OpenAI-shaped endpoints send them already as a string,
// Ollama sends them as an object.
func argumentsOf(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}
	// An object, then: it is already JSON text, and it is used as it
	// stands.
	if len(raw) > 0 && string(raw) == "null" {
		return ""
	}
	return string(raw)
}

// millisOf is nanoseconds as milliseconds, the first of them that was
// said.
func millisOf(values ...*uint64) *uint64 {
	for _, v := range values {
		if v != nil {
			out := *v / 1_000_000
			return &out
		}
	}
	return nil
}

// Events reads an answer and says what it meant, as it comes.
func Events(e Endpoint, pieces <-chan Piece) <-chan Event {
	out := make(chan Event)
	go func() {
		defer close(out)
		parser := NewParser(e)
		for piece := range pieces {
			for _, event := range parser.Piece(piece) {
				out <- event
			}
		}
		for _, event := range parser.End() {
			out <- event
		}
	}()
	return out
}
