package download

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/guohuiyuan/music-lib/model"
)

func TestInterruptedDownloadDoesNotPublishFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100000")
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write([]byte("partial"))
	}))
	defer server.Close()
	dir := t.TempDir()
	_, err := save(context.Background(), library.Resolved{Track: library.Track{Song: model.Song{Source: "test", ID: "1", Name: "title", Ext: "mp3"}}, Input: server.URL}, library.Config{DownloadDir: dir}, nil)
	if err == nil {
		t.Fatal("truncated transfer accepted")
	}
	files, _ := os.ReadDir(dir)
	if len(files) != 0 {
		t.Fatalf("left partial file: %v", files)
	}
}
func TestDownloadCancellationAndHeaders(t *testing.T) {
	arrived := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Referer") != "https://www.bilibili.com/" || r.Header.Get("Cookie") != "fixture=1" {
			t.Error("missing source headers")
		}
		close(arrived)
		w.Header().Set("Content-Type", "audio/mp4")
		w.WriteHeader(200)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer server.Close()
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, e := save(ctx, library.Resolved{Track: library.Track{Song: model.Song{Source: "bilibili", ID: "1", Name: "title"}}, Input: server.URL, Headers: http.Header{"Referer": []string{"https://www.bilibili.com/"}, "Cookie": []string{"fixture=1"}}}, library.Config{DownloadDir: dir}, nil)
		done <- e
	}()
	<-arrived
	cancel()
	select {
	case e := <-done:
		if e == nil {
			t.Fatal("cancellation succeeded")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("download did not cancel")
	}
	files, _ := os.ReadDir(dir)
	if len(files) != 0 {
		t.Fatal("partial file left after cancel")
	}
}
func TestPublishNeverOverwritesAndFilenameIsSafe(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "staged"), filepath.Join(dir, "existing")
	os.WriteFile(a, []byte("new"), 0600)
	os.WriteFile(b, []byte("original"), 0600)
	if err := commit(a, b); err == nil {
		t.Fatal("collision was not reported")
	}
	data, _ := os.ReadFile(b)
	if string(data) != "original" {
		t.Fatal("existing download overwritten")
	}
	for _, name := range []string{`../../escape`, `CON:hello?`, "a\x00b\nc", strings.Repeat("歌", 200)} {
		got := safeName(name)
		if strings.ContainsAny(got, `/\<>:"|?*`) {
			t.Fatal(got)
		}
	}
}
func TestQueueRestoresInterruptedJobsWithoutCookies(t *testing.T) {
	path := filepath.Join(t.TempDir(), "downloads.json")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	q := New(ctx, path)
	q.Add([]library.Track{{Song: model.Song{Source: "netease", ID: "1", Name: "fixture"}}}, library.Config{NeteaseCookie: "secret-fixture"})
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "secret-fixture") {
		t.Fatal("cookie leaked into job history")
	}
	restored := New(ctx, path).Snapshot()
	if len(restored) != 1 || restored[0].State != "queued" {
		t.Fatalf("restored=%v", restored)
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatal("context")
	}
}
