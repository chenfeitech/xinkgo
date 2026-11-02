# Xinkgo 项目说明

> 本文是 Xinkgo 的入口说明：**项目结构、启动步骤、规范要点**。
> 细节规范在 [CONTRIBUTING.md](CONTRIBUTING.md)（代码风格 / 目录约定 / 提交
> / 分支），操作流程在 [HARNESS.md](HARNESS.md)（构建 / 运行 / 测试 / 调试）。

## 1 项目是什么

桌面软件，Go 写成，界面由 [MyGo](https://github.com/egoist/mygo) v0.7.4
绘制。**macOS 是首要目标平台**，Windows 与 Linux 受支持但优先级在其后。

| 项 | 值 |
|---|---|
| 目标平台（优先级） | **P0 macOS**（`darwin/arm64` 开发机首要，`darwin/amd64` 同优先级）→ P1 `windows/amd64` → P2 `windows/arm64`、`linux/amd64`（CI 冒烟） |
| 界面形态 | **原生 UI**（`mygo/ui`，MyGo 自己用 GPU 画，无 HTML/JS/webview）；同时**预留 Web 窗口**能力 |
| 语言 / 框架 | Go 1.27（模块声明 1.27.3）+ MyGo v0.7.4 |
| cgo | **禁用**（`CGO_ENABLED=0`），因此任意机器都能交叉编译到其它平台 |
| 前端工具链 | 无（原生 UI 不需要 Bun/Node；未来接入 Web 窗口时再引入） |
| 应用名 / 标识符 | `Xinkgo` / `com.helight.xinkgo` |
| 版本 | 0.1.0（`mygo.json` 与 `internal/app/app.go` 两处一致，脚本校验） |

优先级的落点（改动平台相关配置时一并检查这四处）：

| 位置 | 体现 |
|---|---|
| `Makefile` | `MACOS_PLATFORMS` 在前；`build` / `package` 默认当前机器（Mac 上即 macOS）；`deps` 是构建与打包的前置 |
| `scripts/check.sh` | 编译顺序：本机平台 → 其余 macOS → Windows → Linux |
| `scripts/package.sh` | 不带参数时打包当前机器（Mac 上即 macOS） |
| `docs/HARNESS.md` | 平台矩阵表与 CI 主产物 |

为什么这样选：

- **macOS 优先**：开发、构建、打包、冒烟都在 macOS 上完成，构建流程、依赖处理
  与产物输出（`.app` + `.dmg`）最先得到保证；问题在这台机器上最早暴露。
- **原生 UI 打底**：单二进制、无 webview 进程、内存与启动开销小，在 macOS 上
  不依赖任何额外运行时。
- **保留 Web 窗口**：主窗口是原生 UI，但应用已通过 `mygo.Bind` 暴露了 Go 服务
  （`system.Info`），随时可以开一个走系统 webview 的窗口、接入前端；
  `internal/app/web.go` 就是这个接缝。
- **纯 Go 无 cgo**：从 macOS 交叉编译 Windows 与 Linux 产物，CI 不需要对应机器
  也能出可执行文件（macOS 的 `.dmg` 与 Windows 的安装器仍需在各自系统上产出）。

## 2 项目结构

```
xinkgo/
├── Makefile                 开发入口：构建 / 运行 / 测试 / 调试 / 打包
├── go.mod  go.sum           Go 模块；mygo 与 CLI 版本在此固定
├── mygo.json                应用名、标识符、版本、入口包、打包配置
├── README.md                仓库首页（指向本文）
├── AGENTS.md                AI / 新人的操作地图
├── cmd/
│   └── xinkgo/main.go       入口：只做装配，不含逻辑
├── internal/
│   ├── app/                 应用生命周期：命名、启动、菜单、窗口、IPC 绑定
│   │   ├── app.go               Run/start：装配与初始化
│   │   ├── window.go            主窗口（原生 UI）
│   │   ├── menu.go              菜单栏
│   │   ├── web.go               Web 窗口预留 + 供页面调用的 Go 服务
│   │   └── platform_darwin.go / platform_other.go   平台差异（退出行为）
│   ├── view/                窗口界面：由状态绘制（home.go + home_test.go）
│   ├── config/              用户配置：读 / 写 / 校验（config.json）
│   ├── export/              导出：把笔记写成自成一页的 HTML（PDF 由它打印）
│   ├── theme/               配色方案表：每套方案明暗两套取色（纯数据 + ui）
│   ├── logging/             slog 初始化（开发到终端，安装版到文件）
│   └── sysinfo/             版本与运行环境信息（纯数据）
├── assets/                  不随产物发布的素材：应用图标的 .icns 与它的 PNG
│   ├── xink.icns            macOS 应用图标（打包时原样写进 .app）
│   └── icon.png             同一图标的 PNG：Windows 图标资源与 Linux
│                            桌面项图标由它生成（scripts/icon.sh 从 .icns 导出）
├── resources/               随应用发布的资源：整目录进产物，由应用运行时读取
├── scripts/
│   ├── check.sh             提交前门禁
│   ├── icon.sh              icon.png 从 xink.icns 导出
│   └── package.sh           打包（含版本一致性校验）
└── docs/
    ├── PROJECT.md           本文：结构 / 启动 / 规范要点
    ├── CONTRIBUTING.md      代码风格 / 目录约定 / 提交 / 分支
    ├── HARNESS.md           构建 / 运行 / 测试 / 调试
    ├── ARCHITECTURE.md      分层与数据流
    └── INDEX.md             文档索引
```

**分层与依赖方向**（只能单向，不得反向或循环）：

```
cmd/xinkgo  →  internal/app  →  internal/{view, config, logging, sysinfo,
                                            export, theme}
                                   view   → sysinfo, theme
                                   logging → config
                                   config  → theme
                                   export  → theme
```

- `internal/view`、`config`、`sysinfo`、`export` **不 import `internal/app`**：它们必须在
  没有运行中的应用时可测。
- `internal/app` 是唯一知道应用生命周期的包（启动、退出、窗口、菜单、IPC）。
- 平台差异用文件名后缀（`xxx_windows.go`），不用运行时 `switch`。

新增代码前先回答"它属于哪一层"；完整放置规则见
[CONTRIBUTING.md](CONTRIBUTING.md) 第 2 节。

## 3 启动步骤

### 3.1 前置条件

- Go 1.27 或更高（低版本会自动下载 1.27.3 toolchain；`Makefile` 已导出
  `GOTOOLCHAIN=auto`）。
- MyGo CLI 不用单独装：`go.mod` 的 `tool` 指令固定了 v0.7.4，`go tool mygo`
  直接可用。
- 打包安装器才需要 NSIS（可选，见下）。

```sh
go version          # go1.27.x
go tool mygo version   # mygo 0.7.4
```

### 3.2 三步跑起来

```sh
git clone <仓库地址> && cd xinkgo

make run          # 从源码运行；等首次编译（分钟级），之后秒级
```

开发时更常用（改 `.go` 文件自动重建重启）：

```sh
make dev          # go tool mygo dev，live reload
```

启动成功后终端（开发模式日志到终端）会看到：

```
level=INFO msg=starting app="Xinkgo 0.1.0 (darwin/arm64)" data="..." development=true
```

窗口是三栏：左边库的文件树，中间打开的笔记（标签页 + 编辑器，300ms 防抖自动
保存），右边 Markdown 预览。菜单 `File` 下有 `Open Vault…`（Cmd-O）、
`New Note`（Cmd-N）、`Save`（Cmd-S）、`Close Tab`（Ctrl-W）。

第一次启动没有库：用 `Open Vault…` 选文件夹；库与会话记在用户数据目录里
（`vaults.json` / `open_notes.json`），与 xink 同构。

### 3.3 构建与打包（macOS 优先）

```sh
make deps                  # 依赖就绪（构建与打包的前置步骤）
make build                 # 当前机器：bin/darwin-arm64/xinkgo
make build-macos           # macOS 两个架构
make package               # 默认当前机器：build/darwin-arm64/ 下的 .app 与 .dmg
make build-windows         # 交叉编译：bin/windows-amd64/xinkgo.exe
PLATFORMS="windows/amd64" make package   # 次要平台，显式指定
```

macOS 打包输出（已在本机验证）：

```
  MyGo!!!!! v0.7.4  build
  ○ Compiling darwin/arm64
  ✓ Compiled darwin/arm64  4.6s
  ○ Signing Xinkgo.app
  ✓ Signed Xinkgo.app  30ms
  ○ Creating xinkgo-0.1.0-arm64.dmg
  ✓ Created xinkgo-0.1.0-arm64.dmg  4.8s
  ✓ Built Xinkgo 0.1.0 in 9.8s
    build/darwin-arm64/
      Xinkgo.app                12.4 MB
      xinkgo-0.1.0-arm64.dmg     3.9 MB
```

Windows 交叉打包同样可用（`Xinkgo.exe` 13.0 MB）；缺少 NSIS 只跳过安装器，
`.exe` 照常产出，需要安装器就在 Windows 上打包（`mygo build` 会自行下载 NSIS）
或本机安装 `makensis`。

### 3.4 提交前

```sh
make check        # gofmt + go vet + go test + 全平台编译（macOS 先）+ 版本一致性
```

## 4 规范要点

### 4.1 代码风格

- 格式只认 `gofmt`；`go vet` 无告警；两者是硬性门禁。
- 命名：包名单个小写词（`config`、`view`），导出用 `MixedCaps`，缩写全大写
  （`ID`、`OS`），测试 `Test<类型>_<行为>`，getter 不加 `Get`。
- 注释：导出标识符必须有、以标识符名开头、解释**为什么**；不写代码已经说明的
  内容。界面文案进 `internal/locale` 的 catalog（英文是源语言），日志用英文。
- 错误：向上抛带 `%w` 上下文；不吞、不 `panic`；库代码不 `log.Fatal`（只有
  `cmd/xinkgo/main.go` 可以）；日志统一 `log/slog`。
- 并发：视图与 MyGo 回调跑在主线程，视图里不做阻塞 I/O；后台 goroutine 改界面
  状态要回到主线程。
- 不引入 cgo 依赖。

### 4.2 目录约定

- `cmd/xinkgo` 只有 `main`；`internal/app` 管生命周期；`internal/view` 画界面；
  `internal/config` 管持久化；纯数据另起小包（如 `internal/sysinfo`）；静态资源
  进 `resources/`；平台差异用 `_windows.go` 这类文件后缀。
- 新增 `internal/` 包必须带同目录 `*_test.go`，并同步更新本文件的目录树与
  [CONTRIBUTING.md](CONTRIBUTING.md) 的放置规则表。

### 4.3 提交信息

Conventional Commits，标题英文祈使式、≤ 50 字符：

```
feat(view): add a web window button

- opens a webview window from the main window
- binds system.Info for pages

Closes #12
```

type：`feat`/`fix`/`refactor`/`perf`/`test`/`docs`/`build`/`chore`；scope 取包名
（`app`、`view`、`config`…）。一次提交 = 一个任务，提交前 `make check` 全绿。
**禁止**在提交信息里出现 `[skip ci]`（会跳过 CI）。

### 4.4 分支策略

| 分支 | 从哪切 | 用途 |
|---|---|---|
| `main` | — | 随时可发布；受保护，禁止直接 push |
| `feature/<slug>` | `main` | 新功能 |
| `bugfix/<slug>` | `main` | 缺陷修复 |
| `release/<version>` | `main` | 发版准备 |
| `hotfix/<version>` | tag | 线上紧急修复 |

- `slug` 小写连字符；一个分支一件事；同步用 rebase 不用 merge，保持线性。
- 合并走 PR，squash merge，合并后删分支。
- **未经明确确认不得把分支合并到 `main`、也不得直接 push `main`**：在分支上
  完成提交后报告分支名，等人拍板。

### 4.5 测试与门禁

- `make test`（`make test-race` 至少在合入前跑一次）。
- 视图测试用 `ui.NewTester`（无窗口点击 / 输入）；目录用 `t.TempDir()`，不写
  真实用户目录；**不依赖 `time.Sleep`**。
- 门禁 `make check`：格式 + vet + 测试 + 五平台编译（**macOS 先**）+ 版本一致性，
  全绿才可提交。`make test` 与门禁默认针对当前机器（macOS）。

### 4.6 版本

`mygo.json` 的 `version` 与 `internal/app/app.go` 的 `const Version` 必须一致
（`scripts/check.sh`、`scripts/package.sh` 校验）。`identifier` 发布后不得更改。

## 5 骨架已包含的能力

- 应用启动、主窗口（原生 UI）、菜单栏（File / Edit / View / Window）。
- 用户配置读写（`config.json`，原子替换，损坏时降级为默认值并告警）。
- 结构化日志（开发到终端、安装版到日志文件，级别可配）。
- 跨平台退出行为（Windows/Linux 关最后一个窗口即退出；macOS 留在 Dock）。
- Web 窗口预留：`mygo.Bind(system{...})` + `openWebWindow()`，页面可调
  `system.Info`。
- 测试示例：`config`（7 个用例）、`view`（5 个用例）、`logging`。
- 门禁脚本与打包脚本。

## 6 下一步（骨架之后）

1. 接入真实业务视图：在 `internal/view` 加窗口，在 `internal/app` 注册菜单。
2. 需要前端时：加 `mygo.config.ts` 与 `src/`，用 `mygo generate` 生成
   `system.Info` 的类型化 TS 客户端，Web 窗口从 `LoadHTML` 换成前端页面。
3. 打包分发：macOS 用 Developer ID 签名并公证（`mygo.json` 的
   `macos.signingIdentity`）；Windows 用 NSIS 安装器（Windows 机器或 CI
   `windows-latest`）与代码签名证书。
4. 自动更新：`mygo keygen` + `mygo.json` 的 `updates`。
5. 加一门语言：`internal/locale/<tag>.json`（键集与 `en.json` 一致）+
   `config.BasicConfig` 的语言枚举 + 设置里的选项，`go tool mygo generate`
   生成取文案的函数与 MyGo 自身文案的 import。
6. CI：`.github/workflows/ci.yml`，macOS runner 出主产物
   （`scripts/package.sh darwin/arm64`），Windows runner 出 `.exe` 与安装器
   （模板见 [HARNESS.md](HARNESS.md) 第 8 节）。
