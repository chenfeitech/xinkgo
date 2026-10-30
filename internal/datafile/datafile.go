// Package datafile is what the app knows of a file that is not a note
// but is made of data: JSON, YAML and TOML.
//
// It says which of them a file is, and of a JSON file it can say two
// more things: whether it is well formed, and what it would look like
// written out evenly. YAML and TOML are named and left as they are, for
// reading them truly takes a library of their own, and this app has
// none: a file that cannot be read is not a file that was misread.
package datafile

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
)

// Kind is which of the three a file is made of.
type Kind int

const (
	// None is a file that is made of none of them.
	None Kind = iota
	JSON
	YAML
	TOML
)

// Name is what the kind is called, as the window says it.
func (k Kind) Name() string {
	switch k {
	case JSON:
		return "JSON"
	case YAML:
		return "YAML"
	case TOML:
		return "TOML"
	default:
		return ""
	}
}

// KindOf is which of the three the file at path is made of, from the
// name it was given.
func KindOf(path string) Kind {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return JSON
	case ".yaml", ".yml":
		return YAML
	case ".toml":
		return TOML
	default:
		return None
	}
}

// Problem is something wrong in the file, and where it stands.
type Problem struct {
	// Line and Col are 0-based, as the editor counts them.
	Line int
	Col  int
	// Message is what is wrong, as the window says it.
	Message string
}

// Check reads a JSON file and says what is wrong with it, and where. A
// file that is well formed has nothing said of it. Of the other kinds,
// which the app does not read, nothing is said at all.
func Check(kind Kind, text string) (Problem, bool) {
	if kind != JSON {
		return Problem{}, false
	}
	var value any
	err := json.Unmarshal([]byte(text), &value)
	if err == nil {
		return Problem{}, false
	}
	if syntax, ok := err.(*json.SyntaxError); ok {
		line, col := placeOf(text, syntax.Offset)
		return Problem{Line: line, Col: col,
			Message: syntax.Error()}, true
	}
	// Something else is wrong with it, but not where: the whole file is
	// what is wrong.
	return Problem{Message: err.Error()}, true
}

// Format writes a JSON file out evenly, with indent spaces to a level. It
// says whether it was written: a file the app does not read, or one that
// is not well formed, is left as it was.
func Format(kind Kind, text string, indent int) (string, bool) {
	if kind != JSON {
		return text, false
	}
	if _, bad := Check(kind, text); bad {
		return text, false
	}
	if indent < 1 {
		indent = 1
	}
	var out bytes.Buffer
	spaces := strings.Repeat(" ", indent)
	if err := json.Indent(&out, []byte(text), "", spaces); err != nil {
		return text, false
	}
	return out.String(), true
}

// TooLarge is whether a file is too large to be written out evenly: a
// file of megabytes is read as it stands.
func TooLarge(size, max uint64) bool {
	if max == 0 {
		return false
	}
	return size > max
}

// placeOf is the line and the column a place in the text stands at.
func placeOf(text string, offset int64) (int, int) {
	if offset < 0 {
		return 0, 0
	}
	line, col := 0, 0
	for i := int64(0); i < offset && i < int64(len(text)); i++ {
		if text[i] == '\n' {
			line++
			col = 0
			continue
		}
		col++
	}
	return line, col
}
