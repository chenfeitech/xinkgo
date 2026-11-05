package view

import (
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/view/geom"
)

// How wide the drawer of plugin views is, beside the rail on the right.
const pluginViewDrawerWidth float32 = 300

// pluginViewDrawer is one visible surface an Obsidian-compatible plugin
// registered. It hangs over the panels like the rail does, on the left of
// the rail so the two never cover each other. Only one surface is shown at
// a time, as the host promises: picking another tells the app to swap the
// page, and picking the one on show closes it.
func (w *Workspace) pluginViewDrawer(c *ui.Context) {
	if len(w.PluginViews) == 0 {
		return
	}
	th := c.Theme()
	ui.Row(c).Absolute().Right(geom.RailGap + geom.RailWidth + geom.RailGap).
		Top(drawerTop).Bottom(drawerFoot).AlignItems(ui.Start).
		Children(func() {
			ui.Column(c).Width(pluginViewDrawerWidth).FillHeight().
				Padding(th.Space(2)).Background(th.Background).
				Border(1, th.Border).Radius(th.Radius+2).
				Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, 0.18)).
				Children(func() {
					// The list of surfaces the enabled plugins offer.
					ui.Row(c).FillWidth().Gap(th.Space(1)).Wrap().
						Children(func() {
							for _, one := range w.PluginViews {
								view := one
								button := ui.Button(c.Key("plugin-view-"+view.ID),
									view.Name).Tooltip(view.Name)
								if w.PluginViewActive == view.ID {
									button.Selected(true)
								}
								if button.Clicked() && w.OnPluginView != nil {
									w.OnPluginView(view.ID)
								}
							}
						})
					ui.Box(c).FillWidth().Height(1).Background(th.Border)
					if w.PluginViewActive == "" || w.PluginPage == nil {
						ui.Text(c, locale.PluginViewPick()).
							FontSize(th.FontSize - 2).
							TextColor(th.TextMuted)
						return
					}
					// The plugin's own page, filled in by its ItemView,
					// drawn in the window it belongs to.
					ui.Box(c).Fill().Radius(th.Radius).Clip().
						Children(func() {
							ui.WebView(c, w.PluginPage).Fill()
						})
				})
		})
}
