package library

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestScanFiltersFormatsAndHonorsCancellation(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"One.mp3", "Two.FLAC", "notes.txt", "Not Music.mp4"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	tracks, err := Scan(context.Background(), dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 2 {
		t.Fatalf("tracks=%+v", tracks)
	}
	filtered, err := Scan(context.Background(), dir, "two")
	if err != nil || len(filtered) != 1 || filtered[0].Name != "Two" {
		t.Fatalf("filtered=%+v err=%v", filtered, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Scan(ctx, dir, ""); err != context.Canceled {
		t.Fatalf("cancel=%v", err)
	}
}

func TestRejectRemovedSources(t *testing.T) {
	for _, source := range []string{"qq", "soda", "kugou", "apple", "unknown"} {
		if _, err := Search(context.Background(), source, "test", Config{}); err == nil {
			t.Fatalf("source %s accepted", source)
		}
	}
}

func TestConfigRoundTripAndMalformedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	c, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	c.NeteaseCookie = "test-cookie"
	c.Volume = 0
	if err := SaveConfig(path, c); err != nil {
		t.Fatal(err)
	}
	c, err = LoadConfig(path)
	if err != nil || c.NeteaseCookie != "test-cookie" || c.Volume != 0 {
		t.Fatalf("config=%+v err=%v", c, err)
	}
	if err := SaveConfig(path, c); err != nil {
		t.Fatalf("replace: %v", err)
	}
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("malformed settings silently discarded")
	}
}
