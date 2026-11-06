package main

import (
	"fmt"
	"math"
	"time"

	"github.com/BMingSY/go-music-dl/native/internal/lyrics"
	"github.com/egoist/mygo/ui"
)

var accent = ui.Hex("#228b69")

func (s *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Grow(1).AlignItems(ui.Stretch).Children(func() {
			ui.Column(c).Width(170).Padding(20, 14).Gap(10).Background(t.Surface).Children(func() {
				ui.Text(c, "MusicDL").FontSize(26).Bold().TextColor(accent)
				ui.Text(c, "NATIVE PREVIEW").FontSize(10).TextColor(t.TextMuted).Margin(0, 0, 20, 0)
				ui.Text(c, "音乐来源").FontSize(11).TextColor(t.TextMuted)
				for _, src := range []struct{ id, label string }{{"netease", "网易云音乐"}, {"bilibili", "Bilibili"}, {"local", "本地音乐"}} {
					b := ui.Button(c.Key(src.id), src.label).Width(142)
					if s.source == src.id && s.page == "library" {
						b.TextColor(accent)
					}
					if b.Clicked() {
						s.selectSource(src.id)
					}
				}
				ui.Divider(c).MarginY(10)
				if ui.Button(c, "我的收藏").Width(142).Clicked() {
					s.page = "favorites"
					s.list = ui.ListState{}
				}
				if ui.Button(c, "播放队列").Width(142).Clicked() {
					s.page = "queue"
					s.list = ui.ListState{}
				}
				ui.Spacer(c)
				ui.Button(c, "悬浮窗").Width(142).OnClick(s.toggleMini)
				if ui.Button(c, "设置").Width(142).Clicked() {
					s.page = "settings"
				}
			})
			ui.Column(c).Grow(1).Padding(26).Gap(16).Children(func() {
				if s.page == "settings" {
					s.settingsView(c)
					return
				}
				title := "发现音乐"
				if s.page == "favorites" {
					title = "我的收藏"
				} else if s.page == "queue" {
					title = "播放队列"
				} else if s.source == "local" {
					title = "本地音乐"
				}
				ui.Text(c, title).FontSize(26).Bold()
				if s.page == "library" {
					ui.Row(c).Gap(10).Children(func() {
						if ui.SearchField(c.Key("query"), &s.query).Grow(1).Label("搜索关键词").Placeholder("搜索歌名、作者，或粘贴分享链接").Submitted() {
							s.search()
						}
						ui.PrimaryButton(c, "搜索").Disabled(s.searching).OnClick(s.search)
						if s.source == "local" {
							ui.Button(c, "选择文件夹").OnClick(s.pickFolder)
						}
					})
				}
				tracks := s.results
				if s.page == "favorites" {
					tracks = s.cfg.Favorites
				} else if s.page == "queue" {
					tracks = s.queue
				}
				ui.Row(c).Gap(10).PaddingX(8).Children(func() {
					ui.Text(c, fmt.Sprintf("%d 首歌曲", len(tracks))).FontSize(12).TextColor(t.TextMuted)
					ui.Spacer(c)
					if s.searching {
						ui.Text(c, "正在查找…").FontSize(12).TextColor(accent)
					}
				})
				ui.List(c.Key("tracks"), &s.list, len(tracks), func(i int) {
					track := tracks[i]
					ui.Row(c.Key(track.Key())).Padding(10, 8).Gap(12).AlignItems(ui.Center).Children(func() {
						ui.Text(c, fmt.Sprintf("%02d", i+1)).Width(28).FontSize(12).TextColor(t.TextMuted)
						ui.Column(c).Grow(1).Gap(3).Children(func() {
							label := ui.Text(c, track.Name).SingleLine().Ellipsis("…").FontSize(14)
							if s.current != nil && track.Key() == s.current.Key() {
								label.TextColor(accent).Bold()
							}
							ui.Text(c, track.Artist).SingleLine().Ellipsis("…").FontSize(12).TextColor(t.TextMuted)
						})
						ui.Text(c, track.FormatDuration()).Width(42).FontSize(12).TextColor(t.TextMuted)
						ui.Button(c.Key("play"), "播放").Label("播放 " + track.Name).OnClick(func() { s.play(i, tracks) })
						fav := "♡"
						if s.isFavorite(track) {
							fav = "♥"
						}
						ui.Button(c.Key("favorite"), fav).Label("收藏 " + track.Name).OnClick(func() { s.favorite(track) })
					})
				}).Grow(1).Children(func() {
					if len(tracks) == 0 {
						ui.Column(c).Padding(30).Gap(8).Children(func() {
							ui.Text(c, "从一首喜欢的歌开始").FontSize(18)
							ui.Text(c, "搜索网易云、Bilibili，或选择本地音乐文件夹。").FontSize(13).TextColor(t.TextMuted)
						})
					}
				})
				ui.Text(c, s.status).FontSize(12).TextColor(t.TextMuted).MaxLines(2)
			})
		})
		ui.Divider(c)
		s.playerView(c)
	})
}

func (s *app) playerView(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Padding(14, 20).Gap(8).Background(t.Surface).Children(func() {
		ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
			s.coverView(c, 52)
			ui.Column(c).Width(215).Gap(4).Children(func() {
				name, artist := "等待播放", "网易云 · Bilibili · 本地音乐"
				if s.current != nil {
					name = s.current.Name
					artist = s.current.Artist
				}
				ui.Text(c, name).Bold().SingleLine().Ellipsis("…")
				ui.Text(c, artist).FontSize(12).TextColor(t.TextMuted).SingleLine().Ellipsis("…")
			})
			ui.Spacer(c)
			ui.Button(c, "上一首").Disabled(len(s.queue) == 0).OnClick(func() { s.step(-1) })
			label := "播放"
			if s.playing {
				label = "暂停"
			}
			if s.loading {
				label = "加载中…"
			}
			ui.PrimaryButton(c, label).Width(86).Disabled(s.player == nil).OnClick(s.toggle)
			ui.Button(c, "下一首").Disabled(len(s.queue) == 0).OnClick(func() { s.step(1) })
			ui.Spacer(c)
			ui.Text(c, "音量").FontSize(12)
			v := s.cfg.Volume
			if ui.Slider(c.Key("volume"), &v, 0, 1).Width(95).Label("音量").Changed() {
				s.volume(v)
			}
		})
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, clockText(s.position)).FontSize(11).Width(38)
			position := s.position.Seconds()
			if ui.Slider(c.Key("seek"), &position, 0, max(1, s.duration.Seconds())).Grow(1).Label("播放进度").Disabled(s.duration <= 0 || s.player == nil).Changed() {
				s.seek(time.Duration(position * float64(time.Second)))
			}
			ui.Text(c, clockText(s.duration)).FontSize(11).Width(38)
		})
	})
}

func (s *app) settingsView(c *ui.Context) {
	t := c.Theme()
	ui.Text(c, "设置").FontSize(26).Bold()
	ui.Text(c, "音乐文件夹").Bold()
	ui.Row(c).Gap(8).Children(func() {
		ui.TextInput(c.Key("dir"), &s.cfg.MusicDir).Grow(1).Label("音乐文件夹")
		ui.Button(c, "浏览文件夹").OnClick(s.pickFolder)
	})
	ui.Text(c, "FFmpeg").Bold()
	ui.Text(c, "用于音频解码。可从系统路径查找，或手动选择可执行文件。").FontSize(12).TextColor(t.TextMuted)
	ui.Row(c).Gap(8).Children(func() {
		ui.TextInput(c.Key("ffmpeg"), &s.cfg.FFmpeg).Grow(1).Label("FFmpeg 路径").Placeholder("自动查找")
		ui.Button(c, "浏览文件").OnClick(s.pickFFmpeg)
	})
	ui.Text(c, "网易云 Cookie").Bold()
	ui.TextInput(c.Key("netease-cookie"), &s.cfg.NeteaseCookie).Password().Label("网易云 Cookie")
	ui.Text(c, "Bilibili Cookie").Bold()
	ui.TextInput(c.Key("bili-cookie"), &s.cfg.BilibiliCookie).Password().Label("Bilibili Cookie")
	ui.Text(c, "Cookie 只保存在这台电脑上，用于访问你已登录账号可用的内容。").FontSize(12).TextColor(t.TextMuted)
	ui.Row(c).Gap(10).Children(func() {
		ui.PrimaryButton(c, "保存设置").OnClick(func() { s.status = "设置已保存"; s.save() })
		ui.Button(c, "检查更新").Disabled(s.updateBusy).OnClick(s.checkUpdate)
	})
	ui.Spacer(c)
	ui.Text(c, s.status).FontSize(12).TextColor(t.TextMuted).MaxLines(3)
}

func (s *app) coverView(c *ui.Context, size float32) {
	if s.cover != nil {
		ui.Image(c, s.cover).Size(size, size).Fit(ui.Cover).Radius(7)
		return
	}
	ui.Column(c).Size(size, size).Center().Background(accent).Radius(7).Children(func() { ui.Text(c, "♫").FontSize(size * 0.5).TextColor(ui.Hex("#ffffff")) })
}

func (s *app) miniView(c *ui.Context) {
	t := c.Theme()
	box := ui.Row(c.Key("mini")).Fill().Padding(7).Gap(10).AlignItems(ui.Center).Background(t.Surface)
	box.Children(func() {
		ui.Column(c.Key("cover-drag")).Size(48, 48).DragWindow().Children(func() { s.coverView(c, 48) })
		ui.Column(c.Key("caption")).Grow(1).Clip().DragWindow().Gap(3).Children(func() {
			text, elapsed := s.miniCaption()
			// Keep an accessible text node; paint overflow without truncating
			// a synchronized line or moving the surrounding window controls.
			ui.Box(c).Height(23).Label(text).Clip().Draw(func(p *ui.Painter, r ui.Rect) {
				span := ui.Span{Text: text, Size: 14, Color: t.Text, Weight: 600}
				width, height := p.MeasureText(0, span)
				shift := float32(0)
				if s.playing && width > r.W {
					shift = float32(math.Mod(max(0, elapsed.Seconds()-0.8)*28, float64(width+32)))
					p.AnimationFrameRate(30)
				}
				p.RichText(r.X-shift, r.Y+(r.H-height)/2, 0, span)
				if shift > 0 {
					p.RichText(r.X-shift+width+32, r.Y+(r.H-height)/2, 0, span)
				}
			})
			artist := "MusicDL"
			if s.current != nil {
				artist = s.current.Artist
			}
			ui.Text(c, artist).FontSize(10).TextColor(t.TextMuted).SingleLine().Ellipsis("…")
		})
		ui.Column(c).Gap(2).Children(func() {
			ui.Button(c.Key("show"), "↗").Label("打开主窗口").Width(28).OnClick(func() {
				if s.win != nil {
					s.win.Show()
					s.win.Focus()
				}
			})
			ui.Button(c.Key("hide"), "×").Label("隐藏悬浮窗").Width(28).OnClick(func() {
				if s.mini != nil {
					s.mini.Hide()
				}
			})
		})
	})
	// The hover controls replace the caption area, leaving the cover and
	// window buttons in place. They are also present while playback is paused.
	if box.Hovered() || !s.playing {
		// A compact overlay anchored to the content, built with native UI.
		ui.Row(c.Key("mini-controls")).Absolute().Left(65).Right(42).Top(8).Height(48).Gap(4).AlignItems(ui.Center).Background(t.Surface).Children(func() {
			ui.Button(c, "‹").Label("上一首").Width(28).Disabled(len(s.queue) == 0).OnClick(func() { s.step(-1) })
			label := "▶"
			if s.playing {
				label = "Ⅱ"
			}
			ui.Button(c, label).Label("播放或暂停").Width(30).Disabled(s.player == nil).OnClick(s.toggle)
			ui.Button(c, "›").Label("下一首").Width(28).Disabled(len(s.queue) == 0).OnClick(func() { s.step(1) })
			ui.Column(c).Grow(1).Gap(2).Children(func() {
				ui.Text(c, caption(s.current)).FontSize(10).SingleLine().Ellipsis("…")
				v := s.cfg.Volume
				if ui.Slider(c, &v, 0, 1).Label("悬浮窗音量").Height(18).Changed() {
					s.volume(v)
				}
			})
		})
	}
}

func (s *app) miniCaption() (string, time.Duration) {
	if s.current == nil {
		return "MusicDL · 随时听歌", 0
	}
	if s.playing {
		if line, elapsed := lyrics.Current(s.lines, s.position); line != "" {
			return line, elapsed
		}
	}
	return caption(s.current), s.position
}

func clockText(d time.Duration) string {
	seconds := max(0, int(d.Seconds()))
	return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
}
