package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mediaFixture(t *testing.T, fault string) (bundleLock, []byte) {
	t.Helper()
	manifest, err := json.Marshal(sources)
	if err != nil {
		t.Fatal(err)
	}
	if fault == "sources" {
		manifest = []byte(`[]`)
	}
	files := map[string][]byte{
		"ffmpeg.exe": []byte("fixture decoder"), "ffprobe.exe": []byte("fixture probe"),
		"avcodec-63.dll": []byte("fixture library"), "media-sources.json": manifest,
		"licenses/ffmpeg-LICENSE.txt": []byte("fixture license"),
	}
	lock := bundleLock{Files: make(map[string]string)}
	var data bytes.Buffer
	gz := gzip.NewWriter(&data)
	tarWriter := tar.NewWriter(gz)
	for name, content := range files {
		sum := sha256.Sum256(content)
		lock.Files[name] = hex.EncodeToString(sum[:])
		if fault == "missing" && name == "ffprobe.exe" {
			continue
		}
		if err := tarWriter.WriteHeader(&tar.Header{Name: name, Size: int64(len(content)), Mode: 0755, Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	// The pinned app archive also has an old application and settings. They
	// must never be installed as part of the media bundle.
	for _, name := range []string{"music-dl.exe", "settings.json"} {
		if err := tarWriter.WriteHeader(&tar.Header{Name: name, Size: 3, Mode: 0755, Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write([]byte("old")); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data.Bytes())
	lock.SHA256 = hex.EncodeToString(sum[:])
	if fault == "file-hash" {
		lock.Files["ffmpeg.exe"] = lock.SHA256
	}
	return lock, data.Bytes()
}

func TestRestorePinnedWindowsMediaAndReuseVerifiedCache(t *testing.T) {
	lock, data := mediaFixture(t, "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(data) }))
	defer server.Close()
	lock.URL = server.URL
	encoded, _ := json.Marshal(lock)
	cache, dest := t.TempDir(), filepath.Join(t.TempDir(), "media")
	if err := restoreWindowsMedia(encoded, cache, dest); err != nil {
		t.Fatal(err)
	}
	for name, sum := range lock.Files {
		if !verified(filepath.Join(dest, filepath.FromSlash(name)), sum) {
			t.Fatalf("media changed: %s", name)
		}
	}
	for _, name := range []string{"music-dl.exe", "settings.json"} {
		if _, err := os.Stat(filepath.Join(dest, name)); !os.IsNotExist(err) {
			t.Fatalf("non-media file was extracted: %s", name)
		}
	}
	server.Close()
	if err := os.WriteFile(filepath.Join(dest, "avcodec-63.dll"), []byte("changed local output"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := restoreWindowsMedia(encoded, cache, dest); err != nil {
		t.Fatalf("verified cache was not usable offline: %v", err)
	}
	if !verified(filepath.Join(dest, "avcodec-63.dll"), lock.Files["avcodec-63.dll"]) {
		t.Fatal("changed output was retained")
	}
}

func TestInvalidPinnedMediaPreservesExistingBundle(t *testing.T) {
	for _, fault := range []string{"archive-hash", "file-hash", "missing", "sources"} {
		t.Run(fault, func(t *testing.T) {
			lock, data := mediaFixture(t, fault)
			if fault == "archive-hash" {
				data = append(data, []byte("tampering")...)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(data) }))
			defer server.Close()
			lock.URL = server.URL
			encoded, _ := json.Marshal(lock)
			dest := t.TempDir()
			marker := filepath.Join(dest, "old-media")
			if err := os.WriteFile(marker, []byte("keep"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := restoreWindowsMedia(encoded, t.TempDir(), dest); err == nil {
				t.Fatal("invalid pinned media was accepted")
			}
			if content, err := os.ReadFile(marker); err != nil || string(content) != "keep" {
				t.Fatal("invalid media replaced the existing bundle")
			}
		})
	}
}

func TestMediaLockRejectsApplicationAndUnsafePaths(t *testing.T) {
	for _, name := range []string{"music-dl.exe", "../outside.dll", "/outside.dll", `..\outside.dll`, "C:/outside.dll", "licenses/../../outside.dll"} {
		t.Run(name, func(t *testing.T) {
			if allowedMediaName(name) {
				t.Fatalf("unsafe media name accepted: %q", name)
			}
		})
	}
}

func TestPinnedMediaDoesNotDuplicateSharedResources(t *testing.T) {
	var lock bundleLock
	if err := json.Unmarshal(windowsBundle, &lock); err != nil {
		t.Fatal(err)
	}
	shared := filepath.Join("..", "..", "resources")
	if err := filepath.WalkDir(filepath.Join(shared, "licenses"), func(file string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		name, err := filepath.Rel(shared, file)
		if err != nil {
			return err
		}
		if _, duplicate := lock.Files[filepath.ToSlash(name)]; duplicate {
			t.Errorf("pinned media duplicates the shared resource %s", name)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for name := range lock.Files {
		if strings.HasPrefix(name, "licenses/") {
			continue
		}
		if _, err := os.Stat(filepath.Join(shared, filepath.FromSlash(name))); err == nil {
			t.Errorf("pinned media duplicates the shared resource %s", name)
		}
	}
}
