package library

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/BMingSY/go-music-dl/internal/playback"
)

func TestLiveReportedBilibiliCollection(t *testing.T) {
	if os.Getenv("MUSIC_DL_LIVE_TESTS") == "" {
		t.Skip("online regression")
	}
	http.DefaultClient.Timeout = 20 * time.Second
	r, e := Browse(context.Background(), "bilibili", "song", "https://www.bilibili.com/video/BV1YmHA6WEJz/", "", 1, Config{})
	if e != nil {
		t.Fatal(e)
	}
	if r.Container == nil || len(r.Tracks) < 2 {
		t.Fatalf("missing reported collection: %+v", r)
	}
	t.Logf("collection=%q tracks=%d first=%q", r.Container.Name, len(r.Tracks), r.Tracks[0].Name)
}

func TestLiveBilibiliHTTPSPlaybackAndSeek(t *testing.T) {
	if os.Getenv("MUSIC_DL_LIVE_TESTS") == "" {
		t.Skip("online playback regression")
	}
	ffmpeg, err := playback.FindFFmpeg(os.Getenv("MUSIC_DL_TEST_FFMPEG"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	r, err := Browse(ctx, "bilibili", "song", "https://www.bilibili.com/video/BV1YmHA6WEJz/", "", 1, Config{})
	if err != nil || len(r.Tracks) == 0 {
		t.Fatal("could not read the public Bilibili collection")
	}
	stream, err := Resolve(ctx, r.Tracks[0], Config{})
	if err != nil {
		t.Fatal("could not resolve the public Bilibili audio")
	}
	d, err := playback.Open(ffmpeg, playback.Input{Location: stream.Input, Headers: stream.Headers})
	if err != nil {
		t.Fatal("could not open the public audio stream")
	}
	defer d.Close()
	stop := context.AfterFunc(ctx, func() { d.Close() })
	defer stop()
	for _, seek := range []bool{false, true} {
		if seek {
			if err := d.SetPosition(2 * playback.Rate); err != nil {
				t.Fatal("could not seek in the remote audio")
			}
		}
		samples := make([]float32, 4096)
		if n, err := d.Read(samples); n == 0 || err != nil {
			t.Fatal("remote audio did not decode")
		}
	}
	t.Log("public Bilibili HTTPS audio decodes before and after seeking")
}
