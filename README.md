# music-dl

基于 [go-music-dl](https://github.com/guohuiyuan/go-music-dl) 进行精简和改造。

## 新增与调整

- 使用 Go + MyGo 原生桌面界面，替换 WebView。
- 更新检查支持直接下载安装，完成后重启更新。
- 增加悬浮窗。

## 功能删减

- 仅保留 Go 桌面端实现。
- 仅支持 Windows 11 x64 和 macOS。
- 音乐源仅保留网易云音乐、Bilibili 和本地音乐。
- 移除歌单分类和桌面挂件。

## 安装

从 [Releases](https://github.com/BMingSY/go-music-dl/releases) 下载 Windows 11 x64 或 macOS 13+ 安装包。macOS 分为 Apple Silicon 和 Intel 两个版本。

## 开发

`cmd/music-dl` 为启动入口，`internal/desktop` 包含桌面界面与交互，`internal/assets` 保存图标和字体。音乐源、下载、歌词和解码分别位于 `internal/library`、`internal/download`、`internal/lyrics` 和 `internal/playback`。

需要 Go 1.27.3。开发时需安装 FFmpeg 和 ffprobe，或在设置中指定路径。

```sh
go run ./cmd/music-dl
```

构建安装包前，按[媒体工具构建说明](tools/bundle-ffmpeg/README.md)准备对应平台的资源。

```sh
go run github.com/egoist/mygo/cmd/mygo@v0.7.4 build
```

测试：

```sh
go vet ./...
MUSIC_DL_REQUIRE_AUDIO_TESTS=1 go test -race ./...
```

GitHub Actions 会在向 `custom` 提交 PR 或推送合并结果时检查工作流和 Go 格式，并分别在 Windows x64、macOS Intel 和 Apple Silicon 上运行 `go vet`、带 FFmpeg 的竞态测试、更新安装冒烟、界面渲染和原生窗口冒烟。界面截图与冒烟结果保留为 CI artifacts。CI 不依赖音频输出设备，音频解码由 FFmpeg 集成测试验证。

macOS CI 使用项目自身的 FFmpeg 构建配方并缓存媒体工具，确保歌词字幕和视频编码功能齐全；Windows 使用支持这些功能的 FFmpeg。测试开始前会检查所需功能。

PR/MR 标题须使用 `feat:`、`fix:`、`ci:` 等 Conventional Commits 前缀，修改标题时会重新校验。安装包构建和发布流程独立保留，仅在推送 `native-v*` 标签或手动运行 **Desktop packages** 时触发；日常测试也可以手动运行 **CI**。

如果 fork 的 Actions 页面显示 **Workflows on this fork have been disabled**，需在该页面点击启用按钮解除 GitHub 对 fork 工作流的暂停。仓库权限显示已启用或单独启用某个工作流，并不代表这个暂停状态已经解除。

基于 [go-music-dl](https://github.com/guohuiyuan/go-music-dl) 和 [music-lib](https://github.com/guohuiyuan/music-lib)，界面使用 [MyGo](https://github.com/egoist/mygo)。遵循 [AGPL-3.0](LICENSE) 许可。
