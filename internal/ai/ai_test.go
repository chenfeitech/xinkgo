package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/helightxu/xinkgo/internal/config"
)

// gather is every piece an answer gave, waited for but not forever.
func gather(t *testing.T, pieces <-chan Piece) []Piece {
	t.Helper()
	out := []Piece{}
	for {
		select {
		case p, open := <-pieces:
			if !open {
				return out
			}
			out = append(out, p)
		case <-time.After(5 * time.Second):
			t.Fatal("the answer never finished")
			return out
		}
	}
}

// dataOf is the pieces that carried words, as text.
func dataOf(pieces []Piece) []string {
	out := make([]string, 0, len(pieces))
	for _, p := range pieces {
		if p.Err == nil {
			out = append(out, string(p.Data))
		}
	}
	return out
}

// errOf is what went wrong in an answer, if anything did.
func errOf(pieces []Piece) error {
	for _, p := range pieces {
		if p.Err != nil {
			return p.Err
		}
	}
	return nil
}

// endpoint is an endpoint that asks the server at url.
func endpoint(kind, url string) Endpoint {
	return Endpoint{
		Kind: kind, BaseURL: url, Model: "qwen",
		Timeout: 5 * time.Second,
	}
}

func TestNewEndpoint_whatTheSettingsDoNotSayIsFilledIn(t *testing.T) {
	cfg := config.DefaultAi()
	cfg.Ollama.BaseURL = "http://localhost:11434/"
	cfg.TimeoutSecs = 42
	cfg.FirstByteTimeoutSecs = 7
	e := NewEndpoint(config.ModelEntry{
		ID: "qwen", Kind: config.ModelOllama, Enabled: true,
	}, cfg)
	// The trailing slash is of no use to a URL that is added to.
	if e.BaseURL != "http://localhost:11434" {
		t.Fatalf("base url = %q", e.BaseURL)
	}
	if e.Model != "qwen" {
		t.Fatalf("model = %q", e.Model)
	}
	if e.Timeout != 42*time.Second || e.FirstByte != 7*time.Second {
		t.Fatalf("timeouts = %v %v", e.Timeout, e.FirstByte)
	}
	// What the model does not say of itself is the server's own.
	if e.Params.KeepAlive != cfg.Ollama.KeepAlive {
		t.Fatalf("keep alive = %q", e.Params.KeepAlive)
	}
}

func TestNewEndpoint_theModelsOwnIsKept(t *testing.T) {
	cfg := config.DefaultAi()
	alive := "10m"
	ctx := uint32(4096)
	e := NewEndpoint(config.ModelEntry{
		ID: "gpt", Kind: config.ModelOpenAICompatible, Enabled: true,
		BaseURL: "https://api.example.com/v1",
		Params: config.AiModelParams{
			KeepAlive: &alive, NumCtx: &ctx, Think: true,
		},
	}, cfg)
	if e.Params.KeepAlive != "10m" {
		t.Fatalf("keep alive = %q", e.Params.KeepAlive)
	}
	if e.Params.NumCtx == nil || *e.Params.NumCtx != 4096 {
		t.Fatalf("num ctx = %v", e.Params.NumCtx)
	}
	if !e.Params.Think {
		t.Fatal("think was asked for and is not")
	}
}

func TestOllamaStream_eachLineComesOnItsOwn(t *testing.T) {
	ctx := uint32(8192)
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/chat" {
				t.Errorf("path = %q, want /api/chat", r.URL.Path)
			}
			_ = json.NewDecoder(r.Body).Decode(&got)
			w.Header().Set("Content-Type", "application/x-ndjson")
			_, _ = io.WriteString(w, `{"message":{"content":"hel"}}`+"\n")
			_, _ = io.WriteString(w, "\n")
			_, _ = io.WriteString(w, `{"done":true}`+"\n")
		}))
	defer server.Close()

	e := endpoint(config.ModelOllama, server.URL)
	e.Params.NumCtx = &ctx
	pieces := gather(t, asked(t, e))
	// The empty line is passed over: two pieces, not three.
	if got := dataOf(pieces); len(got) != 2 {
		t.Fatalf("pieces = %v", got)
	}
	if errOf(pieces) != nil {
		t.Fatalf("err = %v", errOf(pieces))
	}
	if got["stream"] != true {
		t.Fatalf("stream = %v, want true", got["stream"])
	}
	if got["model"] != "qwen" {
		t.Fatalf("model = %v", got["model"])
	}
	options, ok := got["options"].(map[string]any)
	if !ok {
		t.Fatalf("options = %v", got["options"])
	}
	if options["num_ctx"] == nil {
		t.Fatal("num ctx was not asked for")
	}
}

func TestOllamaStream_whatTheRequestSaysIsAsked(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&got)
			_, _ = io.WriteString(w, "{}\n")
		}))
	defer server.Close()

	e := endpoint(config.ModelOllama, server.URL)
	e.Params.KeepAlive = "5m"
	alive := "1m"
	_ = gather(t, asked(t, e, Request{
		Messages: []Message{{Role: User, Content: "hi"}},
		Params:   Params{Think: true, KeepAlive: alive, NumPredict: 64},
	}))
	if got["think"] != true {
		t.Fatalf("think = %v", got["think"])
	}
	if got["keep_alive"] != "1m" {
		t.Fatalf("keep alive = %v, want the request's 1m", got["keep_alive"])
	}
	options := got["options"].(map[string]any)
	if options["num_predict"] != float64(64) {
		t.Fatalf("num predict = %v", options["num_predict"])
	}
	// num_ctx is not said by the request and is the endpoint's own:
	// here, none at all.
	if _, there := options["num_ctx"]; there {
		t.Fatalf("num ctx = %v, want it left out", options["num_ctx"])
	}
}

func TestOpenAIStream_onlyTheDataComesOn(t *testing.T) {
	var auth string
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/chat/completions" {
				t.Errorf("path = %q, want /chat/completions",
					r.URL.Path)
			}
			auth = r.Header.Get("Authorization")
			_, _ = io.WriteString(w, ": a comment, to keep the line open\n")
			_, _ = io.WriteString(w, "event: delta\n")
			_, _ = io.WriteString(w, `data: {"choices":[0]}`+"\n")
			_, _ = io.WriteString(w, "data: and the rest of it\n\n")
			_, _ = io.WriteString(w, "data: [DONE]\n\n")
		}))
	defer server.Close()

	e := endpoint(config.ModelOpenAICompatible, server.URL)
	e.APIKey = "sk-test"
	pieces := gather(t, asked(t, e))
	got := dataOf(pieces)
	want := []string{
		`{"choices":[0]}` + "\nand the rest of it",
		"[DONE]",
	}
	if len(got) != len(want) {
		t.Fatalf("pieces = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("piece %d = %q, want %q", i, got[i], want[i])
		}
	}
	if auth != "Bearer sk-test" {
		t.Fatalf("authorization = %q", auth)
	}
}

func TestStream_aModelThatHasNoAddressIsNotAsked(t *testing.T) {
	for _, kind := range []string{
		config.ModelOllama, config.ModelOpenAICompatible,
	} {
		if _, err := NewProvider(Endpoint{Kind: kind}).
			Stream(context.Background(), Request{}); err == nil {
			t.Fatalf("%s: a model with no address was asked", kind)
		}
	}
}

func TestAsk_aRequestThatNeverStartedIsAskedAgain(t *testing.T) {
	var tried int32
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if atomic.AddInt32(&tried, 1) == 1 {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = io.WriteString(w, "the model is busy")
				return
			}
			_, _ = io.WriteString(w, `{"done":true}`+"\n")
		}))
	defer server.Close()

	cfg := config.DefaultAi()
	cfg.Retry = config.AiRetryConfig{MaxAttempts: 3, BackoffMs: 1}
	e := endpoint(config.ModelOllama, server.URL)
	pieces, err := Ask(context.Background(), cfg, e, Request{})
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if got := dataOf(gather(t, pieces)); len(got) != 1 {
		t.Fatalf("pieces = %v", got)
	}
	if tried != 2 {
		t.Fatalf("tried %d times, want 2", tried)
	}
}

func TestAsk_theFallbackIsAskedWhenTheFirstCannotAnswer(t *testing.T) {
	busy := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
	defer busy.Close()
	well := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, `{"done":true}`+"\n")
		}))
	defer well.Close()

	cfg := config.DefaultAi()
	cfg.Retry = config.AiRetryConfig{MaxAttempts: 2, BackoffMs: 1}
	cfg.Models = []config.ModelEntry{
		{ID: "first", Kind: config.ModelOllama, Enabled: true,
			BaseURL: busy.URL},
		{ID: "second", Kind: config.ModelOllama, Enabled: true,
			BaseURL: well.URL},
	}
	cfg.FallbackModel = "second"
	e := NewEndpoint(cfg.Models[0], cfg)
	pieces, err := Ask(context.Background(), cfg, e, Request{})
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if got := dataOf(gather(t, pieces)); len(got) != 1 {
		t.Fatalf("pieces = %v", got)
	}
}

func TestAsk_anAnswerBrokenOffIsNotAskedTwice(t *testing.T) {
	var tried int32
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&tried, 1)
			_, _ = io.WriteString(w, `{"message":{"content":"hel"}}`+"\n")
			w.(http.Flusher).Flush()
			// The answer breaks off: the connection goes away with
			// the rest of it unwritten.
			panic("broken off")
		}))
	defer server.Close()

	cfg := config.DefaultAi()
	cfg.Retry = config.AiRetryConfig{MaxAttempts: 3, BackoffMs: 1}
	e := endpoint(config.ModelOllama, server.URL)
	pieces, err := Ask(context.Background(), cfg, e, Request{})
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	got := gather(t, pieces)
	if len(dataOf(got)) < 1 {
		t.Fatalf("pieces = %v, want the one that came", got)
	}
	// What came is said, and the asking is over: a second try would
	// bring the same words twice.
	if tried != 1 {
		t.Fatalf("tried %d times, want 1", tried)
	}
}

func TestAsk_aModelThatCannotBeAskedAtAllSaysSo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
	defer server.Close()

	cfg := config.DefaultAi()
	cfg.Retry = config.AiRetryConfig{MaxAttempts: 2, BackoffMs: 1}
	e := endpoint(config.ModelOllama, server.URL)
	pieces, err := Ask(context.Background(), cfg, e, Request{})
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if errOf(gather(t, pieces)) == nil {
		t.Fatal("nothing was said of a model that could not be asked")
	}
}

func TestStream_theStatusIsSaid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, "no such model")
		}))
	defer server.Close()

	_, err := NewProvider(endpoint(config.ModelOllama, server.URL)).
		Stream(context.Background(), Request{})
	if err == nil {
		t.Fatal("a model that answered 404 was not said to have failed")
	}
	if !strings.Contains(err.Error(), "404") ||
		!strings.Contains(err.Error(), "no such model") {
		t.Fatalf("err = %v, want the status and what was said", err)
	}
}

func TestStream_aModelThatSaysNothingIsGivenUpOn(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(400 * time.Millisecond)
			_, _ = io.WriteString(w, "{}\n")
		}))
	defer server.Close()

	e := endpoint(config.ModelOllama, server.URL)
	e.FirstByte = 60 * time.Millisecond
	start := time.Now()
	_, err := NewProvider(e).Stream(context.Background(), Request{})
	if err == nil {
		t.Fatal("a model that said nothing was waited for")
	}
	if waited := time.Since(start); waited > 300*time.Millisecond {
		t.Fatalf("waited %v, want the giving up at 60ms", waited)
	}
}

func TestNdjsonPieces_aReaderThatFailsSaysWhatWentWrong(t *testing.T) {
	want := errors.New("the line went dead")
	pieces := gather(t, ndjsonPieces(io.MultiReader(
		strings.NewReader(`{"one":1}`+"\n"),
		&failingReader{err: want},
	)))
	if got := dataOf(pieces); len(got) != 1 || got[0] != `{"one":1}` {
		t.Fatalf("pieces = %v", got)
	}
	if !errors.Is(errOf(pieces), want) {
		t.Fatalf("err = %v, want %v", errOf(pieces), want)
	}
}

func TestSsePieces_anAnswerWithNoBlankLineAtTheEndIsStillAnAnswer(t *testing.T) {
	pieces := gather(t, ssePieces(strings.NewReader(
		"data: one\r\ndata: two\r\n")))
	if got := dataOf(pieces); len(got) != 1 || got[0] != "one\ntwo" {
		t.Fatalf("pieces = %v", got)
	}
}

func TestSsePieces_aFieldThatIsNotDataIsPassedOver(t *testing.T) {
	pieces := gather(t, ssePieces(strings.NewReader(
		"id: 1\nevent: delta\nretry: 100\ndata: {}\n\n")))
	if got := dataOf(pieces); len(got) != 1 || got[0] != "{}" {
		t.Fatalf("pieces = %v", got)
	}
}

// asked asks the endpoint and gives back what it said, failing the test
// when it could not be asked at all.
func asked(t *testing.T, e Endpoint, req ...Request) <-chan Piece {
	t.Helper()
	if len(req) == 0 {
		req = append(req, Request{
			Messages: []Message{{Role: User, Content: "hi"}},
		})
	}
	pieces, err := NewProvider(e).Stream(context.Background(), req[0])
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	return pieces
}

// failingReader is a reader the bottom falls out of.
type failingReader struct{ err error }

// Read gives back what went wrong.
func (r *failingReader) Read([]byte) (int, error) { return 0, r.err }
