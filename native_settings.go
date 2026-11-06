package main

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo/ui"
)

var shortcutActions = []struct{ id, label string }{{"toggle", "播放 / 暂停"}, {"previous", "上一首"}, {"next", "下一首"}, {"back", "后退 10 秒"}, {"forward", "前进 10 秒"}}

func defaultShortcuts() map[string]library.Shortcut {
	return map[string]library.Shortcut{
		"toggle":   {Key: uint16(ui.KeySpace), Label: "Space"},
		"previous": {Key: uint16(ui.KeyLeft), Modifiers: uint8(ui.Ctrl), Label: "Ctrl + ←"},
		"next":     {Key: uint16(ui.KeyRight), Modifiers: uint8(ui.Ctrl), Label: "Ctrl + →"},
		"back":     {Key: uint16(ui.KeyLeft), Label: "←"},
		"forward":  {Key: uint16(ui.KeyRight), Label: "→"},
	}
}

func (s *app) handleShortcuts(c *ui.Context) {
	if s.modal != "" || s.queryFocused || s.toolsOpen || s.songMenuOpen {
		return
	}
	for action, key := range s.cfg.Shortcuts {
		if key.Key == 0 || !c.Shortcut(ui.Modifiers(key.Modifiers), ui.Key(key.Key)) {
			continue
		}
		switch action {
		case "toggle":
			s.toggle()
		case "previous":
			s.step(-1)
		case "next":
			s.step(1)
		case "back":
			s.seek(s.position - 10*time.Second)
		case "forward":
			s.seek(s.position + 10*time.Second)
		}
	}
	if c.Shortcut(0, ui.KeyPageDown) {
		s.pageNumber++
		s.scroll.ScrollTo(0, 350, ui.Smooth)
	}
	if c.Shortcut(0, ui.KeyPageUp) {
		s.pageNumber = max(1, s.pageNumber-1)
		s.scroll.ScrollTo(0, 350, ui.Smooth)
	}
}

func keyLabel(key ui.Key, mods ui.Modifiers) string {
	label := map[ui.Key]string{ui.KeySpace: "Space", ui.KeyLeft: "←", ui.KeyRight: "→", ui.KeyUp: "↑", ui.KeyDown: "↓", ui.KeyEnter: "Enter", ui.KeyHome: "Home", ui.KeyEnd: "End"}[key]
	if key >= ui.KeyA && key <= ui.KeyZ {
		label = string(rune('A' + key - ui.KeyA))
	}
	if key >= ui.Key0 && key <= ui.Key9 {
		label = string(rune('0' + key - ui.Key0))
	}
	if key >= ui.KeyF1 && key <= ui.KeyF12 {
		label = fmt.Sprintf("F%d", key-ui.KeyF1+1)
	}
	if label == "" {
		return ""
	}
	var parts []string
	for _, m := range []struct {
		mod  ui.Modifiers
		name string
	}{{ui.Ctrl, "Ctrl"}, {ui.Super, "Cmd"}, {ui.Alt, "Alt"}, {ui.Shift, "Shift"}} {
		if mods&m.mod != 0 {
			parts = append(parts, m.name)
		}
	}
	return strings.Join(append(parts, label), " + ")
}

func settingsPanel(c *ui.Context, fn func()) {
	ui.Column(c).Padding(10, 12).Border(1, border).Radius(10).Background(ui.Hex("#f8fafc")).Children(fn)
}
func sectionHeading(c *ui.Context, icon, title, subtitle string) {
	ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(32, 32).Radius(9).Background(ui.Hex("#ecfdf5")).Center().Children(func() { glyph(c, icon, 14, ui.Hex("#059669")) })
		ui.Column(c).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(14).FontWeight(700)
			ui.Text(c, subtitle).FontSize(12).TextColor(muted)
		})
	})
}
func settingToggle(c *ui.Context, value *bool, title, hint string) bool {
	changed := false
	settingsPanel(c, func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			on := *value
			bg, x := ui.Hex("#d1d5db"), float32(3)
			if on {
				bg, x = ui.Hex("#34d399"), 21
			}
			changed = ui.SwitchBase(c, value).Label(title).Size(46, 28).Radius(14).Background(bg).Children(func() {
				ui.Box(c).Absolute().Top(3).Left(x).Size(22, 22).Radius(11).Background(white).Shadow(0, 1, 3, 0, ui.RGBA(0, 0, 0, .25))
			}).Changed()
			if ui.Button(c, title).Padding(0).Background(ui.Transparent).Border(0, ui.Transparent).FontSize(13).FontWeight(600).Clicked() {
				*value = !*value
				changed = true
			}
		})
		if hint != "" {
			ui.Text(c, hint).FontSize(12).LineHeight(1.45).TextColor(muted).Margin(8, 0, 0, 26)
		}
	})
	return changed
}

func (s *app) generalSettings(c *ui.Context) {
	if settingToggle(c, &s.cfg.FloatingLyrics, "播放音乐时显示悬浮歌词", "关闭后，播放音乐时不自动显示悬浮窗。") && s.mini != nil {
		if s.cfg.FloatingLyrics && s.playing {
			s.mini.Show()
		} else {
			s.mini.Hide()
		}
	}
	settingsPanel(c, func() {
		sectionHeading(c, "font", "歌词字体", "同时应用到播放器歌词和悬浮歌词")
		ui.Row(c).Gap(12).Margin(13, 0, 0, 0).Children(func() {
			ui.Column(c).Grow(1.2).Gap(4).Children(func() {
				ui.Text(c, "字体").FontSize(12).FontWeight(600).TextColor(muted)
				font := s.cfg.LyricFont
				if font == "" {
					font = "系统默认"
				}
				if ui.Select(c, &font, []string{"系统默认", "现代黑体", "宋体 / 衬线", "微软雅黑", "苹方", "思源黑体 / Noto Sans CJK", "霞鹜文楷 / 楷体"}).Label("歌词字体").FontSize(14).Height(38).Background(white).Radius(8).Changed() {
					s.cfg.LyricFont = font
				}
			})
			ui.Column(c).Grow(.8).Gap(4).Children(func() {
				size := float64(s.cfg.LyricSize)
				if size == 0 {
					size = 24
				}
				ui.Row(c).Children(func() {
					ui.Text(c, "字号").Grow(1).FontSize(12).FontWeight(600).TextColor(muted)
					ui.Text(c, fmt.Sprintf("%.0f px", size)).FontSize(12).TextColor(ui.Hex("#059669"))
				})
				if ui.Slider(c, &size, 14, 48).Height(30).Label("歌词字号").Changed() {
					s.cfg.LyricSize = float32(size)
				}
			})
		})
		ui.Text(c, "示例歌词 · 让每一句都清晰可见").Font(s.lyricFont()).FontSize(max(14, s.cfg.LyricSize)).FontWeight(750).TextColor(ui.Hex("#047857")).TextAlign(ui.Center).SingleLine().Ellipsis("…").Padding(16, 12).Margin(12, 0, 0, 0).Radius(10).Border(1, ui.Hex("#a7f3d0")).Gradient(ui.Hex("#ecfdf5"), ui.Hex("#f8fafc"), 135)
	})
	settingsPanel(c, func() {
		sectionHeading(c, "keyboard", "应用内全局快捷键", "焦点不在输入框且未打开弹窗时生效")
		ui.Column(c).Gap(8).Margin(13, 0, 0, 0).Children(func() {
			for _, action := range shortcutActions {
				ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
					ui.Text(c, action.label).Width(170).FontSize(13)
					key := s.cfg.Shortcuts[action.id]
					label := key.Label
					if label == "" {
						label = "未设置"
					}
					if s.captureShortcut == action.id {
						label = "请按下组合键…"
					}
					ui.Button(c.Key("capture-"+action.id), label).Label(action.label+"快捷键").Font("Consolas").FontSize(13).Height(37).Grow(1).Background(white).Border(1, border).Radius(8).OnClick(func() { s.captureShortcut = action.id }).HandleInput(func(ev ui.InputEvent) bool {
						if s.captureShortcut != action.id || ev.Kind != ui.InputKeyDown {
							return false
						}
						if ev.Key == ui.KeyBackspace || ev.Key == ui.KeyDelete {
							delete(s.cfg.Shortcuts, action.id)
							s.captureShortcut = ""
							return true
						}
						if ev.Key == ui.KeyPageUp || ev.Key == ui.KeyPageDown {
							return true
						}
						name := keyLabel(ev.Key, ev.Mods)
						if name == "" {
							return false
						}
						for id, existing := range s.cfg.Shortcuts {
							if id != action.id && existing.Key == uint16(ev.Key) && existing.Modifiers == uint8(ev.Mods) {
								s.status = "此快捷键已用于其他操作"
								return true
							}
						}
						s.cfg.Shortcuts[action.id] = library.Shortcut{Key: uint16(ev.Key), Modifiers: uint8(ev.Mods), Label: name}
						s.captureShortcut = ""
						return true
					})
					iconButton(c, "clear-"+action.id, "close", "清除"+action.label+"快捷键", 34).Radius(8).Border(1, border).OnClick(func() { delete(s.cfg.Shortcuts, action.id) })
				})
			}
		})
		ui.Text(c, "点击输入框后按下组合键；Backspace / Delete 清空，Esc 取消。PageUp / PageDown 用于列表翻页。").FontSize(12).LineHeight(1.45).TextColor(muted).Margin(8, 0, 0, 0)
	})
	ui.Column(c).Gap(4).Children(func() {
		ui.Text(c, "每页条数").FontSize(12).FontWeight(600).TextColor(muted)
		pageSize := s.cfg.PageSize
		if pageSize < 20 {
			pageSize = 30
		}
		size := strconv.Itoa(pageSize)
		if ui.Select(c, &size, []string{"20", "30", "50", "100", "200", "300", "400", "500"}).Label("默认每页条数").Background(white).Radius(8).Height(38).FontSize(14).Changed() {
			s.cfg.PageSize, _ = strconv.Atoi(size)
		}
	})
	setting(c, "音乐文件夹", func() {
		ui.Row(c).Gap(8).Children(func() {
			ui.TextInput(c, &s.cfg.MusicDir).Label("音乐文件夹").Background(white).Grow(1).FontSize(14)
			ui.Button(c, "浏览文件夹").OnClick(s.pickFolder)
		})
	})
	settingToggle(c, &s.cfg.ShowMascot, "显示桌面挂件", "")
	ui.Button(c, "").Label("关于 music-dl").Padding(12).Radius(10).Background(ui.Hex("#f8fafc")).Border(1, border).OnClick(func() { s.modal = "about" }).Children(func() { sectionHeading(c, "info", "关于 music-dl", "查看当前版本并检查更新") })
}

func (s *app) lyricFont() string {
	switch s.cfg.LyricFont {
	case "现代黑体":
		if runtime.GOOS == "darwin" {
			return "Helvetica Neue"
		}
		return "Arial"
	case "宋体 / 衬线":
		if runtime.GOOS == "darwin" {
			return "Songti SC"
		}
		return "SimSun"
	case "微软雅黑":
		return "Microsoft YaHei"
	case "苹方":
		return "PingFang SC"
	case "思源黑体 / Noto Sans CJK":
		return "Noto Sans CJK SC"
	case "霞鹜文楷 / 楷体":
		if runtime.GOOS == "darwin" {
			return "Kaiti SC"
		}
		return "KaiTi"
	default:
		return desktopFont()
	}
}
