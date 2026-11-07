package desktop

import (
	"context"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo/ui"
	"github.com/guohuiyuan/music-lib/model"
	qrcode "github.com/skip2/go-qrcode"
)

func (s *app) startLogin(source string) {
	if s.loginCancel != nil {
		s.loginCancel()
	}
	ctx, cancel := context.WithCancel(s.ctx)
	s.loginCancel = cancel
	s.loginID++
	id := s.loginID
	s.loginSource = source
	s.loginStatus = "正在生成二维码…"
	s.loginImage = nil
	s.modal = "login"
	cfg := s.cfg
	go func() {
		session, err := library.CreateLogin(source, cfg)
		if err != nil {
			s.update(func() {
				if s.loginID == id {
					s.loginStatus = "二维码生成失败，请重试"
				}
			})
			return
		}
		if ctx.Err() != nil {
			return
		}
		code, err := qrcode.Encode(session.URL, qrcode.Medium, 240)
		if err != nil {
			return
		}
		s.update(func() {
			if s.loginID != id {
				return
			}
			s.loginImage, _ = ui.DecodeBitmap(code)
			s.loginStatus = "请使用对应平台的 App 扫码登录"
		})
		timer := time.NewTimer(5 * time.Minute)
		defer timer.Stop()
		tick := time.NewTicker(2 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				s.update(func() {
					if s.loginID == id {
						s.loginStatus = "二维码已过期，请重新获取"
					}
				})
				return
			case <-tick.C:
				r, e := library.CheckLogin(source, session.Key, cfg)
				if e != nil || r == nil {
					continue
				}
				done := r.Status == model.QRLoginStatusSuccess || r.Status == model.QRLoginStatusExpired || r.Status == model.QRLoginStatusFailed
				s.update(func() {
					if s.loginID != id || ctx.Err() != nil {
						return
					}
					switch r.Status {
					case model.QRLoginStatusScanned:
						s.loginStatus = "已扫码，请在手机上确认"
					case model.QRLoginStatusSuccess:
						cookie := library.LoginCookie(r)
						if cookie == "" {
							s.loginStatus = "登录未返回 Cookie，请重试"
							return
						}
						if source == "netease" {
							s.cfg.NeteaseCookie = cookie
						} else {
							s.cfg.BilibiliCookie = cookie
						}
						s.save()
						s.loginStatus = "登录成功"
						s.loginImage = nil
					case model.QRLoginStatusExpired:
						s.loginStatus = "二维码已过期，请重新获取"
					case model.QRLoginStatusFailed:
						s.loginStatus = "登录失败，请重试"
					}
				})
				if done {
					return
				}
			}
		}
	}()
}

func (s *app) loginView(c *ui.Context) {
	ui.Column(c).Grow(1).AlignItems(ui.Center).Gap(14).Children(func() {
		if s.loginImage != nil {
			ui.Image(c, s.loginImage).Size(240, 240)
		}
		ui.Text(c, s.loginStatus).FontSize(14).MaxLines(2)
		ui.Button(c, "重新获取二维码").OnClick(func() { s.startLogin(s.loginSource) })
	})
}
