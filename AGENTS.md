# Development

- Put all requested migration changes in PR #1 targeting `custom`. Do not merge partial stages or push implementation commits directly to `main` or `custom`.
- Go with MyGo native UI only. Support Windows 11 x64 and macOS Intel/Apple Silicon; sources are netease, bilibili and local.
- Preserve the upstream desktop layout and interactions when removing unsupported sources. Match the reference mini player for the floating window.
- Run `go vet ./...` and `MUSIC_DL_REQUIRE_AUDIO_TESTS=1 go test -race ./...` with FFmpeg. Verify UI changes with renders and supported-platform smoke tests.
- Never commit cookies, user settings, signing private keys, downloaded music or build outputs.
