package desktop

import (
	"io"
	"net/http"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo/ui"
)

func (s *app) artwork(t library.Track) *ui.Bitmap {
	key := t.Cover
	if t.Source == "local" {
		key = t.Key()
	}
	if key == "" {
		return nil
	}
	if img, ok := s.images[key]; ok {
		return img
	}
	// Headless renders use supplied fixtures and never start network requests.
	if s.win == nil {
		return nil
	}
	if s.imagePending[key] {
		return nil
	}
	s.imagePending[key] = true
	cfg := s.cfg
	go func() {
		select {
		case s.imageSlots <- struct{}{}:
		case <-s.ctx.Done():
			return
		}
		defer func() { <-s.imageSlots }()
		var data []byte
		if t.Source == "local" {
			_, data = library.Metadata(s.ctx, t, cfg)
		} else {
			req, e := http.NewRequestWithContext(s.ctx, "GET", key, nil)
			if e == nil {
				req.Header.Set("User-Agent", library.UserAgent)
				if t.Source == "bilibili" {
					req.Header.Set("Referer", "https://www.bilibili.com/")
				}
				resp, e := (&http.Client{Timeout: 8 * time.Second}).Do(req)
				if e == nil {
					if resp.StatusCode == 200 {
						data, _ = io.ReadAll(io.LimitReader(resp.Body, 5<<20))
					}
					resp.Body.Close()
				}
			}
		}
		s.update(func() { img, _ := ui.DecodeBitmap(data); s.images[key] = img; delete(s.imagePending, key) })
	}()
	return nil
}
