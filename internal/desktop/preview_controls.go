package desktop

import (
	"fmt"
	"io"
	"os"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/BMingSY/go-music-dl/internal/lyrics"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (s *app) previewActions(c *ui.Context) {
	ui.Row(c).Gap(12).Justify(ui.Center).Children(func() {
		s.modeButton(c, "detail-play-mode", "歌词页播放模式：", 44, white).Background(ui.RGBA(0, 0, 0, .65)).Border(1, white.Alpha(.35))
		more := colorIconButton(c, "preview-more", "more", "歌词页更多操作", 44, white, false).Background(ui.RGBA(0, 0, 0, .65)).Border(1, white.Alpha(.35))
		buttonMotion(more, 1.1, 0)
		if more.Clicked() {
			s.previewActionsOpen = !s.previewActionsOpen
		}
		ui.PopoverBase(c, more, &s.previewActionsOpen, func(panel ui.Element) {
			panel.Width(192).Padding(6).Radius(12).Background(ui.RGBA(15, 23, 42, .95)).Border(1, white.Alpha(.25)).Margin(8, 0).Transition(panelTransition)
			ui.Column(c).Gap(3).Children(func() {
				for _, action := range []struct {
					icon, label string
					run         func()
				}{{"image", "更换封面或背景", s.replacePreviewCover}, {"music", "更换本地音频", s.replacePreviewAudio}, {"file", "更换本地歌词", s.replacePreviewLyrics}, {"clapper", "导出视频", s.exportVideo}} {
					button := ui.Button(c.Key(action.label), "").Label(action.label).FillWidth().Height(36).Padding(8, 10).Radius(8).Border(0, ui.Transparent).Background(ui.Transparent).OnClick(func() { s.previewActionsOpen = false; action.run() })
					if button.Hovered() {
						button.Background(white.Alpha(.12))
					}
					button.Children(func() {
						ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() { glyph(c, action.icon, 14, white); ui.Text(c, action.label).FontSize(13).TextColor(white) })
					})
				}
			})
		})
	})
}

func detailSlider(c *ui.Context, label string, value *float64, hi float64) ui.Element {
	e := ui.SliderBase(c, value, 0, hi).Label(label).Height(16)
	at := float32(max(0, min(1, *value/hi)))
	return e.Draw(func(p *ui.Painter, r ui.Rect) {
		bar := ui.Rect{X: r.X, Y: r.Y + (r.H-4)/2, W: r.W, H: 4}
		p.Fill(bar, white.Alpha(.2), 2)
		bar.W *= at
		p.Fill(bar, accent, 2)
		p.Fill(ui.Rect{X: bar.X + bar.W - 6, Y: r.Y + (r.H-12)/2, W: 12, H: 12}, accent, 6)
		p.Fill(ui.Rect{X: bar.X + bar.W - 4, Y: r.Y + (r.H-8)/2, W: 8, H: 8}, white, 4)
	})
}

func (s *app) detailSpeed(c *ui.Context) {
	b := ui.Button(c, fmt.Sprintf("%gx", s.speed)).Label("歌词页播放速度").Width(48).Height(30).Padding(0, 6).Margin(0, 0, 0, 6).Radius(8).FontSize(12).Bold().Background(accent.Alpha(.22)).Border(1, ui.Hex("#34d399").Alpha(.75)).TextColor(white.Alpha(.95))
	if b.Clicked() {
		s.speedOpen = !s.speedOpen
	}
	ui.PopoverBase(c, b, &s.speedOpen, func(panel ui.Element) {
		panel.Padding(4).Radius(8).Border(1, white.Alpha(.3)).Background(ui.RGBA(0, 0, 0, .8)).Transition(panelTransition)
		ui.Row(c).Children(func() {
			for _, speed := range []float64{.5, .75, 1, 1.25, 1.5, 2} {
				ui.Button(c, fmt.Sprintf("%gx", speed)).FontSize(12).Padding(6, 8).Background(ui.Transparent).TextColor(white).OnClick(func() {
					s.speed = speed
					if s.player != nil {
						s.player.SetSpeed(speed)
					}
					s.speedOpen = false
				})
			}
		})
	})
}

func (s *app) previewFile(title string, apply func(string, []byte)) {
	go func() {
		paths, e := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: s.win, Title: title})
		if e != nil || len(paths) == 0 {
			return
		}
		f, e := os.Open(paths[0])
		if e != nil {
			s.update(func() { s.status = "文件读取失败" })
			return
		}
		data, e := io.ReadAll(io.LimitReader(f, 16<<20+1))
		f.Close()
		s.update(func() {
			if e != nil || len(data) > 16<<20 {
				s.status = "文件过大或无法读取"
				return
			}
			apply(paths[0], data)
		})
	}()
}

func (s *app) replacePreviewCover() {
	s.previewFile("更换封面或背景", func(_ string, b []byte) {
		bitmap, e := ui.DecodeBitmap(b)
		if e != nil {
			s.status = "请选择有效的图片文件"
			return
		}
		s.cover = bitmap
		s.previewCover = append([]byte(nil), b...)
		s.status = ""
	})
}

func (s *app) replacePreviewLyrics() {
	s.previewFile("更换本地歌词", func(_ string, b []byte) {
		lines := lyrics.Parse(string(b))
		if len(lines) == 0 {
			s.status = "文件中没有可识别的歌词时间轴"
			return
		}
		s.previewLyrics = string(b)
		s.lines = lines
		s.lastLyric = -1
		s.status = ""
	})
}

func (s *app) replacePreviewAudio() {
	go func() {
		paths, e := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: s.win, Title: "更换本地音频"})
		if e != nil || len(paths) == 0 {
			return
		}
		track, e := library.LocalTrack(paths[0])
		s.update(func() {
			if e != nil {
				s.status = e.Error()
				return
			}
			s.play(0, []library.Track{track})
			s.lyricOpen = true
		})
	}()
}
