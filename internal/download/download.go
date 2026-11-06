package download

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/BMingSY/go-music-dl/internal/playback"
)

type Progress func(done, total int64)

// Fetch writes to a private temporary file. A canceled or failed transfer never
// replaces a complete download. The source ID keeps equal song titles distinct.
func Fetch(ctx context.Context, t library.Track, cfg library.Config, progress Progress) (string, error) {
	r, err := library.Resolve(ctx, t, cfg)
	if err != nil {
		return "", err
	}
	return save(ctx, r, cfg, progress)
}
func save(ctx context.Context, r library.Resolved, cfg library.Config, progress Progress) (string, error) {
	if cfg.DownloadDir == "" {
		return "", errors.New("请先选择下载目录")
	}
	if err := os.MkdirAll(cfg.DownloadDir, 0755); err != nil {
		return "", err
	}
	ext := strings.ToLower(r.Track.Ext)
	switch ext {
	case "mp3", "flac", "m4a", "aac", "ogg", "wav", "wma":
	default:
		ext = "m4a"
	}
	if r.Track.Source == "bilibili" {
		ext = "m4a"
	}
	key := sha256.Sum256([]byte(r.Track.Key()))
	name := safeName(r.Track.Artist+" - "+r.Track.Name) + " [" + r.Track.Source + "-" + hex.EncodeToString(key[:4]) + "]"
	final := filepath.Join(cfg.DownloadDir, name+"."+ext)
	if info, err := os.Stat(final); err == nil && info.Mode().IsRegular() && info.Size() > 0 {
		return final, nil
	}
	f, err := os.CreateTemp(cfg.DownloadDir, ".musicdl-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())
	var body io.ReadCloser
	var size int64
	if r.Track.Source == "local" {
		body, err = os.Open(r.Input)
		if err == nil {
			info, _ := os.Stat(r.Input)
			size = info.Size()
		}
	} else {
		req, e := http.NewRequestWithContext(ctx, "GET", r.Input, nil)
		if e != nil {
			f.Close()
			return "", e
		}
		req.Header = r.Headers.Clone()
		// Credentials may only follow redirects to the same host.
		client := &http.Client{Timeout: 30 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("下载重定向过多")
			}
			if req.URL.Hostname() != via[0].URL.Hostname() {
				req.Header.Del("Cookie")
			}
			return nil
		}}
		resp, e := client.Do(req)
		if e != nil {
			f.Close()
			return "", errors.New("无法连接音频服务器")
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			f.Close()
			return "", fmt.Errorf("音频服务器返回 %d", resp.StatusCode)
		}
		if strings.Contains(resp.Header.Get("Content-Type"), "text/html") || strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
			resp.Body.Close()
			f.Close()
			return "", errors.New("服务器未返回音频文件")
		}
		body = resp.Body
		size = resp.ContentLength
	}
	if err != nil {
		f.Close()
		return "", err
	}
	defer body.Close()
	var done int64
	buf := make([]byte, 128<<10)
	last := time.Now()
	for {
		if err = ctx.Err(); err != nil {
			break
		}
		n, readErr := body.Read(buf)
		if n > 0 {
			var written int
			written, err = f.Write(buf[:n])
			done += int64(written)
			if err != nil {
				break
			}
		}
		if progress != nil && time.Since(last) > 150*time.Millisecond {
			progress(done, size)
			last = time.Now()
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			err = errors.New("音频下载中断")
			break
		}
	}
	if err == nil && size > 0 && size != done {
		err = errors.New("音频文件下载不完整")
	}
	if err == nil && done == 0 {
		err = errors.New("下载内容为空")
	}
	if err == nil {
		err = f.Sync()
	}
	if e := f.Close(); err == nil {
		err = e
	}
	if err != nil {
		return "", err
	}
	if progress != nil {
		progress(done, max(size, done))
	}
	staged := f.Name()
	// Bilibili delivers fragmented MP4; remux it to a seekable M4A, without
	// transcoding. For other sources this also records title/artist/album tags.
	ffmpeg, e := playback.FindFFmpeg(cfg.FFmpeg)
	if e == nil {
		tagged := f.Name() + "." + ext
		defer os.Remove(tagged)
		e = playback.Remux(ctx, ffmpeg, staged, tagged, r.Track.Name, r.Track.Artist, r.Track.Album)
		if e == nil {
			staged = tagged
		} else if r.Track.Source == "bilibili" {
			return "", e
		}
	} else if r.Track.Source == "bilibili" {
		return "", e
	}
	if err = ctx.Err(); err != nil {
		return "", err
	}
	if err = commit(staged, final); err != nil {
		return "", err
	}
	lyric, cover := library.Metadata(ctx, r.Track, cfg)
	if ctx.Err() == nil {
		if lyric != "" {
			_ = writeSidecar(filepath.Join(cfg.DownloadDir, name+".lrc"), []byte(lyric))
		}
		if len(cover) > 0 {
			ext := ".jpg"
			if len(cover) >= 8 && string(cover[1:4]) == "PNG" {
				ext = ".png"
			}
			_ = writeSidecar(filepath.Join(cfg.DownloadDir, name+ext), cover)
		}
	}
	return final, nil
}
func commit(from, to string) error {
	// A hard link atomically publishes the finished file and never overwrites.
	if err := os.Link(from, to); err == nil {
		return nil
	} else if os.IsExist(err) {
		return errors.New("目标文件已存在")
	}
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(to, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if os.IsExist(err) {
		return errors.New("目标文件已存在")
	}
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if err == nil {
		err = out.Sync()
	}
	if e := out.Close(); err == nil {
		err = e
	}
	if err != nil {
		os.Remove(to)
	}
	return err
}
func writeSidecar(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if os.IsExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if e := f.Close(); err == nil {
		err = e
	}
	if err != nil {
		os.Remove(path)
	}
	return err
}
func safeName(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, s)
	r := []rune(strings.Trim(s, " ."))
	if len(r) > 100 {
		r = r[:100]
	}
	if len(r) == 0 {
		return "音乐"
	}
	return string(r)
}
