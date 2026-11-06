// bundle-ffmpeg downloads pinned, checksum-verified decoder builds into MyGo's
// platform resource directory. It never installs anything system-wide.
package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type archive struct {
	url, sum string
	names    []string
}

var archives = map[string][]archive{
	"windows-amd64": {{"https://github.com/BtbN/FFmpeg-Builds/releases/download/autobuild-2026-11-06-12-50/ffmpeg-n9.0.2-47-gb204e94e90-win64-gpl-9.0.zip", "8436c89156c3695659bf968e26bab5b08351ed8c86c3e9dbe3d6cc66a23af0a8", []string{"ffmpeg.exe", "ffprobe.exe", "LICENSE.txt"}}},
	"darwin-amd64": {
		{"https://ffmpeg.martin-riedl.de/download/macos/amd64/1789931006_9.0.2/ffmpeg.zip", "7c6b4125b191cbf773832dc51f424cf2b6bb7da43007d1e066f95909e47cacd4", []string{"ffmpeg"}},
		{"https://ffmpeg.martin-riedl.de/download/macos/amd64/1789931006_9.0.2/ffprobe.zip", "2322438ed2f6319a691291b247d09c69dcaa3a982460d1f269a7e1af335cfdfd", []string{"ffprobe"}},
	},
	"darwin-arm64": {
		{"https://ffmpeg.martin-riedl.de/download/macos/arm64/1789931890_9.0.2/ffmpeg.zip", "c8ed4c4e6978a03c485edbfe4e0a5dc2380f8a30bba5150531b31b094492d924", []string{"ffmpeg"}},
		{"https://ffmpeg.martin-riedl.de/download/macos/arm64/1789931890_9.0.2/ffprobe.zip", "fcbe839537485eaee7a7a8bc5cbc0f90d53617e80943e8a5b2e31cb851197ea6", []string{"ffprobe"}},
	},
}

func main() {
	target := flag.String("platform", runtime.GOOS+"-"+runtime.GOARCH, "windows-amd64, darwin-amd64 or darwin-arm64")
	flag.Parse()
	if err := run(*target); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(target string) error {
	list, ok := archives[target]
	if !ok {
		return fmt.Errorf("unsupported desktop platform %s", target)
	}
	dest := filepath.Join("resources", target)
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	cache = filepath.Join(cache, "music-dl", "build")
	if err = os.MkdirAll(cache, 0755); err != nil {
		return err
	}
	for _, a := range list {
		path := filepath.Join(cache, a.sum+".zip")
		if !verified(path, a.sum) {
			var err error
			for attempt := 0; attempt < 3; attempt++ {
				err = fetch(a.url, path)
				if err == nil {
					break
				}
				time.Sleep(time.Duration(attempt+1) * time.Second)
			}
			if err != nil {
				return err
			}
			if !verified(path, a.sum) {
				os.Remove(path)
				return fmt.Errorf("FFmpeg checksum mismatch")
			}
		}
		if err := extract(path, dest, a.names); err != nil {
			return err
		}
	}
	return nil
}
func verified(path, sum string) bool {
	f, e := os.Open(path)
	if e != nil {
		return false
	}
	defer f.Close()
	h := sha256.New()
	if _, e = io.Copy(h, f); e != nil {
		return false
	}
	return hex.EncodeToString(h.Sum(nil)) == sum
}
func fetch(url, path string) error {
	fmt.Println("Downloading", filepath.Base(url))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 music-dl-build")
	resp, err := (&http.Client{Timeout: 15 * time.Minute}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("FFmpeg download: HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(path + ".part")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err = io.Copy(f, io.LimitReader(resp.Body, 1<<30))
	if e := f.Close(); err == nil {
		err = e
	}
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
func extract(path, dest string, names []string) error {
	z, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer z.Close()
	seen := map[string]bool{}
	for _, f := range z.File {
		base := filepath.Base(f.Name)
		want := false
		for _, name := range names {
			if base == name {
				want = true
				break
			}
		}
		if !want {
			continue
		}
		if seen[base] {
			return fmt.Errorf("duplicate archive entry %s", base)
		}
		seen[base] = true
		if f.Mode()&os.ModeSymlink != 0 || f.UncompressedSize64 > 512<<20 {
			return fmt.Errorf("invalid archive entry %s", base)
		}
		in, e := f.Open()
		if e != nil {
			return e
		}
		out, e := os.OpenFile(filepath.Join(dest, base), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
		if e != nil {
			in.Close()
			return e
		}
		_, e = io.Copy(out, io.LimitReader(in, 512<<20))
		in.Close()
		if closeErr := out.Close(); e == nil {
			e = closeErr
		}
		if e != nil {
			return e
		}
		if !strings.HasPrefix(base, "LICENSE") {
			if e = os.Chmod(filepath.Join(dest, base), 0755); e != nil {
				return e
			}
		}
	}
	for _, n := range names {
		if !seen[n] && !strings.HasPrefix(n, "LICENSE") {
			return fmt.Errorf("archive is missing %s", n)
		}
	}
	return nil
}
