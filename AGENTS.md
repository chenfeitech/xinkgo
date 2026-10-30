# AGENTS.md — Xinkgo 操作地图

> AI 与新人的第一入口。保持简短，指向深层文档。
> 与 `README.md` 冲突时，以本文档 + `docs/` 为准。

## 项目

Windows 桌面软件：Go + [MyGo](https://github.com/egoist/mygo) v0.7.3。
主窗口是**原生 UI**（`mygo/ui`，Go 绘制，无 webview），并预留 Web 窗口。
主目标 Windows amd64，macOS/Linux 用于开发自测。

## 开工前必读（违反会编译失败或架构腐化）

1. **禁用 cgo**：一切必须在 `CGO_ENABLED=0` 下编译。不引入需要 cgo 的依赖，
   跨平台差异用文件名后缀（`xxx_windows.go`）。
2. **依赖方向单向**：`cmd/xinkgo → internal/app → internal/{view,config,logging,sysinfo}`。
   `view`/`config`/`sysinfo` **不得** import `internal/app`。
3. **只有 `internal/app` 知道应用生命周期**：启动、退出、菜单、窗口、IPC 绑定
   都在这里；视图只画界面。
4. **应用目录只能在 `WhenReady` 里取**，且必须先 `mygo.App.SetName`（名字决定
   用户数据目录）。
5. **主线程**：MyGo 回调与窗口方法在主线程。视图函数每帧执行，不做阻塞 I/O。
6. **版本两处一致**：`mygo.json` 的 `version` 与 `internal/app/app.go` 的
   `const Version`，由 `scripts/*.sh` 校验。
7. **不得自行合并到 `main`**：在 `feature/*` 或 `bugfix/*` 分支上开发并提交，
   完成后报告分支名，等用户拍板再合并。

## 命令

```sh
make run            # 从源码运行
make dev            # live reload
make build          # 当前机器 → bin/
make build-windows  # 交叉编译 Windows
make package        # 打包（默认 windows/amd64）
make test           # go test ./...
make check          # 提交前门禁：全部
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
| 加配置项 | `internal/config`（字段 + 默认值 + 校验 + 测试） |
| 加 Web 窗口 / 前端 | `internal/app/web.go` 的绑定服务，`mygo generate` 生成 TS 客户端 |
| 加平台差异行为 | `internal/app/platform_<goos>.go` |
| 改版本号 | `mygo.json` + `internal/app/app.go` 两处 |
