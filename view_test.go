package main

import (
	"context"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	if text, _ := s.miniCaption(); text != "歌名 · 作者" {
		t.Fatalf("paused caption=%q", text)
	}
	s.playing = true
	s.lines = nil
	tt.Frame()
	if text, _ := s.miniCaption(); text != "歌名 · 作者" {
		t.Fatalf("no-lyrics caption=%q", text)
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
	s.current = &s.results[0]
	s.duration = 258 * time.Second
	s.position = 36 * time.Second
	s.playing = true
	s.lines = lyrics.Parse("[00:00]让音乐陪伴每一个日常\n[00:30]此刻的旋律，留在小小的窗口里")
	for name, view := range map[string]func(*ui.Context){"main": s.view, "mini": s.miniView} {
		w, h := 1080, 740
		if name == "mini" {
			w, h = 304, 50
		}
		tt := ui.NewTester(view, w, h)
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
