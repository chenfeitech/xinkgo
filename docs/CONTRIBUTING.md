# 开发规范

Xinkgo 是 Windows 桌面软件，用 Go 写，界面由 [MyGo](https://github.com/egoist/mygo)
绘制。本文是写代码、提交、开分支时遵守的约定。

- 工具与流程：[HARNESS.md](HARNESS.md)
- 项目结构与启动步骤：[PROJECT.md](PROJECT.md)
- 分层与数据流：[ARCHITECTURE.md](ARCHITECTURE.md)

## 1 代码风格

### 1.1 硬性门禁

提交前 `make check` 必须全绿，其中格式与静态检查是硬性的：

| 检查 | 命令 | 要求 |
|---|---|---|
| 格式 | `make fmt-check`（`gofmt -l .`） | 无输出。格式化只认 `gofmt`，不引入 `goimports` 之外的风格差异 |
| 静态检查 | `make vet`（`go vet ./...`） | 无告警 |
| 测试 | `make test` | 全绿 |
| 跨平台编译 | `scripts/check.sh` | Windows amd64/arm64、macOS、Linux 都通过 |

`golangci-lint` 是可选的加分项（`make lint`），未安装不影响门禁。

不引入需要 cgo 的依赖：整个项目必须能在 `CGO_ENABLED=0` 下从任意机器交叉
编译到其它平台。这是 MyGo 的底座，破坏它等于放弃跨平台构建。

**平台优先级**：macOS 是首要目标（P0），Windows/amd64 次之（P1），
Windows/arm64 与 Linux/amd64 为 P2。体现为：`make build` / `make package`
默认当前机器（Mac 上即 macOS）、`make deps` 是构建与打包的前置步骤、
`scripts/check.sh` 的编译顺序为本机 → 其余 macOS → Windows → Linux。
改动构建、依赖或打包配置时，先让 macOS 全绿，再看其它平台。

### 1.2 命名

- 包名：单个小写单词，不用下划线、不用复数（`config`、`view`、`logging`）。
  包名是调用处的前缀，`config.Load` 比 `configs.LoadConfig` 好。
- 导出标识符用 `MixedCaps`，非导出用 `mixedCaps`；不出现下划线。
- 首字母缩写词统一大写：`HTTPClient`、`ID`、`OS`。
- 接口：单方法接口以 `-er` 命名（`Reader`、`Closer`）；接口在**使用方**定义，
  不在实现方。
- 错误值：`err` 变量、`ErrXxx` 导出哨兵、`XxxError` 类型。
- 测试：`Test<类型>_<行为>`，例如 `TestHomeView_counts`、`TestSaveLoad_roundTrip`。
- getter 不加 `Get` 前缀（Go 惯例）：`h.Count()`，不是 `h.GetCount()`。

### 1.3 注释

- 每个导出标识符（`package`、类型、函数、常量、变量）都有文档注释，以该
  标识符名开头：

  ```go
  // Config is the configuration the user can change.
  type Config struct { ... }

  // Load reads the configuration from dir, returning Default when there is
  // no file yet.
  func Load(dir string) (Config, error) { ... }
  ```

- 注释解释**为什么**，不重复代码已经说的**是什么**。循环里的一行注释若只是
  把代码念一遍，删掉。
- 包注释放在包内一个文件里（`config.go`），说明这个包的职责与边界；包内其他
  文件不重复写包注释。
- 复杂算法、平台差异、绕过某处的行为，必须写清原因与出处；临时方案写
  `// TODO(姓名): 原因与预期移除时间`，并在提交说明里提一句。
- 界面文案与日志用英文（与 MyGo 生态一致，便于 i18n 目录接入）；面向用户的
  中文界面在 `locales/` 接入后再定，不要硬编码中文到视图里。

### 1.4 错误处理

- 错误往上抛时带上下文：`fmt.Errorf("read configuration: %w", err)`。
- 不吞掉错误；确实要忽略时写清原因：`_ = f.Close() // 只读文件，关闭无意义`。
- 不在库代码里 `panic`、`log.Fatal`、`os.Exit`：只有 `cmd/xinkgo/main.go` 可以
  `log.Fatal`。
- 日志用 `log/slog`（`slog.Info`、`slog.Warn`、`slog.Error`），不用 `fmt.Print`
  和裸 `log`。
- 用户能补救的错误（配置写坏了）不要中断启动：降级到默认值并记一条
  `Warn`（见 `internal/app/app.go::start`）。

### 1.5 导入与排版

- 导入分三组，组间空一行：标准库 / 第三方 / 本模块（`github.com/helightxu/xinkgo/...`）。
- 行宽建议 100 字符以内（`gofmt` 不断行，超出时手动换行，续行缩进与括号对齐）。
- 一个函数只做一件事；超过一屏就拆。
- 视图函数（`internal/view`）只做"由状态画界面 + 改状态"，不做 I/O、不调
  用应用生命周期接口。

### 1.6 并发与主线程

MyGo 的回调与窗口方法运行在**主线程**：

- 视图函数里不做阻塞 I/O、不 `sleep`；
- 后台 goroutine 想改界面状态，通过窗口的更新入口回到主线程，不要在
  goroutine 里直接碰 UI 状态；
- 公开方法要能从任意 goroutine 安全调用。

## 2 目录结构约定

```
cmd/xinkgo/          只放 main：装配并交给 internal/app，不含业务逻辑
internal/
  app/               应用生命周期：命名、启动、窗口、菜单、IPC 绑定
  view/              窗口界面（MyGo 原生 UI），由状态绘制
  config/            用户配置：读、写、校验
  logging/           slog 初始化
  sysinfo/           版本与运行环境信息（纯数据）
resources/           随应用发布的资源，icon.png 是应用图标
docs/                工程文档
scripts/             带逻辑的脚本（门禁、打包）
mygo.json            应用名、标识符、版本、入口、打包配置
Makefile             开发入口（构建/运行/测试/调试）
```

**放置规则**（新增代码前先回答"它属于哪一层"）：

| 要写的东西 | 放哪 |
|---|---|
| 只在启动时执行一次的装配 | `internal/app` |
| 一个窗口画什么、点一下发生什么 | `internal/view` |
| 读写用户数据目录里的文件 | `internal/config`（或新增 `internal/store`） |
| 纯数据、纯计算，无 IO | 独立小包，如 `internal/sysinfo` |
| 平台差异 | `文件名_<goos>.go`（如 `platform_windows.go`），不用运行时 `switch` |
| 随包发布的静态文件 | `resources/` |

**依赖方向只允许单向**，不允许反向或循环：

```
cmd/xinkgo  →  internal/app  →  internal/{view,config,logging,sysinfo}
                                    └── internal/view → internal/sysinfo
                                    └── internal/logging → internal/config
```

- `internal/view`、`internal/config`、`internal/sysinfo` **不得** import
  `internal/app`：它们必须能在没有运行中的应用时被测试。
- `internal/app` 是唯一知道应用生命周期（启动、退出、窗口、菜单）的包。
- 新增 `internal/xxx` 包时同步更新本表与 [PROJECT.md](PROJECT.md) 的目录树。

## 3 测试约定

- 新增或修改 `internal/` 下的包，必须带同目录 `*_test.go`；`internal/app`
  这类生命周期包的逻辑尽量下沉到可测的包里。
- 目录用 `t.TempDir()`，不写真实用户目录；应用路径用
  `mygo.App.SetPath(mygo.PathUserData, dir)` 重定向。
- 视图测试用 `ui.NewTester`（无需窗口，可点击、输入）：

  ```go
  tt := ui.NewTester(h.View, 520, 400)
  if err := tt.Click("Increment"); err != nil { t.Fatal(err) }
  if !tt.HasText("1") { t.Errorf("texts %q", tt.Texts()) }
  ```

- 测试用例覆盖：正常路径 + 空/缺失 + 损坏输入 + 非法值（见
  `internal/config/config_test.go` 的六个用例）。
- 表驱动优先：`cases := map[string]want{...}` 或切片 + `t.Run(name, ...)`。
- **不依赖时间**：等待条件或事件，不 `time.Sleep` 后断言；需要推进时间的
  用 `ui.Tester` 的时钟。会随机超时的测试是竞态，要修代码或测试，不是加
  超时时间。
- 竞态：`make test-race` 在本地或 CI 至少跑一次再合。

## 4 提交信息格式

采用 [Conventional Commits](https://www.conventionalcommits.org/)：

```
<type>(<scope>): <subject>

<body>

<footer>
```

- **subject**：英文祈使式、小写开头、不加句号、≤ 50 字符。
  `add` 而不是 `added`/`adds`。
- **type**：

  | type | 用途 |
  |---|---|
  | `feat` | 新功能 |
  | `fix` | 修复缺陷 |
  | `refactor` | 重构，行为不变 |
  | `perf` | 性能优化 |
  | `test` | 测试 |
  | `docs` | 文档 |
  | `build` | 构建、依赖、打包 |
  | `chore` | 其他杂项 |

- **scope**：包名或模块名，取自目录：`app`、`view`、`config`、`logging`、
  `sysinfo`、`scripts`、`docs`、`deps`。跨包改动取主要那个，或用 `*`。
- **body**：写**为什么**改、以及改了什么（可选，但非显然的改动必须写）。
- **footer**：`Closes #12`、`Refs #34`；破坏性变更写
  `BREAKING CHANGE: ...`。

示例：

```
fix(config): keep the previous file when saving fails

Saving removed the file before writing the new one, so a crash in
between lost the configuration. Write to a temporary file and rename
it over the old one instead.

Closes #18
```

**提交粒度**：一次提交 = 一个任务（一个 `fix`、一个 `feat`、一个重构）。
不要把一个功能的重构、格式化、依赖升级混进同一提交；反过来，一个改动拆成
五个 WIP 提交也不行。提交前 `make check` 必须全绿。

禁止在提交信息里写 `[skip ci]`（含正文与引用的文本）：CI 会被跳过。

## 5 分支管理策略

### 分支

| 分支 | 从哪切 | 用途 | 合回 |
|---|---|---|---|
| `main` | — | 随时可发布；受保护，禁止直接 push | — |
| `feature/<slug>` | `main` | 新功能 | `main`（PR） |
| `bugfix/<slug>` | `main` | 缺陷修复 | `main`（PR） |
| `release/<version>` | `main` | 发版准备：版本号、变更日志、冒烟 | `main` 并打 tag |
| `hotfix/<version>` | `main` 的 tag | 线上紧急修复 | `main` 与 `release/*` |

- `slug` 用小写字母、数字与连字符：`feature/web-window`、`bugfix/config-lost`。
- 一个分支只做一件事；分支寿命尽量短（建议 ≤ 3 天），长期分支要定期 rebase
  `main`（不 merge `main` 进特性分支来"同步"，用 rebase 保持线性）。
- **禁止直接向 `main` 推送**、**禁止在未经确认时把分支合并到 `main`**：合并
  是人的决定，AI/脚本只负责在分支上提交与推送，完成后报告分支名等待确认。

### 拉取请求

- 标题即 squash 后的提交信息，遵循第 4 节格式。
- 描述写：改了什么、为什么、怎么验证（命令 + 结果摘要）、影响范围。
- 合并前 checklist：

  - [ ] `make check` 全绿（把输出摘要贴进 PR）
  - [ ] 新增/修改的 `internal/` 包有测试
  - [ ] 文档随代码更新（`docs/`、`README.md`）
  - [ ] 涉及界面的改动附截图或说明人工验收项

- 合并方式：squash merge，保持 `main` 线性；合并后删除分支。

### 版本与发布

- 版本号遵循语义化版本；**两处必须一致**：`mygo.json` 的 `version` 与
  `internal/app/app.go` 的 `const Version`（`scripts/check.sh` 与
  `scripts/package.sh` 会校验）。
- 发版：`release/0.2.0` → 改版本 → `make package` → 冒烟 → 合 `main` →
  打 `v0.2.0` tag。
- 应用标识符（`mygo.json` 的 `identifier`）一经发布不得更改：系统用它作为
  配置、权限、文件关联的键。

## 6 代码评审关注点

1. 分层是否被破坏（依赖方向、是否把业务逻辑写进 `cmd/` 或视图）。
2. 错误是否被吞、是否带上下文、是否误用 `panic`。
3. 是否引入 cgo 依赖或破坏跨平台编译。
4. 视图函数是否只画界面；耗时操作是否堵在主线程。
5. 测试是否覆盖边界、是否依赖 `sleep`。
6. 提交与分支是否符合第 4、5 节。
7. 涉及构建、依赖、打包配置的改动：macOS（P0）是否先绿，Windows 与 Linux
   是否仍可编译（`make check` 的输出摘要贴进 PR）。
