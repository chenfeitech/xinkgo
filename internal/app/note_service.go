package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/helightxu/xinkgo/internal/templating"
)

// errOutsideVault is what a path that leaves the vault becomes. The vault
// is the only place these plugins may write.
var errOutsideVault = errors.New("the path is outside the vault")

// resolveVaultPath resolves a vault-relative path. An absolute path, a
// leading ".." or anything that escapes the root is refused, as
// vaultHost.within does for the AI tools. Empty means the vault's own
// folder. It is a free function so the settings pages can use the same
// rule without an appState.
func resolveVaultPath(root, rel string) (string, bool) {
	if root == "" {
		return "", false
	}
	rel = strings.TrimSpace(strings.ReplaceAll(rel, "\\", "/"))
	if rel == "" {
		return root, true
	}
	if filepath.IsAbs(rel) || strings.HasPrefix(rel, "/") ||
		strings.Contains(rel, "\x00") {
		return "", false
	}
	clean := filepath.Clean(rel)
	if clean == ".." ||
		strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", false
	}
	full := filepath.Join(root, clean)
	back, err := filepath.Rel(root, full)
	if err != nil || back == ".." ||
		strings.HasPrefix(back, ".."+string(filepath.Separator)) {
		return "", false
	}
	return full, true
}

// vaultPath resolves a vault-relative path against the open vault.
func (s *appState) vaultPath(rel string) (string, bool) {
	return resolveVaultPath(s.root, rel)
}

// ensureNote opens the note at a vault-relative path, making it from
// content only when it is not there. A file that exists is never
// overwritten. The tree, index and open tabs are brought up to date in
// the same way newNoteIn does.
func (s *appState) ensureNote(relPath, content string) (string, error) {
	full, ok := s.vaultPath(relPath)
	if !ok {
		return "", errOutsideVault
	}
	if s.ws == nil {
		return "", errors.New("the window is not open")
	}
	if info, err := os.Stat(full); err == nil {
		if info.IsDir() {
			return "", fmt.Errorf("%s is a folder", full)
		}
		if _, err := s.ws.Tabs.Open(full); err != nil {
			return "", err
		}
		return full, nil
	}
	if _, err := s.ws.Tabs.Create(full, content); err != nil {
		return "", err
	}
	s.rescan()
	s.idx.Reindex(full)
	s.ws.Tree.Reveal(full)
	return full, nil
}

// templateVars is the one place the Templates plugin's date and time
// formats are read, so a daily note's template and an inserted template
// expand the same way.
func (s *appState) templateVars(title string) templating.Vars {
	o := s.coreOptions()
	return templating.Vars{
		Title:      title,
		Now:        time.Now(),
		DateFormat: o.Templates.DateFormat,
		TimeFormat: o.Templates.TimeFormat,
	}
}

// readTemplate reads one Markdown template from inside the vault.
func (s *appState) readTemplate(rel string) (string, error) {
	full, ok := s.vaultPath(rel)
	if !ok {
		return "", errOutsideVault
	}
	if !strings.EqualFold(filepath.Ext(full), ".md") {
		return "", fmt.Errorf("%s is not a Markdown note", rel)
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// sayNow puts a line at the foot of the window from a command handler,
// which already runs on the main thread. appState.say is for goroutines.
func (s *appState) sayNow(text string) {
	if s.ws != nil {
		s.ws.Say(text)
	}
}
