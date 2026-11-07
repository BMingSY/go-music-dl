package desktop

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/BMingSY/go-music-dl/internal/playback"
	"github.com/egoist/mygo/audio"
	"github.com/guohuiyuan/music-lib/model"
)

type silentPlaybackDecoder struct{}

func (silentPlaybackDecoder) SampleRate() int         { return playback.Rate }
func (silentPlaybackDecoder) Channels() int           { return 2 }
func (silentPlaybackDecoder) Len() int64              { return 180 * playback.Rate }
func (silentPlaybackDecoder) SetPosition(int64) error { return nil }
func (silentPlaybackDecoder) Read(samples []float32) (int, error) {
	clear(samples)
	return len(samples), nil
}

func failedPlaybackDecoder(t *testing.T) *playback.Decoder {
	t.Helper()
	ffmpeg, err := playback.FindFFmpeg(os.Getenv("MUSIC_DL_TEST_FFMPEG"))
	if err != nil {
		if os.Getenv("MUSIC_DL_REQUIRE_AUDIO_TESTS") == "1" {
			t.Fatal(err)
		}
		t.Skip("FFmpeg is required for playback failure tests")
	}
	path := filepath.Join(t.TempDir(), "invalid.wav")
	if err := os.WriteFile(path, []byte("invalid audio"), 0600); err != nil {
		t.Fatal(err)
	}
	d, err := playback.Open(ffmpeg, playback.Input{Location: path})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	_, _ = d.Read(make([]float32, 4096))
	if d.Err() == nil {
		t.Fatal("fixture did not produce a decoding failure")
	}
	return d
}

func pausedPlaybackApp(t *testing.T, source string) *app {
	t.Helper()
	// These tests inspect the synchronous restart state. Cancellation prevents
	// its worker from contacting a real music provider or opening another player.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := newApp(ctx, library.Config{Volume: 0}, filepath.Join(t.TempDir(), "settings.json"))
	s.queue = []library.Track{
		{Song: model.Song{ID: "one", Source: source, Duration: 180}},
		{Song: model.Song{ID: "two", Source: source, Duration: 200}},
	}
	s.queueIndex = 0
	s.current = &s.queue[0]
	s.duration = 180 * time.Second
	s.position = 40 * time.Second // deliberately older than the player's position
	s.player = audio.NewPlayer(silentPlaybackDecoder{})
	s.player.SetVolume(0)
	s.player.Seek(42 * time.Second)
	t.Cleanup(s.stop)
	return s
}

func TestPlaybackFailureKeepsCurrentSongAndSavesHeardPosition(t *testing.T) {
	d := failedPlaybackDecoder(t)
	for _, mode := range []string{"", "one", "shuffle", "loop"} {
		t.Run(mode, func(t *testing.T) {
			s := pausedPlaybackApp(t, "netease")
			s.cfg.PlayMode = mode
			s.playing = true
			s.playbackEnded(s.playID, s.player, d)
			if s.queueIndex != 0 || s.current.ID != "one" || s.playID != 0 || s.loading || s.playing || s.position != 42*time.Second || s.status != d.Err().Error() {
				t.Fatalf("failed playback advanced or lost its position: index=%d id=%d position=%s status=%q", s.queueIndex, s.playID, s.position, s.status)
			}
			restored := newApp(context.Background(), library.Config{}, s.configPath)
			if restored.current == nil || restored.current.ID != "one" || restored.position != 42*time.Second {
				t.Fatal("interrupted song and position were not persisted")
			}
		})
	}
}

func TestResumeReopensFailedOrLongPausedNetworkPlayback(t *testing.T) {
	for _, source := range []string{"netease", "bilibili", "local"} {
		for _, failed := range []bool{false, true} {
			name := source + "/long-pause"
			if failed {
				name = source + "/decode-failure"
			}
			t.Run(name, func(t *testing.T) {
				s := pausedPlaybackApp(t, source)
				s.playbackPausedAt = time.Now().Add(-2 * time.Minute)
				if failed {
					s.decoder = failedPlaybackDecoder(t)
				}
				s.toggle()
				if source == "local" && !failed {
					if s.playID != 0 || s.loading || !s.player.Playing() {
						t.Fatal("a healthy local file was reopened after pausing")
					}
					return
				}
				if s.playID != 1 || !s.loading || s.queueIndex != 0 || s.current.ID != "one" || s.position != 42*time.Second || s.player != nil || s.decoder != nil {
					t.Fatalf("resume did not reopen the same song at the heard position: id=%d index=%d position=%s", s.playID, s.queueIndex, s.position)
				}
			})
		}
	}
}

func TestShortPauseResumesExistingNetworkPlayer(t *testing.T) {
	s := pausedPlaybackApp(t, "netease")
	p := s.player
	s.playbackPausedAt = time.Now().Add(-time.Second)
	s.toggle()
	if s.player != p || s.playID != 0 || s.loading || !p.Playing() || !s.playbackPausedAt.IsZero() {
		t.Fatal("a short pause discarded the healthy buffered player")
	}
}
