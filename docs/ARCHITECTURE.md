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
| `internal/view` | 由状态画界面，处理输入改变状态 | I/O、调应用生命周期接口 |
| `internal/config` | `config.json` 的读、写、校验 | 依赖应用对象（目录由参数传入） |
| `internal/theme` | 配色方案表：每套方案明暗两套取色，视图上色用 | 依赖 `config` / `app`（它是纯数据，只依赖 `ui`） |
| `internal/logging` | 配置 slog 的默认 logger | 业务日志之外的输出策略 |
| `internal/sysinfo` | 版本与环境信息（纯数据） | 任何 I/O |

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
| 自动更新、i18n、托盘 | `mygo.json` 配置 + 对应 MyGo 插件，按官方插件方式接入 |
