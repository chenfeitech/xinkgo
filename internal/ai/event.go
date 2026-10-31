package ai

// EventKind is what happened in an answer.
type EventKind int

// The things that happen in an answer.
const (
	// EventDelta is a piece of the answer's own words.
	EventDelta EventKind = iota
	// EventThinking is a piece of the model's thinking, which is not
	// part of the answer and is kept apart from it.
	EventThinking
	// EventToolCall is the model asking for a tool to be run.
	EventToolCall
	// EventToolResult is what a tool came back with, after the model
	// asked for it: the same words that are handed back to the model.
	EventToolResult
	// EventUsage is what the answer cost.
	EventUsage
	// EventTruncated is a model that ran out of budget and stopped: it
	// came to an end of its own, and what it gave is only half of what
	// it would have said. It is said before EventDone, for a Done is
	// where the one who listens stops listening.
	EventTruncated
	// EventDone is the answer being over.
	EventDone
	// EventFailed is the answer going wrong.
	EventFailed
)

// String is the kind, said plainly.
func (k EventKind) String() string {
	switch k {
	case EventDelta:
		return "delta"
	case EventThinking:
		return "thinking"
	case EventToolCall:
		return "tool call"
	case EventToolResult:
		return "tool result"
	case EventUsage:
		return "usage"
	case EventTruncated:
		return "truncated"
	case EventDone:
		return "done"
	case EventFailed:
		return "failed"
	}
	return "nothing"
}

// Event is one thing that happened in an answer. Which of the fields is
// the one that holds it, the Kind says.
type Event struct {
	Kind EventKind
	// Text is the words: of a delta, of a thinking.
	Text string
	// Call is the tool the model asks for.
	Call ToolCall
	// Result is what a tool came back with.
	Result Result
	// Used is what the answer cost.
	Used Usage
	// Err is what went wrong.
	Err error
}

// Delta is a piece of the answer's own words.
func Delta(text string) Event {
	return Event{Kind: EventDelta, Text: text}
}

// Thinking is a piece of the model's thinking.
func Thinking(text string) Event {
	return Event{Kind: EventThinking, Text: text}
}

// Called is the model asking for a tool.
func Called(call ToolCall) Event {
	return Event{Kind: EventToolCall, Call: call}
}

// Found is what a tool came back with.
func Found(result Result) Event {
	return Event{Kind: EventToolResult, Result: result}
}

// Used is what the answer cost.
func Used(usage Usage) Event { return Event{Kind: EventUsage, Used: usage} }

// Truncated is a model that stopped because it had said as much as it was
// allowed to.
func Truncated() Event { return Event{Kind: EventTruncated} }

// Done is the answer being over.
func Done() Event { return Event{Kind: EventDone} }

// Failed is the answer going wrong.
func Failed(err error) Event { return Event{Kind: EventFailed, Err: err} }

// ToolCall is the model asking for a tool to be run.
type ToolCall struct {
	Name string
	// Arguments is the tool's arguments as one piece of JSON text.
	// Ollama gives them as an object and the OpenAI-shaped endpoints
	// give them as a string; both come out of here as text, and a tool
	// with nothing to be told is the empty string.
	Arguments string
}

// Usage is what an answer cost. What a model does not count, it does not
// say: nothing here is a zero, it is an unsaid thing.
type Usage struct {
	PromptTokens     *uint64
	CompletionTokens *uint64
	DurationMs       *uint64
}
