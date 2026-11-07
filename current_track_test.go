package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo/ui"
	"github.com/guohuiyuan/music-lib/model"
)

func TestLocateCurrentTrackChangesPageAndRevealsCard(t *testing.T) {
	for _, test := range []struct {
		name              string
		otherList, sorted bool
	}{{"another page", false, false}, {"different list", true, false}, {"sorted list", false, true}} {
		t.Run(test.name, func(t *testing.T) {
			s := newApp(context.Background(), library.Config{PageSize: 20}, "")
			for i := 0; i < 95; i++ {
				s.queue = append(s.queue, library.Track{Song: model.Song{ID: fmt.Sprint(i), Name: fmt.Sprintf("定位测试 %03d", i), Source: "netease", Duration: 200}})
			}
			s.current = &s.queue[87]
			s.queueIndex = 87
			s.position = 42 * time.Second
			s.results = append([]library.Track(nil), s.queue...)
			if test.otherList {
				s.results = []library.Track{{Song: model.Song{ID: "other", Name: "其他搜索结果", Source: "netease"}}}
			}
			if test.sorted {
				s.sortMode = "name"
				s.sortReverse = true
			}
			s.pageNumber = 1
			s.playerExpanded = true
			tt := ui.NewTester(s.view, 1080, 740)
			if err := tt.Click("定位当前播放歌曲"); err != nil {
				t.Fatal(err)
			}
			tt.Advance(time.Second)
			wantPage := 5
			if test.sorted {
				wantPage = 1
			}
			if s.pageNumber != wantPage {
				t.Fatalf("page=%d want %d", s.pageNumber, wantPage)
			}
			if test.otherList && s.page != "queue" {
				t.Fatal("did not return to the playing queue")
			}
			if !test.otherList && s.page != "library" {
				t.Fatal("left the existing song list unnecessarily")
			}
			card, ok := tt.Find("歌曲 " + s.current.Name)
			if !ok || card.H < 100 || card.Y < 0 || card.Y+card.H > 674 {
				t.Fatalf("current card not visible above player: %+v found=%v", card, ok)
			}
			if s.player != nil || s.position != 42*time.Second {
				t.Fatal("locating changed playback")
			}
		})
	}
}
