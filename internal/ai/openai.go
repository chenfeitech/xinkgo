package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

// openAI is a model spoken to in the tongue OpenAI made, which is what most
// clouds and most local servers besides Ollama speak: the answer comes back
// as server-sent events.
type openAI struct{ e Endpoint }

// Stream asks the model and hands on the events of its answer as they come.
func (p openAI) Stream(ctx context.Context, req Request) (<-chan Piece, error) {
	if p.e.BaseURL == "" {
		return nil, errors.New("the model has no address")
	}
	body, err := p.body(req)
	if err != nil {
		return nil, err
	}
	header := http.Header{}
	if p.e.APIKey != "" {
		header.Set("Authorization", "Bearer "+p.e.APIKey)
	}
	url := p.e.BaseURL + "/chat/completions"
	response, err := p.e.post(ctx, url, header, body)
	if err != nil {
		return nil, err
	}
	out := make(chan Piece)
	go func() {
		defer close(out)
		defer response.Body.Close()
		for piece := range ssePieces(response.Body) {
			out <- piece
		}
	}()
	return out, nil
}

// body is the request as an OpenAI-shaped endpoint wants it.
func (p openAI) body(req Request) ([]byte, error) {
	params := p.e.params(req)
	out := map[string]any{
		"model":    p.e.modelOf(req),
		"messages": messagesOf(req.Messages),
		"stream":   true,
	}
	if params.Temperature > 0 {
		out["temperature"] = params.Temperature
	}
	if params.NumPredict > 0 {
		out["max_tokens"] = params.NumPredict
	}
	if len(req.Tools) > 0 {
		out["tools"] = json.RawMessage(req.Tools)
		out["tool_choice"] = "auto"
	}
	return bodyOf(out)
}
