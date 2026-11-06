// bundle-ffmpeg builds the audio and export tools from pinned source archives.
// Windows is cross-built with MinGW on Linux; macOS uses Apple's toolchain.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type source struct{ Name, Version, URL, SHA256, License, Mirror string }

var sources = []source{
	{"zlib", "1.3.1", "https://zlib.net/fossils/zlib-1.3.1.tar.gz", "9a93b2b7dfdac77ceba5a558a580e74667dd6fede4585b91eefb60f03b72df23", "LICENSE", ""},
	{"freetype", "2.13.3", "https://download-mirror.savannah.gnu.org/releases/freetype/freetype-2.13.3.tar.xz", "0550350666d427c74daeb85d5ac7bb353acba5f76956395995311a9c6f063289", "docs/FTL.TXT", "https://distfiles.macports.org/freetype/freetype-2.13.3.tar.xz"},
	{"harfbuzz", "10.4.0", "https://github.com/harfbuzz/harfbuzz/releases/download/10.4.0/harfbuzz-10.4.0.tar.xz", "480b6d25014169300669aa1fc39fb356c142d5028324ea52b3a27648b9beaad8", "COPYING", ""},
	{"fribidi", "1.0.16", "https://github.com/fribidi/fribidi/releases/download/v1.0.16/fribidi-1.0.16.tar.xz", "1b1cde5b235d40479e91be2f0e88a309e3214c8ab470ec8a2744d82a5a9ea05c", "COPYING", ""},
	{"libass", "0.17.4", "https://github.com/libass/libass/releases/download/0.17.4/libass-0.17.4.tar.xz", "78f1179b838d025e9c26e8fef33f8092f65611444ffa1bfc0cfac6a33511a05a", "COPYING", ""},
	{"ffmpeg", "9.0.2", "https://ffmpeg.org/releases/ffmpeg-9.0.2.tar.xz", "8c3850283eb25fa026482078a04051e0be17347b09ef81a0849bec15a96e002e", "COPYING.LGPLv2.1", ""},
}

type builder struct {
	target, root, prefix, work, src  string
	cc, cxx, ar, strip, arch, system string
	env                              []string
	jobs                             string
}

func main() {
	target := flag.String("platform", runtime.GOOS+"-"+runtime.GOARCH, "windows-amd64, darwin-amd64 or darwin-arm64")
	jobs := flag.Int("jobs", max(1, min(6, runtime.NumCPU())), "parallel compiler jobs")
	flag.Parse()
	if err := build(*target, *jobs); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func build(target string, jobs int) error {
	if target != "windows-amd64" && target != "darwin-amd64" && target != "darwin-arm64" {
		return fmt.Errorf("unsupported desktop platform %s", target)
	}
	if strings.HasPrefix(target, "darwin-") && runtime.GOOS != "darwin" {
		return fmt.Errorf("macOS media builds require macOS with Xcode command line tools")
	}
	if target == "windows-amd64" && runtime.GOOS != "linux" {
		return fmt.Errorf("build Windows media tools on Linux with MinGW; CI provides the resulting resources")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	// Changing the recipe invalidates compiled objects, while source downloads persist.
	recipe, err := os.ReadFile("tools/bundle-ffmpeg/main.go")
	if err != nil {
		return err
	}
	sum := sha256.Sum256(recipe)
	b := &builder{target: target, root: filepath.Join(cache, "music-dl", "media"), jobs: fmt.Sprint(jobs)}
	b.src = filepath.Join(b.root, "sources")
	b.work = filepath.Join(b.root, target+"-"+hex.EncodeToString(sum[:6]))
	b.prefix = filepath.Join(b.work, "prefix")
	b.arch = "x86_64"
	b.system = "darwin"
	b.cc = "clang"
	b.cxx = "clang++"
	b.ar = "ar"
	b.strip = "strip"
	cflags := "-Os -ffunction-sections -fdata-sections -fPIC"
	ldflags := ""
	if target == "windows-amd64" {
		b.system = "windows"
		b.cc = "x86_64-w64-mingw32-gcc"
		b.cxx = "x86_64-w64-mingw32-g++"
		b.ar = "x86_64-w64-mingw32-ar"
		b.strip = "x86_64-w64-mingw32-strip"
		ldflags = "-static-libgcc -static-libstdc++ -Wl,--gc-sections"
	} else {
		if target == "darwin-arm64" {
			b.arch = "arm64"
		}
		cflags += " -arch " + b.arch + " -mmacosx-version-min=13.0"
		ldflags = "-arch " + b.arch + " -mmacosx-version-min=13.0"
	}
	b.env = append(os.Environ(), "CC="+b.cc, "CXX="+b.cxx, "AR="+b.ar, "STRIP="+b.strip, "CFLAGS="+cflags, "CXXFLAGS="+cflags, "LDFLAGS="+ldflags, "PKG_CONFIG_LIBDIR="+filepath.Join(b.prefix, "lib", "pkgconfig"), "PKG_CONFIG_PATH="+filepath.Join(b.prefix, "lib", "pkgconfig"))
	for _, p := range []string{b.src, b.work, b.prefix} {
		if err = os.MkdirAll(p, 0755); err != nil {
			return err
		}
	}
	for _, s := range sources {
		if err = b.prepare(s); err != nil {
			return err
		}
	}
	if err = b.dependencies(cflags); err != nil {
		return err
	}
	if err = b.ffmpeg(); err != nil {
		return err
	}
	return b.bundle()
}
func (b *builder) run(dir string, args ...string) error {
	fmt.Println("Building:", args[0], strings.Join(args[1:], " "))
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Env = b.env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", args[0], err)
	}
	return nil
}
func (b *builder) source(name string) string {
	for _, s := range sources {
		if s.Name == name {
			return filepath.Join(b.src, s.Name+"-"+s.Version)
		}
	}
	panic(name)
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
func (b *builder) prepare(s source) error {
	archive := filepath.Join(b.root, filepath.Base(s.URL))
	if !verified(archive, s.SHA256) {
		urls := []string{s.URL}
		if s.Mirror != "" {
			urls = append(urls, s.Mirror)
		}
		var err error
		for attempt := 0; attempt < 3; attempt++ {
			for _, url := range urls {
				err = fetchSource(url, archive, s.SHA256)
				if err == nil {
					break
				}
				fmt.Fprintln(os.Stderr, "Source download:", err)
			}
			if err == nil {
				break
			}
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
		if err != nil {
			return err
		}
	}
	if _, err := os.Stat(b.source(s.Name)); err == nil {
		return nil
	}
	return b.run(b.src, "tar", "xf", archive)
}
func fetchSource(url, path, sum string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "music-dl-build")
	resp, err := (&http.Client{Timeout: 5 * time.Minute}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
	}
	f, err := os.Create(path + ".part")
	if err != nil {
		return err
	}
	defer os.Remove(path + ".part")
	_, err = io.Copy(f, io.LimitReader(resp.Body, 128<<20))
	ce := f.Close()
	if err != nil {
		return err
	}
	if ce != nil {
		return ce
	}
	if !verified(path+".part", sum) {
		return fmt.Errorf("source checksum mismatch for %s", url)
	}
	return os.Rename(path+".part", path)
}

func (b *builder) cmake(name string, extra ...string) error {
	dir := filepath.Join(b.work, name)
	if _, e := os.Stat(filepath.Join(dir, ".complete")); e == nil {
		return nil
	}
	args := []string{"cmake", "-S", b.source(name), "-B", dir, "-G", "Ninja", "-DCMAKE_INSTALL_PREFIX=" + b.prefix, "-DCMAKE_PREFIX_PATH=" + b.prefix, "-DCMAKE_BUILD_TYPE=MinSizeRel", "-DBUILD_SHARED_LIBS=OFF", "-DCMAKE_C_COMPILER=" + b.cc, "-DCMAKE_CXX_COMPILER=" + b.cxx}
	if b.system == "windows" {
		args = append(args, "-DCMAKE_SYSTEM_NAME=Windows", "-DCMAKE_RC_COMPILER=x86_64-w64-mingw32-windres")
	} else {
		args = append(args, "-DCMAKE_OSX_ARCHITECTURES="+b.arch, "-DCMAKE_OSX_DEPLOYMENT_TARGET=13.0")
	}
	args = append(args, extra...)
	for _, cmd := range [][]string{args, {"cmake", "--build", dir, "--parallel", b.jobs}, {"cmake", "--install", dir}} {
		if err := b.run(b.work, cmd...); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(dir, ".complete"), nil, 0644)
}
func (b *builder) dependencies(cflags string) error {
	if err := b.cmake("zlib", "-DZLIB_BUILD_EXAMPLES=OFF"); err != nil {
		return err
	}
	zlib := filepath.Join(b.prefix, "lib", "libz.a")
	if b.system == "windows" {
		if err := copyFile(filepath.Join(b.prefix, "lib", "libzlibstatic.a"), zlib); err != nil {
			return err
		}
	}
	pc := filepath.Join(b.prefix, "lib", "pkgconfig")
	if err := os.MkdirAll(pc, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(pc, "zlib.pc"), []byte("prefix="+b.prefix+"\nlibdir=${prefix}/lib\nincludedir=${prefix}/include\nName: zlib\nDescription: zlib\nVersion: 1.3.1\nLibs: -L${libdir} -lz\nCflags: -I${includedir}\n"), 0644); err != nil {
		return err
	}
	if err := b.cmake("freetype", "-DFT_DISABLE_HARFBUZZ=ON", "-DFT_DISABLE_PNG=ON", "-DFT_DISABLE_BZIP2=ON", "-DFT_DISABLE_BROTLI=ON", "-DFT_REQUIRE_ZLIB=ON", "-DZLIB_LIBRARY="+zlib, "-DZLIB_INCLUDE_DIR="+filepath.Join(b.prefix, "include")); err != nil {
		return err
	}
	if err := b.cmake("harfbuzz", "-DHB_HAVE_FREETYPE=ON", "-DHB_BUILD_SUBSET=OFF", "-DHB_BUILD_UTILS=OFF", "-DHB_HAVE_GLIB=OFF", "-DHB_HAVE_ICU=OFF", "-DHB_HAVE_DIRECTWRITE=OFF", "-DHB_HAVE_GDI=OFF", "-DHB_HAVE_UNISCRIBE=OFF"); err != nil {
		return err
	}
	pkg, err := exec.LookPath("pkg-config")
	if err != nil {
		return err
	}
	cross := fmt.Sprintf("[binaries]\nc = '%s'\ncpp = '%s'\nar = '%s'\nstrip = '%s'\npkg-config = '%s'\n[host_machine]\nsystem = '%s'\ncpu_family = '%s'\ncpu = '%s'\nendian = 'little'\n[properties]\nneeds_exe_wrapper = true\n", b.cc, b.cxx, b.ar, b.strip, pkg, b.system, b.arch, b.arch)
	if b.arch == "arm64" {
		cross = strings.ReplaceAll(cross, "cpu_family = 'arm64'", "cpu_family = 'aarch64'")
	}
	if b.system == "darwin" {
		cross += "[built-in options]\nc_args = ['-arch', '" + b.arch + "', '-mmacosx-version-min=13.0']\nc_link_args = ['-arch', '" + b.arch + "', '-mmacosx-version-min=13.0']\n"
	}
	crossPath := filepath.Join(b.work, "cross.ini")
	if err = os.WriteFile(crossPath, []byte(cross), 0644); err != nil {
		return err
	}
	dir := filepath.Join(b.work, "fribidi")
	if _, err = os.Stat(filepath.Join(dir, ".complete")); err != nil {
		for _, cmd := range [][]string{{"meson", "setup", dir, b.source("fribidi"), "--cross-file", crossPath, "--prefix", b.prefix, "--libdir", "lib", "--default-library", "static", "--buildtype", "minsize", "-Ddocs=false", "-Dtests=false", "-Dbin=false"}, {"meson", "compile", "-C", dir, "-j", b.jobs}, {"meson", "install", "-C", dir}} {
			if err = b.run(b.work, cmd...); err != nil {
				return err
			}
		}
		if err = os.WriteFile(filepath.Join(dir, ".complete"), nil, 0644); err != nil {
			return err
		}
	}
	dir = filepath.Join(b.work, "libass")
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if _, err = os.Stat(filepath.Join(dir, ".complete")); err == nil {
		return nil
	}
	args := []string{filepath.Join(b.source("libass"), "configure"), "--prefix=" + b.prefix, "--enable-static", "--disable-shared", "--disable-fontconfig", "--disable-asm"}
	if b.system == "windows" {
		args = append(args, "--host=x86_64-w64-mingw32")
	} else {
		args = append(args, "--host="+b.arch+"-apple-darwin")
	}
	for _, cmd := range [][]string{args, {"make", "-j" + b.jobs}, {"make", "install"}} {
		if err = b.run(dir, cmd...); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(dir, ".complete"), nil, 0644)
}
func (b *builder) ffmpeg() error {
	dir := filepath.Join(b.work, "ffmpeg")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if _, e := os.Stat(filepath.Join(dir, "config.h")); e != nil {
		args := []string{filepath.Join(b.source("ffmpeg"), "configure"), "--prefix=" + b.prefix, "--pkg-config=pkg-config", "--pkg-config-flags=--static", "--extra-cflags=-I" + filepath.Join(b.prefix, "include"), "--disable-everything", "--disable-autodetect", "--disable-debug", "--disable-doc", "--disable-ffplay", "--disable-x86asm", "--enable-small", "--enable-shared", "--disable-static", "--enable-network", "--enable-libass", "--enable-zlib",
			"--enable-protocol=file,http,https,tcp,tls,crypto,pipe,concat",
			"--enable-demuxer=mov,matroska,mp3,flac,ogg,wav,aac,ape,asf,concat,image2,image2pipe,image_png_pipe,image_jpeg_pipe,image_webp_pipe,ass",
			"--enable-muxer=mp3,flac,mp4,ipod,adts,ogg,wav,matroska,ffmetadata,null,image2,pcm_f32le",
			"--enable-decoder=aac,aac_fixed,mp3,mp3float,flac,alac,opus,vorbis,ape,wmav1,wmav2,wmapro,wmalossless,wmavoice,pcm_s16le,pcm_s24le,pcm_s32le,pcm_f32le,pcm_f64le,pcm_u8,pcm_s16be,pcm_s24be,mjpeg,png,webp,bmp,ass,wrapped_avframe",
			"--enable-encoder=aac,flac,pcm_s16le,pcm_f32le,mjpeg,png",
			"--enable-parser=aac,aac_latm,mpegaudio,flac,opus,vorbis,png,webp,mjpeg",
			"--enable-filter=aresample,aformat,anull,volume,atempo,scale,crop,pad,drawbox,subtitles,ass,color,format,anullsrc,null,trim,atrim,setpts,asetpts,concat,overlay,loop,sine",
			"--enable-bsf=aac_adtstoasc,extract_extradata,mp3_header_decompress", "--enable-indev=lavfi", "--enable-swscale", "--enable-swresample"}
		if b.system == "windows" {
			args = append(args, "--target-os=mingw32", "--arch=x86_64", "--enable-cross-compile", "--cross-prefix=x86_64-w64-mingw32-", "--enable-schannel", "--enable-mediafoundation", "--enable-d3d11va", "--enable-dxva2", "--enable-encoder=h264_mf", "--extra-ldflags=-L"+filepath.Join(b.prefix, "lib")+" -static-libgcc -static-libstdc++ -Wl,--gc-sections", "--extra-libs=-lstdc++ -lwinpthread")
		} else {
			args = append(args, "--target-os=darwin", "--arch="+b.arch, "--cc=clang", "--cxx=clang++", "--enable-cross-compile", "--enable-securetransport", "--enable-videotoolbox", "--enable-encoder=h264_videotoolbox", "--install-name-dir=@rpath", "--extra-cflags=-I"+filepath.Join(b.prefix, "include")+" -arch "+b.arch+" -mmacosx-version-min=13.0", "--extra-ldflags=-L"+filepath.Join(b.prefix, "lib")+" -arch "+b.arch+" -mmacosx-version-min=13.0 -Wl,-rpath,@executable_path", "--extra-libs=-lc++")
		}
		if err := b.run(dir, args...); err != nil {
			return err
		}
	}
	for _, cmd := range [][]string{{"make", "-j" + b.jobs}, {"make", "install"}} {
		if err := b.run(dir, cmd...); err != nil {
			return err
		}
	}
	return nil
}
func copyFile(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	o, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	_, err = io.Copy(o, f)
	ce := o.Close()
	if err != nil {
		return err
	}
	return ce
}
func (b *builder) bundle() error {
	dest := filepath.Join("resources", b.target)
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dest, "licenses"), 0755); err != nil {
		return err
	}
	bins := []string{"ffmpeg", "ffprobe"}
	if b.system == "windows" {
		bins = []string{"ffmpeg.exe", "ffprobe.exe"}
	}
	for _, name := range bins {
		if err := copyFile(filepath.Join(b.prefix, "bin", name), filepath.Join(dest, name)); err != nil {
			return err
		}
	}
	pattern := filepath.Join(b.prefix, "lib", "libav*.dylib")
	if b.system == "windows" {
		pattern = filepath.Join(b.prefix, "bin", "*.dll")
	}
	libs, _ := filepath.Glob(pattern)
	if b.system == "darwin" {
		more, _ := filepath.Glob(filepath.Join(b.prefix, "lib", "libsw*.dylib"))
		libs = append(libs, more...)
	}
	for _, path := range libs {
		name := filepath.Base(path)
		if b.system == "windows" && name == "libzlib.dll" {
			continue
		}
		if b.system == "darwin" { // Executables load libfoo.MAJOR.dylib; copy each payload just once.
			if len(strings.Split(name, ".")) != 3 {
				continue
			}
		}
		if err := copyFile(path, filepath.Join(dest, name)); err != nil {
			return err
		}
	}
	if b.system == "windows" {
		cmd := exec.Command(b.cc, "-print-file-name=libwinpthread-1.dll")
		data, err := cmd.Output()
		if err != nil {
			return err
		}
		if err = copyFile(strings.TrimSpace(string(data)), filepath.Join(dest, "libwinpthread-1.dll")); err != nil {
			return err
		}
	}
	for _, s := range sources {
		if err := copyFile(filepath.Join(b.source(s.Name), s.License), filepath.Join(dest, "licenses", s.Name+"-LICENSE.txt")); err != nil {
			return err
		}
	}
	if err := copyFile(filepath.Join(b.source("freetype"), "LICENSE.TXT"), filepath.Join(dest, "licenses", "freetype-NOTICE.txt")); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(sources, "", "  ")
	if err := os.WriteFile(filepath.Join(dest, "media-sources.json"), data, 0644); err != nil {
		return err
	}
	if b.system == "windows" {
		if err := copyFile("resources/licenses/MinGW-winpthreads.txt", filepath.Join(dest, "licenses", "MinGW-winpthreads.txt")); err != nil {
			return err
		}
	}
	var size int64
	filepath.WalkDir(dest, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			f, e := d.Info()
			if e == nil {
				size += f.Size()
			}
		}
		return err
	})
	fmt.Printf("Media resources: %.2f MiB (%s)\n", float64(size)/(1<<20), dest)
	return nil
}
