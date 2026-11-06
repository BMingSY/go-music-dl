package playback

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExportVideoWithLyricsAndCover(t *testing.T) {
	ffmpeg := testFFmpeg(t)
	audio := fixture(t, ffmpeg, "flac")
	cover := image.NewNRGBA(image.Rect(0, 0, 128, 128))
	for y := 0; y < 128; y++ {
		for x := 0; x < 128; x++ {
			cover.Set(x, y, color.NRGBA{uint8(x), uint8(y), 80, 255})
		}
	}
	var bitmap bytes.Buffer
	if err := png.Encode(&bitmap, cover); err != nil {
		t.Fatal(err)
	}
	for _, artwork := range [][]byte{nil, bitmap.Bytes()} {
		output := filepath.Join(t.TempDir(), "带歌词视频.mp4")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := ExportVideo(ctx, ffmpeg, audio, output, "测试 {title}", "作者", "[00:00]first line\n[00:02]第二行", artwork)
		cancel()
		if err != nil {
			t.Fatal(err)
		}
		probe := filepath.Join(filepath.Dir(ffmpeg), "ffprobe"+filepath.Ext(ffmpeg))
		cmd := exec.Command(probe, "-v", "error", "-show_entries", "stream=codec_name,codec_type,width,height:format=duration", "-of", "json", output)
		quietProcess(cmd)
		b, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		var info struct {
			Streams []struct {
				CodecName     string `json:"codec_name"`
				CodecType     string `json:"codec_type"`
				Width, Height int
			}
		}
		if err := json.Unmarshal(b, &info); err != nil {
			t.Fatal(err)
		}
		audioOK, videoOK := false, false
		for _, stream := range info.Streams {
			if stream.CodecType == "audio" {
				audioOK = stream.CodecName == "aac"
			}
			if stream.CodecType == "video" {
				videoOK = stream.CodecName == "h264" && stream.Width == 1280 && stream.Height == 720
			}
		}
		if !audioOK || !videoOK {
			t.Fatalf("invalid exported streams: %s", b)
		}
		before, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel = context.WithCancel(context.Background())
		cancel()
		if err = ExportVideo(ctx, ffmpeg, audio, output, "", "", "", nil); err == nil {
			t.Fatal("canceled export succeeded")
		}
		after, _ := os.ReadFile(output)
		if !bytes.Equal(before, after) {
			t.Fatal("canceled export damaged an existing video")
		}
	}
}

func TestRemuxPreservesAudioAndEmbedsMetadata(t *testing.T) {
	ffmpeg := testFFmpeg(t)
	var bitmap bytes.Buffer
	if err := png.Encode(&bitmap, image.NewNRGBA(image.Rect(0, 0, 64, 64))); err != nil {
		t.Fatal(err)
	}
	for _, ext := range []string{"mp3", "flac", "m4a"} {
		t.Run(ext, func(t *testing.T) {
			input := fixture(t, ffmpeg, ext)
			output := filepath.Join(t.TempDir(), "tagged."+ext)
			err := RemuxMetadata(context.Background(), ffmpeg, input, output, "测试歌曲", "测试作者", "测试专辑", "[00:00]测试歌词", bitmap.Bytes())
			if err != nil {
				t.Fatal(err)
			}
			probe := filepath.Join(filepath.Dir(ffmpeg), "ffprobe"+filepath.Ext(ffmpeg))
			cmd := exec.Command(probe, "-v", "error", "-show_streams", "-show_format", "-of", "json", output)
			quietProcess(cmd)
			b, err := cmd.Output()
			if err != nil {
				t.Fatal(err)
			}
			for _, expected := range []string{"测试歌曲", "测试作者", "测试专辑", "测试歌词", `"attached_pic": 1`, `"codec_type": "audio"`} {
				if !strings.Contains(string(b), expected) {
					t.Errorf("metadata missing %q: %s", expected, b)
				}
			}
		})
	}
}
