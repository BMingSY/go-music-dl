package playback

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	streamFirstChunk = 32 * 1024
	streamChunkSize  = 256 * 1024
	streamAhead      = 16
	streamWorkers    = 4
	streamCacheLimit = 32 // at most 8 MiB of compressed audio, plus active reads
)

type streamChunk struct {
	done  chan struct{}
	data  []byte
	err   error
	ready bool   // protected by streamSource.mu
	used  uint64 // protected by streamSource.mu
}

type streamJob struct {
	index int64
	chunk *streamChunk
}

// streamSource gives FFmpeg a seekable loopback URL backed by bounded,
// parallel read-ahead. It lives for one decoder process, so seeking or stopping
// cancels even requests which are still waiting for the remote server.
type streamSource struct {
	input     Input
	url, path string
	ctx       context.Context
	cancel    context.CancelFunc
	server    *http.Server
	client    *http.Client
	transport *http.Transport
	init      sync.Once
	initErr   error
	ranged    bool
	total     int64
	typeName  string

	mu      sync.Mutex
	closed  bool
	clock   uint64
	chunks  map[int64]*streamChunk
	pending []streamJob
	work    *sync.Cond
	jobs    sync.WaitGroup
	close   sync.Once
}

func newStreamSource(input Input) (*streamSource, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, errors.New("无法启动网络音频缓冲")
	}
	ctx, cancel := context.WithCancel(context.Background())
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DisableCompression = true
	transport.MaxIdleConnsPerHost = streamWorkers
	transport.ResponseHeaderTimeout = 10 * time.Second
	s := &streamSource{
		input: input, path: "/" + rand.Text(), ctx: ctx, cancel: cancel,
		transport: transport, client: &http.Client{Transport: transport},
		chunks: make(map[int64]*streamChunk),
	}
	s.work = sync.NewCond(&s.mu)
	s.jobs.Add(streamWorkers)
	for range streamWorkers {
		go s.worker()
	}
	s.input.Headers = input.Headers.Clone()
	s.url = "http://" + listener.Addr().String() + s.path
	s.server = &http.Server{Handler: s, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = s.server.Serve(listener) }()
	return s, nil
}

func (s *streamSource) Close() {
	s.close.Do(func() {
		s.mu.Lock()
		s.closed = true
		for _, job := range s.pending {
			job.chunk.err = context.Canceled
			job.chunk.ready = true
			close(job.chunk.done)
		}
		s.pending = nil
		s.work.Broadcast()
		s.mu.Unlock()
		s.cancel()
		_ = s.server.Close()
		s.jobs.Wait()
		s.transport.CloseIdleConnections()
	})
}

func (s *streamSource) request(ctx context.Context, byteRange string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.input.Location, nil)
	if err != nil {
		return nil, errors.New("无效音频地址")
	}
	for _, key := range []string{"User-Agent", "Referer", "Cookie"} {
		if value := s.input.Headers.Get(key); value != "" {
			req.Header.Set(key, value)
		}
	}
	req.Header.Set("Accept-Encoding", "identity")
	if byteRange != "" {
		req.Header.Set("Range", byteRange)
	}
	return s.client.Do(req)
}

// fetch validates offsets as well as length: a server ignoring Range must
// never silently substitute the beginning of the song for a later chunk.
func (s *streamSource) fetch(start, end, total int64) ([]byte, string, int64, bool, error) {
	for attempt := 0; attempt < 3; attempt++ {
		if s.ctx.Err() != nil {
			return nil, "", 0, false, s.ctx.Err()
		}
		ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
		resp, err := s.request(ctx, fmt.Sprintf("bytes=%d-%d", start, end))
		if err != nil {
			cancel()
			continue
		}
		if resp.StatusCode == http.StatusOK && total == 0 {
			resp.Body.Close()
			cancel()
			return nil, "", 0, false, nil
		}
		var first, last, size int64
		_, scanErr := fmt.Sscanf(resp.Header.Get("Content-Range"), "bytes %d-%d/%d", &first, &last, &size)
		valid := resp.StatusCode == http.StatusPartialContent && scanErr == nil && size > 0 &&
			first == start && last == min(end, size-1) && (total == 0 || size == total)
		var data []byte
		if valid {
			data, err = io.ReadAll(io.LimitReader(resp.Body, end-start+2))
		}
		resp.Body.Close()
		cancel()
		if valid && err == nil && int64(len(data)) == last-first+1 {
			return data, resp.Header.Get("Content-Type"), size, true, nil
		}
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
			break
		}
	}
	// Never return URLs, cookies or upstream diagnostics to the UI.
	return nil, "", 0, false, errors.New("网络音频分块读取失败")
}

func (s *streamSource) probe() {
	data, contentType, total, ranged, err := s.fetch(0, streamFirstChunk-1, 0)
	s.typeName, s.total, s.ranged, s.initErr = contentType, total, ranged, err
	if err == nil && ranged {
		chunk := &streamChunk{done: make(chan struct{}), data: data, ready: true}
		close(chunk.done)
		s.mu.Lock()
		s.chunks[0] = chunk
		s.mu.Unlock()
	}
}

func streamChunkIndex(offset int64) int64 {
	if offset < streamFirstChunk {
		return 0
	}
	return 1 + (offset-streamFirstChunk)/streamChunkSize
}

func streamChunkStart(index int64) int64 {
	if index == 0 {
		return 0
	}
	return streamFirstChunk + (index-1)*streamChunkSize
}

func (s *streamSource) chunk(index int64, required bool) *streamChunk {
	for {
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			return nil
		}
		s.clock++
		if chunk := s.chunks[index]; chunk != nil {
			chunk.used = s.clock
			s.mu.Unlock()
			return chunk
		}
		if len(s.chunks) >= streamCacheLimit {
			oldest := int64(-1)
			var wait <-chan struct{}
			for key, chunk := range s.chunks {
				wait = chunk.done
				if chunk.ready && (oldest < 0 || chunk.used < s.chunks[oldest].used) {
					oldest = key
				}
			}
			if oldest < 0 {
				s.mu.Unlock()
				if !required {
					return nil
				}
				select {
				case <-wait:
					continue
				case <-s.ctx.Done():
					return nil
				}
			}
			delete(s.chunks, oldest)
		}
		chunk := &streamChunk{done: make(chan struct{}), used: s.clock}
		s.chunks[index] = chunk
		job := streamJob{index: index, chunk: chunk}
		if required {
			// A seek or container-metadata read takes priority over speculative
			// chunks, while prefetch itself stays in playback order.
			s.pending = append([]streamJob{job}, s.pending...)
		} else {
			s.pending = append(s.pending, job)
		}
		s.work.Signal()
		s.mu.Unlock()
		return chunk
	}
}

func (s *streamSource) worker() {
	defer s.jobs.Done()
	for {
		s.mu.Lock()
		for !s.closed && len(s.pending) == 0 {
			s.work.Wait()
		}
		if s.closed {
			s.mu.Unlock()
			return
		}
		job := s.pending[0]
		s.pending[0] = streamJob{}
		s.pending = s.pending[1:]
		s.mu.Unlock()
		s.loadChunk(job.index, job.chunk)
	}
}

func (s *streamSource) loadChunk(index int64, chunk *streamChunk) {
	start := streamChunkStart(index)
	end := min(start+streamChunkSize-1, s.total-1)
	if index == 0 {
		end = min(int64(streamFirstChunk-1), s.total-1)
	}
	chunk.data, _, _, _, chunk.err = s.fetch(start, end, s.total)
	s.mu.Lock()
	chunk.ready = true
	close(chunk.done)
	s.mu.Unlock()
}

func (s *streamSource) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != s.path || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
		http.NotFound(w, r)
		return
	}
	s.init.Do(s.probe)
	if s.initErr != nil {
		http.Error(w, "audio unavailable", http.StatusBadGateway)
		return
	}
	if !s.ranged {
		s.forward(w, r)
		return
	}
	start, end, ok := streamRange(r.Header.Get("Range"), s.total)
	if !ok {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", s.total))
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return
	}
	w.Header().Set("Content-Type", s.typeName)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
	if r.Header.Get("Range") != "" {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, s.total))
		w.WriteHeader(http.StatusPartialContent)
	}
	if r.Method == http.MethodHead {
		return
	}
	for offset := start; offset <= end; {
		index := streamChunkIndex(offset)
		chunk := s.chunk(index, true)
		if chunk == nil {
			panic(http.ErrAbortHandler)
		}
		// Start nearby chunks without waiting for the decoder to consume this
		// one. A fixed window also bounds pending work for very long videos.
		for next := index + 1; next <= index+streamAhead && streamChunkStart(next) < s.total; next++ {
			s.chunk(next, false)
		}
		select {
		case <-chunk.done:
		case <-r.Context().Done():
			return
		case <-s.ctx.Done():
			return
		}
		if chunk.err != nil {
			// Closing an incomplete response preserves the decoder's existing
			// early-end detection, rather than treating a failed chunk as EOF.
			panic(http.ErrAbortHandler)
		}
		from := offset - streamChunkStart(index)
		data := chunk.data[from:min(int64(len(chunk.data)), from+end-offset+1)]
		if _, err := w.Write(data); err != nil {
			return
		}
		if flush, ok := w.(http.Flusher); ok {
			flush.Flush()
		}
		offset += int64(len(data))
	}
}

// A non-Range origin retains its ordinary streaming behavior.
func (s *streamSource) forward(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	stop := context.AfterFunc(s.ctx, cancel)
	defer stop()
	resp, err := s.request(ctx, r.Header.Get("Range"))
	if err != nil {
		http.Error(w, "audio unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for _, key := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges"} {
		if value := resp.Header.Get(key); value != "" {
			w.Header().Set(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	if r.Method != http.MethodHead {
		if _, err := io.Copy(w, resp.Body); err != nil {
			panic(http.ErrAbortHandler)
		}
	}
}

func streamRange(value string, total int64) (int64, int64, bool) {
	if value == "" {
		return 0, total - 1, true
	}
	if !strings.HasPrefix(value, "bytes=") {
		return 0, 0, false
	}
	left, right, ok := strings.Cut(strings.TrimPrefix(value, "bytes="), "-")
	if !ok {
		return 0, 0, false
	}
	if left == "" {
		suffix, err := strconv.ParseInt(right, 10, 64)
		return max(0, total-suffix), total - 1, err == nil && suffix > 0
	}
	start, err := strconv.ParseInt(left, 10, 64)
	if err != nil || start < 0 || start >= total {
		return 0, 0, false
	}
	end := total - 1
	if right != "" {
		end, err = strconv.ParseInt(right, 10, 64)
	}
	return start, min(end, total-1), err == nil && end >= start
}
