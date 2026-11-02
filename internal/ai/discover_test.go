package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/helightxu/xinkgo/internal/config"
)

// tagsBody is what a local Ollama says when it is asked what it holds:
// two models, one of them with a picture reader in it.
const tagsBody = `{"models":[
	{"name":"qwen3:0.6b","model":"qwen3:0.6b","size":522928832,
	 "modified_at":"2026-10-02T03:04:05.6Z",
	 "details":{"family":"qwen3","families":["qwen3"],
	 "parameter_size":"0.6B","quantization_level":"Q8_0"}},
	{"name":"llava:7b","model":"llava:7b","size":4700000000,
	 "details":{"family":"llama","families":["llama","clip"],
	 "parameter_size":"7B"}}
]}`

// showBody is what it says about one model: what it is made of, how much
// of a conversation it holds, and what it can do.
const showBody = `{"details":{"family":"qwen3","families":["qwen3"],
	"parameter_size":"0.6B","quantization_level":"Q8_0"},
	"model_info":{"general.architecture":"qwen3",
	"qwen3.context_length":32768,"qwen3.embedding_length":1024},
	"capabilities":["completion","tools","thinking"]}`

// ollamaServer is a server that answers as a local Ollama answers: what it
// holds, and what one of them can do.
func ollamaServer(t *testing.T, show string) *httptest.Server {
	t.Helper()
	return quietServer(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/api/tags") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(tagsBody))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/api/show") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(show))
			return
		}
		http.NotFound(w, r)
	})
}

// TestOllamaModels_whatTheMachineHoldsAndWhatEachCanDo checks the whole
// asking: both models come back, sorted, and the capabilities of one are
// what the server said of it, in the names the app uses.
func TestOllamaModels_whatTheMachineHoldsAndWhatEachCanDo(t *testing.T) {
	server := ollamaServer(t, showBody)
	defer server.Close()

	got, err := OllamaModels(context.Background(), server.URL+"/")
	if err != nil {
		t.Fatalf("asking went wrong: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("models = %d, want two", len(got))
	}
	if got[0].Name != "llava:7b" || got[1].Name != "qwen3:0.6b" {
		t.Fatalf("names = %q %q, want them sorted",
			got[0].Name, got[1].Name)
	}
	one := got[1]
	if one.Size != 522928832 {
		t.Fatalf("size = %d, want what the server said", one.Size)
	}
	if one.Family != "qwen3" || one.ParameterSize != "0.6B" ||
		one.Quantization != "Q8_0" {
		t.Fatalf("made of = %q %q %q, want what the server said",
			one.Family, one.ParameterSize, one.Quantization)
	}
	if one.ContextLimit != 32768 {
		t.Fatalf("context = %d, want 32768", one.ContextLimit)
	}
	want := []string{CapCompletion, CapTools, CapThinking}
	if strings.Join(one.Capabilities, ",") != strings.Join(want, ",") {
		t.Fatalf("capabilities = %v, want %v", one.Capabilities, want)
	}
	if one.Guessed {
		t.Fatal("said to be guessed, but the server said")
	}
	if one.Modified.Year() != 2026 {
		t.Fatalf("modified = %v, want it read out", one.Modified)
	}
}

// TestOllamaModels_anOldServerThatSaysNothingIsStillListed checks that a
// server too old to say what a model can do is not left out: what is
// guessed at is marked as guessed, and the model is still there.
func TestOllamaModels_anOldServerThatSaysNothingIsStillListed(t *testing.T) {
	server := ollamaServer(t, `{"details":{"family":"llama",
		"families":["llama","clip"]}}`)
	defer server.Close()

	got, err := OllamaModels(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("asking went wrong: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("models = %d, want two", len(got))
	}
	// The one with a picture reader in it is guessed to see.
	if !hasCapability(got[0].Capabilities, CapVision) {
		t.Fatalf("capabilities = %v, want it to see",
			got[0].Capabilities)
	}
	if !got[0].Guessed || !got[1].Guessed {
		t.Fatalf("guessed = %v %v, want both marked as guessed",
			got[0].Guessed, got[1].Guessed)
	}
}

// TestOllamaModels_oneThatCannotBeAskedIsSaid checks that a machine with
// no Ollama on it is said to have none, and said why.
func TestOllamaModels_oneThatCannotBeAskedIsSaid(t *testing.T) {
	server := ollamaServer(t, showBody)
	url := server.URL
	server.Close()

	if _, err := OllamaModels(context.Background(), url); err == nil {
		t.Fatal("nothing was said of a machine that cannot be asked")
	}
}

// TestOpenAIModels_whatTheAddressHolds checks that the models of an
// endpoint that speaks as the clouds do are read out of its answer, and
// told apart by name.
func TestOpenAIModels_whatTheAddressHolds(t *testing.T) {
	server := quietServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-x" {
			t.Errorf("asked as %q, want the key sent", got)
		}
		_, _ = w.Write([]byte(`{"object":"list","data":[
			{"id":"gpt-4o-mini"},{"id":"gpt-4o"},{"id":""}]}`))
	})
	defer server.Close()

	m := config.ModelEntry{ID: "whatever",
		Kind:    config.ModelOpenAICompatible,
		BaseURL: server.URL + "/v1", PlainKey: "sk-x",
		ApiKeySource: config.KeyPlain}
	got, err := OpenAIModels(context.Background(), m, config.DefaultAi())
	if err != nil {
		t.Fatalf("asking went wrong: %v", err)
	}
	if len(got) != 2 || got[0] != "gpt-4o" || got[1] != "gpt-4o-mini" {
		t.Fatalf("held = %v, want the two named, sorted", got)
	}
}

// TestOpenAIModels_oneThatWillNotSayIsSaid checks that an address that
// does not answer is said, and not taken for one that holds nothing.
func TestOpenAIModels_oneThatWillNotSayIsSaid(t *testing.T) {
	server := quietServer(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	})
	defer server.Close()

	m := config.ModelEntry{ID: "whatever",
		Kind:    config.ModelOpenAICompatible,
		BaseURL: server.URL}
	if _, err := OpenAIModels(context.Background(), m,
		config.DefaultAi()); err == nil {
		t.Fatal("nothing was said of an address that would not say")
	}
}

// TestContextLengthOf_howMuchItHolds checks the one key that is looked for
// by how it ends: it is named after what the model is built of, and a
// number given as words is read as well as one given as a number.
func TestContextLengthOf_howMuchItHolds(t *testing.T) {
	info := map[string]json.RawMessage{
		"qwen3.context_length":   json.RawMessage(`32768`),
		"qwen3.embedding_length": json.RawMessage(`1024`),
		"general.architecture":   json.RawMessage(`"qwen3"`),
	}
	if got := contextLengthOf(info); got != 32768 {
		t.Fatalf("context = %d, want 32768", got)
	}
	words := map[string]json.RawMessage{
		"llama.context_length": json.RawMessage(`"4096"`),
	}
	if got := contextLengthOf(words); got != 4096 {
		t.Fatalf("context = %d, want 4096", got)
	}
	if got := contextLengthOf(nil); got != 0 {
		t.Fatalf("context = %d, want 0 when nothing was said", got)
	}
}

// TestCapabilityOfOllama_whatTheAppCallsIt checks the names Ollama uses
// against the ones the app shows.
func TestCapabilityOfOllama_whatTheAppCallsIt(t *testing.T) {
	cases := map[string]string{
		"completion": CapCompletion,
		"tools":      CapTools,
		"thinking":   CapThinking,
		"vision":     CapVision,
		"embedding":  "embedding",
	}
	for in, want := range cases {
		if got := CapabilityOfOllama(in); got != want {
			t.Errorf("%q = %q, want %q", in, got, want)
		}
	}
}

// TestMeasured_whatTryingReallyShowed checks that a try is believed over
// what was thought before, where the try can say something: that it
// answers, and whether it calls tools.
func TestMeasured_whatTryingReallyShowed(t *testing.T) {
	got := Measured(Probe{OK: true, Tools: true, Took: time.Second},
		[]string{CapVision, CapTools})
	if !hasCapability(got, CapCompletion) ||
		!hasCapability(got, CapTools) {
		t.Fatalf("measured = %v, want it to answer and call tools",
			got)
	}
	if !hasCapability(got, CapVision) {
		t.Fatalf("measured = %v, want what a try cannot see kept", got)
	}
	// A model that cannot be reached has been shown nothing at all.
	was := []string{CapVision}
	if got := Measured(Probe{}, was); !hasCapability(got, CapVision) {
		t.Fatalf("measured = %v, want it left alone", got)
	}
	// A model that answers but calls no tools does not call tools.
	got = Measured(Probe{OK: true}, nil)
	if hasCapability(got, CapTools) {
		t.Fatalf("measured = %v, want no tools", got)
	}
}

// TestEndpoint_aModelTheMachineHoldsIsAskedWhereTheMachineIs checks the
// one thing a model found on the machine is for: it names no address of
// its own, and is asked where the machine is — so that the address in the
// settings is the only one there is to change.
func TestEndpoint_aModelTheMachineHoldsIsAskedWhereTheMachineIs(t *testing.T) {
	cfg := config.DefaultAi()
	cfg.Ollama.BaseURL = "http://localhost:11434/"
	entry := config.ModelEntry{ID: "qwen3:0.6b",
		Source: config.ModelSourceOllama, Kind: config.ModelOllama,
		Enabled: true}
	e := NewEndpoint(entry, cfg)
	if e.BaseURL != "http://localhost:11434" {
		t.Fatalf("asked at %q, want the machine's own address",
			e.BaseURL)
	}
	if e.Model != "qwen3:0.6b" {
		t.Fatalf("asked for %q, want the machine's name", e.Model)
	}
	if e.Params.KeepAlive != cfg.Ollama.KeepAlive {
		t.Fatalf("keep alive = %q, want the settings'",
			e.Params.KeepAlive)
	}
}
