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

func TestVerbatimWordsAndBilingualRowsStayTogether(t *testing.T) {
	raw := "[00:01.000]君[00:01.500]の[00:02.000]声[00:03.000]\n[00:01.000]ki mi no ko e[00:03.000]\n[00:01.000]你的声音[00:03.000]\n[00:04.000]次の行[00:06.000]"
	lines := Parse(raw)
	if len(lines) != 2 {
		t.Fatalf("word stamps became rows: %+v", lines)
	}
	if lines[0].Text != "君の声" || lines[0].Translation != "你的声音" || len(lines[0].Words) != 3 {
		t.Fatalf("lost bilingual lyric: %+v", lines[0])
	}
	if text, _ := Current(lines, 2500*time.Millisecond); text != "君の声" {
		t.Fatalf("selected word or translation as whole row: %q", text)
	}
	if lines[0].Words[1].At != 1500*time.Millisecond || lines[0].Words[1].End != 2*time.Second {
		t.Fatal("lost word timing")
	}
}
