package main

import (
	"slices"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (s *app) modalView(c *ui.Context, w, h float32) {
	if s.modal == "history" || s.modal == "downloads" {
		s.utilityModal(c, w, h)
		return
	}
	titles := map[string]string{"settings": "系统设置", "collections": "本地歌单", "login": "扫码登录", "downloads": "下载管理", "about": "关于 music-dl"}
	titles["collections"] = "收藏到我的歌单"
	titles["edit-playlist"] = "新建歌单"
	if s.editPlaylistID != "" {
		titles["edit-playlist"] = "编辑歌单"
	}
	titles["import-playlist"] = "导入歌单"
	titles["update"] = "软件更新"
	width, height := float32(640), min(h-40, 600)
	if s.modal == "about" {
		width = 420
		height = min(h-40, 360)
	}
	if s.modal == "update" {
		width = 420
		height = min(h-40, 320)
	}
	if s.modal == "settings" {
		width = 760
		height = min(h-40, 720)
	}
	if s.modal == "login" {
		width = 420
		height = 430
	}
	if s.modal == "edit-playlist" || s.modal == "collections" {
		s.playlistEditorModal(c, w, h)
		return
	}
	if s.modal == "collections" {
		width = 400
		height = min(h-40, 520)
	}
	if s.modal == "import-playlist" {
		width = 440
		height = min(h-40, 280)
	}
	back := ui.Box(c.Key("modal-backdrop")).Absolute().Fill().Background(ui.RGBA(0, 0, 0, .5)).Center().Modal().Transition(ui.ElementTransition{Enter: &ui.Motion{}, Exit: &ui.Motion{}})
	back.Children(func() {
		ui.Column(c).Size(min(w-40, width), height).Background(white).Radius(18).Border(1, ui.Hex("#dcebe4")).Shadow(0, 20, 40, 0, ui.RGBA(0, 0, 0, .2)).Clip().Transition(panelTransition).Children(func() {
			ui.Row(c).Padding(20, 22, 16, 22).BorderWidth(0, 0, 1, 0).BorderColor(ui.Hex("#edf2f7")).AlignItems(ui.Center).Children(func() {
				ui.Column(c).Grow(1).Gap(4).Children(func() {
					ui.Text(c, titles[s.modal]).FontSize(18).Bold()
					if s.modal == "settings" {
						ui.Text(c, "分类管理播放体验、下载同步与平台 Cookie").FontSize(12).TextColor(muted)
					}
				})
				iconButton(c, "close-modal", "close", "关闭弹窗", 28).OnClick(s.closeModal)
			})
			ui.Column(c).Grow(1).Padding(14, 20, 18, 20).Gap(14).Children(func() {
				switch s.modal {
				case "settings":
					s.settingsView(c)
				case "collections":
					s.savedPlaylistsView(c)
				case "login":
					s.loginView(c)
				case "downloads":
					s.downloadsView(c)
				case "about":
					s.aboutView(c)
				case "update":
					s.updateView(c)
				case "edit-playlist":
					s.editPlaylistView(c)
				case "import-playlist":
					s.importLinkView(c)
				}
			})
		})
	})
	if back.OverlayShortcut(0, ui.KeyEscape) {
		if s.captureShortcut != "" {
			s.captureShortcut = ""
		} else {
			s.closeModal()
		}
	}
}
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
func (s *app) savedPlaylistsView(c *ui.Context) {
	ui.Row(c).AlignItems(ui.Center).Margin(0, 0, 15, 0).Children(func() {
		ui.Text(c, "选择或管理歌单").FontSize(14).TextColor(muted).Grow(1)
		ui.Button(c, "+ 新建").Label("新建歌单").TextColor(white).Background(accent).Border(0, ui.Transparent).Radius(20).Padding(6, 12).FontSize(12).OnClick(func() { s.editPlaylist(nil) })
	})
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Gap(10).Children(func() {
			lists := append([]library.Playlist{{ID: "favorites", Name: "我的收藏", Tracks: s.cfg.Favorites}}, s.cfg.Playlists...)
			for _, p := range lists {
				row := ui.Row(c.Key(p.ID)).Padding(10).Border(1, border).Radius(12).Background(ui.Hex("#faf8f8")).AlignItems(ui.Center)
				if row.Hovered() {
					row.Border(1, accent).Background(ui.Hex("#f0fff4"))
				}
				row.Children(func() {
					label := "加入"
					if p.ID == "favorites" {
						label = "加入收藏"
					}
					ui.Row(c).Grow(1).Gap(12).AlignItems(ui.Center).Label(label).OnClick(func() {
						if len(s.pendingTracks) == 0 {
							s.openLocalPlaylist(p)
							s.modal = ""
							return
						}
						if p.ID == "favorites" {
							for _, t := range s.pendingTracks {
								if !s.isFavorite(t) {
									s.favorite(t)
								}
							}
							s.closeModal()
							return
						}
						s.addToPlaylist(p.ID, s.pendingTracks)
					}).Children(func() {
						ui.Box(c).Size(40, 40).Radius(6).Clip().Children(func() { s.playlistCoverView(c, p.Name, p.Cover, p.ID, 40) })
						ui.Text(c, p.Name).FontSize(14).SingleLine().Ellipsis("…").Grow(1)
					})
					if p.ID != "favorites" {
						iconButton(c, "edit", "edit", "编辑歌单 "+p.Name, 28).Radius(6).Border(1, border).Margin(0, 0, 0, 10).OnClick(func() { s.editPlaylist(&p) })
						iconButton(c, "delete", "trash", "删除歌单 "+p.Name, 28).Radius(6).Border(1, border).Margin(0, 0, 0, 6).OnClick(func() { s.confirmDeletePlaylist(p.ID, p.Name) })
					}
				})
			}
		})
	})
}
func (s *app) confirmDeletePlaylist(id, name string) {
	go func() {
		r, e := mygo.Dialog.Message(mygo.MessageOptions{Parent: s.win, Message: "删除歌单「" + name + "」？", Detail: "音乐文件不会被删除。", Buttons: []string{"取消", "删除"}})
		if e == nil && r.Button == 1 {
			s.update(func() {
				s.cfg.Playlists = slices.DeleteFunc(s.cfg.Playlists, func(p library.Playlist) bool { return p.ID == id })
				s.save()
			})
		}
	}()
}
func (s *app) loginView(c *ui.Context) {
	ui.Column(c).Grow(1).AlignItems(ui.Center).Gap(14).Children(func() {
		if s.loginImage != nil {
			ui.Image(c, s.loginImage).Size(240, 240)
		}
		ui.Text(c, s.loginStatus).FontSize(14).MaxLines(2)
		ui.Button(c, "重新获取二维码").OnClick(func() { s.startLogin(s.loginSource) })
	})
}
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

func (s *app) updateView(c *ui.Context) {
	if s.pendingUpdate == nil {
		return
	}
	ui.Column(c).Gap(6).Children(func() {
		ui.Text(c, "发现新版本 "+s.pendingUpdate.Version).FontSize(22).FontWeight(700).TextColor(accent)
		ui.Text(c, "当前版本 "+mygo.App.Version()).FontSize(13).TextColor(muted)
	})
	ui.Text(c, "下载完成后将自动重启更新。").FontSize(14).TextColor(textColor)
	if s.updateBusy {
		ui.Column(c).Gap(8).Children(func() {
			ui.Text(c, s.updateStatus).FontSize(13).TextColor(muted)
			ui.Box(c).Height(6).Background(border).Radius(3).Clip().Children(func() {
				ui.Box(c).WidthPercent(float32(s.updateProgress * 100)).Height(6).Background(accent).Radius(3)
			})
		})
	} else if s.updateStatus != "发现新版本 "+s.pendingUpdate.Version {
		ui.Text(c, s.updateStatus).FontSize(13).TextColor(muted).MaxLines(2)
	}
	ui.Spacer(c)
	ui.Row(c).Gap(10).Justify(ui.End).Children(func() {
		later := ui.Button(c, "稍后").Label("稍后").Height(40).Padding(8, 18).Radius(8).FontSize(14).Disabled(s.updateBusy).OnClick(s.closeModal)
		label := "下载并重启"
		if s.updateBusy {
			label = "正在下载…"
		}
		download := ui.PrimaryButton(c, label).Label(label).Height(40).Padding(8, 18).Radius(8).FontSize(14).FontWeight(600).Disabled(s.updateBusy).OnClick(s.installUpdate)
		buttonMotion(later, 1, 1)
		buttonMotion(download, 1, 1)
	})
}

func (s *app) playlistEditorModal(c *ui.Context, w, h float32) {
	height := float32(372)
	title := "新建歌单"
	if s.editPlaylistID != "" {
		title = "编辑歌单"
	}
	if s.modal == "collections" {
		height = min(h*.85, 136+float32(len(s.cfg.Playlists)+1)*72)
		title = "收藏到我的歌单"
	}
	if s.playlistError != "" {
		height += 36
	}
	back := ui.Box(c.Key("modal-backdrop")).Absolute().Fill().Background(ui.RGBA(0, 0, 0, .5)).Center().Modal()
	back.Children(func() {
		ui.Column(c).Width(min(w-40, 450)).Height(min(h*.85, height)).Padding(25).Radius(20).Background(white).Shadow(0, 20, 40, 0, ui.RGBA(0, 0, 0, .2)).Transition(panelTransition).Children(func() {
			ui.Row(c).Height(27).Margin(0, 0, 15, 0).AlignItems(ui.Center).Children(func() {
				ui.Text(c, title).FontSize(18).Bold().Grow(1)
				iconButton(c, "close-editor", "close", "关闭弹窗", 20).Background(ui.Transparent).OnClick(s.closeModal)
			})
			if s.modal == "collections" {
				s.savedPlaylistsView(c)
			} else {
				ui.Scroll(c).Grow(1).Children(func() { ui.Column(c).Children(func() { s.editPlaylistView(c) }) })
			}
		})
	})
	if back.OverlayShortcut(0, ui.KeyEscape) {
		s.closeModal()
	}
}
