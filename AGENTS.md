# Development

- Use a topic branch and one PR targeting `custom` for related changes. Do not push implementation commits directly to `main` or `custom`.
- Go with MyGo native UI only. Support Windows 11 x64 and macOS Intel/Apple Silicon; sources are netease, bilibili and local.
- Keep the entry point in `cmd/music-dl`, desktop code and its tests in `internal/desktop`, and embedded assets in `internal/assets`. Keep library, download, lyrics and playback packages independent of desktop UI.
- Preserve the upstream desktop layout and interactions when removing unsupported sources. Match the reference mini player for the floating window.
- Run `go vet ./...` and `MUSIC_DL_REQUIRE_AUDIO_TESTS=1 go test -race ./...` with FFmpeg. Render UI with `go test -run TestRenderPreview ./internal/desktop` and verify supported-platform smoke tests.
- Never commit cookies, user settings, signing private keys, downloaded music or build outputs.
