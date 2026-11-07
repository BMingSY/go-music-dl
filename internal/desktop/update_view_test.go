package desktop

import (
	"context"
	"testing"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestUpdateConfirmationCanBeDismissedAndReopened(t *testing.T) {
	for _, action := range []string{"稍后", "关闭弹窗", "Escape"} {
		t.Run(action, func(t *testing.T) {
			s := newApp(context.Background(), library.Config{}, "")
			s.modal, s.updateBusy = "about", true
			up := &mygo.Update{Version: "0.1.2"}
			s.offerUpdate(up)
			tt := ui.NewTester(s.view, 720, 460)
			tt.Advance(time.Second)
			if s.updateBusy || !tt.HasText("软件更新") || !tt.HasText("发现新版本 0.1.2") {
				t.Fatal("update offer did not use the application modal")
			}
			button, ok := tt.Find("下载并重启")
			if !ok || button.H < 35 || button.Y < 0 || button.Y+button.H > 460 {
				t.Fatalf("download button is clipped: %+v", button)
			}
			if action == "Escape" {
				tt.Key(0, ui.KeyEscape)
			} else if err := tt.Click(action); err != nil {
				t.Fatal(err)
			}
			if s.modal != "about" || s.pendingUpdate != up || s.updateBusy {
				t.Fatal("dismissal lost the offer or started downloading")
			}
			if err := tt.Click("下载更新"); err != nil {
				t.Fatal(err)
			}
			if s.modal != "update" || s.pendingUpdate != up {
				t.Fatal("could not return to the existing update offer")
			}
		})
	}
}

func TestUpdateProgressSurvivesClosingTheModal(t *testing.T) {
	s := newApp(context.Background(), library.Config{}, "")
	s.modal = "about"
	s.offerUpdate(&mygo.Update{Version: "0.1.2"})
	s.updateBusy, s.updateProgress, s.updateStatus = true, .37, "正在下载更新 37%"
	tt := ui.NewTester(s.view, 1080, 740)
	if err := tt.Click("关闭弹窗"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("查看下载进度"); err != nil {
		t.Fatal(err)
	}
	if s.modal != "update" || !s.updateBusy || s.updateProgress != .37 || !tt.HasText("正在下载更新 37%") {
		t.Fatal("closing and reopening lost the active download")
	}
	// A second install request must not reset or start another download.
	s.installUpdate()
	if s.updateProgress != .37 {
		t.Fatal("a repeated install request reset the active download")
	}
}

func TestUpdateCheckDoesNotReopenClosedModal(t *testing.T) {
	s := newApp(context.Background(), library.Config{}, "")
	s.updateBusy = true
	s.offerUpdate(&mygo.Update{Version: "0.1.2"})
	if s.modal != "" || s.updateBusy || s.pendingUpdate == nil {
		t.Fatal("a late check result interrupted the current page")
	}
	s.modal = "about"
	s.offerUpdate(nil)
	if s.pendingUpdate != nil || s.modal != "about" || s.updateStatus != "已经是最新版本" {
		t.Fatal("an up-to-date result offered an installation")
	}
}
