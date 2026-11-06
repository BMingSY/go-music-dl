# music-dl

音乐搜索、播放与下载，支持网易云音乐、Bilibili 和本地音乐。

## 安装

从 [Releases](https://github.com/BMingSY/go-music-dl/releases) 下载 Windows 11 x64 或 macOS 13+ 安装包。macOS 分为 Apple Silicon 和 Intel 两个版本。

- 支持单曲、歌单、专辑搜索和批量下载。
- 支持本地歌单、收藏、播放历史和歌词显示。
- 关闭主窗口后继续在托盘运行，悬浮窗可显示歌词、控制播放和音量。
- 在「设置 → 关于」检查更新，下载完成后重启安装。

## 开发

需要 Go 1.27.3。开发时需安装 FFmpeg 和 ffprobe，或在设置中指定路径。

```sh
go run .
```

构建安装包：

```sh
go run ./tools/bundle-ffmpeg
go run github.com/egoist/mygo/cmd/mygo@v0.7.4 build
```

测试：

```sh
go vet ./...
MUSIC_DL_REQUIRE_AUDIO_TESTS=1 go test -race ./...
```

基于 [go-music-dl](https://github.com/guohuiyuan/go-music-dl) 和 [music-lib](https://github.com/guohuiyuan/music-lib)，界面使用 [MyGo](https://github.com/egoist/mygo)。遵循 [AGPL-3.0](LICENSE) 许可。
