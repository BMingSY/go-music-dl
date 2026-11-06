package download

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/BMingSY/go-music-dl/internal/library"
)

type Job struct {
	Track       library.Track `json:"track"`
	State       string        `json:"state"`
	Path        string        `json:"path,omitempty"`
	Error       string        `json:"error,omitempty"`
	Done, Total int64
	Config      library.Config `json:"-"`
}
type Queue struct {
	mu      sync.Mutex
	jobs    []Job
	wake    chan struct{}
	cancel  context.CancelFunc
	current string
	path    string
	ctx     context.Context
}

func New(ctx context.Context, path string) *Queue {
	q := &Queue{ctx: ctx, path: path, wake: make(chan struct{}, 1)}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &q.jobs)
	}
	for i := range q.jobs {
		if q.jobs[i].State == "queued" || q.jobs[i].State == "downloading" {
			q.jobs[i].State = "canceled"
			q.jobs[i].Error = "上次下载已中断，可点击重试"
		}
	}
	go q.run()
	return q
}
func (q *Queue) Snapshot() []Job {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]Job(nil), q.jobs...)
}
func (q *Queue) Add(tracks []library.Track, cfg library.Config) {
	q.mu.Lock()
	for _, t := range tracks {
		found := false
		for i := range q.jobs {
			j := &q.jobs[i]
			if j.Track.Key() == t.Key() {
				found = true
				if q.current != t.Key() && (j.State == "failed" || j.State == "canceled") {
					*j = Job{Track: t, State: "queued", Config: cfg}
				}
				break
			}
		}
		if !found {
			q.jobs = append(q.jobs, Job{Track: t, State: "queued", Config: cfg})
		}
	}
	q.persist()
	q.mu.Unlock()
	select {
	case q.wake <- struct{}{}:
	default:
	}
}
func (q *Queue) Cancel(key string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for i := range q.jobs {
		j := &q.jobs[i]
		if j.Track.Key() == key && (j.State == "queued" || j.State == "downloading") {
			j.State = "canceled"
			j.Error = ""
			if q.current == key && q.cancel != nil {
				q.cancel()
			}
		}
	}
	q.persist()
}
func (q *Queue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	kept := q.jobs[:0]
	for _, j := range q.jobs {
		if j.State == "queued" || j.State == "downloading" {
			kept = append(kept, j)
		}
	}
	q.jobs = kept
	q.persist()
}
func (q *Queue) persist() {
	if q.path == "" {
		return
	}
	if os.MkdirAll(filepath.Dir(q.path), 0700) != nil {
		return
	}
	b, err := json.Marshal(q.jobs)
	if err != nil {
		return
	}
	f, err := os.CreateTemp(filepath.Dir(q.path), ".downloads-*")
	if err != nil {
		return
	}
	defer os.Remove(f.Name())
	_, err = f.Write(b)
	if e := f.Close(); err == nil {
		err = e
	}
	if err == nil {
		_ = os.Rename(f.Name(), q.path)
	}
}
func (q *Queue) run() {
	for {
		select {
		case <-q.ctx.Done():
			return
		case <-q.wake:
		}
		for {
			q.mu.Lock()
			idx := -1
			for i := range q.jobs {
				if q.jobs[i].State == "queued" {
					idx = i
					break
				}
			}
			if idx < 0 || q.ctx.Err() != nil {
				q.mu.Unlock()
				break
			}
			job := q.jobs[idx]
			key := job.Track.Key()
			q.jobs[idx].State = "downloading"
			ctx, cancel := context.WithCancel(q.ctx)
			q.cancel = cancel
			q.current = key
			q.mu.Unlock()
			path, err := Fetch(ctx, job.Track, job.Config, func(done, total int64) {
				q.mu.Lock()
				defer q.mu.Unlock()
				for i := range q.jobs {
					if q.jobs[i].Track.Key() == key {
						q.jobs[i].Done = done
						q.jobs[i].Total = total
						break
					}
				}
			})
			canceled := ctx.Err() != nil
			cancel()
			q.mu.Lock()
			q.cancel = nil
			q.current = ""
			for i := range q.jobs {
				j := &q.jobs[i]
				if j.Track.Key() != key {
					continue
				}
				if canceled {
					j.State = "canceled"
				} else if err != nil {
					j.State = "failed"
					j.Error = err.Error()
				} else {
					j.State = "done"
					j.Path = path
					j.Error = ""
				}
				j.Config = library.Config{}
				break
			}
			q.persist()
			q.mu.Unlock()
		}
	}
}
