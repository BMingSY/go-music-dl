package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BMingSY/go-music-dl/internal/download"
	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/BMingSY/go-music-dl/internal/lyrics"
	"github.com/egoist/mygo/ui"
	"github.com/guohuiyuan/music-lib/model"
)

func TestNativeViewsAndFavorites(t *testing.T) {
	s := newApp(context.Background(), library.Config{Volume: .7}, filepath.Join(t.TempDir(), "settings.json"))
	s.results = []library.Track{{Song: model.Song{ID: "fixture", Source: "netease", Name: "夜に駆ける", Artist: "YOASOBI", Duration: 258}}, {Song: model.Song{ID: "local-fixture", Source: "local", Name: "本地测试曲目", Artist: "本地作者"}}}
	tt := ui.NewTester(s.view, 1080, 740)
	for _, text := range []string{"网易云音乐", "Bilibili", "本地音乐", "夜に駆ける"} {
		if !tt.HasText(text) {
			t.Errorf("missing %q", text)
		}
	}
	if err := tt.Click("收藏 夜に駆ける"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("加入收藏"); err != nil {
		t.Fatal(err)
	}
	if len(s.cfg.Favorites) != 1 {
		t.Fatalf("favorite action did not persist; status=%q texts=%v", s.status, tt.Texts())
	}
	if err := tt.Click("本地歌单"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("打开我的收藏"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("夜に駆ける") {
		t.Fatal("favorite list missing track")
	}
	if err := tt.Click("系统设置"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("平台 Cookie"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("网易云音乐 Cookie"); !ok {
		t.Fatal("settings not native")
	}
	tt.SetSize(940, 620)
	tt.Frame()
}

func TestMiniLyricsPauseAndNoLyrics(t *testing.T) {
	s := newApp(context.Background(), library.Config{Volume: .7}, "")
	s.current = &library.Track{Song: model.Song{Name: "歌名", Artist: "作者"}}
	s.lines = lyrics.Parse("[00:00]第一行\n[00:02]第二行")
	s.playing = true
	s.position = 2500 * time.Millisecond
	if text, _ := s.miniCaption(); text != "第二行" {
		t.Fatalf("playing caption=%q", text)
	}
	tt := ui.NewTester(s.miniView, 304, 50)
	if _, ok := tt.Find("第二行"); !ok {
		t.Fatal("lyric is not accessible")
	}
	s.playing = false
	tt.Frame()
	if text, _ := s.miniCaption(); text != "歌名" {
		t.Fatalf("paused caption=%q", text)
	}
	s.playing = true
	s.lines = nil
	tt.Frame()
	if text, _ := s.miniCaption(); text != "歌名 · 作者" {
		t.Fatalf("no-lyrics caption=%q", text)
	}
}

func TestSongMenusAndListTools(t *testing.T) {
	s := newApp(context.Background(), library.Config{PageSize: 30}, filepath.Join(t.TempDir(), "settings.json"))
	s.results = []library.Track{{Song: model.Song{ID: "one", Source: "netease", Name: "测试歌曲"}}}
	tt := ui.NewTester(s.view, 1080, 820)
	if _, ok := tt.Find("下载 测试歌曲"); ok {
		t.Fatal("download should start inside the more menu")
	}
	if err := tt.Click("更多操作 测试歌曲"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("下载 测试歌曲"); !ok {
		t.Fatal("more menu did not open")
	}
	if err := tt.Click("列表工具"); err != nil {
		t.Fatal(err)
	}
	if s.songMenuOpen || !s.toolsOpen {
		t.Fatal("clicking another menu did not dismiss the song menu")
	}
	if err := tt.Click("批量操作"); err != nil {
		t.Fatal(err)
	}
	if !s.batch || s.toolsOpen {
		t.Fatal("batch action did not close the tools popover")
	}
	if err := tt.Click("选择 测试歌曲"); err != nil {
		t.Fatal(err)
	}
	if len(s.selectedTracks()) != 1 {
		t.Fatal("batch selection lost the selected track")
	}
}

func TestShortcutCaptureAndSearchInput(t *testing.T) {
	s := newApp(context.Background(), library.Config{}, filepath.Join(t.TempDir(), "settings.json"))
	s.modal = "settings"
	tt := ui.NewTester(s.view, 1080, 820)
	if err := tt.Click("播放 / 暂停快捷键"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Ctrl|ui.Shift, ui.KeyP)
	key := s.cfg.Shortcuts["toggle"]
	if key.Key != uint16(ui.KeyP) || key.Modifiers != uint8(ui.Ctrl|ui.Shift) {
		t.Fatalf("shortcut capture: %#v", key)
	}
	s.cfg.Shortcuts = defaultShortcuts()
	if err := tt.Click("关闭弹窗"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("搜索关键词"); err != nil {
		t.Fatal(err)
	}
	tt.TypeKey(0, ui.KeySpace, " ")
	if s.query != " " {
		t.Fatalf("typing in search was intercepted by playback shortcut: %q", s.query)
	}
}

// Optional artifacts are actual renders of the Go view, not mockup images.
func TestRenderPreview(t *testing.T) {
	dir := os.Getenv("MUSIC_DL_PREVIEW_DIR")
	if dir == "" {
		t.Skip("set MUSIC_DL_PREVIEW_DIR to save renders")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	s := newApp(context.Background(), library.Config{Volume: .7}, "")
	s.results = []library.Track{{Song: model.Song{ID: "1", Source: "netease", Name: "夜に駆ける", Artist: "YOASOBI", Duration: 258}}, {Song: model.Song{ID: "2", Source: "netease", Name: "无声的星空", Artist: "本地音乐", Duration: 216}}, {Song: model.Song{ID: "3", Source: "bilibili", Name: "钢琴与雨", Artist: "Bilibili", Duration: 185}}}
	if fixture := os.Getenv("MUSIC_DL_REVIEW_FIXTURE"); fixture != "" {
		var track library.Track
		b, e := os.ReadFile(filepath.Join(fixture, "track.json"))
		if e != nil {
			t.Fatal(e)
		}
		if e = json.Unmarshal(b, &track); e != nil {
			t.Fatal(e)
		}
		b, e = os.ReadFile(filepath.Join(fixture, "cover.jpg"))
		if e != nil {
			t.Fatal(e)
		}
		s.cover, e = ui.DecodeBitmap(b)
		if e != nil {
			t.Fatal(e)
		}
		s.results[0] = track
		s.images[track.Cover] = s.cover
	}
	s.current = &s.results[0]
	s.duration = 258 * time.Second
	s.position = 36 * time.Second
	s.playing = true
	s.lines = lyrics.Parse("[00:00]让音乐陪伴每一个日常\n[00:30]此刻的旋律，留在小小的窗口里")
	if fixture := os.Getenv("MUSIC_DL_REVIEW_FIXTURE"); fixture != "" {
		b, e := os.ReadFile(filepath.Join(fixture, "lyrics.lrc"))
		if e != nil {
			t.Fatal(e)
		}
		s.lines = lyrics.Parse(string(b))
		data, _ := json.Marshal(s.lines)
		_ = os.WriteFile(filepath.Join(dir, "lyric-groups.json"), data, 0600)
		s.position = 63 * time.Second
		s.duration = time.Duration(s.current.Duration) * time.Second
	}
	for _, name := range []string{"main", "main-playing", "song-menu", "list-tools", "player", "settings", "mini", "mini-paused", "mini-hover", "lyrics", "playlists", "playlist-editor", "collection-picker", "categories", "rankings", "recommend", "history", "downloads", "download-history"} {
		s.playing = name != "main" && name != "song-menu" && name != "mini-paused"
		s.modal = ""
		s.toolsOpen, s.songMenuOpen, s.lyricOpen = false, false, false
		s.page = "library"
		s.collectionKind = "song"
		s.categories = nil
		s.collections = nil
		s.playerExpanded = name == "player"
		view := s.view
		w, h := 1080, 740
		if name == "mini" || name == "mini-paused" || name == "mini-hover" {
			view = s.miniView
			w, h = 236, 50
		}
		if name == "settings" {
			s.modal = "settings"
		}
		if name == "lyrics" {
			s.lyricOpen = true
		}
		if name == "playlists" || name == "playlist-editor" || name == "collection-picker" {
			s.cfg.Playlists = []library.Playlist{{ID: "review", Name: "睡前听的歌", Description: "轻柔的旋律", Tracks: append([]library.Track(nil), s.results...)}}
			if name == "playlists" {
				s.openSavedPlaylists()
				h = 1000
			}
			if name == "playlist-editor" {
				s.editPlaylist(nil)
			}
			if name == "collection-picker" {
				s.pendingTracks = []library.Track{s.results[0]}
				s.modal = "collections"
			}
		}

		if name == "categories" {
			h = 1000
			s.collectionKind = "categories"
			s.categories = []model.PlaylistCategory{{ID: "zh", Name: "华语", Group: "语种", Hot: true}, {ID: "en", Name: "欧美", Group: "语种", Hot: true}, {ID: "jp", Name: "日语", Group: "语种"}, {ID: "pop", Name: "流行", Group: "风格", Hot: true}, {ID: "rock", Name: "摇滚", Group: "风格"}, {ID: "quiet", Name: "安静", Group: "情感"}}
		}
		if name == "rankings" || name == "recommend" {
			h = 1000
			s.collectionKind = "ranks"
			if name == "recommend" {
				s.collectionKind = "recommend"
			}
			for i, title := range []string{"飙升榜", "新歌榜", "原创榜", "热歌榜"} {
				s.collections = append(s.collections, library.Collection{Playlist: model.Playlist{ID: fmt.Sprint(i), Source: "netease", Name: title, TrackCount: 100, PlayCount: 2380000, Extra: map[string]string{"update_frequency": "每天更新"}}, Kind: "rank"})
			}
		}

		if name == "history" {
			s.modal = "history"
			s.cfg.History = append([]library.Track(nil), s.results...)
		}
		if name == "downloads" || name == "download-history" {
			s.modal = "downloads"
			s.downloadTab = ""
			if name == "download-history" {
				s.downloadTab = "history"
			}
			s.jobs = []download.Job{{Track: s.results[0], State: "downloading", Done: 1500000, Total: 5000000}, {Track: s.results[1], State: "done", Done: 3000000, Total: 3000000}, {Track: s.results[2], State: "queued"}}
		}

		tt := ui.NewTester(view, w, h)
		tt.Advance(time.Second)
		if name == "mini-hover" {
			tt.Move(120, 25)
			tt.Advance(time.Second)
		}
		if name == "song-menu" {
			if err := tt.Click("更多操作 " + s.results[0].Name); err != nil {
				t.Fatal(err)
			}
		}
		if name == "list-tools" {
			if err := tt.Click("列表工具"); err != nil {
				t.Fatal(err)
			}
		}
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(f, tt.Image())
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
}
