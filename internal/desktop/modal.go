package desktop

import (
	"github.com/egoist/mygo/ui"
)

func (s *app) closeModal() {
	if s.modal == "update" {
		s.modal = "about"
		return
	}
	s.captureShortcut = ""
	if s.modal == "login" && s.loginCancel != nil {
		s.loginCancel()
		s.loginID++
	}
	s.modal = ""
	s.pendingTracks = nil
}

func (s *app) modalView(c *ui.Context, w, h float32) {
	if s.modal == "history" || s.modal == "downloads" {
		s.utilityModal(c, w, h)
		return
	}
	titles := map[string]string{"settings": "系统设置", "collections": "本地歌单", "login": "扫码登录", "downloads": "下载管理", "about": "关于 music-dl"}
	titles["collections"] = "收藏到我的歌单"
	titles["edit-playlist"] = "新建歌单"
	if s.editPlaylistID != "" {
		titles["edit-playlist"] = "编辑歌单"
	}
	titles["import-playlist"] = "导入歌单"
	titles["update"] = "软件更新"
	width, height := float32(640), min(h-40, 600)
	if s.modal == "about" {
		width = 420
		height = min(h-40, 360)
	}
	if s.modal == "update" {
		width = 420
		height = min(h-40, 320)
	}
	if s.modal == "settings" {
		width = 760
		height = min(h-40, 720)
	}
	if s.modal == "login" {
		width = 420
		height = 430
	}
	if s.modal == "edit-playlist" || s.modal == "collections" {
		s.playlistEditorModal(c, w, h)
		return
	}
	if s.modal == "collections" {
		width = 400
		height = min(h-40, 520)
	}
	if s.modal == "import-playlist" {
		width = 440
		height = min(h-40, 280)
	}
	back := ui.Box(c.Key("modal-backdrop")).Absolute().Fill().Background(ui.RGBA(0, 0, 0, .5)).Center().Modal().Transition(ui.ElementTransition{Enter: &ui.Motion{}, Exit: &ui.Motion{}})
	back.Children(func() {
		ui.Column(c).Size(min(w-40, width), height).Background(white).Radius(18).Border(1, ui.Hex("#dcebe4")).Shadow(0, 20, 40, 0, ui.RGBA(0, 0, 0, .2)).Clip().Transition(panelTransition).Children(func() {
			ui.Row(c).Padding(20, 22, 16, 22).BorderWidth(0, 0, 1, 0).BorderColor(ui.Hex("#edf2f7")).AlignItems(ui.Center).Children(func() {
				ui.Column(c).Grow(1).Gap(4).Children(func() {
					ui.Text(c, titles[s.modal]).FontSize(18).Bold()
					if s.modal == "settings" {
						ui.Text(c, "分类管理播放体验、下载同步与平台 Cookie").FontSize(12).TextColor(muted)
					}
				})
				iconButton(c, "close-modal", "close", "关闭弹窗", 28).OnClick(s.closeModal)
			})
			ui.Column(c).Grow(1).Padding(14, 20, 18, 20).Gap(14).Children(func() {
				switch s.modal {
				case "settings":
					s.settingsView(c)
				case "collections":
					s.savedPlaylistsView(c)
				case "login":
					s.loginView(c)
				case "downloads":
					s.downloadsView(c)
				case "about":
					s.aboutView(c)
				case "update":
					s.updateView(c)
				case "edit-playlist":
					s.editPlaylistView(c)
				case "import-playlist":
					s.importLinkView(c)
				}
			})
		})
	})
	if back.OverlayShortcut(0, ui.KeyEscape) {
		if s.captureShortcut != "" {
			s.captureShortcut = ""
		} else {
			s.closeModal()
		}
	}
}

func (s *app) playlistEditorModal(c *ui.Context, w, h float32) {
	height := float32(372)
	title := "新建歌单"
	if s.editPlaylistID != "" {
		title = "编辑歌单"
	}
	if s.modal == "collections" {
		height = min(h*.85, 136+float32(len(s.cfg.Playlists)+1)*72)
		title = "收藏到我的歌单"
	}
	if s.playlistError != "" {
		height += 36
	}
	back := ui.Box(c.Key("modal-backdrop")).Absolute().Fill().Background(ui.RGBA(0, 0, 0, .5)).Center().Modal()
	back.Children(func() {
		ui.Column(c).Width(min(w-40, 450)).Height(min(h*.85, height)).Padding(25).Radius(20).Background(white).Shadow(0, 20, 40, 0, ui.RGBA(0, 0, 0, .2)).Transition(panelTransition).Children(func() {
			ui.Row(c).Height(27).Margin(0, 0, 15, 0).AlignItems(ui.Center).Children(func() {
				ui.Text(c, title).FontSize(18).Bold().Grow(1)
				iconButton(c, "close-editor", "close", "关闭弹窗", 20).Background(ui.Transparent).OnClick(s.closeModal)
			})
			if s.modal == "collections" {
				s.savedPlaylistsView(c)
			} else {
				ui.Scroll(c).Grow(1).Children(func() { ui.Column(c).Children(func() { s.editPlaylistView(c) }) })
			}
		})
	})
	if back.OverlayShortcut(0, ui.KeyEscape) {
		s.closeModal()
	}
}
