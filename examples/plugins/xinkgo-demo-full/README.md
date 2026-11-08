# Xinkgo Demo: Full

覆盖 Xinkgo 插件宿主全部 14 项能力的示例：Vault 与 Editor 读写、MetadataCache、
`requestUrl`、事件、ribbon / 状态栏、插件自己的 `ItemView` DOM 表面、设置 tab、
Modal / Menu、CM6 注册与兼容 stub。兼容等级 **C4 / LevelEditor**，
`Degraded=true`：DOM、CM6 与 Node 模块都是 shim / stub，不是真 Obsidian。

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

`data`、`commands`、`notice` 启用即有，其余 11 项都要勾：
`vault.read`、`vault.write`、`editor.read`、`editor.write`、`metadata`、
`network`、`events.app`、`ui`、`dom`、`editor.extension`、`compat.stub`。
**先勾授权再启用**，或勾完点一次 **Reload**，否则 ribbon、状态栏和视图注册
不会补回来。`editor.extension` 即使勾上也只是被记录，宿主没有 CM6 编辑器。

## 命令

命令面板里都带 `Xinkgo Demo: Full:` 前缀：

| 命令 | 作用 |
| --- | --- |
| Print the capability tour | 日志打印 14 项能力 |
| Say hello and persist a counter | Notice + 计数 |
| Read the active note | 三个 Vault 适配器 + Editor 读取 |
| Insert a line at the caret | `replaceSelection` |
| Replace the whole buffer | `editor.setValue`，**会改当前笔记** |
| Show selection stats | 需选区，否则命令隐藏 |
| Run a Vault lifecycle | 创建 → 修改 → 追加 → 改名 → 删除，**会改库内文件** |
| Inspect metadata and open the note | MetadataCache / 双链 / `openLinkText` |
| Fetch example.com | `requestUrl`，**只在手动点这条命令时联网** |
| Build DOM samples | Modal / Menu / SuggestModal / MarkdownRenderer |
| Inspect registered editor extensions | CM6 登记的扩展数量 |
| Exercise compatibility stubs | `require("path")` 的惰性 Proxy |
| Toggle a temporary command | `removeCommand` / `addCommand` |
| Where is the demo view? | 提示视图入口 |

## 视图

`registerView` 注册了 `demo-full-view`：在主窗口右侧的**插件视图抽屉**里选中它，
才能看到 `ItemView` 的按钮、行文本和 `assets/badge.svg`。Modal、Menu、
设置 tab 也只画在这个插件自己的 WebView 页面里，不会改 Xinkgo 原生 UI。

## 完整文档

逐条预期、日志位置与排错见
[使用手册](../../../docs/plugin-demos.md)，能力对照与开发约束见
[插件示例总览](../README.md)；以本插件为案例的 API 映射见
[插件开发指南](../../../docs/plugin-development.md)。
