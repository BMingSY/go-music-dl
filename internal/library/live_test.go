package library

import (
	"context"
	"os"
	"testing"
)

// This opt-in diagnostic uses public search only, never account cookies or
// downloads. Keep it out of deterministic PR checks.
func TestLivePublicSearch(t *testing.T) {
	if os.Getenv("MUSIC_DL_LIVE_TESTS") != "1" {
		t.Skip("opt-in live source diagnostic")
	}
	for _, source := range []string{"netease", "bilibili"} {
		t.Run(source, func(t *testing.T) {
			tracks, err := Search(context.Background(), source, "bach", Config{})
			if err != nil {
				t.Fatal(err)
			}
			if len(tracks) == 0 {
				t.Fatal("public search returned no results")
			}
			t.Logf("%s returned %d tracks", source, len(tracks))
		})
	}
}
