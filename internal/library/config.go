package library

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	FloatingLyrics bool                `json:"floating_lyrics"`
	LyricFont      string              `json:"lyric_font"`
	Shortcuts      map[string]Shortcut `json:"shortcuts"`
	EmbedMetadata  bool                `json:"embed_metadata"`
	SaveLyrics     bool                `json:"save_lyrics"`
	LocalFiles     []string            `json:"local_files,omitempty"`
	NeteaseCookie  string              `json:"netease_cookie,omitempty"`
	BilibiliCookie string              `json:"bilibili_cookie,omitempty"`
	MusicDir       string              `json:"music_dir"`
	FFmpeg         string              `json:"ffmpeg,omitempty"`
	Volume         float64             `json:"volume"`
	Favorites      []Track             `json:"favorites,omitempty"`
	Playlists      []Playlist          `json:"playlists,omitempty"`
	History        []Track             `json:"history,omitempty"`
	DownloadDir    string              `json:"download_dir"`
	PageSize       int                 `json:"page_size"`
	LyricSize      float32             `json:"lyric_size"`
	PlayMode       string              `json:"play_mode"`
	ShowMascot     bool                `json:"show_mascot"`
}

type Shortcut struct {
	Key       uint16 `json:"key"`
	Modifiers uint8  `json:"modifiers"`
	Label     string `json:"label"`
}

func ConfigPath() (string, error) {
	if p := os.Getenv("MUSIC_DL_NATIVE_CONFIG"); p != "" {
		return filepath.Abs(p)
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "music-dl", "settings.json"), nil
}

func LoadConfig(path string) (Config, error) {
	home, _ := os.UserHomeDir()
	c := Config{MusicDir: filepath.Join(home, "Music"), Volume: 0.7, DownloadDir: filepath.Join(home, "Music", "music-dl"), PageSize: 30, LyricSize: 24, ShowMascot: true, FloatingLyrics: true, EmbedMetadata: true, SaveLyrics: true}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("读取设置: %w", err)
	}
	c.Volume = max(0, min(1, c.Volume))
	if c.PageSize < 20 || c.PageSize > 500 {
		c.PageSize = 30
	}
	if c.LyricSize < 12 || c.LyricSize > 48 {
		c.LyricSize = 24
	}
	return c, nil
}

func SaveConfig(path string, c Config) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".settings-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
