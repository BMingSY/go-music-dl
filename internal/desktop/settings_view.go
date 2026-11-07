package desktop

import (
	"github.com/egoist/mygo/ui"
)

func setting(c *ui.Context, title string, body func()) {
	ui.Column(c).Padding(12).Border(1, ui.Hex("#e2e8f0")).Radius(10).Background(ui.Hex("#f8fafc")).Gap(10).Children(func() { ui.Text(c, title).FontSize(13).FontWeight(600).TextColor(muted); body() })
}

func (s *app) settingsView(c *ui.Context) {
	ui.Row(c).Padding(4).Gap(5).Background(ui.Hex("#f8fafc")).Border(1, border).Radius(12).Children(func() {
		for _, tab := range []struct{ id, label, icon string }{{"general", "常规", "sliders"}, {"download", "下载与同步", "download"}, {"cookies", "平台 Cookie", "cookie"}} {
			bg := ui.Transparent
			if s.settingsTab == tab.id {
				bg = white
			}
			fg := ui.Hex("#64748b")
			if s.settingsTab == tab.id {
				fg = ui.Hex("#047857")
			}
			b := ui.Button(c, "").Label(tab.label).Grow(1).Background(bg).Border(0, ui.Transparent).Radius(8).Padding(9, 12).OnClick(func() { s.settingsTab = tab.id }).Children(func() {
				ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() { glyph(c, tab.icon, 13, fg); ui.Text(c, tab.label).FontSize(13).FontWeight(700).TextColor(fg) })
			})
			if s.settingsTab == tab.id {
				b.Shadow(0, 2, 8, 0, ui.RGBA(15, 23, 42, .08))
			}
		}
	})
	ui.Scroll(c.Key(s.settingsTab)).Grow(1).Children(func() {
		ui.Column(c).Padding(2, 4).Gap(14).Children(func() {
			switch s.settingsTab {
			case "general":
				s.generalSettings(c)
			case "download":
				settingToggle(c, &s.cfg.EmbedMetadata, "下载时内嵌元数据（封面、歌词）", "支持 MP3、FLAC 和 M4A；播放时使用原始音频流。")
				settingToggle(c, &s.cfg.SaveLyrics, "保存音乐时同时生成同名 LRC", "已有同名歌词不会被覆盖。")
				setting(c, "下载目录", func() {
					ui.Row(c).Gap(8).Children(func() {
						ui.TextInput(c, &s.cfg.DownloadDir).Label("下载目录").Grow(1).FontSize(14)
						ui.Button(c, "选择目录").OnClick(s.pickDownloadFolder)
					})
					ui.Row(c).Gap(8).Children(func() {
						ui.Button(c, "打开目录").OnClick(s.openDownloadDir)
						ui.Button(c, "下载管理").OnClick(func() { s.modal = "downloads" })
					})
				})
				setting(c, "FFmpeg 路径", func() {
					ui.Text(c, "安装包已包含解码器，也可指定自己的 FFmpeg。").FontSize(12).TextColor(muted)
					ui.Row(c).Gap(8).Children(func() {
						ui.TextInput(c, &s.cfg.FFmpeg).Label("FFmpeg 路径").Placeholder("自动查找").Grow(1)
						ui.Button(c, "浏览文件").OnClick(s.pickFFmpeg)
					})
				})
				setting(c, "歌单备份", func() {
					ui.Row(c).Gap(8).Children(func() {
						ui.Button(c, "导出歌单").OnClick(s.exportPlaylists)
						ui.Button(c, "导入歌单").OnClick(s.importPlaylists)
					})
				})
			case "cookies":
				ui.Text(c, "扫码登录后自动保存 Cookie，也可以手动填写。").FontSize(12).TextColor(muted)
				for _, src := range []struct {
					id, name string
					cookie   *string
				}{{"netease", "网易云音乐", &s.cfg.NeteaseCookie}, {"bilibili", "Bilibili", &s.cfg.BilibiliCookie}} {
					setting(c, src.name, func() {
						ui.TextInput(c.Key(src.id), src.cookie).Label(src.name + " Cookie").Password().FontSize(14)
						ui.Row(c).Gap(8).Children(func() {
							ui.Button(c, "扫码登录 "+src.name).OnClick(func() { s.startLogin(src.id) })
							ui.Button(c.Key(src.id+"-logout"), "清除 Cookie").OnClick(func() { *src.cookie = ""; s.save() })
						})
					})
				}
			}
		})
	})
	if s.status != "" {
		ui.Text(c, s.status).FontSize(12).TextColor(muted).MaxLines(2)
	}
	ui.PrimaryButton(c, "保存设置").FillWidth().Height(40).FontSize(14).FontWeight(600).OnClick(func() { s.save(); s.closeModal() })
}
