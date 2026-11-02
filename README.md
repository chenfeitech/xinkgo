# Xinkgo

桌面软件，Go 写成，界面由 [MyGo](https://github.com/egoist/mygo) 绘制。
当前是**可运行的基础骨架**：主窗口是原生 UI（无 webview），并预留了 Web 窗口
能力。

**目标平台优先级**：P0 macOS（`darwin/arm64`、`darwin/amd64`，首要）→
P1 Windows/amd64 → P2 Windows/arm64、Linux/amd64（CI 冒烟）。构建顺序、
默认产物与门禁检查都以 macOS 为先。

## 三步跑起来

```sh
go version          # 需要 1.27 或更高（低版本会自动下载 toolchain）

make run            # 从源码运行（首次编译分钟级）
make dev            # 或：live reload，改 .go 自动重建重启
```

跑起来是一个笔记窗口：左边是库的文件树，中间是打开的笔记（标签页 + 编辑器），
右边是 Markdown 预览。菜单 `File` 里有 `Open Vault…`（Cmd-O）、`New Note`
（Cmd-N）、`Save`（Cmd-S）、`Close Tab`（Ctrl-W）。

第一次跑没有库：用 `Open Vault…` 选一个文件夹，或者像 xink 那样把库路径写进
`vaults.json`。

## 构建与打包

```sh
make deps             # 依赖就绪（构建与打包的前置）
make build            # 当前机器（macOS）→ bin/darwin-arm64/xinkgo
make package          # 打包当前机器 → build/darwin-arm64/Xinkgo.app + .dmg
make build-windows    # 交叉编译 → bin/windows-amd64/xinkgo.exe
make check            # 提交前门禁：格式 + vet + 测试 + 全平台编译（macOS 先）
```

不需要 cgo，不需要 C 编译器，不需要 Node/Bun：在 macOS 上能直接编出 Windows
与 Linux 的可执行文件。只有生成 Windows **安装器**才需要 NSIS（在 Windows 上
`mygo build` 会自行下载）；macOS 的 `.app` 与 `.dmg` 在 macOS 上产出。

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
| 框架 | MyGo v0.7.4（`go.mod` 的 `tool` 固定，`go tool mygo` 运行 CLI） |
| Go | 1.27.3（go.mod 声明） |
| 界面 | 原生 UI：`mygo/ui`（MyGo 用 GPU 绘制），主窗口走 `ui.View` |
| 目标平台 | P0 macOS（arm64、amd64）→ P1 Windows/amd64 → P2 Windows/arm64、Linux/amd64 |
| macOS 产物 | `build/darwin-arm64/Xinkgo.app`（12.4 MB）+ `xinkgo-0.1.0-arm64.dmg` |
| Windows 产物 | `build/windows-amd64/Xinkgo.exe`（13 MB，安装器需 NSIS） |
| cgo | 禁用（`CGO_ENABLED=0`），可任意平台交叉编译 |
| 配置 | `config.json`，在用户数据目录（Windows：`%AppData%\Xinkgo`） |
| 日志 | 开发到终端；安装版到日志目录 `xinkgo.log` |

## 许可

未定（骨架阶段未声明，发布前补齐）。
