# Third-party notices

- go-music-dl / music-lib — https://github.com/guohuiyuan/go-music-dl and https://github.com/guohuiyuan/music-lib; AGPL-3.0. The original desktop design and application icon are retained.
- MyGo — https://github.com/egoist/mygo; MIT.
- Font Awesome Free — https://fontawesome.com; SIL OFL 1.1. Subsets are renamed MusicDL Icons and MusicDL Outline; the original glyph designs and license are retained. See `tools/prepare-assets.py`.
- FFmpeg 9.0.2 — https://ffmpeg.org; LGPL 2.1 or later. Built as shared libraries from the pinned sources and build recipe in `tools/bundle-ffmpeg/`. The separate libraries may be replaced by compatible builds.
- libass 0.17.4 — https://github.com/libass/libass; ISC.
- FreeType 2.13.3 — https://freetype.org; FreeType License.
- HarfBuzz 10.4.0 — https://harfbuzz.github.io; MIT-style license.
- FriBidi 1.0.16 — https://github.com/fribidi/fribidi; LGPL 2.1 or later.
- zlib 1.3.1 — https://zlib.net; zlib license.
- MinGW-w64 winpthreads (Windows) — https://www.mingw-w64.org; MIT and BSD licenses. See `licenses/MinGW-winpthreads.txt`.

Media source URLs and checksums are included in `media-sources.json` beside the executables; license texts are in `licenses/`.

Go module versions and checksums are recorded in `go.mod` and `go.sum`.
