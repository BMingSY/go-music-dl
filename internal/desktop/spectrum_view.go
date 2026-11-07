package desktop

import (
	"math"

	"github.com/egoist/mygo/ui"
)

func spectrumColor(h float64) ui.Color {
	x := float32(1 - math.Abs(math.Mod(h/60, 2)-1))
	lo, hi := float32(.3), float32(1)
	mid := lo + (hi-lo)*x
	var r, g, b float32
	switch int(h/60) % 6 {
	case 0:
		r, g, b = hi, mid, lo
	case 1:
		r, g, b = mid, hi, lo
	case 2:
		r, g, b = lo, hi, mid
	case 3:
		r, g, b = lo, mid, hi
	case 4:
		r, g, b = mid, lo, hi
	default:
		r, g, b = hi, lo, mid
	}
	return ui.RGBA(uint8(r*255), uint8(g*255), uint8(b*255), .9)
}

func (s *app) spectrumView(c *ui.Context, container, disc float32) {
	frameAt := c.Now()
	position := s.position
	ui.Box(c).Absolute().Left(-container*.4).Top(-container*.4).Size(container*1.8, container*1.8).IgnoreInput(true).Draw(func(p *ui.Painter, r ui.Rect) {
		if s.decoder == nil {
			return
		}
		at := position
		if s.playing {
			at += p.Now().Sub(frameAt)
			p.AnimationFrameRate(30)
		}
		spectrum := s.decoder.Spectrum(at)
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		for i := 0; i < 180; i++ {
			index := math.Exp(math.Log(float64(len(spectrum))*.8) * float64(i) / 180)
			lo := min(len(spectrum)-1, int(index))
			hi := min(len(spectrum)-1, lo+1)
			fraction := float32(index - float64(lo))
			value := min(1, (spectrum[lo]*(1-fraction)+spectrum[hi]*fraction)*(1+float32(i)/180*.8))
			height := float32(2)
			if s.playing {
				height += float32(math.Pow(float64(value), 2.5)) * 40
			}
			angle := float32(i) * 360 / 180
			color := spectrumColor(float64(angle))
			p.Transform(ui.Rotation(angle), cx, cy, func() { p.Fill(ui.Rect{X: cx - .75, Y: cy - disc/2 - 2 - height, W: 1.5, H: height}, color, .5) })
		}
	})
}
