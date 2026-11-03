// This file is the page the preview is set on: what a note is read
// as, and the folder the styles of an article are kept in.

package app

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/wechat"
)

// previewSettings is what a note looks like beside what it says.
func (s *appState) previewSettings(c *ui.Context) {
	toggle(c, locale.SettingSyncScroll(), locale.SettingSyncScrollNote(),
		&s.draft.cfg.Preview.SyncScroll)
	choiceNamed(c, locale.SettingSyncMode(), locale.SettingSyncModeNote(),
		&s.draft.cfg.Preview.SyncMode, syncModes, syncModeName)
	choiceNamed(c, locale.SettingImages(), locale.SettingImagesNote(),
		&s.draft.cfg.Preview.ImageDisplay, imageShows, imageShowName)
	slider(c, locale.SettingImageWidth(), locale.SettingImageWidthNote(),
		&s.draft.imageWidth, 20, 100)
	choiceNamed(c, locale.SettingLinkOpen(), locale.SettingLinkOpenNote(),
		&s.draft.cfg.Preview.LinkOpen, linkOpens, linkOpenName)
	slider(c, locale.SettingMaxWidth(), locale.SettingMaxWidthNote(),
		&s.draft.width, 400, 1400)
	slider(c, locale.SettingPreviewPadding(),
		locale.SettingPreviewPaddingNote(), &s.draft.previewPadX,
		float64(config.MinPadding), float64(config.MaxPadding))
	slider(c, locale.SettingPreviewPaddingY(),
		locale.SettingPreviewPaddingYNote(), &s.draft.previewPadY,
		float64(config.MinPadding), float64(config.MaxPadding))
	choiceNamed(c, locale.SettingDensity(), locale.SettingDensityNote(),
		&s.draft.cfg.Preview.Density, densities, densityName)
	group(c, locale.GroupExport(), func() {
		choiceNamed(c, locale.ExportPageSize(),
			locale.ExportPageSizeNote(), &s.draft.cfg.Export.PageSize,
			pages, pageSizeName)
		choiceNamed(c, locale.ExportImages(),
			locale.ExportImagesNote(), &s.draft.cfg.Export.Images,
			exportImages, exportImagesName)
	})
	group(c, locale.GroupWechat(), func() {
		s.wechatDirField(c)
		s.wechatThemeRows(c)
	})
}

// wechatDirField is the folder the styles of WeChat are read from: named
// by hand, or asked of the machine with the button beside it. One file of
// JSON in it is one style, so a folder of one's own is how a style is
// added, and shared, without touching the app.
func (s *appState) wechatDirField(c *ui.Context) {
	th := c.Theme()
	buttons(c, locale.SettingWechatDir(),
		locale.SettingWechatDirNote(s.wechatDir()), func() {
			ui.Row(c).Gap(th.Space(1)).AlignItems(ui.Center).
				Children(func() {
					ui.TextInput(c, &s.draft.wechatDir).
						Placeholder(s.wechatDir()).Width(wechatDirWidth)
					if ui.Button(c.Key("wechat-dir"),
						locale.SettingWechatDirBrowse()).Clicked() {
						s.chooseWechatDir()
					}
				})
		})
}

// wechatThemeRows is what the folder named above holds: a row to a style,
// saying its name, which of it this is, who made it, and what it says of
// itself; and after them the files of the folder that are no styles, and
// why. A folder is read again when another one is named, and not once a
// frame while it is being written.
func (s *appState) wechatThemeRows(c *ui.Context) {
	th := c.Theme()
	if s.draft.wechatRead != s.draft.wechatDir {
		s.draft.wechatRead = s.draft.wechatDir
		s.draft.wechatThemes, s.draft.wechatBad = wechat.OfDir(
			s.draft.wechatDirOf())
	}
	themes, bad := s.draft.wechatThemes, s.draft.wechatBad
	if len(themes) == 0 && len(bad) == 0 {
		field(c, locale.WechatThemesNone(), "", func() {})
		return
	}
	for _, t := range themes {
		field(c, t.Name, wechatByOf(t), func() {})
	}
	if len(themes) > 0 {
		ui.Column(c).FillWidth().Children(func() {
			ui.Text(c, locale.WechatThemesBuiltin(
				strconv.Itoa(len(wechat.Builtin())))).
				FontSize(th.FontSize - 1).TextColor(th.TextMuted)
		})
	}
	// What the folder holds that is no style: a file that will not
	// read, or that names no id, or whose id another has taken.
	for _, why := range bad {
		ui.Column(c).FillWidth().Children(func() {
			ui.Text(c, why).FontSize(th.FontSize - 1).
				TextColor(th.Danger)
		})
	}
}

// wechatByOf is what a row of a style says under its name: which of it
// this is, and who made it, when its file says, and what it says of
// itself when it says anything at all.
func wechatByOf(t wechat.Theme) string {
	var b strings.Builder
	if t.Version != "" {
		b.WriteString(t.Version)
	}
	if t.Author != "" {
		if b.Len() > 0 {
			b.WriteString(" · ")
		}
		b.WriteString(t.Author)
	}
	if t.Description != "" {
		if b.Len() > 0 {
			b.WriteString(" — ")
		}
		b.WriteString(t.Description)
	}
	return b.String()
}
