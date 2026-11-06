package library

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/guohuiyuan/music-lib/model"
	"github.com/guohuiyuan/music-lib/netease"
)

type Playlist struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Tracks      []Track `json:"tracks"`
	Description string  `json:"description,omitempty"`
	Cover       string  `json:"cover,omitempty"`
	Source      string  `json:"source,omitempty"`
	ExternalID  string  `json:"external_id,omitempty"`
	Kind        string  `json:"kind,omitempty"`
	Link        string  `json:"link,omitempty"`
}
type Collection struct {
	model.Playlist
	Kind string
}
type Catalog struct {
	Tracks      []Track
	Collections []Collection
	Categories  []model.PlaylistCategory
	Container   *Collection
}
type collectionProvider interface {
	SearchPlaylist(string) ([]model.Playlist, error)
	GetPlaylistSongs(string) ([]model.Song, error)
	ParsePlaylist(string) (*model.Playlist, []model.Song, error)
	CreateQRLogin() (*model.QRLoginSession, error)
	CheckQRLogin(string) (*model.QRLoginResult, error)
}

func Browse(ctx context.Context, source, kind, query, id string, page int, cfg Config) (Catalog, error) {
	var result Catalog
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if kind == "song" {
		return searchCatalog(ctx, source, query, cfg)
	}
	p, err := provider(source, cfg)
	if err != nil {
		return result, err
	}
	cp := p.(collectionProvider)
	var songs []model.Song
	var lists []model.Playlist
	switch kind {
	case "playlist":
		if id != "" {
			songs, err = cp.GetPlaylistSongs(id)
		} else if isURL(query) {
			var meta *model.Playlist
			meta, songs, err = cp.ParsePlaylist(query)
			if meta != nil {
				result.Container = &Collection{Playlist: *meta, Kind: "playlist"}
			}
		} else {
			lists, err = cp.SearchPlaylist(query)
		}
	case "album":
		n, ok := p.(*netease.Netease)
		if !ok {
			return result, errors.New("此来源不支持专辑搜索")
		}
		if id != "" {
			songs, err = n.GetAlbumSongs(id)
		} else if isURL(query) {
			var meta *model.Playlist
			meta, songs, err = n.ParseAlbum(query)
			if meta != nil {
				result.Container = &Collection{Playlist: *meta, Kind: "album"}
			}
		} else {
			lists, err = n.SearchAlbum(query)
		}
	case "recommend", "mine", "categories", "category", "ranks", "rank":
		n, ok := p.(*netease.Netease)
		if !ok {
			return result, errors.New("此功能支持网易云音乐")
		}
		switch kind {
		case "recommend":
			lists, err = n.GetRecommendedPlaylists()
		case "mine":
			lists, err = n.GetUserPlaylists(max(page, 1), 30)
		case "categories":
			result.Categories, err = n.GetPlaylistCategories()
		case "category":
			lists, err = n.GetCategoryPlaylists(id, max(page, 1), 30)
		case "ranks":
			var ranks []model.TopList
			ranks, err = n.GetTopLists()
			for _, r := range ranks {
				result.Collections = append(result.Collections, Collection{Playlist: model.Playlist{ID: r.ID, Name: r.Name, Cover: r.Cover, Description: r.Description, Source: source, TrackCount: r.TrackCount}, Kind: "rank"})
			}
		case "rank":
			songs, err = n.GetTopListSongs(id)
		}
	default:
		return result, errors.New("不支持的浏览类型")
	}
	if err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return Catalog{}, err
	}
	for _, t := range songs {
		if t.Source == source {
			result.Tracks = append(result.Tracks, Track{Song: t})
		}
	}
	for _, l := range lists {
		if l.Source == source {
			k := "playlist"
			if kind == "album" {
				k = "album"
			}
			result.Collections = append(result.Collections, Collection{Playlist: l, Kind: k})
		}
	}
	return result, nil
}

func isURL(s string) bool { return strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://") }

func CreateLogin(source string, cfg Config) (*model.QRLoginSession, error) {
	p, err := provider(source, cfg)
	if err != nil {
		return nil, err
	}
	return p.(collectionProvider).CreateQRLogin()
}
func CheckLogin(source, key string, cfg Config) (*model.QRLoginResult, error) {
	p, err := provider(source, cfg)
	if err != nil {
		return nil, err
	}
	return p.(collectionProvider).CheckQRLogin(key)
}
func LoginCookie(r *model.QRLoginResult) string {
	if r == nil || r.Status != model.QRLoginStatusSuccess {
		return ""
	}
	if strings.TrimSpace(r.Cookie) != "" {
		return strings.TrimSpace(r.Cookie)
	}
	var keys []string
	for k, v := range r.Cookies {
		if k != "" && v != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	var values []string
	for _, k := range keys {
		values = append(values, k+"="+r.Cookies[k])
	}
	return strings.Join(values, "; ")
}
