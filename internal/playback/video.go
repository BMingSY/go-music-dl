package playback

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/BMingSY/go-music-dl/internal/lyrics"
)

func ExportVideo(ctx context.Context, ffmpeg, audioPath, output, title, artist, lrc string, cover []byte) error {
	dir, err := os.MkdirTemp("", "musicdl-video-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	duration := ProbeDuration(ctx, ffmpeg, Input{Location: audioPath})
	if duration <= 0 {
		return fmt.Errorf("无法读取音频时长")
	}
	if err = os.WriteFile(filepath.Join(dir, "lyrics.ass"), []byte(videoSubtitles(title, artist, lrc, duration)), 0600); err != nil {
		return err
	}
	args := []string{"-nostdin", "-hide_banner", "-loglevel", "error", "-y"}
	filter := "scale=1280:720:force_original_aspect_ratio=increase,crop=1280:720,drawbox=color=black@0.55:t=fill,subtitles=lyrics.ass"
	if len(cover) > 0 {
		if err = os.WriteFile(filepath.Join(dir, "cover.img"), cover, 0600); err != nil {
			return err
		}
		args = append(args, "-loop", "1", "-framerate", "24", "-i", "cover.img")
	} else {
		args = append(args, "-f", "lavfi", "-i", "color=c=0x18252c:s=1280x720:r=24")
	}
	staged, err := os.CreateTemp(filepath.Dir(output), ".musicdl-video-*.mp4")
	if err != nil {
		return err
	}
	name := staged.Name()
	staged.Close()
	defer os.Remove(name)
	args = append(args, "-i", audioPath, "-map", "0:v:0", "-map", "1:a:0", "-vf", filter, "-c:v", "libx264", "-preset", "veryfast", "-crf", "23", "-pix_fmt", "yuv420p", "-c:a", "aac", "-b:a", "192k", "-t", fmt.Sprintf("%.3f", duration.Seconds()), "-shortest", "-movflags", "+faststart", name)
	cmd := exec.CommandContext(ctx, ffmpeg, args...)
	cmd.Dir = dir
	quietProcess(cmd)
	if err = cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("FFmpeg 视频编码失败")
	}
	return os.Rename(name, output)
}
func assText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "{", "\\{")
	s = strings.ReplaceAll(s, "}", "\\}")
	s = strings.ReplaceAll(s, "\r", "")
	return strings.ReplaceAll(s, "\n", "\\N")
}
func assTime(t time.Duration) string {
	n := max(0, int(t/(10*time.Millisecond)))
	return fmt.Sprintf("%d:%02d:%02d.%02d", n/360000, (n/6000)%60, (n/100)%60, n%100)
}
func videoSubtitles(title, artist, lrc string, duration time.Duration) string {
	var b strings.Builder
	b.WriteString("[Script Info]\nScriptType: v4.00+\nPlayResX: 1280\nPlayResY: 720\nWrapStyle: 0\n\n[V4+ Styles]\nFormat: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\nStyle: Title,Arial,44,&H00FFFFFF,&H00FFFFFF,&H70000000,&H70000000,-1,0,0,0,100,100,0,0,1,1,0,8,60,60,90,1\nStyle: Lyric,Arial,36,&H00FFFFFF,&H00FFFFFF,&H70000000,&H70000000,-1,0,0,0,100,100,0,0,1,1,0,5,80,80,60,1\n\n[Events]\nFormat: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")
	fmt.Fprintf(&b, "Dialogue: 0,0:00:00.00,%s,Title,,0,0,0,,%s\\N%s\n", assTime(duration), assText(title), assText(artist))
	lines := lyrics.Parse(lrc)
	for i, l := range lines {
		end := duration
		if i+1 < len(lines) {
			end = lines[i+1].At
		}
		if end <= l.At || l.At >= duration {
			continue
		}
		fmt.Fprintf(&b, "Dialogue: 0,%s,%s,Lyric,,0,0,0,,%s\n", assTime(l.At), assTime(min(end, duration)), assText(l.Text))
	}
	return b.String()
}
