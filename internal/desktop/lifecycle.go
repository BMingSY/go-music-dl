package desktop

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"path/filepath"
	"runtime"
	"time"

	"github.com/BMingSY/go-music-dl/internal/download"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"
)

func (s *app) ready() {
	audio.SetLatency(audio.LatencyPlayback)
	s.downloads = download.New(s.ctx, filepath.Join(filepath.Dir(s.configPath), "downloads.json"))
	s.downloads.Resume(s.cfg)
	s.win = mygo.NewWindow(mygo.WindowOptions{Title: "music-dl", Width: 1200, Height: 820, MinWidth: 940, MinHeight: 620, StateKey: "main", Content: ui.View(s.view)})
	s.mini = mygo.NewWindow(mygo.WindowOptions{Title: "MusicDL Mini", Width: 236, Height: 50, MinWidth: 236, MaxWidth: 236, MinHeight: 50, MaxHeight: 50, Frameless: true, AlwaysOnTop: true, SkipTaskbar: true, DisableResize: true, DisableMaximize: true, DisableFullScreen: true, StateKey: "mini", Content: ui.View(s.miniView)})
	// Keep the new compact dimensions when restoring an older saved window.
	s.mini.SetSize(236, 50)
	s.mini.OnClose(func(e *mygo.CloseEvent) {
		if !s.quitting {
			e.PreventDefault()
			s.mini.Hide()
		}
	})
	s.win.OnClose(func(e *mygo.CloseEvent) {
		if !s.quitting && s.tray != nil {
			e.PreventDefault()
			s.win.Hide()
			s.mini.Show()
		} else if !s.quitting {
			mygo.App.Quit()
		}
	})
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: "显示主窗口", Click: func(*mygo.MenuItem, *mygo.Window) { s.win.Show(); s.win.Focus() }},
		{Label: "显示 / 隐藏悬浮窗", Click: func(*mygo.MenuItem, *mygo.Window) { s.toggleMini() }},
		{Label: "播放 / 暂停", Click: func(*mygo.MenuItem, *mygo.Window) { s.toggle() }},
		{Label: "上一首", Click: func(*mygo.MenuItem, *mygo.Window) { s.step(-1) }},
		{Label: "下一首", Click: func(*mygo.MenuItem, *mygo.Window) { s.step(1) }},
		mygo.Separator(), {Role: mygo.RoleQuit, Label: "退出 MusicDL"},
	})
	var err error
	s.tray, err = mygo.NewTray(mygo.TrayOptions{Icon: trayIcon(), IconIsTemplate: runtime.GOOS == "darwin", ToolTip: "music-dl"})
	if err != nil {
		s.status = "托盘不可用，关闭主窗口将退出。"
	} else {
		s.tray.OnClick(func() { s.win.Show(); s.win.Focus() })
		s.tray.OnRightClick(func() { s.tray.SetMenu(menu); defer s.tray.SetMenu(nil); s.tray.PopUpMenu() })
	}
	if runtime.GOOS == "darwin" {
		mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{{Role: mygo.RoleAppMenu}, {Role: mygo.RoleEditMenu}, {Label: "播放器", Submenu: []*mygo.MenuItem{
			{Label: "播放 / 暂停", Accelerator: "CmdOrCtrl+Shift+P", Click: func(*mygo.MenuItem, *mygo.Window) { s.toggle() }},
			{Label: "悬浮窗", Click: func(*mygo.MenuItem, *mygo.Window) { s.toggleMini() }},
			{Label: "检查更新", Click: func(*mygo.MenuItem, *mygo.Window) { s.checkUpdate() }},
		}}}))
	}
	mygo.NowPlaying.OnCommand(func(cmd mygo.MediaCommand, pos time.Duration) {
		switch cmd {
		case mygo.MediaPlay:
			if !s.playing {
				s.toggle()
			}
		case mygo.MediaPause:
			if s.playing {
				s.toggle()
			}
		case mygo.MediaNext:
			s.step(1)
		case mygo.MediaPrevious:
			s.step(-1)
		case mygo.MediaSeek:
			s.seek(pos)
		}
	})
	go func() {
		t := time.NewTicker(200 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-t.C:
				s.update(func() { s.tick() })
			}
		}
	}()
}

func trayIcon() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	ink := color.NRGBA{R: 34, G: 139, B: 105, A: 255}
	if runtime.GOOS == "darwin" {
		ink = color.NRGBA{A: 255}
	}
	for _, r := range []image.Rectangle{image.Rect(10, 6, 13, 25), image.Rect(23, 3, 26, 22), image.Rect(10, 3, 26, 7), image.Rect(4, 22, 13, 28), image.Rect(17, 19, 26, 25)} {
		draw.Draw(img, r, image.NewUniform(ink), image.Point{}, draw.Src)
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}
