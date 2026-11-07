package ai

import "strings"

// Replacement is the text a model put in <replacement> tags when it was
// asked to rewrite a selection. Only that text goes back into the note;
// the explanation around it stays in the conversation.
func Replacement(answer string) (string, bool) {
	const open, close = "<replacement>", "</replacement>"
	start := strings.Index(answer, open)
	if start < 0 {
		return "", false
	}
	rest := answer[start+len(open):]
	end := strings.Index(rest, close)
	if end < 0 {
		return "", false
	}
	return strings.Trim(rest[:end], "\r\n"), true
}
