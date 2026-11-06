package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BMingSY/go-music-dl/internal/download"
	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/BMingSY/go-music-dl/internal/playback"
	"github.com/egoist/mygo"
)

func (s *app) saveCurrentCover() {
	if s.current == nil {
		return
	}
	t, cfg := *s.current, s.cfg
	go func() {
		_, cover := library.Metadata(s.ctx, t, cfg)
		if len(cover) == 0 {
			s.update(func() { s.status = "此歌曲暂无封面" })
			return
		}
		ext := ".jpg"
		if len(cover) > 4 && string(cover[1:4]) == "PNG" {
			ext = ".png"
		}
		path, e := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: s.win, Title: "保存封面", DefaultPath: t.Name + ext})
		if e == nil && path != "" {
			e = os.WriteFile(path, cover, 0644)
			s.update(func() {
				if e != nil {
					s.status = "保存封面失败"
				} else {
					s.status = "封面已保存"
				}
			})
		}
	}()
}
func (s *app) exportVideo() {
	if s.current == nil {
		return
	}
	t, cfg := *s.current, s.cfg
	go func() {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: s.win, Title: "导出歌词视频", DefaultPath: t.Name + ".mp4"})
		if err != nil || path == "" {
			return
		}
		if !strings.HasSuffix(strings.ToLower(path), ".mp4") {
			path += ".mp4"
		}
		s.update(func() { s.status = "正在导出歌词视频…" })
		ctx, cancel := context.WithTimeout(s.ctx, 45*time.Minute)
		defer cancel()
		ffmpeg, err := playback.FindFFmpeg(cfg.FFmpeg)
		if err == nil {
			var audioPath string
			if t.Source == "local" {
				audioPath = t.Path
			} else {
				audioPath, err = download.Fetch(ctx, t, cfg, nil)
			}
			if err == nil {
				lyric, cover := library.Metadata(ctx, t, cfg)
				err = playback.ExportVideo(ctx, ffmpeg, audioPath, path, t.Name, t.Artist, lyric, cover)
			}
		}
		s.update(func() {
			if err != nil {
				s.status = "视频导出失败: " + err.Error()
			} else {
				s.status = "视频已保存: " + filepath.Base(path)
				mygo.Shell.ShowItemInFolder(path)
			}
		})
	}()
}
