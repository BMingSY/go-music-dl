package main

import "github.com/egoist/mygo/ui"

// The logo's gradient follows the full heading, as background-clip:text does.
type headingGradient struct{}

func (headingGradient) EffectMargin() ui.Point { return ui.Point{} }
func (headingGradient) PaintEffect(p *ui.Painter, box ui.Rect, content *ui.Bitmap, inset ui.Point) {
	scale := p.Scale()
	from, to := ui.Hex("#34d399"), ui.Hex("#059669")
	for x := float32(0); x < box.W; x += 2 {
		width := min(2, box.W-x)
		at := (x + width/2 + box.H/2) / (box.W + box.H)
		mix := func(a, b uint8) uint8 { return uint8(float32(a) + (float32(b)-float32(a))*at) }
		color := ui.RGB(mix(from.R, to.R), mix(from.G, to.G), mix(from.B, to.B))
		p.ImageRegion(content, ui.Rect{X: inset.X + x*scale, Y: inset.Y, W: width * scale, H: box.H * scale}, ui.Rect{X: box.X + x, Y: box.Y, W: width, H: box.H}, ui.ImageOptions{Tint: color})
	}
}
