package main

import (
	"fmt"
	"math"
	"time"

	"github.com/BMingSY/go-music-dl/internal/lyrics"
	"github.com/egoist/mygo/ui"
)

func (s *app) coverView(c *ui.Context, size float32) {
	if s.cover != nil {
		ui.Image(c, s.cover).Size(size, size).Fit(ui.Cover)
		return
	}
	ui.Column(c).Size(size, size).Center().Background(accent).Children(func() { ui.Text(c, "♫").FontSize(size * 0.5).TextColor(ui.Hex("#ffffff")) })
}

func (s *app) miniView(c *ui.Context) {
	s.miniContent(c, s.showMain)
}

func (s *app) showMain() {
	if s.win != nil {
		if s.win.IsMinimized() {
			s.win.Restore()
		}
		s.win.Show()
		s.win.Focus()
	}
}

func miniButton(c *ui.Context, key, icon, label string, size float32) ui.Element {
	button := ui.Button(c.Key(key), "").Label(label).Size(size, size).Padding(0).Radius(size/2).Border(0, ui.Transparent).Background(ui.Transparent).HandleInput(miniControlInput)
	color := muted
	if button.Hovered() {
		button.Background(ui.Hex("#ecfdf5"))
		color = accent
	}
	button.Transition(ui.ElementTransition{Colors: true, Duration: 160 * time.Millisecond})
	buttonMotion(button, 1.14, 0)
	button.Children(func() { glyph(c, icon, 14, color) })
	return button
}

func (s *app) miniContent(c *ui.Context, restore func()) {
	initAssets()
	theme := ui.LightTheme()
	theme.FontSize = 12
	theme.Font = desktopFont()
	c.SetTheme(theme)
	root := ui.Box(c.Key("mini")).Fill().Lang("zh-CN").Background(white).Radius(6).Border(1, ui.Hex("#e2e8f0"))
	root.Label("打开主窗口").Role(ui.RoleGroup).HandleInput(s.miniSurfaceInput).OnClick(func() {
		moved := s.miniDrag.moved
		s.miniDrag.moved = false
		if !moved {
			restore()
		}
	})
	hovered := root.Hovered()
	if !hovered {
		s.miniVolumeOpen = false
	}
	root.Children(func() {
		ui.Row(c).Fill().Padding(5).Gap(8).AlignItems(ui.Center).PassThrough().Children(func() {
			ui.Box(c).Size(38, 38).Radius(3).Clip().Children(func() { s.coverView(c, 38) })
			caption := ui.Column(c.Key("caption")).Grow(1).Gap(1).Clip()
			fade := float32(0)
			if hovered {
				fade = 1
			}
			caption.AnimatePose(ui.Pose{Fade: fade}, hoverAnimation)
			caption.Children(func() {
				text, elapsed := s.miniCaption()
				translation := ""
				if i := lyrics.Index(s.lines, s.position); s.playing && i >= 0 {
					translation = s.lines[i].Translation
				}
				if !s.playing && s.current != nil {
					translation = s.current.Artist
				}
				scrollingCaption(c, text, s.lyricFont(), 12, ui.Hex("#222222"), elapsed, s.playing)
				if translation != "" {
					scrollingCaption(c, translation, s.lyricFont(), 10.5, muted, elapsed, s.playing)
				}
			})
		})
		controls := ui.Row(c.Key("mini-controls")).Absolute().Left(49).Right(3).Top(5).Bottom(5).Background(white.Alpha(.98)).Gap(6).AlignItems(ui.Center).Justify(ui.Center)
		fade := float32(1)
		if hovered {
			fade = 0
		}
		controls.AnimatePose(ui.Pose{Fade: fade}, hoverAnimation).IgnoreInput(!hovered)
		controls.Children(func() {
			miniButton(c, "prev", "prev", "上一首", 28).OnClick(func() { s.step(-1) })
			icon := "play"
			if s.playing {
				icon = "pause"
			}
			miniButton(c, "toggle", icon, "播放或暂停", 28).OnClick(s.toggle)
			miniButton(c, "next", "next", "下一首", 28).OnClick(func() { s.step(1) })
			miniButton(c, "volume", "volume", "调节音量", 28).OnClick(func() { s.miniVolumeOpen = !s.miniVolumeOpen })
			miniButton(c, "hide", "close", "隐藏悬浮窗", 24).OnClick(func() {
				if s.mini != nil {
					s.mini.Hide()
				}
			})
		})
		if s.miniVolumeOpen {
			ui.Row(c).Absolute().Left(57).Right(8).Bottom(1).Height(15).Background(white).Gap(4).AlignItems(ui.Center).Children(func() {
				v := s.cfg.Volume
				if slimSlider(c, "悬浮窗音量", &v, 1).Grow(1).HandleInput(miniControlInput).Changed() {
					s.volume(v)
				}
				ui.Button(c, "完成").FontSize(10).Padding(0).Height(15).HandleInput(miniControlInput).OnClick(func() { s.miniVolumeOpen = false })
			})
		}
	})
}

// Let controls use their normal input handling instead of the surrounding
// surface's window-drag handler. Gaps between controls still belong to it.
func miniControlInput(ui.InputEvent) bool { return false }

type miniDragState struct {
	pressed, moved   bool
	x, y             float32
	windowX, windowY int
}

func (s *app) miniSurfaceInput(ev ui.InputEvent) bool {
	if ev.Button != 0 {
		return false
	}
	switch ev.Kind {
	case ui.InputPointerDown:
		s.miniDrag = miniDragState{pressed: true, x: ev.X, y: ev.Y}
		if s.mini != nil {
			s.miniDrag.windowX, s.miniDrag.windowY = s.mini.Position()
		}
		return true
	case ui.InputPointerMove:
		if !s.miniDrag.pressed {
			return false
		}
		x, y := s.miniDrag.windowX, s.miniDrag.windowY
		if s.mini != nil {
			x, y = s.mini.Position()
		}
		dx := float32(x-s.miniDrag.windowX) + ev.X - s.miniDrag.x
		dy := float32(y-s.miniDrag.windowY) + ev.Y - s.miniDrag.y
		if math.Hypot(float64(dx), float64(dy)) > 4 {
			s.miniDrag.moved = true
		}
		if s.miniDrag.moved && s.mini != nil {
			s.mini.SetPosition(s.miniDrag.windowX+int(math.Round(float64(dx))), s.miniDrag.windowY+int(math.Round(float64(dy))))
		}
		return true
	case ui.InputPointerUp:
		s.miniDrag.pressed = false
		return true
	}
	return false
}
func scrollingCaption(c *ui.Context, text, font string, size float32, color ui.Color, elapsed time.Duration, playing bool) {
	frameAt := c.Now()
	ui.Box(c).Height(size + 5).Label(text).Clip().Draw(func(p *ui.Painter, r ui.Rect) {
		span := ui.Span{Text: text, Size: size, Font: font, Color: color, Weight: 400, Lang: "zh-CN"}
		width, height := p.MeasureText(0, span)
		shift := float32(0)
		if playing && width > r.W {
			shift = float32(math.Mod(max(0, elapsed.Seconds()+p.Now().Sub(frameAt).Seconds()-.8)*25, float64(width+30)))
			p.AnimationFrameRate(30)
		}
		p.RichText(r.X-shift, r.Y+(r.H-height)/2, 0, span)
		if shift > 0 {
			p.RichText(r.X-shift+width+30, r.Y+(r.H-height)/2, 0, span)
		}
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
	if !s.playing {
		return s.current.Name, 0
	}
	return caption(s.current), s.position
}

func clockText(d time.Duration) string {
	seconds := max(0, int(d.Seconds()))
	return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
}
