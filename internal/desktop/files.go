package desktop

import (
	"os"
	"slices"

	"github.com/egoist/mygo"
)

func (s *app) pickFolder() {
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: s.win, Title: "选择音乐文件夹", Directory: true})
		s.update(func() {
			if err != nil {
				s.status = err.Error()
			} else if len(paths) > 0 {
				s.cfg.MusicDir = paths[0]
				s.save()
				if s.source == "local" {
					s.search()
				}
			}
		})
	}()
}

func (s *app) pickFFmpeg() {
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: s.win, Title: "选择 ffmpeg 可执行文件"})
		s.update(func() {
			if err != nil {
				s.status = err.Error()
			} else if len(paths) > 0 {
				s.cfg.FFmpeg = paths[0]
				s.save()
			}
		})
	}()
}

func (s *app) pickDownloadFolder() {
	go func() {
		paths, e := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: s.win, Title: "选择下载目录", Directory: true})
		s.update(func() {
			if e == nil && len(paths) > 0 {
				s.cfg.DownloadDir = paths[0]
				s.save()
			}
		})
	}()
}

func (s *app) openDownloadDir() {
	if s.cfg.DownloadDir != "" {
		_ = os.MkdirAll(s.cfg.DownloadDir, 0755)
		_ = mygo.Shell.OpenPath(s.cfg.DownloadDir)
	}
}

func (s *app) pickLocalFiles() {
	go func() {
		paths, e := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: s.win, Title: "导入本地音乐", Multiple: true, Filters: []mygo.FileFilter{{Name: "音频文件", Extensions: []string{"mp3", "flac", "m4a", "aac", "ogg", "wav", "wma"}}}})
		if e != nil {
			return
		}
		s.update(func() {
			for _, path := range paths {
				if !slices.Contains(s.cfg.LocalFiles, path) {
					s.cfg.LocalFiles = append(s.cfg.LocalFiles, path)
				}
			}
			s.save()
			s.localMusic()
		})
	}()
}
