# Development

- Do not modify `README.md` unless the user explicitly requests changes to it.
- Use a topic branch and one PR targeting `custom` for related changes. Do not push implementation commits directly to `main` or `custom`.
- All PR/MR titles must use a Conventional Commits prefix: `feat:`, `fix:`, `docs:`, `style:`, `refactor:`, `perf:`, `test:`, `build:`, `ci:`, `chore:`, or `revert:`. An optional scope and breaking-change marker are allowed, e.g. `fix(playback): resume interrupted audio` or `feat!: change the library format`. Follow the prefix with a space and a non-empty summary; keep the title aligned with the final scope of the change.
- Go with MyGo native UI only. Support Windows 11 x64 and macOS Apple Silicon; sources are netease, bilibili and local.
- Keep the entry point in `cmd/music-dl`, desktop code and its tests in `internal/desktop`, and embedded assets in `internal/assets`. Keep library, download, lyrics and playback packages independent of desktop UI.
- Preserve the upstream desktop layout and interactions when removing unsupported sources. Match the reference mini player for the floating window.
- Run `go vet ./...` and `MUSIC_DL_REQUIRE_AUDIO_TESTS=1 go test -race ./...` with FFmpeg. Render UI with `go test -run TestRenderPreview ./internal/desktop` and verify supported-platform smoke tests.
- Never commit cookies, user settings, signing private keys, downloaded music or build outputs.
