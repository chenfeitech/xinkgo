# Xinkgo

Windows 桌面软件，Go 写成，界面由 [MyGo](https://github.com/egoist/mygo) 绘制。
当前是**可运行的基础骨架**：主窗口是原生 UI（无 webview），并预留了 Web 窗口
能力。

## 三步跑起来

```sh
go version          # 需要 1.27 或更高（低版本会自动下载 toolchain）

make run            # 从源码运行（首次编译分钟级）
make dev            # 或：live reload，改 .go 自动重建重启
```

跑起来会看到一个窗口：应用名与版本、一个输入框、一个计数器。菜单 `File` 里有
`New Window` 和 `Open Web Window`（Windows 上 Web 窗口需要 WebView2 Runtime，
Win11 自带）。

## 构建与打包

```sh
make build            # 当前机器 → bin/xinkgo
make build-windows    # 交叉编译 → bin/windows-amd64/xinkgo.exe
make package          # 打包 → build/windows-amd64/Xinkgo.exe（约 13 MB）
make check            # 提交前门禁：格式 + vet + 测试 + 全平台编译
```

不需要 cgo，不需要 C 编译器，不需要 Node/Bun：macOS 与 Linux 都能直接编出
Windows 可执行文件。只有生成 Windows **安装器**才需要 NSIS（在 Windows 上
`mygo build` 会自行下载）。

## 文档

| 文档 | 内容 |
|---|---|
| [docs/PROJECT.md](docs/PROJECT.md) | **项目结构、启动步骤、规范要点** |
| [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) | 代码风格、目录约定、提交信息、分支策略 |
| [docs/HARNESS.md](docs/HARNESS.md) | 构建 / 运行 / 测试 / 调试流程 |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | 分层、启动时序、线程模型、扩展点 |
| [docs/INDEX.md](docs/INDEX.md) | 文档索引 |
| [AGENTS.md](AGENTS.md) | AI / 新人的操作地图与技术铁律 |

## 技术事实（避免重复调研）

| 项 | 值 |
|---|---|
| 框架 | MyGo v0.7.3（`go.mod` 的 `tool` 固定，`go tool mygo` 运行 CLI） |
| Go | 1.27.3（go.mod 声明） |
| 界面 | 原生 UI：`mygo/ui`（MyGo 用 GPU 绘制），主窗口走 `ui.View` |
| 主目标 | Windows 10/11 amd64；arm64 亦可编译 |
| cgo | 禁用（`CGO_ENABLED=0`），可任意平台交叉编译 |
| 配置 | `config.json`，在用户数据目录（Windows：`%AppData%\Xinkgo`） |
| 日志 | 开发到终端；安装版到日志目录 `xinkgo.log` |

## 许可

未定（骨架阶段未声明，发布前补齐）。
