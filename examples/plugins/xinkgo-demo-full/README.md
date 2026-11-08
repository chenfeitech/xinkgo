# Xinkgo Demo: Full

覆盖 Xinkgo **全力支持**的全部 11 项能力的示例：Vault 与 Editor 读写、
`workspace.openLinkText`、`requestUrl`、app 事件、ribbon / 状态栏，以及宿主真实
实现的 `ItemView` 视图表面。兼容等级 **C3 / LevelDOM**，`Degraded=true` 只是宿主的
分级标记（视图在插件自己的 WebView 页，不是原生 UI 面板），不代表功能缺失。

宿主能扫描出 14 项能力，但 `metadata`、`editor.extension`、`compat.stub` 只是兼容
占位，本插件**不写**它们；设置 tab、弹层、CM6 注册、外部模块等 no-op API 同样没有
出现。理由见 [插件示例总览](../README.md#不写进插件的能力)。

## 安装

把**整个目录**拷到用户级插件目录，目录名必须保持 `xinkgo-demo-full`：

```sh
PLUGINS="$HOME/Library/Application Support/Xinkgo/plugins"
mkdir -p "$PLUGINS"
cp -R xinkgo-demo-full "$PLUGINS/"
```

Windows 用 `%AppData%\Xinkgo\plugins\`。`assets/badge.svg` 要一起拷过去，
插件视图里的图片从 `/plugin/assets/` 取。拷完点 **Refresh** 再启用。

## 授权

`data`、`commands`、`notice` 启用即有，其余 8 项都要勾：
`vault.read`、`vault.write`、`editor.read`、`editor.write`、`network`、
`events.app`、`ui`、`dom`。**先勾授权再启用**，或勾完点一次 **Reload**，否则
ribbon、状态栏和视图注册不会补回来。

## 命令

命令面板里都带 `Xinkgo Demo: Full:` 前缀：

| 命令 | 作用 |
| --- | --- |
| Print the capability tour | 日志打印 11 项受支持能力 |
| Say hello and persist a counter | Notice + 计数 |
| Read the active note | 三个 Vault 读适配器 + Editor 读取 |
| Insert a line at the caret | `replaceSelection` |
| Replace the whole buffer | `editor.setValue`，**会改当前笔记** |
| Show selection stats | 需选区，否则命令隐藏 |
| Run a Vault lifecycle | 创建 → 修改 → 追加 → 改名 → 删除，**会改库内文件** |
| Open the active note by link | `workspace.openLinkText`，真实打开笔记 |
| Fetch example.com | `requestUrl`，**只在手动点这条命令时联网** |
| Toggle a temporary command | `removeCommand` / `addCommand` |
| Where is the demo view? | 提示视图入口 |

## 视图

`registerView` 注册了 `demo-full-view`：在主窗口右侧的**插件视图抽屉**里选中它，
才能看到 `ItemView` 的按钮、行文本和 `assets/badge.svg`。这是宿主真实实现、能点击
交互的表面；插件 DOM 只在这一页里有效，不会改 Xinkgo 原生 UI。

## 事件与日志

`registerAppEvents()` 监听宿主真实会发的 `file-open`、`vault.modify`、
`editor-change`，外加 DOM click 与 interval。事件不带文件对象，需要路径请调
`workspace.getActiveFile()`。所有 RPC 调用都带 `.catch()`，未授权时弹 Notice 并记日志。

## 完整文档

逐条预期、日志位置与排错见
[使用手册](../../../docs/plugin-demos.md)，能力对照与准入原则见
[插件示例总览](../README.md)；以本插件为案例的逐段讲解见
[插件开发指南](../../../docs/plugin-development.md)。
