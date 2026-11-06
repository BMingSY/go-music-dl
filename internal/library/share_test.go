package library

import "testing"

func TestSharedLinksChooseSourceAndContentKind(t *testing.T) {
	for _, test := range []struct{ query, link, source, kind string }{
		{"分享歌曲 https://music.163.com/#/song?id=123 （来自网易云）", "https://music.163.com/#/song?id=123", "netease", "song"},
		{"https://music.163.com/playlist?id=123", "https://music.163.com/playlist?id=123", "netease", "playlist"},
		{"https://music.163.com/album?id=123", "https://music.163.com/album?id=123", "netease", "album"},
		{"【视频】 https://www.bilibili.com/video/BV123。", "https://www.bilibili.com/video/BV123", "bilibili", "song"},
		{"分享：https://b23.tv/abc", "https://b23.tv/abc", "bilibili", ""},
		{"https://space.bilibili.com/123/favlist?fid=456", "https://space.bilibili.com/123/favlist?fid=456", "bilibili", "playlist"},
		{"https://music.163.com.example.org/song?id=123", "", "", ""},
		{"普通歌名", "", "", ""},
	} {
		link, source, kind := ShareQuery(test.query)
		if link != test.link || source != test.source || kind != test.kind {
			t.Errorf("%q: %q %q %q", test.query, link, source, kind)
		}
	}
}
