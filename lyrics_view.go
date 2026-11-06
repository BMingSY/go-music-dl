package main

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

func (s *app) lyricsView(c *ui.Context) {
	w, h := c.Size()
	now := c.Now()
	if s.visualEpoch.IsZero() {
		s.visualEpoch = now
	}
	if !s.discFrame.IsZero() && s.playing {
		s.discAngle += now.Sub(s.discFrame).Seconds() * 15
	}
	s.discFrame = now
	containerWidth := min(w, 1400)
	discContainer := min(450, max(250, min(w, h)*.45))
	size := discContainer * .85
	fontSize := s.cfg.LyricSize
	if fontSize == 0 {
		fontSize = 24
	}
	if s.player != nil {
		s.position = s.player.Position()
	}
	if s.playing {
		c.After(33 * time.Millisecond)
	}
	ui.Box(c.Key("lyrics-page")).Absolute().Fill().Background(ui.Hex("#080808")).Draw(func(p *ui.Painter, r ui.Rect) {
		if s.cover != nil {
			zoom := float32(1)
			if s.playing {
				t := p.Now().Sub(s.visualEpoch).Seconds()
				zoom += float32((1 - math.Cos(math.Pi*t/20)) * .05)
				p.AnimationFrameRate(30)
			}
			p.Image(s.cover, ui.Rect{X: r.X - r.W*(zoom-1)/2, Y: r.Y - r.H*(zoom-1)/2, W: r.W * zoom, H: r.H * zoom}, ui.Cover)
		}
	}).Children(func() {
		if r := s.previewInfo; r.W > 0 && s.cover != nil {
			ui.Box(c).Absolute().Left(r.X).Top(r.Y).Size(r.W, r.H).Radius(12).Clip().IgnoreInput(true).Children(func() {
				ui.Box(c).Absolute().Fill().Blur(10).Draw(func(p *ui.Painter, layer ui.Rect) {
					p.Image(s.cover, ui.Rect{X: layer.X - r.X, Y: layer.Y - r.Y, W: w, H: h}, ui.Cover)
				})
			})
		}
		ui.Row(c).Absolute().Left((w-containerWidth)/2).Top(h*.05).Size(containerWidth, h*.9).Justify(ui.SpaceBetween).Children(func() {
			ui.Column(c).Width(containerWidth * .45).FillHeight().AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				discBox := ui.Box(c).Size(discContainer, discContainer).Margin(0, 0, h*.04, 0).Center()
				discBox.Children(func() {
					s.spectrumView(c, discContainer, size)
					record := ui.Box(c).Size(size, size).Radius(size/2).Background(ui.Hex("#111111")).Border(2, white.Alpha(.1)).Shadow(0, 0, h*.05, 0, ui.RGBA(0, 0, 0, .8))

					record.Draw(func(p *ui.Painter, r ui.Rect) {
						for inset := float32(4); inset < r.W*.18; inset += 4 {
							p.Stroke(ui.Rect{X: r.X + inset, Y: r.Y + inset, W: r.W - 2*inset, H: r.H - 2*inset}, ui.Hex("#252525"), (r.W-2*inset)/2, 1)
						}
					}).Children(func() {
						if s.cover != nil {
							cover := ui.Image(c, s.cover).Absolute().Left(size*.175).Top(size*.175).Size(size*.65, size*.65).Fit(ui.Cover).Radius(size)
							cover.Rotate(float32(math.Mod(s.discAngle, 360)))
						}
					})
				})
				ui.Column(c).Width(min(400, containerWidth*.45*.8)).Padding(15, 25).Margin(0, 0, h*.03, 0).Radius(12).Background(ui.RGBA(0, 0, 0, .65)).Border(1, white.Alpha(.15)).Shadow(0, 8, 20, 0, ui.RGBA(0, 0, 0, .5)).Draw(func(p *ui.Painter, r ui.Rect) {
					if s.previewInfo != r {
						s.previewInfo = r
						p.AnimationFrame()
					}
				}).Children(func() {
					name, artist := "等待播放", ""
					if s.current != nil {
						name = s.current.Name
						artist = s.current.Artist
					}
					ui.Text(c, name).FontSize(min(32, max(20, w*.03))).FontWeight(800).LineHeight(1.2).TextColor(white).TextAlign(ui.Center).Margin(0, 0, 5, 0).MaxLines(2)
					ui.Text(c, artist).FontSize(min(18, max(14, w*.015))).FontWeight(500).TextColor(white.Alpha(.9)).AlignSelf(ui.Center).Margin(0, 0, 15, 0)
					ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Text(c, clockText(s.position)).FontSize(12).TextColor(white)
						pos := s.position.Seconds()
						if detailSlider(c, "歌词页播放进度", &pos, max(1, s.duration.Seconds())).Grow(1).Label("歌词页播放进度").Disabled(s.player == nil).Changed() {
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
						ui.Row(c).Gap(8).Margin(0, 0, 0, 20).Padding(0, 0, 0, 20).BorderWidth(0, 0, 0, 1).BorderColor(white.Alpha(.2)).AlignItems(ui.Center).Children(func() {
							v := s.cfg.Volume
							colorIconButton(c, "detail-mute", "volume", "静音或恢复音量", 16, white, false).Background(ui.Transparent).OnClick(func() {
								if s.cfg.Volume > 0 {
									s.volume(0)
								} else {
									s.volume(.7)
								}
							})
							if detailSlider(c, "歌词页音量", &v, 1).Width(80).Changed() {
								s.volume(v)
							}
							ui.Text(c, fmt.Sprintf("%.0f%%", s.cfg.Volume*100)).Font("Consolas").Width(35).FontSize(12).TextColor(white.Alpha(.8)).TextAlign(ui.End)
						})
						s.detailSpeed(c)
					})
				})
				if s.status != "" {
					ui.Text(c, s.status).FontSize(12).TextColor(white).MaxLines(2)
				}
				ui.Row(c).Gap(16).Justify(ui.Center).Children(func() {
					coverButton := colorIconButton(c, "cover", "image", "更换封面或背景", 50, white, true).Background(ui.RGBA(0, 0, 0, .7)).Border(1, white.Alpha(.4)).OnClick(s.replacePreviewCover)
					buttonMotion(coverButton, 1.1, 0)
					audioButton := colorIconButton(c, "audio", "music", "更换本地音频", 50, white, false).Background(ui.RGBA(0, 0, 0, .7)).Border(1, white.Alpha(.4)).OnClick(s.replacePreviewAudio)
					buttonMotion(audioButton, 1.1, 0)
					lyricButton := colorIconButton(c, "lrc", "file", "更换本地歌词", 50, white, false).Background(ui.RGBA(0, 0, 0, .7)).Border(1, white.Alpha(.4)).OnClick(s.replacePreviewLyrics)
					buttonMotion(lyricButton, 1.1, 0)
					ui.Button(c, "导出视频").FontSize(16).FontWeight(600).Radius(30).Padding(12, 30).TextColor(white).Background(ui.RGBA(0, 0, 0, .7)).Border(1, white.Alpha(.4)).OnClick(s.exportVideo)
				})
			})
			ui.Column(c).Width(containerWidth*.5).FillHeight().Padding(0, 0, 0, 40).Gap(h * .02).Children(func() {
				ui.Text(c, "LYRICS PREVIEW").FontSize(12).LetterSpacing(2).FontWeight(700).TextColor(white.Alpha(.9)).Background(ui.RGBA(0, 0, 0, .6)).Padding(5, 10).Border(1, white.Alpha(.1)).Radius(4).AlignSelf(ui.Start)
				if len(s.lines) == 0 {
					ui.Text(c, "暂无歌词").FontSize(fontSize).TextColor(ui.Hex("#cccccc")).MarginY(90)
				} else {
					s.lyricRows(c, h)
				}
			})
		})
		closeButton := colorIconButton(c, "close-lyrics", "close", "关闭歌曲详情", 40, white, false).Background(ui.RGBA(0, 0, 0, .6)).Border(1, white.Alpha(.3)).Absolute().Right(20).Top(20).OnClick(func() { s.lyricOpen = false })
		angle := float32(0)
		if closeButton.Hovered() {
			angle = 90
			closeButton.Background(ui.RGBA(0, 0, 0, .9))
		}
		closeButton.AnimatePose(ui.Pose{Rotate: angle}, hoverAnimation)
	})
}
