package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/BMingSY/go-music-dl/internal/download"
	"github.com/guohuiyuan/music-lib/model"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/rand/v2"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/BMingSY/go-music-dl/internal/lyrics"
	"github.com/BMingSY/go-music-dl/internal/playback"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"
)

type app struct {
	previewInfo                                                       ui.Rect
	visualEpoch                                                       time.Time
	discAngle                                                         float64
	discFrame                                                         time.Time
	previewCover                                                      []byte
	previewLyrics                                                     string
	currentCollection                                                 *library.Collection
	editPlaylistID, playlistDescription, playlistCover, playlistError string
	importLink                                                        string
	importBusy                                                        bool
	queueOpen, playerVolumeOpen, speedOpen                            bool
	speed                                                             float64
	queryFocused                                                      bool
	captureShortcut                                                   string
	songMenu                                                          string
	songMenuOpen, toolsOpen                                           bool
	miniVolumeOpen                                                    bool
	selected                                                          map[string]bool
	searchKind                                                        string
	collections                                                       []library.Collection
	categories                                                        []model.PlaylistCategory
	collectionTitle, collectionKind, collectionID                     string
	scroll                                                            ui.ScrollState
	modal, settingsTab, playlistName                                  string
	localPlaylistID                                                   string
	pendingTracks                                                     []library.Track
	pageNumber                                                        int
	batch                                                             bool
	checked                                                           map[string]bool
	images                                                            map[string]*ui.Bitmap
	imagePending                                                      map[string]bool
	imageSlots                                                        chan struct{}
	downloads                                                         *download.Queue
	jobs                                                              []download.Job
	loginID                                                           uint64
	loginCancel                                                       context.CancelFunc
	loginSource, loginStatus                                          string
	loginImage                                                        *ui.Bitmap
	playerExpanded, lyricOpen, sourceCollapsed                        bool
	lyricScroll                                                       ui.ScrollState
	lastLyric                                                         int
	searchCancel                                                      context.CancelFunc
	sortMode                                                          string
	sortReverse                                                       bool

	ctx                         context.Context
	cfg                         library.Config
	configPath                  string
	win, mini                   *mygo.Window
	tray                        *mygo.Tray
	quitting                    bool
	source, query, page, status string
	results, queue              []library.Track
	queueIndex                  int
	current                     *library.Track
	cover                       *ui.Bitmap
	lines                       []lyrics.Line
	list                        ui.ListState
	player                      *audio.Player
	decoder                     *playback.Decoder
	playing, loading, searching bool
	position, duration          time.Duration
	searchID, playID            uint64
	playCancel                  context.CancelFunc
	updateBusy                  bool
	updateProgress              float64
}

func newApp(ctx context.Context, cfg library.Config, path string) *app {
	if cfg.LyricSize < 14 {
		cfg.LyricSize = 24
	}
	if cfg.Shortcuts == nil {
		cfg.Shortcuts = defaultShortcuts()
	}
	return &app{ctx: ctx, cfg: cfg, configPath: path, source: "netease", page: "library", queueIndex: -1, speed: 1,
		selected: map[string]bool{"netease": true, "bilibili": true}, searchKind: "song", pageNumber: 1, settingsTab: "general", checked: map[string]bool{}, images: map[string]*ui.Bitmap{}, imagePending: map[string]bool{}, imageSlots: make(chan struct{}, 6), lastLyric: -1,
	}
}

func (s *app) ready() {
	audio.SetLatency(audio.LatencyPlayback)
	s.downloads = download.New(s.ctx, filepath.Join(filepath.Dir(s.configPath), "downloads.json"))
	s.win = mygo.NewWindow(mygo.WindowOptions{Title: "music-dl", Width: 1200, Height: 820, MinWidth: 940, MinHeight: 620, StateKey: "main", Content: ui.View(s.view)})
	s.mini = mygo.NewWindow(mygo.WindowOptions{Title: "MusicDL Mini", Width: 236, Height: 50, MinWidth: 236, MaxWidth: 236, MinHeight: 50, MaxHeight: 50, Frameless: true, AlwaysOnTop: true, SkipTaskbar: true, DisableResize: true, DisableMaximize: true, DisableFullScreen: true, StateKey: "mini", Content: ui.View(s.miniView)})
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

func (s *app) update(fn func()) {
	mygo.RunOnMain(func() {
		if s.quitting {
			return
		}
		fn()
		s.redraw()
	})
}
func (s *app) redraw() {
	if s.win != nil {
		s.win.Invalidate()
	}
	if s.mini != nil {
		s.mini.Invalidate()
	}
}
func (s *app) tick() {
	if s.downloads != nil {
		s.jobs = s.downloads.Snapshot()
	}
	if s.player == nil {
		return
	}
	s.position = s.player.Position()
	s.playing = s.player.Playing()
	if err := audio.Err(); err != nil {
		s.status = "音频输出不可用，请检查系统声音设备。"
	}
}
func (s *app) toggleMini() {
	if s.mini == nil {
		return
	}
	if s.mini.IsVisible() {
		s.mini.Hide()
	} else {
		s.mini.Show()
	}
}
func (s *app) save() {
	if err := library.SaveConfig(s.configPath, s.cfg); err != nil {
		s.status = "保存设置失败: " + err.Error()
	}
}

func (s *app) selectSource(source string) {
	s.source = source
	s.page = "library"
	s.selected = map[string]bool{source: true}
	s.collections = nil
	s.categories = nil
	s.collectionTitle = ""
	s.pageNumber = 1
	s.results = nil
	s.searchID++
	s.searching = false
	if source == "local" {
		s.search()
	}
}

func (s *app) play(index int, tracks []library.Track) {
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
	s.position = 0
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
		return
	}
	if s.player.Playing() {
		s.player.Pause()
	} else {
		s.player.Play()
	}
	s.playing = s.player.Playing()
	s.syncMedia()
	s.redraw()
}
func (s *app) seek(pos time.Duration) {
	if s.player == nil || s.decoder == nil || s.duration <= 0 {
		return
	}
	pos = max(0, min(s.duration, pos))
	s.decoder.Seek(func() { s.player.Seek(pos) })
	s.position = pos
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
func (s *app) isFavorite(t library.Track) bool {
	for _, f := range s.cfg.Favorites {
		if f.Key() == t.Key() {
			return true
		}
	}
	return false
}
func (s *app) favorite(t library.Track) {
	for i, f := range s.cfg.Favorites {
		if f.Key() == t.Key() {
			s.cfg.Favorites = append(s.cfg.Favorites[:i], s.cfg.Favorites[i+1:]...)
			s.save()
			return
		}
	}
	s.cfg.Favorites = append(s.cfg.Favorites, t)
	s.save()
}

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

func (s *app) checkUpdate() {
	if s.updateBusy {
		return
	}
	if !mygo.Updater.Enabled() {
		s.status = "此构建无法更新，请使用安装包并确认安装目录可写。"
		return
	}
	s.updateBusy = true
	s.status = "正在检查更新…"
	go func() {
		up, err := mygo.Updater.Check(s.ctx)
		if err != nil {
			s.update(func() { s.updateBusy = false; s.status = "检查更新失败: " + err.Error() })
			return
		}
		if up == nil {
			s.update(func() { s.updateBusy = false; s.status = "已经是最新版本" })
			return
		}
		res, err := mygo.Dialog.Message(mygo.MessageOptions{Parent: s.win, Message: "发现新版本 " + up.Version, Detail: "下载完成后将自动重启更新。", Buttons: []string{"下载并重启", "稍后"}})
		if err != nil || res.Button != 0 {
			s.update(func() { s.updateBusy = false })
			return
		}
		err = up.Install(s.ctx, func(n, total int64) {
			s.update(func() {
				if total > 0 {
					s.updateProgress = float64(n) / float64(total)
				}
				s.status = fmt.Sprintf("正在下载更新 %.0f%%", s.updateProgress*100)
			})
		})
		s.update(func() {
			s.updateBusy = false
			if err != nil {
				s.status = "更新失败: " + err.Error()
				return
			}
			s.save()
			mygo.App.Relaunch()
		})
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

func caption(t *library.Track) string {
	if t == nil {
		return "选择一首歌开始播放"
	}
	return strings.TrimSpace(t.Name + " · " + t.Artist)
}
