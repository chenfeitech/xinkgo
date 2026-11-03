// This file is the page the editor is set on.

package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
)

// editorSettings is how the editor behaves.
func (s *appState) editorSettings(c *ui.Context) {
	toggle(c, locale.SettingAutoSave(), locale.SettingAutoSaveNote(),
		&s.draft.cfg.Editor.AutoSave)
	slider(c, locale.SettingAutoSaveDelay(),
		locale.SettingAutoSaveDelayNote(), &s.draft.delay, 200, 5000)
	toggle(c, locale.SettingSoftWrap(), locale.SettingSoftWrapNote(),
		&s.draft.cfg.Editor.SoftWrap)
	toggle(c, locale.SettingLineNumbers(),
		locale.SettingLineNumbersNote(),
		&s.draft.cfg.Editor.ShowLineNumbers)
	toggle(c, locale.SettingIndentTabs(), locale.SettingIndentTabsNote(),
		&s.draft.cfg.Editor.Indent.UseTabs)
	slider(c, locale.SettingIndentSize(), locale.SettingIndentSizeNote(),
		&s.draft.indent, 1, 8)
	slider(c, locale.SettingFontSize(), locale.SettingFontSizeNote(),
		&s.draft.size, 10, 24)
	slider(c, locale.SettingLineHeight(), locale.SettingLineHeightNote(),
		&s.draft.lineHeight, 1.2, 2.2)
	slider(c, locale.SettingEditorPadding(),
		locale.SettingEditorPaddingNote(), &s.draft.editorPad,
		float64(config.MinPadding), float64(config.MaxPadding))
	slider(c, locale.SettingEditorMargin(),
		locale.SettingEditorMarginNote(), &s.draft.editorMargin,
		float64(config.DefaultMarginRight), float64(config.MaxMarginRight))
	choiceNamed(c, locale.SettingCodeTheme(),
		locale.SettingCodeThemeNote(), &s.draft.cfg.Highlight.Theme,
		highlightThemes, highlightName)
}
