package main

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"time"
)

func (s *app) statusToast(c *ui.Context) {
	if s.status == "" {
		s.toastSeen = ""
		return
	}
	if s.status != s.toastSeen {
		s.toastSeen = s.status
		s.toastStarted = c.Now()
	}
	if c.Now().Sub(s.toastStarted) >= 4*time.Second {
		return
	}
	c.After(4*time.Second - c.Now().Sub(s.toastStarted))
	from := ui.Hex("#ecfdf5")
	if strings.Contains(s.status, "失败") || strings.Contains(s.status, "错误") || strings.Contains(s.status, "不可用") {
		from = ui.Hex("#fff1f2")
	}
	ui.Box(c.Key("status-toast")).Lang("zh-CN").Absolute().Right(18).Bottom(105).Width(360).Padding(12, 38, 12, 14).Radius(14).Gradient(from, white.Alpha(.98), 135).Border(1, border).Shadow(0, 14, 32, 0, ui.RGBA(15, 23, 42, .16)).Transition(ui.ElementTransition{Duration: 220 * time.Millisecond, Enter: &ui.Motion{Y: 10}, Exit: &ui.Motion{}}).Children(func() {
		ui.Text(c, s.status).FontSize(14).FontWeight(800).TextColor(textColor).MaxLines(6)
		iconButton(c, "dismiss-status", "close", "关闭提示", 24).Absolute().Right(10).Top(8).Background(ui.Transparent).OnClick(func() { s.toastStarted = c.Now().Add(-4 * time.Second) })
	})
}
