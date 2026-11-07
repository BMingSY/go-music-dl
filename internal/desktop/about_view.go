package desktop

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (s *app) aboutView(c *ui.Context) {
	ui.Text(c, "music-dl").FontSize(28).Bold().TextColor(accent)
	ui.Text(c, "版本 "+mygo.App.Version()).FontSize(14).TextColor(muted)
	ui.Text(c, "网易云音乐 · Bilibili · 本地音乐").FontSize(14)
	ui.Column(c).Gap(7).Children(func() {
		for _, link := range []struct{ label, name, url string }{{"项目地址", "BMingSY/go-music-dl", "https://github.com/BMingSY/go-music-dl"}, {"改自", "guohuiyuan/go-music-dl", "https://github.com/guohuiyuan/go-music-dl"}} {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, link.label).Width(56).FontSize(13).TextColor(muted)
				ui.Link(c, link.name, link.url).FontSize(13).TextColor(accent).Underline()
			})
		}
	})
	if s.pendingUpdate != nil {
		label := "下载更新"
		if s.updateBusy {
			label = "查看下载进度"
		}
		ui.Button(c, label).OnClick(func() { s.modal = "update" })
	} else {
		ui.Button(c, "检查更新").Disabled(s.updateBusy).OnClick(s.checkUpdate)
	}
	ui.Text(c, s.updateStatus).FontSize(13).TextColor(muted).MaxLines(5)
	if s.updateBusy {
		ui.Box(c).Height(5).Background(border).Radius(3).Children(func() { ui.Box(c).WidthPercent(float32(s.updateProgress * 100)).Height(5).Background(accent).Radius(3) })
	}
}
