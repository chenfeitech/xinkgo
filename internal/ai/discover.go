package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/helightxu/xinkgo/internal/config"
)

// How long a listing is waited for, how long before its first piece must
// have come, how many models are read out of one, and how much of one is
// read: a machine holds a few models, and a list of thousands is not one
// a user can choose from.
const (
	listTimeout      = 10 * time.Second
	listFirstByte    = 5 * time.Second
	maxListedModels  = 500
	maxListBodyBytes = 4 << 20
)

// defaultOllamaURL is where a local Ollama is when the user has named
// nowhere: the same place the settings start out naming.
const defaultOllamaURL = "http://localhost:11434"

// OllamaModel is one model a local Ollama holds on the machine: what it
// is called, how much room it takes, what it is made of, and what has
// been found that it can do.
type OllamaModel struct {
	// Name is what Ollama calls it, and what the app asks by.
	Name string
	// Size is how many bytes it takes on the disk.
	Size uint64
	// Family is what the model is built of, and Families every piece of
	// it: a model with a picture reader in it is named twice.
	Family        string
	Families      []string
	ParameterSize string
	Quantization  string
	// ContextLimit is how much of a conversation it holds, when the
	// server said; 0 when it did not.
	ContextLimit uint32
	// Capabilities are what it can do, as the app names them.
	Capabilities []string
	// Guessed is whether those were guessed at rather than read off the
	// server: an older Ollama does not say what a model can do.
	Guessed bool
	// Modified is when it was last changed on the machine.
	Modified time.Time
}

// ollamaTags is the answer to asking a local Ollama what it holds.
type ollamaTags struct {
	Models []struct {
		Name       string    `json:"name"`
		Model      string    `json:"model"`
		Size       uint64    `json:"size"`
		ModifiedAt time.Time `json:"modified_at"`
		Details    struct {
			Family        string   `json:"family"`
			Families      []string `json:"families"`
			ParameterSize string   `json:"parameter_size"`
			Quantization  string   `json:"quantization_level"`
		} `json:"details"`
	} `json:"models"`
}

// ollamaShow is the answer to asking a local Ollama about one model.
type ollamaShow struct {
	Details struct {
		Family        string   `json:"family"`
		Families      []string `json:"families"`
		ParameterSize string   `json:"parameter_size"`
		Quantization  string   `json:"quantization_level"`
	} `json:"details"`
	// ModelInfo is what the model is, key by key: how much of a
	// conversation it holds is one of them.
	ModelInfo    map[string]json.RawMessage `json:"model_info"`
	Capabilities []string                   `json:"capabilities"`
}

// OllamaModels asks a local Ollama which models it holds, and of each what
// it can do: the names come from one asking, and what each can do from one
// more apiece.
//
// A model whose capabilities cannot be read is listed all the same: it is
// on the machine, and a guess is better than a model the user is not told
// about. What is not guessed at, for any of them, is whether it calls
// tools: that is tried, and not guessed.
func OllamaModels(ctx context.Context, baseURL string) ([]OllamaModel, error) {
	e := listingEndpoint(config.ModelOllama, baseURL)
	body, err := e.get(ctx, e.BaseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	var listed ollamaTags
	if err := json.Unmarshal(body, &listed); err != nil {
		return nil, Protocol("the list of models could not be read")
	}
	out := make([]OllamaModel, 0, len(listed.Models))
	for _, one := range listed.Models {
		name := strings.TrimSpace(one.Name)
		if name == "" {
			name = strings.TrimSpace(one.Model)
		}
		if name == "" {
			continue
		}
		m := OllamaModel{
			Name:          name,
			Size:          one.Size,
			Family:        one.Details.Family,
			Families:      append([]string(nil), one.Details.Families...),
			ParameterSize: one.Details.ParameterSize,
			Quantization:  one.Details.Quantization,
			Modified:      one.ModifiedAt,
		}
		shown, err := ollamaShowOf(ctx, e, name)
		if err == nil {
			m.fill(shown)
		} else {
			// An older server does not say what a model can do.
			m.Capabilities = guessOllama(m)
			m.Guessed = true
		}
		out = append(out, m)
		if len(out) >= maxListedModels {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// ollamaShowOf asks a local Ollama about one model: what it is made of,
// and what it can do.
func ollamaShowOf(
	ctx context.Context, e Endpoint, name string,
) (ollamaShow, error) {
	body, err := bodyOf(map[string]any{"model": name})
	if err != nil {
		return ollamaShow{}, err
	}
	response, err := e.post(ctx, e.BaseURL+"/api/show", nil, body)
	if err != nil {
		return ollamaShow{}, err
	}
	defer response.Body.Close()
	read, err := io.ReadAll(io.LimitReader(response.Body, maxListBodyBytes))
	if err != nil {
		return ollamaShow{}, err
	}
	var out ollamaShow
	if err := json.Unmarshal(read, &out); err != nil {
		return ollamaShow{}, Protocol("the model could not be read")
	}
	return out, nil
}

// fill takes what the server said of a model into what is known of it.
func (m *OllamaModel) fill(s ollamaShow) {
	if s.Details.Family != "" {
		m.Family = s.Details.Family
	}
	if len(s.Details.Families) > 0 {
		m.Families = append([]string(nil), s.Details.Families...)
	}
	if s.Details.ParameterSize != "" {
		m.ParameterSize = s.Details.ParameterSize
	}
	if s.Details.Quantization != "" {
		m.Quantization = s.Details.Quantization
	}
	if limit := contextLengthOf(s.ModelInfo); limit > 0 {
		m.ContextLimit = limit
	}
	if len(s.Capabilities) > 0 {
		out := make([]string, 0, len(s.Capabilities))
		for _, one := range s.Capabilities {
			out = append(out, CapabilityOfOllama(one))
		}
		m.Capabilities = out
		return
	}
	m.Capabilities = guessOllama(*m)
	m.Guessed = true
}

// guessOllama is a guess at what a model can do, made of its name and of
// what it is built of: one with a picture reader in it sees, and one whose
// name says it reasons, reasons. Every one of them answers.
func guessOllama(m OllamaModel) []string {
	out := []string{CapCompletion}
	parts := append([]string{m.Name, m.Family}, m.Families...)
	joined := strings.ToLower(strings.Join(parts, " "))
	if containsAny(joined, "clip", "vision", "-vl") {
		out = append(out, CapVision)
	}
	if containsAny(joined, "thinking", "-r1", "reasoner", "reasoning") {
		out = append(out, CapThinking)
	}
	return out
}

// contextLengthOf is how much of a conversation a model holds, read out of
// what the server said of it; 0 when it said nothing that can be read.
//
// The key is named after what the model is built of, `qwen3.context_length`,
// so it is not looked for by its whole name but by how the name ends.
func contextLengthOf(info map[string]json.RawMessage) uint32 {
	for key, raw := range info {
		if !strings.HasSuffix(key, "context_length") {
			continue
		}
		var num json.Number
		if err := json.Unmarshal(raw, &num); err == nil {
			if v, err := num.Float64(); err == nil && v > 0 {
				return uint32(v)
			}
			continue
		}
		var text string
		if err := json.Unmarshal(raw, &text); err == nil {
			if v, err := strconv.ParseFloat(text, 64); err == nil && v > 0 {
				return uint32(v)
			}
		}
	}
	return 0
}

// OpenAIModels asks an endpoint that speaks the tongue OpenAI made which
// models it holds. Most clouds say, and so do most local servers besides
// Ollama; one that does not is told about, and what it holds may still be
// written in by hand.
func OpenAIModels(
	ctx context.Context, m config.ModelEntry, cfg config.AiConfig,
) ([]string, error) {
	e := NewEndpoint(m, cfg)
	if e.Timeout <= 0 {
		e.Timeout = listTimeout
	}
	if e.FirstByte <= 0 {
		e.FirstByte = listFirstByte
	}
	body, err := e.get(ctx, e.BaseURL+"/models", e.authHeader())
	if err != nil {
		return nil, err
	}
	var listed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &listed); err != nil {
		return nil, Protocol("the list of models could not be read")
	}
	names := make([]string, 0, len(listed.Data))
	for _, one := range listed.Data {
		name := strings.TrimSpace(one.ID)
		if name == "" {
			continue
		}
		names = append(names, name)
		if len(names) >= maxListedModels {
			break
		}
	}
	sort.Strings(names)
	return names, nil
}

// listingEndpoint is the endpoint of a server being asked what it holds:
// it names no model of its own, and it is given a while to answer, for a
// machine being started up is slow the first time it is asked.
func listingEndpoint(kind, baseURL string) Endpoint {
	url := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if url == "" {
		url = defaultOllamaURL
	}
	return Endpoint{
		Kind:      kind,
		BaseURL:   url,
		Timeout:   listTimeout,
		FirstByte: listFirstByte,
	}
}

// get asks url for what it holds, and gives back the body of the answer,
// read whole. An answer that is not a good one is said, with as much of it
// as a user can act on.
func (e Endpoint) get(
	ctx context.Context, url string, header http.Header,
) ([]byte, error) {
	timeout := e.Timeout
	if timeout <= 0 {
		timeout = listTimeout
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for name, values := range header {
		for _, v := range values {
			request.Header.Add(name, v)
		}
	}
	client := &http.Client{Timeout: timeout}
	if e.FirstByte > 0 {
		client.Transport = firstByteWithin{
			first: e.FirstByte, base: http.DefaultTransport,
		}
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return nil, Status{
			Code: response.StatusCode, Body: readUpTo(response.Body, 512),
		}
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxListBodyBytes))
	if err != nil {
		return nil, err
	}
	return data, nil
}
