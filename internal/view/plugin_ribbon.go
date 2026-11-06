package view

import "github.com/egoist/mygo/ui"

// How wide the plugin ribbon is, and how large one of its buttons is.
const (
	pluginRibbonWidth  float32 = 34
	pluginRibbonButton float32 = 26
	pluginRibbonIcon   float32 = 14
)

// pluginRibbon is the strip reserved at the left of the file tree for
// plugin icons. It is always drawn while the core plugin stands, so the
// tree does not shift when a plugin is enabled or disabled.
func (w *Workspace) pluginRibbon(c *ui.Context) {
	if !w.coreOn(CorePluginRibbon) {
		return
	}
	th := c.Theme()
	ui.Column(c).Width(pluginRibbonWidth).FillHeight().
		Padding(4, 4).Gap(4).Children(func() {
		for _, one := range w.PluginRibbon {
			action := one
			button := ui.ButtonBase(c.Key("plugin-ribbon-"+action.ID)).
				Label(action.Name).Tooltip(action.Name).
				Size(pluginRibbonButton, pluginRibbonButton).
				Radius(th.Radius).Justify(ui.Center).
				AlignItems(ui.Center)
			button.Children(func() {
				ui.Icon(c, pluginIcon(action.Icon)).
					Size(pluginRibbonIcon, pluginRibbonIcon).
					TextColor(th.TextMuted)
			})
			if button.Clicked() && w.OnPluginAction != nil {
				w.OnPluginAction(action.ID)
			}
		}
		ui.Box(c).Grow(1)
	})
}
