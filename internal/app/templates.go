package app

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/fs"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/templating"
	"github.com/helightxu/xinkgo/internal/view"
)

// templatesSettings is the Templates plugin's own page. It appears in the
// settings sidebar only while the plugin is on.
func (s *appState) templatesSettings(c *ui.Context) {
	th := c.Theme()
	o := &s.draft.coreOpts.Templates
	group(c, locale.GroupTemplates(), func() {
		s.coreFolderSetting(c, "tpl.folder", locale.TplFolder(),
			locale.TplFolderNote(), &o.Folder, true)
		s.coreTextSetting(c, "tpl.date", locale.TplDateFormat(),
			locale.TplDateFormatNote(), &o.DateFormat, true)
		s.coreTextSetting(c, "tpl.time", locale.TplTimeFormat(),
			locale.TplTimeFormatNote(), &o.TimeFormat, true)
		ui.Text(c, locale.TplVarsNote()).FontSize(th.FontSize - 1).
			TextColor(th.TextMuted)
		s.templateListHeader(c, th)
		cardList(c, func() {
			cardBox(c, func() { s.templateRows(c, th) })
		})
		if len(s.draft.tplBad) > 0 {
			ui.Text(c, locale.TplBadTitle()).Bold()
			cardList(c, func() {
				cardBox(c, func() {
					for i, bad := range s.draft.tplBad {
						if i > 0 {
							ui.Box(c).FillWidth().Height(1).
								Background(th.Border)
						}
						ui.Text(c, bad).FontSize(th.FontSize - 1).
							TextColor(th.Danger)
					}
				})
			})
		}
	})
}

// templateListHeader names the list and counts what it holds, as the
// plugin manager counts its cards.
func (s *appState) templateListHeader(c *ui.Context, th *ui.Theme) {
	ui.Row(c).FillWidth().Gap(th.Space(1)).AlignItems(ui.Center).
		Padding(th.Space(1), settingsRowPad).Children(func() {
		ui.Text(c, locale.TplListTitle()).Bold()
		if n := len(s.coreTemplateList()); n > 0 {
			ui.Badge(c, locale.TplCount(strconv.Itoa(n))).
				FontSize(11)
		}
		ui.Box(c).Grow(1)
	})
}

// templateRows is the folder's contents inside one card. An empty folder
// gets a quiet hint instead of a bare line of text.
func (s *appState) templateRows(c *ui.Context, th *ui.Theme) {
	templates := s.coreTemplateList()
	if len(templates) == 0 {
		ui.Column(c).FillWidth().Gap(th.Space(1)).
			Padding(th.Space(3), th.Space(2)).
			AlignItems(ui.Center).Children(func() {
			ui.Icon(c, view.IconSettingsFile).Size(24, 24).
				TextColor(th.TextMuted)
			ui.Text(c, locale.TplListNone()).TextColor(th.TextMuted)
			ui.Text(c, locale.TplEmptyHint()).
				FontSize(th.FontSize - 1).TextColor(th.TextMuted)
		})
		return
	}
	for i, one := range templates {
		if i > 0 {
			ui.Box(c).FillWidth().Height(1).Background(th.Border)
		}
		s.templateRow(c, th, one)
	}
}

// templateRow is one template: an icon, its name with a Markdown badge,
// and a line of where it stands, how big it is and when it changed.
func (s *appState) templateRow(c *ui.Context, th *ui.Theme,
	one coreTemplateInfo) {
	ui.Row(c).FillWidth().Gap(th.Space(2)).AlignItems(ui.Center).
		Padding(th.Space(1.5), th.Space(1)).Children(func() {
		ui.Icon(c, view.IconSettingsFile).Size(18, 18).
			TextColor(th.TextMuted)
		ui.Column(c).Grow(1).Gap(th.Space(0.5)).Children(func() {
			ui.Row(c).Gap(th.Space(1)).AlignItems(ui.Center).
				Children(func() {
					ui.Text(c, one.Name).Bold()
					ui.Badge(c, ".md").FontSize(11)
				})
			ui.Text(c, locale.TplMeta(one.Rel,
				formatTemplateSize(one.Size),
				one.ModTime.Format("2006-01-02 15:04"))).
				FontSize(th.FontSize - 1).TextColor(th.TextMuted).
				SingleLine().Ellipsis("…")
		})
	})
}

// formatTemplateSize writes a file size in the units a template folder
// actually holds; the numbers are data, not a message to translate.
func formatTemplateSize(size int64) string {
	switch {
	case size < 1024:
		return fmt.Sprintf("%d B", size)
	case size < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
}

// templateNames reads the template folder once per command. Commands are
// not drawn in a frame, so a single ReadDir here is fine and always fresh.
func (s *appState) templateNames() []string {
	o := s.coreOptions()
	infos, _ := readTemplateDir(s.root, o.Templates.Folder)
	names := make([]string, 0, len(infos))
	for _, one := range infos {
		names = append(names, one.Name)
	}
	return names
}

// templatesInsert opens the chooser and puts the expanded template at the
// caret of the note on show.
func (s *appState) templatesInsert() {
	if !s.coreEnabled("templates") || s.ws == nil {
		return
	}
	if s.ws.ActiveBasename() == "" {
		s.sayNow(locale.SayTemplateNoActive())
		return
	}
	names := s.templateNames()
	if len(names) == 0 {
		s.sayNow(locale.SayTemplateFolderNotSet())
		return
	}
	s.ws.OpenPicker(locale.TplPickerInsert(), names, s.insertTemplateNamed)
}

func (s *appState) insertTemplateNamed(name string) {
	text, err := s.templateText(name)
	if err != nil {
		slog.Warn("template not read", "name", name, "err", err)
		s.sayNow(locale.SayTemplateNotRead())
		return
	}
	expanded, err := templating.Expand(text,
		s.templateVars(s.ws.ActiveBasename()))
	if err != nil {
		s.sayNow(locale.SayOptionBadFormat())
		return
	}
	if !s.ws.InsertInActive(expanded) {
		s.sayNow(locale.SayTemplateNoActive())
		return
	}
	s.sayNow(locale.SayTemplateInserted())
}

// templatesCreateNote makes a new note in the tree's folder from a
// template. The title variable is the new note's own name.
func (s *appState) templatesCreateNote() {
	if !s.coreEnabled("templates") || s.ws == nil {
		return
	}
	names := s.templateNames()
	if len(names) == 0 {
		s.sayNow(locale.SayTemplateFolderNotSet())
		return
	}
	s.ws.OpenPicker(locale.TplPickerCreate(), names,
		s.createNoteFromTemplate)
}

func (s *appState) createNoteFromTemplate(name string) {
	text, err := s.templateText(name)
	if err != nil {
		slog.Warn("template not read", "name", name, "err", err)
		s.sayNow(locale.SayTemplateNotRead())
		return
	}
	dir := s.ws.Tree.DirOf()
	if dir == "" {
		dir = s.root
	}
	file := fs.UniqueName(dir, "untitled.md")
	title := strings.TrimSuffix(file, filepath.Ext(file))
	expanded, err := templating.Expand(text, s.templateVars(title))
	if err != nil {
		s.sayNow(locale.SayOptionBadFormat())
		return
	}
	rel, ok := s.relativeToVault(filepath.Join(dir, file))
	if !ok {
		s.sayNow(locale.SayOptionOutsideVault())
		return
	}
	if _, err := s.ensureNote(rel, expanded); err != nil {
		slog.Warn("note from template not made", "err", err)
		s.sayNow(locale.SayTemplateNotMade())
		return
	}
	s.sayNow(locale.SayDailyOpened(title))
}

// templateText reads and expands nothing: it only reads one template from
// the configured folder. Expansion is the caller's decision.
func (s *appState) templateText(name string) (string, error) {
	o := s.coreOptions()
	rel := filepath.Join(o.Templates.Folder, name+".md")
	return s.readTemplate(rel)
}
