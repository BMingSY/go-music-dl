package desktop

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/egoist/mygo"
)

func (s *app) checkUpdate() {
	if s.updateBusy {
		return
	}
	s.modal = "about"
	if !mygo.Updater.Enabled() {
		s.updateStatus = "此安装暂不支持自动更新，请使用安装包更新。"
		return
	}
	s.updateBusy = true
	s.pendingUpdate = nil
	s.updateProgress = 0
	s.updateStatus = "正在检查更新…"
	go func() {
		up, err := mygo.Updater.Check(s.ctx)
		if err != nil {
			log.Printf("checking for updates: %v", err)
			s.update(func() { s.updateBusy = false; s.updateStatus = updateCheckMessage(err) })
			return
		}
		s.update(func() { s.offerUpdate(up) })
	}()
}

func (s *app) offerUpdate(up *mygo.Update) {
	s.updateBusy = false
	s.pendingUpdate = up
	if up == nil {
		s.updateStatus = updateCheckMessage(nil)
		return
	}
	s.updateStatus = "发现新版本 " + up.Version
	if s.modal == "about" {
		s.modal = "update"
	}
}

func (s *app) installUpdate() {
	if s.pendingUpdate == nil || s.updateBusy {
		return
	}
	up := s.pendingUpdate
	s.updateBusy = true
	s.updateProgress = 0
	s.updateStatus = "正在下载更新…"
	go func() {
		err := up.Install(s.ctx, func(n, total int64) {
			s.update(func() {
				if total > 0 {
					s.updateProgress = float64(n) / float64(total)
				}
				s.updateStatus = fmt.Sprintf("正在下载更新 %.0f%%", s.updateProgress*100)
			})
		})
		s.update(func() {
			s.updateBusy = false
			if err != nil {
				log.Printf("installing update: %v", err)
				s.updateStatus = "更新失败，请稍后重试。"
				return
			}
			s.save()
			mygo.App.Relaunch()
		})
	}()
}

func updateCheckMessage(err error) string {
	if err == nil {
		return "已经是最新版本"
	}
	// MyGo 0.7.4 keeps ErrNotPublished internal. Match only its wrapped
	// sentinel, not arbitrary HTTP errors or a missing update manifest.
	for cause := err; cause != nil; cause = errors.Unwrap(cause) {
		if cause.Error() == "no release is published" {
			return "暂无可用更新"
		}
	}
	if errors.Is(err, context.Canceled) {
		return "已取消检查更新"
	}
	var networkError net.Error
	if errors.Is(err, context.DeadlineExceeded) || errors.As(err, &networkError) && networkError.Timeout() {
		return "检查更新超时，请稍后重试。"
	}
	return "暂时无法检查更新，请稍后重试。"
}
