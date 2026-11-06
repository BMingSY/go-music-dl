package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/BMingSY/go-music-dl/internal/library"
	"github.com/egoist/mygo"
)

func main() {
	http.DefaultClient.Timeout = 25 * time.Second
	smoke := flag.String("smoke-test", "", "capture native windows to this directory and exit")
	smokeAudio := flag.String("smoke-audio", "", "optional audio fixture for playback/pause/seek smoke")
	flag.Parse()
	path, err := library.ConfigPath()
	if err != nil {
		log.Fatal(err)
	}
	if *smoke != "" {
		path = filepath.Join(*smoke, "settings.json")
	}
	cfg, err := library.LoadConfig(path)
	if err != nil {
		log.Fatal(err)
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
		if *smoke != "" {
			go func() {
				err := runSmoke(s, *smoke, *smokeAudio)
				mygo.RunOnMain(func() { smokeErr = err; mygo.App.Quit() })
			}()
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
	if smokeErr != nil {
		log.Fatal(smokeErr)
	}
	if *smoke != "" {
		message := "PASS: native windows, capture, tray lifecycle, close-to-hide\n"
		if *smokeAudio != "" {
			message += "PASS: native audio output, playback, pause, seek, hidden-window playback\n"
		}
		if err := os.WriteFile(filepath.Join(*smoke, "result.txt"), []byte(message), 0600); err != nil {
			log.Fatal(err)
		}
	}
}
