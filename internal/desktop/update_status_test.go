package desktop

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo/ui"
)

func TestUpdateCheckMessagesDistinguishNoReleaseFromFailures(t *testing.T) {
	noRelease := fmt.Errorf("mygo: checking for updates: %w", fmt.Errorf("%w in BMingSY/go-music-dl tagged native-v", errors.New("no release is published")))
	for _, test := range []struct {
		name string
		err  error
		want string
	}{
		{"no newer version", nil, "已经是最新版本"},
		{"no published release", noRelease, "暂无可用更新"},
		{"timeout", fmt.Errorf("checking: %w", context.DeadlineExceeded), "检查更新超时，请稍后重试。"},
		{"network timeout", &net.DNSError{IsTimeout: true}, "检查更新超时，请稍后重试。"},
		{"canceled", context.Canceled, "已取消检查更新"},
		{"missing manifest", errors.New("mygo: checking for updates: HTTP 404"), "暂时无法检查更新，请稍后重试。"},
		{"invalid manifest", errors.New("mygo: checking for updates: incomplete manifest"), "暂时无法检查更新，请稍后重试。"},
		{"unrelated response", errors.New("server returned no release is published as a malformed response"), "暂时无法检查更新，请稍后重试。"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := updateCheckMessage(test.err); got != test.want {
				t.Fatalf("message=%q want %q", got, test.want)
			}
		})
	}
}

func TestAboutUpdateStatusIsInlineAndNotDuplicatedInToast(t *testing.T) {
	s := newApp(context.Background(), library.Config{}, "")
	s.modal = "about"
	s.updateStatus = "暂无可用更新"
	tt := ui.NewTester(s.view, 1080, 740)
	count := 0
	for _, text := range tt.Texts() {
		if text == s.updateStatus {
			count++
		}
		if text == "检查更新失败" || text == "mygo: checking for updates" {
			t.Fatal("raw update error shown in UI")
		}
	}
	if count != 1 {
		t.Fatalf("update status appears %d times, want once", count)
	}
	if s.status != "" {
		t.Fatal("update status leaked into unrelated global status")
	}
	for _, link := range []struct{ name, url string }{{"BMingSY/go-music-dl", "https://github.com/BMingSY/go-music-dl"}, {"guohuiyuan/go-music-dl", "https://github.com/guohuiyuan/go-music-dl"}} {
		if err := tt.Click(link.name); err != nil {
			t.Fatal(err)
		}
		urls := tt.OpenedURLs()
		if len(urls) == 0 || urls[len(urls)-1] != link.url {
			t.Fatalf("%s opens the wrong project", link.name)
		}
	}
}
