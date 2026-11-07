package main

import (
	"fmt"
	"github.com/egoist/mygo/ui"
)

func browsePanel(c *ui.Context, icon, title, subtitle string, count int, body func()) {
	ui.Column(c).Padding(24).Margin(0, 0, 24, 0).Radius(20).Background(white.Alpha(.95)).Border(1, white.Alpha(.5)).Shadow(0, 4, 6, 0, ui.RGBA(0, 0, 0, .05)).Children(func() {
		ui.Column(c).Margin(0, 0, 16, 0).Children(func() {
			ui.Row(c).Gap(8).AlignSelf(ui.Start).AlignItems(ui.Center).Children(func() {
				glyph(c, icon, 22.4, accent)
				ui.Text(c, title).FontSize(22.4).FontWeight(800).LetterSpacing(-.5).TextColor(accent)
			})
			ui.Text(c, subtitle).FontSize(13).TextColor(muted).Margin(6, 0, 0, 0)
		})
		ui.Row(c).Padding(6).Margin(0, 0, 18, 0).Radius(14).Background(ui.Hex("#f1f5f9")).Children(func() {
			ui.Row(c).Padding(8, 14).Gap(6).Radius(10).Background(white).Shadow(0, 2, 6, 0, accent.Alpha(.15)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "网易云音乐").FontSize(14).Bold().TextColor(accent)
				ui.Text(c, fmt.Sprint(count)).FontSize(11).Bold().Padding(1, 7).MinWidth(18).Radius(20).Background(ui.Hex("#d1fae5")).TextColor(ui.Hex("#059669"))
			})
		})
		body()
	})
}
func (s *app) browseEmpty(c *ui.Context) bool {
	if s.searching || s.status != "" {
		text := s.status
		if text == "" {
			text = "正在加载…"
		}
		ui.Text(c, text).FontSize(13).TextColor(muted).TextAlign(ui.Center).Padding(24, 16).Radius(14).Border(1, border).Background(ui.Hex("#f8fafc"))
		return true
	}
	return false
}
func (s *app) sourcePlaylistsView(c *ui.Context) {
	title, icon, subtitle := "每日推荐歌单", "fire", "按渠道查看各平台为你推荐的歌单。"
	if s.collectionKind == "mine" {
		title, icon, subtitle = "我收藏的歌单", "heart", "查看已登录平台中你创建和收藏的歌单，未登录的渠道请先在设置中扫码登录。"
	}
	width, _ := c.Size()
	browsePanel(c, icon, title, subtitle, len(s.collections), func() {
		if s.browseEmpty(c) {
			return
		}
		s.collectionCards(c, min(900, width-40)-50, 16, 170, true)
	})
}
func (s *app) collectionCards(c *ui.Context, width, gap, minWidth float32, compact bool) {
	cols := max(1, int((width+gap)/(minWidth+gap)))
	size := (width - float32(cols-1)*gap) / float32(cols)
	ui.Grid(c).Columns(cols).Gap(gap).Children(func() {
		for _, p := range s.collections {
			card := ui.Column(c.Key(p.Source+":"+p.ID)).Background(white).Radius(16).Clip().Shadow(0, 4, 6, 0, ui.RGBA(0, 0, 0, .05))
			cardMotion(card, 5)
			card.Children(func() {
				ui.Box(c).OnClick(func() { s.openCollection(p) }).Children(func() {
					s.playlistCoverView(c, p.Name, p.Cover, p.ID, size)
					ui.Text(c, p.Source).Absolute().Right(5).Top(5).FontSize(10).TextColor(accent).Background(ui.Hex("#e6fffa")).Radius(4).Padding(2, 8)
				})
				ui.Column(c).Padding(12).Gap(6).Children(func() {
					fontSize := float32(14)
					if compact {
						fontSize = 13
					}
					ui.Text(c, p.Name).FontSize(fontSize).Bold().MaxLines(2).OnClick(func() { s.openCollection(p) })
					if p.Creator != "" {
						ui.Text(c, p.Creator).FontSize(12).TextColor(muted).SingleLine().Ellipsis("…")
					}
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						count := fmt.Sprintf("共 %d 首", p.TrackCount)
						if p.TrackCount == 0 {
							count = "进入查看"
						}
						ui.Text(c, count).FontSize(11).TextColor(ui.Hex("#a0aec0")).Grow(1)
						ui.Button(c, "导入本地").FontSize(12).FontWeight(600).TextColor(accent).Radius(30).Padding(6, 12).Background(ui.Hex("#f7f9ff")).Border(1, ui.Hex("#dbe3ff")).Disabled(s.importBusy).OnClick(func() { s.importRemoteCollection(p) })
					})
					if !compact {
						ui.Button(c, "下载整个歌单").FillWidth().Height(32).FontSize(12).Radius(20).TextColor(accent).Border(1, ui.Hex("#b7ebd6")).Background(ui.Hex("#f0fff9")).OnClick(func() { s.downloadRemoteCollection(p) })
					}
				})
			})
		}
	})
}
func (s *app) rankingsView(c *ui.Context) {
	width, _ := c.Size()
	width = min(900, width-40) - 50
	cols := max(1, int((width+16)/186))
	size := (width - float32(cols-1)*16) / float32(cols)
	browsePanel(c, "ranking", "排行榜", "选择音乐源与榜单，查看热门、飙升等榜单歌曲。", len(s.collections), func() {
		if s.browseEmpty(c) {
			return
		}
		ui.Grid(c).Columns(cols).Gap(16).Children(func() {
			for _, p := range s.collections {
				card := ui.Column(c.Key(p.ID)).Radius(16).Clip().Background(white).Shadow(0, 4, 12, 0, ui.RGBA(15, 23, 42, .06)).OnClick(func() { s.openCollection(p) })
				cardMotion(card, 4)
				card.Children(func() {
					ui.Box(c).Size(size, size).Children(func() {
						if p.Cover != "" {
							s.playlistCoverView(c, p.Name, p.Cover, p.ID, size)
						} else {
							ui.Column(c).Fill().Padding(18).Gradient(ui.Hex("#ecfdf5"), ui.Hex("#dbeafe"), 145).Gap(10).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
								glyph(c, "ranking", 34, ui.Hex("#064e3b"))
								ui.Text(c, p.Name).FontSize(13).FontWeight(800).TextColor(ui.Hex("#064e3b")).TextAlign(ui.Center).MaxLines(3)
							})
						}
						if frequency := p.Extra["update_frequency"]; frequency != "" {
							ui.Text(c, frequency).Absolute().Left(8).Bottom(8).Padding(3, 8).FontSize(10).Bold().TextColor(white).Background(ui.RGBA(15, 23, 42, .72)).Radius(20)
						}
					})
					ui.Column(c).Padding(10, 12, 12, 12).Gap(6).Children(func() {
						ui.Text(c, p.Name).FontSize(14).Bold().LineHeight(1.35).MaxLines(2)
						ui.Row(c).Children(func() {
							if p.TrackCount > 0 {
								ui.Text(c, fmt.Sprintf("♫ %d 首", p.TrackCount)).FontSize(11).TextColor(ui.Hex("#a0aec0"))
							}
							ui.Spacer(c)
							if p.PlayCount > 0 {
								ui.Text(c, compactCount(p.PlayCount)).FontSize(11).TextColor(ui.Hex("#a0aec0"))
							}
						})
					})
				})
			}
		})
	})
}
func compactCount(n int) string {
	if n >= 100000000 {
		return fmt.Sprintf("%.1f亿", float64(n)/1e8)
	}
	if n >= 10000 {
		return fmt.Sprintf("%.1f万", float64(n)/1e4)
	}
	return fmt.Sprint(n)
}
func (s *app) browseCurrentView(c *ui.Context) {
	if s.page != "library" || s.collectionTitle == "" || s.collectionKind != "rank" {
		return
	}
	label, back, icon, description := "当前榜单", "返回排行榜", "ranking", "点击歌曲即可播放或下载，也可以导入到本地歌单。"
	ui.Row(c).Padding(14, 18).Gap(14).Margin(0, 0, 18, 0).Radius(16).Background(white.Alpha(.95)).Border(1, white.Alpha(.5)).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Children(func() {
			ui.Row(c).Gap(6).Padding(3, 10).AlignSelf(ui.Start).Radius(30).Background(ui.Hex("#ecfdf5")).Border(1, ui.Hex("#bbf7d0")).Children(func() { glyph(c, icon, 12, accent); ui.Text(c, label).FontSize(12).Bold().TextColor(accent) })
			ui.Text(c, "网易云音乐 · "+s.collectionTitle).FontSize(18).FontWeight(800).LetterSpacing(-.2).Margin(8, 0, 0, 0)
			ui.Text(c, description).FontSize(12).TextColor(muted).Margin(4, 0, 0, 0)
		})
		ui.Button(c, "← "+back).Padding(8, 18).Radius(20).FontSize(14).FontWeight(600).Background(ui.Hex("#fff7ed")).TextColor(ui.Hex("#9a3412")).Border(1, ui.Hex("#fed7aa")).OnClick(func() {
			s.browse("ranks", "netease", "")
		})
	})
}
