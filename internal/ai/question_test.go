package ai

import (
	"context"
	"testing"

	"github.com/helightxu/xinkgo/internal/config"
)

func TestQuestionFromCall(t *testing.T) {
	q := questionFromCall(ToolCall{Name: ToolAskUser, Arguments: `{
		"question":"Which tone?","header":"Tone",
		"options":["Formal","Casual",""],"allow_other":true}`})
	if q.Text != "Which tone?" || q.Header != "Tone" ||
		len(q.Options) != 2 || !q.AllowOther {
		t.Fatalf("question = %+v", q)
	}
	if q := questionFromCall(ToolCall{Name: ToolAskUser,
		Arguments: "not json"}); q.Text == "" {
		t.Error("an unreadable call left the user with no question")
	}
}

func TestLoop_askUserEndsTheTurnWithACard(t *testing.T) {
	call := `{"message":{"tool_calls":[{"function":{"name":"ask_user",` +
		`"arguments":{"question":"Which tone?","options":["Formal","Casual"]` +
		`}}}]},"done":false}` + "\n" + `{"done":true}` + "\n"
	server, asked := serverOf(t, call)
	defer server.Close()

	cfg := loopCfg()
	e := endpoint(config.ModelOllama, server.URL)
	events := eventsOf(t, Loop(context.Background(), cfg, e,
		toolHost{root: t.TempDir()}, Request{
			Messages: []Message{{Role: User, Content: "hi"}},
		}, 4))
	kindsOf(t, events, []EventKind{EventQuestion, EventDone})
	if events[0].Asked.Text != "Which tone?" {
		t.Errorf("question = %+v", events[0].Asked)
	}
	if *asked != 1 {
		t.Errorf("asked %d times, want the turn to stop at the question",
			*asked)
	}
}
