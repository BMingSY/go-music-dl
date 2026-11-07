package desktop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
)

type playbackSession struct {
	Queue      []library.Track `json:"queue"`
	Index      int             `json:"index"`
	PositionMS int64           `json:"position_ms"`
	Speed      float64         `json:"speed"`
}

func (s *app) playbackPath() string {
	if s.configPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(s.configPath), "playback.json")
}

func (s *app) savePlayback() error {
	path := s.playbackPath()
	if path == "" || s.current == nil {
		return nil
	}
	state := playbackSession{Queue: append([]library.Track(nil), s.queue...), Index: -1, Speed: s.speed}
	for i := range state.Queue {
		if state.Queue[i].Key() == s.current.Key() {
			state.Index = i
		}
		if state.Queue[i].Source != "local" {
			state.Queue[i].URL = ""
		}
	}
	if state.Index < 0 {
		track := *s.current
		if track.Source != "local" {
			track.URL = ""
		}
		state.Queue = append([]library.Track{track}, state.Queue...)
		state.Index = 0
	}
	if s.duration > 0 {
		state.Queue[state.Index].Duration = int(s.duration.Seconds())
	}
	position := s.position
	if s.player != nil {
		position = s.player.Position()
	}
	state.PositionMS = max(0, position.Milliseconds())
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".playback-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	if e := f.Close(); err == nil {
		err = e
	}
	if err == nil {
		err = os.Rename(f.Name(), path)
	}
	if err == nil {
		s.playbackSaved = time.Now()
	}
	return err
}

func (s *app) restorePlayback() {
	if s.playbackPath() == "" {
		return
	}
	data, err := os.ReadFile(s.playbackPath())
	if err != nil {
		return
	}
	var state playbackSession
	if json.Unmarshal(data, &state) != nil || state.Index < 0 || state.Index >= len(state.Queue) {
		return
	}
	current := state.Queue[state.Index]
	if current.Source != "netease" && current.Source != "bilibili" && current.Source != "local" {
		return
	}
	for i := range state.Queue {
		if state.Queue[i].Source != "local" {
			state.Queue[i].URL = ""
		}
	}
	s.queue = state.Queue
	s.queueIndex = state.Index
	s.current = &s.queue[state.Index]
	s.duration = time.Duration(max(0, current.Duration)) * time.Second
	s.position = resumePosition(time.Duration(max(0, state.PositionMS))*time.Millisecond, s.duration)
	if state.Speed >= .5 && state.Speed <= 2 {
		s.speed = state.Speed
	}
	s.playerExpanded = true
	s.playing = false
	s.playbackSaved = time.Now()
}

func resumePosition(position, duration time.Duration) time.Duration {
	if position < 0 || duration > 0 && position >= duration {
		return 0
	}
	return position
}
