package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func releaseFixture(t *testing.T) (config, string) {
	t.Helper()
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	c := config{Slug: "music-dl", Version: "0.1.0"}
	c.Updates.PublicKey = base64.StdEncoding.EncodeToString(pub)
	c.Updates.GitHub, c.Updates.TagPrefix = "BMingSY/go-music-dl", "native-v"
	root := t.TempDir()
	for _, target := range []string{"windows-amd64", "darwin-arm64"} {
		name := "music-dl-0.1.0-" + target + ".tar.gz"
		data := []byte("signed fixture " + target)
		sum := sha256.Sum256(data)
		m := manifest{Version: c.Version, URL: "https://github.com/BMingSY/go-music-dl/releases/download/native-v0.1.0/" + name, Size: int64(len(data)), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(key, sum[:]))}
		encoded, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, name, data)
		writeFixture(t, root, "update-"+target+".json", encoded)
	}
	for _, name := range []string{"music-dl-setup-0.1.0-amd64.exe", "music-dl-0.1.0-arm64.dmg"} {
		writeFixture(t, root, name, []byte("installer fixture"))
	}
	return c, root
}

func writeFixture(t *testing.T, root, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyRelease(t *testing.T) {
	c, root := releaseFixture(t)
	// Stale artifacts from the retired platform must not be published.
	writeFixture(t, root, "music-dl-0.1.0-amd64.dmg", []byte("retired Intel installer"))
	writeFixture(t, root, "update-darwin-amd64.json", []byte("retired Intel manifest"))
	out := filepath.Join(t.TempDir(), "release-assets")
	if err := verify(c, root, out); err != nil {
		t.Fatal(err)
	}
	files, err := os.ReadDir(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 7 {
		t.Fatalf("got %d assets, want 6 assets and checksums", len(files))
	}
	sums, err := os.ReadFile(filepath.Join(out, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.Split(strings.TrimSpace(string(sums)), "\n")) != 6 {
		t.Fatal("checksums do not cover every published asset")
	}
}

func TestRejectInvalidRelease(t *testing.T) {
	for _, fault := range []string{"archive", "version", "url", "signature", "size", "missing"} {
		t.Run(fault, func(t *testing.T) {
			c, root := releaseFixture(t)
			if fault == "archive" {
				writeFixture(t, root, "music-dl-0.1.0-darwin-arm64.tar.gz", []byte("tampered archive"))
			} else if fault == "missing" {
				if err := os.Remove(filepath.Join(root, "music-dl-setup-0.1.0-amd64.exe")); err != nil {
					t.Fatal(err)
				}
			} else {
				name := "update-darwin-arm64.json"
				data, err := os.ReadFile(filepath.Join(root, name))
				if err != nil {
					t.Fatal(err)
				}
				var m manifest
				if err := json.Unmarshal(data, &m); err != nil {
					t.Fatal(err)
				}
				switch fault {
				case "version":
					m.Version = "1.0.0"
				case "url":
					m.URL = strings.Replace(m.URL, "native-v0.1.0", "v0.1.0", 1)
				case "signature":
					m.Signature = ""
				case "size":
					m.Size++
				}
				data, err = json.Marshal(m)
				if err != nil {
					t.Fatal(err)
				}
				writeFixture(t, root, name, data)
			}
			out := filepath.Join(t.TempDir(), "release-assets")
			if err := verify(c, root, out); err == nil {
				t.Fatal("invalid release accepted")
			}
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Fatal("invalid release staged for publishing")
			}
		})
	}
}
