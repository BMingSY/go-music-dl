package playback

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func testFFmpeg(t *testing.T) string {
	t.Helper()
	// FFmpeg honors http_proxy for loopback but not all NO_PROXY variants.
	// Synthetic HTTP fixtures must never go through the developer's proxy.
	for _, key := range []string{"http_proxy", "https_proxy", "all_proxy", "HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY"} {
		t.Setenv(key, "")
	}
	p, err := FindFFmpeg(os.Getenv("MUSIC_DL_TEST_FFMPEG"))
	if err != nil {
		if os.Getenv("MUSIC_DL_REQUIRE_AUDIO_TESTS") == "1" {
			t.Fatal(err)
		}
		t.Skip("FFmpeg is required for codec integration tests")
	}
	return p
}

//go:embed testdata/sine.mp3
var mp3Fixture []byte

func fixture(t *testing.T, ffmpeg, ext string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "音频 fixture."+ext)
	if ext == "mp3" {
		if err := os.WriteFile(path, mp3Fixture, 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, ffmpeg, "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000:duration=4", "-y", path)
	quietProcess(cmd)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture: %s %v", b, err)
	}
	return path
}

func TestDecodeAndSeekWAVFLACAndAAC(t *testing.T) {
	ffmpeg := testFFmpeg(t)
	for _, ext := range []string{"wav", "flac", "m4a"} {
		t.Run(ext, func(t *testing.T) {
			path := fixture(t, ffmpeg, ext)
			d, err := Open(ffmpeg, Input{Location: path, Duration: 4 * time.Second})
			if err != nil {
				t.Fatal(err)
			}
			defer d.Close()
			buf := make([]float32, 4096)
			n, err := d.Read(buf)
			if err != nil || n != len(buf) {
				t.Fatalf("read n=%d err=%v", n, err)
			}
			var energy float64
			for _, v := range buf {
				energy += math.Abs(float64(v))
			}
			if energy < 1 {
				t.Fatal("decoded silence")
			}
			if err := d.SetPosition(2 * Rate); err != nil {
				t.Fatal(err)
			}
			total := 0
			for {
				n, err = d.Read(buf)
				total += n
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			// AAC can include encoder padding, but seek must discard the first half.
			if total < Rate*2*18/10 || total > Rate*2*22/10 {
				t.Fatalf("post-seek samples=%d", total)
			}
			if d.Err() != nil {
				t.Fatal(d.Err())
			}
		})
	}
}

func TestHTTPPlaybackSendsHeadersAndSeeks(t *testing.T) {
	ffmpeg := testFFmpeg(t)
	path := fixture(t, ffmpeg, "flac")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Referer") != "https://www.bilibili.com/" || r.Header.Get("Cookie") != "fixture=1" {
			http.Error(w, "headers", http.StatusForbidden)
			return
		}
		http.ServeFile(w, r, path)
	}))
	defer server.Close()
	d, err := Open(ffmpeg, Input{Location: server.URL + "/audio.flac", Duration: 4 * time.Second, Headers: http.Header{"Referer": {"https://www.bilibili.com/"}, "Cookie": {"fixture=1"}}})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := d.SetPosition(Rate); err != nil {
		t.Fatal(err)
	}
	if n, err := d.Read(make([]float32, 2048)); n == 0 || err != nil {
		t.Fatalf("n=%d err=%v decoder=%v", n, err, d.Err())
	}
}

func TestTruncatedHTTPAudioIsNotSuccessfulEnd(t *testing.T) {
	ffmpeg := testFFmpeg(t)
	data, err := os.ReadFile(fixture(t, ffmpeg, "wav"))
	if err != nil {
		t.Fatal(err)
	}
	var complete atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if complete.Load() {
			http.ServeContent(w, r, "audio.wav", time.Time{}, bytes.NewReader(data))
			return
		}
		// The HTTP response completes normally, but its WAV header still
		// describes four seconds. FFmpeg can exit successfully after one.
		w.Header().Set("Content-Type", "audio/wav")
		w.Header().Set("Content-Length", fmt.Sprint(len(data)/4))
		_, _ = w.Write(data[:len(data)/4])
	}))
	defer server.Close()
	d, err := Open(ffmpeg, Input{Location: server.URL + "/audio.wav", Duration: 4 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	buf := make([]float32, 4096)
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
	if total == 0 || total >= 2*Rate*2 {
		t.Fatalf("fixture did not end early: %d samples", total)
	}
	if d.Err() == nil {
		t.Fatal("truncated audio was reported as a successful end; the playlist would skip to the next song")
	}
	complete.Store(true)
	if err := d.SetPosition(2 * Rate); err != nil {
		t.Fatal(err)
	}
	total = 0
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
	if d.Err() != nil || total < 2*Rate*19/10 || total > 2*Rate*21/10 {
		t.Fatalf("reopened stream did not finish normally after seeking: samples=%d err=%v", total, d.Err())
	}
}

func TestCloseInterruptsNetworkRead(t *testing.T) {
	ffmpeg := testFFmpeg(t)
	entered := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case entered <- struct{}{}:
		default:
		}
		<-r.Context().Done()
	}))
	defer server.Close()
	d, err := Open(ffmpeg, Input{Location: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { _, _ = d.Read(make([]float32, 2048)); close(done) }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		d.Close()
		t.Fatal("request never started")
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("close left decoder blocked")
	}
}

func TestRejectHeaderInjection(t *testing.T) {
	_, err := inputArgs(Input{Location: "https://example.com/audio", Headers: http.Header{"Cookie": {"ok=1\r\nInjected: yes"}}})
	if err == nil {
		t.Fatal("newline header accepted")
	}
}

func TestLocalPathMayContainPercent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "100% music.wav")
	if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := inputArgs(Input{Location: path}); err != nil {
		t.Fatal(err)
	}
}
