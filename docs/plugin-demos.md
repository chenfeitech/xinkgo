# 插件示例：安装与使用

日期：2026-11-08。适用：`examples/plugins/xinkgo-demo-basic` 与
`examples/plugins/xinkgo-demo-full`。本文面向**使用者**：从拷目录到看到第一条
提示。插件机制、能力对照与准入原则见
[../examples/plugins/README.md](../examples/plugins/README.md)，设计边界见
[design/obsidian-plugins.md](design/obsidian-plugins.md)；想自己写插件见
[plugin-development.md](plugin-development.md)。

两个示例只使用 Xinkgo **全力支持**的能力。宿主为了兼容 Obsidian 而提供的空壳、
惰性代理和 no-op 注册没有写进插件，所以你在这里看到的每条命令都有真实效果。

## 先说结论：要不要编译

**用户机器上不需要编译，Xinkgo 也不编译插件代码。** 宿主用
`os.ReadFile` 原样读出 `main.js`，通过 `<script src="/plugin/main.js">` 在插件
自己的隔离 WebView 页里执行；`shim.js` 提供 `require("obsidian")`，
`runtime.js` 取 `module.exports` 作为插件类并调用 `onload()`。`styles.css` 只是
读取后去掉 `@import` 和远程 `url()`，`assets/*` 原样提供，都不做构建。
从社区浏览器一键安装时，下载的是 release 里**已经构建好的**
`manifest.json` / `main.js` / `styles.css`，落地即用，不需要 Node 或 esbuild。

前提是包里的 `main.js` 已经是**单文件 CommonJS bundle**：

- 必须有 `module.exports`（或 `exports.default`），不能有 ESM 的
  `import` / `export` 语句；否则静态扫描判为 C0 不支持，安装会被拒绝。
- 不能有相对导入或动态模块名：宿主不解析 `./foo`，也不提供真实的 Node /
  Electron / CodeMirror 模块，这类依赖不应出现在插件里。
- 所有代码都要打进 `main.js`，只把 `obsidian` 留在 external。

也就是说：TypeScript、ESM 或拆成多个文件的插件源码，**必须由插件作者在发布前
打包成 `main.js`**，只是这一步发生在作者侧，不在你的机器上。我们这两个示例是
手写的单文件 CommonJS，所以零构建、拷过去就能跑。

另外，`daily-notes` / `templates` / `web-viewer` 等 13 个**内置核心插件**不是
JS 插件，是编译进应用的 Go 原生 UI，只能在设置里开关，不能从外部安装，也不存在
用户编译这一步。

## 两个示例

| 目录 | 定位 | 兼容等级 | 降级 |
| --- | --- | --- | --- |
| `xinkgo-demo-basic` | 最小可用插件，只走原生适配器 | C2 / `LevelNative` | 否 |
| `xinkgo-demo-full` | 覆盖宿主全力支持的全部 11 项能力 | C3 / `LevelDOM` | 是（见下） |

两者都 `Supported=true`。Full 的 `Degraded=true` 是宿主的分级规则：用到
`registerView` 这类 DOM 级能力就会标降级，含义是「视图在插件自己的 WebView 页，
不是原生 UI 面板」。这个视图由宿主真实实现、能点能交互，不是占位。

## 安装

插件包就是一个目录，**目录名必须等于 `manifest.json` 里的 `id`**，里面至少要有
`manifest.json` 和 `main.js`。宿主按下面三处发现，后面的来源覆盖前面的同 id 包：

| 来源 | 路径 |
| --- | --- |
| 用户级（推荐） | macOS：`~/Library/Application Support/Xinkgo/plugins/<id>/`；Windows：`%AppData%\Xinkgo\plugins\<id>\` |
| Obsidian 兼容 | `<vault>/.obsidian/plugins/<id>/` |
| 库内原生 | `<vault>/.xink/plugins/<id>/` |

macOS 上从仓库拷贝：

```sh
PLUGINS="$HOME/Library/Application Support/Xinkgo/plugins"
mkdir -p "$PLUGINS"
cp -R examples/plugins/xinkgo-demo-basic "$PLUGINS/"
cp -R examples/plugins/xinkgo-demo-full "$PLUGINS/"
```

也可以在设置窗口 ▸ **Community plugins** ▸ **Installed** 里对插件点
**Open folder** 直接打开用户级目录。拷完点 **Refresh** 让宿主重新扫描。

另一种是设置窗口 ▸ **Browse** 里的一键安装：宿主从 GitHub release 下载三个文件。
这种方式**只适用于已发布的社区插件**，我们这两个示例请在本地拷目录。

## 启用与授权

在 **Installed** 页打开插件开关，再展开卡片里的 **Grants** 勾选能力。
`data`、`commands`、`notice` 是启用即有的生命周期能力，其余默认全关。

两种安装路径的授权行为现在一致：

| 安装方式 | 授权行为 |
| --- | --- |
| 手动拷目录 / 改包内容 | 不写任何授权，**需要逐项勾选** |
| Browse 一键安装 | 同样不写授权，状态栏提示「授权后重载」 |

早期版本的一键安装会自动授予扫描出的全部能力，与设计文档
[§5.1](design/obsidian-plugins.md) 的「默认全关」不一致，2026-11-09 起已修正。
安装后先勾 Grants 再 **Reload**，ribbon、状态栏和视图才会注册成功。

两个示例建议这样勾：

- `xinkgo-demo-basic`：`ui`、`vault.read`、`editor.read`、`editor.write`。
- `xinkgo-demo-full`：`vault.read`、`vault.write`、`editor.read`、
  `editor.write`、`network`、`events.app`、`ui`、`dom`，共 8 项。

**顺序很重要**：ribbon 图标、状态栏和插件视图都是在 `onload()` 里注册的，
当时能力没授权，那次注册就被宿主拒绝，事后勾选不会补回来。所以要么先勾授权再
启用，要么勾完点一次 **Reload**。少勾一项时对应 RPC 会拒绝，插件用 `Notice`
和日志报出原因，不会让插件加载失败。

## 怎么确认插件真的跑起来了

同时看这五个信号：

1. 命令面板（`Cmd/Ctrl+P`）里出现带 `Xinkgo Demo: Basic:` 或
   `Xinkgo Demo: Full:` 前缀的命令。
2. 主窗口左侧插件 ribbon 出现图标（需要 `ui`）。
3. 状态栏出现 `Demo Basic: N runs` / `Demo Full: N runs`。
4. **Installed** 页里插件状态是 enabled，等级显示 C2 / C3。
5. 日志里出现 `msg="plugin console"` 且带 `id=<插件 id>` 的行：两个 demo 在
   加载和执行命令时都会写日志。
6. 设置 ▸ **Community plugins** ▸ **Official library** 里，两个 demo 都在清单中；
   已安装的库内插件在 **Installed** 页带 `Verified` 徽标。

启用瞬间还会弹一条 `Xinkgo Demo: ... loaded` 的 Notice。

## Basic：逐条试

每条命令都会弹一条 Notice、让状态栏计数 +1（计数存进 `data.json`），并在日志里
留一行 `Xinkgo Demo: Basic | notice | <内容>`。

| 命令 | 预期 |
| --- | --- |
| Say hello | `Hello from Xinkgo Demo: Basic (run #N)` |
| Read the active note | `<路径>: N characters`；没打开笔记时提示 `Open a note first` |
| Count words in the active note | `N words in the active note` |
| Insert today's date at the caret | 光标处插入 `YYYY-MM-DD` |
| Describe the selection | 显示选区前 40 字；**没有选区时这条命令在面板里自动隐藏** |

左侧 ribbon 的 Say hello 图标等价于第一条命令。切换笔记时 `file-open` 事件会写
一行日志。

## Full：逐条试

先勾完 8 项授权并 Reload。以下命令都在命令面板里，带
`Xinkgo Demo: Full:` 前缀。

| 命令 | 预期 |
| --- | --- |
| Print the capability tour | 日志打印 11 项受支持能力 |
| Say hello and persist a counter | Notice + 状态栏计数 |
| Read the active note | 日志给出 `vault.read` / `cachedRead` / `adapter.read` 的字符数与 Editor 读取结果 |
| Insert a line at the caret | 光标处插入一行带时间戳的文本 |
| Replace the whole buffer | **整篇替换当前笔记内容** |
| Show selection stats | 显示选区字符数；无选区时命令隐藏 |
| Run a Vault lifecycle | 在库里 `demo-full/` 下创建 → 修改 → 追加 → 改名 → 删除一个笔记；需已打开笔记 |
| Open the active note by link | 通过 `workspace.openLinkText` 打开当前笔记，Notice 给出路径 |
| Fetch example.com | `requestUrl` 返回状态 200；**只在你手动点这条命令时联网** |
| Toggle a temporary command | 命令面板里多 / 少一条 `Temporary command` |
| Where is the demo view? | Notice 提示视图入口 |

`Replace the whole buffer`、`Insert a line at the caret` 会改当前笔记，
`Run a Vault lifecycle` 会在库里创建并删除文件，建议在测试库或测试笔记上试。

### 插件视图

`registerView` 注册了 `demo-full-view`：在主窗口右侧的**插件视图抽屉**里选中它，
宿主才会创建 `WorkspaceLeaf` stub、构造 `ItemView` 并把 `containerEl` 挂到插件
自己的 WebView 页面。页面里有一个「Add a row」按钮、逐行文本和一张从
`/plugin/assets/badge.svg` 取出的图片，用来证明插件 DOM 和静态资源确实工作。

插件 DOM 只在这一页里有效，不会改 Xinkgo 原生 UI，也不会出现在原生设置窗口。
插件真正的配置请用 `loadData/saveData`，`data.json` 写在用户数据目录的
`plugins/<id>/data.json`（即插件包所在目录）。若把插件放进库内的
`.obsidian/plugins/` 或 `.xink/plugins/`，`data.json` 也会写进库里，注意别把
私密配置随库同步。

## 日志在哪看

插件的 `console.log` / `console.error` 经桥转成宿主日志，行里有
`msg="plugin console"`、`id=<插件 id>`、`text=...`，例如：

```text
level=INFO msg="plugin console" id=xinkgo-demo-full level=INFO text="Xinkgo Demo: Full | capabilities | data, commands, ..."
```

| 运行方式 | 位置 |
| --- | --- |
| `make run` / `make dev`（`MYGO_ENV != production`） | 终端 stderr |
| 安装版 `.app` / `.exe` | macOS：`~/Library/Logs/Xinkgo/xinkgo.log`；Windows：`%AppData%\Xinkgo\logs\xinkgo.log`；Linux：用户数据目录下的 `logs/` |

可以用 `XINKGO_LOG_LEVEL=debug` 提高日志级别。

## 常见问题

| 现象 | 原因与处理 |
| --- | --- |
| 命令面板里找不到命令 | 插件没启用；或 `checkCallback` 判定要隐藏（需要选区、需要打开笔记） |
| Notice 显示 `is not granted` | 对应能力没勾。勾完建议 Reload，让 onload 里的注册重跑一次 |
| 左侧 ribbon 没有图标 | `ui` 未授权，或授权晚于启用 → 勾完 Reload |
| 视图抽屉里没有 `demo-full-view` | `dom` 未授权，或授权晚于启用 → 勾完 Reload |
| `Fetch example.com` 报 network 未授权 | 勾 `network`；该命令不会在启用时自动联网 |
| 改了 `main.js` / `manifest.json` 不生效 | 点 **Reload**；改 `manifest.json` 尤其要重载 |
| 插件管理器显示 C0 / 不支持 | 目录名 ≠ `id`、缺必填字段、ESM 语法或没有 `module.exports` |
| 插件状态 failed | 看日志里的 `error` 行，常见是 `onload()` 抛了未捕获异常 |
| 状态栏计数一直是 0 | `saveData` 需要 `data`（启用即有）；检查日志里有没有 RPC 拒绝 |
| 一键安装后 ribbon / 视图没出现 | 安装不再自动授权；勾完 Grants 后点 **Reload** |

## 边界

宿主没有真实 CodeMirror 6 编辑器、DOM workspace、原生设置窗口或 Node/Electron
模块；`styles.css` 只作用于插件自己的页面。`metadata`、`editor.extension`、
`compat.stub` 这三项虽能被扫描出来，但只是兼容占位，两个 demo 都不使用；完整的
准入原则与黑名单见
[../examples/plugins/README.md](../examples/plugins/README.md)。
