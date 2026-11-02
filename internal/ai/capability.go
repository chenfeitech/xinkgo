package ai

import (
	"strings"

	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
)

// The capabilities a model may have, as the app knows them.
const (
	CapCompletion = "completion"
	CapTools      = "tools"
	CapThinking   = "thinking"
	CapVision     = "vision"
)

// Known are the capabilities the app has a name for, in the order they
// are shown. A model may name one the app has never heard of: it is shown
// as it stands, and is not thrown away.
var Known = []string{CapCompletion, CapTools, CapThinking, CapVision}

// Capability is what one of them means.
type Capability struct {
	// Label is the name shown in the settings.
	Label string
	// Meaning is what the capability is.
	Meaning string
	// Usage is what having it costs, or what it is for.
	Usage string
}

// Info is what the app can say of a capability. One it has never heard of
// — and the vendors do make up new ones — is said to be unknown, and is
// kept rather than dropped.
func Info(id string) Capability {
	switch id {
	case CapCompletion:
		return Capability{
			Label:   locale.AiCapabilityCompletion(),
			Meaning: locale.AiCapabilityCompletionMeaning(),
			Usage:   locale.AiCapabilityCompletionUsage(),
		}
	case CapTools:
		return Capability{
			Label:   locale.AiCapabilityTools(),
			Meaning: locale.AiCapabilityToolsMeaning(),
			Usage:   locale.AiCapabilityToolsUsage(),
		}
	case CapThinking:
		return Capability{
			Label:   locale.AiCapabilityThinking(),
			Meaning: locale.AiCapabilityThinkingMeaning(),
			Usage:   locale.AiCapabilityThinkingUsage(),
		}
	case CapVision:
		return Capability{
			Label:   locale.AiCapabilityVision(),
			Meaning: locale.AiCapabilityVisionMeaning(),
			Usage:   locale.AiCapabilityVisionUsage(),
		}
	}
	return Capability{
		Label:   locale.AiCapabilityUnknown(),
		Meaning: locale.AiCapabilityUnknownMeaning(),
		Usage:   locale.AiCapabilityUnknownUsage(),
	}
}

// Infer is a guess at what a model can do, made of its name and of where
// it is asked. It is a guess and not a promise: what a model says of
// itself is not to be trusted either, and the only word worth taking is
// the one [Probe] brings back.
//
// A local Ollama is not guessed at: it is asked what it can do, and that
// is a better answer than any guess.
func Infer(kind, baseURL, modelID string) []string {
	if kind == config.ModelOllama {
		return nil
	}
	model := strings.ToLower(modelID)
	host := hostOf(baseURL)
	out := []string{}

	// What a vendor writes in the name of a model is the most to be
	// trusted of what there is to go on.
	if containsAny(model, "reasoner", "thinking", "-r1") {
		out = append(out, CapCompletion, CapThinking)
	}
	if containsAny(model, "vision", "-vl") &&
		!hasCapability(out, CapVision) {
		out = append(out, CapVision)
	}
	if len(out) > 0 {
		return out
	}
	// Failing that, a well-known address or a well-known name is
	// enough to guess that it answers and calls tools.
	if isKnownHost(host) ||
		containsAny(model, "deepseek", "glm", "moonshot", "qwen",
			"gpt") {
		out = append(out, CapCompletion, CapTools)
	}
	return out
}

// Effective is the capabilities of a model as they are shown: what the
// user chose by hand, and failing that what is already known of it, and
// failing that the guess.
//
// The guess is never written down: written down it would look like a
// thing the user had chosen, and would stay there when the vendor
// changes.
func Effective(m config.ModelEntry) []string {
	if m.CapabilitiesPinned || len(m.Capabilities) > 0 {
		return m.Capabilities
	}
	return Infer(m.Kind, m.BaseURL, m.ID)
}

// hostOf is the host a base URL names, lower case, or "" when it names
// none that can be read.
func hostOf(baseURL string) string {
	rest := baseURL
	if at := strings.Index(rest, "://"); at >= 0 {
		rest = rest[at+3:]
	}
	if at := strings.IndexAny(rest, "/:?"); at >= 0 {
		rest = rest[:at]
	}
	return strings.ToLower(strings.TrimSpace(rest))
}

// isKnownHost is whether the app knows the address. Only the well-known
// ones are named: a guess about an address the app does not know is
// worse than no guess at all.
func isKnownHost(host string) bool {
	if host == "" {
		return false
	}
	for _, known := range []string{
		"api.deepseek.com",
		"open.bigmodel.cn",
		"api.moonshot.cn",
		"dashscope.aliyuncs.com",
		"api.openai.com",
	} {
		if host == known || strings.HasSuffix(host, known) {
			return true
		}
	}
	return false
}

// containsAny is whether s holds any of these.
func containsAny(s string, of ...string) bool {
	for _, one := range of {
		if strings.Contains(s, one) {
			return true
		}
	}
	return false
}

// hasCapability is whether the list holds it.
func hasCapability(list []string, want string) bool {
	for _, one := range list {
		if one == want {
			return true
		}
	}
	return false
}

// CapabilityOfOllama is what the app calls the thing Ollama calls name.
// One it has never heard of is kept as it stands: the vendors do make
// them up, and a name the user can see is better than none at all.
func CapabilityOfOllama(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "completion":
		return CapCompletion
	case "tools", "tool":
		return CapTools
	case "thinking", "think":
		return CapThinking
	case "vision":
		return CapVision
	}
	return strings.TrimSpace(name)
}

// Measured is what trying a model out has really shown of what it can do:
// that it answers, and that it calls tools when it was asked to call one.
//
// What a try cannot find out — whether it thinks first, whether it sees —
// is left as it was known before, and a model that could not be reached
// is left alone altogether: it has been shown nothing.
func Measured(p Probe, was []string) []string {
	if !p.OK {
		return was
	}
	out := []string{CapCompletion}
	if p.Tools {
		out = append(out, CapTools)
	}
	for _, one := range was {
		if hasCapability(out, one) {
			continue
		}
		switch one {
		case CapThinking, CapVision:
			out = append(out, one)
		}
	}
	return out
}
