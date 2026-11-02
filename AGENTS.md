# AGENTS.md — Xinkgo 操作地图

> AI 与新人的第一入口。保持简短，指向深层文档。
> 与 `README.md` 冲突时，以本文档 + `docs/` 为准。

## 项目

桌面软件：Go + [MyGo](https://github.com/egoist/mygo) v0.7.4。
主窗口是**原生 UI**（`mygo/ui`，Go 绘制，无 webview），并预留 Web 窗口。

**目标平台优先级**：P0 macOS（`darwin/arm64`、`darwin/amd64`，首要，构建/依赖/
产物都先保证它）→ P1 Windows/amd64 → P2 Windows/arm64、Linux/amd64。

## 开工前必读（违反会编译失败或架构腐化）

1. **禁用 cgo**：一切必须在 `CGO_ENABLED=0` 下编译。不引入需要 cgo 的依赖，
   跨平台差异用文件名后缀（`xxx_windows.go`）。
2. **依赖方向单向**：`cmd/xinkgo → internal/app → internal/{view,config,logging,sysinfo}`。
   `view`/`config`/`sysinfo` **不得** import `internal/app`。
3. **只有 `internal/app` 知道应用生命周期**：启动、退出、菜单、窗口、IPC 绑定
   都在这里；视图只画界面。
4. **应用目录只能在 `WhenReady` 里取**，且必须先 `mygo.App.SetName`（名字决定
   用户数据目录）。
5. **界面文案进 catalog**：用户看得见的字不写死在 Go 里，写
   `internal/locale/en.json`（源语言）与各语言的 JSON，用生成的 `locale.Xxx()`
   取；切语言时菜单靠 `OnLocaleChanged` 重建，界面靠每帧重画自动生效。
6. **主线程**：MyGo 回调与窗口方法在主线程。视图函数每帧执行，不做阻塞 I/O。
7. **版本两处一致**：`mygo.json` 的 `version` 与 `internal/app/app.go` 的
   `const Version`，由 `scripts/*.sh` 校验。
8. **不得自行合并到 `main`**：在 `feature/*` 或 `bugfix/*` 分支上开发并提交，
   完成后报告分支名，等用户拍板再合并。
9. **macOS 优先**：改构建、依赖或打包配置时，先保证 macOS（P0）的构建、依赖
   与产物无误，再验证 Windows（P1）与 Linux（P2）。

## 命令

```sh
make deps           # 依赖就绪（下面的构建与打包都会先跑它）
make run            # 从源码运行
make dev            # live reload
make build          # 当前机器（macOS）→ bin/darwin-arm64/
make build-macos    # macOS 两个架构
make build-windows  # 交叉编译 Windows
make package        # 打包当前机器 + Windows/amd64（Mac 上即 macOS：.app + .dmg，以及 Xinkgo.exe）
make package-macos  # macOS 两个架构
make package-windows # Windows 两个架构
make test           # go test ./...
make check          # 提交前门禁：全部，macOS 先编译
make doctor         # 环境自检
```

无 make 时（Windows）：`scripts/check.sh`（Git Bash）或 Makefile 打印的等价
`go` 命令。

## 深层文档

| 文档 | 用途 |
|---|---|
| [docs/PROJECT.md](docs/PROJECT.md) | 项目结构、启动步骤、规范要点 |
| [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) | 代码风格、目录约定、提交信息、分支策略 |
| [docs/HARNESS.md](docs/HARNESS.md) | 构建 / 运行 / 测试 / 调试、故障排查 |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | 分层、启动时序、线程模型、扩展点 |
| [docs/INDEX.md](docs/INDEX.md) | 文档索引 |

## 常见任务落在哪

| 要做的事 | 改哪 |
|---|---|
| 加一个窗口或界面 | `internal/view` 加视图；`internal/app` 加打开入口与菜单项 |
| 加界面文案 / 加一门语言 | `internal/locale`：en.json 加键（+ zh-Hans.json），`go tool mygo generate`，代码调 `locale.Xxx()`；见 [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) 第 6 节 |
| 加配置项 | `internal/config`（字段 + 默认值 + 校验 + 测试） |
| 加 Web 窗口 / 前端 | `internal/app/web.go` 的绑定服务，`mygo generate` 生成 TS 客户端 |
| 加平台差异行为 | `internal/app/platform_<goos>.go` |
| 改版本号 | `mygo.json` + `internal/app/app.go` 两处 |
