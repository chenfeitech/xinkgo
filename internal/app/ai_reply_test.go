package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/helightxu/xinkgo/internal/locale"
)

func TestAiUndo_restoresOnlyWhenUnchanged(t *testing.T) {
	s := providerApp(t)
	path := filepath.Join(s.root, "a.md")
	if err := os.WriteFile(path, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := s.aiUndo(path, "old", "new"); got != locale.AiUndoOk() {
		t.Fatalf("undo said %q", got)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "old" {
		t.Errorf("note = %q, want the before text", data)
	}
}

func TestAiUndo_refusesAChangedNote(t *testing.T) {
	s := providerApp(t)
	path := filepath.Join(s.root, "a.md")
	if err := os.WriteFile(path, []byte("edited since"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := s.aiUndo(path, "old", "new"); got !=
		locale.AiUndoConflict() {
		t.Fatalf("undo said %q", got)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "edited since" {
		t.Error("a note the user changed was overwritten")
	}
}

func TestAiReply_readOnlyDeniesAppending(t *testing.T) {
	s := providerApp(t)
	s.aiReply("daily", "an answer")
	rel, err := s.dailyRelPath(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	full, _ := s.vaultPath(rel)
	if _, err := os.Stat(full); err == nil {
		t.Fatal("read-only permission still wrote a daily note")
	}
}
