package main

import (
	"fmt"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo/ui"
)

func (s *app) playerArtwork(c *ui.Context) {
	ui.Box(c).Size(66, 66).Background(accent).Label("打开歌曲详情").OnClick(func() {
		if s.current != nil {
			s.lyricOpen = true
		}
	}).Children(func() {
		if s.cover != nil {
			ui.Image(c, s.cover).Fill().Fit(ui.Cover)
		}
		icon, size := "play", float32(30)
		if s.playing {
			icon, size = "pause", 20
		}
		button := ui.Button(c, "").Label("播放或暂停").Size(size, size).Radius(size/2).Padding(0).Border(2, white).Background(ui.Transparent).Absolute().OnClick(s.toggle).Children(func() { glyph(c, icon, size*.5, white) })
		if s.playing {
			button.Right(4).Bottom(4)
		} else {
			button.Left(18).Top(18)
		}
	})
}

func (s *app) playerView(c *ui.Context) {
	if !s.playerExpanded {
		ui.Row(c).Absolute().Left(0).Bottom(0).Children(func() {
			s.playerArtwork(c)
			ui.Button(c, "›").Width(18).Height(66).Radius(0).Padding(0).Background(ui.Hex("#e6e6e6")).Border(0, ui.Transparent).Label("展开播放器").OnClick(func() { s.playerExpanded = true })
		})
		return
	}
	ui.Column(c.Key("player")).Absolute().Left(0).Bottom(0).Width(418).Children(func() {
		if s.queueOpen && len(s.queue) > 0 {
			s.queueView(c)
		}
		ui.Row(c).Height(66).Background(white).Shadow(0, 0, 10, 0, ui.RGBA(0, 0, 0, .15)).Children(func() {
			s.playerArtwork(c)
			ui.Column(c).Grow(1).Height(66).Padding(14, 7, 0, 10).BorderWidth(1, 0, 0, 0).BorderColor(ui.Hex("#e9e9e9")).Children(func() {
				ui.Row(c).Height(20).Margin(0, 0, 13, 5).Gap(4).Clip().AlignItems(ui.Center).Children(func() {
					name, artist := "选择一首歌开始播放", ""
					if s.current != nil {
						name, artist = s.current.Name, "- "+s.current.Artist
					}
					ui.Text(c, name).FontSize(14).TextColor(ui.Hex("#111111")).SingleLine().Ellipsis("…")
					ui.Text(c, artist).FontSize(12).TextColor(ui.Hex("#666666")).SingleLine().Ellipsis("…")
				})
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					pos := s.position.Seconds()
					if slimSlider(c, "播放进度", &pos, max(1, s.duration.Seconds())).Grow(1).Disabled(s.current == nil).Changed() {
						s.seek(time.Duration(pos * float64(time.Second)))
					}
					ui.Text(c, clockText(s.position)+" / "+clockText(s.duration)).FontSize(11).TextColor(ui.Hex("#999999"))
					playerIcon(c, "prev", "prev", "上一首").OnClick(func() { s.step(-1) })
					playerIcon(c, "next", "next", "下一首").OnClick(func() { s.step(1) })
					volume := playerIcon(c, "volume", "volume", "播放器音量")
					if volume.Clicked() {
						s.playerVolumeOpen = !s.playerVolumeOpen
					}
					ui.PopoverBase(c, volume, &s.playerVolumeOpen, func(panel ui.Element) {
						panel.Padding(8).Radius(4).Background(white).Border(1, border).Margin(4, 0, 0, 0)
						v := s.cfg.Volume
						if slimSlider(c, "音量", &v, 1).Width(80).Changed() {
							s.volume(v)
						}
					})
					s.modeButton(c, "mode", "播放模式：", 16, ui.Hex("#666666"))
					playerIcon(c, "queue", "list", "播放队列").OnClick(s.toggleQueue)
				})
			})
			ui.Button(c, "‹").Width(18).Height(66).Radius(0).Padding(0).Background(ui.Hex("#e6e6e6")).Border(0, ui.Transparent).Label("收起播放器").OnClick(func() { s.playerExpanded = false })
		})
		if s.current != nil {
			ui.Row(c).Absolute().Bottom(72).Left(17).Width(371).Gap(6).Children(func() {
				s.modeButton(c, "playmode", "切换播放模式：", 32, white).Height(28).Radius(6).Background(accent).Border(1, ui.Hex("#059669"))
				colorIconButton(c, "favorite", "heart", "收藏当前歌曲", 32, ui.Hex("#047857"), true).Height(28).Radius(6).Border(1, ui.Hex("#cce7da")).OnClick(func() { s.pendingTracks = []library.Track{*s.current}; s.modal = "collections" })
				colorIconButton(c, "download", "download", "下载当前歌曲", 32, ui.Hex("#047857"), false).Height(28).Radius(6).Border(1, ui.Hex("#cce7da")).OnClick(func() { s.addDownloads([]library.Track{*s.current}) })
				ui.Spacer(c)
				button := ui.Button(c, fmt.Sprintf("%gx", s.speed)).Label("播放速度").Size(52, 28).Padding(0).Radius(6).Background(accent).Border(1, ui.Hex("#059669")).FontSize(13).FontWeight(700).TextColor(white)
				if button.Clicked() {
					s.speedOpen = !s.speedOpen
				}
				ui.PopoverBase(c, button, &s.speedOpen, func(panel ui.Element) {
					panel.Width(162).Padding(6).Radius(8).Border(1, accent.Alpha(.35)).Background(white).Margin(6, 0, 0, 0)
					ui.Grid(c).Columns(3).Gap(4).Children(func() {
						for _, speed := range []float64{.5, .75, 1, 1.25, 1.5, 2} {
							ui.Button(c, fmt.Sprintf("%gx", speed)).Label(fmt.Sprintf("速度 %gx", speed)).Padding(6, 4).FontSize(12).Radius(5).OnClick(func() {
								s.speed = speed
								if s.player != nil {
									s.player.SetSpeed(speed)
								}
								s.speedOpen = false
							})
						}
					})
				})
			})
		}
	})
}

func playerIcon(c *ui.Context, key, icon, label string) ui.Element {
	return colorIconButton(c, key, icon, label, 16, ui.Hex("#666666"), false).Radius(0).Background(ui.Transparent)
}

func slimSlider(c *ui.Context, label string, value *float64, hi float64) ui.Element {
	item := ui.SliderBase(c, value, 0, hi).Label(label).Height(12)
	at := float32(max(0, min(1, *value/hi)))
	hovered := item.Hovered() || item.Focused()
	return item.Draw(func(p *ui.Painter, r ui.Rect) {
		bar := ui.Rect{X: r.X, Y: r.Y + (r.H-2)/2, W: r.W, H: 2}
		p.Fill(bar, ui.Hex("#cdcdcd"), 0)
		bar.W *= at
		p.Fill(bar, accent, 0)
		if hovered {
			p.Fill(ui.Rect{X: bar.X + bar.W - 5, Y: r.Y + (r.H-10)/2, W: 10, H: 10}, accent, 5)
		}
	})
}

func (s *app) queueView(c *ui.Context) {
	ui.Column(c).Width(400).Background(white).Border(1, ui.Hex("#eeeeee")).Margin(0, 0, 34, 0).Children(func() {
		ui.Row(c).Padding(8, 12).Gap(8).Background(ui.Hex("#f8fafc")).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf("播放队列 · %d 首", len(s.queue))).FontSize(12).TextColor(muted).Grow(1)
			ui.Button(c, "清空队列").FontSize(11).OnClick(func() { s.queue = nil; s.queueIndex = -1; s.queueOpen = false })
		})
		ui.Scroll(c.Key("player-queue-scroll")).Label("播放队列列表").Height(min(220, float32(len(s.queue)*33))).TrackScroll(&s.queueScroll).Children(func() {
			ui.Column(c).Children(func() {
				for index, track := range s.queue {
					bg := white
					if s.current != nil && s.current.Key() == track.Key() {
						bg = ui.Hex("#e9e9e9")
					}
					row := ui.Row(c.Key(track.Key())).Label("队列歌曲 "+track.Name).Height(33).Gap(8).Padding(0, 8, 0, 12).Background(bg).AlignItems(ui.Center).OnClick(func() { s.play(index, s.queue) })
					if s.queueReveal && s.current != nil && s.current.Key() == track.Key() {
						row.ScrollIntoView()
					}
					row.Children(func() {
						ui.Text(c, fmt.Sprint(index+1)).FontSize(12).TextColor(muted)
						ui.Text(c, track.Name).FontSize(12).Grow(1).SingleLine().Ellipsis("…")
						ui.Text(c, track.Artist).FontSize(12).TextColor(muted).MaxWidth(125).SingleLine().Ellipsis("…")
						iconButton(c, "remove", "close", "移出队列 "+track.Name, 25).Background(ui.Transparent).OnClick(func() {
							s.queue = append(s.queue[:index:index], s.queue[index+1:]...)
							if index <= s.queueIndex {
								s.queueIndex--
							}
						})
					})
				}
			})
		})
	})
	s.queueReveal = false
}
