package desktop

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo"
)

func metadataCover(t *testing.T, value uint8) []byte {
	t.Helper()
	im := image.NewRGBA(image.Rect(0, 0, 2, 2))
	im.Set(0, 0, color.RGBA{R: value, A: 255})
	var b bytes.Buffer
	if err := png.Encode(&b, im); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestLongPauseResumeKeepsDisplayedMetadata(t *testing.T) {
	for _, override := range []bool{false, true} {
		name := "provider"
		if override {
			name = "local-override"
		}
		t.Run(name, func(t *testing.T) {
			s := pausedPlaybackApp(t, "bilibili")
			cover := metadataCover(t, 100)
			lyric := "[00:01]first\n[00:40]second"
			s.applyPlaybackMetadata(lyric, cover)
			if override {
				s.previewCover = bytes.Clone(cover)
				s.previewLyrics = lyric
			}
			bitmap, line := s.cover, &s.lines[0]
			s.lastLyric = 1
			s.lyricScroll.Y = 125
			s.duration = 180*time.Second + 500*time.Millisecond
			s.playbackPausedAt = time.Now().Add(-2 * time.Minute)
			s.toggle()
			if !s.loading || s.playID != 1 || s.position != 42*time.Second {
				t.Fatal("fixture did not reopen the paused stream")
			}
			if s.cover != bitmap || &s.lines[0] != line || s.lastLyric != 1 || s.lyricScroll.Y != 125 || !bytes.Equal(s.coverData, cover) || s.lyricText != lyric {
				t.Fatal("reopening the same song cleared or reset its displayed metadata")
			}
			if s.duration != 180*time.Second+500*time.Millisecond {
				t.Fatal("reopening discarded the previously probed duration")
			}
			if override && (!bytes.Equal(s.previewCover, cover) || s.previewLyrics != lyric) {
				t.Fatal("reopening discarded the user's local overrides")
			}
		})
	}
}

func TestChangingSongClearsPreviousMetadata(t *testing.T) {
	for _, source := range []string{"bilibili", "netease"} {
		t.Run(source, func(t *testing.T) {
			s := pausedPlaybackApp(t, "bilibili")
			cover := metadataCover(t, 100)
			s.applyPlaybackMetadata("[00:01]old song", cover)
			s.previewCover = cover
			s.previewLyrics = "[00:01]override"
			s.lastLyric = 3
			next := s.queue[1]
			next.Source = source
			if source != "bilibili" {
				next.ID = s.current.ID // Identical IDs from different providers are distinct songs.
			}
			s.play(0, []library.Track{next})
			if s.cover != nil || s.coverData != nil || s.lines != nil || s.lyricText != "" || s.previewCover != nil || s.previewLyrics != "" || s.lastLyric != -1 {
				t.Fatal("a different song inherited the previous song's metadata")
			}
		})
	}
}

func TestMetadataRefreshReusesUnchangedContentAndKeepsValidFallback(t *testing.T) {
	s := pausedPlaybackApp(t, "bilibili")
	cover := metadataCover(t, 100)
	lyric := "[00:01]first\n[00:40]second"
	s.applyPlaybackMetadata(lyric, cover)
	bitmap, line := s.cover, &s.lines[0]
	s.lastLyric = 1
	s.lyricScroll.Y = 125
	for _, refresh := range []struct {
		name, lyric string
		cover       []byte
	}{
		{"unchanged", lyric, bytes.Clone(cover)},
		{"empty-response", "", nil},
		{"invalid-response", "invalid lyrics", []byte("invalid image")},
	} {
		t.Run(refresh.name, func(t *testing.T) {
			s.applyPlaybackMetadata(refresh.lyric, refresh.cover)
			if s.cover != bitmap || &s.lines[0] != line || s.lastLyric != 1 || s.lyricScroll.Y != 125 || !bytes.Equal(s.coverData, cover) || s.lyricText != lyric {
				t.Fatal("refresh replaced unchanged content or erased valid fallback metadata")
			}
		})
	}
	newCover := metadataCover(t, 200)
	s.applyPlaybackMetadata(lyric, newCover)
	if s.cover == bitmap || !bytes.Equal(s.coverData, newCover) || &s.lines[0] != line || s.lastLyric != 1 {
		t.Fatal("a changed cover was not applied independently of unchanged lyrics")
	}
	bitmap = s.cover
	s.applyPlaybackMetadata("[00:02]new lyrics", nil)
	if s.cover != bitmap || s.lyricText != "[00:02]new lyrics" || s.lines[0].Text != "new lyrics" || s.lastLyric != -1 {
		t.Fatal("valid new lyrics were not applied independently of the retained cover")
	}
}

func TestMetadataRefreshPreservesLocalOverrides(t *testing.T) {
	s := pausedPlaybackApp(t, "bilibili")
	cover := metadataCover(t, 100)
	lyric := "[00:01]local lyrics"
	s.applyPlaybackMetadata(lyric, cover)
	s.previewCover, s.previewLyrics = cover, lyric
	bitmap, line := s.cover, &s.lines[0]
	s.lastLyric = 0
	s.applyPlaybackMetadata("[00:02]provider lyrics", metadataCover(t, 200))
	if s.cover != bitmap || &s.lines[0] != line || s.lastLyric != 0 || !bytes.Equal(s.previewCover, cover) || s.previewLyrics != lyric {
		t.Fatal("provider refresh overwrote the user's local cover or lyrics")
	}
}

func TestNowPlayingRefreshRetainsArtworkAndPlaybackState(t *testing.T) {
	mygo.NowPlaying.Clear()
	t.Cleanup(mygo.NowPlaying.Clear)
	s := pausedPlaybackApp(t, "bilibili")
	s.current.Name = "current song"
	cover := metadataCover(t, 100)
	s.applyPlaybackMetadata("[00:01]lyrics", cover)
	s.syncNowPlayingMetadata()
	mygo.NowPlaying.SetPlayback(mygo.PlaybackPaused, 42*time.Second)
	s.playbackPausedAt = time.Now().Add(-2 * time.Minute)
	s.toggle()
	s.syncNowPlayingMetadata()
	s.applyPlaybackMetadata("", nil)
	s.syncNowPlayingMetadata()
	info, showing := mygo.NowPlaying.Info()
	if !showing || !bytes.Equal(info.Artwork, cover) || info.State != mygo.PlaybackPaused || info.Position != 42*time.Second {
		t.Fatal("unchanged metadata reset the system artwork or playback state")
	}
	s.previewCover = metadataCover(t, 200)
	s.syncNowPlayingMetadata()
	info, _ = mygo.NowPlaying.Info()
	if !bytes.Equal(info.Artwork, s.previewCover) {
		t.Fatal("system artwork did not use the displayed local override")
	}
}
