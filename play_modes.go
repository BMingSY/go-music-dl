package main

import "github.com/egoist/mygo/ui"

func playbackMode(mode string) (string, string) {
	switch mode {
	case "loop":
		return "repeat", "列表循环"
	case "one":
		return "repeat", "单曲循环"
	case "shuffle":
		return "shuffle", "随机播放"
	default:
		return "list", "顺序播放"
	}
}

func (s *app) modeButton(c *ui.Context, key, label string, size float32, color ui.Color) ui.Element {
	icon, name := playbackMode(s.cfg.PlayMode)
	button := ui.Button(c.Key(key), "").Label(label+name).Tooltip(name).Size(size, size).Padding(0).Radius(size/2).Background(ui.Transparent).Border(0, ui.Transparent).OnClick(s.cycleMode)
	buttonMotion(button, 1.1, 0)
	button.Children(func() {
		glyph(c, icon, max(12, min(20, size*.55)), color)
		if s.cfg.PlayMode == "one" {
			ui.Text(c, "1").Absolute().Right(1).Bottom(0).FontSize(max(8, size*.28)).Bold().TextColor(color)
		}
	})
	return button
}

func (s *app) toggleQueue() {
	s.queueOpen = !s.queueOpen
	if !s.queueOpen {
		return
	}
	s.queueReveal = true
}
