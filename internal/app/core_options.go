package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/plugin"
	"github.com/helightxu/xinkgo/internal/templating"
	"github.com/helightxu/xinkgo/internal/webviewer"
)

// refreshCoreLists reads the template folder once, when the options page
// is opened or its folder changes. A view frame never reads the disk.
func (st *settings) refreshCoreLists(root string, o plugin.CoreOptions) {
	st.tplFolderRead = o.Templates.Folder
	st.tplFiles, st.tplBad = readTemplateDir(root, o.Templates.Folder)
}

// coreTemplateInfo is what the Templates options page shows about one
// file: its name, where it stands in the vault, and how big and how old
// it is. Read once when the list is refreshed, never once a frame.
type coreTemplateInfo struct {
	Name    string
	Rel     string
	Size    int64
	ModTime time.Time
}

// readTemplateDir lists the Markdown files of a vault-relative folder.
// A file that cannot be used is named separately so the page can say so
// without hiding the rest.
func readTemplateDir(root, rel string) ([]coreTemplateInfo, []string) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return nil, nil
	}
	full, ok := resolveVaultPath(root, rel)
	if !ok {
		return nil, []string{errOutsideVault.Error()}
	}
	entries, err := os.ReadDir(full)
	if err != nil {
		return nil, []string{err.Error()}
	}
	var files []coreTemplateInfo
	for _, one := range entries {
		if one.IsDir() {
			continue
		}
		if !strings.EqualFold(filepath.Ext(one.Name()), ".md") {
			continue
		}
		info, err := one.Info()
		if err != nil {
			continue
		}
		files = append(files, coreTemplateInfo{
			Name: strings.TrimSuffix(one.Name(),
				filepath.Ext(one.Name())),
			Rel:     filepath.ToSlash(filepath.Join(rel, one.Name())),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) <
			strings.ToLower(files[j].Name)
	})
	return files, nil
}

// validateCoreOptions checks the formats and paths a committed change may
// hold. A bad value is not written: the draft keeps it so the user can fix
// it, and the page says what is wrong.
func validateCoreOptions(root string, o plugin.CoreOptions) error {
	for _, format := range []string{
		o.DailyNotes.DateFormat, o.Templates.DateFormat,
		o.Templates.TimeFormat,
	} {
		if err := templating.Validate(format); err != nil {
			return err
		}
	}
	if _, err := webviewer.SearchURL(o.WebViewer.SearchEngine, "test"); err != nil {
		return err
	}
	if root == "" {
		return nil
	}
	for _, path := range []string{
		o.DailyNotes.Folder, o.DailyNotes.TemplatePath,
		o.Templates.Folder, o.WebViewer.SaveFolder,
		o.WebViewer.SaveTemplate, o.WebViewer.RuleFile,
	} {
		if strings.TrimSpace(path) == "" {
			continue
		}
		if _, ok := resolveVaultPath(root, path); !ok {
			return errOutsideVault
		}
	}
	return nil
}

// commitCoreOptions writes the draft once, on a committed field, and
// refreshes the cached lists in case a folder changed.
func (s *appState) commitCoreOptions() {
	if err := validateCoreOptions(s.root, s.draft.coreOpts); err != nil {
		s.sayNow(locale.SayOptionBadFormat())
		if err == errOutsideVault {
			s.sayNow(locale.SayOptionOutsideVault())
		}
		return
	}
	s.setCoreOptions(s.draft.coreOpts)
	s.draft.coreOpts = s.coreOptions()
	s.draft.refreshCoreLists(s.root, s.draft.coreOpts)
}

// coreTextSetting is one text field of a built-in plugin's options page.
// It commits on Enter/blur, shows a live preview of a format, and says so
// when the format cannot be used.
func (s *appState) coreTextSetting(c *ui.Context, key, name, note string,
	value *string, format bool) {
	th := c.Theme()
	settingRow(c, name, note, settingsControlWidth, func() {
		if ui.TextInput(c.Key("core-"+key), value).Submitted() {
			s.commitCoreOptions()
		}
	})
	if !format {
		return
	}
	var line string
	if err := templating.Validate(*value); err != nil {
		line = locale.SettingBadFormat()
		ui.Text(c, line).FontSize(th.FontSize - 1).TextColor(th.Danger)
		return
	}
	if shown, err := templating.Format(time.Now(), *value); err == nil {
		ui.Text(c, locale.SettingFormatPreview(shown)).
			FontSize(th.FontSize - 1).TextColor(th.TextMuted)
	}
}

// coreFolderSetting is a vault-relative path with a Browse button. The
// dialog runs off the main thread; the chosen path comes back to it.
func (s *appState) coreFolderSetting(c *ui.Context, key, name, note string,
	value *string, directory bool) {
	settingRow(c, name, note, settingsControlWidth, func() {
		if ui.TextInput(c.Key("core-"+key), value).Grow(1).Submitted() {
			s.commitCoreOptions()
		}
		if ui.Button(c.Key("core-browse-"+key),
			locale.SettingBrowse()).Clicked() {
			go s.pickCorePath(key, directory)
		}
	})
}

// coreBoolSetting is one switch of a built-in plugin's options page.
func (s *appState) coreBoolSetting(c *ui.Context, key, name, note string,
	value *bool) {
	settingRow(c, name, note, settingsControlWidth, func() {
		if ui.Switch(c.Key("core-"+key), value).Label("").Changed() {
			s.commitCoreOptions()
		}
	})
}

// pickCorePath asks for a folder or a Markdown file and stores it as a
// vault-relative path. A path outside the vault is refused.
func (s *appState) pickCorePath(key string, directory bool) {
	opts := mygo.OpenDialogOptions{
		Title:     locale.DialogChooseFolder(),
		Directory: directory,
	}
	if !directory {
		opts.Title = locale.DialogChooseFolder()
		opts.Filters = []mygo.FileFilter{{
			Name: "Markdown", Extensions: []string{"md"},
		}}
	}
	picked, err := mygo.Dialog.Open(opts)
	if err != nil || len(picked) == 0 {
		return
	}
	rel, ok := s.relativeToVault(picked[0])
	mygo.RunOnMain(func() {
		if !ok {
			s.sayNow(locale.SayOptionOutsideVault())
			return
		}
		s.applyCorePath(key, rel)
		s.commitCoreOptions()
	})
}

// relativeToVault turns an absolute path into a vault-relative one.
func (s *appState) relativeToVault(abs string) (string, bool) {
	if s.root == "" {
		return "", false
	}
	rel, err := filepath.Rel(s.root, abs)
	if err != nil || rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

// applyCorePath puts a chosen path into the right field of the draft.
func (s *appState) applyCorePath(key, rel string) {
	o := &s.draft.coreOpts
	switch key {
	case "daily.folder":
		o.DailyNotes.Folder = rel
	case "daily.template":
		o.DailyNotes.TemplatePath = rel
	case "tpl.folder":
		o.Templates.Folder = rel
	case "web.folder":
		o.WebViewer.SaveFolder = rel
	case "web.template":
		o.WebViewer.SaveTemplate = rel
	case "web.rule":
		o.WebViewer.RuleFile = rel
	}
}

// coreTemplateList is the cached list of templates, read when the page
// opened or the folder changed.
func (s *appState) coreTemplateList() []coreTemplateInfo {
	return s.draft.tplFiles
}
