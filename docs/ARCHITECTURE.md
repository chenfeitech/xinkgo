# 架构与分层

骨架阶段只有少量代码，但分层先定死，后续功能按层长。

## 1 分层

```
┌──────────────────────────────────────────────┐
│ cmd/xinkgo        main：装配，无逻辑          │
├──────────────────────────────────────────────┤
│ internal/app      生命周期：命名 / 启动 /      │
│                   菜单 / 窗口 / IPC 绑定      │
├─────────────┬──────────┬──────────┬──────────┤
│ view        │ config   │ logging  │ sysinfo  │
│ 界面（绘制）│ 持久化   │ 日志     │ 版本信息 │
└─────────────┴──────────┴──────────┴──────────┘
        ↑ 依赖方向单向，view/config/sysinfo 不依赖 app
```

| 层 | 职责 | 不许做 |
|---|---|---|
| `cmd/xinkgo` | 调 `app.Run()`，失败 `log.Fatal` | 写业务逻辑 |
| `internal/app` | 应用生命周期、窗口与菜单、IPC 绑定 | 画界面、直接读写配置格式 |
| `internal/view` | 由状态画界面，处理输入改变状态 | I/O、调应用生命周期接口、写任何文件（要写就回调 `app`） |
| `internal/config` | `config.json` 的读、写、校验 | 依赖应用对象（目录由参数传入） |
| `internal/theme` | 配色方案表：每套方案明暗两套取色，视图上色用 | 依赖 `config` / `app`（它是纯数据，只依赖 `ui`） |
| `internal/export` | 把笔记写成自成一页的 HTML（PDF 由系统 webview 打印它） | 读文件、弹对话框（那些是 `internal/app` 的事） |
| `internal/wechat` | 公众号排版：样式表与行内样式渲染（剪贴板里的 HTML 片段） | 读笔记的图片、写剪贴板（那些是 `internal/app` 的事） |
| `internal/logging` | 配置 slog 的默认 logger | 业务日志之外的输出策略 |
| `internal/sysinfo` | 版本与环境信息（纯数据） | 任何 I/O |
| `internal/plugin` | Obsidian 插件 manifest、社区目录、能力扫描与兼容等级（纯数据） | 磁盘 I/O、窗口、依赖 `app` |
| `internal/textfile` | 文本文件判类、内容嗅探、BOM/编码、源码围栏 | 写文件、画界面、依赖 `app`/`view` |
| `internal/view/geom` | 几何：面板与编辑区宽度、留白、边距、换行列数、侧栏带 | 画东西、读 `ui.Context`；它是纯计算，可脱离窗口单测 |
| 领域包 `vault` `note` `index` `search` `ai` `task` `skill` `sync` `snapshot` `markdown` `highlight` `datafile` `media` `textfile` `wechat` `theme` `plugin` | 数据与算法：笔记、库、索引、模型、样式、文本判类、插件报告 | 写用户数据、弹对话框（那是 `internal/app` 的事）；`vault`/`note`/`search`/`textfile` 只读库内文件 |
| 持久化 `config` `state` `secret` `fs` | 各管各的文件，见第 7 节 | 互相写对方的文件 |

## 2 启动时序

```
main()
  └─ app.Run()
       ├─ mygo.App.SetName(Name)          // 先命名：它决定用户数据目录
       ├─ WhenReady(...)                  // 此时应用目录才可用
       │    ├─ sysinfo.Current(name, version)
       │    ├─ start(info)
       │    │    ├─ Path(PathUserData) → config.Load / 损坏则降级并回写
       │    │    ├─ Path(PathLogs)     → logging.Setup
       │    │    └─ mygo.Bind(system{info})   // 供页面调用
       │    ├─ home := view.NewHome(info); home.OnWeb = openWebWindow
       │    ├─ SetMenu(mainMenu(info))
       │    └─ openMainWindow()          // 原生 UI 窗口，StateKey="main"
       ├─ OnWindowAllClosed / OnActivate  // 平台退出行为
       └─ mygo.App.Run()                 // 阻塞到退出
```

要点：

- **目录只能在 `WhenReady` 之后取**：用户数据目录由应用名决定。
- **配置损坏不阻断启动**：降级为默认值 + `Warn`，并尝试回写。
- **版本两处一致**：`mygo.json`（打包注入）与 `internal/app/app.go`（`go run`
  时的回退），由 `scripts/*.sh` 校验。

## 3 线程模型

- MyGo 的回调（`WhenReady`、菜单 `Click`、视图函数）与窗口方法跑在**主线程**。
- 视图函数每帧都会调用：只画界面 + 改状态，**不做阻塞 I/O、不 sleep**。
- 后台 goroutine 需要改变界面状态时，通过窗口的更新入口回到主线程，不要直接
  改 UI 状态。
- 对外暴露的方法要能从任意 goroutine 安全调用。

## 4 状态与持久化

| 数据 | 真值 | 位置 |
|---|---|---|
| 用户配置 | `config.json` | 用户数据目录（`config.Load/Save`） |
| 配色方案 | `internal/theme` 的方案表（代码里） | 只存方案名，取色不落盘 |
| 窗口位置尺寸 | MyGo 窗口状态 | `WindowOptions.StateKey` |
| 应用版本 | 打包注入 / 源码常量 | `sysinfo.Info` |
| 日志 | 追加写文件（安装版） | 日志目录 `xinkgo.log` |

`config.Save` 先写 `config.json.tmp` 再替换目标文件（Windows 不允许 rename
覆盖已存在文件，先 `Remove` 再 `Rename`），保证写坏时旧文件还在。

## 5 扩展点

| 想加什么 | 改哪 |
|---|---|
| 新窗口 / 新界面 | `internal/view` 加视图，`internal/app` 加打开入口与菜单项 |
| 新配置项 | `internal/config` 加字段 + 默认值 + 校验 + 测试 |
| 新配色方案 | `internal/theme` 的 `schemes` 加一套（明暗两套 `Palette` + 代码预设）；下拉、色卡、校验都从这张表读 |
| 新明暗行为 | `config.Basic.Appearance` + `mygo.Theme.SetSource`（`app.setAppearance`） |
| Web 窗口 / 前端 | `internal/app/web.go` 的绑定服务 + `mygo generate` 生成 TS 客户端 |
| 平台差异行为 | `internal/app/platform_<goos>.go` |
| 新界面文案 / 新语言 | `internal/locale` 的 `en.json` 与 `zh-Hans.json`，`go tool mygo generate` 生成 `locale.Xxx()` |
| Obsidian 兼容插件 | `internal/plugin` 加纯解析/扫描；`internal/app/plugin_*.go` 加协议、宿主、RPC、目录与管理器；`resources/obsidian-shim/` 加 JS shim |
| 自动更新、托盘 | `mygo.json` 配置 + 对应 MyGo 插件，按官方插件方式接入 |

## 6 多语言

界面文案不写在代码里，写在 **catalog**（`internal/locale`，一个 JSON 文件一门
语言）：`en.json` 是源语言，英文写在那里；`zh-Hans.json` 是简体中文；代码调
`go tool mygo generate` 生成的 `locale.Xxx()`，它按**应用当前语言**说话。

```
internal/locale/en.json      ← 源语言（英文），键名 + 文案 + @键 的元数据
internal/locale/zh-Hans.json ← 翻译，键集与源文件一致
internal/locale/catalog_gen.go ← 生成，勿改：embed 两个文件、注册 catalog、
                                 每个键一个函数
mygo.json                    ← "i18n": {"dir": "internal/locale", "source": "en"}
```

要点：

- **键名说“这句话是干什么的”**（`menuOpenVault`、`sayNoteNotSaved`），不说它的
  英文原文；加了新语言才不用改键名。键名是 `[A-Za-z][A-Za-z0-9_]*`（无点号）。
- **带数字 / 名字的文案用 ICU**：`"{count, plural, one {# note} other {# notes}}"`、
  `"{name} is as it was again"`；生成的是 `locale.NoteCount(count int)` 这样的
  带参函数。中文只有 `other` 一种复数形式。
- **语言由配置选**：`config.json` 的 `basic.language`，`system`（跟随系统）/ `en`
  / `zh-Hans`；设置 ▸ 基本 ▸ 语言里改，`internal/app` 把它交给
  `mygo.App.SetLocale`。MyGo 自己的控件、对话框、菜单角色（如“撤销”）也跟着换。
- **菜单要重建，界面不用**：菜单是建好就不再变的，所以 `OnLocaleChanged` 里
  `refreshMenu()`；`internal/view` 每帧重画，文案在画的时候取，切语言即刻生效。
- **MyGo 自己的中文由生成代码引入**：`catalog_gen.go` 会按 catalog 的语言自动
  `import _ "github.com/egoist/mygo/locales/zh-Hans"`。
- **测试跑在英文下**：没有应用运行时 `i18n.Locale()` 取源语言，所以断言文案的
  测试写**英文**，不要写中文，也不要写键名。
- **域内文案**（`vault.Status.String()`、`task.Kind.Label()`、`media.Kind.String()`
  、Markdown 工具栏的 `Label()`/`Desc()`）也走 catalog：`internal/locale` 是叶子
  包，谁都可以 import，不会破坏依赖方向。

校对命令：`go tool mygo i18n check`（缺键、复数形式、参数不一致都会报）。

## 8 `internal/app` 自己怎么分

`app` 是唯一碰磁盘、弹对话框、起窗口的一层，所以它天生最长。为了让“改哪件事”
不用在 800 行里找，它按**事情**分文件，而不是按技术角色分：

| 文件 | 装什么 |
|---|---|
| `app.go` | 装配与生命周期：`Run`、启动恢复、`appState`、版本 |
| `app_vault.go` | 库的打开、记住、切换，以及切换后的重建索引 |
| `app_apply.go` | 配置改动对运行中的 app 做什么：外观、语言、重画菜单、退出前写盘 |
| `watch.go` | 盯着库的外部改动，交回主线程 |
| `menu.go` / `window.go` / `host.go` | 菜单、窗口、IPC 绑定 |
| `ask.go` | 问模型的编排（回调给 `view`） |
| `files.go` / `export.go` / `wechat.go` | 文件对话框、导出、公众号（都要 I/O，所以在这里） |
| `settings*.go` | 设置窗口，一页一个文件 |

两条规则：

- **不为了“分层”把业务下沉。** `ask.go`/`export.go`/`wechat.go` 看着像编排，
  但它们要 I/O、要对话框、要碰 `appState`，而下沉就意味着领域包开始做 I/O ——
  那才是破坏分层。它们留在 `app`，按文件分开。
- **`app` 里再出现 600 行的文件，就按上面的方式再切一次。**

## 7 持久化与状态

五个包往磁盘写东西，各管各的文件，互不重叠。想知道“这条设置 / 这个状态存在
哪、谁写的”，看这张表：

| 包 | 管的文件 | 谁读 | 谁写 | 什么时候写 | 读不到时 |
|---|---|---|---|---|---|
| `config` | `config.json`（应用目录） | `app`、`view` | `app`（设置窗口） | 设置里改动即写 | 取默认值，不报错 |
| `state` | `open_notes.json` / `vaults.json` / `layout.json` / `ai_sessions.json` / `plugins.json`（应用目录） | `app`（启动恢复）、`view`（面板宽度） | `app`（关窗、切库、对话存盘、插件启停与授权） | 退出、切库、`OnLayoutChanged` 落定后、插件状态变化时 | 取可用值（空列表 / 默认布局 / 不启用插件），至多写一条日志 |
| `snapshot` | 库外：应用目录下按库分文件夹的笔记快照 | `app`（“回到改动前”） | `app`（模型改动笔记前） | 改动写入笔记之前 | 没有快照就是没有可回退的 |
| `secret` | 系统钥匙串（不是文件） | `app`（读密钥给模型） | `app`（设置里填了密钥） | 填密钥时 | 系统无处存放则告知，密钥退回设置里 |
| `fs` | 库里的笔记与目录（不是应用状态） | `app` | `app`（导入、改名、移动） | 用户动作时 | 返回错误，不动磁盘 |

规则：

- **领域包不碰磁盘。** `note` / `vault` / `index` / `task` / `ai` 只处理内存里的
  数据；要落盘的是 `app` 的事，由它调上面这些包。
- **`view` 只读不写。** `view` 可以读 `config`（画界面要按设置）、读
  `state.Layout`（面板宽度），但它不写任何文件：要写就回调给 `app`。
- **`session` 不存文件。** 它是“哪些笔记开着、按什么顺序、哪个是当前的”的
  规则（纯路径运算），开着的笔记由 `state` 存。
- **读不到不能挡住启动。** 这条对 `state` 是硬的：文件缺失、损坏、版本不同，
  都给一个能用的值。
