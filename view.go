package main

import (
	"fmt"
	"math"
	"runtime"
	"strings"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo/ui"
)

var accent = ui.Hex("#10b981")
var white = ui.Hex("#ffffff")
var textColor = ui.Hex("#2d3748")
var muted = ui.Hex("#718096")
var border = ui.Hex("#e2e8f0")

func desktopFont() string {
	if runtime.GOOS == "windows" {
		return "Segoe UI, Roboto, Helvetica Neue, Arial, Microsoft YaHei, sans-serif"
	}
	return "system-ui"
}

func (s *app) view(c *ui.Context) {
	initAssets()
	theme := ui.LightTheme()
	theme.Accent = accent
	theme.Text = textColor
	theme.TextMuted = muted
	theme.Border = border
	theme.FontSize = 16
	theme.Font = desktopFont()
	c.SetTheme(theme)
	w, h := c.Size()
	ui.Box(c).Fill().Lang("zh-CN").Background(ui.Hex("#eef2f5")).Draw(func(p *ui.Painter, r ui.Rect) {
		r.H = 600
		p.Image(pageBackdrop(int(r.W)), r, ui.Contain)
	}).Children(func() {
		if s.lyricOpen {
			s.queryFocused = false
			s.lyricsView(c)
			return
		}
		content := ui.Box(c).Fill()
		if s.modal != "" {
			content.Blur(5).Cache()
		}
		content.Children(func() {
			ui.Scroll(c.Key("page-scroll")).Fill().TrackScroll(&s.scroll).Children(func() {
				ui.Column(c).Width(min(900, w-40)).AlignSelf(ui.Center).Padding(40, 0, 150, 0).Children(func() {
					s.searchBox(c)
					s.browseCurrentView(c)
					if s.page == "playlists" {
						s.localPlaylistsView(c)
					} else if s.page == "library" && s.collectionKind == "categories" {
						s.categoriesView(c)
					} else if s.page == "library" && s.collectionKind == "ranks" {
						s.rankingsView(c)
					} else if s.page == "library" && (s.collectionKind == "recommend" || s.collectionKind == "mine") {
						s.sourcePlaylistsView(c)
					} else if s.page == "library" && len(s.categories) > 0 {
						s.categoriesView(c)
					} else if s.page == "library" && len(s.collections) > 0 {
						s.collectionsView(c)
					} else {
						s.resultsView(c)
					}
				})
			})
			s.toolbar(c, h)
			s.playerView(c)
		})
		if s.modal != "" {
			s.modalView(c, w, h)
		}
	})
	s.statusToast(c)
	s.handleShortcuts(c)
}
func (s *app) searchBox(c *ui.Context) {
	ui.Column(c.Key("search-box")).Background(white.Alpha(.95)).Border(1, white.Alpha(.5)).Radius(20).Padding(30).Shadow(0, 4, 6, 0, ui.RGBA(0, 0, 0, .05)).Margin(0, 0, 30, 0).Children(func() {
		ui.Row(c).Justify(ui.Center).Gap(14).Effect(headingGradient{}).Cache().Height(45).Margin(0, 0, 25, 0).AlignItems(ui.Center).Children(func() {
			glyph(c, "music", 35, white)
			ui.Text(c, "music-dl").FontSize(35.2).LetterSpacing(-1).FontWeight(800).TextColor(white)
		})
		ui.Row(c).Justify(ui.Center).Gap(20).Height(25).Margin(0, 0, 20, 0).AlignItems(ui.Center).Children(func() {
			for _, o := range []struct{ id, label string }{{"song", "单曲搜索"}, {"playlist", "歌单搜索"}, {"album", "专辑搜索"}} {
				color := muted
				if s.searchKind == o.id {
					color = accent
				}
				ui.RadioBase(c.Key(o.id), &s.searchKind, o.id).Label(o.label).Gap(8).AlignItems(ui.Center).Children(func() {
					bg, bd := white, ui.Hex("#a0aec0")
					if s.searchKind == o.id {
						bg = accent
						bd = accent
					}
					ui.Box(c).Size(13, 13).Radius(7).Border(1, bd).Background(bg).Center().Children(func() {
						if s.searchKind == o.id {
							ui.Box(c).Size(5, 5).Radius(3).Background(white)
						}
					})
					ui.Text(c, o.label).FontSize(16).FontWeight(600).TextColor(color)
				})
			}
		})
		ui.Row(c).Gap(10).Margin(0, 0, 25, 0).Children(func() {
			input := ui.TextInput(c.Key("query"), &s.query).Label("搜索关键词").Placeholder("输入关键词或粘贴链接，支持单曲 / 歌单 / 专辑").PlaceholderColor(ui.Hex("#757575")).Grow(1).Height(50).Padding(14, 24).Radius(50).Border(2, border).Background(ui.Hex("#f8fafc")).FontSize(16)
			s.queryFocused = input.Focused()
			input.Transition(ui.ElementTransition{Colors: true, Duration: 300 * time.Millisecond})
			if input.Focused() {
				input.Border(2, accent).Background(white).Shadow(0, 0, 0, 4, accent.Alpha(.15))
			}
			if input.Submitted() {
				s.search()
			}
			searchButton := ui.Button(c, " ").Label("搜索").Width(80).Height(50).Radius(50).Border(0, ui.Transparent).Gradient(ui.Hex("#34d399"), ui.Hex("#059669"), 135).Shadow(0, 4, 12, 0, accent.Alpha(.3)).Disabled(s.searching).OnClick(s.search)
			buttonMotion(searchButton, 1, 2)
			searchButton.Children(func() {
				if s.searching {
					glyph(c, "spinner", 18, white).Rotate(float32(math.Mod(float64(c.Now().UnixMilli())/700*360, 360)))
					c.After(33 * time.Millisecond)
				} else {
					glyph(c, "search", 18, white)
				}
			})
		})
		ui.Row(c).Wrap().Justify(ui.Center).Gap(15).Margin(0, 0, 20, 0).Children(func() {
			pill(c, "fire", "每日推荐歌单", "#ff9a9e", "#fecfef", func() { s.browse("recommend", "netease", "") })
			pill(c, "folder", "本地歌单", "#f68084", "#f68084", s.openSavedPlaylists)
			pill(c, "music", "本地音乐", "#f68084", "#f68084", s.localMusic)
			pill(c, "grid", "歌单分类", "#38bdf8", "#6366f1", func() { s.browse("categories", "netease", "") })
			pill(c, "ranking", "排行榜", "#fb923c", "#ef4444", func() { s.browse("ranks", "netease", "") })
			pill(c, "heart", "我的歌单", "#34d399", "#06b6d4", func() { s.browse("mine", "netease", "") })
		})
		sourceHeader := ui.Row(c).AlignItems(ui.Center)
		if !s.sourceCollapsed {
			sourceHeader.Padding(0, 0, 8, 0).Margin(0, 0, 12, 0).BorderWidth(0, 0, 1, 0).BorderColor(ui.Hex("#edf2f7"))
		}
		sourceHeader.Children(func() {
			ui.Button(c, "").Padding(0).Background(ui.Transparent).Border(0, ui.Transparent).Label("搜索源设置").OnClick(func() { s.sourceCollapsed = !s.sourceCollapsed }).Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "搜索源设置").FontSize(14.4).Bold().TextColor(muted)
					angle := float32(0)
					if s.sourceCollapsed {
						angle = 180
					}
					glyph(c, "caret-up", 11, muted).AnimatePose(ui.Pose{Rotate: angle}, hoverAnimation)
				})
			})
			ui.Spacer(c)
			ui.Button(c, "全选").FontSize(12).TextColor(muted).Background(ui.Transparent).Border(0, ui.Transparent).OnClick(func() {
				for _, src := range []string{"netease", "bilibili", "local"} {
					s.selected[src] = true
				}
			})
			ui.Button(c, "清空").FontSize(12).TextColor(muted).Background(ui.Transparent).Border(0, ui.Transparent).OnClick(func() { s.selected = map[string]bool{} })
		})
		if !s.sourceCollapsed {
			ui.Grid(c).Columns(5).Gap(12).Children(func() {
				for _, src := range []struct{ id, label string }{{"netease", "网易云音乐"}, {"bilibili", "Bilibili"}, {"local", "本地音乐"}} {
					enabled := s.searchKind == "song" || (s.searchKind == "playlist" && src.id != "local") || (s.searchKind == "album" && src.id == "netease")
					active := enabled && s.selected[src.id]
					bg, fg, bd := white, ui.Hex("#4a5568"), border
					if active {
						bg = ui.Hex("#eff3ff")
						fg = accent
						bd = accent
					}
					button := ui.Button(c.Key("source-"+src.id), "").Label(src.label).Disabled(!enabled).Padding(10, 8).Height(58).Radius(12).Background(bg).Border(2, bd).OnClick(func() { s.selected[src.id] = !s.selected[src.id] }).Children(func() {
						ui.Column(c).AlignItems(ui.Center).Gap(3).Children(func() {
							ui.Text(c, src.id).FontSize(14).Bold().TextColor(fg)
							ui.Text(c, src.label).FontSize(11).TextColor(ui.Hex("#a0aec0"))
						})
					})
					if active {
						button.Shadow(0, 4, 6, 0, accent.Alpha(.15))
					}
				}
			})
		}
	})
}
func (s *app) resultsView(c *ui.Context) {
	tracks := s.shownTracks()
	local := s.source == "local" && s.page == "library"
	if len(tracks) == 0 && !local {
		if s.searching {
			ui.Text(c, "正在查找…").TextColor(muted).AlignSelf(ui.Center)
		}
		return
	}
	size := s.cfg.PageSize
	if size < 20 {
		size = 30
	}
	pages := max(1, (len(tracks)+size-1)/size)
	s.pageNumber = max(1, min(pages, s.pageNumber))
	start := (s.pageNumber - 1) * size
	end := min(start+size, len(tracks))
	ui.Row(c).Background(white.Alpha(.85)).Border(1, ui.Hex("#edf2f7")).Radius(12).Padding(10, 12).Margin(0, 0, 15, 0).Shadow(0, 4, 6, 0, ui.RGBA(0, 0, 0, .05)).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Gap(6).Children(func() {
			title := fmt.Sprintf("♫ 找到  %d  首歌曲", len(tracks))
			if local {
				title = fmt.Sprintf("♫ 本地音乐（共 %d 首）", len(tracks))
			}
			if s.collectionTitle != "" {
				title = fmt.Sprintf("♫ %s（共 %d 首）", s.collectionTitle, len(tracks))
			}
			switch s.page {
			case "favorites":
				title = "♥ 我的收藏"
			case "history":
				title = "◷ 播放历史"
			case "queue":
				title = "♫ 播放队列"
			}
			if s.collectionTitle == "" && s.page == "library" && !local {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					glyph(c, "music", 14, textColor)
					ui.Text(c, "找到").FontSize(14).FontWeight(700)
					ui.Text(c, fmt.Sprint(len(tracks))).FontSize(14).FontWeight(700).TextColor(accent)
					ui.Text(c, "首歌曲").FontSize(14).FontWeight(700)
				})
			} else {
				ui.Text(c, title).FontSize(14).FontWeight(700)
			}
			if len(tracks) > 0 {
				ui.Text(c, fmt.Sprintf("当前第 %d / %d 页，显示 %d - %d / %d", s.pageNumber, pages, start+1, end, len(tracks))).FontSize(12).TextColor(muted)
			}
		})
		ui.Spacer(c)
		if local {
			ui.Button(c, "上传文件").FontSize(12).MarginX(4).Radius(8).OnClick(s.pickLocalFiles)
			ui.Button(c, "上传文件夹").FontSize(12).MarginX(4).Radius(8).OnClick(s.pickFolder)
		}
		if s.currentCollection != nil && s.page == "library" {
			ui.Button(c, "下载整个歌单").Radius(18).TextColor(accent).FontSize(12).OnClick(func() { s.addDownloads(tracks) })
			ui.Button(c, "导入本地").Radius(18).TextColor(accent).FontSize(12).OnClick(func() { s.importCollection(*s.currentCollection, tracks) })
		}
		s.listTools(c)
	})
	if s.batch {
		ui.Row(c).Padding(12).Background(white).Radius(12).Gap(8).Margin(0, 0, 15, 0).AlignItems(ui.Center).Children(func() {
			ui.Button(c, "全选").OnClick(func() {
				for _, t := range tracks {
					s.checked[t.Key()] = true
				}
			})
			ui.Button(c, "清空").OnClick(func() { s.checked = map[string]bool{} })
			ui.Text(c, fmt.Sprintf("已选 %d", len(s.selectedTracks()))).FontSize(12).TextColor(muted)
			ui.Spacer(c)
			ui.Button(c, "批量收藏").OnClick(func() { s.pendingTracks = s.selectedTracks(); s.modal = "collections" })
			ui.Button(c, "批量下载").OnClick(func() { s.addDownloads(s.selectedTracks()) })
		})
	}
	ui.Column(c).Gap(15).Children(func() {
		for i := start; i < end; i++ {
			s.trackCard(c, tracks[i], i, tracks)
		}
	})
	if pages > 1 {
		ui.Row(c).Justify(ui.Center).Gap(12).Margin(20, 0, 0, 0).Children(func() {
			ui.Button(c, "上一页").Disabled(s.pageNumber == 1).OnClick(func() { s.pageNumber--; s.scroll.ScrollTo(0, 350, ui.Smooth) })
			ui.Text(c, fmt.Sprintf("%d / %d", s.pageNumber, pages))
			ui.Button(c, "下一页").Disabled(s.pageNumber == pages).OnClick(func() { s.pageNumber++; s.scroll.ScrollTo(0, 350, ui.Smooth) })
		})
	}
}
func (s *app) trackCard(c *ui.Context, t library.Track, i int, tracks []library.Track) {
	bg, bd := white, ui.Transparent
	active := s.current != nil && s.current.Key() == t.Key() && s.playing
	if active {
		bg = ui.Hex("#eff3ff")
		bd = accent
	}
	row := ui.Row(c.Key(t.Key())).MinHeight(112).Padding(15).AlignItems(ui.Center).Background(bg).Border(2, bd).Radius(16).Shadow(0, 4, 6, 0, ui.RGBA(0, 0, 0, .05))
	if active {
		row.Shadow(0, 8, 20, 0, accent.Alpha(.25))
		row.AnimatePose(ui.Pose{Scale: 1.01}, hoverAnimation)
	} else {
		cardMotion(row, 3)
	}
	row.Children(func() {
		if s.batch {
			selected := s.checked[t.Key()]
			if ui.Checkbox(c, &selected, "").Label("选择 " + t.Name).Changed() {
				if selected {
					s.checked[t.Key()] = true
				} else {
					delete(s.checked, t.Key())
				}
			}
		}
		ui.Box(c).Size(64, 64).Margin(0, 15, 0, 0).OnClick(func() {
			if s.current == nil || s.current.Key() != t.Key() {
				s.play(i, tracks)
			}
			s.lyricOpen = true
		}).Children(func() { s.trackCover(c, t, 64) })
		ui.Column(c).Grow(1).Margin(0, 10, 0, 0).Children(func() {
			ui.Text(c, t.Name).FontSize(16).FontWeight(600).Margin(0, 0, 6, 0).SingleLine().Ellipsis("…").OnClick(func() {
				if t.Link != "" {
					c.OpenURL(t.Link)
				}
			})
			ui.Row(c).Gap(4).AlignItems(ui.Center).Margin(0, 0, 8, 0).Children(func() {
				glyph(c, "user", 11, muted).Font("MusicDL Outline")
				ui.Text(c, t.Artist).FontSize(13).LineHeight(1.45).TextColor(muted).SingleLine().Ellipsis("…").OnClick(func() {
					s.query = t.Artist
					s.selected = map[string]bool{t.Source: true}
					s.searchKind = "song"
					s.search()
				})
				if t.Album != "" {
					ui.Text(c, "·").FontSize(13).TextColor(muted)
					ui.Text(c, t.Album).FontSize(13).TextColor(muted).SingleLine().Ellipsis("…").OnClick(func() {
						if t.Source == "netease" {
							s.query = t.Album
							s.browse("album", t.Source, t.AlbumID)
						}
					})
				}
			})
			ui.Row(c).Gap(6).Children(func() {
				for n, tag := range []string{t.Source, t.FormatDuration(), t.FormatSize(), t.FormatBitrate()} {
					if tag != "" && tag != "-" && (n != 1 || t.Duration > 0) && (n != 2 || t.Size > 0) && (n != 3 || t.Bitrate > 0) {
						bg, fg := ui.Hex("#edf2f7"), muted
						if n == 0 {
							tag = strings.ToUpper(tag)
							bg, fg = ui.Hex("#ebf8ff"), ui.Hex("#3182ce")
							if t.Source == "local" {
								tag = "本地"
								bg, fg = ui.Hex("#fff5f5"), ui.Hex("#e53e3e")
							}
						}
						ui.Text(c, tag).FontSize(10).FontWeight(600).Padding(3, 8).Radius(6).Background(bg).TextColor(fg)
					}
				}
			})
		})
		ui.Row(c).Gap(8).Children(func() {
			playIcon, playLabel := "play", "播放 "
			if active {
				playIcon, playLabel = "stop", "停止 "
			}
			play := colorIconButton(c, "play", playIcon, playLabel+t.Name, 36, white, false).Gradient(ui.Hex("#34d399"), ui.Hex("#059669"), 135).Shadow(0, 4, 10, 0, accent.Alpha(.3)).OnClick(func() {
				if active {
					s.toggle()
					s.seek(0)
				} else {
					s.play(i, tracks)
				}
			})
			if active {
				play.Gradient(ui.Hex("#ef4444"), ui.Hex("#ef4444"), 0).Shadow(0, 4, 10, 0, ui.Hex("#ef4444").Alpha(.3))
			}
			if s.source != "local" || s.page != "library" {
				colorIconButton(c, "switch", "switch", "换源 "+t.Name, 36, ui.Hex("#9a3412"), false).Background(ui.Hex("#fff7ed")).Border(1, ui.Hex("#fed7aa")).OnClick(func() {
					s.query = t.Name + " " + t.Artist
					s.selected = map[string]bool{"netease": true, "bilibili": true}
					s.searchKind = "song"
					s.search()
				})
			}
			icon, label, outline := "heart", "收藏 "+t.Name, !s.isFavorite(t)
			if s.page == "collection" || s.page == "favorites" {
				icon, label, outline = "trash", "移出歌单 "+t.Name, false
			}
			heart := colorIconButton(c, "favorite", icon, label, 36, ui.Hex("#fc8181"), outline).Background(ui.Hex("#fff5f5")).Border(1, ui.Hex("#fed7d7")).OnClick(func() {
				if s.page == "collection" || s.page == "favorites" {
					s.removeFromCollection(t)
					return
				}
				s.pendingTracks = []library.Track{t}
				s.modal = "collections"
			})
			if s.isFavorite(t) {
				heart.Background(ui.Hex("#ffe4e6"))
			}
			more := iconButton(c, "more", "more", "更多操作 "+t.Name, 36).Background(ui.Hex("#f7fafc")).Border(1, border)
			if more.Clicked() {
				s.songMenuOpen = s.songMenu != t.Key() || !s.songMenuOpen
				s.songMenu = t.Key()
			}
			if s.songMenu == t.Key() {
				ui.PopoverBase(c, more, &s.songMenuOpen, func(panel ui.Element) {
					panel.Transition(panelTransition).AttachTo(more, ui.AnchorBottomRight, ui.AnchorTopRight).Margin(8, 0, 0, 0).Padding(8).Border(1, border).Radius(13).Background(white.Alpha(.98)).Shadow(0, 12, 30, 0, ui.RGBA(15, 23, 42, .18))
					ui.Grid(c).Columns(2).Gap(7).Children(func() {
						if t.Source != "local" {
							iconButton(c, "download", "download", "下载 "+t.Name, 36).Border(1, border).OnClick(func() { s.songMenuOpen = false; s.addDownloads([]library.Track{t}) })
							iconButton(c, "save-as", "cloud", "另存为 "+t.Name, 36).Border(1, border).OnClick(func() { s.songMenuOpen = false; s.downloadToFolder(t) })
						}
						iconButton(c, "lyrics", "file", "歌词 "+t.Name, 36).Border(1, border).OnClick(func() { s.songMenuOpen = false; s.saveLyrics(t) })
						colorIconButton(c, "cover", "image", "保存封面 "+t.Name, 36, muted, true).Border(1, border).OnClick(func() { s.songMenuOpen = false; s.saveCover(t) })
						if t.Source == "local" {
							colorIconButton(c, "delete", "trash", "删除本地音乐 "+t.Name, 36, ui.Hex("#e53e3e"), false).Background(ui.Hex("#fff5f5")).Border(1, ui.Hex("#fed7d7")).OnClick(func() { s.songMenuOpen = false; s.trashLocalTrack(t) })
						}
					})
				})
			}
		})
	})
}
func (s *app) trackCover(c *ui.Context, t library.Track, size float32) {
	if img := s.artwork(t); img != nil {
		ui.Image(c, img).Size(size, size).Fit(ui.Cover).Radius(10)
	} else {
		ui.Box(c).Size(size, size).Background(ui.Hex("#eeeeee")).Radius(10).Center().Children(func() { ui.Text(c, "🎵").FontSize(size * .375) })
	}
}
func (s *app) collectionsView(c *ui.Context) {
	width, _ := c.Size()
	s.collectionResultHeader(c, len(s.collections), s.searchKind)
	s.collectionCards(c, min(900, width-40), 20, 180, false)
}
func (s *app) toolbar(c *ui.Context, h float32) {
	ui.Column(c).Absolute().Right(14).Top(max(20, (h-320)/2)).Gap(6).AlignItems(ui.Center).Children(func() {
		for _, b := range []struct {
			key, icon, label string
			fn               func()
		}{{"settings", "gear", "系统设置", func() { s.modal = "settings" }}, {"local", "folder", "本地音乐", s.localMusic}, {"downloads", "history", "下载管理", func() { s.modal = "downloads" }}, {"history", "clock", "播放历史", func() { s.modal = "history"; s.utilityPage = 1 }}, {"mini", "mini", "悬浮窗", s.toggleMini}} {
			button := iconButton(c, b.key, b.icon, b.label, 36).Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, .12)).OnClick(b.fn)
			buttonMotion(button, 1.1, 0)
		}
		ui.Divider(c).Width(20).MarginY(4)
		iconButton(c, "top", "up", "回到顶部", 36).OnClick(func() { s.scroll.ScrollTo(0, 0, ui.Smooth) })
		iconButton(c, "current", "headphones", "定位当前播放歌曲", 36).OnClick(func() {
			s.page = "queue"
			s.collections = nil
			s.categories = nil
			s.pageNumber = 1
			s.scroll.ScrollTo(0, 350, ui.Smooth)
		})
	})
}
func (s *app) cycleMode() {
	modes := []string{"", "loop", "one", "shuffle"}
	for i, m := range modes {
		if m == s.cfg.PlayMode {
			s.cfg.PlayMode = modes[(i+1)%len(modes)]
			break
		}
	}
	labels := map[string]string{"": "顺序播放", "loop": "列表循环", "one": "单曲循环", "shuffle": "随机播放"}
	s.status = labels[s.cfg.PlayMode]
	s.save()
}
