# Obsidian 插件兼容宿主

日期：2026-11-04。状态：**P0 spike 与降级兼容扩展已实现，完整 P1/P2/P3 未实现**。
本文先定边界、架构与路线，再记录 P0 的实际落点；完全兼容 Obsidian 仍不是本期目标。

参考的同类扩展设计：[wechat.md](wechat.md)（外部样式目录 + 元数据 + 发现规则）、
[wechat-style-guide.md](wechat-style-guide.md)（面向扩展作者的指引）。项目分层见
[ARCHITECTURE.md](../ARCHITECTURE.md)，人工验收统一登记在
[ACCEPTANCE.md](../ACCEPTANCE.md)。

## 1 结论先行

Xinkgo **可以做“Obsidian 兼容插件宿主”，但不能诚实地承诺“所有 Obsidian 插件
直接兼容”**。原因不是缺一两个 API，而是应用形态不同：

- Obsidian 插件运行在 Electron 的 DOM workspace 里，`obsidian` 模块返回的是
  真实 DOM、WorkspaceLeaf、MarkdownView、CodeMirror 6 和 Node/Electron 能力。
- Xinkgo 主窗口是 `mygo/ui` 原生 UI，编辑器不是 CodeMirror 6，预览不是 DOM，
  `Workspace` 也不是 `WorkspaceLeaf`。官方 `obsidian.d.ts` 本次拉取的 master 版
  有 302 个顶层导出、103 个 class、112 个 interface、108 处 `HTMLElement` 引用。
- 官方明确说明社区插件没有被可靠地限制权限：可访问本机文件、可联网、可安装
  额外程序，默认是 Restricted Mode，用户要手工信任并启用。见
  [Plugin security](https://help.obsidian.md/Extending+Obsidian/Plugin+security)。

因此推荐 A 方案：在 Xinkgo 内实现一个**有边界的兼容宿主**，让一批只依赖核心
API 的插件不用改代码即可加载；用兼容性报告明确告诉用户哪些插件可用、哪些
降级、哪些不支持。B 方案是完整复刻 Obsidian 应用壳：把主窗口换成 WebView +
CodeMirror 6 + DOM workspace，等于重写 Xinkgo，不建议作为本期目标。

本文的“直接兼容”口径：

1. 目录布局兼容：`.obsidian/plugins/<id>/` 里的 `manifest.json`、`main.js`、
   `styles.css`、`data.json` 可被发现，不用先复制到 Xinkgo 专用目录。
2. 加载方式兼容：CommonJS `main.js` 通过 `require('obsidian')` 拿到 shim，
   `export default class X extends Plugin` 被实例化并调用 `onload()`。
3. 生命周期兼容：`onload/onunload`、`Component.register*`、`loadData/saveData`、
   `onUserEnable`、`onExternalSettingsChange` 有对应语义。
4. 核心 API 兼容：命令、Notice、状态栏、Vault 读写、当前笔记与 Editor 子集、
   MetadataCache 子集、`requestUrl`、设置数据等。
5. 不兼容：DOM workspace、CM6 编辑器扩展、任意 Node/Electron、
   作用于原生 UI 的 `styles.css`。这些必须报出来，不做静默 no-op。

## 2 调研事实

### 2.1 插件包与构建

官方样例仓库 [obsidian-sample-plugin](https://github.com/obsidianmd/obsidian-sample-plugin)
与开发者文档确认了插件的物理形态：

| 文件 | 作用 |
|---|---|
| `manifest.json` | `id` / `name` / `version` / `minAppVersion` / `description` / `author` 必填；`authorUrl` / `fundingUrl` / `isDesktopOnly` 可选字段也要容忍 |
| `main.js` | esbuild 打出的 CommonJS bundle，`format: 'cjs'`、`target: 'es2021'`、`outfile: 'main.js'` |
| `styles.css` | 可选，Obsidian 会注入应用 DOM |
| `data.json` | 在 Obsidian 里是 `Plugin.loadData/saveData` 的默认落点，位于插件目录内；Xinkgo 会改写到应用数据目录 |
| `versions.json` | 发布用映射，宿主不需要 |

样例 `esbuild.config.mjs` 把 `obsidian`、`electron`、8 个 `@codemirror/*`、
3 个 `@lezer/*` 以及 `builtinModules` 标为 external。也就是说 `main.js` 运行时
会 `require('obsidian')`，也可能 `require('electron')`、`require('fs')` 或
`require('@codemirror/state')`。宿主必须自己提供模块解析，不能把 `main.js`
当普通浏览器 ES module 加载。

官方 [Manifest](https://docs.obsidian.md/Reference/Manifest) 还要求 `id` 只用
小写字母和连字符、不能以 `plugin` 结尾、不能包含 `obsidian`，且本地开发时
目录名与 `id` 一致。`obsidian.d.ts` 的 `PluginManifest.dir` 是运行时注入的
库内相对路径，不是插件作者写在文件里的字段。

### 2.2 生命周期与 API 面

`Plugin extends Component`，关键入口是 `onload(): void | Promise<void>`、
`onunload()`、`loadData()`、`saveData()`、`addCommand()`、`removeCommand()`、
`addRibbonIcon()`、`addStatusBarItem()`、`addSettingTab()`、`registerView()`、
`registerMarkdownPostProcessor()`、`registerMarkdownCodeBlockProcessor()`、
`registerEditorExtension()`、`registerEditorSuggest()`、
`registerObsidianProtocolHandler()` 等。`Component` 负责
`register/registerEvent/registerDomEvent/registerInterval/addChild/removeChild`
的卸载清理。

`App` 暴露 `vault / workspace / metadataCache / fileManager / keymap / scope /
renderContext / secretStorage`；`Vault` 既有同步方法
（`getAbstractFileByPath/getRoot/getFileByPath`），也有异步方法
（`read/cachedRead/modify/create/delete/rename`）；`Workspace` 与
`MarkdownView/Editor` 大量依赖 DOM 与 CM6。官方
[Build a plugin](https://docs.obsidian.md/Plugins/Getting+started/Build+a+plugin)
说明插件需要用户手工启用，改 `manifest.json` 后要重载；
[Anatomy of a plugin](https://github.com/obsidianmd/obsidian-developer-docs/blob/master/en/Plugins/Getting%20started/Anatomy%20of%20a%20plugin.md)
说明 `onload/onunload` 分别是获取资源和释放资源的时机。

### 2.3 MyGo 提供的宿主能力

MyGo v0.7.4 的 `docs/frontend.md` 已经给出“插件 UI”的推荐做法：注册自定义
`plugin://` scheme，WebView 用 `URL: "plugin://" + plugin.ID + "/"` 加
`PageOptions{Untrusted:true}`，Go 侧用 `mygo.CallerPage(r.Context())` 认出
是哪一页在请求，用 `Event.EmitPage()` 把消息送回那一页。非信任页：

- 不能调用任何已绑定 Go 方法，也不能用 `window.mygo.window`；
- 收不到 `Broadcast` 或发给宿主窗口的 `Emit`，只能收 `EmitPage`；
- 默认拿不到权限、不能开新窗口/下载、不能导航到别的 origin；
- 仍能读取自己 scheme 的响应，正好用来做插件桥。

MyGo 另外提供 `mygo.NewEvent[T]()`、`mygo.NewEphemeralSession()`、
`Page.SetWindowOpenHandler`、`Page.OnWillNavigate`、
`Page.SetPermissionHandler`、`Page.OnWillDownload`、
`PageOptions.ContentRules`、`PageOptions.BackgroundThrottling`、
`Page.EvalContext`、`WebView.Destroy`、`mygo.WebViewAvailable()`。随包资源
放在项目的 `resources/`，运行时用 `mygo.App.Path(mygo.PathResources)` 取；
`resources/obsidian-shim/` 与 `resources/obsidian-plugins/compatible.json`
已随 P0 落地，打包时需确认它们进了各平台产物。

## 3 目标、默认口径与兼容等级

### 3.1 默认口径

| 事项 | 默认决定 |
|---|---|
| 自动启用 | 否。发现插件只展示报告，用户显式启用后才执行 `main.js` |
| 插件目录 | 识别 `<vault>/.obsidian/plugins/<id>/`（兼容）、`<vault>/.xink/plugins/<id>/`（原生）、`<appdata>/plugins/<id>/`（用户级） |
| 同 id 优先级 | `.xink` > `.obsidian` > 用户级；重复项在管理器里列为 issue |
| 权限 | 默认拒绝写库、网络、编辑器写入、自定义 View；逐插件授权 |
| `data.json` | 写在 `<appdata>/plugins/<id>/data.json`，不写回库内插件目录，避免随库同步泄露设置 |
| `styles.css` | 只作用于插件自己的 WebView，不注入原生 UI |
| 版本 | `apiVersion` 是“兼容档位”，不是 Xinkgo 的 `Version`；不改 `mygo.json` |
| 失败策略 | 一个插件报错不阻断启动、不影响其它插件；错误进日志与管理器 |

P0 的实际口径：手工放进插件目录的包默认没有任何额外授权；从“浏览”页
一键安装已检查过的兼容插件时，会按安装时二次扫描的结果授予所需能力，
用户仍可在管理器里逐项撤销。`network`、`vault.write` 等能力不会被静态
扫描之外的代码隐式放大。

### 3.2 兼容性等级

| 等级 | 含义 | 判定 | 本设计支持 |
|---|---|---|---|
| C0 | 包布局 | manifest 合法、`main.js` 是 CommonJS、目录名与 id 一致、无 ESM 顶层语法 | P0 |
| C1 | 无头核心 | 只用 `Plugin/Component/Events`、`loadData/saveData`、`addCommand`、`Notice`、少量纯函数 | P0 |
| C2 | 原生 UI 映射 | 增加 ribbon/status/命令面板/设置 tab/Modal/Menu/Vault/Editor/requestUrl | P1 |
| C3 | 插件 DOM 表面 | `registerView/ItemView`、`styles.css`、Markdown post processor | 受限兼容：DOM shim 能加载，UI/预览效果不保证 |
| C4 | 编辑器与系统内部 | `registerEditorExtension`、`EditorSuggest`、CM6/Lezer、Node/Electron、DOM workspace、OS 协议 | 降级兼容：惰性 stub 让插件尽量加载，相关功能不可用 |

C0 到 C2 是原生适配目标；C3/C4 可以在浏览器里安装，但必须显示
「受限/降级」标记和能力说明。ESM、缺少默认导出、真实系统能力依赖仍拒绝。
官方样例插件用到 ribbon、Notice、命令、Modal、设置 tab、全局 click 事件、
`setInterval`，属于 C2 + 少量 C3，可作为 P1 的人工验收目标。

## 4 架构

### 4.1 分层

```text
cmd/xinkgo
  -> internal/app                    生命周期：发现、协议、WebView、权限、编排
       -> internal/plugin           纯数据：manifest、能力扫描、兼容报告，无磁盘 I/O
       -> internal/state            plugins.json：启用状态与逐插件授权
       -> internal/view             原生 UI 适配：命令/图标/状态栏/插件管理器，只画与回调
       -> internal/vault/index/fs/… 已有的库、索引、文件能力
       -> resources/obsidian-shim/  JS shim、runtime、DOM helpers、moment vendor

cmd/xinkgo → internal/app → internal/{view,config,state,plugin,...}
view / config / state / plugin 均不 import internal/app
```

`internal/plugin` 只接收 `[]byte` 和内存结构：`ParseManifest`、`ScanSource`、
`Report`、`Capability`、`Level`。目录扫描、文件读取、WebView 创建、协议注册、
用户授权全部留在 `internal/app`。这样符合 AGENTS.md 的依赖方向，也让纯解析
与能力扫描可以在没有应用运行时单测。

### 4.2 一插件一隔离页

每个启用的插件创建一个 `mygo.WebView`：

```text
mygo.WebViewOptions{
  URL:  "xinkgo-plugin://<host>/",
  Page: mygo.PageOptions{
    Untrusted:            true,
    Session:              mygo.NewEphemeralSession(),
    ContentRules:         denyExternalNetwork,
    BackgroundThrottling: mygo.ThrottlingOn, // 声明 timer 并获授权才改 Off
  },
}
```

`<host>` 不是原始插件 id，而是 Go 侧生成的稳定 token；注册表把
`*mygo.Page -> pluginRuntime` 和 `host -> pluginRuntime` 双向映射。
`mygo.CallerPage(r.Context())` 是识别插件的唯一可信依据，URL host 只做二次
校验，JSON 体里的 `pluginID` 一律不信任。

WebView 默认不放到 `ui.WebView` 里：MyGo 的 WebView 在不显示时仍保留页面，
适合承载无头插件。插件请求自定义 View、Modal 或设置 tab 时，app 才把这块
WebView 放进一个原生容器或独立窗口。P2 先支持“一个插件同一时刻一个可见
表面”，多表面并存留到后续；这点要在兼容报告里写明。

### 4.3 自定义协议与桥

在 `WhenReady` 里、创建任何窗口之前注册：

```go
mygo.Protocol.HandleFunc("xinkgo-plugin", func(w http.ResponseWriter, r *http.Request) {
    p := mygo.CallerPage(r.Context())
    inst := instanceOf[p] // 由 app 持有，加锁
    if inst == nil || r.URL.Host != inst.Host {
        http.NotFound(w, r)
        return
    }
    // 按 path/method 分发；每个方法再查 inst.Grants
})
```

端点：

| 路径 | 作用 |
|---|---|
| `/` | Go 生成的宿主 HTML，按序加载 `/config.js`、`/shim.js`、`/plugin/main.js`、`/runtime.js` |
| `/config.js` | 注入插件 id、host、CSRF token、兼容档位、manifest |
| `/shim.js` | `require('obsidian')` 的实现、DOM helpers、类与常量 |
| `/plugin/main.js` | 插件包里的 CommonJS bundle，只读，从插件目录按白名单提供 |
| `/plugin/styles.css` | 经过 Go 清洗与作用域前缀的 CSS |
| `/plugin/assets/*` | 插件包内静态资源，路径必须落在包根之内 |
| `/runtime.js` | 实例化默认导出、调用 `onload`、订阅 `EmitPage`、处理卸载 |
| `/state` | 当前笔记、编辑器文本、选区、文件树、索引、主题、语言等快照 |
| `/rpc` | JSON-RPC：插件调 Go；Go 按能力表逐项校验 |
| `/log` | 插件 `console` 输出，带大小上限，进应用日志与管理器 |

`/rpc` 请求形如：

```json
{"id":17,"method":"vault.read","params":{"path":"note.md"},"token":"..."}
```

Go 从 `CallerPage` 得到插件实例，从实例取授权表，从 `method` 走显式
`switch`，再执行文件/网络/UI 动作；不实现“按名字调用任意 Go 服务”。响应只回
JSON 可序列化数据，不传 Go 对象、不传函数。大文本与二进制走 `/state` 或
`/plugin/assets/*`，`EmitPage` 只带小控制消息：

```text
{"type":"command","requestID":"...","commandID":"..."}
{"type":"stateChanged","revision":42}
{"type":"unload","requestID":"..."}
```

插件 shim 用 `window.mygo.on("xinkgo-plugin-event", handler)` 收事件，用
`fetch("/rpc", ...)` 回结果。即使插件找到了全局 `mygo.call`，非信任页的调用
也会被 MyGo 拒绝。Go 侧要显式装 deny 处理器：`SetWindowOpenHandler` 返回 nil、
`OnWillNavigate` 只允许本 origin、`SetPermissionHandler` 返回 false、
`OnWillDownload` 拒绝下载。`ContentRules` 与 CSP 是纵深防御，真正的边界是
`Untrusted` + 协议 handler 的能力校验。

CSP 建议：

```text
default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline';
img-src 'self' data:; connect-src 'self'; font-src 'self';
object-src 'none'; base-uri 'none'; form-action 'none'
```

不开放 `unsafe-eval`；正常流程也不用 `Page.Eval` 绕过 CSP。用 `eval`/动态
`require` 的插件会在运行时被报为不兼容。

### 4.4 CommonJS 加载与生命周期

宿主 HTML 不用内联脚本，按序加载同 scheme 的 JS。一个插件独占一页，所以
全局 `module/exports/require` 不会互相污染。`/shim.js` 在全局定义
`module`、`exports` 和一个严格 `require`：

```js
const obsidian = window.__XINKGO_OBSIDIAN__;
window.require = function require(name) {
  if (name === 'obsidian') return obsidian;
  if (name.startsWith('./') || name.startsWith('../')) return loadAsset(name);
  throw new XinkgoCapabilityError('unsupported module: ' + name);
};
window.module = { exports: {} };
window.exports = window.module.exports;
```

`/runtime.js` 的加载顺序：

1. 取 `module.exports.default || module.exports` 作为插件类；不是函数就报
   `main.js has no default Plugin class`。
2. 构造 `new Klass(appShim, manifest)`，其中 `manifest.dir` 注入为虚拟的库内
   相对路径；`instanceof Plugin/TFile/MarkdownView` 由 shim 的类保证。
3. `await plugin.onload()`，带超时（例如 10s）；期间插件通过 `/rpc` 注册命令、
   ribbon、状态栏、设置 tab 等。
4. 拉取 `/state` 建立同步 API 的本地缓存，POST `/rpc` 的 `ready`。
5. Go 把状态改为 running，刷新原生 UI；插件失败则标 failed 并保留日志。

卸载：Go 发 `EmitPage({type:"unload"})`；shim 调 `plugin.onunload()`，再执行
`Component.unload()` 清理 `register*`、事件、interval、DOM listener、子
Component；POST `/rpc` 的 `unloadDone`。Go 收到 ack 或超时后删掉命令、图标、
状态栏、菜单、View 注册，最后 `WebView.Destroy()`。强制销毁也能清掉页面内的
timer，但 Go 侧的原生注册必须自己删。插件进程崩溃由
`OnRenderProcessGone` 记录并标 failed，不做无限自动重启。

### 4.5 同步 API 与异步桥

Obsidian 有不少同步 API，例如 `Vault.getAbstractFileByPath()`、
`Workspace.getActiveViewOfType()`、`Editor.getValue()`；HTTP 风格的桥天然是
异步的。解决方式是**本地状态镜像**：

- Go 通过 `/state` 提供 `activeFile`、`editorText`、`selection`、`mode`、
  `fileTree`、`indexedMetadata`、`theme`、`language`、`platform`、
  `apiVersion`、`commandVisibility` 的快照与 `revision`。
- `TFile/TFolder/MarkdownView/Editor` 都是 shim 里的轻量对象，字段从快照读；
  同步 getter 返回缓存值，缓存缺失返回 `null`/空值，不阻塞主线程。
- `read/modify/create/delete/rename/replaceSelection/setValue/requestUrl` 等
  都是异步 `/rpc`；Go 完成后用 `stateChanged` 事件带新 `revision`，插件再拉
  快照。编辑器写操作带递增序号，避免插件连续调用乱序。
- `checkCallback/editorCheckCallback` 不在命令面板每帧远程询问：shim 在快照
  变化或命令注册时算好 `visible`，POST 给 Go 缓存；`paletteItems()` 只读缓存。

这条规则要写进兼容报告：`cachedRead` 之外的方法若插件假设“同步拿到磁盘最新
值”，在 Xinkgo 里可能拿到上一帧快照。

## 5 权限、路径与 CSS

### 5.1 能力

能力用字符串集合表示，存在 `<appdata>/plugins.json`，由 `internal/state`
拥有；默认全关。管理器展示每个插件的“声明能力 / 扫描能力 / 已授权能力”。

| 能力 | 允许的动作 | 默认 |
|---|---|---|
| `data` | 自己的 `loadData/saveData` | 启用插件即有 |
| `commands` | `addCommand/removeCommand`、命令面板项 | 启用插件即有 |
| `notice` | `Notice`、状态提示 | 启用插件即有 |
| `ui.ribbon` / `ui.status` | 原生图标与状态栏项 | 关 |
| `vault.read` / `vault.write` | 库内读 / 写，含二进制 base64 | 关 |
| `editor.read` / `editor.write` | 当前文本、选区、替换 | 关 |
| `metadata` | 双链、标签、大纲、缓存元数据 | 关 |
| `network` | `requestUrl` 到用户允许的 host 白名单 | 关 |
| `dom` | Modal、设置 tab、自定义 View、插件 CSS | 关 |
| `preview.postprocess` | Markdown 后处理 | 关 |
| `events.app` | 宿主窗口的 click / file-open / vault-change 事件 | 关 |
| `editor.extension` | CM6 扩展 | 永不授予，C4 |

`require('electron')`、`require('fs')`、`require('child_process')`、
`@codemirror/*`、`@lezer/*` 一律拒绝。插件 manifest 可以带一个可选
`xinkgo: {"capabilities": [...]}` 字段声明能力，Obsidian 会忽略未知字段；
实际授权仍以静态扫描 + 用户勾选 + 运行时校验为准。

### 5.2 路径

- 库内路径统一按 `normalizePath` 语义处理：正斜杠、去掉首尾 `/`、折叠 `.`、
  拒绝 `..` 和绝对路径；解析后再用 `filepath.Rel` + `EvalSymlinks` 确认仍在
  库根内。Windows 的大小写与 macOS 的大小写不敏感都要覆盖测试。
- 插件包目录只读：`main.js`、`styles.css`、`assets/*` 可 GET；`data.json`、
  写包内文件一律拒绝。
- `manifest.dir` 是虚拟路径。插件若 `adapter.read(`${manifest.dir}/data.json`)`，
  Go 映射到 `<appdata>/plugins/<id>/data.json`；若 `adapter.read(`${manifest.dir}/icon.png`)`，
  映射到包内只读资源。
- 请求体、返回体、二进制 base64、`console` 日志都设上限；`requestUrl` 用
  Go `net/http` 发，带超时、host 白名单和响应大小上限，不绕过 CSP 的
  `connect-src`。
- `onExternalSettingsChange` 由 app 的文件监视器观察 `<appdata>/plugins/<id>/data.json`
  后发事件；库内插件包改动不自动重载代码。

### 5.3 CSS 与 DOM helpers

`styles.css` 由 Go 读出后加 `[data-plugin-id="<id>"]` 作用域前缀，去掉
`@import`、外部 `url()`、可疑表达式，只允许 `data:` 与 `/plugin/assets/*`。
它只能影响插件 WebView 内的 DOM 根节点，不能给原生 UI、编辑器或普通预览
上色。`addIcon/setIcon` 的自定义 SVG 也只在插件 DOM 内有效；原生 ribbon
先回落到通用图标 + tooltip。

shim 要实现 Obsidian 常用的 DOM helper 子集：`createEl/createDiv/createSpan`、
`setText/empty/addClass/removeClass/toggleClass/setAttr`、`createFragment`、
`activeDocument/activeWindow`、`sanitizeHTMLToDom`、`htmlToMarkdown`、
`normalizePath`、`Platform`、`getLanguage`、`apiVersion`、
`requireApiVersion`、`moment`。`moment` 用带许可证的纯 JS vendor 文件，
不放进 Go 依赖；未实现的方法抛 `XinkgoCapabilityError`，并在兼容报告里记名。

## 6 API 映射表

| Obsidian API | Xinkgo 实现 | 阶段 | 限制 |
|---|---|---|---|
| `Plugin.onload/onunload/loadData/saveData` | shim + `/rpc` data 端点 | P0 | `data.json` 在应用数据目录 |
| `Component.register/registerEvent/registerDomEvent/registerInterval/addChild` | shim 生命周期表 | P0 | DOM 事件只在插件页；宿主事件需授权 |
| `Events.on/off/offref/trigger` | shim emitter + Go 事件转发 | P0 | 不跨插件广播 |
| `addCommand/removeCommand` | Go 命令表 + `paletteItems()` | P0 | id/name 加插件前缀；`hotkeys` 暂不自动注册 |
| `Notice` | app `Say` / 原生提示 | P0 | 不返回真实 `noticeEl` |
| `Vault.read/cachedRead/modify/create/delete/rename/adapter.*` | `/rpc` + `internal/fs` | P1 | 路径沙箱、能力、异步 |
| `TFile/TFolder/getAbstractFileByPath/getRoot` | shim 类 + `/state` 文件树快照 | P1 | `stat` 等字段由快照填 |
| `MetadataCache.getFileCache/getCache/getFirstLinkpathDest` | `internal/index` 的 JSON 投影 | P1 | 只做已索引字段 |
| `Workspace.getActiveViewOfType/activeEditor/getLeaf/openLinkText` | `/state` 镜像 + 命令 | P1 | 没有 `WorkspaceLeaf/containerEl` |
| `Editor.getValue/getSelection/replaceSelection/setValue/getLine` | 镜像 + 序号化 `/rpc` | P1 | 不是 CM6；无事务与 decoration |
| `requestUrl` | Go `net/http` + host 白名单 | P1 | 响应 JSON/base64 转换，不读任意 origin |
| `addRibbonIcon/addStatusBarItem` | `view.Workspace` 原生按钮/状态项 | P1 | 返回占位 `HTMLElement`，`setText` 推给 Go |
| `PluginSettingTab/Setting/Modal/Menu` | 独立插件 WebView 表面 | P2 | 原生设置窗口不能直接画 DOM |
| `registerView/ItemView` | 侧栏或独立窗口里的 `ui.WebView` | P2 | 一个插件一次一个可见表面 |
| `registerMarkdownPostProcessor/registerMarkdownCodeBlockProcessor` | HTML 预览模式 + DOM 序列化 | P3 | 无实时 DOM、无跨渲染生命周期 |
| `styles.css/addIcon/setIcon` | 作用域化插件 WebView | P2 | 不作用原生 UI |
| `registerEditorExtension/registerEditorSuggest` | 拒绝 | 不做 | 需要 CM6，见 C4 |
| `require('electron'\|'fs'\|'child_process'\|'@codemirror/*')` | 拒绝 | 不做 | 运行时报 `CapabilityError` |
| `registerObsidianProtocolHandler/registerCliHandler/secretStorage` | 不做 | 不做 | 无 OS URL scheme / CLI / 插件钥匙串接口 |

Markdown 后处理若进入 P3，路径是：app 用现有 `internal/export` 或
`internal/markdown` 生成一页 HTML，插件 runtime 通过自定义 scheme 取到这页，
在 detached DOM 上跑 post processor，把结果 HTML POST 回 Go，Go 再加载进预览
WebView。`getSectionInfo` 从渲染时附的 source map 取；`MarkdownRenderChild`
在下一次渲染时卸载。依赖真实 DOM 事件监听器、异步挂载到主文档、或读取
`ctx.containerEl` 兄弟节点的插件仍不支持。

## 7 代码落点

| 位置 | 内容 |
|---|---|
| `internal/plugin/` | `plugin.go` / `manifest.go` / `scan.go` / `catalog.go`：纯数据的 manifest、能力扫描、兼容等级、社区目录与兼容清单解析，不碰磁盘 |
| `internal/state/plugins.go` | `plugins.json` 的 `Enabled` 与 `Grants`，缺失/损坏回落 |
| `internal/app/plugin_discovery.go` | 三个目录的发现、优先级、重复 id、坏文件 issue；只在 `WhenReady` 后取应用目录 |
| `internal/app/plugin_host.go` | `Protocol.HandleFunc`、实例注册表、非信任 WebView、deny 处理器、CSP、状态快照、启停/重载/超时/unload ack |
| `internal/app/plugin_rpc.go` | `/rpc` 显式方法表、路径沙箱、Vault/Editor/网络/UI 动作、`data.json` |
| `internal/app/plugin_catalog.go` | 官方社区目录拉取、兼容清单、安装时二次扫描与写入用户插件目录 |
| `internal/app/plugin_settings.go` | 设置窗口里的插件分组：已安装列表、浏览器、授权勾选、启用/停用/重载、打开目录、状态 |
| `internal/view/plugin.go` | `PluginCommand` 与 `Workspace` 的回调字段；视图只画命令并报告 ID，不碰磁盘 |
| `internal/view/palette.go` | 把插件命令并进 `paletteItems()`，读 Go 缓存的可见性 |
| `resources/obsidian-shim/` | `host.html`、`shim.js`、`runtime.js`：CommonJS 包装、`obsidian` shim、DOM helpers、最小 `moment`、事件与命令分发 |
| `resources/obsidian-plugins/compatible.json` | P0 已静态检查过的兼容插件清单；用户可在 `<appdata>/plugins/compatible.json` 覆盖 |
| `internal/locale` | 宿主新文案的 en + zh-Hans 键，跑 `go tool mygo generate` |
| `docs/ACCEPTANCE.md` | J1 ~ J8 人工验收项，不写在设计文档里 |

已同步更新 [PROJECT.md](../PROJECT.md) 的目录树与
[ARCHITECTURE.md](../ARCHITECTURE.md) 的依赖表、扩展点、持久化表。`cmd/xinkgo`
不变，`go.mod` 不新增依赖，`mygo.json` 的版本不变；shim 是随包 JS 资源，
不把 Node/esbuild 加进运行期或 `make deps`。若用 TypeScript 写 shim，编译产物
入库，构建门禁仍只认 Go。

所有宿主可见文案走 `internal/locale`；插件自己写的名字、说明、按钮文字是数据，
不进 catalog，也不尝试翻译。协议 handler 跑在 goroutine，改界面状态一律
`mygo.RunOnMain`；`NewWebView/Destroy` 在主线程；视图每帧不做 I/O。macOS
P0 先验，Windows P1、Linux P2 再验；`mygo.WebViewAvailable()` 为假时整个
插件功能禁用并说明原因，原生主窗口照常工作。

## 8 兼容报告与插件管理器

`internal/plugin.Report` 至少包含：id、name、version、author、source、dir、
level、required capabilities、unsupported imports、unsupported API calls、
issues、ready。静态扫描是 Go 侧对 `main.js` 文本的保守词法扫描，不引入
cgo/Node/JS 引擎；能识别 `require("...")`、`registerEditorExtension`、
`WorkspaceLeaf/containerEl/activeDocument` 等模式，动态 `require(expr)`
标为“运行时可能失败”。它不是安全边界：最终以运行时 `require` proxy 和
Go 能力表为准。

插件管理器是原生 UI，列出每个包：名字、版本、作者、来源目录、C0 ~ C4 等级、
所需能力、issue、启用开关、Reload、Open folder、授权勾选、最近日志。首次打开
显示官方安全提示：插件是第三方代码，用户需要信任作者。一个坏 manifest、
缺失 `main.js` 或扫描失败的插件不妨碍其它插件；禁用状态不执行任何插件 JS。

“浏览”页在设置窗口的 Plugins 分组里，读取官方 `community-plugins.json` 与
`community-plugin-stats.json`，默认只显示
`resources/obsidian-plugins/compatible.json` 里已检查过的条目。安装时重新下载
`manifest.json` / `main.js` / 可选 `styles.css`，再次跑能力扫描；扫描不过就拒绝
安装。这样兼容清单过期也不会让硬不兼容的插件被直接装进来。清单来自对官方
统计里最热门前 400 个插件的静态扫描，按下载量取前 120 个可加载候选，加上
早期 8 个已检查核心插件，共 128 条；其中 120 条明确标为 C3/C4 降级兼容。

每个可见的兼容插件还显示下载/安装次数、最近更新、版本、当前版本下载量、
最低 Obsidian 版本、是否仅桌面端、仓库、作者主页、赞助、Stars、许可证、最近
提交和兼容检查日期。Stars/许可证/最近提交来自 shields.io 的公开 JSON，
manifest 字段来自 release 资产；任何一项拉取失败都只省略该行，不影响目录。

## 9 分阶段路线

### P0：宿主 spike

P0 已实现：`internal/plugin` 纯解析与扫描、`plugins.json`、
`xinkgo-plugin://` 协议 handler、一插件一非信任 WebView、CommonJS shim、
`Plugin.onload/onunload`、`loadData/saveData`、`addCommand/removeCommand`、
`Notice`、`Vault`/`Editor`/`Workspace`/`MetadataCache` 子集、状态快照、
命令面板、设置窗口的插件分组与浏览器、兼容清单、一键安装、官方统计与
release 详情展示；另加惰性模块 stub、DOM/设置/菜单/View/CM6 占位 API，
把可安装集合从 8 个扩到 128 个。实际 shim 用最小 `moment` 实现而非完整
vendor；未知导出与 external 模块返回惰性 stub，硬不兼容项仍拒绝。
浏览器与已安装列表按官方 `downloads` 降序；目录树左侧预留固定 plugin ribbon；
最常用的社区插件按 [obsidian-core-plugins.md](obsidian-core-plugins.md) 标为 Core。
fixture 形态如下：

```js
const { Plugin, Notice } = require('obsidian');
module.exports = class Demo extends Plugin {
  async onload() {
    this.addCommand({ id: 'hello', name: 'Hello', callback: () => new Notice('Hello') });
  }
};
```

已覆盖的自动化条件：manifest/扫描/目录/状态/路径沙箱/命令面板单测；
`require('fs')` 等外部模块与非信任页调用 Go 服务由运行时边界拒绝；
禁用/重载会移除原生注册；一个插件失败不影响宿主。macOS 先过
`scripts/check.sh`；真机人工验收见 [ACCEPTANCE.md](../ACCEPTANCE.md) J1 ~ J8。

### P1：核心 API 与官方样例

Vault 读写、TFile/TFolder、MetadataCache、当前 Editor 子集、`requestUrl`、
ribbon/status、命令可见性检查、hotkeys 冲突处理、宿主事件授权。目标是让
官方 sample plugin 的命令、Notice、ribbon、status bar 这些不依赖 DOM 表面的
行为跑起来；设置 tab 与 Modal 留到 P2。

### P2：插件 DOM 表面

自定义 View、`styles.css` 作用域、DOM helpers、插件设置页、Modal 窗口、
`activeDocument` 的全局 click 映射、多插件隔离与资源上限、
`onExternalSettingsChange`。明确“一个插件一次一个可见表面”的限制，并给出
兼容报告。

### P3：评估后处理器与 CM6

Markdown post processor 走 HTML 序列化预览模式，只承诺静态 DOM 变换。
`registerEditorExtension`/`EditorSuggest` 仍拒绝；若产品上必须支持，唯一
诚实路线是 B 方案：把编辑器换成 WebView + CodeMirror 6，并同步 DOM
workspace，这属于重写，不在本路线承诺内。

## 10 测试与人工验收

已落地的单测：`internal/plugin` 的 manifest 合法/缺字段/非法 id/目录名不一致/
未知字段、兼容档位比较、能力扫描（允许、拒绝、动态 require）、社区目录与
兼容清单过滤；`internal/state` 的 `plugins.json` 往返与损坏回落；
`internal/app` 的插件发现、坏包 issue、路径穿越拒绝；`internal/view` 的插件
命令进入命令面板并被选中。协议 handler 的 `CallerPage`/host/token 校验、
能力拒绝与方法白名单留在真机验收里覆盖。跑 `make test-race` 至少一次。

人工验收已登记到 [ACCEPTANCE.md](../ACCEPTANCE.md) 的 J1 ~ J8：
macOS 启用/禁用/重载样例插件、命令面板、Notice、ribbon/status、设置 tab、
Modal、Vault 读写、网络拒绝、CSS 只作用插件表面、卸载后计时器与图标都消失、
Windows/Linux 的 WebView 差异、无 WebView 时的回落。提交前跑
`scripts/check.sh`，确认 `CGO_ENABLED=0` 下全平台编译仍通过，且
`resources/obsidian-shim/` 真进了 `.app` 与各平台产物。

## 11 明确不做与替代方案

| 不做 | 原因 | 替代 |
|---|---|---|
| 100% Obsidian 插件兼容 | 需要 DOM workspace、CM6、Node/Electron 与整套应用壳 | 兼容等级 + 报告 + 插件作者按 C1/C2 约束写 |
| 用 Go 内嵌 JS 引擎跑 `main.js` | 没有 DOM/Electron 仍要写全套 shim；容易引入 cgo 依赖 | 用 MyGo 系统 WebView，非信任页隔离 |
| 把插件源码转译成 Go 原生扩展 | 不是“直接兼容”，且每个插件都要维护 | 原生插件 API 另行设计，独立路线 |
| `mygo.BindAs` 暴露通用插件服务 | 非信任页不能调 Go；可信页又会把所有绑定服务暴露给插件 | 自定义 scheme + `CallerPage` + 显式方法表 |
| `styles.css` 注入原生 UI | 原生 UI 没有 DOM/CSS | 插件自己的 WebView 表面 |
| 把降级 API 伪装成完整支持 | 插件会以为功能生效，排障困难 | 标 C3/C4 降级、显示所需能力，stub 只保证尽量加载 |

需要拍板的三个默认口径（不阻塞 P0，按本文推荐值推进）：是否接受
`.obsidian/plugins` 作为默认兼容入口（推荐是）、是否给 `vault.write`
与 `network` 默认授权（推荐否）、是否投入 P3 的 CM6 编辑器替换（推荐否，
先做兼容报告）。

## 12 参考来源

- [obsidian-api/obsidian.d.ts](https://github.com/obsidianmd/obsidian-api/blob/master/obsidian.d.ts)
- [obsidian-sample-plugin](https://github.com/obsidianmd/obsidian-sample-plugin)
- [obsidian-developer-docs: Manifest](https://github.com/obsidianmd/obsidian-developer-docs/blob/master/en/Reference/Manifest.md)
- [obsidian-developer-docs: Build a plugin](https://docs.obsidian.md/Plugins/Getting+started/Build+a+plugin)
- [obsidian-developer-docs: Anatomy of a plugin](https://github.com/obsidianmd/obsidian-developer-docs/blob/master/en/Plugins/Getting%20started/Anatomy%20of%20a%20plugin.md)
- [Obsidian Plugin security](https://help.obsidian.md/Extending+Obsidian/Plugin+security)
- [MyGo v0.7.4 frontend / untrusted pages](https://github.com/egoist/mygo/blob/v0.7.4/docs/frontend.md)
- [MyGo v0.7.4 distribution / resources](https://github.com/egoist/mygo/blob/v0.7.4/docs/distribution.md)
- [MyGo v0.7.4 windows / bindings](https://github.com/egoist/mygo/blob/v0.7.4/docs/windows.md)
