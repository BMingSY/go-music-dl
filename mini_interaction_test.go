package main

import (
	"context"
	"testing"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo/ui"
	"github.com/guohuiyuan/music-lib/model"
)

func TestMiniCoverBlankControlsAndDrag(t *testing.T) {
	s := newApp(context.Background(), library.Config{Volume: .5}, "")
	restored := 0
	tt := ui.NewTester(func(c *ui.Context) { s.miniContent(c, func() { restored++ }) }, 236, 50)
	// Cover, top padding and the free space between buttons all open main.
	for _, point := range []ui.Point{{X: 20, Y: 25}, {X: 130, Y: 2}, {X: 48, Y: 25}} {
		tt.ClickAt(point.X, point.Y)
		tt.Advance(time.Second)
	}
	if restored != 3 {
		t.Fatalf("cover/background restored main %d times, want 3", restored)
	}
	if _, ok := tt.Find("打开主窗口"); !ok {
		t.Fatal("background has no accessible restore action")
	}
	// There are five controls; restoring main is the background action.
	for _, label := range []string{"上一首", "播放或暂停", "下一首", "调节音量", "隐藏悬浮窗"} {
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
		if restored != 3 {
			t.Fatalf("%s also opened the main window", label)
		}
	}
	tt.Press(20, 25)
	tt.Move(35, 30)
	tt.Release(35, 30)
	if s.miniDrag.pressed || restored != 3 {
		t.Fatalf("drag was treated as a click: moved=%v restores=%d", s.miniDrag.moved, restored)
	}
	tt.ClickAt(20, 25)
	if restored != 4 {
		t.Fatal("a click after dragging did not open main")
	}
}

func TestMiniButtonsHoverWithoutClippedTooltip(t *testing.T) {
	s := newApp(context.Background(), library.Config{}, "")
	tt := ui.NewTester(s.miniView, 236, 50)
	tt.Move(48, 48)
	tt.Advance(time.Second)
	for _, label := range []string{"上一首", "播放或暂停", "下一首", "调节音量", "隐藏悬浮窗"} {
		r, ok := tt.Find(label)
		if !ok {
			t.Fatal(label)
		}
		tt.Move(r.X+r.W/2, r.Y+r.H/2)
		tt.Advance(time.Second)
		img := tt.Image()
		pixel := img.RGBAAt(int(r.X+4), int(r.Y+r.H/2))
		if pixel.G <= pixel.R || pixel.G <= pixel.B {
			t.Fatalf("%s has no green hover feedback: %+v", label, pixel)
		}
		for y := 0; y < 4; y++ {
			for x := 50; x < 230; x++ {
				p := img.RGBAAt(x, y)
				if p.R < 60 && p.G < 60 && p.B < 60 {
					t.Fatalf("%s drew a clipped black tooltip at %d,%d", label, x, y)
				}
			}
		}
	}
}

func TestTrackCardOnlyShowsAvailableAudioFields(t *testing.T) {
	s := newApp(context.Background(), library.Config{}, "")
	track := library.Track{Song: model.Song{ID: "reported", Name: "我跨四海寻【宝莲】", Source: "bilibili", Duration: 172}}
	s.results = []library.Track{track}
	tt := ui.NewTester(s.view, 1080, 740)
	for _, text := range tt.Texts() {
		if text == "-" {
			t.Fatal("unknown size/bitrate badge is still visible")
		}
	}
	if !tt.HasText("02:52") {
		t.Fatal("known duration was hidden")
	}
	s.results[0].Size = 4 * 1024 * 1024
	s.results[0].Bitrate = 192
	tt.Frame()
	if !tt.HasText("4.00 MB") || !tt.HasText("192 kbps") {
		t.Fatal("valid media values were hidden")
	}
	s.results[0].Size = 0
	tt.Frame()
	if tt.HasText("4.00 MB") || !tt.HasText("192 kbps") {
		t.Fatal("missing one field hid another known field")
	}
}

func TestDesktopBrowseEntries(t *testing.T) {
	s := newApp(context.Background(), library.Config{}, "")
	tt := ui.NewTester(s.view, 1080, 740)
	if _, ok := tt.Find("歌单分类"); ok {
		t.Fatal("removed category entry is still visible")
	}
	for _, label := range []string{"每日推荐歌单", "本地歌单", "本地音乐", "排行榜", "我的歌单"} {
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("remaining entry %q is missing", label)
		}
	}
}
