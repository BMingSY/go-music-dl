package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// The older app in this archive is never extracted or executed.
//
//go:embed windows-bundle.json
var windowsBundle []byte

type bundleLock struct {
	URL    string            `json:"url"`
	SHA256 string            `json:"sha256"`
	Files  map[string]string `json:"files"`
}

func restoreWindowsMedia(encoded []byte, cache, dest string) error {
	var lock bundleLock
	if err := json.Unmarshal(encoded, &lock); err != nil {
		return err
	}
	validHash := func(sum string) bool {
		data, err := hex.DecodeString(sum)
		return err == nil && len(data) == sha256.Size
	}
	if !validHash(lock.SHA256) || lock.URL == "" || len(lock.Files) == 0 {
		return fmt.Errorf("invalid Windows media lock")
	}
	for name, sum := range lock.Files {
		if !validHash(sum) || !allowedMediaName(name) {
			return fmt.Errorf("invalid locked media file %q", name)
		}
	}
	if err := os.MkdirAll(cache, 0755); err != nil {
		return err
	}
	archive := filepath.Join(cache, "windows-media-"+lock.SHA256+".tar.gz")
	if !verified(archive, lock.SHA256) {
		if err := fetchSource(lock.URL, archive, lock.SHA256); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(filepath.Dir(dest), ".windows-media-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	if err := extractLockedMedia(archive, stage, lock.Files); err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(stage, "media-sources.json"))
	if err != nil {
		return err
	}
	var manifest []source
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}
	if !slices.Equal(manifest, sources) {
		return fmt.Errorf("pinned Windows media does not match the source versions; update the media lock explicitly")
	}
	// On-access protection can remove a file after extraction. Missing media
	// must stop the build rather than producing an incomplete installer.
	for name, sum := range lock.Files {
		if !verified(filepath.Join(stage, filepath.FromSlash(name)), sum) {
			return fmt.Errorf("Windows media missing or changed after extraction: %s", name)
		}
	}
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	if err := os.Rename(stage, dest); err != nil {
		return err
	}
	fmt.Printf("Verified and restored %d pinned Windows media files (%s)\n", len(lock.Files), dest)
	return nil
}

func extractLockedMedia(archive, dest string, files map[string]string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	reader := tar.NewReader(gz)
	seen := make(map[string]bool)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		sum, want := files[header.Name]
		if !want {
			continue
		}
		name := header.Name
		if !allowedMediaName(name) ||
			header.Typeflag != tar.TypeReg || seen[name] || header.Size < 0 || header.Size > 32<<20 {
			return fmt.Errorf("invalid locked media entry %q", name)
		}
		file := filepath.Join(dest, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
			return err
		}
		output, err := os.OpenFile(file, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
		if err != nil {
			return err
		}
		hash := sha256.New()
		_, err = io.Copy(io.MultiWriter(output, hash), reader)
		closeErr := output.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		if hex.EncodeToString(hash.Sum(nil)) != sum {
			return fmt.Errorf("Windows media checksum mismatch: %s", name)
		}
		seen[name] = true
	}
	for name := range files {
		if !seen[name] {
			return fmt.Errorf("Windows media archive is missing %s", name)
		}
	}
	return nil
}

func allowedMediaName(name string) bool {
	if path.Clean(name) != name || strings.ContainsAny(name, "\\:") || !filepath.IsLocal(filepath.FromSlash(name)) {
		return false
	}
	if strings.HasPrefix(name, "licenses/") && strings.HasSuffix(name, ".txt") {
		return true
	}
	return path.Dir(name) == "." && (name == "ffmpeg.exe" || name == "ffprobe.exe" || name == "media-sources.json" || strings.HasSuffix(name, ".dll"))
}
