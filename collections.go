package main

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo/ui"
	"github.com/guohuiyuan/music-lib/model"
)

func (s *app) openSavedPlaylists() {
	s.page = "playlists"
	s.collections = nil
	s.categories = nil
	s.collectionTitle = ""
	s.pendingTracks = nil
	s.playlistError = ""
	s.modal = ""
	s.scroll = ui.ScrollState{}
}
func (s *app) editPlaylist(p *library.Playlist) {
	s.editPlaylistID = ""
	s.playlistName = ""
	s.playlistDescription = ""
	s.playlistCover = ""
	s.playlistError = ""
	if p != nil {
		s.editPlaylistID = p.ID
		s.playlistName = p.Name
		s.playlistDescription = p.Description
		s.playlistCover = p.Cover
	}
	s.modal = "edit-playlist"
}
func (s *app) storePlaylist(p library.Playlist) bool {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		s.playlistError = "请输入歌单名称"
		return false
	}
	if utf8.RuneCountInString(p.Name) > 30 {
		s.playlistError = "歌单名称最多 30 个字"
		return false
	}
	if p.Cover != "" {
		u, e := url.Parse(p.Cover)
		if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			s.playlistError = "封面请填写有效的图片 URL"
			return false
		}
	}
	if p.ID == "" {
		p.ID = fmt.Sprint(time.Now().UnixNano())
	}
	cfg := s.cfg
	cfg.Playlists = append([]library.Playlist(nil), s.cfg.Playlists...)
	replaced := false
	for i := range cfg.Playlists {
		if cfg.Playlists[i].ID == p.ID {
			cfg.Playlists[i] = p
			replaced = true
			break
		}
	}
	if !replaced {
		cfg.Playlists = append(cfg.Playlists, p)
	}
	if err := library.SaveConfig(s.configPath, cfg); err != nil {
		s.playlistError = "歌单保存失败：" + err.Error()
		s.status = s.playlistError
		return false
	}
	s.cfg = cfg
	s.playlistError = ""
	return true
}
func (s *app) saveEditedPlaylist() {
	p := library.Playlist{ID: s.editPlaylistID, Name: s.playlistName, Description: s.playlistDescription, Cover: s.playlistCover, Tracks: append([]library.Track(nil), s.pendingTracks...)}
	if p.ID != "" {
		for _, old := range s.cfg.Playlists {
			if old.ID == p.ID {
				p.Tracks = old.Tracks
				p.Source = old.Source
				p.ExternalID = old.ExternalID
				p.Kind = old.Kind
				p.Link = old.Link
			}
		}
	}
	if !s.storePlaylist(p) {
		return
	}
	s.pendingTracks = nil
	s.modal = ""
	s.status = "歌单已保存"
	if s.page != "playlists" {
		s.openSavedPlaylists()
	}
}
func (s *app) editPlaylistView(c *ui.Context) {
	for _, field := range []struct {
		label, placeholder string
		value              *string
	}{{"歌单名称 *", "例如: 睡前听的歌", &s.playlistName}, {"描述", "简单描述一下...", &s.playlistDescription}, {"封面图片 (URL)", "留空则自动生成随机封面", &s.playlistCover}} {
		ui.Column(c).Gap(4).Margin(0, 0, 15, 0).Children(func() {
			ui.Text(c, field.label).FontSize(12).FontWeight(600).TextColor(muted)
			ui.TextInput(c, field.value).Label("编辑"+strings.TrimSuffix(field.label, " *")).Placeholder(field.placeholder).FontSize(14).Height(38).Padding(10).Radius(8).Border(1, border).Background(white)
		})
	}
	if s.playlistError != "" {
		ui.Text(c, s.playlistError).FontSize(12).TextColor(ui.Hex("#e53e3e")).MaxLines(3)
	}
	ui.Button(c, "保存歌单").FillWidth().Height(45).Margin(10, 0, 0, 0).Radius(10).FontSize(15).FontWeight(600).TextColor(white).Gradient(ui.Hex("#34d399"), ui.Hex("#059669"), 135).OnClick(s.saveEditedPlaylist)
}
func (s *app) localPlaylistsView(c *ui.Context) {
	playlists := append([]library.Playlist(nil), s.cfg.Playlists...)
	if len(s.cfg.Favorites) > 0 {
		playlists = append([]library.Playlist{{ID: "favorites", Name: "我的收藏", Tracks: s.cfg.Favorites}}, playlists...)
	}
	ui.Row(c).Padding(10, 12).Margin(0, 0, 20, 0).Background(white.Alpha(.85)).Border(1, ui.Hex("#edf2f7")).Radius(12).AlignItems(ui.Center).Children(func() {
		glyph(c, "folder", 16, textColor).Margin(0, 8, 0, 0)
		ui.Text(c, fmt.Sprintf("本地歌单（共 %d 个）", len(playlists))).FontSize(16).Bold().Grow(1)
		ui.PrimaryButton(c, "+ 新建歌单").Label("新建歌单").Radius(20).Padding(8, 18).FontSize(14).OnClick(func() { s.editPlaylist(nil) })
	})
	if len(playlists) == 0 {
		ui.Column(c).Padding(45).Gap(12).Radius(16).Background(white).AlignItems(ui.Center).Children(func() {
			glyph(c, "folder", 36, muted)
			ui.Text(c, "暂无歌单").FontSize(16).TextColor(muted)
			ui.Text(c, "新建歌单，或从搜索结果导入歌单").FontSize(13).TextColor(muted)
		})
		return
	}
	width, _ := c.Size()
	contentWidth := min(900, width-40)
	cols := max(1, int((contentWidth+20)/200))
	size := (contentWidth - float32(cols-1)*20) / float32(cols)
	s.collectionResultHeader(c, len(playlists), "playlist")
	ui.Grid(c).Columns(cols).Gap(20).Children(func() {
		for _, p := range playlists {
			card := ui.Column(c.Key(p.ID)).Background(white).Radius(16).Clip().Shadow(0, 4, 6, 0, ui.RGBA(0, 0, 0, .05))
			cardMotion(card, 5)
			card.Children(func() {
				ui.Box(c).Label(func() string {
					if p.ID == "favorites" {
						return "打开我的收藏"
					}
					return "打开歌单 " + p.Name
				}()).Size(size, size).OnClick(func() { s.openLocalPlaylist(p) }).Children(func() { s.playlistCoverView(c, p.Name, p.Cover, p.ID, size) })
				ui.Column(c).Padding(12).Gap(6).Children(func() {
					ui.Text(c, p.Name).FontSize(14).Bold().MaxLines(2).OnClick(func() { s.openLocalPlaylist(p) })
					subtitle := p.Description
					if subtitle == "" {
						subtitle = "自建歌单"
					}
					if p.Source != "" {
						subtitle = "外部导入 · " + p.Source
					}
					ui.Text(c, subtitle).FontSize(12).TextColor(muted).MaxLines(2)
					ui.Row(c).AlignItems(ui.Center).Children(func() {
						ui.Text(c, fmt.Sprintf("共 %d 首", len(p.Tracks))).FontSize(11).TextColor(muted).Grow(1)
						if p.ID != "favorites" {
							iconButton(c, "edit", "edit", "编辑歌单 "+p.Name, 28).Radius(6).Border(1, border).OnClick(func() { s.editPlaylist(&p) })
							iconButton(c, "delete", "trash", "删除歌单 "+p.Name, 28).Radius(6).Border(1, border).OnClick(func() { s.confirmDeletePlaylist(p.ID, p.Name) })
						}
					})
					ui.Button(c, "下载整个歌单").FillWidth().Height(32).FontSize(12).Radius(8).OnClick(func() { s.addDownloads(p.Tracks) })
				})
			})
		}
	})
}
func (s *app) playlistCoverView(c *ui.Context, name, cover, id string, size float32) {
	if cover != "" {
		if img := s.artwork(library.Track{Song: model.Song{Source: "netease", ID: "playlist:" + id, Cover: cover}}); img != nil {
			ui.Image(c, img).Size(size, size).Fit(ui.Cover)
			return
		}
	}
	colors := [][2]string{{"#34d399", "#059669"}, {"#38bdf8", "#6366f1"}, {"#ff9a9e", "#f68084"}}
	choice := 0
	for _, r := range id {
		choice = (choice + int(r)) % len(colors)
	}
	ui.Column(c).Size(size, size).Gradient(ui.Hex(colors[choice][0]), ui.Hex(colors[choice][1]), 135).AlignItems(ui.Center).Justify(ui.Center).Gap(12).Children(func() {
		glyph(c, "music", min(42, size*.5), white.Alpha(.85))
		if size >= 100 {
			ui.Text(c, name).FontSize(18).Bold().TextColor(white).TextAlign(ui.Center).MaxLines(2).PaddingX(18)
		}
	})
}
func (s *app) openSavedPlaylist(p library.Playlist) {
	s.page = "collection"
	s.localPlaylistID = p.ID
	s.collectionTitle = p.Name
	s.currentCollection = nil
	s.categories = nil
	s.collections = nil
	s.pageNumber = 1
	s.scroll = ui.ScrollState{}
}
func (s *app) importCollection(meta library.Collection, tracks []library.Track) {
	p := library.Playlist{Name: meta.Name, Cover: meta.Cover, Description: meta.Description, Source: meta.Source, ExternalID: meta.ID, Kind: meta.Kind, Link: meta.Link, Tracks: append([]library.Track(nil), tracks...)}
	for _, old := range s.cfg.Playlists {
		if old.Source == p.Source && old.ExternalID == p.ExternalID && old.Kind == p.Kind {
			p.ID = old.ID
			break
		}
	}
	if len(p.Tracks) == 0 {
		s.playlistError = "歌单中没有可导入的歌曲"
		s.status = s.playlistError
		return
	}
	if s.storePlaylist(p) {
		s.status = fmt.Sprintf("已导入「%s」，共 %d 首歌曲", p.Name, len(p.Tracks))
		s.modal = ""
	}
}
func (s *app) importRemoteCollection(meta library.Collection) {
	if s.importBusy {
		return
	}
	s.importBusy = true
	cfg := s.cfg
	go func() {
		ctx, cancel := context.WithTimeout(s.ctx, 45*time.Second)
		defer cancel()
		r, e := library.Browse(ctx, meta.Source, meta.Kind, "", meta.ID, 1, cfg)
		s.update(func() {
			s.importBusy = false
			if e != nil {
				s.status = "歌单导入失败：" + e.Error()
				return
			}
			s.importCollection(meta, r.Tracks)
		})
	}()
}
func (s *app) importLinkView(c *ui.Context) {
	ui.Text(c, "粘贴网易云歌单、专辑或 Bilibili 合集链接").FontSize(13).TextColor(muted)
	ui.TextInput(c, &s.importLink).Label("歌单链接").Placeholder("https://...").FontSize(14).Height(42).Radius(8).Border(1, border).Background(white)
	if s.playlistError != "" {
		ui.Text(c, s.playlistError).FontSize(12).TextColor(ui.Hex("#e53e3e")).MaxLines(3)
	}
	label := "导入歌单"
	if s.importBusy {
		label = "正在导入…"
	}
	ui.PrimaryButton(c, label).Label("确认导入歌单").FillWidth().Height(42).Disabled(s.importBusy).OnClick(s.importPlaylistLink)
}
func (s *app) importPlaylistLink() {
	if s.importBusy {
		return
	}
	link, source, kind := library.ShareQuery(s.importLink)
	if source == "" {
		s.playlistError = "请粘贴网易云音乐或 Bilibili 的分享链接"
		return
	}
	if kind == "" {
		kind = "playlist"
	}
	s.importBusy = true
	s.playlistError = ""
	cfg := s.cfg
	go func() {
		ctx, cancel := context.WithTimeout(s.ctx, 45*time.Second)
		defer cancel()
		r, e := library.Browse(ctx, source, kind, link, "", 1, cfg)
		s.update(func() {
			s.importBusy = false
			if e != nil {
				s.playlistError = "导入失败：" + e.Error()
				return
			}
			meta := library.Collection{Playlist: model.Playlist{Name: "导入的歌单", Source: source, ID: link, Link: link}, Kind: kind}
			if r.Container != nil {
				meta = *r.Container
			} else if len(r.Tracks) > 0 {
				meta.Name = r.Tracks[0].Name
			}
			s.importCollection(meta, r.Tracks)
		})
	}()
}

func (s *app) openLocalPlaylist(p library.Playlist) {
	if p.ID == "favorites" {
		s.page = "favorites"
		s.pageNumber = 1
		s.collectionTitle = ""
		s.collections = nil
		s.categories = nil
		s.scroll = ui.ScrollState{}
		return
	}
	s.openSavedPlaylist(p)
}
func (s *app) downloadRemoteCollection(meta library.Collection) {
	cfg := s.cfg
	go func() {
		ctx, cancel := context.WithTimeout(s.ctx, 45*time.Second)
		defer cancel()
		r, e := library.Browse(ctx, meta.Source, meta.Kind, "", meta.ID, 1, cfg)
		s.update(func() {
			if e != nil {
				s.status = "歌单读取失败：" + e.Error()
				return
			}
			s.addDownloads(r.Tracks)
		})
	}()
}

func (s *app) collectionResultHeader(c *ui.Context, count int, kind string) {
	unit, icon := "个歌单", "list"
	if kind == "album" {
		unit, icon = "张专辑", "disc"
	}
	ui.Column(c).Padding(10, 12).Gap(6).Margin(0, 0, 15, 0).Background(white.Alpha(.85)).Border(1, ui.Hex("#edf2f7")).Radius(12).Shadow(0, 4, 6, 0, ui.RGBA(0, 0, 0, .05)).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			glyph(c, icon, 14, textColor)
			ui.Text(c, "找到").FontSize(14).Bold()
			ui.Text(c, fmt.Sprint(count)).FontSize(14).Bold().TextColor(accent)
			ui.Text(c, unit).FontSize(14).Bold()
		})
		ui.Text(c, fmt.Sprintf("当前第 1 / 1 页，显示 1 - %d / %d", count, count)).FontSize(12).TextColor(muted)
	})
}
