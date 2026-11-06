package main

import (
	"fmt"
	"math"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo/ui"
	"github.com/guohuiyuan/music-lib/model"
)

var accent = ui.Hex("#10b981")
var white = ui.Hex("#ffffff")
var textColor = ui.Hex("#2d3748")
var muted = ui.Hex("#718096")
var border = ui.Hex("#e2e8f0")

func (s *app) view(c *ui.Context) {
	initAssets()
	theme := ui.LightTheme()
	theme.Accent = accent
	theme.Text = textColor
	theme.TextMuted = muted
	theme.Border = border
	theme.FontSize = 16
	theme.Font = "Segoe UI"
	c.SetTheme(theme)
	w, h := c.Size()
	ui.Box(c).Fill().Background(ui.Hex("#eef2f5")).Draw(func(p *ui.Painter, r ui.Rect) {
		r.H = 600
		for _, g := range []struct {
			x     float32
			color string
		}{{1, "#722743"}, {.5, "#2f3d6a"}, {0, "#111015"}} {
			radius := float32(math.Hypot(float64(max(g.x, 1-g.x)*r.W), float64(r.H))) * .5
			p.FillRadialGradient(r, ui.RadialGradient{Center: ui.Point{X: g.x, Y: .000001}, Radius: radius, Stops: []ui.GradientStop{{Color: ui.Hex(g.color), At: 0}, {Color: ui.Transparent, At: 1}}}, 0)
		}
	}).Children(func() {
		ui.Scroll(c.Key("page-scroll")).Fill().TrackScroll(&s.scroll).Children(func() {
			ui.Column(c).Width(min(900, w-80)).AlignSelf(ui.Center).Padding(40, 0, 150, 0).Children(func() {
				s.searchBox(c)
				if len(s.categories) > 0 {
					s.categoriesView(c)
				} else if len(s.collections) > 0 {
					s.collectionsView(c)
				} else {
					s.resultsView(c)
				}
			})
		})
		s.toolbar(c, h)
		if s.cfg.ShowMascot && w > 1100 && mascot != nil {
			ui.Image(c, mascot).Size(150, 150).Absolute().Right(70).Bottom(130).Fit(ui.Contain)
		}
		s.playerView(c)
		if s.lyricOpen {
			s.lyricsView(c)
		}
		if s.modal != "" {
			s.modalView(c, w, h)
		}
	})
}
func (s *app) searchBox(c *ui.Context) {
	ui.Column(c.Key("search-box")).Background(white.Alpha(.95)).Border(1, white.Alpha(.5)).Radius(20).Padding(30).Shadow(0, 4, 6, 0, ui.RGBA(0, 0, 0, .05)).Margin(0, 0, 30, 0).Children(func() {
		ui.Row(c).Justify(ui.Center).Gap(10).Margin(0, 0, 25, 0).AlignItems(ui.Center).Children(func() {
			glyph(c, "music", 35, accent)
			ui.Text(c, "music-dl").FontSize(35.2).FontWeight(800).TextColor(accent)
		})
		ui.Row(c).Justify(ui.Center).Gap(20).Margin(0, 0, 20, 0).Children(func() {
			for _, o := range []struct{ id, label string }{{"song", "单曲搜索"}, {"playlist", "歌单搜索"}, {"album", "专辑搜索"}} {
				color := muted
				if s.searchKind == o.id {
					color = accent
				}
				ui.Radio(c.Key(o.id), &s.searchKind, o.id, o.label).FontSize(16).FontWeight(600).TextColor(color)
			}
		})
		ui.Row(c).Gap(10).Margin(0, 0, 25, 0).Children(func() {
			if ui.TextInput(c.Key("query"), &s.query).Label("搜索关键词").Placeholder("输入关键词或粘贴链接，支持单曲 / 歌单 / 专辑").Grow(1).Height(50).Padding(14, 24).Radius(50).Border(2, border).Background(ui.Hex("#f8fafc")).FontSize(16).Submitted() {
				s.search()
			}
			ui.Button(c, " ").Label("搜索").Width(80).Height(50).Radius(50).Border(0, ui.Transparent).Gradient(ui.Hex("#34d399"), ui.Hex("#059669"), 135).Shadow(0, 4, 12, 0, accent.Alpha(.3)).Disabled(s.searching).OnClick(s.search).Children(func() { glyph(c, "search", 18, white) })
		})
		ui.Row(c).Wrap().Justify(ui.Center).Gap(15).Margin(0, 0, 20, 0).Children(func() {
			pill(c, "fire", "每日推荐歌单", "#ff9a9e", "#fecfef", func() { s.browse("recommend", "netease", "") })
			pill(c, "folder", "本地歌单", "#f68084", "#f68084", func() { s.modal = "collections" })
			pill(c, "music", "本地音乐", "#f68084", "#f68084", s.localMusic)
			pill(c, "grid", "歌单分类", "#38bdf8", "#6366f1", func() { s.browse("categories", "netease", "") })
			pill(c, "rank", "排行榜", "#fb923c", "#ef4444", func() { s.browse("ranks", "netease", "") })
			pill(c, "heart", "我的歌单", "#34d399", "#06b6d4", func() { s.browse("mine", "netease", "") })
		})
		ui.Row(c).AlignItems(ui.Center).Padding(0, 0, 8, 0).Margin(0, 0, 12, 0).BorderWidth(0, 0, 1, 0).BorderColor(ui.Hex("#edf2f7")).Children(func() {
			ui.Button(c, "").Padding(0).Background(ui.Transparent).Border(0, ui.Transparent).Label("搜索源设置").OnClick(func() { s.sourceCollapsed = !s.sourceCollapsed }).Children(func() { ui.Text(c, "搜索源设置  ⌃").FontSize(14.4).Bold().TextColor(muted) })
			ui.Spacer(c)
			ui.Button(c, "全选").FontSize(12).TextColor(muted).Background(ui.Transparent).Border(0, ui.Transparent).OnClick(func() {
				for _, src := range []string{"netease", "bilibili", "local"} {
					s.selected[src] = true
				}
			})
			ui.Button(c, "清空").FontSize(12).TextColor(muted).Background(ui.Transparent).Border(0, ui.Transparent).OnClick(func() { s.selected = map[string]bool{} })
		})
		if !s.sourceCollapsed {
			ui.Grid(c).Columns(5).Gap(12).Children(func() {
				for _, src := range []struct{ id, label string }{{"netease", "网易云音乐"}, {"bilibili", "Bilibili"}, {"local", "本地音乐"}} {
					active := s.selected[src.id]
					bg, fg, bd := white, ui.Hex("#4a5568"), border
					if active {
						bg = ui.Hex("#eff3ff")
						fg = accent
						bd = accent
					}
					ui.Button(c.Key("source-"+src.id), "").Label(src.label).Padding(10, 8).Height(58).Radius(12).Background(bg).Border(2, bd).OnClick(func() { s.selected[src.id] = !s.selected[src.id] }).Children(func() {
						ui.Column(c).AlignItems(ui.Center).Gap(3).Children(func() {
							ui.Text(c, src.id).FontSize(14).Bold().TextColor(fg)
							ui.Text(c, src.label).FontSize(11).TextColor(ui.Hex("#a0aec0"))
						})
					})
				}
			})
		}
		if s.status != "" {
			ui.Text(c, s.status).FontSize(13).TextColor(muted).Margin(14, 0, 0, 0).MaxLines(3)
		}
	})
}
func (s *app) resultsView(c *ui.Context) {
	tracks := s.shownTracks()
	if len(tracks) == 0 {
		if s.searching {
			ui.Text(c, "正在查找…").TextColor(muted).AlignSelf(ui.Center)
		}
		return
	}
	size := s.cfg.PageSize
	if size < 20 {
		size = 30
	}
	pages := (len(tracks) + size - 1) / size
	s.pageNumber = max(1, min(pages, s.pageNumber))
	start := (s.pageNumber - 1) * size
	end := min(start+size, len(tracks))
	ui.Row(c).Background(white.Alpha(.9)).Radius(12).Padding(12).Margin(0, 0, 16, 0).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Gap(6).Children(func() {
			title := fmt.Sprintf("♫ 找到  %d  首歌曲", len(tracks))
			if s.collectionTitle != "" {
				title = fmt.Sprintf("♫ %s（共 %d 首）", s.collectionTitle, len(tracks))
			}
			switch s.page {
			case "favorites":
				title = "♥ 我的收藏"
			case "history":
				title = "◷ 播放历史"
			case "queue":
				title = "♫ 播放队列"
			}
			ui.Text(c, title).FontSize(16).FontWeight(600)
			ui.Text(c, fmt.Sprintf("当前第 %d / %d 页，显示 %d - %d / %d", s.pageNumber, pages, start+1, end, len(tracks))).FontSize(12).TextColor(muted)
		})
		ui.Spacer(c)
		if s.source == "local" && s.page == "library" {
			ui.Button(c, "选择文件夹").OnClick(s.pickFolder)
		}
		if s.collectionTitle != "" {
			ui.Button(c, "下载整个歌单").Radius(18).TextColor(accent).FontSize(12).OnClick(func() { s.addDownloads(tracks) })
			ui.Button(c, "导入本地").Radius(18).TextColor(accent).FontSize(12).OnClick(func() { s.pendingTracks = tracks; s.playlistName = s.collectionTitle; s.modal = "collections" })
		}
		ui.Button(c, "列表工具").Radius(18).FontSize(12).TextColor(accent).OnClick(func() { s.modal = "tools" })
	})
	if s.batch {
		ui.Row(c).Padding(12).Background(white).Radius(12).Gap(8).Margin(0, 0, 15, 0).AlignItems(ui.Center).Children(func() {
			ui.Button(c, "全选").OnClick(func() {
				for _, t := range tracks {
					s.checked[t.Key()] = true
				}
			})
			ui.Button(c, "清空").OnClick(func() { s.checked = map[string]bool{} })
			ui.Text(c, fmt.Sprintf("已选 %d", len(s.selectedTracks()))).FontSize(12).TextColor(muted)
			ui.Spacer(c)
			ui.Button(c, "批量收藏").OnClick(func() { s.pendingTracks = s.selectedTracks(); s.modal = "collections" })
			ui.Button(c, "批量下载").OnClick(func() { s.addDownloads(s.selectedTracks()) })
		})
	}
	ui.Column(c).Gap(15).Children(func() {
		for i := start; i < end; i++ {
			s.trackCard(c, tracks[i], i, tracks)
		}
	})
	if pages > 1 {
		ui.Row(c).Justify(ui.Center).Gap(12).Margin(20, 0, 0, 0).Children(func() {
			ui.Button(c, "上一页").Disabled(s.pageNumber == 1).OnClick(func() { s.pageNumber--; s.scroll.ScrollTo(0, 350, ui.Smooth) })
			ui.Text(c, fmt.Sprintf("%d / %d", s.pageNumber, pages))
			ui.Button(c, "下一页").Disabled(s.pageNumber == pages).OnClick(func() { s.pageNumber++; s.scroll.ScrollTo(0, 350, ui.Smooth) })
		})
	}
}
func (s *app) trackCard(c *ui.Context, t library.Track, i int, tracks []library.Track) {
	bg, bd := white, ui.Transparent
	if s.current != nil && s.current.Key() == t.Key() {
		bg = ui.Hex("#eff3ff")
		bd = accent
	}
	ui.Row(c.Key(t.Key())).Padding(15).Gap(15).AlignItems(ui.Center).Background(bg).Border(2, bd).Radius(16).Shadow(0, 4, 6, 0, ui.RGBA(0, 0, 0, .05)).Children(func() {
		if s.batch {
			selected := s.checked[t.Key()]
			if ui.Checkbox(c, &selected, "").Label("选择 " + t.Name).Changed() {
				if selected {
					s.checked[t.Key()] = true
				} else {
					delete(s.checked, t.Key())
				}
			}
		}
		s.trackCover(c, t, 64)
		ui.Column(c).Grow(1).Gap(6).Children(func() {
			ui.Text(c, t.Name).FontSize(16).FontWeight(600).SingleLine().Ellipsis("…")
			ui.Text(c, t.Artist+" · "+t.Album).FontSize(13).TextColor(muted).SingleLine().Ellipsis("…")
			ui.Row(c).Gap(6).Children(func() {
				for _, tag := range []string{t.Source, t.FormatDuration(), t.FormatSize(), t.FormatBitrate()} {
					if tag != "" {
						ui.Text(c, tag).FontSize(10).FontWeight(600).Padding(3, 7).Radius(4).Background(ui.Hex("#edf2f7")).TextColor(muted)
					}
				}
			})
		})
		ui.Button(c.Key("play"), "").Size(36, 36).Radius(18).Padding(0).Label("播放 "+t.Name).Gradient(ui.Hex("#34d399"), ui.Hex("#059669"), 135).Border(0, ui.Transparent).OnClick(func() { s.play(i, tracks) }).Children(func() { glyph(c, "play", 14, white) })
		iconButton(c, "switch", "repeat", "换源 "+t.Name, 36).Background(ui.Hex("#fff7ed")).Border(1, ui.Hex("#fed7aa")).OnClick(func() {
			s.query = t.Name + " " + t.Artist
			s.selected = map[string]bool{"netease": true, "bilibili": true}
			s.searchKind = "song"
			s.search()
		})
		heart := iconButton(c, "favorite", "heart", "收藏 "+t.Name, 36).Background(ui.Hex("#fff5f5")).Border(1, ui.Hex("#fed7d7")).OnClick(func() { s.pendingTracks = []library.Track{t}; s.modal = "collections" })
		if s.isFavorite(t) {
			heart.Background(ui.Hex("#ffe4e6"))
		}
		iconButton(c, "download", "download", "下载 "+t.Name, 36).Border(1, border).OnClick(func() { s.addDownloads([]library.Track{t}) })
		iconButton(c, "lyrics", "file", "歌词 "+t.Name, 36).Border(1, border).OnClick(func() { s.saveLyrics(t) })
		iconButton(c, "details", "image", "歌曲详情 "+t.Name, 36).Border(1, border).OnClick(func() {
			if s.current == nil || s.current.Key() != t.Key() {
				s.play(i, tracks)
			}
			s.lyricOpen = true
		})
	})
}
func (s *app) trackCover(c *ui.Context, t library.Track, size float32) {
	if img := s.artwork(t); img != nil {
		ui.Image(c, img).Size(size, size).Fit(ui.Cover).Radius(8)
	} else {
		ui.Box(c).Size(size, size).Background(ui.Hex("#f7fafc")).Radius(8).Center().Children(func() { glyph(c, "music", size*.35, ui.Hex("#cbd5e0")) })
	}
}
func (s *app) collectionsView(c *ui.Context) {
	ui.Grid(c).Columns(4).Gap(20).Children(func() {
		for _, p := range s.collections {
			ui.Column(c.Key(p.Source+":"+p.ID)).Background(white).Radius(16).Clip().Shadow(0, 4, 6, 0, ui.RGBA(0, 0, 0, .05)).OnClick(func() { s.openCollection(p) }).Children(func() {
				s.trackCover(c, library.Track{Song: model.Song{Source: p.Source, ID: p.ID, Cover: p.Cover}}, 205)
				ui.Column(c).Padding(12).Gap(6).Children(func() {
					ui.Text(c, p.Name).FontSize(14).Bold().MaxLines(2)
					ui.Text(c, p.Creator).FontSize(12).TextColor(muted).SingleLine().Ellipsis("…")
					ui.Text(c, fmt.Sprintf("%d 首歌曲", p.TrackCount)).FontSize(11).TextColor(muted)
				})
			})
		}
	})
}
func (s *app) categoriesView(c *ui.Context) {
	ui.Column(c).Padding(20).Radius(16).Background(white).Gap(16).Children(func() {
		ui.Text(c, "歌单分类").Bold()
		ui.Row(c).Wrap().Gap(10).Children(func() {
			for _, cat := range s.categories {
				ui.Button(c.Key(cat.ID), cat.Name).Radius(20).FontSize(13).OnClick(func() { s.browse("category", "netease", cat.ID) })
			}
		})
	})
}
func (s *app) toolbar(c *ui.Context, h float32) {
	ui.Column(c).Absolute().Right(14).Top(max(20, (h-320)/2)).Gap(6).AlignItems(ui.Center).Children(func() {
		for _, b := range []struct {
			key, icon, label string
			fn               func()
		}{{"settings", "gear", "系统设置", func() { s.modal = "settings" }}, {"local", "folder", "本地音乐", s.localMusic}, {"downloads", "history", "下载管理", func() { s.modal = "downloads" }}, {"history", "clock", "播放历史", func() { s.page = "history"; s.collections = nil; s.categories = nil; s.pageNumber = 1 }}, {"mini", "mini", "悬浮窗", s.toggleMini}} {
			iconButton(c, b.key, b.icon, b.label, 36).Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, .12)).OnClick(b.fn)
		}
		ui.Divider(c).Width(20).MarginY(4)
		iconButton(c, "top", "up", "回到顶部", 36).OnClick(func() { s.scroll.ScrollTo(0, 0, ui.Smooth) })
		iconButton(c, "current", "headphones", "定位当前播放歌曲", 36).OnClick(func() {
			s.page = "queue"
			s.collections = nil
			s.categories = nil
			s.pageNumber = 1
			s.scroll.ScrollTo(0, 350, ui.Smooth)
		})
	})
}
func (s *app) playerView(c *ui.Context) {
	if !s.playerExpanded {
		ui.Row(c).Absolute().Left(0).Bottom(0).Children(func() {
			ui.Button(c, "").Size(66, 66).Radius(0).Background(accent).Border(0, ui.Transparent).Label("播放或暂停").OnClick(s.toggle).Children(func() {
				name := "play"
				if s.playing {
					name = "pause"
				}
				glyph(c, name, 24, white)
			})
			ui.Button(c, "›").Width(18).Height(66).Radius(0).Label("展开播放器").OnClick(func() { s.playerExpanded = true })
		})
		return
	}
	ui.Column(c.Key("player")).Absolute().Left(0).Bottom(0).Width(400).Background(white).Shadow(0, 0, 10, 0, ui.RGBA(0, 0, 0, .15)).Children(func() {
		ui.Row(c).Height(66).Gap(8).Children(func() {
			ui.Box(c).Size(66, 66).Background(accent).OnClick(s.toggle).Children(func() {
				if s.cover != nil {
					ui.Image(c, s.cover).Fill().Fit(ui.Cover)
				}
				ui.Box(c).Absolute().Fill().Center().Children(func() {
					name := "play"
					if s.playing {
						name = "pause"
					}
					glyph(c, name, 22, white)
				})
			})
			ui.Column(c).Grow(1).Padding(7, 0).Gap(8).Children(func() {
				ui.Row(c).Gap(5).Children(func() {
					ui.Text(c, caption(s.current)).FontSize(12).SingleLine().Ellipsis("…").Grow(1)
					ui.Button(c, "‹").Label("收起播放器").Padding(0).Width(18).Height(18).OnClick(func() { s.playerExpanded = false })
				})
				pos := s.position.Seconds()
				if ui.Slider(c.Key("seek"), &pos, 0, max(1, s.duration.Seconds())).Height(10).Label("播放进度").Disabled(s.player == nil || s.duration <= 0).Changed() {
					s.seek(time.Duration(pos * float64(time.Second)))
				}
				ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
					ui.Text(c, clockText(s.position)+" / "+clockText(s.duration)).FontSize(10).TextColor(muted)
					iconButton(c, "prev", "prev", "上一首", 18).OnClick(func() { s.step(-1) })
					iconButton(c, "next", "next", "下一首", 18).OnClick(func() { s.step(1) })
					v := s.cfg.Volume
					glyph(c, "volume", 12, muted)
					if ui.Slider(c.Key("volume"), &v, 0, 1).Width(45).Height(14).Label("音量").Changed() {
						s.volume(v)
					}
					iconButton(c, "mode", "repeat", "切换播放模式", 18).OnClick(s.cycleMode)
					iconButton(c, "queue", "list", "播放队列", 18).OnClick(func() { s.page = "queue"; s.collections = nil; s.categories = nil; s.pageNumber = 1 })
					iconButton(c, "lyric", "file", "歌曲详情", 18).OnClick(func() { s.lyricOpen = true })
				})
			})
		})
	})
}
func (s *app) cycleMode() {
	modes := []string{"", "loop", "one", "shuffle"}
	for i, m := range modes {
		if m == s.cfg.PlayMode {
			s.cfg.PlayMode = modes[(i+1)%len(modes)]
			break
		}
	}
	labels := map[string]string{"": "顺序播放", "loop": "列表循环", "one": "单曲循环", "shuffle": "随机播放"}
	s.status = labels[s.cfg.PlayMode]
	s.save()
}
