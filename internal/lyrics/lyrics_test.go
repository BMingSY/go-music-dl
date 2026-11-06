package lyrics

import (
	"testing"
	"time"
)

func TestTimedLyricsOffsetsAndRepeatedTimestamps(t *testing.T) {
	lines := Parse("[offset:-200]\n[00:02.50]第二行\n[00:00.10][00:04.100]重复\n[ar:作者]\n[00:01]第一行\n[00:99]坏时间")
	if len(lines) != 4 {
		t.Fatalf("lines=%+v", lines)
	}
	for _, tc := range []struct {
		at   time.Duration
		want string
	}{{0, "重复"}, {799 * time.Millisecond, "重复"}, {800 * time.Millisecond, "第一行"}, {2300 * time.Millisecond, "第二行"}, {5 * time.Second, "重复"}} {
		got, _ := Current(lines, tc.at)
		if got != tc.want {
			t.Errorf("at %s = %q, want %q", tc.at, got, tc.want)
		}
	}
	if got, _ := Current(Parse("[01:00]稍后"), time.Second); got != "" {
		t.Fatalf("early lyric = %q", got)
	}
}
