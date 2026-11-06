package main

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (s *app) modalView(c *ui.Context, w, h float32) {
	titles := map[string]string{"settings": "系统设置", "collections": "本地歌单", "login": "扫码登录", "downloads": "下载管理", "tools": "列表工具", "about": "关于 music-dl"}
	width, height := float32(640), min(h-40, 600)
	if s.modal == "settings" {
		width = 760
		height = min(h-40, 720)
	}
	if s.modal == "login" {
		width = 420
		height = 430
	}
	back := ui.Box(c.Key("modal-backdrop")).Absolute().Fill().Background(ui.RGBA(0, 0, 0, .5)).Center().Modal()
	back.Children(func() {
		ui.Column(c).Size(min(w-40, width), height).Background(white).Radius(18).Border(1, ui.Hex("#dcebe4")).Shadow(0, 20, 40, 0, ui.RGBA(0, 0, 0, .2)).Clip().Children(func() {
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
				case "tools":
					s.toolsView(c)
				case "about":
					s.aboutView(c)
				}
			})
		})
	})
	if back.OverlayShortcut(0, ui.KeyEscape) {
		s.closeModal()
	}
}
func setting(c *ui.Context, title string, body func()) {
	ui.Column(c).Padding(12).Border(1, ui.Hex("#e2e8f0")).Radius(10).Background(ui.Hex("#f8fafc")).Gap(10).Children(func() { ui.Text(c, title).FontSize(13).FontWeight(600).TextColor(muted); body() })
}
func (s *app) settingsView(c *ui.Context) {
	ui.Row(c).Padding(4).Gap(5).Background(ui.Hex("#f8fafc")).Border(1, border).Radius(12).Children(func() {
		for _, tab := range []struct{ id, label string }{{"general", "常规"}, {"download", "下载与同步"}, {"cookies", "平台 Cookie"}} {
			bg := ui.Transparent
			if s.settingsTab == tab.id {
				bg = white
			}
			ui.Button(c, tab.label).Grow(1).FontSize(13).FontWeight(700).Background(bg).Border(0, ui.Transparent).Radius(8).Padding(9, 12).OnClick(func() { s.settingsTab = tab.id })
		}
	})
	ui.Scroll(c.Key(s.settingsTab)).Grow(1).Children(func() {
		ui.Column(c).Padding(2, 4).Gap(14).Children(func() {
			switch s.settingsTab {
			case "general":
				setting(c, "悬浮歌词", func() { ui.Button(c, "显示 / 隐藏悬浮窗").OnClick(s.toggleMini) })
				setting(c, "歌词字体", func() {
					size := float64(s.cfg.LyricSize)
					if size == 0 {
						size = 24
					}
					if ui.Slider(c, &size, 14, 48).Label("歌词字号").Changed() {
						s.cfg.LyricSize = float32(size)
					}
					ui.Text(c, fmt.Sprintf("字号 %.0f px", size)).FontSize(12).TextColor(muted)
					ui.Text(c, "示例歌词 · 让每一句都清晰可见").FontSize(float32(size)).TextColor(accent)
				})
				setting(c, "音乐文件夹", func() {
					ui.Row(c).Gap(8).Children(func() {
						ui.TextInput(c, &s.cfg.MusicDir).Label("音乐文件夹").Grow(1).FontSize(14)
						ui.Button(c, "浏览文件夹").OnClick(s.pickFolder)
					})
				})
				setting(c, "界面", func() {
					ui.Checkbox(c, &s.cfg.ShowMascot, "显示桌面挂件")
					ui.Text(c, "关闭主窗口后继续播放，可从托盘或悬浮窗恢复。").FontSize(12).TextColor(muted)
				})
				setting(c, "关于", func() { ui.Button(c, "关于 music-dl").OnClick(func() { s.modal = "about" }) })
			case "download":
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
	ui.Row(c).Gap(8).Children(func() {
		ui.Text(c, s.status).FontSize(12).TextColor(muted).Grow(1).MaxLines(2)
		ui.PrimaryButton(c, "保存设置").OnClick(func() { s.save(); s.closeModal() })
	})
}
func (s *app) savedPlaylistsView(c *ui.Context) {
	ui.Row(c).Gap(8).Children(func() {
		ui.TextInput(c, &s.playlistName).Label("歌单名称").Placeholder("新歌单名称").Grow(1)
		ui.PrimaryButton(c, "新建歌单").OnClick(func() { s.savePlaylist(s.playlistName, s.pendingTracks) })
	})
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Gap(10).Children(func() {
			ui.Row(c).Padding(12).Background(ui.Hex("#fff5f5")).Radius(10).Gap(10).Children(func() {
				glyph(c, "heart", 16, ui.Hex("#f68084"))
				ui.Text(c, "我的收藏").Grow(1)
				ui.Button(c, "打开").Label("打开我的收藏").OnClick(func() {
					s.page = "favorites"
					s.collections = nil
					s.categories = nil
					s.pageNumber = 1
					s.closeModal()
				})
				if len(s.pendingTracks) > 0 {
					ui.Button(c, "加入收藏").OnClick(func() {
						for _, t := range s.pendingTracks {
							if !s.isFavorite(t) {
								s.favorite(t)
							}
						}
						s.closeModal()
					})
				}
			})
			for _, p := range s.cfg.Playlists {
				ui.Row(c.Key(p.ID)).Padding(12).Border(1, border).Radius(10).Gap(8).Children(func() {
					ui.Column(c).Grow(1).Gap(4).Children(func() {
						ui.Text(c, p.Name).Bold()
						ui.Text(c, fmt.Sprintf("%d 首歌曲", len(p.Tracks))).FontSize(12).TextColor(muted)
					})
					if len(s.pendingTracks) > 0 {
						ui.Button(c, "加入").OnClick(func() { s.addToPlaylist(p.ID, s.pendingTracks) })
					}
					ui.Button(c, "打开").OnClick(func() {
						s.page = "collection"
						s.localPlaylistID = p.ID
						s.collectionTitle = p.Name
						s.collections = nil
						s.categories = nil
						s.pageNumber = 1
						s.closeModal()
					})
					iconButton(c, "delete", "trash", "删除歌单 "+p.Name, 28).OnClick(func() { s.confirmDeletePlaylist(p.ID, p.Name) })
				})
			}
		})
	})
	ui.Row(c).Gap(8).Children(func() {
		ui.Button(c, "导入歌单").OnClick(s.importPlaylists)
		ui.Button(c, "导出歌单").OnClick(s.exportPlaylists)
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
func (s *app) downloadsView(c *ui.Context) {
	labels := map[string]string{"queued": "等待下载", "downloading": "正在下载", "done": "下载完成", "failed": "下载失败", "canceled": "已取消"}
	ui.Row(c).Gap(8).Children(func() {
		ui.Button(c, "打开下载目录").OnClick(s.openDownloadDir)
		ui.Button(c, "清理已结束记录").OnClick(func() {
			if s.downloads != nil {
				s.downloads.Clear()
				s.jobs = s.downloads.Snapshot()
			}
		})
	})
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Gap(10).Children(func() {
			for _, j := range s.jobs {
				ui.Column(c.Key(j.Track.Key())).Padding(12).Border(1, border).Radius(10).Gap(8).Children(func() {
					ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Column(c).Grow(1).Gap(4).Children(func() {
							ui.Text(c, j.Track.Name).FontSize(14).Bold().SingleLine().Ellipsis("…")
							ui.Text(c, labels[j.State]+"  "+j.Error).FontSize(12).TextColor(muted).MaxLines(2)
						})
						if j.State == "queued" || j.State == "downloading" {
							ui.Button(c, "取消").OnClick(func() { s.downloads.Cancel(j.Track.Key()) })
						} else if j.State == "failed" || j.State == "canceled" {
							ui.Button(c, "重试").OnClick(func() { s.addDownloads([]library.Track{j.Track}) })
						} else if j.State == "done" {
							ui.Button(c, "打开").OnClick(func() { mygo.Shell.ShowItemInFolder(j.Path) })
						}
					})
					if j.State == "downloading" {
						progress := float64(0)
						if j.Total > 0 {
							progress = float64(j.Done) / float64(j.Total)
						}
						ui.Box(c).Height(4).Background(border).Radius(2).Children(func() { ui.Box(c).WidthPercent(float32(progress * 100)).Height(4).Background(accent).Radius(2) })
						ui.Text(c, fmt.Sprintf("%.1f / %.1f MB", float64(j.Done)/(1<<20), float64(max(0, j.Total))/(1<<20))).FontSize(11).TextColor(muted)
					}
				})
			}
			if len(s.jobs) == 0 {
				ui.Text(c, "暂无下载记录").TextColor(muted).Padding(30)
			}
		})
	})
}
func (s *app) toolsView(c *ui.Context) {
	setting(c, "每页显示", func() {
		size := strconv.Itoa(s.cfg.PageSize)
		if ui.Select(c, &size, []string{"20", "30", "50", "100", "200", "300", "400", "500"}).Changed() {
			s.cfg.PageSize, _ = strconv.Atoi(size)
			s.pageNumber = 1
			s.save()
		}
	})
	setting(c, "歌曲排序", func() {
		ui.Row(c).Gap(8).Children(func() {
			for _, o := range []struct{ id, label string }{{"", "默认"}, {"quality", "音质"}, {"size", "大小"}, {"duration", "时长"}, {"name", "歌名"}} {
				ui.Radio(c, &s.sortMode, o.id, o.label)
			}
			ui.Checkbox(c, &s.sortReverse, "降序")
		})
	})
	ui.Button(c, "播放全部").OnClick(func() { s.play(0, s.shownTracks()); s.closeModal() })
	ui.Button(c, "批量操作").OnClick(func() { s.batch = !s.batch; s.closeModal() })
	ui.Button(c, "下载全部").OnClick(func() { s.addDownloads(s.shownTracks()); s.closeModal() })
	if s.page == "collection" {
		ui.Button(c, "移除已选歌曲").OnClick(func() {
			for i := range s.cfg.Playlists {
				p := &s.cfg.Playlists[i]
				if p.ID == s.localPlaylistID {
					p.Tracks = slices.DeleteFunc(p.Tracks, func(t library.Track) bool { return s.checked[t.Key()] })
				}
			}
			s.save()
			s.closeModal()
		})
	}
	if s.page == "history" {
		ui.Button(c, "清空播放历史").OnClick(func() { s.cfg.History = nil; s.save(); s.closeModal() })
	}
}
func (s *app) aboutView(c *ui.Context) {
	ui.Text(c, "music-dl").FontSize(28).Bold().TextColor(accent)
	ui.Text(c, "版本 "+mygo.App.Version()).FontSize(14).TextColor(muted)
	ui.Text(c, "网易云音乐 · Bilibili · 本地音乐").FontSize(14)
	ui.Button(c, "检查更新").Disabled(s.updateBusy).OnClick(s.checkUpdate)
	ui.Text(c, s.status).FontSize(13).TextColor(muted).MaxLines(5)
	if s.updateBusy {
		ui.Box(c).Height(5).Background(border).Radius(3).Children(func() { ui.Box(c).WidthPercent(float32(s.updateProgress * 100)).Height(5).Background(accent).Radius(3) })
	}
}
