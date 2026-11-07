package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/BMingSY/go-music-dl/internal/lyrics"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/guohuiyuan/music-lib/model"
)

func runSmoke(s *app, dir, file string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(dir, "result.txt"))
	time.Sleep(time.Second)
	var err error
	mygo.RunOnMain(func() {
		if s.win.Page() != nil || s.mini.Page() != nil {
			err = fmt.Errorf("native window unexpectedly has a web page")
		}
		if s.tray == nil {
			err = fmt.Errorf("tray unavailable")
		}
	})
	if err != nil {
		return err
	}
	if file != "" {
		mygo.RunOnMain(func() {
			s.cfg.Volume = 0
			s.play(0, []library.Track{{Path: file, Song: model.Song{ID: file, Source: "local", Name: "原生音频验证", Artist: "合成测试音", Duration: 12}}})
		})
		if err := waitSmoke(s, func() bool {
			return s.player != nil && s.player.Playing() && s.player.Position() > 300*time.Millisecond
		}); err != nil {
			return err
		}
		if err := audio.Err(); err != nil {
			return fmt.Errorf("native output: %w", err)
		}
		mygo.RunOnMain(s.toggle)
		time.Sleep(500 * time.Millisecond) // drain the device's playback buffer
		var at time.Duration
		mygo.RunOnMain(func() { at = s.player.Position() })
		time.Sleep(400 * time.Millisecond)
		mygo.RunOnMain(func() {
			if s.player.Playing() || s.player.Position()-at > 100*time.Millisecond {
				err = fmt.Errorf("paused player kept advancing")
			}
			s.seek(3 * time.Second)
			s.toggle()
		})
		if err != nil {
			return err
		}
		if err := waitSmoke(s, func() bool { return s.player.Playing() && s.player.Position() > 3300*time.Millisecond }); err != nil {
			return err
		}
		if _, e := os.Stat(strings.TrimSuffix(file, filepath.Ext(file)) + ".lrc"); e == nil {
			if err := waitSmoke(s, func() bool { return len(s.lines) > 0 }); err != nil {
				return fmt.Errorf("local timed lyrics: %w", err)
			}
			mygo.RunOnMain(func() {
				s.position = s.player.Position()
				i := lyrics.Index(s.lines, s.position)
				text, _ := s.miniCaption()
				if i < 0 || text != s.lines[i].Text {
					err = fmt.Errorf("mini lyric did not follow playback after seek")
				}
				s.lyricOpen = true
			})
			if err != nil {
				return err
			}
			time.Sleep(600 * time.Millisecond)
			b, e := s.win.CapturePage()
			if e != nil {
				return e
			}
			if e = os.WriteFile(filepath.Join(dir, "lyrics.png"), b, 0600); e != nil {
				return e
			}
			mygo.RunOnMain(func() { s.lyricOpen = false })
		}
	}
	mygo.RunOnMain(func() {
		s.win.Close()
		if s.win.IsVisible() || !s.mini.IsVisible() {
			err = fmt.Errorf("closing main did not preserve mini window")
		}
	})
	if err != nil {
		return err
	}
	if file != "" {
		var before time.Duration
		mygo.RunOnMain(func() { before = s.player.Position() })
		time.Sleep(500 * time.Millisecond)
		mygo.RunOnMain(func() {
			if s.player.Position() <= before {
				err = fmt.Errorf("playback stopped with hidden main window")
			}
			if e := s.decoder.Err(); e != nil {
				err = e
			}
		})
		if err != nil {
			return err
		}
	}
	if file != "" {
		var savedPosition time.Duration
		mygo.RunOnMain(func() {
			s.position = s.player.Position()
			savedPosition = s.position
			if e := s.savePlayback(); e != nil {
				err = e
				return
			}
			s.stop()
			s.current = nil
			s.queue = nil
			s.queueIndex = -1
			s.position = 0
			s.restorePlayback()
			if s.current == nil || s.player != nil || s.playing || absDuration(s.position-savedPosition) > time.Millisecond {
				err = fmt.Errorf("saved playback did not restore paused at its previous position")
				return
			}
			s.toggle()
		})
		if err != nil {
			return err
		}
		if e := waitSmoke(s, func() bool {
			return s.player != nil && s.player.Playing() && s.player.Position() > savedPosition+200*time.Millisecond
		}); e != nil {
			return fmt.Errorf("resume saved playback: %w", e)
		}
	}
	mygo.RunOnMain(func() { s.win.Show() })
	time.Sleep(300 * time.Millisecond)
	for name, w := range map[string]*mygo.Window{"main": s.win, "mini": s.mini} {
		b, e := w.CapturePage()
		if e != nil {
			return e
		}
		if e = os.WriteFile(filepath.Join(dir, name+".png"), b, 0600); e != nil {
			return e
		}
	}
	return nil
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func waitSmoke(s *app, ready func() bool) error {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		ok := false
		var err error
		mygo.RunOnMain(func() {
			ok = ready()
			if !s.loading && s.player == nil {
				err = fmt.Errorf("playback: %s", s.status)
			}
		})
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("playback did not reach expected state before timeout")
}
