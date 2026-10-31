package ai

import (
	"context"
	"time"

	"github.com/helightxu/xinkgo/internal/config"
)

// probeTool is the tool a model is asked to call when it is being tried
// out. It does nothing: what is being found out is only whether the model
// asks for it at all.
var probeTool = []byte(`[{"type":"function","function":{` +
	`"name":"noop","description":"连接测试用：不需要调用它",` +
	`"parameters":{"type":"object","properties":{}}}}]`)

// Probe is what trying a model out found.
type Probe struct {
	// OK is whether the model answered at all.
	OK bool
	// Err is what went wrong, when something did.
	Err error
	// Tools is whether the model really asked for a tool. What a model
	// says of itself is not to be trusted — one declares tools and then
	// answers 500 when they are sent — so it is asked to call one.
	Tools bool
	// Took is how long the model took to answer.
	Took time.Duration
}

// Try tries a model out: one small question, and, with tools, one tool
// it may call. It is how the app finds out what a model can do, rather
// than believing what it says of itself.
func Try(
	ctx context.Context, cfg config.AiConfig, m config.ModelEntry,
	withTools bool,
) Probe {
	started := time.Now()
	endpoint := NewEndpoint(m, cfg)
	req := Request{
		Model: m.ID,
		Messages: []Message{{
			Role:    User,
			Content: "ping",
		}},
		// Asking for eight words is enough to know the model is there,
		// and no more: a probe is not a conversation.
		Params: Params{NumPredict: 8, Think: false},
	}
	if withTools {
		req.Tools = probeTool
	}
	pieces, err := Ask(ctx, cfg, endpoint, req)
	if err != nil {
		return Probe{Err: err, Took: time.Since(started)}
	}
	out := Probe{OK: true, Took: time.Since(started)}
	for event := range Events(endpoint, pieces) {
		switch event.Kind {
		case EventToolCall:
			if event.Call.Name == "noop" {
				out.Tools = true
			}
		case EventFailed:
			// A model that answered and then broke off is not one
			// that answered.
			out.OK = false
			out.Err = event.Err
		}
	}
	out.Took = time.Since(started)
	return out
}
