// Verify the signed update archives before publishing a release.
package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type config struct {
	Slug, Version string
	Updates       struct{ PublicKey, GitHub, TagPrefix string }
}

type manifest struct {
	Version, URL, Signature string
	Size                    int64
}

func main() {
	if len(os.Args) != 3 {
		fail(fmt.Errorf("usage: verify-release <artifacts> <output>"))
	}
	var c config
	data, err := os.ReadFile("mygo.json")
	fail(err)
	fail(json.Unmarshal(data, &c))
	fail(verify(c, os.Args[1], os.Args[2]))
}

func verify(c config, root, out string) error {
	pub, err := base64.StdEncoding.DecodeString(c.Updates.PublicKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid update public key")
	}
	expected := map[string]string{}
	for _, target := range []string{"windows-amd64", "darwin-arm64"} {
		expected[c.Slug+"-"+c.Version+"-"+target+".tar.gz"] = ""
		expected["update-"+target+".json"] = ""
	}
	expected[c.Slug+"-setup-"+c.Version+"-amd64.exe"] = ""
	expected[c.Slug+"-"+c.Version+"-arm64.dmg"] = ""
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if previous, ok := expected[d.Name()]; ok {
			if previous != "" || d.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("duplicate or symlink release asset: %s", d.Name())
			}
			expected[d.Name()] = path
		}
		return nil
	}); err != nil {
		return err
	}
	hashes := map[string][]byte{}
	for name, path := range expected {
		if path == "" {
			return fmt.Errorf("missing release asset: %s", name)
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		h := sha256.New()
		size, err := io.Copy(h, f)
		f.Close()
		if err != nil {
			return err
		}
		if size == 0 {
			return fmt.Errorf("empty release asset: %s", name)
		}
		hashes[name] = h.Sum(nil)
	}
	for _, target := range []string{"windows-amd64", "darwin-arm64"} {
		var m manifest
		data, err := os.ReadFile(expected["update-"+target+".json"])
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &m); err != nil {
			return err
		}
		name := c.Slug + "-" + c.Version + "-" + target + ".tar.gz"
		info, err := os.Stat(expected[name])
		if err != nil {
			return err
		}
		wantURL := "https://github.com/" + c.Updates.GitHub + "/releases/download/" + c.Updates.TagPrefix + c.Version + "/" + name
		sig, err := base64.StdEncoding.DecodeString(m.Signature)
		if err != nil || m.Version != c.Version || m.URL != wantURL || m.Size != info.Size() || !ed25519.Verify(pub, hashes[name], sig) {
			return fmt.Errorf("invalid version, URL, size or signature for %s", target)
		}
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(out)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("output directory must be empty")
	}
	names := make([]string, 0, len(expected))
	for name := range expected {
		names = append(names, name)
	}
	sort.Strings(names)
	var sums strings.Builder
	for _, name := range names {
		if err := copyFile(expected[name], filepath.Join(out, name)); err != nil {
			return err
		}
		fmt.Fprintf(&sums, "%x  %s\n", hashes[name], name)
	}
	if err := os.WriteFile(filepath.Join(out, "SHA256SUMS"), []byte(sums.String()), 0644); err != nil {
		return err
	}
	fmt.Printf("Verified %d release assets for %s\n", len(names), c.Version)
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	closeErr := out.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func fail(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
