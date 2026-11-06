package lyrics

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Word struct {
	At, End time.Duration
	Text    string
}
type Line struct {
	At          time.Duration
	Text        string
	Translation string
	End         time.Duration
	Words       []Word
}

var stamp = regexp.MustCompile(`[\[<](\d+):(\d{1,2})(?:[.:](\d{1,3}))?[\]>]`)
var offsetTag = regexp.MustCompile(`(?i)\[offset:([+-]?\d+)\]`)

func timestamp(row string, m []int, offset time.Duration) time.Duration {
	minutes, _ := strconv.Atoi(row[m[2]:m[3]])
	seconds, _ := strconv.Atoi(row[m[4]:m[5]])
	if seconds >= 60 {
		return -1
	}
	frac := ""
	if m[6] >= 0 {
		frac = row[m[6]:m[7]]
	}
	for len(frac) < 3 {
		frac += "0"
	}
	ms, _ := strconv.Atoi(frac)
	return max(0, time.Duration(minutes*60+seconds)*time.Second+time.Duration(ms)*time.Millisecond+offset)
}

// Parse handles ordinary LRC and music-lib's interleaved word timestamps.
// Only consecutive leading timestamps repeat a whole line.
func Parse(raw string) []Line {
	var offset time.Duration
	if m := offsetTag.FindStringSubmatch(raw); len(m) == 2 {
		n, _ := strconv.Atoi(m[1])
		offset = time.Duration(n) * time.Millisecond
	}
	var parsed []Line
	for _, row := range strings.Split(raw, "\n") {
		matches := stamp.FindAllStringSubmatchIndex(row, -1)
		if len(matches) == 0 || strings.TrimSpace(row[:matches[0][0]]) != "" {
			continue
		}
		enhanced := false
		for i := 1; i < len(matches); i++ {
			if strings.TrimSpace(row[matches[i-1][1]:matches[i][0]]) != "" {
				enhanced = true
				break
			}
		}
		if !enhanced {
			text := strings.TrimSpace(row[matches[len(matches)-1][1]:])
			if text == "" {
				continue
			}
			for _, m := range matches {
				if at := timestamp(row, m, offset); at >= 0 {
					parsed = append(parsed, Line{At: at, Text: text})
				}
			}
			continue
		}
		line := Line{At: timestamp(row, matches[0], offset)}
		if line.At < 0 {
			continue
		}
		var text strings.Builder
		for i, m := range matches {
			end := len(row)
			var until time.Duration
			if i+1 < len(matches) {
				end = matches[i+1][0]
				until = timestamp(row, matches[i+1], offset)
			}
			part := row[m[1]:end]
			at := timestamp(row, m, offset)
			if at < 0 {
				continue
			}
			if part != "" {
				line.Words = append(line.Words, Word{At: at, End: until, Text: part})
				text.WriteString(part)
			} else if i == len(matches)-1 {
				line.End = at
			}
		}
		line.Text = strings.TrimSpace(text.String())
		if line.Text != "" {
			parsed = append(parsed, line)
		}
	}
	sort.SliceStable(parsed, func(i, j int) bool { return parsed[i].At < parsed[j].At })
	var lines []Line
	for _, line := range parsed {
		if len(lines) > 0 && lines[len(lines)-1].At == line.At {
			previous := &lines[len(lines)-1]
			if line.Text != previous.Text {
				previous.Translation = line.Text
			}
			continue
		}
		lines = append(lines, line)
	}
	for i := range lines {
		if lines[i].End <= lines[i].At {
			if i+1 < len(lines) {
				lines[i].End = lines[i+1].At
			} else {
				lines[i].End = lines[i].At + 5*time.Second
			}
		}
		for j := range lines[i].Words {
			if lines[i].Words[j].End <= lines[i].Words[j].At {
				lines[i].Words[j].End = lines[i].End
			}
		}
	}
	return lines
}

func Index(lines []Line, pos time.Duration) int {
	return sort.Search(len(lines), func(i int) bool { return lines[i].At > pos }) - 1
}
func Current(lines []Line, pos time.Duration) (string, time.Duration) {
	i := Index(lines, pos)
	if i < 0 {
		return "", 0
	}
	return lines[i].Text, pos - lines[i].At
}
