package library

import (
	"net/url"
	"regexp"
	"strings"
)

var shareURL = regexp.MustCompile(`https?://[^\s<>"'，。）》』」】]+`)

// ShareQuery accepts the surrounding text copied by the source's share action.
// Only recognized domains select a source automatically.
func ShareQuery(query string) (link, source, kind string) {
	for _, candidate := range shareURL.FindAllString(query, -1) {
		candidate = strings.TrimRight(candidate, ").,;!?")
		u, err := url.Parse(candidate)
		if err != nil {
			continue
		}
		host := strings.ToLower(u.Hostname())
		switch {
		case host == "music.163.com" || strings.HasSuffix(host, ".music.163.com") || host == "163cn.tv":
			source = "netease"
		case host == "bilibili.com" || strings.HasSuffix(host, ".bilibili.com") || host == "b23.tv":
			source = "bilibili"
		default:
			continue
		}
		path := strings.ToLower(u.Path + "/" + u.Fragment)
		switch {
		case strings.Contains(path, "playlist"), strings.Contains(path, "favlist"), strings.Contains(path, "medialist"), strings.Contains(path, "/list/ml"):
			kind = "playlist"
		case source == "netease" && strings.Contains(path, "album"):
			kind = "album"
		case strings.Contains(path, "song"), strings.Contains(path, "/video/"), strings.Contains(path, "/audio/au"):
			kind = "song"
		}
		return candidate, source, kind
	}
	return "", "", ""
}
