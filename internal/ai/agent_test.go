package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/index"
	"github.com/helightxu/xinkgo/internal/search"
	"github.com/helightxu/xinkgo/internal/skill"
)

// toolHost is a vault of one note, for a loop to look through.
type toolHost struct {
	root string
}

func (h toolHost) ReadNote(path string) (string, bool) {
	if filepath.Clean(path) == filepath.Join(h.root, "a.md") {
		return "the note says: hello", true
	}
	return "", false
}

func (h toolHost) Search(query string, limit int) []search.Hit {
	return nil
}

func (h toolHost) Tags() []index.TagCount { return nil }

func (h toolHost) Backlinks(path string) []index.LinkRef { return nil }

func (h toolHost) Outline(path string) []string { return nil }

func (h toolHost) ListNotes(limit int) []NoteRef {
	return []NoteRef{{Path: filepath.Join(h.root, "a.md"), Title: "A"}}
}

func (h toolHost) Root() string { return h.root }

func (h toolHost) Skill(name string) (*skill.Skill, bool) {
	return nil, false
}

// loopCfg is the settings a loop is run with: no waiting between tries,
// and one tool answer long enough to read.
func loopCfg() config.AiConfig {
	cfg := config.DefaultAi()
	cfg.Retry = config.AiRetryConfig{MaxAttempts: 1, BackoffMs: 1}
	cfg.MaxToolChars = 4000
	cfg.MaxSteps = 4
	return cfg
}

// eventsOf is every event a loop gave, waited for but not forever.
func eventsOf(t *testing.T, events <-chan Event) []Event {
	t.Helper()
	out := []Event{}
	for {
		select {
		case event, open := <-events:
			if !open {
				return out
			}
			out = append(out, event)
		case <-time.After(5 * time.Second):
			t.Fatal("the loop never finished")
			return out
		}
	}
}

// serverOf is a model that answers with what the test says, one request
// at a time.
func serverOf(t *testing.T, answers ...string) (*httptest.Server, *int) {
	t.Helper()
	asked := 0
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if asked >= len(answers) {
				t.Errorf("the model was asked more than %d times",
					len(answers))
				_, _ = w.Write([]byte(`{"done":true}` + "\n"))
				return
			}
			answer := answers[asked]
			asked++
			_, _ = w.Write([]byte(answer))
		}))
	return server, &asked
}

// TestLoop_aModelThatReachesForNothingIsAskedOnce checks that a loop
// with no tool to run is the same as a single asking: no worse, and no
// longer.
func TestLoop_aModelThatReachesForNothingIsAskedOnce(t *testing.T) {
	server, asked := serverOf(t, `{"message":{"content":"here"}}`+"\n"+
		`{"done":true}`+"\n")
	defer server.Close()

	cfg := loopCfg()
	e := endpoint(config.ModelOllama, server.URL)
	events := eventsOf(t, Loop(context.Background(), cfg, e,
		toolHost{root: t.TempDir()}, Request{
			Messages: []Message{{Role: User, Content: "hi"}},
		}, 4))
	kindsOf(t, events, []EventKind{EventDelta, EventUsage, EventDone})
	if *asked != 1 {
		t.Fatalf("asked %d times, want once", *asked)
	}
}

// TestLoop_whatAToolFoundIsGivenBackToTheModel checks the whole round:
// the model reaches, the tool runs, and what it found goes back with the
// next asking.
func TestLoop_whatAToolFoundIsGivenBackToTheModel(t *testing.T) {
	root := t.TempDir()
	call := `{"message":{"tool_calls":[{"function":{"name":"read_note",` +
		`"arguments":{"path":"` + filepath.Join(root, "a.md") +
		`"}}}]},"done":false}` + "\n" + `{"done":true}` + "\n"
	server, asked := serverOf(t, call,
		`{"message":{"content":"it says hello"}}`+"\n"+`{"done":true}`+"\n")
	defer server.Close()

	cfg := loopCfg()
	e := endpoint(config.ModelOllama, server.URL)
	events := eventsOf(t, Loop(context.Background(), cfg, e,
		toolHost{root: root}, Request{
			Messages: []Message{{Role: User, Content: "what does a say"}},
		}, 4))
	kindsOf(t, events, []EventKind{
		EventToolCall, EventToolResult, EventDelta, EventUsage,
		EventDone,
	})
	if *asked != 2 {
		t.Fatalf("asked %d times, want twice", *asked)
	}
	// What the tool found is handed on, the same words the model gets.
	found := false
	for _, event := range events {
		if event.Kind == EventToolResult &&
			strings.Contains(event.Result.Output, "the note says") {
			found = true
		}
	}
	if !found {
		t.Fatalf("events = %v, want what the tool found", events)
	}
}

// TestLoop_aModelThatKeepsReachingIsStopped checks that a model which
// never comes to an answer is given up on, and the user told why.
func TestLoop_aModelThatKeepsReachingIsStopped(t *testing.T) {
	root := t.TempDir()
	call := `{"message":{"tool_calls":[{"function":{"name":"read_note",` +
		`"arguments":{"path":"` + filepath.Join(root, "a.md") +
		`"}}}]},"done":false}` + "\n" + `{"done":true}` + "\n"
	answers := make([]string, 6)
	for i := range answers {
		answers[i] = call
	}
	server, _ := serverOf(t, answers...)
	defer server.Close()

	cfg := loopCfg()
	cfg.Retry = config.AiRetryConfig{MaxAttempts: 1, BackoffMs: 1}
	e := endpoint(config.ModelOllama, server.URL)
	events := eventsOf(t, Loop(context.Background(), cfg, e,
		toolHost{root: root}, Request{
			Messages: []Message{{Role: User, Content: "go on"}},
		}, 3))
	last := events[len(events)-1]
	if last.Kind != EventFailed || last.Err != ErrSteps {
		t.Fatalf("last = %+v, want %v", last, ErrSteps)
	}
}

// TestLoop_anAnswerThatBrokeOffIsNotAskedAgain checks that a stream that
// went dead ends the loop, rather than being asked again.
func TestLoop_anAnswerThatBrokeOffIsNotAskedAgain(t *testing.T) {
	server := quietServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"message":{"content":"hel"}}` + "\n"))
			w.(http.Flusher).Flush()
			panic("broken off")
		}))
	defer server.Close()

	cfg := loopCfg()
	e := endpoint(config.ModelOllama, server.URL)
	events := eventsOf(t, Loop(context.Background(), cfg, e,
		toolHost{root: t.TempDir()}, Request{
			Messages: []Message{{Role: User, Content: "hi"}},
		}, 4))
	last := events[len(events)-1]
	if last.Kind != EventFailed {
		t.Fatalf("last = %+v, want a failure", last)
	}
	// The words that came are not thrown away.
	if len(events) < 2 || events[0].Kind != EventDelta {
		t.Fatalf("events = %v, want what came before it broke", events)
	}
}

// TestLoop_noStepsAtAllIsStillOneAsking checks that a step limit of
// nothing at all is taken as one, and not as none.
func TestLoop_noStepsAtAllIsStillOneAsking(t *testing.T) {
	server, asked := serverOf(t, `{"done":true}`+"\n")
	defer server.Close()

	cfg := loopCfg()
	e := endpoint(config.ModelOllama, server.URL)
	eventsOf(t, Loop(context.Background(), cfg, e,
		toolHost{root: t.TempDir()}, Request{
			Messages: []Message{{Role: User, Content: "hi"}},
		}, 0))
	if *asked != 1 {
		t.Fatalf("asked %d times, want once", *asked)
	}
}

// kindsOf is what happened, in that order.
func kindsOf(t *testing.T, events []Event, want []EventKind) {
	t.Helper()
	got := make([]EventKind, 0, len(events))
	for _, event := range events {
		got = append(got, event.Kind)
	}
	if len(got) != len(want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events = %v, want %v", got, want)
		}
	}
}
