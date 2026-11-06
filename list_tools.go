package main

import (
	"strconv"

	"github.com/egoist/mygo/ui"
)

func (s *app) listTools(c *ui.Context) {
	button := ui.Button(c, "").Label("列表工具").Height(34).Padding(7, 10).Radius(10).Background(ui.Hex("#f7fffb")).Border(1, ui.Hex("#cceee0")).Children(func() {
		ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
			glyph(c, "sliders", 12, ui.Hex("#047857"))
			ui.Text(c, "列表工具").FontSize(12).FontWeight(800).TextColor(ui.Hex("#047857"))
			glyph(c, "chevron", 10, ui.Hex("#047857"))
		})
	})
	if button.Clicked() {
		s.toolsOpen = !s.toolsOpen
	}
	ui.PopoverBase(c, button, &s.toolsOpen, func(panel ui.Element) {
		panel.AttachTo(button, ui.AnchorBottomRight, ui.AnchorTopRight).Width(312).Margin(8, 0, 0, 0).Padding(12).Border(1, ui.Hex("#d9eee5")).Radius(14).Background(white.Alpha(.98)).Shadow(0, 18, 42, 0, ui.RGBA(15, 23, 42, .16))
		ui.Row(c).Padding(1, 2, 10, 2).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "列表工具").FontSize(13).FontWeight(800).Grow(1)
			ui.Text(c, "显示与操作").FontSize(11).FontWeight(600).TextColor(ui.Hex("#94a3b8"))
		})
		toolField(c, "每页显示", func() {
			size := strconv.Itoa(max(20, s.cfg.PageSize)) + "条/页"
			if s.cfg.PageSize == 0 {
				size = "30条/页"
			}
			if ui.Select(c, &size, []string{"20条/页", "30条/页", "50条/页", "100条/页", "200条/页", "300条/页", "400条/页", "500条/页"}).Label("每页条数").Grow(1).Height(32).FontSize(12).Radius(8).Changed() {
				s.cfg.PageSize, _ = strconv.Atoi(size[:len(size)-len("条/页")])
				s.pageNumber = 1
				s.save()
			}
		})
		toolField(c, "歌曲排序", func() {
			modes := map[string]string{"": "默认排序", "quality": "按音质", "size": "按大小", "duration": "按时长", "name": "按歌名"}
			selected := modes[s.sortMode]
			if ui.Select(c, &selected, []string{"默认排序", "按音质", "按大小", "按时长", "按歌名"}).Label("歌曲排序").Grow(1).Height(32).FontSize(12).Radius(8).Changed() {
				for mode, label := range modes {
					if label == selected {
						s.sortMode = mode
					}
				}
			}
			icon := "down"
			if s.sortReverse {
				icon = "up"
			}
			colorIconButton(c, "sort-direction", icon, "切换排序方向", 32, ui.Hex("#047857"), false).Radius(8).Border(1, ui.Hex("#d8e8e1")).OnClick(func() { s.sortReverse = !s.sortReverse })
		})
		ui.Row(c).Gap(7).Padding(10, 0, 0, 0).BorderWidth(1, 0, 0, 0).BorderColor(ui.Hex("#eef5f1")).Children(func() {
			ui.Button(c, "").Label("播放全部").Grow(1).Height(34).Radius(9).Gradient(ui.Hex("#34d399"), ui.Hex("#059669"), 135).Border(1, ui.Hex("#059669")).OnClick(func() { s.toolsOpen = false; s.play(0, s.shownTracks()) }).Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					glyph(c, "play", 12, white)
					ui.Text(c, "播放全部").FontSize(12).FontWeight(800).TextColor(white)
				})
			})
			ui.Button(c, "批量操作").Grow(1).Height(34).Radius(9).FontSize(12).FontWeight(800).Background(white).Border(1, ui.Hex("#d8e8e1")).OnClick(func() { s.toolsOpen = false; s.batch = !s.batch })
		})
	})
}

func toolField(c *ui.Context, label string, fn func()) {
	ui.Row(c).Padding(9, 2).Gap(6).AlignItems(ui.Center).BorderWidth(1, 0, 0, 0).BorderColor(ui.Hex("#eef5f1")).Children(func() {
		ui.Text(c, label).Width(76).FontSize(12).FontWeight(700).TextColor(ui.Hex("#64748b"))
		fn()
	})
}
