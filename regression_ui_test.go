package main

import (
	"context"
	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/BMingSY/go-music-dl/internal/lyrics"
	"github.com/egoist/mygo/ui"
	"github.com/guohuiyuan/music-lib/model"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestCreateEditAddAndReloadPlaylist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s := newApp(context.Background(), library.Config{Volume: .6}, path)
	s.openSavedPlaylists()
	tt := ui.NewTester(s.view, 1200, 820)
	if err := tt.Click("新建歌单"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("编辑歌单名称"); err != nil {
		t.Fatal(err)
	}
	tt.Type("睡前听的歌")
	if err := tt.Click("编辑描述"); err != nil {
		t.Fatal(err)
	}
	tt.Type("双语歌曲")
	tt.Frame()
	if err := tt.Click("保存歌单"); err != nil {
		t.Fatal(err)
	}
	cfg, err := library.LoadConfig(path)
	if err != nil || len(cfg.Playlists) != 1 {
		t.Fatalf("create not saved: %v %+v", err, cfg.Playlists)
	}
	s.results = []library.Track{{Song: model.Song{ID: "one", Source: "netease", Name: "测试歌曲"}}}
	s.page = "library"
	tt.Frame()
	if err := tt.Click("收藏 测试歌曲"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("加入"); err != nil {
		t.Fatal(err)
	}
	cfg, err = library.LoadConfig(path)
	if err != nil || len(cfg.Playlists[0].Tracks) != 1 {
		t.Fatalf("song not added: %v %+v", err, cfg.Playlists)
	}
	s.openSavedPlaylists()
	tt.Frame()
	tt.Scroll(600, 700, 0, 500)
	if err := tt.Click("编辑歌单 睡前听的歌"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("编辑歌单名称"); err != nil {
		t.Fatal(err)
	}
	modifier := ui.Ctrl
	if runtime.GOOS == "darwin" {
		modifier = ui.Super
	}
	tt.Key(modifier, ui.KeyA)
	tt.Type("修改后的歌单")
	tt.Frame()
	if err := tt.Click("保存歌单"); err != nil {
		t.Fatal(err)
	}
	cfg, err = library.LoadConfig(path)
	if err != nil || cfg.Playlists[0].Name != "修改后的歌单" || len(cfg.Playlists[0].Tracks) != 1 {
		t.Fatalf("edit lost songs: %v %+v", err, cfg.Playlists)
	}
	reloaded := newApp(context.Background(), cfg, path)
	reloaded.openSavedPlaylist(cfg.Playlists[0])
	if len(reloaded.shownTracks()) != 1 {
		t.Fatal("reload lost playlist content")
	}
}
func TestMiniHoverAndBilingualLyrics(t *testing.T) {
	s := newApp(context.Background(), library.Config{Volume: .5}, "")
	s.playing = true
	s.current = &library.Track{Song: model.Song{Name: "歌名", Artist: "作者"}}
	s.lines = lyrics.Parse("[00:01]Hello[00:02] world[00:04]\n[00:01]你好世界[00:04]")
	s.position = 2500 * time.Millisecond
	tt := ui.NewTester(s.miniView, 236, 50)
	tt.Move(-50, -50)
	tt.Advance(time.Second)
	if _, ok := tt.Find("Hello world"); !ok {
		t.Fatal("original lyric missing")
	}
	if _, ok := tt.Find("你好世界"); !ok {
		t.Fatal("translated lyric missing")
	}
	if _, ok := tt.Find("收藏当前歌曲"); ok {
		t.Fatal("mini still has favorite control")
	}
	tt.Move(120, 25)
	tt.Advance(time.Second)
	for _, label := range []string{"上一首", "下一首", "播放或暂停", "调节音量"} {
		if _, ok := tt.Find(label); !ok {
			t.Errorf("hover control %q missing", label)
		}
	}
}
func TestImportCollectionRetainsSongsAndDeduplicates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s := newApp(context.Background(), library.Config{}, path)
	meta := library.Collection{Playlist: model.Playlist{ID: "remote", Name: "合集", Source: "bilibili"}, Kind: "playlist"}
	tracks := []library.Track{{Song: model.Song{ID: "song|1", Name: "第一首", Source: "bilibili"}}}
	s.importCollection(meta, tracks)
	s.importCollection(meta, tracks)
	cfg, err := library.LoadConfig(path)
	if err != nil || len(cfg.Playlists) != 1 || len(cfg.Playlists[0].Tracks) != 1 {
		t.Fatalf("import failed: %v %+v", err, cfg.Playlists)
	}
}
