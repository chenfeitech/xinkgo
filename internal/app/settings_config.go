// This file is the page the configuration itself is kept on: where
// it stands, and the way out of it and back into it.

package app

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
)

// configFile is the settings themselves: kept as a file, given away and
// taken in again.
func (s *appState) configFile(c *ui.Context) {
	buttons(c, locale.GroupConfigFile(),
		filepath.Join(s.dir, "config.json"), func() {
			if ui.Button(c.Key("export-config"), locale.ConfigExport()).
				Clicked() {
				s.exportConfig()
			}
			if ui.Button(c.Key("import-config"), locale.ConfigImport()).
				Clicked() {
				s.importConfig()
			}
			if ui.Button(c.Key("default-config"),
				locale.ConfigReset()).Clicked() {
				s.draft.cfg = config.Default()
				s.draft.read(&s.draft.cfg)
			}
		})
}

// exportConfig writes the settings to a file the user names, so that
// they may be taken to another machine.
func (s *appState) exportConfig() {
	go func() {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Title:       locale.DialogSaveSettings(),
			DefaultPath: filepath.Join(s.dir, "xinkgo-config.json"),
			Filters:     []mygo.FileFilter{{Name: "JSON", Extensions: []string{"json"}}},
		})
		if err != nil || path == "" {
			return
		}
		data, err := json.MarshalIndent(s.config, "", "  ")
		if err != nil {
			slog.Warn("the settings could not be written out", "err", err)
			return
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			slog.Warn("the settings could not be written out", "err", err)
		}
	}()
}

// importConfig reads the settings from a file the user names.
func (s *appState) importConfig() {
	go func() {
		picked, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title:   locale.DialogImportSettings(),
			Filters: []mygo.FileFilter{{Name: "JSON", Extensions: []string{"json"}}},
		})
		if err != nil || len(picked) == 0 {
			return
		}
		data, err := os.ReadFile(picked[0])
		if err != nil {
			slog.Warn("the settings could not be read", "err", err)
			return
		}
		var cfg config.AppConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			slog.Warn("that is not a settings file", "err", err)
			return
		}
		cfg.Sanitize()
		mygo.RunOnMain(func() {
			s.config = cfg
			s.draft.cfg = cfg
			s.draft.read(&cfg)
			if err := s.config.Save(s.dir); err != nil {
				slog.Warn("the settings could not be written", "err", err)
			}
			s.applyConfig()
		})
	}()
}
