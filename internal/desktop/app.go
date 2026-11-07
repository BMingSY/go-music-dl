package desktop

import (
	"context"
	"strings"
	"time"

	"github.com/BMingSY/go-music-dl/internal/download"
	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/BMingSY/go-music-dl/internal/lyrics"
	"github.com/BMingSY/go-music-dl/internal/playback"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"
)

type app struct {
	miniDrag                                                          miniDragState
	playbackSaved                                                     time.Time
	utilityPage                                                       int
	downloadTab, downloadFilter                                       string
	toastSeen                                                         string
	toastStarted                                                      time.Time
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
	queueScroll                                                       ui.ScrollState
	queueReveal                                                       bool
	revealTrack                                                       string
	previewActionsOpen                                                bool
	speed                                                             float64
	queryFocused                                                      bool
	captureShortcut                                                   string
	songMenu                                                          string
	songMenuOpen, toolsOpen                                           bool
	miniVolumeOpen                                                    bool
	selected                                                          map[string]bool
	searchKind                                                        string
	collections                                                       []library.Collection
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
	pendingUpdate               *mygo.Update
	updateStatus                string
	updateProgress              float64
}

func newApp(ctx context.Context, cfg library.Config, path string) *app {
	if cfg.LyricSize < 14 {
		cfg.LyricSize = 24
	}
	if cfg.Shortcuts == nil {
		cfg.Shortcuts = defaultShortcuts()
	}
	s := &app{ctx: ctx, cfg: cfg, configPath: path, source: "netease", page: "library", queueIndex: -1, speed: 1,
		selected: map[string]bool{"netease": true, "bilibili": true}, searchKind: "song", pageNumber: 1, settingsTab: "general", checked: map[string]bool{}, images: map[string]*ui.Bitmap{}, imagePending: map[string]bool{}, imageSlots: make(chan struct{}, 6), lastLyric: -1,
	}
	s.restorePlayback()
	return s
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

func (s *app) save() {
	if err := library.SaveConfig(s.configPath, s.cfg); err != nil {
		s.status = "保存设置失败: " + err.Error()
	} else if err := s.savePlayback(); err != nil {
		s.status = "保存播放进度失败: " + err.Error()
	}
}

func caption(t *library.Track) string {
	if t == nil {
		return "选择一首歌开始播放"
	}
	return strings.TrimSpace(t.Name + " · " + t.Artist)
}
