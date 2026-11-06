package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/guohuiyuan/music-lib/model"
	qrcode "github.com/skip2/go-qrcode"
)

func (s *app) search() {
	if link, source, kind := library.ShareQuery(s.query); source != "" {
		s.query = link
		s.selected = map[string]bool{source: true}
		if kind != "" {
			s.searchKind = kind
		}
	}
	s.browse(s.searchKind, "", "")
}
func (s *app) browse(kind, source, id string) {
	if s.searchCancel != nil {
		s.searchCancel()
	}
	ctx, cancel := context.WithCancel(s.ctx)
	s.searchCancel = cancel
	s.searchID++
	seq := s.searchID
	s.searching = true
	s.status = "正在查找…"
	s.page = "library"
	s.pageNumber = 1
	s.scroll = ui.ScrollState{}
	s.collectionTitle = ""
	s.currentCollection = nil
	s.collectionKind = kind
	s.collectionID = id
	cfg, query := s.cfg, strings.TrimSpace(s.query)
	var sources []string
	if source != "" {
		sources = []string{source}
	} else {
		for _, src := range []string{"netease", "bilibili", "local"} {
			if s.selected[src] && (kind == "song" || src != "local") && (kind != "album" || src == "netease") {
				sources = append(sources, src)
			}
		}
	}
	if len(sources) == 0 {
		s.searching = false
		s.status = "请选择支持此功能的音乐来源"
		return
	}
	s.source = ""
	if len(sources) == 1 {
		s.source = sources[0]
	}
	go func() {
		type result struct {
			data library.Catalog
			err  error
		}
		ch := make(chan result, len(sources))
		for _, src := range sources {
			go func() { r, e := library.Browse(ctx, src, kind, query, id, 1, cfg); ch <- result{r, e} }()
		}
		var all library.Catalog
		var errs []string
		for range sources {
			r := <-ch
			if r.err != nil {
				errs = append(errs, r.err.Error())
			} else {
				all.Tracks = append(all.Tracks, r.data.Tracks...)
				all.Collections = append(all.Collections, r.data.Collections...)
				all.Categories = append(all.Categories, r.data.Categories...)
				if r.data.Container != nil {
					all.Container = r.data.Container
				}
			}
		}
		s.update(func() {
			if seq != s.searchID || ctx.Err() != nil {
				return
			}
			s.searching = false
			s.results = all.Tracks
			s.collections = all.Collections
			s.categories = all.Categories
			s.checked = map[string]bool{}
			s.list = ui.ListState{}
			s.status = strings.Join(errs, "；")
			if all.Container != nil {
				s.currentCollection = all.Container
				s.collectionTitle = all.Container.Name
				s.collectionKind = all.Container.Kind
				s.collectionID = all.Container.ID
				s.searchKind = all.Container.Kind
			}
			if len(all.Tracks)+len(all.Collections)+len(all.Categories) == 0 && s.status == "" {
				s.status = "没有找到结果"
			}
			s.redraw()
		})
	}()
}
func (s *app) openCollection(c library.Collection) {
	s.browse(c.Kind, c.Source, c.ID)
	s.collectionTitle = c.Name
	s.currentCollection = &c
}
func (s *app) localMusic() {
	s.query = ""
	s.searchKind = "song"
	s.selectSource("local")
	s.pageNumber = 1
}
func (s *app) shownTracks() []library.Track {
	tracks := s.results
	switch s.page {
	case "favorites":
		tracks = s.cfg.Favorites
	case "queue":
		tracks = s.queue
	case "history":
		tracks = s.cfg.History
	case "collection":
		for _, p := range s.cfg.Playlists {
			if p.ID == s.localPlaylistID {
				tracks = p.Tracks
				break
			}
		}
	}
	if s.sortMode != "" {
		tracks = append([]library.Track(nil), tracks...)
		sort.SliceStable(tracks, func(i, j int) bool {
			a, b := tracks[i], tracks[j]
			var less bool
			switch s.sortMode {
			case "duration":
				less = a.Duration < b.Duration
			case "quality":
				less = a.Bitrate < b.Bitrate
			case "size":
				less = a.Size < b.Size
			default:
				less = a.Name < b.Name
			}
			if s.sortReverse {
				a, b = b, a
				switch s.sortMode {
				case "duration":
					less = a.Duration < b.Duration
				case "quality":
					less = a.Bitrate < b.Bitrate
				case "size":
					less = a.Size < b.Size
				default:
					less = a.Name < b.Name
				}
			}
			return less
		})
	}
	return tracks
}
func (s *app) recordHistory(t library.Track) {
	history := []library.Track{t}
	for _, old := range s.cfg.History {
		if old.Key() != t.Key() {
			history = append(history, old)
		}
		if len(history) >= 200 {
			break
		}
	}
	s.cfg.History = history
	s.save()
}
func (s *app) addDownloads(tracks []library.Track) {
	if s.downloads == nil {
		return
	}
	s.downloads.Add(tracks, s.cfg)
	s.status = fmt.Sprintf("已加入下载队列：%d 首", len(tracks))
	s.jobs = s.downloads.Snapshot()
}
func (s *app) selectedTracks() []library.Track {
	var out []library.Track
	for _, t := range s.shownTracks() {
		if s.checked[t.Key()] {
			out = append(out, t)
		}
	}
	return out
}
func (s *app) savePlaylist(name string, tracks []library.Track) {
	name = strings.TrimSpace(name)
	if name == "" {
		s.status = "请输入歌单名称"
		return
	}
	s.cfg.Playlists = append(s.cfg.Playlists, library.Playlist{ID: fmt.Sprint(time.Now().UnixNano()), Name: name, Tracks: append([]library.Track(nil), tracks...)})
	s.playlistName = ""
	s.pendingTracks = nil
	s.save()
}
func (s *app) addToPlaylist(id string, tracks []library.Track) {
	for i := range s.cfg.Playlists {
		p := &s.cfg.Playlists[i]
		if p.ID != id {
			continue
		}
		for _, t := range tracks {
			if !slices.ContainsFunc(p.Tracks, func(v library.Track) bool { return v.Key() == t.Key() }) {
				p.Tracks = append(p.Tracks, t)
			}
		}
		s.save()
		break
	}
	s.pendingTracks = nil
	s.modal = ""
	s.status = "已加入歌单"
}
func (s *app) startLogin(source string) {
	if s.loginCancel != nil {
		s.loginCancel()
	}
	ctx, cancel := context.WithCancel(s.ctx)
	s.loginCancel = cancel
	s.loginID++
	id := s.loginID
	s.loginSource = source
	s.loginStatus = "正在生成二维码…"
	s.loginImage = nil
	s.modal = "login"
	cfg := s.cfg
	go func() {
		session, err := library.CreateLogin(source, cfg)
		if err != nil {
			s.update(func() {
				if s.loginID == id {
					s.loginStatus = "二维码生成失败，请重试"
				}
			})
			return
		}
		if ctx.Err() != nil {
			return
		}
		code, err := qrcode.Encode(session.URL, qrcode.Medium, 240)
		if err != nil {
			return
		}
		s.update(func() {
			if s.loginID != id {
				return
			}
			s.loginImage, _ = ui.DecodeBitmap(code)
			s.loginStatus = "请使用对应平台的 App 扫码登录"
		})
		timer := time.NewTimer(5 * time.Minute)
		defer timer.Stop()
		tick := time.NewTicker(2 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				s.update(func() {
					if s.loginID == id {
						s.loginStatus = "二维码已过期，请重新获取"
					}
				})
				return
			case <-tick.C:
				r, e := library.CheckLogin(source, session.Key, cfg)
				if e != nil || r == nil {
					continue
				}
				done := r.Status == model.QRLoginStatusSuccess || r.Status == model.QRLoginStatusExpired || r.Status == model.QRLoginStatusFailed
				s.update(func() {
					if s.loginID != id || ctx.Err() != nil {
						return
					}
					switch r.Status {
					case model.QRLoginStatusScanned:
						s.loginStatus = "已扫码，请在手机上确认"
					case model.QRLoginStatusSuccess:
						cookie := library.LoginCookie(r)
						if cookie == "" {
							s.loginStatus = "登录未返回 Cookie，请重试"
							return
						}
						if source == "netease" {
							s.cfg.NeteaseCookie = cookie
						} else {
							s.cfg.BilibiliCookie = cookie
						}
						s.save()
						s.loginStatus = "登录成功"
						s.loginImage = nil
					case model.QRLoginStatusExpired:
						s.loginStatus = "二维码已过期，请重新获取"
					case model.QRLoginStatusFailed:
						s.loginStatus = "登录失败，请重试"
					}
				})
				if done {
					return
				}
			}
		}
	}()
}
func (s *app) closeModal() {
	s.captureShortcut = ""
	if s.modal == "login" && s.loginCancel != nil {
		s.loginCancel()
		s.loginID++
	}
	s.modal = ""
	s.pendingTracks = nil
}
func (s *app) artwork(t library.Track) *ui.Bitmap {
	key := t.Cover
	if t.Source == "local" {
		key = t.Key()
	}
	if key == "" {
		return nil
	}
	if img, ok := s.images[key]; ok {
		return img
	}
	if s.imagePending[key] {
		return nil
	}
	s.imagePending[key] = true
	cfg := s.cfg
	go func() {
		select {
		case s.imageSlots <- struct{}{}:
		case <-s.ctx.Done():
			return
		}
		defer func() { <-s.imageSlots }()
		var data []byte
		if t.Source == "local" {
			_, data = library.Metadata(s.ctx, t, cfg)
		} else {
			req, e := http.NewRequestWithContext(s.ctx, "GET", key, nil)
			if e == nil {
				req.Header.Set("User-Agent", library.UserAgent)
				if t.Source == "bilibili" {
					req.Header.Set("Referer", "https://www.bilibili.com/")
				}
				resp, e := (&http.Client{Timeout: 8 * time.Second}).Do(req)
				if e == nil {
					if resp.StatusCode == 200 {
						data, _ = io.ReadAll(io.LimitReader(resp.Body, 5<<20))
					}
					resp.Body.Close()
				}
			}
		}
		s.update(func() { img, _ := ui.DecodeBitmap(data); s.images[key] = img; delete(s.imagePending, key) })
	}()
	return nil
}
func (s *app) pickDownloadFolder() {
	go func() {
		paths, e := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: s.win, Title: "选择下载目录", Directory: true})
		s.update(func() {
			if e == nil && len(paths) > 0 {
				s.cfg.DownloadDir = paths[0]
				s.save()
			}
		})
	}()
}
func (s *app) exportPlaylists() {
	lists := append([]library.Playlist(nil), s.cfg.Playlists...)
	go func() {
		path, e := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: s.win, Title: "导出歌单", DefaultPath: "music-dl-playlists.json"})
		if e != nil || path == "" {
			return
		}
		b, e := json.MarshalIndent(lists, "", "  ")
		if e == nil {
			e = os.WriteFile(path, b, 0600)
		}
		s.update(func() {
			if e != nil {
				s.status = "导出失败"
			} else {
				s.status = "歌单已导出"
			}
		})
	}()
}
func (s *app) importPlaylists() {
	go func() {
		paths, e := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: s.win, Title: "导入歌单 JSON"})
		if e != nil || len(paths) == 0 {
			return
		}
		f, e := os.Open(paths[0])
		if e != nil {
			return
		}
		b, e := io.ReadAll(io.LimitReader(f, 16<<20))
		f.Close()
		var lists []library.Playlist
		if e == nil {
			e = json.Unmarshal(b, &lists)
		}
		s.update(func() {
			if e != nil {
				s.status = "歌单文件格式不正确"
				return
			}
			for _, p := range lists {
				var tracks []library.Track
				for _, t := range p.Tracks {
					if t.Source == "netease" || t.Source == "bilibili" || t.Source == "local" {
						tracks = append(tracks, t)
					}
				}
				s.savePlaylist(p.Name, tracks)
			}
			s.status = "歌单已导入"
		})
	}()
}
func (s *app) saveLyrics(t library.Track) {
	cfg := s.cfg
	go func() {
		lyric, _ := library.Metadata(s.ctx, t, cfg)
		if lyric == "" {
			s.update(func() { s.status = "此歌曲暂无歌词" })
			return
		}
		path, e := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: s.win, Title: "保存歌词", DefaultPath: t.Name + ".lrc"})
		if e == nil && path != "" {
			e = os.WriteFile(path, []byte(lyric), 0644)
			s.update(func() {
				if e != nil {
					s.status = "歌词保存失败"
				} else {
					s.status = "歌词已保存"
				}
			})
		}
	}()
}
func (s *app) openDownloadDir() {
	if s.cfg.DownloadDir != "" {
		_ = os.MkdirAll(s.cfg.DownloadDir, 0755)
		_ = mygo.Shell.OpenPath(s.cfg.DownloadDir)
	}
}
func supportedTrack(t library.Track) error {
	if t.Source != "netease" && t.Source != "bilibili" && t.Source != "local" {
		return errors.New("不支持的来源")
	}
	if t.Source == "local" && !filepath.IsAbs(t.Path) {
		return errors.New("本地路径必须为绝对路径")
	}
	return nil
}

func (s *app) removeFromCollection(t library.Track) {
	if s.page == "favorites" {
		s.cfg.Favorites = slices.DeleteFunc(s.cfg.Favorites, func(track library.Track) bool { return track.Key() == t.Key() })
	}
	if s.page == "collection" {
		for i := range s.cfg.Playlists {
			if s.cfg.Playlists[i].ID == s.localPlaylistID {
				s.cfg.Playlists[i].Tracks = slices.DeleteFunc(s.cfg.Playlists[i].Tracks, func(track library.Track) bool { return track.Key() == t.Key() })
			}
		}
	}
	s.save()
}

func (s *app) trashLocalTrack(t library.Track) {
	if t.Source != "local" || !filepath.IsAbs(t.Path) {
		return
	}
	go func() {
		result, err := mygo.Dialog.Message(mygo.MessageOptions{Parent: s.win, Message: "将「" + t.Name + "」移到回收站？", Detail: t.Path, Buttons: []string{"取消", "移到回收站"}})
		if err != nil || result.Button != 1 {
			return
		}
		s.update(func() {
			if s.current != nil && s.current.Key() == t.Key() {
				s.playID++
				s.stop()
				s.current = nil
				s.loading = false
			}
		})
		err = mygo.Shell.TrashItem(t.Path)
		s.update(func() {
			if err != nil {
				s.status = "无法移到回收站: " + err.Error()
				return
			}
			s.cfg.LocalFiles = slices.DeleteFunc(s.cfg.LocalFiles, func(path string) bool { return path == t.Path })
			s.cfg.Favorites = slices.DeleteFunc(s.cfg.Favorites, func(track library.Track) bool { return track.Key() == t.Key() })
			for i := range s.cfg.Playlists {
				s.cfg.Playlists[i].Tracks = slices.DeleteFunc(s.cfg.Playlists[i].Tracks, func(track library.Track) bool { return track.Key() == t.Key() })
			}
			s.results = slices.DeleteFunc(s.results, func(track library.Track) bool { return track.Key() == t.Key() })
			s.queue = slices.DeleteFunc(s.queue, func(track library.Track) bool { return track.Key() == t.Key() })
			s.save()
		})
	}()
}

func (s *app) pickLocalFiles() {
	go func() {
		paths, e := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: s.win, Title: "导入本地音乐", Multiple: true, Filters: []mygo.FileFilter{{Name: "音频文件", Extensions: []string{"mp3", "flac", "m4a", "aac", "ogg", "wav", "wma"}}}})
		if e != nil {
			return
		}
		s.update(func() {
			for _, path := range paths {
				if !slices.Contains(s.cfg.LocalFiles, path) {
					s.cfg.LocalFiles = append(s.cfg.LocalFiles, path)
				}
			}
			s.save()
			s.localMusic()
		})
	}()
}
