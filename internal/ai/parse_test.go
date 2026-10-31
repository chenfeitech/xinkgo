package ai

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/helightxu/xinkgo/internal/config"
)

// ollamaEndpoint is a local model the parser reads a line at a time from.
func ollamaEndpoint() Endpoint {
	return Endpoint{Kind: config.ModelOllama, Model: "qwen"}
}

// cloudEndpoint is a model the parser reads an event at a time from.
func cloudEndpoint() Endpoint {
	return Endpoint{Kind: config.ModelOpenAICompatible, Model: "gpt"}
}

// read is what a parser makes of these lines. The answer is not said to
// have stopped: stopping is its own test.
func read(e Endpoint, lines ...string) []Event {
	parser := NewParser(e)
	out := []Event{}
	for _, line := range lines {
		out = append(out, parser.Piece(Piece{Data: []byte(line)})...)
	}
	return out
}

// kinds is what happened, in order.
func kinds(events []Event) []EventKind {
	out := make([]EventKind, 0, len(events))
	for _, e := range events {
		out = append(out, e.Kind)
	}
	return out
}

func TestParser_ollamaSaysItsWords(t *testing.T) {
	events := read(ollamaEndpoint(),
		`{"message":{"content":"hel"}}`,
		`{"message":{"content":"lo"}}`,
		`{"done":true,"prompt_eval_count":12,"eval_count":3,`+
			`"eval_duration":2500000000}`,
	)
	if got, want := kinds(events), []EventKind{
		EventDelta, EventDelta, EventUsage, EventDone,
	}; !sameKinds(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	if events[0].Text != "hel" || events[1].Text != "lo" {
		t.Fatalf("words = %q %q", events[0].Text, events[1].Text)
	}
	used := events[2].Used
	if used.PromptTokens == nil || *used.PromptTokens != 12 {
		t.Fatalf("prompt tokens = %v", used.PromptTokens)
	}
	if used.CompletionTokens == nil || *used.CompletionTokens != 3 {
		t.Fatalf("completion tokens = %v", used.CompletionTokens)
	}
	if used.DurationMs == nil || *used.DurationMs != 2500 {
		t.Fatalf("duration = %v, want 2500ms", used.DurationMs)
	}
}

func TestParser_theThinkingIsKeptApartFromTheAnswer(t *testing.T) {
	events := read(ollamaEndpoint(),
		`{"message":{"thinking":"let me see"}}`,
		`{"message":{"content":"yes"}}`,
		`{"done":true}`,
	)
	if got, want := kinds(events), []EventKind{
		EventThinking, EventDelta, EventUsage, EventDone,
	}; !sameKinds(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	if events[0].Text != "let me see" {
		t.Fatalf("thinking = %q", events[0].Text)
	}
}

func TestParser_ollamaGivesACallWhole(t *testing.T) {
	// Ollama gives the arguments as an object; they come out as text.
	events := read(ollamaEndpoint(),
		`{"message":{"tool_calls":[{"function":{"name":"read_note",`+
			`"arguments":{"path":"a.md"}}}]}}`,
	)
	if len(events) != 1 || events[0].Kind != EventToolCall {
		t.Fatalf("events = %v", events)
	}
	if events[0].Call.Name != "read_note" {
		t.Fatalf("name = %q", events[0].Call.Name)
	}
	if events[0].Call.Arguments != `{"path":"a.md"}` {
		t.Fatalf("arguments = %q", events[0].Call.Arguments)
	}
}

func TestParser_aCallWithNothingToSayHasNoArguments(t *testing.T) {
	events := read(ollamaEndpoint(),
		`{"message":{"tool_calls":[{"function":{"name":"noop"}}]}}`,
	)
	if len(events) != 1 || events[0].Call.Arguments != "" {
		t.Fatalf("events = %v", events)
	}
}

func TestParser_aCallWhoseArgumentsAreAlreadyTextComesThrough(t *testing.T) {
	events := read(ollamaEndpoint(),
		`{"message":{"tool_calls":[{"function":{"name":"noop",`+
			`"arguments":"{\"a\":1}"}}]}}`,
	)
	if len(events) != 1 ||
		events[0].Call.Arguments != `{"a":1}` {
		t.Fatalf("events = %v", events)
	}
}

func TestParser_aModelOutOfBudgetIsSaidToBeCutShort(t *testing.T) {
	// Both a natural end and running out of budget say done: true.
	// Only done_reason tells them apart, and it must be said before
	// Done, for a Done is where listening stops.
	events := read(ollamaEndpoint(), `{"done":true,"done_reason":"length"}`)
	if got, want := kinds(events), []EventKind{
		EventUsage, EventTruncated, EventDone,
	}; !sameKinds(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
}

func TestParser_aNaturalEndIsNotSaidToBeCutShort(t *testing.T) {
	for _, reason := range []string{"stop", "unload", ""} {
		line := `{"done":true}`
		if reason != "" {
			line = `{"done":true,"done_reason":"` + reason + `"}`
		}
		events := read(ollamaEndpoint(), line)
		for _, e := range events {
			if e.Kind == EventTruncated {
				t.Fatalf("%s: said cut short", reason)
			}
		}
	}
}

func TestParser_aLineThatIsNotJSONSaysNothing(t *testing.T) {
	if events := read(ollamaEndpoint(), "not json at all"); len(events) != 0 {
		t.Fatalf("events = %v, want nothing", kinds(events))
	}
}

func TestParser_whatTheModelComplainsOfIsSaid(t *testing.T) {
	events := read(ollamaEndpoint(), `{"error":"model not found"}`)
	if len(events) != 1 || events[0].Kind != EventFailed {
		t.Fatalf("events = %v", kinds(events))
	}
	if !strings.Contains(events[0].Err.Error(), "model not found") {
		t.Fatalf("err = %v", events[0].Err)
	}
}

func TestParser_theCloudGivesItsCallsAPieceAtATime(t *testing.T) {
	// The arguments come a few characters at a time, and only the index
	// says which call a fragment belongs to.
	events := read(cloudEndpoint(),
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":`+
			`{"name":"read_note","arguments":"{\"pa"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":`+
			`{"arguments":"th\":\"a.md\"}"}}]}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
	)
	if got, want := kinds(events), []EventKind{
		EventToolCall, EventUsage, EventDone,
	}; !sameKinds(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	// The call is handed on whole, and only once the answer is over.
	call := events[0].Call
	if call.Name != "read_note" {
		t.Fatalf("name = %q, want the one from the first fragment",
			call.Name)
	}
	if call.Arguments != `{"path":"a.md"}` {
		t.Fatalf("arguments = %q", call.Arguments)
	}
}

func TestParser_twoCallsAreKeptApart(t *testing.T) {
	events := read(cloudEndpoint(),
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":`+
			`{"name":"one"}},{"index":1,"function":{"name":"two"}}]}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
	)
	calls := []ToolCall{}
	for _, e := range events {
		if e.Kind == EventToolCall {
			calls = append(calls, e.Call)
		}
	}
	if len(calls) != 2 || calls[0].Name != "one" || calls[1].Name != "two" {
		t.Fatalf("calls = %v", calls)
	}
}

func TestParser_theThinkingComesInEitherField(t *testing.T) {
	// Which field carries it is not agreed on: some say reasoning,
	// some say reasoning_content. Both are read.
	for _, field := range []string{"reasoning", "reasoning_content"} {
		events := read(cloudEndpoint(),
			`{"choices":[{"delta":{"`+field+`":"hmm"}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
		)
		found := false
		for _, e := range events {
			if e.Kind == EventThinking && e.Text == "hmm" {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s: the thinking was not read", field)
		}
	}
}

func TestParser_theFirstFrameAndTheUsageFrameHaveNoWords(t *testing.T) {
	// The first frame may carry nothing but a role, and the usage frame
	// may carry no choices at all: neither is there for the taking.
	events := read(cloudEndpoint(),
		`{"choices":[{"delta":{"role":"assistant"}}]}`,
		`{"choices":[],"usage":{"prompt_tokens":4}}`,
		`{"choices":[{"delta":{"content":"hi"}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
	)
	if got, want := kinds(events), []EventKind{
		EventDelta, EventUsage, EventDone,
	}; !sameKinds(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	if events[1].Used.PromptTokens == nil || *events[1].Used.PromptTokens != 4 {
		t.Fatalf("prompt tokens = %v", events[1].Used.PromptTokens)
	}
}

func TestParser_theSentinelEndsTheAnswer(t *testing.T) {
	events := read(cloudEndpoint(),
		`{"choices":[{"delta":{"content":"hi"}}]}`,
		"[DONE]",
		`{"choices":[{"delta":{"content":"late"}}]}`,
	)
	if got, want := kinds(events), []EventKind{
		EventDelta, EventUsage, EventDone,
	}; !sameKinds(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	// What comes after the sentinel has nowhere to land.
	for _, e := range events {
		if e.Kind == EventDelta && e.Text == "late" {
			t.Fatal("words came after the answer was over")
		}
	}
}

func TestParser_theCloudOutOfBudgetIsSaidToBeCutShort(t *testing.T) {
	events := read(cloudEndpoint(),
		`{"choices":[{"delta":{},"finish_reason":"length"}]}`)
	if got, want := kinds(events), []EventKind{
		EventUsage, EventTruncated, EventDone,
	}; !sameKinds(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
}

func TestParser_theCloudComplainsInAnObject(t *testing.T) {
	events := read(cloudEndpoint(),
		`{"error":{"message":"bad key","type":"auth"}}`)
	if len(events) != 1 || events[0].Kind != EventFailed {
		t.Fatalf("events = %v", kinds(events))
	}
	if !strings.Contains(events[0].Err.Error(), "bad key") {
		t.Fatalf("err = %v", events[0].Err)
	}
}

func TestEnd_anAnswerThatStoppedWithoutSayingSoBrokeOff(t *testing.T) {
	parser := NewParser(ollamaEndpoint())
	_ = parser.Piece(Piece{Data: []byte(`{"message":{"content":"hel"}}`)})
	events := parser.End()
	if len(events) != 1 || events[0].Kind != EventFailed {
		t.Fatalf("events = %v", kinds(events))
	}
	if !errors.Is(events[0].Err, ErrTruncated) {
		t.Fatalf("err = %v, want %v", events[0].Err, ErrTruncated)
	}
}

func TestEnd_anAnswerThatWasOverHasNothingMoreToSay(t *testing.T) {
	parser := NewParser(ollamaEndpoint())
	_ = parser.Piece(Piece{Data: []byte(`{"done":true}`)})
	if events := parser.End(); len(events) != 0 {
		t.Fatalf("events = %v, want nothing", kinds(events))
	}
}

func TestEvents_theAnswerIsReadAsItComes(t *testing.T) {
	pieces := make(chan Piece, 4)
	pieces <- Piece{Data: []byte(`{"message":{"content":"hel"}}`)}
	pieces <- Piece{Data: []byte(`{"message":{"content":"lo"}}`)}
	pieces <- Piece{Data: []byte(`{"done":true}`)}
	close(pieces)

	events := []Event{}
	for e := range Events(ollamaEndpoint(), pieces) {
		events = append(events, e)
	}
	if got, want := kinds(events), []EventKind{
		EventDelta, EventDelta, EventUsage, EventDone,
	}; !sameKinds(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
}

func TestFailure_whatWentWrongIsSaidInWordsAPersonCanActOn(t *testing.T) {
	e := ollamaEndpoint()
	cases := []struct {
		name string
		err  error
		want error
	}{
		{"a key that is refused", Status{Code: 401}, ErrAuth},
		{"a model that is busy", Status{Code: 429}, ErrRateLimit},
		{"a model that timed out", context.DeadlineExceeded, ErrTimeout},
		{"a model that said nothing", ErrSilent, ErrTimeout},
		{"a model that was let go", context.Canceled, ErrCanceled},
		{"a line that went dead", ErrTruncated, ErrTruncated},
		{"an address that is not there", Network("no such host"), nil},
	}
	for _, c := range cases {
		got := failure(c.err, e.Model)
		if c.want != nil && !errors.Is(got, c.want) {
			t.Fatalf("%s: got %v, want %v", c.name, got, c.want)
		}
		if c.want == nil && got == nil {
			t.Fatalf("%s: nothing was said", c.name)
		}
	}
}

func TestFailure_aModelTheServerHasNeverHeardOfIsNamed(t *testing.T) {
	err := failure(Status{Code: 404, Body: "no such model"}, "qwen")
	if !strings.Contains(err.Error(), "qwen") {
		t.Fatalf("err = %v, want the model named", err)
	}
}

func TestFailure_aSlowAnswerIsATimeout(t *testing.T) {
	err := failure(&net.OpError{
		Op: "dial", Err: &timeoutError{},
	}, "qwen")
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("err = %v, want %v", err, ErrTimeout)
	}
}

func TestRetryable_whatIsWorthAskingTwice(t *testing.T) {
	worth := []error{ErrRateLimit, ErrTimeout, ErrTruncated, Network("x")}
	for _, err := range worth {
		if !Retryable(err) {
			t.Fatalf("%v: is worth asking again", err)
		}
	}
	useless := []error{ErrAuth, ErrNoModel, Protocol("gibberish"),
		NoSuchModel("qwen"), ErrCanceled}
	for _, err := range useless {
		if Retryable(err) {
			t.Fatalf("%v: comes back the same however often it is asked",
				err)
		}
	}
}

func TestArgumentsOf_everyShapeComesOutAsText(t *testing.T) {
	cases := []struct{ in, want string }{
		{`{"a":1}`, `{"a":1}`},
		{`"already text"`, "already text"},
		{`null`, ""},
		{``, ""},
	}
	for _, c := range cases {
		if got := argumentsOf([]byte(c.in)); got != c.want {
			t.Fatalf("argumentsOf(%s) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestKinds_saidPlainly(t *testing.T) {
	if EventDelta.String() != "delta" || EventDone.String() != "done" {
		t.Fatalf("kinds = %v %v", EventDelta, EventDone)
	}
}

// sameKinds is whether two answers happened the same way.
func sameKinds(got, want []EventKind) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// timeoutError is a network error that came of waiting too long.
type timeoutError struct{}

func (timeoutError) Error() string   { return "i/o timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }
