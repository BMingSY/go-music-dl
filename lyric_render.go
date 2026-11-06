package main

import (
	"github.com/BMingSY/go-music-dl/internal/lyrics"
	"github.com/egoist/mygo/ui"
	"time"
)

func outlineText(p *ui.Painter, x, y float32, span ui.Span, outline ui.Color, width float32) {
	color := span.Color
	span.Color = outline
	for _, d := range [][2]float32{{-2, -2}, {0, -2}, {2, -2}, {2, 0}, {2, 2}, {0, 2}, {-2, 2}, {-2, 0}, {-1, -1}, {1, -1}, {-1, 1}, {1, 1}, {0, 3}, {0, -3}, {3, 0}, {-3, 0}} {
		p.RichText(x+d[0]*width/2, y+d[1]*width/2, 0, span)
	}
	span.Color = color
	p.RichText(x, y, 0, span)
}
func (s *app) lyricWord(c *ui.Context, word lyrics.Word, size float32, position time.Duration, playing bool) {
	frameAt := c.Now()
	ui.Text(c, word.Text).Font(s.lyricFont()).FontSize(size).FontWeight(800).LineHeight(1.6).TextColor(ui.Transparent).SingleLine().DrawOver(func(p *ui.Painter, r ui.Rect) {
		now := position
		if playing {
			now += p.Now().Sub(frameAt)
			p.AnimationFrameRate(30)
		}
		span := ui.Span{Text: word.Text, Font: s.lyricFont(), Size: size, Weight: 800, Color: white}
		width, height := p.MeasureText(0, span)
		x, y := r.X, r.Y+(r.H-height)/2
		outlineText(p, x, y, span, accent, 2)
		progress := float32(0)
		if now >= word.End && word.End > word.At {
			progress = 1
		} else if now > word.At && word.End > word.At {
			progress = float32(now-word.At) / float32(word.End-word.At)
		}
		if progress > 0 {
			p.Clip(ui.Rect{X: x - 3, Y: y - 4, W: width*min(1, progress) + 3, H: height + 8}, 0, func() { span.Color = accent; outlineText(p, x, y, span, white, 2) })
		}
	})
}
func (s *app) lyricRows(c *ui.Context, height float32) {
	idx := lyrics.Index(s.lines, s.position)
	scroll := ui.Scroll(c.Key("lyric-scroll")).Grow(1).TrackScroll(&s.lyricScroll).Scrollbars(false).Effect(lyricEdgeMask{})
	scroll.Children(func() {
		ui.Column(c).Padding(height*.35, 10, height*.45, 0).Children(func() {
			for i, line := range s.lines {
				active := i == idx
				width, _ := c.Size()
				scale := s.cfg.LyricSize / 24
				if scale <= 0 {
					scale = 1
				}
				size := min(24, max(16, width*.012)) * scale
				if active {
					size = min(36, max(22, width*.02)) * scale
				}
				if size <= 0 {
					size = 16
				}
				row := ui.Column(c.Key(i)).Padding(height*.015, 10).Radius(8).Gap(4).Label(line.Text).OnClick(func() { s.seek(line.At) })
				row.Transition(ui.ElementTransition{Duration: 400 * time.Millisecond, Colors: true, Position: true, Size: true})
				if active {
					row.Background(ui.RGBA(0, 0, 0, .65)).Border(1, white.Alpha(.1)).Padding(height*.015, 10, height*.015, 20).Shadow(0, 6, 15, 0, ui.RGBA(0, 0, 0, .5))
				} else if row.Hovered() {
					row.Background(ui.RGBA(0, 0, 0, .4))
				}
				row.Children(func() {
					if len(line.Words) > 0 {
						ui.Row(c).Wrap().Gap(5).Children(func() {
							for _, word := range line.Words {
								s.lyricWord(c, word, size, s.position, s.playing)
							}
						})
					} else {
						color := white.Alpha(.85)
						weight := 600
						if active {
							color = white
							weight = 800
						}
						ui.Text(c, line.Text).Font(s.lyricFont()).FontSize(size).FontWeight(weight).LineHeight(1.6).TextColor(color).MaxLines(8)
					}
					if line.Translation != "" {
						ui.Text(c, line.Translation).Font(s.lyricFont()).FontSize(size * .708).FontWeight(650).LineHeight(1.4).TextColor(white).MaxLines(4)
					}
				})
				if active && s.lastLyric != idx {
					rect, viewport := row.Bounds(), scroll.Bounds()
					if rect.H > 0 && viewport.H > 0 {
						s.lyricScroll.ScrollTo(0, max(0, s.lyricScroll.Y+rect.Y+rect.H/2-viewport.Y-viewport.H/2), ui.Animation{Duration: 400 * time.Millisecond, Ease: ui.EaseOut})
						s.lastLyric = idx
					} else {
						c.After(time.Millisecond)
					}
				}
			}
		})
	})
}

// Draw the scroll layer through the same 10% edge fade as the upstream mask.
// Image strips keep this effect on the native renderer without a shader compiler.
type lyricEdgeMask struct{}

func (lyricEdgeMask) EffectMargin() ui.Point { return ui.Point{} }
func (lyricEdgeMask) PaintEffect(p *ui.Painter, box ui.Rect, content *ui.Bitmap, inset ui.Point) {
	scale := p.Scale()
	edge := box.H * .1
	strip := float32(1) / scale
	for y := float32(0); y < box.H; y += strip {
		height := min(strip, box.H-y)
		alpha := min(1, min((y+height/2)/edge, (box.H-y-height/2)/edge))
		p.ImageRegion(content, ui.Rect{X: inset.X, Y: inset.Y + y*scale, W: box.W * scale, H: height * scale}, ui.Rect{X: box.X, Y: box.Y + y, W: box.W, H: height}, ui.ImageOptions{Tint: white.Alpha(alpha)})
	}
}
