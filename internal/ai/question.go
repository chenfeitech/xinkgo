package ai

import (
	"encoding/json"
	"strings"
)

// questionFromCall turns an ask_user tool call into the card the user
// sees. A tool call that cannot be read still asks something, so the
// model never leaves the user with a silent turn.
func questionFromCall(call ToolCall) Question {
	q := Question{ID: call.Name, Text: "What should I do next?"}
	var args struct {
		Question   string   `json:"question"`
		Header     string   `json:"header"`
		Options    []string `json:"options"`
		AllowOther bool     `json:"allow_other"`
		Secret     bool     `json:"secret"`
	}
	if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
		return q
	}
	if text := strings.TrimSpace(args.Question); text != "" {
		q.Text = text
	}
	q.Header = strings.TrimSpace(args.Header)
	q.AllowOther = args.AllowOther
	q.Secret = args.Secret
	for _, one := range args.Options {
		if label := strings.TrimSpace(one); label != "" {
			q.Options = append(q.Options, label)
		}
	}
	return q
}
