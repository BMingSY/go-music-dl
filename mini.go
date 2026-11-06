package main

import (
	"fmt"
	"github.com/BMingSY/go-music-dl/internal/lyrics"
	"github.com/egoist/mygo/ui"
	"math"
	"time"
)

func (s *app) coverView(c *ui.Context, size float32) {
	if s.cover != nil {
		ui.Image(c, s.cover).Size(size, size).Fit(ui.Cover).Radius(7)
		return
	}
	ui.Column(c).Size(size, size).Center().Background(accent).Radius(7).Children(func() { ui.Text(c, "♫").FontSize(size * 0.5).TextColor(ui.Hex("#ffffff")) })
}

func (s *app) miniView(c *ui.Context) {
	initAssets()
	theme := ui.LightTheme()
	theme.FontSize = 12
	c.SetTheme(theme)
	ui.Row(c.Key("mini")).Fill().Padding(4).Gap(8).AlignItems(ui.Center).Background(white).Border(1, ui.Hex("#eeeeee")).Children(func() {
		box := ui.Box(c.Key("cover-drag")).Size(40, 40).DragWindow()
		box.Children(func() { s.coverView(c, 40) })
		ui.Column(c.Key("caption")).Grow(1).Clip().DragWindow().Gap(2).Children(func() {
			text, elapsed := s.miniCaption()
			frameAt := c.Now()
			ui.Box(c).Height(20).Label(text).Clip().Draw(func(p *ui.Painter, r ui.Rect) {
				span := ui.Span{Text: text, Size: 12, Color: ui.Hex("#222222"), Weight: 400}
				width, height := p.MeasureText(0, span)
				shift := float32(0)
				if s.playing && width > r.W {
					shift = float32(math.Mod(max(0, elapsed.Seconds()+p.Now().Sub(frameAt).Seconds()-.8)*25, float64(width+30)))
					p.AnimationFrameRate(30)
				}
				p.RichText(r.X-shift, r.Y+(r.H-height)/2, 0, span)
				if shift > 0 {
					p.RichText(r.X-shift+width+30, r.Y+(r.H-height)/2, 0, span)
				}
			})
			if !s.playing && s.current != nil {
				ui.Text(c, s.current.Artist).FontSize(10).TextColor(muted).SingleLine().Ellipsis("…")
			}
		})
		ui.Row(c).Gap(0).AlignItems(ui.Center).Children(func() {
			heart := iconButton(c, "favorite", "heart", "收藏当前歌曲", 23).OnClick(func() {
				if s.current != nil {
					s.favorite(*s.current)
				}
			})
			if s.current != nil && s.isFavorite(*s.current) {
				heart.Background(ui.Hex("#ffe4e6"))
			}
			play := "play"
			if s.playing {
				play = "pause"
			}
			iconButton(c, "toggle", play, "播放或暂停", 23).OnClick(s.toggle)
			volume := iconButton(c, "volume", "volume", "调节音量", 23)
			if volume.Hovered() {
				v := s.cfg.Volume
				if ui.Slider(c.Key("mini-volume"), &v, 0, 1).Absolute().Bottom(1).Right(26).Width(82).Height(10).Label("悬浮窗音量").Changed() {
					s.volume(v)
				}
			}
			iconButton(c, "queue", "list", "播放队列", 23).OnClick(func() {
				s.page = "queue"
				s.collections = nil
				s.categories = nil
				if s.win != nil {
					s.win.Show()
					s.win.Focus()
				}
			})
		})
		ui.Column(c).Gap(0).Children(func() {
			iconButton(c, "hide", "close", "隐藏悬浮窗", 15).OnClick(func() {
				if s.mini != nil {
					s.mini.Hide()
				}
			})
			ui.Button(c, "□").Label("打开主窗口").Size(15, 15).Padding(0).Border(0, ui.Transparent).Background(white).FontSize(10).OnClick(func() {
				if s.win != nil {
					s.win.Show()
					s.win.Focus()
				}
			})
		})
	})
}
func (s *app) miniCaption() (string, time.Duration) {
	if s.current == nil {
		return "MusicDL · 随时听歌", 0
	}
	if s.playing {
		if line, elapsed := lyrics.Current(s.lines, s.position); line != "" {
			return line, elapsed
		}
	}
	return caption(s.current), s.position
}

func clockText(d time.Duration) string {
	seconds := max(0, int(d.Seconds()))
	return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
}
