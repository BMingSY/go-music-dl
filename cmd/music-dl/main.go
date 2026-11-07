package main

import (
	"flag"
	"log"

	"github.com/BMingSY/go-music-dl/internal/desktop"
)

func main() {
	smoke := flag.String("smoke-test", "", "capture native windows to this directory and exit")
	smokeAudio := flag.String("smoke-audio", "", "optional audio fixture for playback/pause/seek smoke")
	flag.Parse()
	if err := desktop.Run(desktop.Options{SmokeDir: *smoke, SmokeAudio: *smokeAudio}); err != nil {
		log.Fatal(err)
	}
}
