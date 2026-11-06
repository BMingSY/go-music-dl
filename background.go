package main

import (
	"image"
	"image/color"
	"math"

	"github.com/egoist/mygo/ui"
)

var backdropWidth int
var backdrop *ui.Bitmap

// CSS radial-gradient defaults to an ellipse reaching the farthest corner.
// The native radial primitive is circular, so render this static backdrop once
// per window width to retain the original gradient's proportions.
func pageBackdrop(width int) *ui.Bitmap {
	width = max(1, width)
	if backdrop != nil && width == backdropWidth {
		return backdrop
	}
	img := image.NewNRGBA(image.Rect(0, 0, width, 600))
	for y := 0; y < 600; y++ {
		for x := 0; x < width; x++ {
			r, g, b := float64(238), float64(242), float64(245)
			for _, layer := range []struct {
				at    float64
				color [3]float64
			}{{1, [3]float64{114, 39, 66}}, {.5, [3]float64{47, 62, 106}}, {0, [3]float64{17, 15, 21}}} {
				center := float64(width) * layer.at
				rx := max(center, float64(width)-center) * math.Sqrt2
				ry := 600 * math.Sqrt2
				alpha := max(0, 1-math.Hypot((float64(x)-center)/rx, float64(y)/ry)*2)
				r = layer.color[0]*alpha + r*(1-alpha)
				g = layer.color[1]*alpha + g*(1-alpha)
				b = layer.color[2]*alpha + b*(1-alpha)
			}
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(r + .5), G: uint8(g + .5), B: uint8(b + .5), A: 255})
		}
	}
	backdropWidth = width
	backdrop = ui.NewBitmap(img)
	return backdrop
}
