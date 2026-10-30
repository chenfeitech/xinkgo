// Package ai is the app asking a model for words.
//
// Two kinds of model are spoken to: a local Ollama, in the tongue it
// speaks itself, and any endpoint that speaks the tongue OpenAI made,
// which is what most clouds and most local servers do besides. Which of
// them a model is, is said of it in the settings and not guessed from
// its address.
//
// What is asked is one thing, whatever the model: a few messages, and
// what to make of them. What comes back is a stream of pieces, and this
// package hands them on as they arrive, unread: what they mean is for
// another to say.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/helightxu/xinkgo/internal/config"
)

// Role is who says a message.
type Role string

// The speakers of a conversation.
const (
	System    Role = "system"
	User      Role = "user"
	Assistant Role = "assistant"
	Tool      Role = "tool"
)

// Message is one thing said.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// Params is what a request asks the model to do with its words.
type Params struct {
	Temperature float32
	// NumPredict is the longest answer the model may give, in tokens;
	// 0 leaves it to the model.
	NumPredict uint32
	// NumCtx is how much of the conversation the model holds, when it is
	// not left to the model's own.
	NumCtx *uint32
	// Think asks for the thinking, where a model has any.
	Think bool
	// KeepAlive is how long Ollama keeps the model loaded.
	KeepAlive string
}

// Request is one thing asked of a model.
type Request struct {
	Model    string
	Messages []Message
	Params   Params
	// Tools are the tools the model may call, named and described as the
	// endpoint understands them: nil for none.
	Tools []byte
}

// Endpoint is where a model is asked, and as whom.
type Endpoint struct {
	// Kind is which tongue it speaks: config.ModelOllama or
	// config.ModelOpenAICompatible.
	Kind    string
	BaseURL string
	Model   string
	APIKey  string
	Params  Params
	// Timeout is how long the whole answer may take, FirstByte how long
	// before the first piece of it must have come.
	Timeout   time.Duration
	FirstByte time.Duration
}

// NewEndpoint is the endpoint of a model the settings hold.
func NewEndpoint(m config.ModelEntry, cfg config.AiConfig) Endpoint {
	e := Endpoint{
		Kind:    m.Kind,
		BaseURL: strings.TrimRight(m.BaseURL, "/"),
		Model:   m.ID,
		Params: Params{
			Temperature: m.Params.Temperature,
			NumPredict:  m.Params.NumPredict,
			NumCtx:      m.Params.NumCtx,
			Think:       m.Params.Think,
		},
		Timeout:   time.Duration(cfg.TimeoutSecs) * time.Second,
		FirstByte: time.Duration(cfg.FirstByteTimeoutSecs) * time.Second,
	}
	if m.Kind == config.ModelOllama && e.BaseURL == "" {
		e.BaseURL = strings.TrimRight(cfg.Ollama.BaseURL, "/")
	}
	if m.Params.KeepAlive != nil {
		e.Params.KeepAlive = *m.Params.KeepAlive
	} else {
		e.Params.KeepAlive = cfg.Ollama.KeepAlive
	}
	return e
}

// Piece is one piece of what the model sent: its words unread, or what
// went wrong instead of them.
type Piece struct {
	// Data is one line of NDJSON, or one `data:` of an SSE stream: what
	// it means is for another to say.
	Data []byte
	Err  error
}

// Provider is a model that can be asked.
type Provider interface {
	// Stream asks req and gives back the pieces as they come. The chan
	// is closed when the answer is finished or has failed.
	Stream(ctx context.Context, req Request) (<-chan Piece, error)
}

// NewProvider is the provider that speaks the endpoint's tongue.
func NewProvider(e Endpoint) Provider {
	if e.Kind == config.ModelOpenAICompatible {
		return openAI{e}
	}
	return ollama{e}
}

// Ask asks a model, and when it cannot answer, asks again — and then
// asks the model the user named as the one to fall back on.
//
// Only a request that failed before it gave anything back is tried
// again: an answer that broke off in the middle is not one to ask for
// twice, for the words would come twice.
func Ask(
	ctx context.Context, cfg config.AiConfig, primary Endpoint,
	req Request,
) (<-chan Piece, error) {
	var fallback *Endpoint
	if cfg.FallbackModel != "" && cfg.FallbackModel != primary.Model {
		if m := modelByID(cfg, cfg.FallbackModel); m != nil {
			e := NewEndpoint(*m, cfg)
			fallback = &e
		}
	}
	attempts := int(cfg.Retry.MaxAttempts)
	if attempts < 1 {
		attempts = 1
	}
	backoff := time.Duration(cfg.Retry.BackoffMs) * time.Millisecond

	out := make(chan Piece)
	go func() {
		defer close(out)
		models := []Endpoint{primary}
		if fallback != nil {
			models = append(models, *fallback)
		}
		for _, model := range models {
			// All but the first are the way out when the one before
			// could not answer at all.
			gave := false
			for attempt := 0; attempt < attempts; attempt++ {
				if attempt > 0 {
					if !sleep(ctx, backoff) {
						return
					}
				}
				pieces, err := NewProvider(model).Stream(ctx, req)
				if err != nil {
					continue
				}
				for p := range pieces {
					if p.Err != nil {
						// Broken off in the middle: said, and not
						// asked again.
						out <- p
						return
					}
					gave = true
					out <- p
				}
				return
			}
			if gave {
				return
			}
		}
		out <- Piece{Err: errors.New("the model could not be asked")}
	}()
	return out, nil
}

// modelByID is the model of that name in the settings.
func modelByID(cfg config.AiConfig, id string) *config.ModelEntry {
	for i := range cfg.Models {
		if cfg.Models[i].ID == id && cfg.Models[i].Enabled {
			return &cfg.Models[i]
		}
	}
	return nil
}

// sleep waits, or gives up when the caller has stopped waiting.
func sleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// post sends body to url and gives back the response, its body still
// unread. The whole answer, read and all, has Timeout to come in; a model
// that says nothing for FirstByte is given up on before that.
func (e Endpoint) post(
	ctx context.Context, url string, header http.Header, body []byte,
) (*http.Response, error) {
	timeout := e.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url,
		bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	for name, values := range header {
		for _, v := range values {
			request.Header.Add(name, v)
		}
	}
	client := &http.Client{Timeout: timeout}
	if e.FirstByte > 0 {
		// When the first piece must have come, and it has not, the
		// request is given up on rather than waited for.
		client.Transport = firstByteWithin{
			first: e.FirstByte, base: http.DefaultTransport,
		}
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 400 {
		message := readUpTo(response.Body, 512)
		response.Body.Close()
		return nil, fmt.Errorf("the model answered %d: %s",
			response.StatusCode, message)
	}
	return response, nil
}

// params is what to ask the model for: what the request says, and for what
// it leaves unsaid, what the endpoint was set up with.
func (e Endpoint) params(req Request) Params {
	p := req.Params
	if p.Temperature == 0 {
		p.Temperature = e.Params.Temperature
	}
	if p.NumPredict == 0 {
		p.NumPredict = e.Params.NumPredict
	}
	if p.NumCtx == nil {
		p.NumCtx = e.Params.NumCtx
	}
	if !p.Think {
		p.Think = e.Params.Think
	}
	if p.KeepAlive == "" {
		p.KeepAlive = e.Params.KeepAlive
	}
	return p
}

// modelOf is the model named by the request, or the endpoint's own when it
// names none.
func (e Endpoint) modelOf(req Request) string {
	if req.Model != "" {
		return req.Model
	}
	return e.Model
}

// firstByteWithin gives up on a response whose first piece is slow.
type firstByteWithin struct {
	first time.Duration
	base  http.RoundTripper
}

// RoundTrip waits for the answer's head, but only for so long.
func (t firstByteWithin) RoundTrip(r *http.Request) (*http.Response, error) {
	type result struct {
		res *http.Response
		err error
	}
	done := make(chan result, 1)
	go func() {
		base := t.base
		if base == nil {
			base = http.DefaultTransport
		}
		res, err := base.RoundTrip(r)
		done <- result{res, err}
	}()
	select {
	case out := <-done:
		return out.res, out.err
	case <-time.After(t.first):
		return nil, errors.New("the model said nothing")
	}
}

// readUpTo is the first of a body, as much as count bytes of it.
func readUpTo(r io.Reader, count int) string {
	data, _ := io.ReadAll(io.LimitReader(r, int64(count)))
	return strings.TrimSpace(string(data))
}

// messagesOf is the messages as an endpoint wants them.
func messagesOf(messages []Message) []map[string]any {
	out := make([]map[string]any, 0, len(messages))
	for _, m := range messages {
		out = append(out, map[string]any{
			"role":    string(m.Role),
			"content": m.Content,
		})
	}
	return out
}

// bodyOf is a request written out as JSON.
func bodyOf(v any) ([]byte, error) { return json.Marshal(v) }
