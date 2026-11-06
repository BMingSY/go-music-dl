package library

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCaptureReviewFixture(t *testing.T) {
	dir := os.Getenv("MUSIC_DL_REVIEW_FIXTURE")
	if dir == "" {
		t.Skip("optional real-world UI fixture")
	}
	http.DefaultClient.Timeout = 20 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	tracks, e := Search(ctx, "netease", "ハロウィンナイトパーティー", Config{})
	if e != nil {
		t.Fatal(e)
	}
	if len(tracks) == 0 {
		t.Fatal("reference song not found")
	}
	track := tracks[0]
	track.URL = ""
	lyric, cover := Metadata(ctx, track, Config{})
	if len(lyric) == 0 || len(cover) == 0 {
		t.Fatal("reference artwork/lyrics missing")
	}
	if e = os.MkdirAll(dir, 0700); e != nil {
		t.Fatal(e)
	}
	b, e := json.Marshal(track)
	if e != nil {
		t.Fatal(e)
	}
	for name, data := range map[string][]byte{"track.json": b, "lyrics.lrc": []byte(lyric), "cover.jpg": cover} {
		if e = os.WriteFile(filepath.Join(dir, name), data, 0600); e != nil {
			t.Fatal(e)
		}
	}
	t.Logf("reference: %s / %s", track.Name, track.Artist)
}
