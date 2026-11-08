package desktop

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo/ui"
)

type testStartupService struct {
	enabled bool
	err     error
	calls   []bool
}

func (s *testStartupService) OpenAtLogin() bool { return s.enabled }
func (s *testStartupService) SetOpenAtLogin(enabled bool) error {
	s.calls = append(s.calls, enabled)
	if s.err == nil {
		s.enabled = enabled
	}
	return s.err
}

func TestStartupToggleUsesSystemStateAndAppliesImmediately(t *testing.T) {
	s := newApp(context.Background(), library.Config{}, "")
	service := &testStartupService{}
	s.startup, s.modal = service, "settings"
	tt := ui.NewTester(s.view, 1080, 740)
	tt.Advance(time.Second)
	if service.enabled || len(service.calls) != 0 {
		t.Fatal("opening settings changed the startup registration")
	}
	if err := tt.Click("开机自启动"); err != nil {
		t.Fatal(err)
	}
	if !service.enabled || len(service.calls) != 1 || !service.calls[0] || !tt.HasText("已开启开机自启动") {
		t.Fatalf("toggle did not enable startup immediately: %+v, status=%q", service, s.status)
	}
	// Simulate the user disabling the login item in system settings while
	// the app is running. The next click must enable it again.
	service.enabled = false
	tt.Frame()
	if err := tt.Click("开机自启动"); err != nil {
		t.Fatal(err)
	}
	if !service.enabled || len(service.calls) != 2 || !service.calls[1] {
		t.Fatalf("toggle ignored the changed system state: %+v", service)
	}
	if err := tt.Click("开机自启动"); err != nil {
		t.Fatal(err)
	}
	if service.enabled || len(service.calls) != 3 || service.calls[2] || !tt.HasText("已关闭开机自启动") {
		t.Fatalf("toggle did not remove startup: %+v, status=%q", service, s.status)
	}
}

func TestStartupToggleFailureKeepsSystemStateAndAllowsRetry(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "enable", true: "disable"}[enabled], func(t *testing.T) {
			s := newApp(context.Background(), library.Config{}, "")
			service := &testStartupService{enabled: enabled, err: errors.New("fixture: permission denied")}
			s.startup, s.modal = service, "settings"
			tt := ui.NewTester(s.view, 1080, 740)
			tt.Advance(time.Second)
			if err := tt.Click("开机自启动"); err != nil {
				t.Fatal(err)
			}
			if service.enabled != enabled || len(service.calls) != 1 || service.calls[0] == enabled || !tt.HasText("设置开机自启动失败：fixture: permission denied") {
				t.Fatalf("startup failure was not reported: %+v, status=%q", service, s.status)
			}
			service.err = nil
			if err := tt.Click("开机自启动"); err != nil {
				t.Fatal(err)
			}
			if service.enabled == enabled || len(service.calls) != 2 || service.calls[1] == enabled {
				t.Fatalf("startup failure could not be retried: %+v", service)
			}
		})
	}
}
