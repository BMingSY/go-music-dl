package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo/ui"
	"github.com/guohuiyuan/music-lib/model"
)

func TestPlaybackSessionRestoresPausedWithoutMediaRequests(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s := newApp(context.Background(), library.Config{}, path)
	s.queue = []library.Track{{Song: model.Song{ID: "one", Name: "第一首", Source: "netease", Duration: 180, URL: "https://example.com/secret-media-url"}}, {Song: model.Song{ID: "two", Name: "第二首", Source: "bilibili", Duration: 200}}}
	s.queueIndex = 1
	s.current = &s.queue[1]
	s.position = 42500 * time.Millisecond
	s.speed = 1.25
	s.playing = true
	if err := s.savePlayback(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(s.playbackPath())
	if err != nil || bytes.Contains(data, []byte("secret-media-url")) {
		t.Fatal("session persisted an expiring audio URL")
	}
	restored := newApp(context.Background(), library.Config{}, path)
	if restored.current == nil || restored.current.ID != "two" || restored.queueIndex != 1 || len(restored.queue) != 2 || restored.position != s.position || restored.speed != 1.25 {
		t.Fatalf("restored state differs: %+v", restored)
	}
	if restored.player != nil || restored.decoder != nil || restored.loading || restored.playing {
		t.Fatal("startup opened or played media before pressing play")
	}
	restored.seek(60 * time.Second)
	reopened := newApp(context.Background(), library.Config{}, path)
	if reopened.position != 60*time.Second {
		t.Fatal("seeking while restored and paused was not saved")
	}
}

func TestInvalidPlaybackSessionDoesNotPreventStartup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "playback.json"), []byte(`{"queue":[],"index":99}`), 0600); err != nil {
		t.Fatal(err)
	}
	s := newApp(context.Background(), library.Config{}, path)
	if s.current != nil || s.queueIndex != -1 {
		t.Fatal("invalid session was restored")
	}
}

func TestQueueOpensAtCurrentTrackAndModesStayInSync(t *testing.T) {
	s := newApp(context.Background(), library.Config{}, filepath.Join(t.TempDir(), "settings.json"))
	for i := 0; i < 464; i++ {
		s.queue = append(s.queue, library.Track{Song: model.Song{ID: fmt.Sprint(i), Name: fmt.Sprintf("歌曲 %03d", i), Source: "netease", Duration: 200}})
	}
	s.playerExpanded = true
	s.queueIndex = 300
	s.current = &s.queue[300]
	tt := ui.NewTester(s.view, 1080, 740)
	if err := tt.Click("播放队列"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	viewport, ok := tt.Find("播放队列列表")
	if !ok {
		t.Fatal("queue did not open")
	}
	row, ok := tt.Find("队列歌曲 歌曲 300")
	if !ok || row.Y < viewport.Y || row.Y+row.H > viewport.Y+viewport.H+1 {
		t.Fatalf("current song is outside the opened queue: row=%+v viewport=%+v", row, viewport)
	}
	for _, mode := range []string{"列表循环", "单曲循环", "随机播放", "顺序播放"} {
		_, before := playbackMode(s.cfg.PlayMode)
		if err := tt.Click("播放模式：" + before); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
		for _, prefix := range []string{"播放模式：", "切换播放模式："} {
			if _, ok := tt.Find(prefix + mode); !ok {
				t.Fatalf("%s did not update to %s", prefix, mode)
			}
		}
		if s.status == mode {
			t.Fatal("mode change still leaves a floating text status")
		}
	}
}

func TestLyricsExtraActionsAreCollapsed(t *testing.T) {
	s := newApp(context.Background(), library.Config{}, "")
	s.current = &library.Track{Song: model.Song{ID: "song", Source: "netease", Name: "测试歌曲"}}
	s.lyricOpen = true
	tt := ui.NewTester(s.view, 1080, 740)
	for _, label := range []string{"更换封面或背景", "更换本地音频", "更换本地歌词", "导出视频"} {
		if _, ok := tt.Find(label); ok {
			t.Fatalf("%s is visible before opening more", label)
		}
	}
	if _, ok := tt.Find("歌词页播放模式：顺序播放"); !ok {
		t.Fatal("lyrics page has no mode icon")
	}
	if err := tt.Click("歌词页更多操作"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	for _, label := range []string{"更换封面或背景", "更换本地音频", "更换本地歌词", "导出视频"} {
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("%s missing from more menu", label)
		}
	}
}
