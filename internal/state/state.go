// Package state is what the app keeps between two runs: which notes were
// open, which vault, and how the window was laid out — four files in the
// application's own folder: `open_notes.json`, `vaults.json`,
// `layout.json` and `ai_sessions.json`.
//
// None of it is allowed to keep the app from starting: a file that is
// missing, or that cannot be read, or that another version wrote, gives a
// usable value and, at most, something to say in the log.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/helightxu/xinkgo/internal/session"
	"github.com/helightxu/xinkgo/internal/vault"
)

// The files in the application's user data directory.
const (
	// OpenNotesFile is which notes are open, and which one is active,
	// one list to a vault: the app shows one vault at a time, and a
	// vault the user comes back to has the notes they left open in it.
	// It holds paths, never the text of a note.
	OpenNotesFile = "open_notes.json"
	// VaultsFile is the list of vaults, and which one is open.
	VaultsFile = "vaults.json"
	// LegacyVaultFile is what versions before the list had: one folder.
	// It is read once, and never written again.
	LegacyVaultFile = "vault.json"
	// LayoutFile is how the window was laid out.
	LayoutFile = "layout.json"
	// AiSessionsFile is what was said to a model, one conversation to a
	// note.
	AiSessionsFile = "ai_sessions.json"
	// LayoutVersion is the layout this version writes. A file written by
	// another version, with another number, is thrown away and the
	// default layout built, as xink does.
	LayoutVersion = 1
)

// OpenNotes is the notes that were open, in the order their tabs were in.
type OpenNotes struct {
	Paths []string `json:"paths"`
	// Active is the path of the tab that was active: a path, and not a
	// position, because a note that is gone moves the positions of all
	// the others.
	Active *string `json:"active"`
}

// OpenNotesByVault is the notes that were open in each vault, by the
// vault's identifier: one list to a vault, so that going from one to
// another and back brings back what was open in each.
type OpenNotesByVault map[string]OpenNotes

// Of is the notes that were open in the vault of that identifier: none,
// when the app has not kept any for it.
func (all OpenNotesByVault) Of(vaultID string) OpenNotes {
	if all == nil {
		return OpenNotes{}
	}
	return all[vaultID]
}

// LoadOpenNotes reads the notes that were open in each vault. A file that
// is missing or unreadable opens nothing, and says so.
func LoadOpenNotes(dir string) (OpenNotesByVault, error) {
	data, err := os.ReadFile(filepath.Join(dir, OpenNotesFile))
	if errors.Is(err, os.ErrNotExist) {
		return OpenNotesByVault{}, nil
	}
	if err != nil {
		return OpenNotesByVault{}, fmt.Errorf("read %s: %w",
			OpenNotesFile, err)
	}
	var all OpenNotesByVault
	if err := json.Unmarshal(data, &all); err != nil {
		return OpenNotesByVault{}, fmt.Errorf("parse %s: %w",
			OpenNotesFile, err)
	}
	if all == nil {
		return OpenNotesByVault{}, nil
	}
	// Each list is brought into the shape the app can use.
	for id, on := range all {
		all[id] = NormalizeOpenNotes(on)
	}
	return all, nil
}

// LoadLegacyOpenNotes reads the file versions before the vaults wrote:
// one list of notes, that of the vault which was open. A file of the
// shape the versions since write gives nothing, and says so.
func LoadLegacyOpenNotes(dir string) (OpenNotes, error) {
	var all OpenNotesByVault
	data, err := os.ReadFile(filepath.Join(dir, OpenNotesFile))
	if errors.Is(err, os.ErrNotExist) {
		return OpenNotes{}, nil
	}
	if err != nil {
		return OpenNotes{}, fmt.Errorf("read %s: %w", OpenNotesFile, err)
	}
	if err := json.Unmarshal(data, &all); err == nil && all != nil {
		// More than one vault: written by a version that knows them.
		return OpenNotes{}, nil
	}
	var on OpenNotes
	if err := json.Unmarshal(data, &on); err != nil {
		return OpenNotes{}, fmt.Errorf("parse %s: %w", OpenNotesFile, err)
	}
	return NormalizeOpenNotes(on), nil
}

// SaveOpenNotes writes the notes that are open in each vault, whole or
// not at all: the app is killed by a crash or by the machine going off
// often enough that a half-written list of open notes is a real thing to
// guard against.
func SaveOpenNotes(dir string, all OpenNotesByVault) error {
	if all == nil {
		all = OpenNotesByVault{}
	}
	kept := make(OpenNotesByVault, len(all))
	for id, on := range all {
		kept[id] = NormalizeOpenNotes(on)
	}
	return writeJSON(filepath.Join(dir, OpenNotesFile), kept, true)
}

// WithOpenNotes is the list of every vault, with the notes of the vault
// named given as on. A vault of no identifier keeps nothing, for there
// is no vault to bring its notes back to.
func (all OpenNotesByVault) WithOpenNotes(vaultID string, on OpenNotes) OpenNotesByVault {
	if vaultID == "" {
		return all
	}
	if all == nil {
		all = OpenNotesByVault{}
	}
	all[vaultID] = on
	return all
}

// NormalizeOpenNotes puts the notes in a shape the app can use: no note
// twice, in the order they were opened, and an active note that is one of
// them.
func NormalizeOpenNotes(on OpenNotes) OpenNotes {
	paths := session.DedupPaths(on.Paths)
	if paths == nil {
		paths = []string{}
	}
	out := OpenNotes{Paths: paths}
	if len(paths) == 0 {
		return out
	}
	active := session.ResolveActive(paths, "")
	if on.Active != nil {
		active = session.ResolveActive(paths, *on.Active)
	}
	out.Active = &active
	return out
}

// LegacyVault is the one folder the versions before the list wrote.
type LegacyVault struct {
	Root string `json:"root"`
}

// LoadVaults reads the list of vaults. When there is no list, but there is
// the one folder of an older version, that folder becomes the first vault
// and the list is written: the user's vault is not lost by upgrading.
func LoadVaults(dir string) (vault.Registry, error) {
	data, err := os.ReadFile(filepath.Join(dir, VaultsFile))
	if err == nil {
		var reg vault.Registry
		if err := json.Unmarshal(data, &reg); err != nil {
			return vault.Registry{}, fmt.Errorf("parse %s: %w",
				VaultsFile, err)
		}
		if reg.Vaults == nil {
			reg.Vaults = []vault.VaultEntry{}
		}
		return reg, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return vault.Registry{}, fmt.Errorf("read %s: %w", VaultsFile, err)
	}
	return migrateLegacyVault(dir)
}

// migrateLegacyVault turns the folder of an older version into the first
// vault of the list, and writes the list.
func migrateLegacyVault(dir string) (vault.Registry, error) {
	empty := vault.Registry{Vaults: []vault.VaultEntry{}}
	data, err := os.ReadFile(filepath.Join(dir, LegacyVaultFile))
	if errors.Is(err, os.ErrNotExist) {
		return empty, nil
	}
	if err != nil {
		return empty, fmt.Errorf("read %s: %w", LegacyVaultFile, err)
	}
	var legacy LegacyVault
	if err := json.Unmarshal(data, &legacy); err != nil {
		return empty, fmt.Errorf("parse %s: %w", LegacyVaultFile, err)
	}
	info, err := os.Stat(legacy.Root)
	if err != nil || !info.IsDir() {
		// A folder that is gone is not worth migrating.
		return empty, nil
	}
	reg := empty
	e := reg.Add(legacy.Root)
	reg.Active = &e.ID
	if err := SaveVaults(dir, reg); err != nil {
		return reg, err
	}
	return reg, nil
}

// SaveVaults writes the list of vaults, replacing the old file only once
// the new one is on disk.
func SaveVaults(dir string, reg vault.Registry) error {
	if reg.Vaults == nil {
		reg.Vaults = []vault.VaultEntry{}
	}
	return writeJSON(filepath.Join(dir, VaultsFile), reg, true)
}

// Layout is how the window was laid out: how wide the side panels were.
type Layout struct {
	Version int `json:"version"`
	// Left and Right are the widths of the file tree and the preview, in
	// DIPs; 0 hides the panel. The note in the middle takes what is left,
	// so that making the window wider gives the text more room.
	Left  float32 `json:"left"`
	Right float32 `json:"right"`
	// LastLeft and LastRight are the widths the panels had before they
	// were shut, so that opening one again gives back what it was and
	// not a width thought up on the spot.
	LastLeft  float32 `json:"last_left,omitempty"`
	LastRight float32 `json:"last_right,omitempty"`
}

// TreeShown is whether the file tree stands. A panel of no width is one
// that was shut.
func (l Layout) TreeShown() bool { return l.Left > 0 }

// PreviewShown is whether the preview stands.
func (l Layout) PreviewShown() bool { return l.Right > 0 }

// DefaultLayout is the layout of a first start.
func DefaultLayout() Layout {
	return Layout{Version: LayoutVersion, Left: 260, Right: 420}
}

// LoadLayout reads the layout. One written by another version, or one that
// cannot be read, gives the default: a window the user can use is worth
// more than the shape it had.
func LoadLayout(dir string) (Layout, error) {
	data, err := os.ReadFile(filepath.Join(dir, LayoutFile))
	if errors.Is(err, os.ErrNotExist) {
		return DefaultLayout(), nil
	}
	if err != nil {
		return DefaultLayout(), fmt.Errorf("read %s: %w", LayoutFile, err)
	}
	var l Layout
	if err := json.Unmarshal(data, &l); err != nil {
		return DefaultLayout(), fmt.Errorf("parse %s: %w", LayoutFile, err)
	}
	if l.Version != LayoutVersion {
		return DefaultLayout(), nil
	}
	if l.Left < 0 {
		l.Left = 0
	}
	if l.Right < 0 {
		l.Right = 0
	}
	return l, nil
}

// SaveLayout writes the layout, whole or not at all, as the open notes
// are written: a half-written file is no layout at all, and the window
// would come up with panels of no width.
func SaveLayout(dir string, l Layout) error {
	l.Version = LayoutVersion
	return writeJSON(filepath.Join(dir, LayoutFile), l, true)
}

// writeJSON writes v as indented JSON. With atomic, the old file is
// replaced only once the new one is on disk, which is what the vaults and
// the AI's sessions need: they are the app's own records, and losing them
// is losing the user's work.
func writeJSON(path string, v any, atomic bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", filepath.Base(path), err)
	}
	data = append(data, '\n')
	if !atomic {
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", filepath.Base(path), err)
		}
		return nil
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(tmp), err)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err,
		os.ErrNotExist) {
		return fmt.Errorf("replace %s: %w", filepath.Base(path), err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace %s: %w", filepath.Base(path), err)
	}
	return nil
}
