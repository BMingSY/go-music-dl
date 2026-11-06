# Development workflow

- This is BMingSY's fork. `main` is the upstream baseline; `custom` is the integration branch for the customized application.
- Make every repository change on a feature branch. Open a PR targeting `custom`, run the relevant checks, review the diff, and merge through GitHub. Never push implementation commits directly to `main` or `custom`.
- The first migration stage lives in `native/`, a separate Go module pinned to MyGo v0.7.4. It must create only native UI windows, with no WebView, HTML frontend, or browser audio dependency.
- Supported desktop targets are Windows 11 amd64 and macOS arm64/amd64. The native app exposes only `netease`, `bilibili`, and `local` sources.
- Keep legacy code intact until the native validation stage is complete and a later PR explicitly migrates/removes it.
- In `native/`, run `go test ./...` and `go vet ./...`. With FFmpeg available, set `MUSIC_DL_REQUIRE_AUDIO_TESTS=1` to require codec/seek/cancellation tests. Run `go test -race ./...` on supported CI hosts. UI and playback changes also require the appropriate native-window smoke checks.
- Never commit cookies, user configuration, signing private keys, downloaded audio, or generated binaries. Describe preview limitations and distinguish headless tests, real-window smoke tests, and manual listening.
