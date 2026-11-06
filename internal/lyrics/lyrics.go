package lyrics

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Line struct {
	At   time.Duration
	Text string
}

var stamp = regexp.MustCompile(`\[(\d+):(\d{1,2})(?:[.:](\d{1,3}))?\]`)
var offsetTag = regexp.MustCompile(`(?i)\[offset:([+-]?\d+)\]`)

func Parse(raw string) []Line {
	var offset time.Duration
	if m := offsetTag.FindStringSubmatch(raw); len(m) == 2 {
		n, _ := strconv.Atoi(m[1])
		offset = time.Duration(n) * time.Millisecond
	}
	var lines []Line
	for _, row := range strings.Split(raw, "\n") {
		matches := stamp.FindAllStringSubmatch(row, -1)
		text := strings.TrimSpace(stamp.ReplaceAllString(row, ""))
		if text == "" {
			continue
		}
		for _, m := range matches {
			minutes, _ := strconv.Atoi(m[1])
			seconds, _ := strconv.Atoi(m[2])
			if seconds >= 60 {
				continue
			}
			frac := m[3]
			for len(frac) < 3 {
				frac += "0"
			}
			ms, _ := strconv.Atoi(frac)
			at := time.Duration(minutes*60+seconds)*time.Second + time.Duration(ms)*time.Millisecond + offset
			lines = append(lines, Line{max(0, at), text})
		}
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].At < lines[j].At })
	return lines
}

func Current(lines []Line, pos time.Duration) (string, time.Duration) {
	i := sort.Search(len(lines), func(i int) bool { return lines[i].At > pos }) - 1
	if i < 0 {
		return "", 0
	}
	return lines[i].Text, pos - lines[i].At
}
