// Package desktop implements the native music-dl desktop application.
package desktop

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo"
)

// Options controls the optional native smoke test.
type Options struct {
	SmokeDir   string
	SmokeAudio string
}

// Run loads the user configuration and runs the desktop event loop.
func Run(options Options) error {
	http.DefaultClient.Timeout = 25 * time.Second
	path, err := library.ConfigPath()
	if err != nil {
		return err
	}
	if options.SmokeDir != "" {
		path = filepath.Join(options.SmokeDir, "settings.json")
	}
	cfg, err := library.LoadConfig(path)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := newApp(ctx, cfg, path)
	mygo.App.SetName("music-dl")
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { s.save(); s.quitting = true; cancel() })
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) {
		s.stop()
		if s.tray != nil {
			s.tray.Destroy()
		}
	})
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.OnActivate(func(bool) {
		if s.win != nil {
			s.win.Show()
			s.win.Focus()
		}
	})
	var smokeErr error
	mygo.App.WhenReady(func() {
		s.ready()
		if options.SmokeDir != "" {
			go func() {
				err := runSmoke(s, options.SmokeDir, options.SmokeAudio)
				mygo.RunOnMain(func() { smokeErr = err; mygo.App.Quit() })
			}()
		}
	})
	if err := mygo.App.Run(); err != nil {
		return err
	}
	if smokeErr != nil {
		return smokeErr
	}
	if options.SmokeDir != "" {
		message := "PASS: native windows, capture, tray lifecycle, close-to-hide\n"
		if options.SmokeAudio != "" {
			message += "PASS: native audio output, playback, pause, seek, hidden-window playback\n"
			message += "PASS: saved queue and position restore paused; pressing play resumes from the saved position\n"
		}
		return os.WriteFile(filepath.Join(options.SmokeDir, "result.txt"), []byte(message), 0600)
	}
	return nil
}
