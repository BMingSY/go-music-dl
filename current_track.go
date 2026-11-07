package main

import "github.com/BMingSY/go-music-dl/internal/library"

func trackIndex(tracks []library.Track, key string) int {
	for i, track := range tracks {
		if track.Key() == key {
			return i
		}
	}
	return -1
}

func (s *app) locateCurrentTrack() {
	if s.current == nil {
		return
	}
	s.cancelBrowse()
	key := s.current.Key()
	tracks := s.shownTracks()
	index := trackIndex(tracks, key)
	showsSongs := s.page == "queue" || s.page == "collection" || s.page == "favorites" || s.page == "history" || s.page == "library" && len(s.collections) == 0 && s.collectionKind != "recommend" && s.collectionKind != "mine" && s.collectionKind != "ranks"
	if !showsSongs || index < 0 {
		s.page = "queue"
		s.collections = nil
		s.currentCollection = nil
		s.collectionTitle = ""
		s.collectionKind = ""
		tracks = s.shownTracks()
		index = trackIndex(tracks, key)
	}
	if index < 0 {
		return
	}
	pageSize := s.cfg.PageSize
	if pageSize < 20 {
		pageSize = 30
	}
	s.pageNumber = index/pageSize + 1
	s.toolsOpen = false
	s.songMenuOpen = false
	s.queueOpen = false
	s.revealTrack = key
}
