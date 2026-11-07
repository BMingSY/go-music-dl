package desktop

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo"
)

func (s *app) isFavorite(t library.Track) bool {
	for _, f := range s.cfg.Favorites {
		if f.Key() == t.Key() {
			return true
		}
	}
	return false
}

func (s *app) favorite(t library.Track) {
	for i, f := range s.cfg.Favorites {
		if f.Key() == t.Key() {
			s.cfg.Favorites = append(s.cfg.Favorites[:i], s.cfg.Favorites[i+1:]...)
			s.save()
			return
		}
	}
	s.cfg.Favorites = append(s.cfg.Favorites, t)
	s.save()
}

func (s *app) recordHistory(t library.Track) {
	t.PlayedAt = time.Now().Unix()
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

func (s *app) addToPlaylist(id string, tracks []library.Track) {
	for _, p := range s.cfg.Playlists {
		if p.ID != id {
			continue
		}
		p.Tracks = append([]library.Track(nil), p.Tracks...)
		for _, t := range tracks {
			if !slices.ContainsFunc(p.Tracks, func(v library.Track) bool { return v.Key() == t.Key() }) {
				p.Tracks = append(p.Tracks, t)
			}
		}
		if !s.storePlaylist(p) {
			return
		}
		s.pendingTracks = nil
		s.modal = ""
		s.status = "已加入歌单"
		return
	}
	s.playlistError = "歌单不存在，请重新选择"
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
