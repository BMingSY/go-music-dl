package desktop

import (
	"fmt"
	"time"

	"github.com/BMingSY/go-music-dl/internal/download"
	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (s *app) utilityModal(c *ui.Context, w, h float32) {
	title, icon, subtitle := "播放历史", "clock", "此设备最近播放过的歌曲"
	width := float32(622)
	rows := len(s.cfg.History)
	rowHeight := float32(0)
	for _, t := range s.cfg.History[:min(rows, 20)] {
		if t.PlayedAt > 0 {
			rowHeight += 81
		} else {
			rowHeight += 61
		}
	}
	height := min(680, h-32, 194+max(160, min(420, rowHeight)))
	if s.modal == "downloads" {
		title, icon, subtitle = "下载管理", "history", "任务自动保存，重启后继续；中断歌曲会重新下载。"
		width = 682
		itemsHeight := float32(0)
		for _, job := range s.jobs {
			if job.Hidden || s.downloadTab == "history" {
				continue
			}
			itemsHeight += 69
			if job.State == "downloading" || job.Error != "" {
				itemsHeight += 28
			}
		}
		baseHeight := float32(302)
		if s.downloadTab == "history" {
			baseHeight = 282
			for _, job := range s.jobs {
				if job.State == "done" || job.State == "skipped" || job.State == "failed" {
					itemsHeight += 43
				}
			}
		}
		height = min(680, h-32, baseHeight+max(180, min(420, itemsHeight)))
	}
	back := ui.Box(c.Key("utility-backdrop")).Absolute().Fill().Background(ui.RGBA(0, 0, 0, .5)).Center().Modal()
	back.Children(func() {
		ui.Column(c).Size(min(w-32, width), height).Radius(18).Border(1, ui.Hex("#dcebe4")).Background(white).Shadow(0, 24, 56, 0, ui.RGBA(15, 23, 42, .22)).Clip().Transition(panelTransition).Children(func() {
			ui.Row(c).Padding(17, 20, 14, 20).BorderWidth(0, 0, 1, 0).BorderColor(ui.Hex("#edf2f7")).AlignItems(ui.Start).Children(func() {
				ui.Column(c).Grow(1).Children(func() {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() { glyph(c, icon, 16, ui.Hex("#059669")); ui.Text(c, title).FontSize(16).Bold() })
					ui.Text(c, subtitle).FontSize(12).LineHeight(1.45).TextColor(muted).Margin(4, 0, 0, 0)
				})
				iconButton(c, "close-utility", "close", "关闭弹窗", 30).Radius(9).Background(ui.Transparent).OnClick(s.closeModal)
			})
			ui.Column(c).Grow(1).Padding(14, 20, 16, 20).Children(func() {
				if s.modal == "history" {
					s.historyView(c)
				} else {
					s.downloadsView(c)
				}
			})
		})
	})
	if back.OverlayShortcut(0, ui.KeyEscape) {
		s.closeModal()
	}
}

func utilityCount(c *ui.Context, text string) {
	ui.Text(c, text).FontSize(12).Bold().TextColor(ui.Hex("#047857")).Background(ui.Hex("#f0fdf4")).Border(1, ui.Hex("#d1fae5")).Height(30).Padding(0, 10).Radius(20)
}

func utilityDanger(c *ui.Context, label string, action func()) {
	ui.Button(c, label).Height(30).Padding(5, 10).Radius(20).FontSize(12).FontWeight(600).TextColor(ui.Hex("#dc2626")).Background(ui.Hex("#fee2e2")).Border(1, ui.Hex("#fca5a5")).OnClick(action)
}

func (s *app) utilityPagination(c *ui.Context, total, size int) {
	pages := max(1, (total+size-1)/size)
	s.utilityPage = max(1, min(pages, s.utilityPage))
	ui.Row(c).Margin(12, 0, 0, 0).Gap(9).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
		ui.Button(c, "‹").Label("记录上一页").Size(30, 30).Padding(0).Radius(9).Border(1, ui.Hex("#cce7da")).Background(ui.Hex("#f0fdf4")).TextColor(ui.Hex("#047857")).Disabled(s.utilityPage <= 1).OnClick(func() { s.utilityPage-- })
		ui.Text(c, fmt.Sprintf("%d / %d", s.utilityPage, pages)).Width(76).FontSize(12).Bold().TextColor(ui.Hex("#64748b")).TextAlign(ui.Center)
		ui.Button(c, "›").Label("记录下一页").Size(30, 30).Padding(0).Radius(9).Border(1, ui.Hex("#cce7da")).Background(ui.Hex("#f0fdf4")).TextColor(ui.Hex("#047857")).Disabled(s.utilityPage >= pages).OnClick(func() { s.utilityPage++ })
	})
}

func (s *app) confirmUtility(title, message string, apply func()) {
	go func() {
		r, e := mygo.Dialog.Message(mygo.MessageOptions{Parent: s.win, Title: title, Message: message, Buttons: []string{"取消", "确定"}})
		if e == nil && r.Button == 1 {
			s.update(apply)
		}
	}()
}

func (s *app) historyView(c *ui.Context) {
	ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Margin(0, 0, 12, 0).Children(func() {
		label := fmt.Sprintf("共 %d 条播放记录", len(s.cfg.History))
		if len(s.cfg.History) == 0 {
			label = "暂无播放记录"
		}
		utilityCount(c, label)
		utilityDanger(c, "清空历史", func() {
			s.confirmUtility("清空播放历史", "确定清空此设备的播放记录？", func() { s.cfg.History = nil; s.save() })
		})
	})
	s.utilityPage = max(1, min(max(1, (len(s.cfg.History)+19)/20), s.utilityPage))
	start := (s.utilityPage - 1) * 20
	end := min(start+20, len(s.cfg.History))
	ui.Scroll(c.Key("history-items")).Grow(1).MinHeight(160).Border(1, border).Radius(12).Children(func() {
		ui.Column(c).Children(func() {
			if len(s.cfg.History) == 0 {
				ui.Text(c, "暂无播放记录").FontSize(13).TextColor(muted).TextAlign(ui.Center).Padding(65, 20)
			}
			for index := start; index < end; index++ {
				track := s.cfg.History[index]
				row := ui.Row(c.Key(track.Key())).Padding(10).Gap(10).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(ui.Hex("#edf2f7"))
				if row.Hovered() {
					row.Background(ui.Hex("#f8fafc"))
				}
				row.Children(func() {
					ui.Box(c).Size(40, 40).Radius(10).Clip().Background(ui.Hex("#ecfdf5")).Center().Children(func() {
						if img := s.artwork(track); img != nil {
							ui.Image(c, img).Fill().Fit(ui.Cover)
						} else {
							glyph(c, "music", 18, ui.Hex("#059669"))
						}
					})
					ui.Column(c).Grow(1).Gap(3).Children(func() {
						ui.Text(c, track.Name).FontSize(13).FontWeight(800).SingleLine().Ellipsis("…")
						ui.Text(c, track.Artist+" · "+track.Source).FontSize(12).TextColor(muted).SingleLine().Ellipsis("…")
						if track.PlayedAt > 0 {
							ui.Text(c, time.Unix(track.PlayedAt, 0).Format("2006-01-02 15:04")).FontSize(11).TextColor(ui.Hex("#94a3b8"))
						}
					})
					ui.Button(c, "▶ 播放").Label("播放历史 "+track.Name).Padding(5, 9).Height(30).Radius(8).FontSize(12).FontWeight(800).Border(1, ui.Hex("#a7f3d0")).Background(ui.Hex("#ecfdf5")).TextColor(ui.Hex("#047857")).OnClick(func() { s.play(index, s.cfg.History) })
				})
			}
		})
	})
	s.utilityPagination(c, len(s.cfg.History), 20)
}

func (s *app) downloadsView(c *ui.Context) {
	ui.Row(c).Gap(20).Margin(0, 0, 16, 0).BorderWidth(0, 0, 1, 0).BorderColor(border).Children(func() {
		for _, tab := range []struct{ id, label string }{{"", "下载队列"}, {"history", "历史记录"}} {
			button := ui.Button(c, tab.label).Label("下载管理 "+tab.label).Padding(10, 2).Background(ui.Transparent).FontSize(13).Bold().TextColor(ui.Hex("#64748b")).Border(0, ui.Transparent).OnClick(func() { s.downloadTab = tab.id; s.utilityPage = 1 })
			if s.downloadTab == tab.id {
				button.TextColor(ui.Hex("#047857")).BorderWidth(0, 0, 2, 0).BorderColor(accent)
			}
		}
	})
	if s.downloadTab == "history" {
		s.downloadHistoryView(c)
		return
	}
	counts := map[string]int{}
	visible := 0
	var jobs []download.Job
	for _, j := range s.jobs {
		if j.Hidden {
			continue
		}
		counts[j.State]++
		visible++
		match := s.downloadFilter == "" || s.downloadFilter == j.State || (s.downloadFilter == "active" && (j.State == "queued" || j.State == "downloading"))
		if match {
			jobs = append(jobs, j)
		}
	}
	ui.Text(c, fmt.Sprintf("共 %d 项 · 等待 %d · 下载中 %d · 成功 %d · 失败 %d", visible, counts["queued"], counts["downloading"], counts["done"]+counts["skipped"], counts["failed"])).FontSize(12).TextColor(muted).Margin(0, 0, 12, 0)
	ui.Row(c).Gap(8).AlignItems(ui.Center).Margin(0, 0, 12, 0).Children(func() {
		ui.Text(c, "显示任务").FontSize(12).TextColor(muted)
		names := map[string]string{"": "全部", "active": "等待与下载中", "done": "成功", "skipped": "跳过", "failed": "失败", "canceled": "已取消"}
		selected := names[s.downloadFilter]
		if ui.Select(c, &selected, []string{"全部", "等待与下载中", "成功", "跳过", "失败", "已取消"}).Label("显示下载任务").Width(130).FontSize(12).Radius(8).Changed() {
			for key, name := range names {
				if name == selected {
					s.downloadFilter = key
					break
				}
			}
			s.utilityPage = 1
		}

		ui.Spacer(c)
		ui.Button(c, "刷新").Height(30).FontSize(12).Radius(20).OnClick(func() {
			if s.downloads != nil {
				s.jobs = s.downloads.Snapshot()
			}
		})
		utilityDanger(c, "清空任务", func() {
			s.confirmUtility("清空下载队列", "确定清空全部任务？等待任务将停止排队；正在下载的一首会完成写入。已下载文件和下载记录会保留。", func() {
				if s.downloads != nil {
					s.downloads.ClearPending()
					s.jobs = s.downloads.Snapshot()
				}
			})
		})
	})
	ui.Text(c, "关闭此面板不会停止下载。重启应用后自动恢复等待任务；此处显示任务状态和总体进度。").FontSize(12).LineHeight(1.6).TextColor(ui.Hex("#64748b")).Margin(0, 0, 12, 0)
	s.utilityPage = max(1, min(max(1, (len(jobs)+9)/10), s.utilityPage))
	start := (s.utilityPage - 1) * 10
	end := min(start+10, len(jobs))
	ui.Scroll(c.Key("download-items")).Grow(1).Border(1, border).Radius(12).Children(func() {
		ui.Column(c).Children(func() {
			if len(jobs) == 0 {
				ui.Text(c, "暂无下载任务").FontSize(13).TextColor(muted).TextAlign(ui.Center).Padding(65, 20)
			}
			for _, job := range jobs[start:end] {
				s.downloadJobView(c, job)
			}
		})
	})
	s.utilityPagination(c, len(jobs), 10)
}

func downloadState(state string) string {
	label := map[string]string{"queued": "等待下载", "downloading": "正在下载", "done": "成功", "skipped": "跳过", "failed": "失败", "canceled": "已取消"}[state]
	if label == "" {
		return state
	}
	return label
}

func (s *app) downloadJobView(c *ui.Context, j download.Job) {
	ui.Column(c.Key(j.Track.Key())).Padding(12).BorderWidth(0, 0, 1, 0).BorderColor(ui.Hex("#edf2f7")).Gap(6).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).Gap(3).Children(func() {
				ui.Text(c, j.Track.Name).FontSize(13).Bold().SingleLine().Ellipsis("…")
				ui.Text(c, j.Track.Artist+" · "+j.Track.Source).FontSize(12).TextColor(muted).SingleLine().Ellipsis("…")
			})
			ui.Text(c, downloadState(j.State)).FontSize(11).TextColor(muted)
			if j.State == "queued" || j.State == "downloading" {
				ui.Button(c, "取消").FontSize(12).OnClick(func() {
					if s.downloads != nil {
						s.downloads.Cancel(j.Track.Key())
					}
				})
			} else if j.State == "failed" || j.State == "canceled" {
				ui.Button(c, "重试").FontSize(12).OnClick(func() { s.addDownloads([]library.Track{j.Track}) })
			} else {
				ui.Button(c, "打开").FontSize(12).OnClick(func() { mygo.Shell.ShowItemInFolder(j.Path) })
			}
		})
		if j.Error != "" {
			ui.Text(c, j.Error).FontSize(12).TextColor(ui.Hex("#b91c1c")).MaxLines(2)
		}
		if j.State == "downloading" {
			progress := float64(0)
			if j.Total > 0 {
				progress = float64(j.Done) / float64(j.Total)
			}
			ui.Progress(c, progress).Height(6)
			ui.Text(c, fmt.Sprintf("%.1f / %.1f MiB", float64(j.Done)/(1<<20), float64(max(0, j.Total))/(1<<20))).FontSize(11).TextColor(muted)
		}
	})
}

func (s *app) downloadHistoryView(c *ui.Context) {
	windowWidth, _ := c.Size()
	available := min(float32(640), windowWidth-74) - 56
	widths := []float32{available * .32, available * .23, available * .12, available * .15, available * .18}
	var history []download.Job
	for _, j := range s.jobs {
		if j.State == "done" || j.State == "skipped" || j.State == "failed" {
			history = append(history, j)
		}
	}
	ui.Row(c).Margin(0, 0, 12, 0).AlignItems(ui.Center).Children(func() {
		utilityCount(c, fmt.Sprintf("共 %d 条记录", len(history)))
		ui.Spacer(c)
		ui.Button(c, "本地音乐").FontSize(12).Height(30).OnClick(func() { s.closeModal(); s.localMusic() })
		utilityDanger(c, "清空记录", func() {
			s.confirmUtility("清空下载记录", "确定清空下载记录？已下载的音乐文件会保留。", func() {
				if s.downloads != nil {
					s.downloads.Clear()
					s.jobs = s.downloads.Snapshot()
				}
			})
		})
	})
	ui.Row(c).Padding(10, 12).Gap(8).Background(ui.Hex("#f8fafc")).Border(1, border).Radius(8).Children(func() {
		for i, col := range []string{"歌曲", "歌手", "来源", "状态", "时间"} {
			ui.Text(c, col).FontSize(11).FontWeight(800).TextColor(ui.Hex("#64748b")).Width(widths[i])
		}
	})
	s.utilityPage = max(1, min(max(1, (len(history)+19)/20), s.utilityPage))
	start := (s.utilityPage - 1) * 20
	end := min(start+20, len(history))
	ui.Scroll(c).Grow(1).Border(1, border).Radius(12).Children(func() {
		ui.Column(c).Children(func() {
			if len(history) == 0 {
				ui.Text(c, "暂无下载记录").FontSize(13).TextColor(muted).TextAlign(ui.Center).Padding(65, 20)
			}
			for _, j := range history[start:end] {
				ui.Row(c.Key(j.Track.Key())).Padding(10, 12).Gap(8).BorderWidth(0, 0, 1, 0).BorderColor(ui.Hex("#edf2f7")).AlignItems(ui.Center).OnClick(func() {
					if j.Path != "" {
						mygo.Shell.ShowItemInFolder(j.Path)
					}
				}).Children(func() {
					ui.Text(c, j.Track.Name).FontSize(13).Bold().Width(widths[0]).SingleLine().Ellipsis("…")
					ui.Text(c, j.Track.Artist).FontSize(12).TextColor(muted).Width(widths[1]).SingleLine().Ellipsis("…")
					ui.Text(c, j.Track.Source).FontSize(12).TextColor(ui.Hex("#047857")).Width(widths[2])
					ui.Box(c).Width(widths[3]).Children(func() {
						color, background := ui.Hex("#047857"), ui.Hex("#ecfdf5")
						if j.State == "failed" {
							color, background = ui.Hex("#be123c"), ui.Hex("#fff1f2")
						}
						if j.State == "skipped" {
							color, background = ui.Hex("#a16207"), ui.Hex("#fffbeb")
						}
						ui.Text(c, downloadState(j.State)).FontSize(11).Bold().TextColor(color).Background(background).Radius(20).Padding(4, 7)
					})
					text := "—"
					if j.FinishedAt > 0 {
						text = time.Unix(j.FinishedAt, 0).Format("01-02 15:04")
					}
					ui.Text(c, text).FontSize(11).TextColor(muted).Width(widths[4])
				})
			}
		})
	})
	s.utilityPagination(c, len(history), 20)
}
