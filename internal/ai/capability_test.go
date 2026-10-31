package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/helightxu/xinkgo/internal/config"
)

func TestInfo_everyKnownCapabilityIsSaidPlainly(t *testing.T) {
	for _, id := range Known {
		if info := Info(id); info.Label == "" || info.Meaning == "" {
			t.Errorf("%s: said nothing (%+v)", id, info)
		}
	}
}

func TestInfo_aCapabilityTheAppHasNeverHeardOfIsKept(t *testing.T) {
	info := Info("embedding")
	if !strings.Contains(info.Label, "未知") {
		t.Fatalf("label = %q, want it said to be unknown", info.Label)
	}
	// Unknown is not the same as one of the known ones.
	if info == Info(CapTools) {
		t.Fatal("an unknown capability reads as a known one")
	}
}

func TestInfer_aLocalModelIsNotGuessedAt(t *testing.T) {
	// A local Ollama is asked what it can do; a guess is not better
	// than that.
	if got := Infer(config.ModelOllama, "", "qwen3"); len(got) != 0 {
		t.Fatalf("inferred %v, want nothing", got)
	}
}

func TestInfer_theNameSaysWhatItIs(t *testing.T) {
	cases := []struct {
		id   string
		want []string
	}{
		{"deepseek-reasoner", []string{CapCompletion, CapThinking}},
		{"qwen3-thinking", []string{CapCompletion, CapThinking}},
		{"deepseek-r1", []string{CapCompletion, CapThinking}},
		{"qwen2.5-vl-7b", []string{CapVision}},
	}
	for _, c := range cases {
		got := Infer(config.ModelOpenAICompatible, "", c.id)
		if !sameCaps(got, c.want) {
			t.Errorf("%s: inferred %v, want %v", c.id, got, c.want)
		}
	}
}

func TestInfer_aWellKnownAddressOrNameIsEnoughToGuess(t *testing.T) {
	for _, one := range []struct{ url, id string }{
		{"https://api.deepseek.com/v1", "whatever"},
		{"https://open.bigmodel.cn/api/paas/v4", "whatever"},
		{"https://api.moonshot.cn/v1", "whatever"},
		{"https://dashscope.aliyuncs.com/compatible-mode/v1", "x"},
		{"https://api.openai.com/v1", "whatever"},
		{"https://example.com/v1", "glm-4"},
		{"https://example.com/v1", "gpt-4o"},
	} {
		got := Infer(config.ModelOpenAICompatible, one.url, one.id)
		if !sameCaps(got, []string{CapCompletion, CapTools}) {
			t.Errorf("%s %s: inferred %v, want completion and tools",
				one.url, one.id, got)
		}
	}
}

func TestInfer_anAddressTheAppDoesNotKnowIsNotGuessedAt(t *testing.T) {
	if got := Infer(config.ModelOpenAICompatible,
		"https://model.example.com/v1", "some-model"); len(got) != 0 {
		t.Fatalf("inferred %v, want nothing rather than a wrong guess",
			got)
	}
}

func TestInfer_theNameIsReadWithoutRegardToCase(t *testing.T) {
	if got := Infer(config.ModelOpenAICompatible, "",
		"DeepSeek-Reasoner"); !sameCaps(got,
		[]string{CapCompletion, CapThinking}) {
		t.Fatalf("inferred %v", got)
	}
}

func TestEffective_whatTheUserChoseByHandWins(t *testing.T) {
	m := config.ModelEntry{
		ID: "qwen", Kind: config.ModelOpenAICompatible,
		BaseURL: "https://api.openai.com/v1",
		// The guess would say completion and tools; the user says
		// none of them, and none of them it is.
		CapabilitiesPinned: true,
	}
	if got := Effective(m); len(got) != 0 {
		t.Fatalf("got %v, want the user's own empty list", got)
	}
	m.Capabilities = []string{CapVision}
	if got := Effective(m); !sameCaps(got, []string{CapVision}) {
		t.Fatalf("got %v, want the user's own", got)
	}
}

func TestEffective_whatIsKnownOfTheModelComesBeforeTheGuess(t *testing.T) {
	m := config.ModelEntry{
		ID: "qwen", Kind: config.ModelOllama,
		// Pulled from the server, and not guessed at.
		Capabilities: []string{CapCompletion, CapThinking},
	}
	if got := Effective(m); !sameCaps(got,
		[]string{CapCompletion, CapThinking}) {
		t.Fatalf("got %v", got)
	}
}

func TestEffective_theGuessIsMadeWhenNothingElseIsKnown(t *testing.T) {
	m := config.ModelEntry{
		ID: "gpt-4o", Kind: config.ModelOpenAICompatible,
		BaseURL: "https://api.openai.com/v1",
	}
	if got := Effective(m); !sameCaps(got,
		[]string{CapCompletion, CapTools}) {
		t.Fatalf("got %v, want the guess", got)
	}
	// The guess is not written down: asking again guesses again.
	if len(m.Capabilities) != 0 {
		t.Fatalf("the guess was written into the model: %v",
			m.Capabilities)
	}
}

func TestHostOf_whatABaseURLNames(t *testing.T) {
	cases := map[string]string{
		"https://api.openai.com/v1":            "api.openai.com",
		"http://localhost:11434":               "localhost",
		"https://dashscope.aliyuncs.com/a?b=c": "dashscope.aliyuncs.com",
		"localhost:11434":                      "localhost",
		"":                                     "",
	}
	for in, want := range cases {
		if got := hostOf(in); got != want {
			t.Errorf("hostOf(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestProbe_aModelThatAnswersIsSaidToBeThere(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"done":true}` + "\n"))
		}))
	defer server.Close()

	cfg := config.DefaultAi()
	m := config.ModelEntry{
		ID: "qwen", Kind: config.ModelOllama, Enabled: true,
		BaseURL: server.URL,
	}
	got := Try(context.Background(), cfg, m, false)
	if !got.OK {
		t.Fatalf("probe = %+v, want it said to be there", got)
	}
	if got.Tools {
		t.Fatal("no tool was offered, and one was said to be called")
	}
}

func TestProbe_aModelThatCallsTheToolIsSaidTo(t *testing.T) {
	var sent []byte
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			buf := make([]byte, 4096)
			n, _ := r.Body.Read(buf)
			sent = buf[:n]
			_, _ = w.Write([]byte(
				`{"message":{"content":"hi"},"done":true}` + "\n"))
		}))
	defer server.Close()

	cfg := config.DefaultAi()
	m := config.ModelEntry{
		ID: "qwen", Kind: config.ModelOllama, Enabled: true,
		BaseURL: server.URL,
	}
	got := Try(context.Background(), cfg, m, true)
	if !got.OK {
		t.Fatalf("probe = %+v", got)
	}
	// The tool is sent whether the model calls it or not: it cannot
	// call what it was not given.
	if !strings.Contains(string(sent), "noop") {
		t.Fatalf("sent %s, want the tool in it", sent)
	}
	if got.Tools {
		t.Fatal("the model called nothing, and was said to have " +
			"called the tool")
	}
}

func TestProbe_aModelThatAsksForTheToolIsSaidTo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(
				`{"message":{"tool_calls":[{"function":` +
					`{"name":"noop","arguments":{}}}]}}` + "\n"))
			_, _ = w.Write([]byte(`{"done":true}` + "\n"))
		}))
	defer server.Close()

	cfg := config.DefaultAi()
	m := config.ModelEntry{
		ID: "qwen", Kind: config.ModelOllama, Enabled: true,
		BaseURL: server.URL,
	}
	got := Try(context.Background(), cfg, m, true)
	if !got.Tools {
		t.Fatalf("probe = %+v, want it said the model calls tools",
			got)
	}
}

func TestProbe_aModelThatCannotBeReachedSaysSo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
	defer server.Close()

	cfg := config.DefaultAi()
	cfg.Retry = config.AiRetryConfig{MaxAttempts: 1, BackoffMs: 1}
	m := config.ModelEntry{
		ID: "qwen", Kind: config.ModelOllama, Enabled: true,
		BaseURL: server.URL,
	}
	got := Try(context.Background(), cfg, m, false)
	if got.OK || got.Err == nil {
		t.Fatalf("probe = %+v, want it said to have failed", got)
	}
	if d := got.Took; d <= 0 || d > 30*time.Second {
		t.Fatalf("took %v", d)
	}
}

func TestProbe_aModelThatBreaksOffIsNotSaidToBeThere(t *testing.T) {
	server := quietServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"message":{"content":"h"}}` + "\n"))
			w.(http.Flusher).Flush()
			panic("broken off")
		}))
	defer server.Close()

	cfg := config.DefaultAi()
	cfg.Retry = config.AiRetryConfig{MaxAttempts: 1, BackoffMs: 1}
	m := config.ModelEntry{
		ID: "qwen", Kind: config.ModelOllama, Enabled: true,
		BaseURL: server.URL,
	}
	got := Try(context.Background(), cfg, m, false)
	if got.OK {
		t.Fatalf("probe = %+v, want it said to have broken off", got)
	}
}

// sameCaps is whether two lists name the same capabilities.
func sameCaps(got, want []string) bool {
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
