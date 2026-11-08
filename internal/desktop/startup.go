package desktop

import "github.com/egoist/mygo/ui"

// startupService is supplied by MyGo at runtime. Keep the OS as the source
// of truth so disabling the login item in system settings is respected.
type startupService interface {
	OpenAtLogin() bool
	SetOpenAtLogin(bool) error
}

func (s *app) startupSettings(c *ui.Context) {
	enabled := s.startup != nil && s.startup.OpenAtLogin()
	if !settingToggle(c, &enabled, "开机自启动", "登录系统后自动启动 music-dl，修改后立即生效。") || s.startup == nil {
		return
	}
	if err := s.startup.SetOpenAtLogin(enabled); err != nil {
		s.status = "设置开机自启动失败：" + err.Error()
		return
	}
	if enabled {
		s.status = "已开启开机自启动"
	} else {
		s.status = "已关闭开机自启动"
	}
}
