package library

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
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
