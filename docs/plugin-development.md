# 插件开发指南

日期：2026-11-08。面向**插件作者**。本文用仓库里的两个示例当作贯穿案例：
[xinkgo-demo-basic](../examples/plugins/xinkgo-demo-basic/main.js) 是最小骨架，
[xinkgo-demo-full](../examples/plugins/xinkgo-demo-full/main.js) 是全力支持能力的清单。
安装与试用见 [plugin-demos.md](plugin-demos.md)，能力对照与准入清单见
[../examples/plugins/README.md](../examples/plugins/README.md)，宿主设计见
[design/obsidian-plugins.md](design/obsidian-plugins.md)。

官方插件库的准入契约、三层模型与迁移路线见
[design/xinkgo-plugin-library.md](design/xinkgo-plugin-library.md)；本文是
L1 插件的作者视角，契约的代码定义在其 §3 与 `internal/plugin/contract.go`。

## 1 边界与准入原则

Xinkgo 的第三方插件是 **Obsidian 兼容插件**，不是原生 Go 扩展：

- 每个启用的插件独占一个隔离 WebView 页。页面通过自定义 scheme 拿资源，
  Go 侧用 `/rpc` 桥和 `xinkgo-plugin-event` 事件与页面通信。
- 主窗口是 `mygo/ui` 原生 UI，编辑器不是 CodeMirror 6，预览不是 DOM，
  `Workspace` 不是 `WorkspaceLeaf`。插件的 DOM 只在自己那一页里有效。
- `obsidian` 模块由 [shim.js](../resources/obsidian-shim/shim.js) 提供，
  实现的是 Obsidian API 的一个子集；超出子集的部分是惰性 stub 或直接拒绝。
- 宿主**不编译插件代码**。放进包里的 `main.js` 必须已经是单文件 CommonJS
  bundle；TypeScript / ESM / 多文件源码要由作者在发布前打好包。

**准入原则（本仓库插件的硬要求）**：插件只使用 Xinkgo 有真实实现的能力。
宿主能扫描出 14 项能力，其中 11 项可以写进插件，3 项只是兼容占位，
**不因为 Obsidian 支持就写进来**。

| 可以写进插件（11 项） | 不要写进插件（3 项） |
| --- | --- |
| `data`、`commands`、`notice`、`vault.read`、`vault.write`、`editor.read`、`editor.write`、`network`、`events.app`、`ui`、`dom` | `metadata`（缓存是空对象、无事件源）、`editor.extension`（只登记、无 CM6）、`compat.stub`（外部模块是惰性 Proxy） |

同一原则下，一批 Obsidian API 虽然能被 shim 接住，但宿主只做 no-op 或空壳，
也**不要**出现在插件里：`addSettingTab` / `PluginSettingTab` / `new Setting(`、
`Modal` / `SuggestModal` / `FuzzySuggestModal` / `Menu` / `MenuItem`、
`MarkdownRenderer` / `HoverPopover`、Markdown 后处理器、`EditorSuggest`、
协议 / hover / bases / CLI 注册、`secretStorage`、`addIcon` / `setIcon`、
非 `obsidian` 的 `require(...)`。配置用 `loadData/saveData`，提示用 `Notice`，
视图用 `registerView`。

两个 demo 都由
[demo_plugins_test.go](../internal/plugin/demo_plugins_test.go) 的
`TestDemoPlugins_useOnlySupportedAPIs` 守着这份黑名单：一旦 `main.js` 里重新
出现这些 API，测试就失败。第 7 节给出完整的避坑清单与判定依据。

## 2 包结构与 manifest

一个插件包就是一个目录，**目录名必须等于 `id`**：

```text
<id>/
  manifest.json   # 必需
  main.js         # 必需
  styles.css      # 可选，带上就会被算作 ui 能力
  assets/         # 可选，只能经 /plugin/assets/<path> 读取
  data.json       # 运行时由 loadData/saveData 生成，不随包发布
```

宿主按三处发现，后者覆盖前者的同 id 包：

| 来源 | 路径 |
| --- | --- |
| 用户级 | `<appdata>/plugins/<id>/`（macOS：`~/Library/Application Support/Xinkgo/plugins/`） |
| Obsidian 兼容 | `<vault>/.obsidian/plugins/<id>/` |
| 库内原生 | `<vault>/.xink/plugins/<id>/` |

`manifest.json` 的必填字段与校验在
[internal/plugin/manifest.go](../internal/plugin/manifest.go)：

| 字段 | 规则 |
| --- | --- |
| `id` | `^[a-z][a-z0-9-]*$`，且等于目录名 |
| `name` / `description` / `author` | 非空 |
| `version` / `minAppVersion` | 语义版本 `x.y.z`；缺失的 `minAppVersion` 按 `0.0.0` 处理 |
| `authorUrl` / `fundingUrl` / `isDesktopOnly` | 可选，容忍未知字段 |
| `xinkgo.capabilities` | Xinkgo 自定义提示，Obsidian 忽略；**宿主不据此授权** |

`minAppVersion` 不能超过宿主的 `plugin.APIProfile`（当前 `1.0.0`），否则报告里
多一条 `apiProfile` issue 且 `Supported=false`。这不是 Obsidian 版本号，是
Xinkgo 模拟的 API profile。

两个案例的差别：

- [basic/manifest.json](../examples/plugins/xinkgo-demo-basic/manifest.json)
  只有必填字段，能力完全由扫描得出。
- [full/manifest.json](../examples/plugins/xinkgo-demo-full/manifest.json)
  写了 `xinkgo.capabilities` 的 11 项作为文档，且测试会断言这份提示与静态
  扫描结果完全一致，防止文档和代码各说各话。

## 3 加载链路：不编译，但必须是 CommonJS 产物

[host.html](../resources/obsidian-shim/host.html) 按序加载四个脚本：

```text
/config.js        -> window.__XINKGO__（插件 id、manifest、已授权能力、apiVersion、token）
/shim.js          -> window.module / exports / require，以及全部 obsidian 导出
/plugin/main.js   -> 你的 bundle，同步执行，把类写到 module.exports
/runtime.js       -> shim.boot()
```

`boot()` 的行为在 [runtime.js](../resources/obsidian-shim/runtime.js) 与
`shim.js` 末尾：

1. 取 `module.exports.default || module.exports`，不是函数就报
   `main.js has no default Plugin class`。
2. `new Klass(app, manifest)`，其中 `app` 是 shim 构造的 App 对象。
3. 拉 `/state` 建本地缓存，然后 `await plugin.onload()`。
4. 刷新命令可见性，POST `/rpc` 的 `ready`，Go 把状态改成 running。
5. 卸载时调 `plugin.onunload()`，再跑 `Component.unload()` 清理
   `register*`、事件、interval、DOM listener 与子 Component。

`window.require` 是严格解析，不是 Node 的模块系统：

| 写法 | 结果 |
| --- | --- |
| `require("obsidian")` | 返回 shim 导出；未知名属性走惰性 Proxy |
| 相对导入 / 动态模块名 | 宿主**不解析**，扫描记 `relativeImport` / `dynamicRequire` |
| 其他模块名（含 `fs`、`path`、`electron`、`@codemirror/*`） | 扫描记 `stubImport` / `unknownImport`；运行时要么抛 `XinkgoCapabilityError`，要么返回惰性 Proxy。**不要写进插件** |

结论：**所有代码都要打进 `main.js`，只把 `obsidian` 留在 external**。这与
Obsidian 官方样例不同（它还会把 `electron`、`@codemirror/*` 和 Node
builtinModules 标为 external，因为 Electron 能解析）。一个合适的 esbuild 配置：

```js
// build.mjs —— 作者侧执行，产出 main.js
import * as esbuild from "esbuild";

await esbuild.build({
  entryPoints: ["src/main.ts"],
  bundle: true,
  format: "cjs",
  platform: "browser",
  target: "es2021",
  outfile: "main.js",
  external: ["obsidian"], // 只留这一个
});
```

静态扫描的硬停止只有两类（见
[internal/plugin/scan.go](../internal/plugin/scan.go)）：ESM 语法
（`esmSyntax`）和没有 `module.exports` 类导出（`missingDefaultExport`），
manifest 非法会判 C0 不支持，`minAppVersion` 过新则是 `Supported=false` 但
等级不变。其余 API 都只是把兼容等级抬到 C3/C4 并加一条 `shimAPI` 注解，
`shimAPI` **不是失败**。

一个容易踩的坑：扫描是纯文本 `strings.Contains`。注释里出现
`registerView` 之类也会命中能力规则。写注释时别把 API 名当普通词随手带上，
否则报告里会多出你没打算要的能力。

## 4 最小插件：逐段读 xinkgo-demo-basic

[basic/main.js](../examples/plugins/xinkgo-demo-basic/main.js) 全部只有三件套：

```js
var obsidian = require("obsidian");

var DemoBasic = (function () {
  function DemoBasic(app, manifest) {
    obsidian.Plugin.call(this, app, manifest);
  }
  DemoBasic.prototype = Object.create(obsidian.Plugin.prototype);
  DemoBasic.prototype.constructor = DemoBasic;
  // ...
  return DemoBasic;
})();

module.exports = DemoBasic;
```

`onload()` 里按这个顺序做事，可以直接照抄成新插件的骨架：

1. `this.loadData()` 恢复状态（返回 Promise，`onload` 可以返回 Promise）。
2. `this.addStatusBarItem()`，`setText()` 更新状态栏。
3. `this.addCommand({...})` 注册若干命令。
4. `this.addRibbonIcon(icon, title, callback)` 注册左侧图标。
5. `this.registerEvent(this.app.workspace.on("file-open", cb))`，卸载时自动解绑。
6. `new obsidian.Notice(...)` 给一条加载提示。

命令有四种回调形态，宿主在 `command.add` 和状态刷新时都会用 `checking=true`
问一次可见性，所以 **checking 分支不能有副作用**：

| 形态 | 签名 | 用途 |
| --- | --- | --- |
| `callback` | `()` | 普通命令 |
| `editorCallback` | `(editor, view)` | 需要编辑器，宿主自动传 Editor |
| `editorCheckCallback` | `(checking, editor, view)` | 有选区才显示，见 basic 的 `selection-only` |
| `checkCallback` | `(checking)` | 有打开笔记才显示，见 full 的 `vault-lifecycle` |

`addCommand` 会把 ID 变成 `<pluginID>:<localID>`，名称前缀插件名；命令面板里
看到的是 `Xinkgo Demo: Basic: Say hello` 这种形式。

**从零做一个新插件的六步**：

1. 复制 `xinkgo-demo-basic/` 到用户级插件目录，目录改名成你的 `id`。
2. 改 `manifest.json` 的 `id` / `name` / `version` / `description`，
   确认 `minAppVersion` 不超过 `1.0.0`。
3. 改 `main.js` 里的类名与命令，先只留一个 `callback` 命令。
4. 设置窗口 ▸ **Community plugins** ▸ **Installed** 点 **Refresh**。
5. 展开 Grants 勾上这个命令需要的能力（见第 5 节），**再**打开开关。
6. 命令面板里跑一次，看 Notice、状态栏和日志。之后每次改 `main.js` 都点
   **Reload**，改 `manifest.json` 尤其要重载。

## 5 能力、授权与兼容等级

报告里的 `Capabilities` 来自 `ScanSource` 对 `main.js` 的文本匹配，共 14 项
（`plugin.AllCapabilities`）。用户仍要在 Grants 里逐项授权，运行时才会放行。
下表是**准入清单**：前 11 项可以写进插件，后 3 项不要写。

| 能力 | 触发它的典型 API | 运行时校验 | 准入 |
| --- | --- | --- | --- |
| `data` | `loadData` / `saveData` | 启用即有 | 用 |
| `commands` | `addCommand` / `removeCommand` | 启用即有 | 用 |
| `notice` | `new Notice` | 启用即有 | 用 |
| `ui` | `addRibbonIcon` / `addStatusBarItem`；随包带 `styles.css` 也会自动加上 | `status.add` / `status.set` / `action.add` 等 RPC | 用 |
| `vault.read` | `vault.read` / `cachedRead` / `adapter.read` / `workspace.openLinkText` | 所有 `vault.*` 读与 `workspace.open` RPC | 用 |
| `vault.write` | `vault.create` / `modify` / `append` / `rename` / `delete` / `createFolder` | 所有 `vault.*` 写 RPC | 用 |
| `editor.read` | `getValue` / `getSelection` / `getActiveViewOfType` | 本地 `/state` 缓存，无 RPC 校验 | 用 |
| `editor.write` | `replaceSelection` / `editor.setValue` | `editor.write` RPC | 用 |
| `network` | `requestUrl` | `network.request` RPC | 用 |
| `events.app` | `registerEvent` / `registerDomEvent` / `registerInterval` | 本地事件注册，无 RPC 校验 | 用 |
| `dom` | `registerView` / `ItemView` / `containerEl` | `view.register` RPC | 用 |
| `metadata` | `MetadataCache` / `getFileCache` / `getFirstLinkpathDest` | `metadata.get` 返回空 map | **不用** |
| `editor.extension` | `registerEditorExtension` / `EditorSuggest` | 只登记，无 CM6 编辑器 | **不用** |
| `compat.stub` | 非 `obsidian` 的 `require(...)` | shim 的 `granted("compat.stub")` | **不用** |

注意：**报告里的能力不等于每一项都有运行时 RPC 校验**（`editor.read`、
`events.app` 是本地行为）；**带 `styles.css` 就会自动多一个 `ui` 能力**，
哪怕 CSS 里什么都没写。

等级由命中的最高规则决定（`internal/plugin/plugin.go`）：

| 等级 | 含义 | 降级 |
| --- | --- | --- |
| C1 `LevelCore` | 只用命令 / Notice / 数据 | 否 |
| C2 `LevelNative` | 用到原生适配器（ribbon、Vault、Editor、network） | 否 |
| C3 `LevelDOM` | 需要插件自己的 DOM 表面 | 是 |
| C4 `LevelEditor` | 需要 CM6 / 外部模块 stub | 是 |

按准入原则写的插件**最高只会到 C3**。`Degraded=true` 在这里的含义是
「视图运行在插件自己的 WebView 页，不是原生 UI 面板」，不是功能缺失：
`registerView` 的视图由宿主真实实现、能点能交互。C4 意味着引入了本仓库
禁止的能力，应当修掉而不是接受。

两个案例的扫描结果由
[demo_plugins_test.go](../internal/plugin/demo_plugins_test.go) 固定：

- basic 命中 7 项能力、全是 C1/C2 规则 → `C2` / `Supported=true` /
  `Degraded=false`，`Imports` 只有 `obsidian`。
- full 命中 11 项受支持能力，其中 `registerView` 把等级抬到 `LevelDOM` →
  `C3` / `Supported=true` / `Degraded=true`，`Imports` 仍只有 `obsidian`，
  没有 `stubImport` issue。

**授权时机是最常见的坑**。ribbon、状态栏和视图都在 `onload()` 里通过
`fire()` 注册，当时能力没授权，那次 RPC 就被拒绝且不会重放；事后在 Grants
里勾上也补不回来。所以要么先勾授权再启用，要么勾完点一次 **Reload**。
另外从 **Browse** 一键安装时宿主会自动把扫描出的能力**全部写入授权**，
手动拷目录则完全不写，两种路径的差别见 [plugin-demos.md](plugin-demos.md)。

## 6 真实能力的 API 映射：以 full 为案例

[full/main.js](../examples/plugins/xinkgo-demo-full/main.js) 的 11 条命令基本
一一对应一组真实能力。下面按类别给出可用 API，写自己的插件时可以直接查。

### 6.1 数据、命令、Notice

`loadData()` / `saveData(data)` 走 `data.load` / `data.save`，是唯一需要落盘的
配置手段。`data.json` 写在**插件包所在目录**：用户级安装时是
`<appdata>/plugins/<id>/data.json`；放进库内 `.obsidian/plugins/` 时也会写进
库里，私密配置要留意。`addCommand` / `removeCommand` 见 full 的
`toggleCommand()`，它演示了运行期增删一条 `Temporary command`。
`new Notice(text, duration)` 会同时弹原生提示；`Notice.setMessage()` /
`hide()` 也可用。

### 6.2 Vault

shim 的 `Vault` 用 `/state` 里的 `files` 做同步查询，用 RPC 做读写：

| 类别 | API |
| --- | --- |
| 同步（本地） | `getRoot` / `getAbstractFileByPath` / `getFileByPath` / `getFolderByPath` / `getName` / `getResourcePath` |
| 异步读（需 `vault.read`） | `read` / `cachedRead` / `readBinary` |
| 异步写（需 `vault.write`） | `create` / `createBinary` / `createFolder` / `modify` / `modifyBinary` / `append` / `delete` / `trash` / `rename` |
| `adapter`（同一套 RPC） | `exists` / `stat` / `list` / `read` / `readBinary` / `write` / `writeBinary` / `append` / `mkdir` / `rmdir` / `remove` / `rename` / `trash` / `getResourcePath` |

`read` / `cachedRead` / `adapter.read` 都接受 `TFile` 或路径字符串。二进制走
base64：`readBinary` 返回 `ArrayBuffer`，写接口接受 `ArrayBuffer` 或 base64
字符串。路径按 `normalizePath` 语义处理（正斜杠、去掉首尾 `/`、折叠 `.`、拒绝
`..` 和绝对路径），Go 侧再用 `filepath.Rel` 确认仍在库根内。

full 的 `readNote()` 用三种读法读同一个文件，`vaultLifecycle()` 依次演示
`createFolder` → `create` → `modify` → `cachedRead` → `append` → `rename` →
`delete`。注意生命周期最后会把文件删掉，别在真实笔记上试。

### 6.3 Editor

`Editor` 是 `/state` 里 `editorText` + 选区/光标偏移的一层封装，位置用
`{line, ch}` 与字符偏移互相换算（`posToOffset` / `offsetToPos`）：

| 类别 | API |
| --- | --- |
| 读（本地） | `getValue` / `getLine` / `lineCount` / `lastLine` / `getCursor` / `getSelection` / `listSelections` / `getRange` / `somethingSelected` |
| 写（需 `editor.write`） | `setValue` / `setLine` / `setCursor` / `setSelection` / `replaceRange` / `replaceSelection` / `transaction` |

写操作最终都汇到 `fire("editor.write", {text, selectionStart, selectionEnd})`，
Go 侧更新编辑器并回发 `editor-change`。它是 `fire()`，即发后不管：宿主拒绝时
只会记一行日志，不会 reject 到调用方。所以**不要依赖写操作的返回值判断成功**，
要看授权和日志。

basic 的 `countWords()` 用 `getValue()`，`insert-date` 用
`replaceSelection()`；full 另有 `replace-buffer` 演示 `editor.setValue()`。

### 6.4 Workspace 与打开笔记

`workspace.getActiveFile()` / `getActiveViewOfType(MarkdownView)` /
`activeEditor` 都是本地的，依赖 `state.hasActiveNote`。`openLinkText(linktext)`
走 `workspace.open` RPC（需要 `vault.read`），会真的在原生窗口里打开笔记并发
`file-open` 事件。full 的 `openNoteByLink()` 就是这条命令。

`getLeaf()` 返回轻量 `WorkspaceLeaf` stub、`changeLayout()` 直接抛
`XinkgoCapabilityError`、`getLayout()` 返回 `{}`：这些属于第 7 节的避坑清单，
不要依赖它们做布局。

### 6.5 requestUrl

`requestUrl({url, method, headers, body, throw})` 走 `network.request`，需要
`network` 授权。Go 侧限制：只接受 `http` / `https`、URL 里不能带 userinfo、
15 秒超时、响应体最多 1 MiB、默认 `throw=true` 时状态码 `>= 400` 会 reject。
`body` 支持字符串或 `ArrayBuffer`（转 base64）。

返回对象有 `status` / `headers` / `arrayBuffer` / `json` / `text`，后三个是
Promise（真实 Obsidian 里 `text` 是字符串）。兼容两边的写法是
`Promise.resolve(response.text).then(...)`，full 的 `requestSample()` 就是这么
写的。宿主在 [shim.js](../resources/obsidian-shim/shim.js) 里给这三个便捷分支
挂了 rejection handler，所以请求失败时不会冒出未处理的 rejection 把插件标记成
failed。

**不要在 `onload()` 里自动联网**。full 把 `requestUrl` 放在一条显式命令里。

### 6.6 原生 UI 与图标

`addRibbonIcon(icon, title, callback)` 与 `addStatusBarItem()` 是**真实的**原生
注册（需要 `ui`）：前者进主窗口左侧插件 ribbon，后者进状态栏。
原生 ribbon 只认 `calendar-days` / `layout-template` / `globe` 三个图标名，
其余一律回落到通用插件图标，tooltip 用 `addRibbonIcon` 的 `title`。
basic 用的 `calendar-days` 有专用图标，full 用的 `layers` / `puzzle` 会回落。

`window` 上还有一批 Obsidian 风格的 DOM helper，装在
`HTMLElement.prototype` / `Element.prototype` / `DocumentFragment.prototype`：
`setText` / `empty` / `createEl` / `createDiv` / `createSpan` / `addClass` /
`removeClass` / `toggleClass` / `setAttr` / `setAttrs` / `appendText` /
`detach` / `find` / `findAll`，以及全局 `createFragment()`、
`activeDocument`、`activeWindow`。

**注意**：`activeDocument` / `activeWindow` 挂在全局 `window` 上，不在
`obsidian` 导出里。写 `obsidian.activeDocument` 拿到的是惰性 Proxy，不是真的
`document`；full 用的是 `window.activeDocument`。

其他真实可用的导出：`moment`（只实现 `format` 的 `YYYY/MM/DD/HH/mm/ss`、
`add`、`startOf` 等子集）、`normalizePath` / `getLinkpath` /
`sanitizeHTMLToDom` / `htmlToMarkdown` / `arrayBufferToBase64` /
`base64ToArrayBuffer` / `prepareSimpleSearch` / `getFrontMatterInfo` /
`getLanguage` / `apiVersion` / `requireApiVersion` / `Platform`；
`App` 侧有 `isDarkMode()` / `loadLocalStorage()` / `saveLocalStorage()` /
`keymap` / `scope`。`FileManager.generateMarkdownLink()` 是纯字符串拼接，
不依赖元数据索引，可用。

### 6.7 事件

`Component.registerEvent(ref)` 保证卸载时自动 `off()`；`registerDomEvent(el,
type, cb)` 和 `registerInterval(id)` 同理。宿主实际会发的具名事件：

| 事件名 | 触发源 | shim 的投递目标 |
| --- | --- | --- |
| `file-open` | 打开笔记、`workspace.openLinkText` | `workspace` |
| `editor-change` | `editor.write` RPC | `workspace` |
| `vault-open` | 切换库 | `workspace` |
| `vault.modify` / `vault.create` / `vault.delete` / `vault.rename` | 对应 Vault 写 RPC | `vault`（去掉前缀） |

每次 `updateState()` 还会发一个 `stateChanged` 控制消息，shim 收到后刷新
`/state` 并重算命令可见性；具名事件额外触发一次 `workspace.trigger("stateChanged")`。

两个限制：事件回调**不带文件对象**（Go 的事件消息只有 `Name`），把它当信号用，
要拿路径请调 `workspace.getActiveFile()`；`metadata.*` 分支在 shim 里存在，
但当前宿主没有对应事件源，full 已移除这类监听。

full 的 `registerAppEvents()` 监听 `file-open`、`vault.modify`、
`editor-change`，再加一个真实的 DOM click 与 interval。

### 6.8 插件自己的 DOM 表面

`registerView(viewType, creator)` 需要 `dom` 授权，宿主把视图列进主窗口右侧的
**插件视图抽屉**；用户选中它时才会调用 `creator(leaf)`，构造
`WorkspaceLeaf` stub，把 `view.containerEl` 挂进页面的
`#xinkgo-plugin-root`，然后调 `view.onOpen()` / `onResize()`。一个插件一次只有
一个可见表面，切换视图会先 `closeView()` 调 `onClose()`。

`ItemView` 提供 `containerEl`（带 `contentEl`）、`getViewType` /
`getDisplayText` / `getIcon` / `getState` / `setState` / `onOpen` / `onClose` /
`onResize`。full 的 `DemoView` 用 `contentEl.createEl()` 建了标题、段落、按钮
和一张图片，按钮点击往 DOM 里加行 —— 这是插件页里**真实**的 DOM，不是 stub。

## 7 不要写进插件的 API：避坑清单

这一节列出的 API 在 shim 里都「能调用」，但宿主没有真实实现。把它们写进
插件，只会得到一个能被扫描出来、却没有效果的能力，并且会把报告抬到 C4。
两个 demo 已全部移除，黑名单由测试固定。

| API | 宿主实际行为 |
| --- | --- |
| `addSettingTab(new PluginSettingTab(app, plugin))` | 立刻调用 `tab.display()`，但只画在插件 WebView 页，**不进 Xinkgo 原生设置窗口** |
| `new Setting(containerEl)` | 创建 DOM 结构，控件是惰性对象，只有 `setValue` / `onChange` / `setIcon` / `setCta` 等有限方法，没有 `onClick` / `setButtonText` |
| `Modal` / `SuggestModal` / `FuzzySuggestModal` | DOM 节点，但 `open()` 挂到插件页 `document.body`，`getSuggestions` 等返回空 |
| `Menu` / `MenuItem` | 本地对象，`showAtMouseEvent` 是 no-op |
| `MarkdownRenderer.renderMarkdown()` | `Promise.resolve()`，不渲染、不改原生预览 |
| `HoverPopover` | 只有容器节点 |
| `registerMarkdownPostProcessor` / `registerMarkdownCodeBlockProcessor` | 只登记，宿主不调用 |
| `registerEditorExtension` / `registerEditorSuggest` / `EditorSuggest` | 只登记进数组，没有 CodeMirror 6 编辑器可挂载 |
| `registerObsidianProtocolHandler` / `registerExtensions` / `registerHoverLinkSource` / `registerBasesView` / `registerCliHandler` | no-op |
| `app.secretStorage` | `getSecret()` 返回 `null`，其余返回 resolved Promise，不碰系统钥匙串 |
| `addIcon` / `setIcon` / `getIcon` | 只在插件页注册 SVG，不是全局图标 |
| `loadMermaid()` / `loadMathJax()` | 返回惰性 stub |
| `MetadataCache.getFileCache` / `getCache` / `getFirstLinkpathDest` | `{}` 或按文件名在 `state.files` 里找，没有双链 / 标签 / 大纲索引 |
| `FileManager.processFrontMatter()` | 读取原文但不解析 frontmatter，回调拿到空对象 |
| `workspace.getLeaf()` / `changeLayout()` / `getLayout()` | 轻量 stub / 抛异常 / `{}` |
| 非 `obsidian` 的 `require(...)` | 惰性 Proxy 或 `XinkgoCapabilityError`，不接触文件系统、Node 或 Electron |

判定方法很简单：看宿主有没有真实实现、用户能不能观察到真实效果。
命令、Notice、ribbon、状态栏、Vault / Editor 读写、`openLinkText`、
`requestUrl`、事件、`registerView` 都有；上表这些没有。配置用
`loadData/saveData`，提示用 `Notice`，视图用 `registerView`。

## 8 CSS、静态资源与 data.json

`styles.css` 由 `/plugin/styles.css` 提供。宿主只做两件事：删掉 `@import`，
把 `url(https?://...)` 替换成 `none`；**没有**加 `[data-plugin-id]` 作用域前缀，
隔离靠的是「一插件一页」。所以 CSS 里可以用 `var(--text-normal, #20242b)`
这种带回退的写法，但别指望给原生 UI、编辑器或普通预览上色。宿主没有承诺
向插件页注入 `--text-*` 等主题变量，所以**务必写回退值**。full 的
[styles.css](../examples/plugins/xinkgo-demo-full/styles.css) 是可照抄的例子。

静态资源只能从 `/plugin/assets/<path>` 取：`pluginFilePath()` 会拒绝空路径、
NUL、`..` 穿越和逃出插件目录的路径，`servePluginAsset()` 还有 4 MiB 上限，
Content-Type 由内容探测。full 的 `assets/badge.svg` 在 `DemoView.onOpen()`
里用 `createEl("img")` + `setAttr("src", "/plugin/assets/badge.svg")` 加载。
`vault.getResourcePath(file)` 生成的是 `/vault/<path>`，读取需要 `vault.read`。

`data.json` 的落点见 6.1。`onExternalSettingsChange()` 在 shim 里有定义，
但当前宿主没有文件监视器去触发它，改配置请走 Reload。

## 9 编码规范

1. **先查准入清单**。第 5 节的 11 项能力、第 7 节的避坑清单就是边界；
   不要因为 `shim.js` 里有个同名导出就认为能力可用。
2. **所有会走 RPC 的 Promise 都带 `.catch()`**。未授权时 `/rpc` 返回
   `{ok:false}`，`rpc()` 抛出，没接住就变成未处理 rejection，宿主会把插件
   标成 failed。`vaultLifecycle()`、`readNote()`、`openNoteByLink()`、
   `requestSample()` 末尾都有兜底，并用 `Notice` + `console.error` 说明原因。
3. **不引入外部模块**，也不在 `onload()` 顶层做联网。full 把 `requestUrl`
   放在显式命令里，开机绝不联网。
4. **事件只当信号**，需要路径调 `workspace.getActiveFile()`；卸载靠
   `Component.register*` 自动清理。
5. 日志走 `console.log`（会转成宿主日志），面向用户的失败走 `Notice`，
   参照 full 的 `log()` / `fail()` 两个 helper。

`internal/plugin/demo_plugins_test.go` 里的
`TestDemoPlugins_useOnlySupportedAPIs` 把避坑清单里曾经写进 demo 的条目做成
字符串黑名单，对两个 demo 的 `main.js` 逐个断言；第 7 节是更完整的知识清单。
你自己的插件也建议加同一份检查。

## 10 调试与验证

| 手段 | 看什么 |
| --- | --- |
| 插件管理器 ▸ Installed 卡片 | `Level`（C1~C4）、`Supported` / `Degraded`、`Capabilities`、`Imports`、`APIs`、`Issues` |
| 日志 `msg="plugin console"` | 插件 `console.log` / `console.error`，带 `id` 与 `text` |
| **Refresh** | 重新扫描三个发现目录，识别新包 / 改过的 manifest |
| **Reload** | 卸载并重建实例与 WebView，重新执行 `main.js` |
| `XINKGO_LOG_LEVEL=debug` | 提高宿主日志级别 |

日志位置：开发版（`make run` / `MYGO_ENV != production`）在终端 stderr；
安装版在 `~/Library/Logs/Xinkgo/xinkgo.log`（Windows：
`%AppData%\Xinkgo\logs\xinkgo.log`）。

常见报错对照：

| 报错 | 原因 |
| --- | --- |
| `esmSyntax` / `missingDefaultExport` | 包里是 ESM 或没有 `module.exports`，扫描判 C0，安装被拒绝 |
| `apiProfile` | `minAppVersion` 超过宿主 `APIProfile` |
| `badManifest` | 必填字段缺失、`id` 不合法、目录名 ≠ `id` |
| `main.js has no default Plugin class` | `module.exports` 不是函数 |
| `X is not granted` | 对应能力没勾，RPC 被拒 |
| `duplicateID` | 三个发现目录里出现了同 id 包 |
| `stubImport` / `unsupported module` | 引入了外部模块，按准入清单应当删掉 |

本仓库给两个 demo 做了静态门禁
[demo_plugins_test.go](../internal/plugin/demo_plugins_test.go)，可以直接照着
给 `examples/` 里的插件加测试：

```go
func TestMyPlugin_scan(t *testing.T) {
	m, err := plugin.ParseManifest(manifestData)
	if err != nil {
		t.Fatal(err)
	}
	if err := plugin.ValidateManifest(m, "my-plugin"); err != nil {
		t.Fatal(err)
	}
	r := plugin.ScanPackage(m, mainJS, dir, "examples", hasCSS)
	if !r.Supported || r.Level != plugin.LevelNative {
		t.Fatalf("report = %+v", r)
	}
	for _, cap := range []plugin.Capability{
		plugin.CapabilityCommands, plugin.CapabilityVaultRead,
	} {
		if !r.HasCapability(cap) {
			t.Errorf("missing %s", cap)
		}
	}
}
```

注意 `internal/plugin` 是 Go 的 internal 包，**只有本仓库能 import**；外部插件
作者无法直接复用。仓库外至少做两件事：`node --check main.js` 验语法，以及
安装后核对管理器里的等级 / 能力 / Issues。运行期行为可以用一个最小 DOM
沙箱加载 `shim.js` + `main.js`（本仓库验证两个 demo 时就是这么做的）。

## 11 发布

1. 源码用 TypeScript 或 JS 都行，构建成**单个** `main.js`（第 3 节的 esbuild
   配置），只把 `obsidian` 留 external。
2. 手写或从构建产出 `styles.css`，把 `assets/` 一起放进包目录。
3. `manifest.json` 的 `id` 与目录名一致，`version` / `minAppVersion` 合法；
   `xinkgo.capabilities` 只写准入清单里的能力。
4. GitHub release 放 `manifest.json`、`main.js`、`styles.css` 三个 asset；
   宿主的 `install()` 从 `releases/latest/download/<asset>` 下载。**不需要**
   `versions.json`，也不需要把 `data.json` 打进包。
5. 想出现在 Browse 的「已检查」清单里，需要进
   `resources/obsidian-plugins/compatible.json`；用户也可以在
   `<appdata>/plugins/compatible.json` 覆盖。

发布前自查：`main.js` 里没有 ESM 语法、相对导入、动态 `require` 或黑名单 API；
报告等级不高于 C3；所有 RPC Promise 都有 `.catch()`；`onload()` 不做联网；
扫描出的能力清单与 manifest / README 一致。

## 12 两个 demo 的案例索引

| 主题 | basic 里的位置 | full 里的位置 |
| --- | --- | --- |
| CommonJS 骨架 | `var DemoBasic` / `module.exports` | `var DemoFull` / `module.exports` |
| 生命周期与持久化 | `onload` / `onunload` / `record` | `onload` / `onunload` / `record` |
| 四种命令回调 | `hello`、`insert-date`、`selection-only` | `tour`、`insert-summary`、`selection-stats`、`vault-lifecycle` |
| 原生 UI | `addStatusBarItem` / `addRibbonIcon` | 同左 |
| Vault 读 | `readNote` | `readNote` 的三个适配器 |
| Vault 写 | 无 | `vaultLifecycle` |
| Editor 读写 | `countWords` / `insert-date` | `readNote` / `insert-summary` / `replace-buffer` |
| 打开笔记 | 无 | `openNoteByLink` / `workspace.openLinkText` |
| 网络 | 无 | `requestSample` |
| 事件 | `file-open` | `registerAppEvents` |
| DOM 表面与资源 | 无 | `DemoView` / `registerViewSurface` / `assets/badge.svg` |
| CSS | 最小样式 | `styles.css` |
| 避坑清单里的 API | 无 | 无（已全部移除，由测试固定） |

建议阅读顺序：先通读 basic 的 150 余行，跑一遍命令；再按第 6 节的顺序读 full
的各个方法；最后改一个自己的 `id` 和命令，把 basic 复制成新插件。

## 13 参考

- [plugin-demos.md](plugin-demos.md)：安装、授权、逐条命令预期与排错
- [../examples/plugins/README.md](../examples/plugins/README.md)：能力对照、
  准入原则与「不写进插件的能力」
- [design/obsidian-plugins.md](design/obsidian-plugins.md)：宿主架构、权限与
  API 映射的调研口径
- [../resources/obsidian-shim/shim.js](../resources/obsidian-shim/shim.js)、
  [host.html](../resources/obsidian-shim/host.html)、
  [runtime.js](../resources/obsidian-shim/runtime.js)：运行时的实际行为
- [../internal/plugin/scan.go](../internal/plugin/scan.go)、
  [plugin.go](../internal/plugin/plugin.go)、
  [manifest.go](../internal/plugin/manifest.go)：扫描规则、等级与能力定义
- [../internal/app/plugin_rpc.go](../internal/app/plugin_rpc.go)、
  [plugin_host.go](../internal/app/plugin_host.go)：RPC 方法与运行时校验
