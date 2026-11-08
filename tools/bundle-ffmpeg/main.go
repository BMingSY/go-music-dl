// bundle-ffmpeg prepares immutable Windows media or builds Apple Silicon media.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	target := flag.String("platform", runtime.GOOS+"-"+runtime.GOARCH, "windows-amd64 or darwin-arm64")
	jobs := flag.Int("jobs", max(1, min(6, runtime.NumCPU())), "parallel compiler jobs")
	fromSource := flag.Bool("from-source", false, "explicitly rebuild Windows media from source instead of using the pinned release bundle")
	flag.Parse()
	if err := prepareMedia(*target, *jobs, *fromSource); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func prepareMedia(target string, jobs int, fromSource bool) error {
	if target != "windows-amd64" && target != "darwin-arm64" {
		return fmt.Errorf("unsupported desktop platform %s", target)
	}
	if target == "windows-amd64" && !fromSource {
		cache, err := os.UserCacheDir()
		if err != nil {
			return err
		}
		return restoreWindowsMedia(windowsBundle, filepath.Join(cache, "music-dl", "media"), filepath.Join("resources", target))
	}
	if target == "darwin-arm64" && runtime.GOOS != "darwin" {
		return fmt.Errorf("macOS media builds require macOS with Xcode command line tools")
	}
	if target == "windows-amd64" && runtime.GOOS != "linux" {
		return fmt.Errorf("build Windows media tools from source on Linux with MinGW")
	}
	return buildFromSources(target, jobs)
}
