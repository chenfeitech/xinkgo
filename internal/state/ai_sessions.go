package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// How many conversations are kept, and how much of each: the file cannot
// grow forever, and a conversation of two hundred things said is already
// more than anyone reads back.
const (
	MaxAiSessions = 30
	MaxAiBlocks   = 200
)

// AiBlock is one thing said in a conversation, as it is kept on disk.
//
// What is kept is what was said, and not what the model was told: the
// note it was about is on the disk already, and is read again when it is
// needed.
type AiBlock struct {
	Role string `json:"role"`
	Text string `json:"text"`
	// Thinking is the model's thinking, kept apart from the answer.
	Thinking string `json:"thinking,omitempty"`
	// Cut is an answer the model broke off because it had said as much
	// as it was allowed to.
	Cut bool `json:"cut,omitempty"`
}

// AiSessions is every conversation, one to a note, kept by the note's
// path.
type AiSessions map[string][]AiBlock

// LoadAiSessions reads every conversation. A file that is missing, or that
// cannot be read, or that another version wrote, gives no conversations
// and at most something to say: a conversation is a thing that can be
// lost, and the app must start without it.
func LoadAiSessions(dir string) (AiSessions, error) {
	name := filepath.Join(dir, AiSessionsFile)
	data, err := os.ReadFile(name)
	if errors.Is(err, os.ErrNotExist) {
		return AiSessions{}, nil
	}
	if err != nil {
		return AiSessions{}, fmt.Errorf("read %s: %w", AiSessionsFile, err)
	}
	var all AiSessions
	if err := json.Unmarshal(data, &all); err != nil {
		return AiSessions{}, fmt.Errorf("parse %s: %w",
			AiSessionsFile, err)
	}
	if all == nil {
		all = AiSessions{}
	}
	return all, nil
}

// SaveAiSessions writes every conversation, trimmed to what is kept. The
// old file is replaced only once the new one is on disk: a conversation
// half written is a conversation lost.
func SaveAiSessions(dir string, all AiSessions) error {
	TrimAiSessions(all)
	return writeJSON(filepath.Join(dir, AiSessionsFile), all, true)
}

// ApplyAiSession is one note's conversation put into the table. An empty
// one is the same as none: it is how a conversation is forgotten.
//
// It is kept apart from [SaveAiSessions] so that it can be tested without
// writing to the application's own directory, where the user's
// conversations are.
func ApplyAiSession(all AiSessions, path string, blocks []AiBlock) {
	if len(blocks) == 0 {
		delete(all, path)
		return
	}
	all[path] = blocks
}

// TrimAiSessions keeps the conversations down to what is worth keeping:
// the newest two hundred things said in each, and thirty conversations in
// all.
//
// Which thirty is not remembered — a conversation has no time on it — so
// it is the same thirty every time, chosen by the note's path. That it is
// always the same is worth more than which ones they are.
func TrimAiSessions(all AiSessions) {
	for path, blocks := range all {
		if len(blocks) > MaxAiBlocks {
			all[path] = blocks[len(blocks)-MaxAiBlocks:]
		}
	}
	if len(all) <= MaxAiSessions {
		return
	}
	paths := make([]string, 0, len(all))
	for path := range all {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths[:len(paths)-MaxAiSessions] {
		delete(all, path)
	}
}
