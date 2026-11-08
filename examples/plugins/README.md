# 插件示例

两个可以直接安装的 Obsidian 兼容插件示例，用来展示 Xinkgo 插件宿主的「基础能力」和
「能力全集」。它们不是产品功能，只是把宿主真实支持的 API 面走一遍，方便验证插件机制、
写新插件时对照，以及在兼容性报告里看到不同等级长什么样。

| 示例 | 定位 | 预期等级 | 降级 |
| --- | --- | --- | --- |
| `xinkgo-demo-basic` | 最小可用插件：命令、提示、持久化数据、ribbon、状态栏、读当前笔记 | C2 / `LevelNative` | 否 |
| `xinkgo-demo-full` | 覆盖宿主全部 14 项能力，包括插件自己的 DOM 表面与兼容 stub | C4 / `LevelEditor` | 是 |

两者都 `Supported=true`；`xinkgo-demo-full` 的 `Degraded=true` 是刻意的，用来演示
「能加载，但 DOM / CM6 / Node 模块不是真 Obsidian」这一类插件在管理器里的呈现。
面向使用者的逐步安装与试用手册见
[docs/plugin-demos.md](../../docs/plugin-demos.md)，两个插件目录内也各有一份简短
README，拷目录时会一起带过去。想自己写插件看
[插件开发指南](../../docs/plugin-development.md)，它以这两个 demo 为案例逐段讲解。

## 安装

插件包就是一个目录，目录名必须等于 `manifest.json` 里的 `id`，里面至少要有
`manifest.json` 和 `main.js`，`styles.css`、`assets/` 可选。宿主按下面三处发现，
后面的来源覆盖前面的同 id 包：

| 来源 | 路径 |
| --- | --- |
| 用户级（推荐） | macOS：`~/Library/Application Support/Xinkgo/plugins/<id>/`；Windows：`%AppData%\Xinkgo\plugins\<id>\` |
| Obsidian 兼容 | `<vault>/.obsidian/plugins/<id>/` |
| 库内原生 | `<vault>/.xink/plugins/<id>/` |

也可以在插件设置窗口对任一插件点 **Open folder** 直接打开用户级插件目录。macOS 上的
复制示例：

```sh
PLUGINS="$HOME/Library/Application Support/Xinkgo/plugins"
mkdir -p "$PLUGINS"
cp -R examples/plugins/xinkgo-demo-basic "$PLUGINS/"
cp -R examples/plugins/xinkgo-demo-full "$PLUGINS/"
```

改完包内容后在插件管理器点 **Refresh**；改了 `manifest.json` 需要 **Reload** 插件。

## 启用与授权

设置窗口 → **Community plugins** → **Installed**，先启用插件，再展开卡片里的
**Grants** 逐项勾选。`data`、`commands`、`notice` 是启用插件即有的生命周期能力，
其余默认全关。

注意两种安装路径不同：手动拷目录时宿主不写任何授权，需要逐项勾；从 **Browse**
一键安装时，宿主会把静态扫描出的能力**全部写入授权并自动启用**，与
[设计文档 §5.1](../../docs/design/obsidian-plugins.md) 的「默认全关」口径不一致。
ribbon、状态栏、视图都在 `onload()` 里注册，当时没授权就不会补回来，所以要么
先勾授权再启用，要么勾完点一次 **Reload**。

`xinkgo-demo-full` 要跑完所有命令，需要勾选这 11 项：`vault.read`、`vault.write`、
`editor.read`、`editor.write`、`metadata`、`network`、`events.app`、`ui`、`dom`、
`editor.extension`、`compat.stub`。少勾一项时对应 RPC 会拒绝，插件用 `Notice` 和日志
报出原因，不会让插件加载失败。注意 `editor.extension` 即使勾上也只是被记录，宿主
没有 CodeMirror 6 编辑器可供挂载。

`xinkgo-demo-basic` 建议至少勾 `ui`、`vault.read`、`editor.read`、`editor.write`，
否则 ribbon / 状态栏和笔记读写会被拒绝。

## xinkgo-demo-basic

命令（命令面板里都带 `Xinkgo Demo: Basic:` 前缀）：

| 命令 | 演示 |
| --- | --- |
| Say hello | `Notice` + `loadData/saveData` 计数 |
| Read the active note | `vault.read` |
| Count words in the active note | `editor.getValue`、`getActiveViewOfType` |
| Insert today's date at the caret | `editorCallback`、`replaceSelection` |
| Describe the selection | `editorCheckCallback`（无选区时命令自动隐藏）、`getSelection` |

左侧 ribbon 有一个 Say hello 图标，状态栏显示运行次数，另外监听 `file-open` 事件。
它不碰插件 DOM，所以扫描只会命中 C1/C2 的规则，报告为非降级。

## xinkgo-demo-full

### 命令

| 命令 | 演示 |
| --- | --- |
| Print the capability tour | 在插件日志里打印 14 项能力 |
| Say hello and persist a counter | `Notice`、`loadData/saveData`、状态栏 |
| Read the active note | `vault.read`、`cachedRead`、`adapter.read`、`editor.getValue/getSelection/getCursor/getRange` |
| Insert a line at the caret | `editorCallback`、`replaceSelection` |
| Replace the whole buffer | `editor.setValue` |
| Show selection stats | `editorCheckCallback` 动态可见性 |
| Run a Vault lifecycle | `checkCallback` 动态可见性；`createFolder/create/modify/append/rename/delete` |
| Inspect metadata and open the note | `MetadataCache.getFileCache`、`getFirstLinkpathDest`、`workspace.openLinkText`、`FileManager` |
| Fetch example.com | `requestUrl`，**只在手动执行该命令时联网**，需要 `network` |
| Build DOM samples | `Modal`、`Menu`、`MenuItem`、`SuggestModal`、`FuzzySuggestModal`、`MarkdownRenderer`、`HoverPopover`、`addIcon/setIcon`、`activeDocument/activeWindow` |
| Inspect registered editor extensions | `registerEditorExtension`、`EditorSuggest`、`registerEditorSuggest` 的登记状态 |
| Exercise compatibility stubs | `require("path")` 的 `compat.stub`、`secretStorage`、协议/扩展/hover 注册 |
| Toggle a temporary command | `removeCommand` / `addCommand` |
| Where is the demo view? | 提示视图入口 |

### 视图与设置 tab

插件注册了一个 `demo-full-view`：在主窗口右侧的**插件视图抽屉**里选中它，宿主才会创建
`WorkspaceLeaf` stub、调用 `ItemView` 并把 `containerEl` 挂到插件自己的 WebView 页面。
视图里有一个按钮、一行行文本和一张从 `/plugin/assets/badge.svg` 取出的图片，用来证明
插件 DOM 与静态资源确实工作。

`addSettingTab(new PluginSettingTab(...))` 会在 `onload` 里立刻调用 `display()`，用
`new Setting(...)` 画出两个控件。这些控件是惰性 stub，页面也只是插件自己的 WebView，
**不会**出现在 Xinkgo 的原生设置窗口。插件自己的配置请用 `loadData/saveData`，宿主把
`data.json` 写到插件包所在目录：用户级安装时是 `<appdata>/plugins/<id>/data.json`，
库内安装时则会落进库目录（与设计文档 §5.2 的口径不同，注意私密配置不要随库同步）。

### 日志

插件的 `console.log` / `console.error` 通过桥转成宿主日志，行里有
`msg="plugin console"` 与 `id=<插件 id>`；开发版输出到终端，安装版写到
`~/Library/Logs/Xinkgo/xinkgo.log`（Windows：`%AppData%\Xinkgo\logs\xinkgo.log`）。
所有降级点都写进日志：
CM6 扩展只登记不生效、Markdown 后处理器只登记不改原生预览、协议 handler / hover /
bases / CLI 注册是 no-op、`secretStorage` 是页内 stub、`require("path")` 在授权后返回
一个惰性 Proxy。

## 能力对照

| 能力 | Basic | Full | 主要 API |
| --- | --- | --- | --- |
| `data` | 有 | 有 | `loadData` / `saveData` |
| `commands` | 有 | 有 | `addCommand` / `removeCommand` |
| `notice` | 有 | 有 | `new Notice` |
| `vault.read` | 有 | 有 | `vault.read`，Full 另有 `cachedRead` / `adapter.read` |
| `vault.write` | 无 | 有 | `vault.create/modify/append/rename/delete`、`createFolder` |
| `editor.read` | 有 | 有 | `getValue` / `getSelection` / `getActiveViewOfType` |
| `editor.write` | 有 | 有 | `replaceSelection`，Full 另有 `editor.setValue` |
| `metadata` | 无 | 有 | `MetadataCache` / `getFileCache` / `getFirstLinkpathDest` |
| `network` | 无 | 有 | `requestUrl` |
| `events.app` | 无 | 有 | `registerDomEvent` / `registerInterval` / `activeDocument` / `activeWindow` |
| `ui` | 有 | 有 | `addRibbonIcon` / `addStatusBarItem` / `styles.css`，Full 另有 `addIcon` / `setIcon` |
| `dom` | 无 | 有 | `registerView` / `ItemView` / `containerEl` / `addSettingTab` / `new Setting(` / `Modal` / `Menu` / `MarkdownRenderer` |
| `editor.extension` | 无 | 有 | `registerEditorExtension` / `EditorSuggest` / `registerEditorSuggest` |
| `compat.stub` | 无 | 有 | `require("path")` 的惰性 Proxy |

`manifest.json` 里的 `xinkgo.capabilities` 只是给作者看的声明提示，Obsidian 会忽略它；
实际授权以 `main.js` 静态扫描 + 用户勾选 + 运行时校验为准。
`xinkgo-demo-basic` 故意不写这段提示，能力完全由扫描得出；`xinkgo-demo-full` 写上
14 项作为文档，并由测试校验提示与扫描结果一致。

## 明确不做

- 真实的 CodeMirror 6 编辑器扩展、editor suggest 与 Markdown 后处理：宿主是原生编辑器，
  这些注册只被记录。
- 原生设置窗口、原生菜单、原生 Modal 与 `WorkspaceLeaf` 布局：插件 DOM 只在自己的
  WebView 页面里有效，`styles.css` 也不会给原生 UI 上色。
- 开机自动联网、访问本机文件系统、加载 Node/Electron 模块：外部模块要么拒绝，要么在
  `compat.stub` 授权下变成惰性 Proxy。
- CLI Agent、外部网络搜索等与插件宿主无关的能力。

## 写插件时的硬约束

`main.js` 必须是 CommonJS：`require("obsidian")`、`module.exports = 一个继承
obsidian.Plugin 的类`。不能有 ESM 语法、相对导入、动态模块名；`manifest.json` 的 `id`
必须匹配 `^[a-z][a-z0-9-]*$` 且等于目录名，`version` / `minAppVersion` 必须是语义版本，
`minAppVersion` 不能超过宿主的 `APIProfile`（当前 `1.0.0`）。

这两个示例由 `internal/plugin/demo_plugins_test.go` 校验：包结构、无硬性扫描错误、
两个等级、以及 Full 的能力全集与 manifest 声明一致。改动示例后跑：

```sh
go test ./internal/plugin -run DemoPlugins
make check
```
