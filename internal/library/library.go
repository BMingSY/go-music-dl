package library

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dhowden/tag"
	"github.com/guohuiyuan/music-lib/bilibili"
	"github.com/guohuiyuan/music-lib/model"
	"github.com/guohuiyuan/music-lib/netease"
)

const UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/134.0 Safari/537.36"

type Track struct {
	model.Song
	Path string `json:"path,omitempty"`
}

func (t Track) Key() string { return t.Source + ":" + t.ID }

type Resolved struct {
	Track   Track
	Input   string
	Headers http.Header
	Lyrics  string
	Cover   []byte
}

type Provider interface {
	Search(string) ([]model.Song, error)
	Parse(string) (*model.Song, error)
	GetDownloadURL(*model.Song) (string, error)
	GetLyrics(*model.Song) (string, error)
}

func provider(source string, cfg Config) (Provider, error) {
	switch source {
	case "netease":
		return netease.New(cfg.NeteaseCookie), nil
	case "bilibili":
		return bilibili.New(cfg.BilibiliCookie), nil
	default:
		return nil, fmt.Errorf("不支持的来源: %s", source)
	}
}

func Search(ctx context.Context, source, query string, cfg Config) ([]Track, error) {
	result, err := searchCatalog(ctx, source, query, cfg)
	return result.Tracks, err
}

func searchCatalog(ctx context.Context, source, query string, cfg Config) (Catalog, error) {
	if source != "local" {
		p, err := provider(source, cfg)
		if err != nil {
			return Catalog{}, err
		}
		return searchRemote(ctx, source, strings.TrimSpace(query), p)
	}
	tracks, err := searchLocal(ctx, query, cfg)
	return Catalog{Tracks: tracks}, err
}

func searchLocal(ctx context.Context, query string, cfg Config) ([]Track, error) {
	tracks, err := Scan(ctx, cfg.MusicDir, query)
	if os.IsNotExist(err) {
		err = nil
	}
	seen := map[string]bool{}
	for _, t := range tracks {
		seen[t.Path] = true
	}
	for _, path := range cfg.LocalFiles {
		if seen[path] {
			continue
		}
		t, e := LocalTrack(path)
		if e == nil && (query == "" || strings.Contains(strings.ToLower(t.Name+" "+t.Artist), strings.ToLower(query))) {
			tracks = append(tracks, t)
		}
	}
	return tracks, err
}

func searchRemote(ctx context.Context, source, query string, p Provider) (Catalog, error) {
	var result Catalog
	query = strings.TrimSpace(query)
	if query == "" {
		return result, errors.New("请输入歌名、作者或分享链接")
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	var songs []model.Song
	var err error
	if u, e := url.Parse(query); e == nil && (u.Scheme == "https" || u.Scheme == "http") {
		// Bilibili video URLs may name a single video, multipart video or a
		// season. ParsePlaylist handles all three without throwing away the
		// container metadata or turning "playlist link detected" into an error.
		if cp, ok := p.(collectionProvider); ok && source == "bilibili" {
			meta, items, e := cp.ParsePlaylist(query)
			err, songs = e, items
			if meta != nil && (len(items) > 1 || meta.Extra["type"] == "season") {
				result.Container = &Collection{Playlist: *meta, Kind: "playlist"}
			}
		} else {
			song, e := p.Parse(query)
			err = e
			if song != nil {
				songs = []model.Song{*song}
			}
		}
	} else {
		songs, err = p.Search(query)
	}
	if err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	tracks := make([]Track, 0, len(songs))
	for _, song := range songs {
		if song.Source == source {
			tracks = append(tracks, Track{Song: song})
		}
	}
	result.Tracks = tracks
	return result, nil
}

func Resolve(ctx context.Context, t Track, cfg Config) (Resolved, error) {
	t.Extra = maps.Clone(t.Extra)
	r := Resolved{Track: t, Headers: make(http.Header)}
	if t.Source == "local" {
		info, err := os.Stat(t.Path)
		if err != nil || !info.Mode().IsRegular() {
			return r, errors.New("本地文件不存在或不可读")
		}
		r.Input = t.Path
		r.Lyrics, r.Cover = localMetadata(t.Path)
		return r, nil
	}
	p, err := provider(t.Source, cfg)
	if err != nil {
		return r, err
	}
	r.Input, err = p.GetDownloadURL(&r.Track.Song)
	if err != nil {
		return r, err
	}
	u, err := url.Parse(r.Input)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return r, errors.New("来源未返回有效音频地址")
	}
	r.Headers.Set("User-Agent", UserAgent)
	if t.Source == "netease" {
		r.Headers.Set("Referer", "https://music.163.com/")
		r.Headers.Set("Cookie", cfg.NeteaseCookie)
	} else {
		r.Headers.Set("Referer", "https://www.bilibili.com/")
		r.Headers.Set("Cookie", cfg.BilibiliCookie)
	}
	if err := ctx.Err(); err != nil {
		return r, err
	}
	return r, nil
}

// Artwork and lyrics are optional and loaded independently of audio startup.
func Metadata(ctx context.Context, t Track, cfg Config) (string, []byte) {
	t.Extra = maps.Clone(t.Extra)
	if t.Source == "local" {
		return localMetadata(t.Path)
	}
	p, err := provider(t.Source, cfg)
	if err != nil {
		return "", nil
	}
	lyrics, _ := p.GetLyrics(&t.Song)
	if ctx.Err() != nil || t.Cover == "" {
		return lyrics, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.Cover, nil)
	if err != nil {
		return lyrics, nil
	}
	req.Header.Set("User-Agent", UserAgent)
	if t.Source == "bilibili" {
		req.Header.Set("Referer", "https://www.bilibili.com/")
	}
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return lyrics, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return lyrics, nil
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	return lyrics, b
}

func Scan(ctx context.Context, dir, query string) ([]Track, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	query = strings.ToLower(strings.TrimSpace(query))
	var result []Track
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".mp3", ".flac", ".m4a", ".aac", ".ogg", ".wav", ".wma":
		default:
			return nil
		}
		t := Track{Path: path, Song: model.Song{ID: path, Source: "local", Name: strings.TrimSuffix(d.Name(), filepath.Ext(d.Name())), Artist: "未知作者", Ext: strings.TrimPrefix(ext, ".")}}
		if f, e := os.Open(path); e == nil {
			if meta, e := tag.ReadFrom(f); e == nil {
				if meta.Title() != "" {
					t.Name = meta.Title()
				}
				if meta.Artist() != "" {
					t.Artist = meta.Artist()
				}
				t.Album = meta.Album()
			}
			f.Close()
		}
		if query == "" || strings.Contains(strings.ToLower(t.Name+" "+t.Artist+" "+t.Album), query) {
			result = append(result, t)
		}
		return nil
	})
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, err
}

func localMetadata(path string) (string, []byte) {
	var lyrics string
	var cover []byte
	if f, err := os.Open(path); err == nil {
		if m, err := tag.ReadFrom(f); err == nil {
			lyrics = m.Lyrics()
			if pic := m.Picture(); pic != nil {
				cover = pic.Data
			}
		}
		f.Close()
	}
	base := strings.TrimSuffix(path, filepath.Ext(path))
	if b, err := os.ReadFile(base + ".lrc"); err == nil {
		lyrics = string(b)
	}
	if len(cover) == 0 {
		for _, name := range []string{base + ".jpg", base + ".png", filepath.Join(filepath.Dir(path), "cover.jpg")} {
			if info, err := os.Stat(name); err == nil && info.Size() <= 5<<20 {
				if b, err := os.ReadFile(name); err == nil {
					cover = b
					break
				}
			}
		}
	}
	return lyrics, cover
}

func LocalTrack(path string) (Track, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return Track{}, err
	}
	info, err := os.Stat(abs)
	if err != nil || !info.Mode().IsRegular() {
		return Track{}, errors.New("本地文件不可读")
	}
	t := Track{Path: abs, Song: model.Song{ID: abs, Source: "local", Name: strings.TrimSuffix(info.Name(), filepath.Ext(abs)), Artist: "未知作者", Ext: strings.TrimPrefix(strings.ToLower(filepath.Ext(abs)), "."), Size: info.Size()}}
	f, err := os.Open(abs)
	if err != nil {
		return t, err
	}
	defer f.Close()
	if meta, e := tag.ReadFrom(f); e == nil {
		if meta.Title() != "" {
			t.Name = meta.Title()
		}
		if meta.Artist() != "" {
			t.Artist = meta.Artist()
		}
		t.Album = meta.Album()
	}
	return t, nil
}
