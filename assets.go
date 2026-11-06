package main

import (
	"embed"
	"github.com/egoist/mygo/ui"
	"sync"
)

//go:embed assets/*.ttf assets/*.png
var assets embed.FS
var assetsOnce sync.Once
var mascot *ui.Bitmap
var icons = map[string]string{
	"music": "\uf001", "search": "\uf002", "fire": "\uf06d", "folder": "\uf07c", "grid": "\uf009", "rank": "\uf091", "heart": "\uf004", "gear": "\uf013", "download": "\uf019", "history": "\uf1da", "clock": "\uf017", "up": "\uf062", "down": "\uf063", "headphones": "\uf025", "play": "\uf04b", "pause": "\uf04c", "prev": "\uf048", "next": "\uf051", "repeat": "\uf01e", "shuffle": "\uf074", "list": "\uf03a", "close": "\uf00d", "volume": "\uf028", "file": "\uf15c", "image": "\uf03e", "back": "\uf060", "mini": "\uf2d0", "user": "\uf007", "chevron": "\uf107", "sliders": "\uf1de", "plus": "\uf067", "trash": "\uf2ed", "link": "\uf0c1", "video": "\uf008",
	"font": "\uf031", "keyboard": "\uf11c", "info": "\uf05a", "cookie": "\uf564", "more": "\uf141", "stop": "\uf04d", "cloud": "\uf0ed", "switch": "\uf021", "ranking": "\ue561",
}

func initAssets() {
	assetsOnce.Do(func() {
		for _, f := range []struct{ file, name string }{{"fa-solid-900.ttf", "MusicDL Icons"}, {"fa-regular-400.ttf", "MusicDL Outline"}} {
			b, _ := assets.ReadFile("assets/" + f.file)
			_ = ui.RegisterFont(b, f.name)
		}
		b, _ := assets.ReadFile("assets/mascot-0.png")
		mascot, _ = ui.DecodeBitmap(b)
	})
}
func glyph(c *ui.Context, name string, size float32, color ui.Color) ui.Element {
	return ui.Text(c, icons[name]).Font("MusicDL Icons").FontSize(size).TextColor(color).SingleLine()
}
func iconButton(c *ui.Context, key, icon, label string, size float32) ui.Element {
	return colorIconButton(c, key, icon, label, size, muted, false)
}
func colorIconButton(c *ui.Context, key, icon, label string, size float32, color ui.Color, outline bool) ui.Element {
	return ui.Button(c.Key(key), "").Label(label).Tooltip(label).Size(size, size).Padding(0).Radius(size/2).Background(white).Border(0, ui.Transparent).Children(func() {
		i := glyph(c, icon, 14, color)
		if outline {
			i.Font("MusicDL Outline")
		}
	})
}
func pill(c *ui.Context, icon, label, from, to string, fn func()) {
	ui.Button(c.Key(label), "").Label(label).Radius(20).Padding(8, 18).Height(37).Border(0, ui.Transparent).Gradient(ui.Hex(from), ui.Hex(to), 135).Shadow(0, 4, 15, 0, ui.Hex(from).Alpha(.35)).OnClick(fn).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() { glyph(c, icon, 14, white); ui.Text(c, label).FontSize(14).FontWeight(600).TextColor(white) })
	})
}
