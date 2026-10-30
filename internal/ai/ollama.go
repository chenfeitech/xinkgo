package ai

import (
	"context"
	"encoding/json"
	"errors"
)

// ollama is a model spoken to in the tongue Ollama speaks itself: the whole
// request is one JSON, and the answer comes back one JSON object to a line.
type ollama struct{ e Endpoint }

// Stream asks the model and hands on the lines of its answer as they come.
func (p ollama) Stream(ctx context.Context, req Request) (<-chan Piece, error) {
	if p.e.BaseURL == "" {
		return nil, errors.New("the model has no address")
	}
	body, err := p.body(req)
	if err != nil {
		return nil, err
	}
	response, err := p.e.post(ctx, p.e.BaseURL+"/api/chat", nil, body)
	if err != nil {
		return nil, err
	}
	out := make(chan Piece)
	go func() {
		defer close(out)
		defer response.Body.Close()
		for piece := range ndjsonPieces(response.Body) {
			out <- piece
		}
	}()
	return out, nil
}

// body is the request as Ollama wants it.
func (p ollama) body(req Request) ([]byte, error) {
	params := p.e.params(req)
	options := map[string]any{}
	if params.Temperature > 0 {
		options["temperature"] = params.Temperature
	}
	if params.NumPredict > 0 {
		options["num_predict"] = params.NumPredict
	}
	if params.NumCtx != nil {
		options["num_ctx"] = *params.NumCtx
	}
	out := map[string]any{
		"model":    p.e.modelOf(req),
		"messages": messagesOf(req.Messages),
		"stream":   true,
	}
	if len(options) > 0 {
		out["options"] = options
	}
	// Thinking is asked for at the top: a model that has none of it is
	// not asked.
	if params.Think {
		out["think"] = true
	}
	if params.KeepAlive != "" {
		out["keep_alive"] = params.KeepAlive
	}
	if len(req.Tools) > 0 {
		out["tools"] = json.RawMessage(req.Tools)
	}
	return bodyOf(out)
}
