// test-update runs a signed upgrade and relaunch in an isolated installation.
// It uses a fresh key and generated fixture apps, never a user's installed app.
package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		fmt.Println("Update installation smoke requires Windows or macOS")
		return nil
	}
	work, err := os.MkdirTemp("", "musicdl-upgrade-test-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	pk := base64.StdEncoding.EncodeToString(pub)
	src := filepath.Join(work, "fixture.go")
	if err = os.WriteFile(src, []byte(fixture), 0600); err != nil {
		return err
	}
	var oldBinary, newRoot string
	for _, version := range []string{"1.0.0", "1.0.1"} {
		root := filepath.Join(work, version)
		binary := filepath.Join(root, "musicdl-test.exe")
		if runtime.GOOS == "darwin" {
			binary = filepath.Join(root, "MusicDLTest.app", "Contents", "MacOS", "musicdl-test")
			if err = os.MkdirAll(filepath.Dir(binary), 0755); err != nil {
				return err
			}
			plist := `<?xml version="1.0"?><plist version="1.0"><dict><key>CFBundleExecutable</key><string>musicdl-test</string><key>CFBundleIdentifier</key><string>io.github.bmingsy.musicdl.updatetest</string><key>CFBundleName</key><string>MusicDLTest</string><key>CFBundlePackageType</key><string>APPL</string><key>CFBundleShortVersionString</key><string>` + version + `</string></dict></plist>`
			if err = os.WriteFile(filepath.Join(root, "MusicDLTest.app", "Contents", "Info.plist"), []byte(plist), 0644); err != nil {
				return err
			}
		} else {
			if err = os.MkdirAll(root, 0755); err != nil {
				return err
			}
		}
		flags := "-X github.com/egoist/mygo.production=1 -X github.com/egoist/mygo.packageVersion=" + version + " -X github.com/egoist/mygo.packageName=MusicDLTest -X github.com/egoist/mygo.packageUpdateKey=" + pk + " -X github.com/egoist/mygo.packageUpdateFeed=https://example.invalid/update.json"
		cmd := exec.Command("go", "build", "-ldflags", flags, "-o", binary, src)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err = cmd.Run(); err != nil {
			return err
		}
		if version == "1.0.0" {
			oldBinary = binary
		} else {
			newRoot = root
		}
	}
	archive := filepath.Join(work, "update.tar.gz")
	if err = pack(newRoot, archive); err != nil {
		return err
	}
	data, err := os.ReadFile(archive)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	manifest := map[string]any{"version": "1.0.1", "url": archive, "size": len(data), "signature": base64.StdEncoding.EncodeToString(ed25519.Sign(key, sum[:]))}
	b, _ := json.Marshal(manifest)
	if err = os.WriteFile(filepath.Join(work, "manifest.json"), b, 0600); err != nil {
		return err
	}
	cmd := exec.Command(oldBinary, work)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	timer := time.NewTimer(60 * time.Second)
	defer timer.Stop()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case err := <-done:
			if err != nil {
				return err
			}
			done = nil
		case <-timer.C:
			_ = cmd.Process.Kill()
			return fmt.Errorf("update/relaunch did not finish before timeout; work=%s", work)
		case <-tick.C:
			if b, e := os.ReadFile(filepath.Join(work, "error.txt")); e == nil {
				return fmt.Errorf("update: %s", b)
			}
			if b, e := os.ReadFile(filepath.Join(work, "pass.txt")); e == nil {
				fmt.Print(string(b))
				return nil
			}
		}
	}
}
func pack(root, dest string) error {
	f, e := os.Create(dest)
	if e != nil {
		return e
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	e = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		h, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		h.Name = strings.ReplaceAll(rel, "\\", "/")
		if err = tw.WriteHeader(h); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(tw, in)
		return err
	})
	if err := tw.Close(); e == nil {
		e = err
	}
	if err := gz.Close(); e == nil {
		e = err
	}
	if err := f.Close(); e == nil {
		e = err
	}
	return e
}

const fixture = `package main
import("context";"fmt";"io";"os";"path/filepath";"github.com/egoist/mygo";"github.com/egoist/mygo/ui")
type source struct{dir string;bad bool}
func(s *source)Manifest(context.Context,bool)(io.ReadCloser,error){return os.Open(filepath.Join(s.dir,"manifest.json"))}
func(s *source)Open(_ context.Context,path string)(io.ReadCloser,error){if s.bad{return os.Open(filepath.Join(s.dir,"bad.tar.gz"))};return os.Open(path)}
func main(){dir:=os.Args[1];if mygo.App.Version()=="1.0.1"{os.WriteFile(filepath.Join(dir,"pass.txt"),[]byte("PASS: invalid signature rejected; signed update installed; restarted into 1.0.1\n"),0600);return}
 fail:=func(err error){os.WriteFile(filepath.Join(dir,"error.txt"),[]byte(err.Error()),0600);mygo.App.Quit()}
 mygo.App.WhenReady(func(){mygo.NewWindow(mygo.WindowOptions{Title:"MusicDL update verification",Width:300,Height:100,Content:ui.View(func(c *ui.Context){ui.Text(c,"Checking signed update and restart")})});go func(){ctx:=context.Background();src:=&source{dir:dir,bad:true};mygo.Updater.SetSource(src);up,err:=mygo.Updater.Check(ctx);if err!=nil||up==nil{fail(fmt.Errorf("check: %v",err));return};archive,err:=os.ReadFile(filepath.Join(dir,"update.tar.gz"));if err!=nil{fail(err);return};archive[len(archive)/2]^=1;os.WriteFile(filepath.Join(dir,"bad.tar.gz"),archive,0600);if err=up.Install(ctx,nil);err==nil{fail(fmt.Errorf("tampered archive installed"));return};src.bad=false;if err=up.Install(ctx,nil);err!=nil{fail(err);return};mygo.App.Relaunch()}()});if err:=mygo.App.Run();err!=nil{fail(err)}}
`
