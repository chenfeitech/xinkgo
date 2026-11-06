package app

import (
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/templating"
)

// dailyNotesSettings is the Daily notes plugin's own options page.
func (s *appState) dailyNotesSettings(c *ui.Context) {
	th := c.Theme()
	o := &s.draft.coreOpts.DailyNotes
	group(c, locale.GroupDailyNote(), func() {
		s.coreTextSetting(c, "daily.date", locale.DailyDateFormat(),
			locale.DailyDateFormatNote(), &o.DateFormat, true)
		s.coreFolderSetting(c, "daily.folder", locale.DailyFolder(),
			locale.DailyFolderNote(), &o.Folder, true)
		s.coreFolderSetting(c, "daily.template", locale.DailyTemplate(),
			locale.DailyTemplateNote(), &o.TemplatePath, false)
		s.coreBoolSetting(c, "daily.startup",
			locale.DailyOpenOnStartup(),
			locale.DailyOpenOnStartupNote(), &o.OpenOnStartup)
		ui.Text(c, locale.DailyTemplateHint()).
			FontSize(th.FontSize - 1).TextColor(th.TextMuted)
	})
}

// dailyOpenToday opens or makes today's note. It is also the startup
// command and the ribbon icon's action.
func (s *appState) dailyOpenToday() {
	if s.coreEnabled("daily-notes") {
		s.dailyOpenAt(time.Now())
	}
}

// dailyOpenOnStartup is called after the previous tabs are restored.
func (s *appState) dailyOpenOnStartup() {
	if !s.coreEnabled("daily-notes") {
		return
	}
	if s.coreOptions().DailyNotes.OpenOnStartup {
		s.dailyOpenAt(time.Now())
	}
}

// dailyRelPath is the vault-relative path of one day's note.
func (s *appState) dailyRelPath(day time.Time) (string, error) {
	o := s.coreOptions()
	name, err := templating.Format(day, o.DailyNotes.DateFormat)
	if err != nil {
		return "", err
	}
	return filepath.Join(o.DailyNotes.Folder, name+".md"), nil
}

// dailyContent reads the configured template and expands it with the
// Templates plugin's date and time formats. A missing or broken template
// is not fatal: the note is still made, empty, and the user is told.
func (s *appState) dailyContent(title string) (string, bool) {
	o := s.coreOptions()
	if strings.TrimSpace(o.DailyNotes.TemplatePath) == "" {
		return "", false
	}
	text, err := s.readTemplate(o.DailyNotes.TemplatePath)
	if err != nil {
		slog.Warn("daily note template not read", "err", err)
		return "", true
	}
	expanded, err := templating.ExpandFile(o.DailyNotes.TemplatePath,
		text, s.templateVars(title))
	if err != nil {
		slog.Warn("daily note template not expanded", "err", err)
		return "", true
	}
	return expanded, false
}

// dailyOpenAt is the one way a daily note is opened. A note that is there
// is opened, never overwritten; one that is not is made from the template.
func (s *appState) dailyOpenAt(day time.Time) {
	if s.root == "" {
		s.sayNow(locale.SayDailyNoVault())
		return
	}
	rel, err := s.dailyRelPath(day)
	if err != nil {
		s.sayNow(locale.SayDailyBadFormat())
		return
	}
	title := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
	content, warned := s.dailyContent(title)
	if warned {
		s.sayNow(locale.SayTemplateNotRead())
	}
	if _, err := s.ensureNote(rel, content); err != nil {
		slog.Warn("daily note not made", "path", rel, "err", err)
		s.sayNow(locale.SayDailyNotMade())
		return
	}
	s.sayNow(locale.SayDailyOpened(title))
}

// dailyOpenAdjacent steps to the nearest existing daily note before or
// after the one on show. It never creates a note for a date that has none.
func (s *appState) dailyOpenAdjacent(delta int) {
	if !s.coreEnabled("daily-notes") || s.root == "" {
		s.sayNow(locale.SayDailyNoVault())
		return
	}
	o := s.coreOptions()
	format := o.DailyNotes.DateFormat
	if err := templating.Validate(format); err != nil {
		s.sayNow(locale.SayDailyBadFormat())
		return
	}
	dir, ok := s.vaultPath(o.DailyNotes.Folder)
	if !ok {
		s.sayNow(locale.SayDailyNotMade())
		return
	}
	current := time.Now()
	if s.ws != nil && s.ws.Tabs.Active != "" {
		rel, err := filepath.Rel(dir, s.ws.Tabs.Active)
		if err == nil && rel != ".." &&
			!strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			name := strings.TrimSuffix(filepath.ToSlash(rel), ".md")
			if parsed, err := templating.Parse(name, format); err == nil {
				current = parsed
			}
		}
	}
	var dates []time.Time
	// A format such as YYYY/MM/DD puts notes in nested folders, so walk
	// the daily folder instead of reading only its top level.
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry,
		walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(d.Name()), ".md") {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		name := strings.TrimSuffix(filepath.ToSlash(rel), ".md")
		if parsed, err := templating.Parse(name, format); err == nil {
			dates = append(dates, parsed)
		}
		return nil
	})
	if err != nil {
		s.sayNow(locale.SayDailyNotMade())
		return
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	dayStart := time.Date(current.Year(), current.Month(), current.Day(),
		0, 0, 0, 0, time.Local)
	if delta < 0 {
		for i := len(dates) - 1; i >= 0; i-- {
			if dates[i].Before(dayStart) {
				s.dailyOpenAt(dates[i])
				return
			}
		}
		s.sayNow(locale.SayDailyNoEarlier())
		return
	}
	for _, one := range dates {
		if one.After(dayStart) {
			s.dailyOpenAt(one)
			return
		}
	}
	s.sayNow(locale.SayDailyNoLater())
}
