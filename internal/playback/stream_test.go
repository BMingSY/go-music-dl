package playback

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func testStreamSource(t *testing.T, input Input) *streamSource {
	t.Helper()
	s, err := newStreamSource(input)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}

func readStreamRange(t *testing.T, s *streamSource, value string) (*http.Response, []byte, error) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, s.url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Range", value)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return resp, data, err
}

func waitStreamChunk(t *testing.T, s *streamSource, index int64) {
	t.Helper()
	s.mu.Lock()
	chunk := s.chunks[index]
	s.mu.Unlock()
	if chunk == nil {
		t.Fatalf("chunk %d was not prefetched", index)
	}
	select {
	case <-chunk.done:
		if chunk.err != nil {
			t.Fatal(chunk.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("chunk %d did not finish prefetching", index)
	}
}

func TestStreamPrefetchServesCachedAudioDuringOriginStall(t *testing.T) {
	data := make([]byte, streamChunkSize*64)
	for i := range data {
		data[i] = byte(i*31 + i/257)
	}
	var stalled atomic.Bool
	var active, peak atomic.Int32
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := active.Add(1)
		defer active.Add(-1)
		for old := peak.Load(); n > old; old = peak.Load() {
			if peak.CompareAndSwap(old, n) {
				break
			}
		}
		if r.Header.Get("Cookie") != "fixture=1" || r.Header.Get("Referer") != "https://www.bilibili.com/" || r.Header.Get("User-Agent") != "musicdl-test" {
			http.Error(w, "headers", http.StatusForbidden)
			return
		}
		if stalled.Load() {
			<-r.Context().Done()
			return
		}
		http.ServeContent(w, r, "audio.m4a", time.Time{}, bytes.NewReader(data))
	}))
	defer origin.Close()
	s := testStreamSource(t, Input{Location: origin.URL, Headers: http.Header{
		"Cookie": {"fixture=1"}, "Referer": {"https://www.bilibili.com/"}, "User-Agent": {"musicdl-test"},
	}})
	_, got, err := readStreamRange(t, s, "bytes=0-1023")
	if err != nil || !bytes.Equal(got, data[:1024]) {
		t.Fatalf("first read: len=%d err=%v", len(got), err)
	}
	// The client has stopped reading after 1 KiB. Later chunks must still
	// arrive, independently of playback's small PCM buffer.
	for index := int64(0); index <= streamAhead; index++ {
		waitStreamChunk(t, s, index)
	}
	stalled.Store(true)
	start := streamChunkStart(4) + 37
	end := streamChunkStart(6) + 91
	resp, got, err := readStreamRange(t, s, fmt.Sprintf("bytes=%d-%d", start, end))
	if err != nil || resp.StatusCode != http.StatusPartialContent || !bytes.Equal(got, data[start:end+1]) {
		t.Fatalf("cached range during stall: status=%d len=%d err=%v", resp.StatusCode, len(got), err)
	}
	if peak.Load() > streamWorkers {
		t.Fatalf("too many upstream requests: %d", peak.Load())
	}
	closed := make(chan struct{})
	go func() { s.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("closing left stalled prefetch requests running")
	}
}

func TestStreamRetriesTruncatedChunksAndKeepsOrder(t *testing.T) {
	data := bytes.Repeat([]byte("audio range fixture"), streamChunkSize/3)
	var attempts atomic.Int32
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start, end, ok := streamRange(r.Header.Get("Range"), int64(len(data)))
		if !ok {
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(data)))
		w.Header().Set("Content-Length", fmt.Sprint(end-start+1))
		w.WriteHeader(http.StatusPartialContent)
		if start == streamFirstChunk && attempts.Add(1) < 3 {
			_, _ = w.Write(data[start : start+17])
			return
		}
		_, _ = w.Write(data[start : end+1])
	}))
	defer origin.Close()
	s := testStreamSource(t, Input{Location: origin.URL})
	_, got, err := readStreamRange(t, s, "")
	if err != nil || !bytes.Equal(got, data) || attempts.Load() != 3 {
		t.Fatalf("reassembled audio: len=%d attempts=%d err=%v", len(got), attempts.Load(), err)
	}
}

func TestStreamRejectsWrongRangeInsteadOfPlayingWrongAudio(t *testing.T) {
	data := bytes.Repeat([]byte("audio"), streamChunkSize)
	var attempts atomic.Int32
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start, end, _ := streamRange(r.Header.Get("Range"), int64(len(data)))
		if start > 0 {
			attempts.Add(1)
			w.Header().Set("Content-Range", fmt.Sprintf("bytes 0-%d/%d", end-start, len(data)))
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write(data[:end-start+1])
			return
		}
		http.ServeContent(w, r, "audio", time.Time{}, bytes.NewReader(data))
	}))
	defer origin.Close()
	s := testStreamSource(t, Input{Location: origin.URL})
	_, got, err := readStreamRange(t, s, "bytes=0-294911")
	if err == nil || len(got) != streamFirstChunk || attempts.Load() < 3 {
		t.Fatalf("invalid range was accepted: len=%d attempts=%d err=%v", len(got), attempts.Load(), err)
	}
}

func TestStreamCacheRemainsBoundedForLongAudio(t *testing.T) {
	data := bytes.Repeat([]byte("long audio"), streamChunkSize*10)
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "audio", time.Time{}, bytes.NewReader(data))
	}))
	defer origin.Close()
	s := testStreamSource(t, Input{Location: origin.URL})
	_, got, err := readStreamRange(t, s, "")
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("long audio: len=%d err=%v", len(got), err)
	}
	s.mu.Lock()
	count := len(s.chunks)
	s.mu.Unlock()
	if count > streamCacheLimit {
		t.Fatalf("cache grew to %d chunks", count)
	}
	// A seek back into an evicted range must still return the right bytes.
	_, got, err = readStreamRange(t, s, "bytes=123-456")
	if err != nil || !bytes.Equal(got, data[123:457]) {
		t.Fatalf("backward seek: len=%d err=%v", len(got), err)
	}
}

func TestStreamFallsBackForOriginWithoutRanges(t *testing.T) {
	data := bytes.Repeat([]byte("ordinary stream"), 4096)
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write(data)
	}))
	defer origin.Close()
	s := testStreamSource(t, Input{Location: origin.URL})
	resp, got, err := readStreamRange(t, s, "bytes=100-")
	if err != nil || resp.StatusCode != http.StatusOK || !bytes.Equal(got, data) {
		t.Fatalf("fallback: status=%d len=%d err=%v", resp.StatusCode, len(got), err)
	}
}

func TestHTTPDecoderFinishesFromPrefetchAfterOriginFails(t *testing.T) {
	ffmpeg := testFFmpeg(t)
	data, err := os.ReadFile(fixture(t, ffmpeg, "wav"))
	if err != nil {
		t.Fatal(err)
	}
	var offline atomic.Bool
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if offline.Load() {
			http.Error(w, "offline", http.StatusServiceUnavailable)
			return
		}
		http.ServeContent(w, r, "audio.wav", time.Time{}, bytes.NewReader(data))
	}))
	defer origin.Close()
	d, err := Open(ffmpeg, Input{Location: origin.URL, Duration: 4 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	buf := make([]float32, 4096)
	total, err := d.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	d.mu.Lock()
	source := d.proc.source
	d.mu.Unlock()
	for index := int64(0); index <= streamChunkIndex(int64(len(data)-1)); index++ {
		waitStreamChunk(t, source, index)
	}
	offline.Store(true)
	for {
		n, err := d.Read(buf)
		total += n
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if d.Err() != nil || total != 4*Rate*2 {
		t.Fatalf("cached playback stopped early: samples=%d err=%v", total, d.Err())
	}
}

func TestHTTPSeekReplacesPrefetchAndDecodesAACAtNewPosition(t *testing.T) {
	ffmpeg := testFFmpeg(t)
	path := fixture(t, ffmpeg, "m4a") // AAC with container metadata at the end
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path)
	}))
	defer origin.Close()
	d, err := Open(ffmpeg, Input{Location: origin.URL, Duration: 4 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	buf := make([]float32, 4096)
	if _, err := d.Read(buf); err != nil {
		t.Fatal(err)
	}
	d.mu.Lock()
	old := d.proc.source
	d.mu.Unlock()
	marked := false
	d.Seek(func() { marked = true })
	if !marked || old.ctx.Err() == nil {
		t.Fatal("seek did not invalidate and cancel the old stream")
	}
	if err := d.SetPosition(2 * Rate); err != nil {
		t.Fatal(err)
	}
	total := 0
	for {
		n, err := d.Read(buf)
		total += n
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if d.Err() != nil || total < Rate*2*18/10 || total > Rate*2*22/10 {
		t.Fatalf("AAC seek: samples=%d err=%v", total, d.Err())
	}
}

func TestStreamRangeRequests(t *testing.T) {
	for _, tt := range []struct {
		header     string
		start, end int64
		ok         bool
	}{
		{"", 0, 99, true}, {"bytes=0-", 0, 99, true}, {"bytes=17-42", 17, 42, true},
		{"bytes=-12", 88, 99, true}, {"bytes=90-999", 90, 99, true}, {"bytes=-200", 0, 99, true},
		{"bytes=100-", 0, 0, false}, {"bytes=9-1", 0, 0, false}, {"bytes=1-2,4-5", 0, 0, false},
		{"bytes=-0", 0, 0, false}, {"bytes=", 0, 0, false}, {"garbage", 0, 0, false},
	} {
		t.Run(tt.header, func(t *testing.T) {
			start, end, ok := streamRange(tt.header, 100)
			if ok != tt.ok || ok && (start != tt.start || end != tt.end) {
				t.Fatalf("range %q = %d-%d, %v", tt.header, start, end, ok)
			}
		})
	}
}
