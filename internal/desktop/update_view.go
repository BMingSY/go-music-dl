package desktop

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (s *app) updateView(c *ui.Context) {
	if s.pendingUpdate == nil {
		return
	}
	ui.Column(c).Gap(6).Children(func() {
		ui.Text(c, "发现新版本 "+s.pendingUpdate.Version).FontSize(22).FontWeight(700).TextColor(accent)
		ui.Text(c, "当前版本 "+mygo.App.Version()).FontSize(13).TextColor(muted)
	})
	ui.Text(c, "下载完成后将自动重启更新。").FontSize(14).TextColor(textColor)
	if s.updateBusy {
		ui.Column(c).Gap(8).Children(func() {
			ui.Text(c, s.updateStatus).FontSize(13).TextColor(muted)
			ui.Box(c).Height(6).Background(border).Radius(3).Clip().Children(func() {
				ui.Box(c).WidthPercent(float32(s.updateProgress * 100)).Height(6).Background(accent).Radius(3)
			})
		})
	} else if s.updateStatus != "发现新版本 "+s.pendingUpdate.Version {
		ui.Text(c, s.updateStatus).FontSize(13).TextColor(muted).MaxLines(2)
	}
	ui.Spacer(c)
	ui.Row(c).Gap(10).Justify(ui.End).Children(func() {
		later := ui.Button(c, "稍后").Label("稍后").Height(40).Padding(8, 18).Radius(8).FontSize(14).Disabled(s.updateBusy).OnClick(s.closeModal)
		label := "下载并重启"
		if s.updateBusy {
			label = "正在下载…"
		}
		download := ui.PrimaryButton(c, label).Label(label).Height(40).Padding(8, 18).Radius(8).FontSize(14).FontWeight(600).Disabled(s.updateBusy).OnClick(s.installUpdate)
		buttonMotion(later, 1, 1)
		buttonMotion(download, 1, 1)
	})
}
