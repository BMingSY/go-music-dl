package library

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config belongs to the native preview; it never overwrites the legacy database.
type Config struct {
	NeteaseCookie  string  `json:"netease_cookie,omitempty"`
	BilibiliCookie string  `json:"bilibili_cookie,omitempty"`
	MusicDir       string  `json:"music_dir"`
	FFmpeg         string  `json:"ffmpeg,omitempty"`
	Volume         float64 `json:"volume"`
	Favorites      []Track `json:"favorites,omitempty"`
}

func ConfigPath() (string, error) {
	if p := os.Getenv("MUSIC_DL_NATIVE_CONFIG"); p != "" {
		return filepath.Abs(p)
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "music-dl-native-preview", "settings.json"), nil
}

func LoadConfig(path string) (Config, error) {
	home, _ := os.UserHomeDir()
	c := Config{MusicDir: filepath.Join(home, "Music"), Volume: 0.7}
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
