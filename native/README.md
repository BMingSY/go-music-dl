# MusicDL Native 验证版

独立的 MyGo 原生桌面验证版，用于在全面迁移前验证播放和系统窗口能力。主窗口与悬浮窗均使用 Go `ui.View`，没有 HTML/JavaScript、WebView 或浏览器音频。旧客户端暂时保留，后续迁移仍通过面向 `custom` 的 PR 完成。

## 当前能力

- 网易云音乐、Bilibili：搜索歌曲或解析单曲分享链接，使用已有 music-lib 的对应 Go 实现。
- 本地音乐：选择文件夹，扫描 MP3、FLAC、M4A、AAC、Ogg、WAV、WMA，读取标签、封面和同名 LRC。
- FFmpeg 流式解码，MyGo 原生音频输出；播放/暂停、音量、上一首/下一首、队列及已知时长音频的进度拖动。
- 收藏、Cookie 和设置存储在独立的本地配置目录。
- 可拖动、置顶的歌词悬浮窗；暂停或没有歌词时显示歌曲信息；悬停显示控制按钮。
- 关闭主窗口后收到托盘并显示悬浮窗，托盘菜单可恢复窗口、控制播放或退出；macOS 使用菜单栏图标。
- 系统媒体信息和媒体按键。
- 已接入 MyGo 的检查、下载、签名校验、安装和重启 API。此验证版尚未配置正式更新公钥和发布清单，因此明确提示应用内升级尚未启用；不跳转到上游发布页，也不假装更新成功。

## 运行

需要 Go 1.27.3，以及 FFmpeg 和 ffprobe。程序会查找 PATH、可执行文件旁、macOS App Resources、Homebrew 的常见位置，也可在设置中指定 FFmpeg 路径。FFmpeg 是独立的音频解码依赖，客户端代码为 Go。

```sh
cd native
go run .
```

支持 Windows 11 x64，以及 macOS Apple Silicon / Intel。项目使用独立的 go.mod，不更改旧项目的工具链。

```sh
# Windows 包
go run github.com/egoist/mygo/cmd/mygo@v0.7.4 build -platform windows/amd64
# macOS 包，在 Mac 上执行以完成签名和磁盘映像打包
go run github.com/egoist/mygo/cmd/mygo@v0.7.4 build -platform darwin/arm64
go run github.com/egoist/mygo/cmd/mygo@v0.7.4 build -platform darwin/amd64
```

PR 的 `Native preview` 工作流为三个目标执行测试并上传构建产物。验证包尚未捆绑 FFmpeg；使用者需要安装它或在设置中指定路径。

## 数据与测试

设置位于 `os.UserConfigDir()/music-dl-native-preview/settings.json`，不会读写旧客户端数据库。可通过 `MUSIC_DL_NATIVE_CONFIG` 指定测试配置路径。Cookie 不进入仓库、日志或网络测试数据。

```sh
go vet ./...
MUSIC_DL_REQUIRE_AUDIO_TESTS=1 go test -race -timeout=180s ./...
# FFmpeg 不在 PATH 时可指定 MUSIC_DL_TEST_FFMPEG 的绝对路径。

# 在内存中实际渲染 Go 视图，输出主窗口和悬浮窗 PNG。
MUSIC_DL_PREVIEW_DIR=artifacts/render go test -run TestRenderPreview -count=1 .

# 在 Windows / macOS 桌面会话中验证真实原生窗口和关闭后隐藏行为。
go run . --smoke-test artifacts/smoke
```

音频测试用 FFmpeg 合成测试音，不访问真实账号：覆盖 WAV、FLAC、AAC/M4A 解码及定位、带请求头的 HTTP 音频、网络读取中断后的进程回收。它们不等同于对所有平台资源的可用性保证，也不替代真机听音。

## 后续迁移范围

这是播放链路与窗口能力验证版，还不是旧客户端的完整替代：扫码登录、完整歌单/专辑浏览、下载管理、数据迁移和正式签名自更新发布需要后续 PR。旧的 Rust/WebView/移动端代码只会在迁移验收后移除。MyGo 原生界面符合去除 WebView 的目标；音频解码仍依赖 FFmpeg。
