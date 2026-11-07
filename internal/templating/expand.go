package templating

import (
	"fmt"
	"strings"
	"time"
)

// Vars is what a template may ask of the app when it is expanded.
type Vars struct {
	Title      string
	Now        time.Time
	DateFormat string
	TimeFormat string
	// Extras are variables a caller supplies itself, such as the tags
	// of the vault or the text of a selection. They are looked up before
	// the built-in ones, so a caller may add a name without changing
	// this package's list.
	Extras map[string]string
}

// Expand replaces the variables Xinkgo knows in text. An unknown variable
// is left exactly as it was: a template written for Templater must not be
// mangled by a host that does not implement it. A bad inline format is an
// error, so half-expanded text is never written to a note.
func Expand(text string, v Vars) (string, error) {
	if v.Now.IsZero() {
		v.Now = time.Now()
	}
	if v.DateFormat == "" {
		v.DateFormat = DefaultDateFormat
	}
	if v.TimeFormat == "" {
		v.TimeFormat = DefaultTimeFormat
	}
	var out strings.Builder
	for {
		start := strings.Index(text, "{{")
		if start < 0 {
			out.WriteString(text)
			return out.String(), nil
		}
		out.WriteString(text[:start])
		rest := text[start+2:]
		end := strings.Index(rest, "}}")
		if end < 0 {
			// No closing brace: keep the rest verbatim.
			out.WriteString(text[start:])
			return out.String(), nil
		}
		name := rest[:end]
		replacement, known, err := expandOne(name, v)
		if err != nil {
			return "", err
		}
		if !known {
			// Unknown variable: put its own text back.
			out.WriteString(text[start : start+2+end+2])
		} else {
			out.WriteString(replacement)
		}
		text = rest[end+2:]
	}
}

func expandOne(name string, v Vars) (string, bool, error) {
	if extra, ok := v.Extras[name]; ok {
		return extra, true, nil
	}
	switch {
	case name == "title":
		return v.Title, true, nil
	case name == "date":
		out, err := Format(v.Now, v.DateFormat)
		return out, true, err
	case strings.HasPrefix(name, "date:"):
		out, err := Format(v.Now, strings.TrimPrefix(name, "date:"))
		return out, true, err
	case name == "time":
		out, err := Format(v.Now, v.TimeFormat)
		return out, true, err
	case strings.HasPrefix(name, "time:"):
		out, err := Format(v.Now, strings.TrimPrefix(name, "time:"))
		return out, true, err
	default:
		return "", false, nil
	}
}

// ExpandFile expands the text of a template read from the vault. It is a
// separate entry point so callers that read a file can report where a bad
// format came from.
func ExpandFile(path, text string, v Vars) (string, error) {
	out, err := Expand(text, v)
	if err != nil {
		return "", fmt.Errorf("expand %s: %w", path, err)
	}
	return out, nil
}
