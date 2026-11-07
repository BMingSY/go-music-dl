package desktop

import (
	"time"

	"github.com/egoist/mygo/ui"
)

var hoverAnimation = ui.Animation{Duration: 200 * time.Millisecond, Ease: ui.CubicBezier(.25, .1, .25, 1)}

var panelTransition = ui.ElementTransition{Duration: 180 * time.Millisecond, Enter: &ui.Motion{Y: -4, Scale: .98}, Exit: &ui.Motion{Y: -4, Scale: .98}}

func cardMotion(e ui.Element, lift float32) {
	y := float32(0)
	if e.Hovered() {
		y = -lift
		e.Shadow(0, 10, 20, 0, ui.RGBA(0, 0, 0, .1))
	}
	e.AnimatePose(ui.Pose{Y: y}, hoverAnimation).Transition(ui.ElementTransition{Colors: true, Duration: 200 * time.Millisecond})
}

func buttonMotion(e ui.Element, scale, lift float32) {
	pose := ui.Pose{}
	if e.Hovered() {
		pose.Y = -lift
		pose.Scale = scale
	}
	if e.Pressed() {
		pose.Scale = .94
		pose.Y = 0
	}
	e.AnimatePose(pose, hoverAnimation)
}
