package desktop

import (
	"context"
	"sort"
	"strings"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo/ui"
)

func (s *app) selectSource(source string) {
	s.source = source
	s.page = "library"
	s.selected = map[string]bool{source: true}
	s.collections = nil
	s.collectionTitle = ""
	s.pageNumber = 1
	s.results = nil
	s.searchID++
	s.searching = false
	if source == "local" {
		s.search()
	}
}

func (s *app) search() {
	if link, source, kind := library.ShareQuery(s.query); source != "" {
		s.query = link
		s.selected = map[string]bool{source: true}
		if kind != "" {
			s.searchKind = kind
		}
	}
	s.browse(s.searchKind, "", "")
}

func (s *app) browse(kind, source, id string) {
	if s.searchCancel != nil {
		s.searchCancel()
	}
	ctx, cancel := context.WithCancel(s.ctx)
	s.searchCancel = cancel
	s.searchID++
	seq := s.searchID
	s.searching = true
	s.status = "正在查找…"
	s.page = "library"
	s.pageNumber = 1
	s.scroll = ui.ScrollState{}
	s.collectionTitle = ""
	s.currentCollection = nil
	s.collectionKind = kind
	s.collectionID = id
	cfg, query := s.cfg, strings.TrimSpace(s.query)
	var sources []string
	if source != "" {
		sources = []string{source}
	} else {
		for _, src := range []string{"netease", "bilibili", "local"} {
			if s.selected[src] && (kind == "song" || src != "local") && (kind != "album" || src == "netease") {
				sources = append(sources, src)
			}
		}
	}
	if len(sources) == 0 {
		s.searching = false
		s.status = "请选择支持此功能的音乐来源"
		return
	}
	s.source = ""
	if len(sources) == 1 {
		s.source = sources[0]
	}
	go func() {
		type result struct {
			data library.Catalog
			err  error
		}
		ch := make(chan result, len(sources))
		for _, src := range sources {
			go func() { r, e := library.Browse(ctx, src, kind, query, id, 1, cfg); ch <- result{r, e} }()
		}
		var all library.Catalog
		var errs []string
		for range sources {
			r := <-ch
			if r.err != nil {
				errs = append(errs, r.err.Error())
			} else {
				all.Tracks = append(all.Tracks, r.data.Tracks...)
				all.Collections = append(all.Collections, r.data.Collections...)
				if r.data.Container != nil {
					all.Container = r.data.Container
				}
			}
		}
		s.update(func() {
			if seq != s.searchID || ctx.Err() != nil {
				return
			}
			s.searching = false
			s.results = all.Tracks
			s.collections = all.Collections
			s.checked = map[string]bool{}
			s.list = ui.ListState{}
			s.status = strings.Join(errs, "；")
			if all.Container != nil {
				s.currentCollection = all.Container
				s.collectionTitle = all.Container.Name
				s.collectionKind = all.Container.Kind
				s.collectionID = all.Container.ID
				s.searchKind = all.Container.Kind
			}
			if len(all.Tracks)+len(all.Collections) == 0 && s.status == "" {
				s.status = "没有找到结果"
			}
			s.redraw()
		})
	}()
}

func (s *app) openCollection(c library.Collection) {
	s.browse(c.Kind, c.Source, c.ID)
	s.collectionTitle = c.Name
	s.currentCollection = &c
}

func (s *app) localMusic() {
	s.query = ""
	s.searchKind = "song"
	s.selectSource("local")
	s.pageNumber = 1
}

func (s *app) shownTracks() []library.Track {
	tracks := s.results
	switch s.page {
	case "favorites":
		tracks = s.cfg.Favorites
	case "queue":
		tracks = s.queue
	case "history":
		tracks = s.cfg.History
	case "collection":
		for _, p := range s.cfg.Playlists {
			if p.ID == s.localPlaylistID {
				tracks = p.Tracks
				break
			}
		}
	}
	if s.sortMode != "" {
		tracks = append([]library.Track(nil), tracks...)
		sort.SliceStable(tracks, func(i, j int) bool {
			a, b := tracks[i], tracks[j]
			var less bool
			switch s.sortMode {
			case "duration":
				less = a.Duration < b.Duration
			case "quality":
				less = a.Bitrate < b.Bitrate
			case "size":
				less = a.Size < b.Size
			default:
				less = a.Name < b.Name
			}
			if s.sortReverse {
				a, b = b, a
				switch s.sortMode {
				case "duration":
					less = a.Duration < b.Duration
				case "quality":
					less = a.Bitrate < b.Bitrate
				case "size":
					less = a.Size < b.Size
				default:
					less = a.Name < b.Name
				}
			}
			return less
		})
	}
	return tracks
}

func (s *app) cancelBrowse() {
	if s.searchCancel != nil {
		s.searchCancel()
		s.searchCancel = nil
	}
	s.searchID++
	s.searching = false
}
