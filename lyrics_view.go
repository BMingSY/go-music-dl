package main

import (
	"math"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo/ui"
)

func (s *app) lyricsView(c *ui.Context) {
	w, h := c.Size()
	containerWidth := min(w, 1400)
	size := min(450, max(250, min(w, h)*.45)) * .85
	fontSize := s.cfg.LyricSize
	if fontSize == 0 {
		fontSize = 24
	}
	ui.Box(c.Key("lyrics-page")).Absolute().Fill().Background(ui.Hex("#080808")).Draw(func(p *ui.Painter, r ui.Rect) {
		if s.cover != nil {
			p.Image(s.cover, r, ui.Cover)
			p.Fill(r, ui.RGBA(0, 0, 0, .55), 0)
		}
	}).Children(func() {
		ui.Row(c).Absolute().Left((w-containerWidth)/2).Top(h*.05).Size(containerWidth, h*.9).Justify(ui.SpaceBetween).Children(func() {
			ui.Column(c).Width(containerWidth * .45).FillHeight().AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				record := ui.Box(c).Size(size, size).Margin(0, 0, h*.04, 0).Radius(size/2).Background(ui.Hex("#111111")).Border(2, white.Alpha(.1)).Shadow(0, 0, h*.05, 0, ui.RGBA(0, 0, 0, .8))
				if s.playing {
					record.Rotate(float32(math.Mod(s.position.Seconds()*15, 360)))
				}
				record.Draw(func(p *ui.Painter, r ui.Rect) {
					for inset := float32(4); inset < r.W*.18; inset += 4 {
						p.Stroke(ui.Rect{X: r.X + inset, Y: r.Y + inset, W: r.W - 2*inset, H: r.H - 2*inset}, ui.Hex("#252525"), (r.W-2*inset)/2, 1)
					}
				}).Children(func() {
					if s.cover != nil {
						ui.Image(c, s.cover).Absolute().Left(size*.18).Top(size*.18).Size(size*.64, size*.64).Fit(ui.Cover).Radius(size)
					}
				})
				ui.Column(c).Width(min(400, containerWidth*.45*.8)).Padding(15, 25).Margin(0, 0, h*.03, 0).Radius(12).Background(ui.RGBA(0, 0, 0, .65)).Border(1, white.Alpha(.15)).Shadow(0, 8, 20, 0, ui.RGBA(0, 0, 0, .5)).Children(func() {
					name, artist := "等待播放", ""
					if s.current != nil {
						name = s.current.Name
						artist = s.current.Artist
					}
					ui.Text(c, name).FontSize(min(32, max(20, w*.03))).FontWeight(800).LineHeight(1.2).TextColor(white).TextAlign(ui.Center).Margin(0, 0, 5, 0).MaxLines(2)
					ui.Text(c, artist).FontSize(min(18, max(14, w*.015))).FontWeight(500).TextColor(white.Alpha(.9)).AlignSelf(ui.Center).Margin(0, 0, 15, 0)
					ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
						ui.Text(c, clockText(s.position)).FontSize(12).TextColor(white)
						pos := s.position.Seconds()
						if ui.Slider(c, &pos, 0, max(1, s.duration.Seconds())).Grow(1).Label("歌词页播放进度").Disabled(s.player == nil).Changed() {
							s.seek(time.Duration(pos * float64(time.Second)))
						}
						ui.Text(c, clockText(s.duration)).FontSize(12).TextColor(white)
					})
					ui.Row(c).Margin(15, 0, 0, 0).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
						colorIconButton(c, "prev", "prev", "上一首", 28, white.Alpha(.8), false).Background(ui.Transparent).OnClick(func() { s.step(-1) })
						name := "play"
						if s.playing {
							name = "pause"
						}
						colorIconButton(c, "toggle", name, "播放或暂停", 40, white, false).MarginX(8).Background(white.Alpha(.15)).OnClick(s.toggle)
						colorIconButton(c, "next", "next", "下一首", 28, white.Alpha(.8), false).Background(ui.Transparent).OnClick(func() { s.step(1) })
						v := s.cfg.Volume
						glyph(c, "volume", 16, white).Margin(0, 8, 0, 20)
						if ui.Slider(c, &v, 0, 1).Width(65).Label("歌词页音量").Changed() {
							s.volume(v)
						}
					})
				})
				if s.status != "" {
					ui.Text(c, s.status).FontSize(12).TextColor(white).MaxLines(2)
				}
				ui.Row(c).Gap(16).Justify(ui.Center).Children(func() {
					colorIconButton(c, "cover", "image", "保存封面", 50, white, true).Background(ui.RGBA(0, 0, 0, .7)).Border(1, white.Alpha(.4)).OnClick(s.saveCurrentCover)
					colorIconButton(c, "audio", "music", "下载音频", 50, white, false).Background(ui.RGBA(0, 0, 0, .7)).Border(1, white.Alpha(.4)).OnClick(func() {
						if s.current != nil {
							s.addDownloads([]library.Track{*s.current})
						}
					})
					colorIconButton(c, "lrc", "file", "保存歌词", 50, white, false).Background(ui.RGBA(0, 0, 0, .7)).Border(1, white.Alpha(.4)).OnClick(func() {
						if s.current != nil {
							s.saveLyrics(*s.current)
						}
					})
					ui.Button(c, "导出视频").Radius(30).Padding(12, 30).TextColor(white).Background(ui.RGBA(0, 0, 0, .7)).Border(1, white.Alpha(.4)).OnClick(s.exportVideo)
				})
			})
			ui.Column(c).Width(containerWidth*.5).FillHeight().Padding(0, 0, 0, 40).Gap(h * .02).Children(func() {
				ui.Text(c, "LYRICS PREVIEW").FontSize(12).LetterSpacing(2).FontWeight(700).TextColor(white.Alpha(.9)).Background(ui.RGBA(0, 0, 0, .6)).Padding(5, 10).Border(1, white.Alpha(.1)).Radius(4).AlignSelf(ui.Start)
				if len(s.lines) == 0 {
					ui.Text(c, "暂无歌词").FontSize(fontSize).TextColor(ui.Hex("#cccccc")).MarginY(90)
				} else {
					idx := -1
					for i, line := range s.lines {
						if line.At <= s.position {
							idx = i
						}
					}
					if idx != s.lastLyric {
						s.lyricScroll.ScrollTo(0, max(0, float32(idx)*(fontSize+34)-(h-160)/2), ui.Smooth)
						s.lastLyric = idx
					}
					ui.Scroll(c).Grow(1).TrackScroll(&s.lyricScroll).Children(func() {
						ui.Column(c).Padding(45, 10, h*.3, 0).Children(func() {
							for i, line := range s.lines {
								color := ui.Hex("#999999")
								if i == idx {
									color = white
								}
								item := ui.Text(c, line.Text).Font(s.lyricFont()).FontSize(fontSize).FontWeight(600).LineHeight(1.6).TextColor(color).Padding(h*.015, 10).Radius(8).OnClick(func() { s.seek(line.At) })
								if i == idx {
									item.FontWeight(800).Padding(h*.015, 10, h*.015, 20).Background(ui.RGBA(0, 0, 0, .65)).Border(1, white.Alpha(.1)).Shadow(0, 6, 15, 0, ui.RGBA(0, 0, 0, .5))
								}
							}
						})
					})
				}
			})
		})
		colorIconButton(c, "close-lyrics", "close", "关闭歌曲详情", 40, white, false).Background(ui.RGBA(0, 0, 0, .6)).Border(1, white.Alpha(.3)).Absolute().Right(20).Top(20).OnClick(func() { s.lyricOpen = false })
	})
}
