// This file is the page the look of the window is chosen on: the
// appearance it follows, and the scheme it is painted in.

package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/theme"
)

// appearance is how the window looks: light or dark, and in the colors
// of which scheme.
func (s *appState) appearance(c *ui.Context) {
	group(c, locale.GroupAppearance(), func() {
		choiceNamed(c, locale.SettingLanguage(),
			locale.SettingLanguageNote(), &s.draft.cfg.Basic.Language,
			languages, languageName)
		choiceNamed(c, locale.SettingLook(), locale.SettingLookNote(),
			&s.draft.cfg.Basic.Appearance, appearances, appearanceName)
		field(c, locale.SettingTheme(), locale.SettingThemeNote(), func() {
			ui.Select(c, &s.draft.cfg.Basic.Theme, theme.Names())
		})
		s.themeCards(c)
		toggle(c, locale.SettingShowAllFiles(),
			locale.SettingShowAllFilesNote(),
			&s.draft.cfg.Basic.ShowAllFiles)
		slider(c, locale.SettingTreeFontSize(),
			locale.SettingTreeFontSizeNote(), &s.draft.treeSize, 10, 20)
	})
}

// themeCards is a card of every scheme, each drawn in the colors of the
// one it stands for and in the appearance the window has: choosing one is
// a click on it, and the one chosen is the one with the accent around it.
// They stand in a grid three cards wide, so that a card is as wide as the
// room there is for it, and not wider however narrow the window is.
func (s *appState) themeCards(c *ui.Context) {
	th := c.Theme()
	ui.Grid(c).Columns(3).Gap(th.Space(2)).FillWidth().
		Padding(th.Space(1), settingsRowPad, th.Space(3)).
		Children(func() {
			for _, sc := range theme.All() {
				s.themeCard(c, sc)
			}
		})
}

// themeCard is one card: the scheme's own colors, and its name to click.
func (s *appState) themeCard(c *ui.Context, sc theme.Scheme) {
	th := c.Theme()
	// The card is drawn in the scheme it stands for, in the appearance
	// the window has now: it is what the scheme looks like, and not a
	// picture of it.
	t := sc.Of(th.Dark)
	chosen := s.draft.cfg.Basic.Theme == sc.ID
	card := ui.Themed(c, t).FillWidth().Padding(th.Space(2)).
		Gap(th.Space(1)).Radius(6)
	if chosen {
		card = card.Border(2, t.Accent)
	} else {
		card = card.Border(1, t.Border)
	}
	card.Children(func() {
		ui.Row(c).Gap(th.Space(1)).Children(func() {
			for _, one := range []ui.Color{
				t.Background, t.Surface, t.Text, t.Accent,
			} {
				ui.Box(c).Size(themeSwatch, themeSwatch).
					Background(one).Radius(themeSwatch / 2)
			}
		})
		if chosen {
			if ui.PrimaryButton(c.Key("scheme-"+sc.ID), sc.Title).
				Clicked() {
				s.draft.cfg.Basic.Theme = sc.ID
			}
		} else if ui.Button(c.Key("scheme-"+sc.ID), sc.Title).Clicked() {
			s.draft.cfg.Basic.Theme = sc.ID
		}
		ui.Text(c, sc.Note).FontSize(th.FontSize - 2).TextColor(t.TextMuted)
	})
}

// followCodeTheme is the code preset going with the scheme the user has
// just chosen: it follows the scheme, unless the user chose a preset of
// their own, which is a choice kept.
func (s *appState) followCodeTheme(next *config.AppConfig) {
	if next.Basic.Theme == s.config.Basic.Theme {
		return
	}
	if s.config.Highlight.Theme != theme.Code(s.config.Basic.Theme) {
		return
	}
	next.Highlight.Theme = theme.Code(next.Basic.Theme)
}
