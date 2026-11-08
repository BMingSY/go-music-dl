package desktop

import (
	"bytes"

	"github.com/BMingSY/go-music-dl/internal/lyrics"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// Metadata is refreshed after reopening a stream. Keep the displayed content
// until a valid replacement is ready; empty results can mean a network failure.
func (s *app) applyPlaybackMetadata(lyric string, cover []byte) {
	if s.previewLyrics == "" && lyric != "" && lyric != s.lyricText {
		if lines := lyrics.Parse(lyric); len(lines) > 0 {
			s.lines = lines
			s.lyricText = lyric
			s.lastLyric = -1
		}
	}
	if len(s.previewCover) == 0 && len(cover) > 0 && !bytes.Equal(cover, s.coverData) {
		if bitmap, err := ui.DecodeBitmap(cover); err == nil {
			s.cover = bitmap
			s.coverData = bytes.Clone(cover)
		}
	}
}

func (s *app) syncNowPlayingMetadata() {
	if s.current == nil {
		return
	}
	track := s.current
	cover := s.coverData
	if len(s.previewCover) > 0 {
		cover = s.previewCover
	}
	info := mygo.NowPlayingInfo{Title: track.Name, Artist: track.Artist, Album: track.Album, Artwork: cover, Duration: s.duration, Commands: mygo.MediaPlay | mygo.MediaPause | mygo.MediaNext | mygo.MediaPrevious | mygo.MediaSeek}
	old, showing := mygo.NowPlaying.Info()
	if showing && old.Title == info.Title && old.Artist == info.Artist && old.Album == info.Album && old.Duration == info.Duration && old.Commands == info.Commands && bytes.Equal(old.Artwork, info.Artwork) {
		return
	}
	mygo.NowPlaying.Set(info)
}
