# Media tools

The package uses a small shared FFmpeg build for decoding, metadata and video export. H.264 encoding uses Windows Media Foundation or macOS VideoToolbox. The build keeps the audio formats supported by the app, HTTP/TLS, image covers and libass subtitles; it does not include an unrelated video player or a full codec distribution.

Requirements: CMake, Ninja, Meson, pkg-config, GNU make and tar. On Linux, install the MinGW x86_64 C/C++ toolchain for Windows builds. On macOS, install Xcode command line tools; either Mac architecture can be built from an Intel or Apple Silicon host.

```sh
go run ./tools/bundle-ffmpeg -platform windows-amd64 # Linux host
go run ./tools/bundle-ffmpeg -platform darwin-arm64  # macOS host
go run ./tools/bundle-ffmpeg -platform darwin-amd64  # macOS host
```

Source downloads are pinned by SHA-256 in `main.go`. Sources and compiled dependencies are cached under the OS cache directory, and the output is copied to `resources/<platform>/`. The cache key includes the build recipe. CI builds these resources before packaging; Windows developers can use the CI media artifact or their own FFmpeg for development.

The generated `media-sources.json` contains the corresponding source archives. No upstream source is patched. Library licenses and notices ship with each package.
