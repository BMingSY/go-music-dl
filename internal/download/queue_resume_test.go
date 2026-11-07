package download

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/guohuiyuan/music-lib/model"
)

func TestResumeRestoresPendingDirectoryWithoutPersistingCookies(t *testing.T) {
	path := filepath.Join(t.TempDir(), "downloads.json")
	jobs := []Job{{Track: library.Track{Song: model.Song{ID: "pending", Source: "netease"}}, State: "downloading", Directory: "chosen-folder"}, {Track: library.Track{Song: model.Song{ID: "canceled", Source: "netease"}}, State: "canceled"}}
	data, _ := json.Marshal(jobs)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	queue := New(ctx, path)
	queue.Resume(library.Config{NeteaseCookie: "private-test-cookie", DownloadDir: "default-folder"})
	restored := queue.Snapshot()
	if restored[0].State != "queued" || restored[0].Config.DownloadDir != "chosen-folder" || restored[0].Config.NeteaseCookie != "private-test-cookie" {
		t.Fatal("pending work did not recover its runtime configuration")
	}
	if restored[1].State != "canceled" {
		t.Fatal("explicitly canceled work was resumed")
	}
	persisted, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(persisted, []byte("private-test-cookie")) {
		t.Fatal("cookie persisted with download tasks")
	}
}
func TestClearQueuePreservesActiveWriteAndCompletedHistory(t *testing.T) {
	q := &Queue{jobs: []Job{{State: "queued"}, {State: "downloading"}, {State: "done", Path: "saved.mp3"}}}
	q.ClearPending()
	jobs := q.Snapshot()
	if len(jobs) != 3 || jobs[0].State != "canceled" || jobs[1].State != "downloading" || jobs[2].Path != "saved.mp3" {
		t.Fatal("clearing the queue changed the active write or download history")
	}
	for _, job := range jobs {
		if !job.Hidden {
			t.Fatal("cleared task still visible in queue")
		}
	}
}
