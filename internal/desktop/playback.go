package desktop

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/BMingSY/go-music-dl/internal/lyrics"
	"github.com/BMingSY/go-music-dl/internal/playback"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"
)

func (s *app) tick() {
	if s.downloads != nil {
		s.jobs = s.downloads.Snapshot()
	}
	if s.player == nil {
		return
	}
	s.position = s.player.Position()
	s.playing = s.player.Playing()
	if s.playing && time.Since(s.playbackSaved) >= 10*time.Second {
		_ = s.savePlayback()
	}
	if err := audio.Err(); err != nil {
		s.status = "音频输出不可用，请检查系统声音设备。"
	}
}

func (s *app) play(index int, tracks []library.Track) {
	s.playFrom(index, tracks, 0)
}

func (s *app) playFrom(index int, tracks []library.Track, position time.Duration) {
	if index < 0 || index >= len(tracks) {
		return
	}
	s.previewCover = nil
	s.previewLyrics = ""
	s.stop()
	s.playID++
	id := s.playID
	s.queue = append([]library.Track(nil), tracks...)
	s.queueIndex = index
	track := tracks[index]
	s.current = &track
	s.playerExpanded = true
	s.cover = nil
	s.lines = nil
	s.position = resumePosition(position, time.Duration(track.Duration)*time.Second)
	s.duration = time.Duration(track.Duration) * time.Second
	s.loading = true
	s.status = "正在准备播放…"
	ctx, cancel := context.WithCancel(s.ctx)
	s.playCancel = cancel
	cfg := s.cfg
	go func() {
		ffmpeg, err := playback.FindFFmpeg(cfg.FFmpeg)
		if err != nil {
			s.playError(id, err)
			return
		}
		r, err := library.Resolve(ctx, track, cfg)
		if err != nil {
			s.playError(id, err)
			return
		}
		if ctx.Err() != nil {
			return
		}
		input := playback.Input{Location: r.Input, Headers: r.Headers, Duration: time.Duration(track.Duration) * time.Second}
		input.Duration = playback.ProbeDuration(ctx, ffmpeg, input)
		if ctx.Err() != nil {
			return
		}
		d, err := playback.Open(ffmpeg, input)
		if err != nil {
			s.playError(id, err)
			return
		}
		if ctx.Err() != nil {
			d.Close()
			return
		}
		context.AfterFunc(ctx, func() { _ = d.Close() })
		s.update(func() {
			if id != s.playID || ctx.Err() != nil {
				d.Close()
				return
			}
			s.decoder = d
			s.player = audio.NewPlayer(d)
			s.duration = input.Duration
			s.position = resumePosition(position, s.duration)
			if s.position > 0 {
				d.Seek(func() { s.player.Seek(s.position) })
			}
			s.loading = false
			s.status = ""
			s.recordHistory(track)
			p := s.player
			p.SetVolume(s.cfg.Volume)
			p.SetSpeed(s.speed)
			p.OnEnd(func() {
				s.update(func() {
					if id != s.playID || s.player != p {
						return
					}
					s.playing = false
					if err := d.Err(); err != nil {
						s.status = err.Error()
						return
					}
					if s.cfg.PlayMode == "one" {
						s.play(s.queueIndex, s.queue)
					} else if s.cfg.PlayMode == "shuffle" && len(s.queue) > 0 {
						s.play(rand.IntN(len(s.queue)), s.queue)
					} else if s.queueIndex+1 < len(s.queue) {
						s.play(s.queueIndex+1, s.queue)
					} else if s.cfg.PlayMode == "loop" {
						s.play(0, s.queue)
					} else {
						s.status = "播放完毕"
						s.syncMedia()
					}
				})
			})
			p.Play()
			s.playing = true
			if s.cfg.FloatingLyrics && s.mini != nil {
				s.mini.Show()
			}
			mygo.NowPlaying.Set(mygo.NowPlayingInfo{Title: track.Name, Artist: track.Artist, Album: track.Album, Duration: s.duration, Commands: mygo.MediaPlay | mygo.MediaPause | mygo.MediaNext | mygo.MediaPrevious | mygo.MediaSeek})
			s.syncMedia()
		})
		lyric, cover := library.Metadata(ctx, track, cfg)
		if ctx.Err() != nil {
			return
		}
		s.update(func() {
			if id != s.playID {
				return
			}
			if s.previewLyrics == "" {
				s.lines = lyrics.Parse(lyric)
				s.lastLyric = -1
			}
			if len(cover) > 0 && len(s.previewCover) == 0 {
				s.cover, _ = ui.DecodeBitmap(cover)
			}
			mygo.NowPlaying.Set(mygo.NowPlayingInfo{Title: track.Name, Artist: track.Artist, Album: track.Album, Artwork: cover, Duration: s.duration, Commands: mygo.MediaPlay | mygo.MediaPause | mygo.MediaNext | mygo.MediaPrevious | mygo.MediaSeek})
			s.syncMedia()
		})
	}()
}

func (s *app) playError(id uint64, err error) {
	s.update(func() {
		if id == s.playID {
			s.loading = false
			s.playing = false
			s.status = err.Error()
		}
	})
}

func (s *app) stop() {
	if s.playCancel != nil {
		s.playCancel()
		s.playCancel = nil
	}
	if s.player != nil {
		s.player.Close()
		s.player = nil
	}
	if s.decoder != nil {
		s.decoder.Close()
		s.decoder = nil
	}
	s.playing = false
}

func (s *app) toggle() {
	if s.player == nil {
		if !s.loading && s.current != nil && s.queueIndex >= 0 && s.queueIndex < len(s.queue) {
			s.playFrom(s.queueIndex, s.queue, s.position)
		}
		return
	}
	if s.player.Playing() {
		s.player.Pause()
	} else {
		s.player.Play()
	}
	s.playing = s.player.Playing()
	_ = s.savePlayback()
	s.syncMedia()
	s.redraw()
}

func (s *app) seek(pos time.Duration) {
	if s.current == nil || s.duration <= 0 {
		return
	}
	pos = max(0, min(s.duration, pos))
	if s.decoder != nil && s.player != nil {
		s.decoder.Seek(func() { s.player.Seek(pos) })
	}
	s.position = pos
	_ = s.savePlayback()
	s.syncMedia()
	s.redraw()
}

func (s *app) step(delta int) {
	if len(s.queue) == 0 {
		return
	}
	next := (s.queueIndex + delta + len(s.queue)) % len(s.queue)
	s.play(next, s.queue)
}

func (s *app) syncMedia() {
	if s.player == nil {
		return
	}
	state := mygo.PlaybackPaused
	if s.player.Playing() {
		state = mygo.PlaybackPlaying
	}
	mygo.NowPlaying.SetPlayback(state, s.player.Position())
}

func (s *app) volume(v float64) {
	s.cfg.Volume = max(0, min(1, v))
	if s.player != nil {
		s.player.SetVolume(s.cfg.Volume)
	}
	s.redraw()
}
