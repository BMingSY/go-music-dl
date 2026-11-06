// Package playback bridges FFmpeg's streaming decoder to MyGo's native audio
// output. No browser, HTML audio element, or auxiliary player window is used.
package playback

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const Rate = 48000

type Input struct {
	Location string
	Headers  http.Header
	Duration time.Duration
}

type process struct {
	cmd  *exec.Cmd
	out  io.ReadCloser
	done chan struct{}
	err  error // published by closing done
}

func (p *process) stop() {
	_ = p.out.Close()
	_ = p.cmd.Process.Kill()
	<-p.done
}

type Decoder struct {
	mu      sync.Mutex
	input   Input
	ffmpeg  string
	proc    *process
	closed  bool
	lastErr error
	buf     []byte // used only by MyGo's decoder goroutine
}

func FindFFmpeg(configured string) (string, error) {
	if configured != "" {
		p, err := exec.LookPath(configured)
		if err != nil {
			return "", errors.New("配置的 FFmpeg 不可执行")
		}
		return p, nil
	}
	exe, _ := os.Executable()
	name := "ffmpeg"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	for _, p := range []string{filepath.Join(filepath.Dir(exe), name), filepath.Join(filepath.Dir(exe), "..", "Resources", name), "/opt/homebrew/bin/ffmpeg", "/usr/local/bin/ffmpeg"} {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p, nil
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}
	return "", errors.New("未找到 FFmpeg，请在设置中选择 ffmpeg 可执行文件")
}

func Open(ffmpeg string, input Input) (*Decoder, error) {
	if strings.TrimSpace(input.Location) == "" {
		return nil, errors.New("音频地址为空")
	}
	d := &Decoder{input: input, ffmpeg: ffmpeg}
	if err := d.SetPosition(0); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *Decoder) SampleRate() int { return Rate }
func (d *Decoder) Channels() int   { return 2 }
func (d *Decoder) Len() int64 {
	if d.input.Duration <= 0 {
		return -1
	}
	return int64(d.input.Duration.Seconds() * Rate)
}

func inputArgs(input Input) ([]string, error) {
	var args []string
	if strings.HasPrefix(input.Location, "http://") || strings.HasPrefix(input.Location, "https://") {
		u, err := url.Parse(input.Location)
		if err != nil || u.Host == "" {
			return nil, errors.New("无效音频地址")
		}
		args = append(args, "-rw_timeout", "15000000", "-reconnect", "1", "-reconnect_delay_max", "3")
		var h strings.Builder
		for _, k := range []string{"User-Agent", "Referer", "Cookie"} {
			v := input.Headers.Get(k)
			if strings.ContainsAny(v, "\r\n") {
				return nil, errors.New("请求头不能含有换行符")
			}
			if v != "" {
				h.WriteString(k + ": " + v + "\r\n")
			}
		}
		if h.Len() > 0 {
			args = append(args, "-headers", h.String())
		}
	} else {
		// Treat Windows drive letters as file paths; never accept ffmpeg's
		// concat, pipe, file URL or other protocol expressions from a provider.
		if info, err := os.Stat(input.Location); err != nil || !info.Mode().IsRegular() {
			return nil, errors.New("本地音频文件不可读")
		}
	}
	return args, nil
}

func (d *Decoder) start(frame int64) (*process, error) {
	args := []string{"-nostdin", "-hide_banner", "-loglevel", "error"}
	netArgs, err := inputArgs(d.input)
	if err != nil {
		return nil, err
	}
	args = append(args, netArgs...)
	if frame > 0 {
		args = append(args, "-ss", strconv.FormatFloat(float64(frame)/Rate, 'f', 6, 64))
	}
	args = append(args, "-i", d.input.Location, "-map", "0:a:0", "-vn", "-sn", "-dn", "-ac", "2", "-ar", strconv.Itoa(Rate), "-f", "f32le", "pipe:1")
	cmd := exec.Command(d.ffmpeg, args...)
	quietProcess(cmd)
	// Do not put signed URLs or cookies from ffmpeg's diagnostics in logs/UI.
	cmd.Stderr = io.Discard
	out, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmd.Stdout = writer
	if err = cmd.Start(); err != nil {
		out.Close()
		writer.Close()
		return nil, errors.New("无法启动 FFmpeg 解码器")
	}
	writer.Close()
	p := &process{cmd: cmd, out: out, done: make(chan struct{})}
	go func() { p.err = cmd.Wait(); close(p.done) }()
	return p, nil
}

func (d *Decoder) SetPosition(frame int64) error {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return os.ErrClosed
	}
	old := d.proc
	d.proc = nil
	d.mu.Unlock()
	if old != nil {
		old.stop()
	}
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return os.ErrClosed
	}
	// Starting while holding mu makes Close wait until the new process is
	// registered, so cancellation cannot leave a child running after it returns.
	p, err := d.start(max(0, frame))
	d.proc = p
	d.lastErr = err
	d.mu.Unlock()
	return err
}

func (d *Decoder) Read(samples []float32) (int, error) {
	d.mu.Lock()
	p, closed := d.proc, d.closed
	d.mu.Unlock()
	if closed || p == nil {
		return 0, io.EOF
	}
	want := (len(samples) / 2) * 8
	if cap(d.buf) < want {
		d.buf = make([]byte, want)
	}
	buf := d.buf[:want]
	n, err := io.ReadFull(p.out, buf)
	count := n / 8 * 2
	for i := 0; i < count; i++ {
		samples[i] = math.Float32frombits(binary.LittleEndian.Uint32(buf[i*4:]))
	}
	if err != nil {
		<-p.done
		d.mu.Lock()
		if d.proc == p && !d.closed && p.err != nil {
			d.lastErr = errors.New("音频解码失败，请检查文件、来源权限或网络连接")
		}
		d.mu.Unlock()
		return count, io.EOF
	}
	return count, nil
}

// Seek marks MyGo's old stream generation stale before interrupting its Read.
// Holding mu prevents SetPosition from installing a new process before we have
// detached the old one. Call only while this decoder's player is still open.
func (d *Decoder) Seek(markSeek func()) {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return
	}
	markSeek()
	p := d.proc
	d.proc = nil
	d.mu.Unlock()
	if p != nil {
		p.stop()
	}
}

func (d *Decoder) Close() error {
	d.mu.Lock()
	d.closed = true
	p := d.proc
	d.proc = nil
	d.mu.Unlock()
	if p != nil {
		p.stop()
	}
	return nil
}

func (d *Decoder) Err() error { d.mu.Lock(); defer d.mu.Unlock(); return d.lastErr }

func ProbeDuration(ctx context.Context, ffmpeg string, input Input) time.Duration {
	if input.Duration > 0 {
		return input.Duration
	}
	name := "ffprobe"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	probe := filepath.Join(filepath.Dir(ffmpeg), name)
	if _, err := os.Stat(probe); err != nil {
		p, e := exec.LookPath(name)
		if e != nil {
			return 0
		}
		probe = p
	}
	args, err := inputArgs(input)
	if err != nil {
		return 0
	}
	// ffprobe does not accept ffmpeg's reconnect input options.
	args = removeReconnect(args)
	args = append(args, "-v", "quiet", "-show_entries", "format=duration", "-of", "json", input.Location)
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, probe, args...)
	quietProcess(cmd)
	b, err := cmd.Output()
	if err != nil {
		return 0
	}
	var result struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if json.Unmarshal(b, &result) != nil {
		return 0
	}
	seconds, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil || math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds < 0 {
		return 0
	}
	return time.Duration(seconds * float64(time.Second))
}

func removeReconnect(args []string) []string {
	var out []string
	for i := 0; i < len(args); i += 2 {
		if !strings.HasPrefix(args[i], "-reconnect") {
			out = append(out, args[i:i+2]...)
		}
	}
	return out
}

func Description() string { return fmt.Sprintf("FFmpeg → PCM %d Hz → MyGo audio", Rate) }
