package main

import (
	"math"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/BMingSY/go-music-dl/internal/lyrics"
	"github.com/egoist/mygo/ui"
)

func (s *app) lyricsView(c *ui.Context) {
	w, h := c.Size()
	size := min(w*.28, h*.48)
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
		ui.Row(c).Fill().Padding(45, 55).Gap(60).Children(func() {
			ui.Column(c).Width(w * .42).AlignItems(ui.Center).Gap(24).Children(func() {
				ui.Spacer(c)
				record := ui.Box(c).Size(size, size).Radius(size/2).Background(ui.Hex("#141414")).Border(2, ui.Hex("#282828"))
				if s.playing {
					record.Rotate(float32(math.Mod(s.position.Seconds()*12, 360)))
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
				ui.Spacer(c)
				ui.Column(c).FillWidth().Padding(20).Gap(16).Radius(16).Background(ui.RGBA(0, 0, 0, .75)).Border(1, ui.Hex("#333333")).Children(func() {
					name, artist := "等待播放", ""
					if s.current != nil {
						name = s.current.Name
						artist = s.current.Artist
					}
					ui.Text(c, name).FontSize(32).Bold().TextColor(white).AlignSelf(ui.Center).SingleLine().Ellipsis("…")
					ui.Text(c, artist).FontSize(20).TextColor(ui.Hex("#cccccc")).AlignSelf(ui.Center)
					ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
						ui.Text(c, clockText(s.position)).FontSize(12).TextColor(white)
						pos := s.position.Seconds()
						if ui.Slider(c, &pos, 0, max(1, s.duration.Seconds())).Grow(1).Label("歌词页播放进度").Disabled(s.player == nil).Changed() {
							s.seek(time.Duration(pos * float64(time.Second)))
						}
						ui.Text(c, clockText(s.duration)).FontSize(12).TextColor(white)
					})
					ui.Row(c).Gap(18).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
						iconButton(c, "prev", "prev", "上一首", 36).OnClick(func() { s.step(-1) })
						name := "play"
						if s.playing {
							name = "pause"
						}
						iconButton(c, "toggle", name, "播放或暂停", 44).OnClick(s.toggle)
						iconButton(c, "next", "next", "下一首", 36).OnClick(func() { s.step(1) })
						v := s.cfg.Volume
						glyph(c, "volume", 16, white)
						if ui.Slider(c, &v, 0, 1).Width(80).Label("歌词页音量").Changed() {
							s.volume(v)
						}
					})
				})
				ui.Row(c).Gap(16).Justify(ui.Center).Children(func() {
					iconButton(c, "cover", "image", "保存封面", 48).OnClick(s.saveCurrentCover)
					iconButton(c, "audio", "music", "下载音频", 48).OnClick(func() {
						if s.current != nil {
							s.addDownloads([]library.Track{*s.current})
						}
					})
					iconButton(c, "lrc", "file", "保存歌词", 48).OnClick(func() {
						if s.current != nil {
							s.saveLyrics(*s.current)
						}
					})
					ui.Button(c, "导出视频").Radius(24).Height(48).OnClick(s.exportVideo)
				})
			})
			ui.Column(c).Grow(1).Gap(24).Children(func() {
				ui.Text(c, "LYRICS PREVIEW").FontSize(16).FontWeight(700).TextColor(white).Padding(8, 12).Border(1, ui.Hex("#333333")).Radius(5).AlignSelf(ui.Start)
				if len(s.lines) == 0 {
					ui.Text(c, "暂无歌词").FontSize(fontSize).TextColor(ui.Hex("#cccccc")).MarginY(90)
				} else {
					text, _ := lyrics.Current(s.lines, s.position)
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
						ui.Column(c).Gap(34).Padding(45, 0, h*.3, 0).Children(func() {
							for _, line := range s.lines {
								color := ui.Hex("#999999")
								if line.Text == text {
									color = white
								}
								ui.Text(c, line.Text).FontSize(fontSize).Bold().TextColor(color).OnClick(func() { s.seek(line.At) })
							}
						})
					})
				}
			})
		})
		iconButton(c, "close-lyrics", "close", "关闭歌曲详情", 44).Absolute().Right(25).Top(25).OnClick(func() { s.lyricOpen = false })
	})
}
