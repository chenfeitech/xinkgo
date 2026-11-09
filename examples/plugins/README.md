# 插件示例

两个可以直接安装的 Obsidian 兼容插件示例，用来展示 Xinkgo 插件宿主的「基础能力」和
「能力全集」。它们不是产品功能，只是把宿主**真正支持**的 API 面走一遍，方便验证插件
机制、写新插件时对照，以及在兼容性报告里看到不同等级长什么样。

> **准入原则**：插件里只写 Xinkgo 有真实实现的能力。宿主为了兼容 Obsidian 而提供的
> 空壳、惰性代理和 no-op 注册，不因为「Obsidian 有」就写进插件。下文的
> 「不写进插件的能力」列出了本次从全面 demo 里移除的部分。

| 示例 | 定位 | 预期等级 | 降级 |
| --- | --- | --- | --- |
| `xinkgo-demo-basic` | 最小可用插件：命令、提示、持久化数据、ribbon、状态栏、读当前笔记 | C2 / `LevelNative` | 否 |
| `xinkgo-demo-full` | 覆盖宿主全力支持的全部 11 项能力，含真实插件视图表面 | C3 / `LevelDOM` | 是（见下） |

两者都 `Supported=true`。Full 的 `Degraded=true` 来自宿主的分级规则：只要用了
`registerView` 这类 DOM 级能力就标降级，含义是「视图运行在插件自己的 WebView 页，
不是原生 UI 面板」，**不是**功能缺失——这个视图是宿主真实实现、能点能交互的。

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

注意两种安装路径都**不写任何授权**：手动拷目录和从 **Browse** 一键安装都需要
逐项勾选，安装状态会提示「授权后重载」。早期版本的一键安装会自动全授权，与
[设计文档 §5.1](../../docs/design/obsidian-plugins.md) 的「默认全关」不一致，
2026-11-09 起已修正。
ribbon、状态栏、视图都在 `onload()` 里注册，当时没授权就不会补回来，所以要么
先勾授权再启用，要么勾完点一次 **Reload**。

设置 ▸ **Community plugins** ▸ **Official library** 列出官方插件库；库内插件
已安装时会在 **Installed** 页带 `Verified` 徽标，未通过能力契约的库内插件会被
拒绝启用。外部兼容插件不在此保证范围内。

`xinkgo-demo-full` 要跑完所有命令，需要勾选这 8 项：`vault.read`、`vault.write`、
`editor.read`、`editor.write`、`network`、`events.app`、`ui`、`dom`。少勾一项时对应
RPC 会拒绝，插件用 `Notice` 和日志报出原因，不会让插件加载失败。

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
每条命令都会同时弹 `Notice` 和写一行 `console.log`（经桥转成宿主日志），便于从终端
或 `xinkgo.log` 观察。它不碰插件 DOM，所以扫描只会命中 C1/C2 的规则，报告为非降级。

## xinkgo-demo-full

### 命令

| 命令 | 演示 |
| --- | --- |
| Print the capability tour | 在插件日志里打印 11 项受支持能力 |
| Say hello and persist a counter | `Notice`、`loadData/saveData`、状态栏 |
| Read the active note | `vault.read`、`cachedRead`、`adapter.read`、`editor.getValue/getSelection/getCursor/getRange` |
| Insert a line at the caret | `editorCallback`、`replaceSelection` |
| Replace the whole buffer | `editor.setValue` |
| Show selection stats | `editorCheckCallback` 动态可见性 |
| Run a Vault lifecycle | `checkCallback` 动态可见性；`createFolder/create/modify/append/rename/delete` |
| Open the active note by link | `workspace.openLinkText`，真实打开笔记并触发 `file-open` |
| Fetch example.com | `requestUrl`，**只在手动执行该命令时联网**，需要 `network` |
| Toggle a temporary command | `removeCommand` / `addCommand` |
| Where is the demo view? | 提示视图入口 |

### 插件视图

插件注册了一个 `demo-full-view`：在主窗口右侧的**插件视图抽屉**里选中它，宿主才会创建
`WorkspaceLeaf` stub、调用 `ItemView` 并把 `containerEl` 挂到插件自己的 WebView 页面。
视图里有一个「Add a row」按钮、逐行文本和一张从 `/plugin/assets/badge.svg` 取出的
图片，用来证明插件 DOM 与静态资源确实工作。这是宿主**真实实现**的表面，不是占位。

插件自己的配置请用 `loadData/saveData`，宿主把 `data.json` 写到插件包所在目录：
用户级安装时是 `<appdata>/plugins/<id>/data.json`，库内安装时则会落进库目录
（与设计文档 §5.2 的口径不同，注意私密配置不要随库同步）。

### 事件与日志

`registerAppEvents()` 监听宿主真实会发的 `file-open`、`vault.modify`、
`editor-change`，外加一个真实的 DOM click 与 interval。事件只作信号、不带文件对象，
要拿路径就调 `workspace.getActiveFile()`。

插件的 `console.log` / `console.error` 通过桥转成宿主日志，行里有
`msg="plugin console"` 与 `id=<插件 id>`；开发版输出到终端，安装版写到
`~/Library/Logs/Xinkgo/xinkgo.log`（Windows：`%AppData%\Xinkgo\logs\xinkgo.log`）。
两个 demo 的所有 RPC 调用都带 `.catch()`，未授权时用 `Notice` + 日志说明原因。

## 能力对照

这 11 项是宿主**全力支持**、两个 demo 允许使用的能力：

| 能力 | Basic | Full | 主要 API |
| --- | --- | --- | --- |
| `data` | 有 | 有 | `loadData` / `saveData` |
| `commands` | 有 | 有 | `addCommand` / `removeCommand` |
| `notice` | 有 | 有 | `new Notice` |
| `vault.read` | 有 | 有 | `vault.read`，Full 另有 `cachedRead` / `adapter.read` |
| `vault.write` | 无 | 有 | `vault.create/modify/append/rename/delete`、`createFolder` |
| `editor.read` | 有 | 有 | `getValue` / `getSelection` / `getActiveViewOfType` |
| `editor.write` | 有 | 有 | `replaceSelection`，Full 另有 `editor.setValue` |
| `network` | 无 | 有 | `requestUrl` |
| `events.app` | 无 | 有 | `registerEvent` / `registerDomEvent` / `registerInterval` |
| `ui` | 有 | 有 | `addRibbonIcon` / `addStatusBarItem` / `styles.css` |
| `dom` | 无 | 有 | `registerView` / `ItemView` / `containerEl` / DOM helpers |

`manifest.json` 里的 `xinkgo.capabilities` 只是给作者看的声明提示，Obsidian 会忽略它；
实际授权以 `main.js` 静态扫描 + 用户勾选 + 运行时校验为准。
`xinkgo-demo-basic` 故意不写这段提示，能力完全由扫描得出；`xinkgo-demo-full` 写上
11 项作为文档，并由测试校验提示与扫描结果一致。

## 不写进插件的能力

宿主能扫描出 14 项能力，但其中 3 项只是兼容占位，已从全面 demo 里移除：

| 能力 | 宿主实际行为 | 为什么不写 |
| --- | --- | --- |
| `metadata` | `getFileCache` / `getCache` 返回 `{}`，`metadata.get` 返回空 map，`metadataCache` 无事件源 | 没有真实的双链 / 标签 / 大纲元数据索引 |
| `editor.extension` | 扩展只登记进数组 | 编辑器不是 CodeMirror 6，没有可挂载的扩展点 |
| `compat.stub` | 非 `obsidian` 模块返回惰性 Proxy | 不接触 Node / Electron / 文件系统，功能不可用 |

同一原则下，下面这些 Obsidian API 也**没有**出现在 demo 里，因为宿主只做 no-op
或空壳：`addSettingTab` / `PluginSettingTab` / `new Setting(`、`Modal` /
`SuggestModal` / `FuzzySuggestModal` / `Menu` / `MenuItem`、`MarkdownRenderer` /
`HoverPopover`、Markdown 后处理器、`EditorSuggest`、协议 / hover / bases / CLI
注册、`secretStorage`、`addIcon` / `setIcon`、外部 `require("path")`。
设置请用 `loadData/saveData`，提示请用 `Notice`，视图请用 `registerView`。

`internal/plugin/demo_plugins_test.go` 里的 `TestDemoPlugins_useOnlySupportedAPIs`
把这份黑名单做成断言：两个 demo 的 `main.js` 一旦重新出现这些 API，测试就会失败。

## 写插件时的硬约束

`main.js` 必须是 CommonJS：`require("xinkgo")`（官方插件；外部兼容插件用
`require("obsidian")`，两者指向同一个模块对象）、`module.exports = 一个继承
xinkgo.Plugin 的类`。不能有 ESM 语法、相对导入、动态模块名；`manifest.json` 的 `id`
必须匹配 `^[a-z][a-z0-9-]*$` 且等于目录名，`version` / `minAppVersion` 必须是语义版本，
`minAppVersion` 不能超过宿主的 `APIProfile`（当前 `1.0.0`）。

这两个示例由 `internal/plugin/demo_plugins_test.go` 校验：包结构、无硬性扫描错误、
basic 的 C2 与 7 项能力、full 的 C3 与 11 项能力、manifest 声明与扫描一致、
能力黑名单，以及视图引用的静态资源。改动示例后跑：

```sh
go test ./internal/plugin -run DemoPlugins
make check
```
