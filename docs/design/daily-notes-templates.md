# Daily notes / Templates / Web viewer 内置核心插件设计（v1 · 待拍板）

日期：2026-11-06（同日增补 Web viewer 并实现）。状态：**v1 已实现**，在
`feature/daily-notes-templates` 分支上。文件名沿用首次评审稿（当时只有两个
插件），现覆盖三个内置核心插件。2026-11-06 又按需求把 Web viewer 改为预览面板
标签页、优化模板列表，并修掉长标题溢出，见 §12 第 4、5 条。

> 与 [obsidian-core-plugins.md](obsidian-core-plugins.md) 的关系：该文 §1 原把
> Daily notes / Templates 列为「无 / 未内置」且没有 Web viewer 一行，本文是
> 补这三格的具体设计；三格已随本轮改为「已内置」。同时更新了
> [INDEX.md](../INDEX.md) 的状态段，并把人工验收项写进
> [ACCEPTANCE.md](../ACCEPTANCE.md)。

## 1 结论先行

1. 三者做成**内置核心插件**：Xinkgo 自己实现的原生 Go 能力，开关写进
   `plugins.json` 的 `core`，不做成 Obsidian 兼容插件。理由：兼容清单里的热门
   插件目前全是 C3/C4 降级、DOM 表面是空 stub；创建/写入笔记和加载网页都是
   核心能力，不该押在降级宿主上，更不该让「已安装」冒充「能用」。前两个完全
   不碰 WebView；Web viewer 是三者里唯一需要系统 WebView 的一个。
2. 现有「内置核心插件」机制只有 `CorePlugin{ID, Default}` + 开关 +
   `view.Workspace.CoreFeatures` 门控，**没有设置页、命令、ribbon、每插件配置、
   启停发布**。三个都需要这几样，所以 §4 提出五处最小改造；不引入完整插件
   SDK，也不改 JS shim。**Web viewer 推翻了 v1 的一处假设**：它持有必须释放的
   原生 WebView，因此开关要带最小启停钩子与可用性判定（§4.6）。
3. 新增一个纯领域包 `internal/templating`：把 Moment 风格的日期/时间格式翻成
   Go layout，并展开模板变量。Daily notes 与 Templates 共用它，避免两处各写
   一套日期格式，也保证「Daily notes 的模板文件」和「插入模板」展开结果一致。
4. 插件选项落在 `plugins.json` 新增的 `options` 字段（与 `core` / `enabled` /
   `grants` 同一文件、同一套原子读写），**不动 `config.json`**：后者是与 xink
   兼容的应用偏好，且导出/导入配置不该悄悄带上某个插件的模板目录。
5. 设置窗口侧栏新增动态的 **Plugin options** 分组（用 `ui.SidebarSection`）；
   只有开启且声明了 `Settings` 的内置插件才出现自己的一页（共三页）。这正是
   Obsidian「核心插件开启后才出现配置页」的形态。
6. 三者默认 **关闭**（对齐 Obsidian 默认）；现有 10 个内置核心插件仍默认开启。
7. Web viewer 不做成「再开一个裸 Web 窗口」：现有 `openWebURL` 已经是整窗加载
   远程页面、没有任何拒绝处理器。改为**预览面板里的标签页**（2026-11-06 按
   需求从「独立窗口」改来，见 §12 第 4 条），并把远程页面显式声明为非信任页、
   独立会话、只放行 http(s)（§6.5）。

## 2 参照：Obsidian 的三个插件提供什么

按官方帮助页的行为列出，作为 v1 的对齐范围（不是全量复刻）：

| | Daily notes | Templates |
|---|---|---|
| 设置项 | 日期格式（默认 `YYYY-MM-DD`）；新文件位置（库内相对目录，默认库根）；模板文件位置（可选，指向库内一篇模板笔记）；启动时打开今日笔记（开关） | 模板文件夹位置（库内相对目录）；日期格式（默认 `YYYY-MM-DD`）；时间格式（默认 `HH:mm`）；只读列出文件夹里的模板 |
| 命令 | 打开今日日记、打开前一天、打开后一天 | 插入模板、从模板新建笔记 |
| 入口 | 左侧 ribbon 的日历图标；命令面板；菜单 | 左侧 ribbon 的模板图标；命令面板；菜单 |
| 模板变量 | 复用 Templates 的日期/时间格式展开模板文件 | `{{title}}`、`{{date}}`、`{{time}}`、`{{date:格式}}`、`{{time:格式}}` |

Web viewer 单独列，因为它的形态和前两个（都只在库里写 Markdown）完全不同：

| Obsidian Web viewer | 行为 |
|---|---|
| 工作区里的标签页 | 网页在应用内以标签打开，可重排、拆分、弹出为独立窗口；新标签页有地址栏 |
| 外部链接 | 设置项「Open external links in Obsidian」默认开启，把外链接进应用内的 Web viewer |
| 阅读视图 | 工具条上的眼镜按钮，用 Mozilla Readability 抽正文 |
| 保存到库 | 工具条按钮把网页存成 Markdown，保存位置在 Web viewer 设置里指定 |
| 广告拦截 | 默认开启，可配规则列表与更新频率；官方同时提示第三方插件能读到 Web viewer 的 cookie |
| 设置项 | 主页、搜索引擎、保存位置、广告拦截开关与规则列表、清理浏览器缓存/数据 |
| 命令 | `Web viewer: Open web viewer` 及导航/保存命令 |

参考：<https://help.obsidian.md/Plugins/Daily+notes>、
<https://help.obsidian.md/Plugins/Templates>、
<https://help.obsidian.md/plugins/web-viewer>、
[obsidian-core-plugins.md](obsidian-core-plugins.md) §5。

v1 明确不对齐的部分见 §9：Templater 的 `tp.*` / `{{cursor}}` / JS 执行、
完整的 Moment token、`{{date+1d}}` 之类日期运算、日历视图、热键绑定；Web viewer
另不做完整 Readability、远程规则订阅、下载、扩展与多账户会话（§9）。

## 3 现状盘点与缺口

| 现状 | 位置 | 对这三个插件意味着什么 |
|---|---|---|
| 内置核心注册表：`CorePlugin{ID, Default}`，10 项，ID 稳定写入 `plugins.json` | `internal/plugin/core.go` | 没有「有设置页/有命令/有 ribbon/有选项」的声明位 |
| 开关链路：设置页 `builtinView` → `pluginHost.setCore` → `applyCoreFeatures` → `ws.CoreFeatures` → `w.coreOn(id)` | `internal/app/plugin_core.go`、`plugin_settings.go`、`internal/view/plugin.go` | 开关只推一个 `map[string]bool`，不会发布命令或 ribbon；关掉插件只隐藏界面 |
| `ws.PluginCommands` / `PluginRibbon` / `PluginViews` **只**由兼容插件宿主 `refreshCommands()` 全量覆盖 | `internal/app/plugin_host.go:347`、`internal/view/workspace.go` | 内置插件无法贡献命令/图标；`runCommand` / `runAction` 也只查 `h.instances` |
| 设置窗口是 8 个硬编码 `settingsTab`，侧栏是平铺 `ui.SidebarItem` | `internal/app/settings.go` | 没有「插件选项」分组，也没有按开关动态出现页面的机制 |
| `plugins.json` 只有 `enabled` / `grants` / `core` | `internal/state/plugins.go` | 没有每插件配置；`config.json` 是与 xink 兼容的 6 段固定结构，塞插件选项会破坏兼容与导出语义 |
| 笔记创建是 `newNoteIn`：`fs.UniqueName` + `Tabs.Create("# title\n")` | `internal/app/files.go` | 日记要「按日期算路径 → 存在则打开、不存在才按模板创建 → reveal / reindex」，需要可复用的原语 |
| 文案走 catalog：`en.json` + `zh-Hans.json` + `go tool mygo generate` | `internal/locale` | 三个设置页、9 个命令、picker、Web viewer 工具条与状态提示全部要新增键，不能写死在 Go 里 |
| 已有 WebView 能力：`win.NewWebView` + `ui.WebView(c, page)` 嵌进原生 UI；`Page.LoadURL` / `GoBack` / `Eval` / `OnWillNavigate` / `SetWindowOpenHandler` / `SetPermissionHandler` / `OnWillDownload` / `SetContentRules`；`NewSession` / `NewEphemeralSession` | `mygo@v0.7.4`、`internal/view/plugin_view_drawer.go`、`internal/view/wechat.go`、`internal/app/plugin_host.go` | Web viewer 不必自己造浏览器，但要把这些能力按「远程非信任页」重新配一遍 |
| `openWebURL(address)` 用 `mygo.NewWindow` 整窗 `LoadURL`，`openOutside` 在 `config.Preview.LinkOpen == in_app` 时调它 | `internal/app/web.go` | 这是目前唯一的「应用内看网页」，没有地址栏、没有导航按钮、没有拒绝处理器；Web viewer 开启后应接管这条路径，关闭时保持原行为 |
| `mygo.WebViewAvailable()` 守卫插件宿主、公众号预览、PDF 导出 | `internal/app/plugin_host.go`、`wechat.go`、`export.go` | Web viewer 同样要有回落：无 WebView 的机器开关不可开，且说明原因 |
| 铁律：无 cgo、依赖方向 `app → view/...`、主线程、视图每帧不做阻塞 I/O | `AGENTS.md`、`ARCHITECTURE.md` | 模板目录列举必须缓存；弹目录对话框必须在 goroutine 里 `RunOnMain` 回来；`view` 不得 import `app` |

Web viewer 还带来一条新的安全口径：现成的 `createWebView()` 给兼容插件页设了
`Untrusted: true`、独立的 `NewEphemeralSession()`、拒绝弹窗/下载/权限与域名白名单，
那是**插件自己的页面**。远程网页是另一种东西，必须另立一套规则（§6.5），不能
复用插件页的 `xinkgo-plugin://` 沙箱，也不能让它落到默认会话里读到应用前端的
数据。

另外两处现有小问题会顺手修掉，但范围严格受限：`keymap.Binding.Label/Desc` 是
包级变量里的中文硬编码（见 §9，v1 不碰）；`plugin_ribbon.go` 把所有图标画成
同一个 `IconPluginRibbon`，`PluginAction.Icon` 字段至今没被使用（§4.4 会用到）。

## 4 插件机制的必要改造

原则是**让内置插件和兼容插件走同一套视图贡献面**，而不是再造一套插件框架。
只有三个内置插件需要这能力，因此只加一个最小的释放钩子，不引入完整生命周期
接口、事件总线或插件 SDK（钩子的必要性见 §4.6）。

### 4.1 注册表变成「贡献描述」

`internal/plugin/core.go` 仍是纯数据，扩展 `CorePlugin`：

```go
type CorePlugin struct {
    ID           string   // 稳定，写进 plugins.json
    Default      bool
    Settings     bool     // 开启后在侧栏 Plugin options 里出现自己的一页
    Commands     []string // 贡献的稳定命令 ID，如 "daily-notes:open-today"
    Ribbon       bool     // 是否在插件 ribbon 放图标
    NeedsWebView bool     // 开启前需 mygo.WebViewAvailable() 为真
}
```

新增三项 `{ID: "daily-notes", Default: false, Settings: true, ...}`、
`{ID: "templates", Default: false, Settings: true, ...}` 与
`{ID: "web-viewer", Default: false, Settings: true, NeedsWebView: true, ...}`。
`CoreIDs()` / `IsCore()` / `DefaultCore()` 语义不变；`DefaultCore()` 仍只列 `Default: true`
的项，所以缺字段时这三个插件取到 `false`，显式 `false` 也照旧保留。

`Ribbon` 是**单一真值来源**：`coreRibbonActions(id)` 先查注册表，只有
`Ribbon: true` 的插件才贡献按钮，按钮取该插件的第一条命令、图标名来自
`coreRibbonIcons`。一致性单测遍历注册表，凡声明 `Ribbon` 的必须真的有按钮、
按钮 ID 必须能在命令表里路由。这样不会再出现「注册表说有、贡献表漏实现」
的情况（本轮 Templates 就曾漏过一次，见 §12 第 3 条）。

顺带补一个冲突防护：`plugin_discovery.go` 的 `discover()` 遇到与内置核心 ID
同名的第三方包，记为 `reservedCoreID` issue 而不是覆盖；命令 ID 一律用
`daily-notes:` / `templates:` / `web-viewer:` 命名空间，避免和第三方
`插件ID:命令ID` 撞车。

### 4.2 一个统一的命令/图标发布与分发器

`internal/app/core_contrib.go`（新）持有静态贡献表：命令 ID → `locale.Xxx()`
取名的 `view.PluginCommand{Visible: ...}`、ribbon 动作、以及
`func(*appState)` 处理器。`pluginHost.publishCoreContributions()` 做三件事：

1. 从 `h.saved.CoreOf(id)` 算出开启集合，写 `ws.CoreFeatures`（原
   `applyCoreFeatures` 的工作）。
2. 把开启插件的命令、ribbon 与 `refreshCommands()` 里的第三方贡献**合并**后
   一次性写 `ws.PluginCommands` / `ws.PluginRibbon`。今天 `refreshCommands()`
   是全量覆盖，必须改成先取内置贡献再合并，否则任一方的启停会把另一方抹掉。
3. `setCore()` 与 `startEnabled()` 都调用它，保证开关、安装、卸载、重载后列表
   一致。

分发改在 `appState` 一层：`ws.OnPluginCommand = st.runCommand`、
`ws.OnPluginAction = st.runAction`（原来是 `st.plugins.*`）。`runCommand` 先查
内置处理器表，未命中再交给 `pluginHost.runCommand`。`app.go` 的接线随之改两行。

### 4.3 每插件选项持久化

`internal/plugin/options.go`（新，纯数据）定义选项与默认值/校验：

```go
type CoreOptions struct {
    DailyNotes DailyNotesOptions `json:"daily_notes"`
    Templates  TemplatesOptions  `json:"templates"`
    WebViewer  WebViewerOptions  `json:"web_viewer"`
}
type DailyNotesOptions struct {
    DateFormat    string `json:"date_format"`     // 默认 YYYY-MM-DD
    Folder        string `json:"folder"`          // 库内相对目录，空 = 库根
    TemplatePath  string `json:"template_path"`   // 库内相对 .md 路径，可空
    OpenOnStartup bool   `json:"open_on_startup"` // 默认 false
}
type TemplatesOptions struct {
    Folder     string `json:"folder"`       // 库内相对目录，空 = 未设置
    DateFormat string `json:"date_format"` // 默认 YYYY-MM-DD
    TimeFormat string `json:"time_format"` // 默认 HH:mm
}
type WebViewerOptions struct {
    Homepage     string `json:"homepage"`       // 空 = about:blank 新标签页
    SearchEngine string `json:"search_engine"`  // 含 {query} 的 URL 模板
    SaveFolder   string `json:"save_folder"`    // 库内相对目录，空 = 库根
    SaveTemplate string `json:"save_template"`  // 可选，复用 Templates
    AdBlock      bool   `json:"ad_block"`       // 默认 true
    RuleFile     string `json:"rule_file"`      // 可选，本地 content-rules JSON
    AllowLocal   bool   `json:"allow_local"`    // 默认 false，见 §6.5
    ClearOnQuit  bool   `json:"clear_on_quit"`  // 默认 false
}
```

`state.PluginState` 加 `Options *plugin.CoreOptions \`json:"options,omitempty"\``，
`OptionsOf()` 在字段缺失时回落 `DefaultCoreOptions()`，`WithOptions()` 返回副本，
`copy()` / `normalized()` 保留它，`SavePlugins` 沿用原子写。空字符串按默认填，
显式 `false` 不丢。示例（`plugins.json`）：

```json
{
  "core": { "daily-notes": true, "templates": true, "web-viewer": true },
  "options": {
    "daily_notes": { "date_format": "YYYY-MM-DD", "folder": "Daily notes",
                     "template_path": "Templates/Daily.md" },
    "templates": { "folder": "Templates", "date_format": "YYYY-MM-DD",
                   "time_format": "HH:mm" },
    "web_viewer": { "homepage": "", "search_engine": "https://duckduckgo.com/?q={query}",
                    "save_folder": "Clippings", "ad_block": true,
                    "allow_local": false }
  }
}
```

所有路径一律**库内相对**：解析时 `filepath.Clean` + `filepath.Rel` 拒绝绝对
路径与 `..`，与 `vaultHost.within` 口径一致；库根为空时所有「创建」命令只报状态
提示，不写盘。不往用户库里写配置文件（Obsidian 写 `.obsidian/*.json`，Xinkgo
目前只在 `.obsidian/plugins`、`.xink/plugins` **读**），代价是切库后同一组相对
目录会沿用 —— 见 §10 待拍板第 2 条。

### 4.4 设置页与视图的小改动

- `settingsTabs()` 变成动态：`tabCorePlugins` / `tabCommunityPlugins` 之后追加
  一个 `ui.SidebarSection`（`internal/locale` 的 `settingsSectionPluginOptions`），
  里面只为 `s.plugins.coreOn(id) && spec.Settings` 生成 `SidebarItem`。
  `chosenSettings()` 在选中页已失效（插件被关掉）时回落到 `tabCorePlugins`。
  `isPluginTab()` 扩展三个选项页 ID，但**不**触发社区目录网络刷新；改为打开时
  读一次模板目录清单并缓存进 `settings` draft（照 `wechat.OfDir` 的做法，
  目录变了才重读，绝不每帧读盘）。
- `builtinView()` 给有 `Settings` 的插件行加一个「选项」按钮，直接跳到对应页；
  `builtinCoreText()` 补三个 case；`NeedsWebView` 且 `WebViewAvailable()` 为假的
  行把开关画成禁用并说明原因（复用 `locale.PluginNoWebView()` 的口径）。
- `view.PluginCommand` 加 `Group string`（本地化后的插件名）；`paletteItems()`
  副标题优先显示 `Group`，不再对内置命令露出 `daily-notes:open-today` 这种原始
  ID。
- `view.PluginAction.Icon` 终于被用上：`pluginIcon(name)` 把 `calendar-days`、
  `layout-template`、`globe` 映射到新增的 `ui.SVG`，未知名回落 `IconPluginRibbon`。
- `view.Workspace` 加 `InsertInActive(text) bool` 与 `ActiveBasename()`：模板
  插入需要往当前编辑器光标处写文本，app 通过这两个方法操作视图状态，`view`
  不反向依赖 `app`。

### 4.5 明确不改造的部分

不改 `internal/plugin` 的 manifest / 能力扫描 / 兼容等级；不改
`resources/obsidian-shim/`；不给 Daily notes / Templates 发 WebView，不给任何
内置插件发 `loadData/saveData` 与能力授权；Web viewer 的 WebView 是它自己的，
不复用兼容插件宿主的隔离页与 `xinkgo-plugin://` 协议。不引入新 Go 依赖、
不引入 cgo、不改 `mygo.json` 版本；`PageOptions.DevTools` 在生产构建里保持关闭。
macOS P0 先验证（§8）。

### 4.6 Web viewer 逼出来的两处机制补强

前两个插件只写库里的 Markdown，开关关掉就是「少画几个按钮」。Web viewer 不同：
它持有原生 `mygo.WebView`、一个独立 `Session` 和一个窗口，关掉时不释放会留下
可见页面与缓存数据。因此把 §4 的契约补两点，仍不是完整插件生命周期：

1. **最小释放钩子**：`coreContribution` 加 `onDisable func(*appState)`（可空）。
   `setCore(id, false)` 在写完 `plugins.json`、重新发布贡献**之后**调用它；
   Web viewer 的钩子按「销毁所有 WebView → 关窗 → 清缓存状态」的顺序执行，
   全部经 `mygo.RunOnMain`。反过来 `onEnable` 不做任何建窗/建页：WebView
   只在用户第一次执行 `web-viewer:open` 时按需创建，避免启动期和帧里做原生
   I/O。没有资源的插件（Daily notes / Templates）钩子为空。
2. **可用性判定**：`CorePlugin.NeedsWebView` 是纯数据标记，`app` 在
   `publishCoreContributions()` 里用 `mygo.WebViewAvailable()` 与之相与，决定
   命令是否 `Visible`、ribbon 是否出现、设置页是否可编辑、以及开关是否可点。
   不可用时 `openOutside` 一律退回 `mygo.Shell.OpenExternal`，不出现「点了没反应」。

这纠正了 v1 初稿「内置插件不需要生命周期」的判断：需要的只是**资源释放**，
不是 `onload/onunload` 事件总线。若以后还有第四个持有资源的插件，把钩子表
换成统一的 `onDisable` 列表即可，不必改注册表。

## 5 共享领域包 `internal/templating`

纯函数、无 I/O、无新依赖，Daily notes 与 Templates 共用；Web viewer 的
「保存到库」也复用它的 `Format` 给剪藏笔记取名：

- `Layout(format string) (string, error)`：Moment token → Go layout。支持
  `YYYY YY MMMM MMM MM M dddd ddd DD D HH H hh h mm ss A a`，分隔符
  `- _ / . : , 空格`，以及 `[字面量]`（方括号内的内容按字面输出）。**最长
  优先**替换，避免 `MM` 先吃掉 `MMMM`。未识别的字母 token 直接报错，不静默
  产出错误文件名。
- `Format(t time.Time, format string) (string, error)` = `t.Format(Layout(...))`；
  `Parse(name, format string) (time.Time, error)` = `time.ParseInLocation(
  Layout(...), name, time.Local)`。同一个翻译器同时服务「算文件名」和
  「从文件名反解日期」，前后两天导航不需要另写解析器。
- `Expand(text string, v Vars) (string, error)`：`Vars{Title, Now, DateFormat,
  TimeFormat}`。替换 `{{title}}`、`{{date}}`、`{{time}}`、
  `{{date:格式}}`、`{{time:格式}}`。**未知变量原样保留**（例如 Templater 的
  `{{cursor}}` 不该被吃掉），格式不合法时返回错误并在状态栏说明，而不是把
  半展开的文本写进笔记。
- `Validate(format string) error` 给设置页做即时校验：输入框旁边一行显示
  「今天的笔记会叫 2026-11-06.md」或红字错误，纯计算、每帧可算。

日期一律用 `time.Now().In(time.Local)`，跨天/跨月用 `AddDate(0, 0, ±1)`
（不自己加减 86400 秒，避开夏令时）。CJK 标题按 Go 字符串原样处理。

## 6 三个插件的功能设计

### 6.1 共用的笔记创建原语

`internal/app/note_service.go`（新）：`ensureNote(relPath, content string)`。

1. 解析并校验 `relPath` 在库内；`os.MkdirAll` 建父目录。
2. 文件已存在 → 只 `Tabs.Open(path)`，绝不覆盖（`fs.WriteNewFile` 本身也拒绝
   已存在）。
3. 不存在 → `Tabs.Create(path, content)`；随后 `s.rescan()`、
   `s.idx.Reindex(path)`、`s.ws.Tree.Reveal(path)`，与 `newNoteIn` 一致。
4. 失败只写日志 + `ws.Say(locale.Say...())`，不弹窗、不中断启动。

### 6.2 Daily notes

- 文件名：`templating.Format(今天, opts.DateFormat) + ".md"`；路径 =
  `filepath.Join(root, opts.Folder, 文件名)`。`Folder` 为空即库根。
- 内容：设了 `TemplatePath` 时读该模板（必须库内、`.md`、存在），用
  `templating.Expand` 展开；**日期/时间格式取 Templates 的选项**（关闭
  Templates 时用默认值），`{{title}}` 取日记文件名。没设模板或模板读不了 →
  建空笔记，并在状态栏说明原因。对齐 Obsidian：日记本身默认是空的。
- 命令（ID / 行为）：
  - `daily-notes:open-today` → `ensureNote(今日路径, 展开后的模板)`。
  - `daily-notes:open-previous` / `daily-notes:open-next` → 从当前日记
    （解析不出来则用今天）出发，扫描日记目录里能按 `DateFormat` 解析的文件名，
    取最近的前一个/后一个日期并打开；没有则状态栏提示、不新建笔记。扫描是
    一次 `os.ReadDir`，不每帧做。
  - ribbon 图标 `calendar-days` → 等价于 open-today；菜单 File 组加
    「打开今日日记」。
- `OpenOnStartup`：`restore()` 末尾、库已打开且插件开启时调用 open-today。
  无库、或 `restoreOpenNotes()` 已打开同一路径时不会重复建标签（`Tabs.Open`
  本身按路径去重）。

### 6.3 Templates

- 模板目录：`opts.Folder`，空表示未设置。设置页列出目录里的 `.md`
  （名字 + 坏文件说明，缓存，照 `wechat.OfDir` 的形态），并提供
  「浏览…」按钮：goroutine 里 `mygo.Dialog.Open(Directory: true)`，
  `mygo.RunOnMain` 回来把绝对路径折成库内相对路径；选到库外就拒绝并提示。
- 命令：
  - `templates:insert` → 打开模板选择框 → `Expand` → `ws.InsertInActive(结果)`。
    没有活动笔记时只提示，不新建。`{{title}}` 取活动笔记的文件名。
  - `templates:create-note` → 选择框 → 在当前目录（`Tree.DirOf()`）用
    `fs.UniqueName` 生成 `untitled.md` 系列 → `ensureNote(路径, 展开结果)`。
    菜单 Note 组加两个入口。
- ribbon 图标 `layout-template` → 等价于 `templates:insert`；图标、命令与
  Daily notes / Web viewer 一起由 `coreRibbonActions` 从注册表生成。
- 选择框：新增 `internal/view/picker.go`，`ui.Modal` + 模糊匹配 + ↑/↓/Enter/
  Esc，与 `palette.go` 同构；`Workspace` 持 `Picker{Open, Title, Items, Mode,
  OnPick func(string)}`，app 只在打开时灌一次候选，视图只画并回报选择。
  不复用命令面板：它有 `pluginPaletteLimit = 10` 的上限，且「插入」与「新建」
  需要不同标题和回退行为。
- 展开失败（格式非法、模板读不了、变量格式串不合法）一律不写盘，状态栏说明。

### 6.4 前两页设置长什么样

**Daily notes**（侧栏 Plugin options ▸ Daily notes）：日期格式（文本 +
实时预览「今天的笔记会叫 …」）、新文件位置（文本 + 浏览…）、模板文件位置
（文本 + 浏览…，说明它复用 Templates 的日期/时间格式）、启动时打开今日笔记
（开关）、以及一句「模板由 Templates 提供日期/时间格式」的说明。

**Templates**：模板文件夹位置（文本 + 浏览…）、日期格式、时间格式（各带实时
预览）、模板清单（只读，空目录/未设置时的提示）、变量说明一行。

### 6.5 Web viewer

**形态**：预览面板里的标签页。预览面板顶部有一条标签条：固定的「预览」标签
（笔记的 Markdown 预览）在左，其后每个网页一个标签，末尾是「+」新标签。网页
标签激活时，预览面板的内容区换成地址栏 + `ui.WebView(c, page).Fill()`；点
「预览」标签回到笔记预览，网页页保持存活。标签条与工具条由
`internal/view/webviewer.go` 画，`previewPanel` 在 `WebTabActive != ""` 时
调用 `webPane` 而不画 Markdown。**每个标签一个 `mygo.WebView`**（MyGo 明确保留
未显示的页），上限 8 个，关标签即 `Destroy()`。

关键约束：`ui.WebView` 只接受**同窗口**创建的原生页，所以 WebView 必须由主窗口
`s.win.NewWebView(...)` 创建，而不是另开一个窗口。`web-viewer:open` 只确保预览
面板可见（`SetPreviewShown(true)`）并激活一个网页标签；从外链进来时新开标签。
即便「Page preview」内置插件被关掉，只要有网页标签激活，预览面板仍然绘制。

**分层**：`internal/webviewer`（新，纯函数）做 URL 规则：补 `https://`、把非
URL 输入折成搜索引擎（`url.QueryEscape`）、scheme 白名单、私网/回环判定、
标题→文件名清洗、搜索模板 `{query}` 校验。`internal/view/webviewer.go` 只画
chrome 并回报回调：`Workspace` 持 `WebTabs []WebTabInfo`、`WebTabActive`、
`WebAddress`、`WebStatus` 与 `OnWebNavigate/OnWebBack/OnWebForward/OnWebReload/
OnWebHome/OnWebReader/OnWebSave/OnWebOpenExternal/OnWebNewTab/OnWebSelectTab/
OnWebCloseTab/OnWebShowPreview`；`internal/app/web_viewer.go` 拥有
`*mygo.WebView`、会话与规则，`webPublish` 把缓存状态推给 workspace。`view`
不 import `app`，回调由 `boot()` 接线。

**远程页的安全边界**（本插件最要紧的部分）：

- `WebViewOptions{URL: "about:blank", Page: PageOptions{Untrusted: true,
  Session: mygo.NewSession("xinkgo-webviewer"), ContentRules: compiled}}`
  （`DevTools` 保持默认关闭）。独立命名会话与兼容插件的 ephemeral 会话、应用前端的默认
  会话都隔开；Obsidian 提示的「第三方插件能读到 Web viewer 的 cookie」在
  Xinkgo 不成立，前提是插件宿主继续用独立会话 —— 写进验收项复查。
- `OnWillNavigate` 只放行 `http:` / `https:` / `about:blank`，拒绝 `file:`、
  `javascript:` 与自定义 scheme。`AllowLocal` 默认关时拒绝 loopback 与私网
  （`localhost`、`127/8`、`::1`、`10/8`、`172.16/12`、`192.168/16`、
  `169.254/16`），避免网页探测本机服务；打开时放行并给出提示。
- `SetWindowOpenHandler`：脚本自己弹的（`UserInitiated == false`）直接返回
  `nil`；用户点的 `target="_blank"` 在 handler 里把 `req.URL` 交给 Go 新建
  标签，同样返回 `nil` 拒绝原生弹窗，不让网页自己开窗口。
- `SetPermissionHandler` 恒 false，`OnWillDownload` 恒 `PreventDefault`，
  `TrustedOrigins` 留空；远程页不能调 `system.Info` 等任何绑定服务。
- `Page.Eval` 只由 Go 主动发起（阅读视图、保存到库），不向远程页注入 Go 桥。

**导航**：地址栏 `Submitted()` 后走 `internal/webviewer.Normalize` 再
`Page.LoadURL`；`OnPageTitleUpdated` / `OnDidFinishLoad` / `OnDidFailLoad`
把标题、加载态、错误写回 app 缓存后推给 view；`GoBack/GoForward/
CanGoBack/Reload/Stop` 直接调 `Page` 方法。视图每帧只读缓存，不调
`Page.URL()`、不做 `Eval`。

**阅读视图**：眼镜按钮 → goroutine 里 `Page.Eval` 跑随包 JS
（`internal/app/webviewer_assets/reader.js`，`go:embed` 进二进制，见 §12；
轻量启发式：取 `<article>` / `[role=main]`
/ 最大文本块，去掉 `script/style/nav/aside`）→ 得到 HTML → `LoadHTML(html,
原URL)` 灌进**该标签另建的第二个 WebView**。切换只改 `ui.WebView` 指向哪一页，
原网页保持存活，返回时切回。不 vendor Mozilla Readability、不引新依赖，
保真度登记为缺口。

**保存到库**：`Page.Eval` 取 `{title, text, url}` → 生成带 `source`/`date`
frontmatter 的 Markdown → 文件名用 `templating.Format(今天,
Templates.DateFormat)` 加清洗后的标题 → `ensureNote(SaveFolder/文件名, 内容)`
（§6.1）；设了 `SaveTemplate` 就先 `templating.Expand`。第三个插件因此把前两个
的笔记创建原语与模板展开真正复用起来，并自动获得库内路径校验、不覆盖、
reveal 与 reindex。

**广告拦截**：内嵌 `internal/app/webviewer_assets/content-rules.json`（几十条常见广告/
跟踪域，不是 EasyList）、可选用户 `RuleFile`。`mygo.CompileContentRules(id,
rules)` **在 goroutine 里编译**（官方说长列表要数秒），结果经 `RunOnMain` 放进
`PageOptions.ContentRules`，运行中改设置用 `Page.SetContentRules`。平台差异
写清楚：macOS / Linux（WebKitGTK ≥ 2.24）走 WebKit content blocker，Windows
走软件匹配且 WebSocket 不受控；规则正则是 WebKit 的受限子集，不承诺兼容外部
订阅列表。v1 不拉远程列表、不做「更新频率」。

**外部链接**：`openOutside` 在插件开启、`WebViewAvailable()` 为真且
`config.Preview.LinkOpen == in_app` 时把 URL 交给 Web viewer 新开标签；关闭或
不可用时保持现有 `openWebURL` 裸窗口 → 失败再回退 `Shell.OpenExternal`。若
Web viewer 因策略拒绝该地址，直接交给用户的系统浏览器，不再让本应用里防护更弱的
窗口加载它。
设置页**不复制**这个开关，只放一行只读说明加跳到「设置 ▸ 预览」的按钮，
避免两个真值来源。

**第三页设置**：主页、搜索引擎（含 `{query}` 校验与实时预览）、剪藏目录 +
可选模板、广告拦截开关 + 规则文件 + 「清理浏览数据」按钮
（`Session.ClearData(mygo.SiteDataAll)`，确认后执行）、允许本地地址（默认关）、
退出时清理（默认关）。无 WebView 时整页只显示原因。

**命令与入口**：`web-viewer:open` / `web-viewer:save-to-vault` /
`web-viewer:toggle-reader` / `web-viewer:open-in-browser`，ribbon `globe`，
菜单 View 组。`Visible` 由 `NeedsWebView && WebViewAvailable()` 门控。

**线程与平台**：`s.win.NewWebView` / `Destroy` 都在主线程（按需
`mygo.RunOnMain`）；`Eval` 与 `CompileContentRules` 在 goroutine，结果回主线程
发布；`ui.WebView` 只接受同窗口的 `NativeWebView`，因此所有页都挂在主窗口。
macOS WKWebView 先验，Windows 需 WebView2 Runtime，Linux 需 WebKitGTK ≥ 2.24；
三端 `CompileContentRules` 的降级行为写进验收。`CGO_ENABLED=0` 不变 —— 原生
WebView 由 MyGo 提供，仓库已经在用（插件宿主、公众号预览、PDF 导出）。

三页都用现有 `group()` / `field()` / `settingRow()` / `cardBox()` 组件；文案
全部走 catalog。文本字段用 `Submitted()`（回车/失焦）提交、下拉与开关用
`Changed()`，提交时调 `pluginHost.setOptions()` 原子写 `plugins.json` 并重新
发布贡献 —— 不像 `settingsView` 对 `config.json` 那样每帧比较后写盘，避免每敲
一个字符就写一次文件。

## 7 代码落点

新增：

| 文件 | 内容 |
|---|---|
| `internal/templating/{format,expand}.go` + `_test.go` | 日期/时间格式翻译、解析、变量展开、校验；纯函数单测 |
| `internal/plugin/options.go` + `_test.go` | `CoreOptions`、默认值、`Sanitized()`；纯数据单测 |
| `internal/app/core_contrib.go` | 贡献表、内置命令处理器、`runCommand`/`runAction` 分发、`publishCoreContributions` |
| `internal/app/core_options.go` | settings draft 的选项读写、`setOptions`、目录清单缓存 |
| `internal/app/note_service.go` | `ensureNote` 与库内路径解析/校验 |
| `internal/app/daily_notes.go` | 路径计算、三条命令、ribbon、启动打开、设置页 |
| `internal/app/templates.go` | 插入/新建、选择框灌数据、设置页 |
| `internal/webviewer/{url,rules}.go` + `_test.go` | URL 归一化、scheme/私网白名单、搜索模板、标题→文件名；纯函数 |
| `internal/app/web_viewer.go` | 预览标签生命周期、主窗口 WebView 创建/销毁、会话、导航事件、阅读视图、保存到库、广告拦截编译 |
| `internal/view/webviewer.go` + `_test.go` | 预览标签条、地址栏与工具条、`ui.WebView` 放置与回报 |
| `internal/app/webviewer_assets/{reader.js,content-rules.json}` | 阅读视图脚本与基础拦截规则，`go:embed` 进二进制（见 §12） |
| `internal/view/picker.go` + `_test.go` | 模板选择框与键盘操作 |
| `docs/design/daily-notes-templates.md` | 本文 |

修改（均为小范围）：

| 文件 | 改动 |
|---|---|
| `internal/plugin/core.go` | `CorePlugin` 字段扩展、新增三项、`CoreIDs` 不变 |
| `internal/state/plugins.go` | `Options` 字段、`OptionsOf/WithOptions`、`copy/normalized` |
| `internal/app/plugin_core.go` | `applyCoreFeatures` 并入贡献发布；`setCore` 一并刷新命令 |
| `internal/app/plugin_host.go` | `refreshCommands` 改为合并内置贡献；`runCommand/runAction` 仍只管第三方 |
| `internal/app/app.go` | 两个 `OnPlugin*` 接到 `appState` 分发器；`OnWeb*` 回调接线；`restore()` 末尾启动打开；`OnBeforeQuit` 的退出清理 |
| `internal/app/web.go` | `openOutside` 在插件开启时改投 Web viewer，关闭时保持裸窗口 |
| `internal/app/settings.go` | 动态侧栏分组、三个 tab ID、draft 选项字段、失效回落 |
| `internal/app/plugin_settings.go` | `builtinCoreText` 三个 case、`builtinView` 的「选项」按钮与不可用开关 |
| `internal/app/plugin_discovery.go` | 保留 ID 冲突检查 |
| `internal/app/menu.go` | File / Note / View 组各加入口 |
| `internal/view/{plugin,plugin_ribbon,palette,workspace,preview,icons}.go` | `Group`、`pluginIcon`、picker 状态、`InsertInActive`/`ActiveBasename`、Web 标签状态与 `previewPanel` 分流、三个 SVG |
| `internal/locale/{en,zh-Hans}.json` + 生成的 `catalog_gen.go` | 新增键，`go tool mygo generate` |
| `docs/{INDEX.md,ACCEPTANCE.md}`、`docs/design/obsidian-core-plugins.md` | 状态、验收项（K 组）、核心插件对照表 |

## 8 分期、测试与门禁

按依赖顺序做，每期单独提交（`make check` 全绿后再提交）：

1. **A 机制**：§4 全部 + catalog 键 + `internal/state` 往返测试。此时用户看
   不到新功能，只是三个默认关的开关与三页空白设置页。
2. **B Templates**：`internal/templating` → 模板目录清单 + picker + 插入/新建
   + 设置页。
3. **C Daily notes**：路径与命令、前后导航、ribbon、启动打开、设置页；模板
   展开直接复用 B 的 `templating.Expand` 与 Templates 的格式选项。
4. **D Web viewer**：`internal/webviewer` 纯规则 → 窗口与标签 → 安全钩子 →
   阅读视图 / 保存到库 → 广告拦截 → 设置页与外链接管；`ensureNote`、
   `templating.Expand` 直接复用 B/C 的成果。放最后是因为它依赖前两期，
   且是唯一引入原生 WebView 的一期。
5. **E 收口**：文档、验收项、门禁与跨平台。

自动化覆盖（照现有 `ui.Tester` / 纯函数单测习惯）：格式 token、字面量、
非法 token、`Format`↔`Parse` 往返、五种变量展开与未知变量保留、选项默认值/
显式 false/旧文件缺字段、`plugins.json` 往返、`ensureNote` 存在不覆盖、
日期边界（月末/跨年/夏令时）、库外路径与 `..` 拒绝、开关关掉后命令与 ribbon
同时消失、分发器在内置与第三方 ID 间的路由、三页设置渲染、picker 键盘选择、
启动开关不建重复标签；Web viewer 另加：URL 归一化与搜索模板、scheme/私网
白名单、`{query}` 转义、标题→文件名清洗、规则 JSON 解析失败回落、
标签上限与销毁、退出清理、`onDisable` 后不再有存活 WebView、保存到库的
路径校验与不覆盖、无 WebView 时开关与命令不可用。

门禁：`scripts/check.sh`（gofmt / vet / `go test ./...` / 五个平台
`CGO_ENABLED=0 go build` / 版本一致 / 资源检查）。阅读视图脚本与拦截规则已
`go:embed`，不新增随包资源，因此 `check-resources.sh` 无需改动（§12）。
macOS（P0）先跑通 `make build` 与真机手验，再验
Windows/amd64（P1，需 WebView2 Runtime），最后 Windows/arm64 与
Linux/amd64（P2，需 WebKitGTK ≥ 2.24）编译。人工验收项拟为 K1 ~ K16，写入
`docs/ACCEPTANCE.md`：开关与设置页出现/消失、日期格式预览、目录浏览与库外
拒绝、今日日记创建与重复打开、前后导航、启动打开、模板插入光标处、从模板
新建、模板变量与非法格式、切语言后界面文案；K11 ~ K16 为 Web viewer：
地址栏与导航按钮、外链接管与关闭后回落、阅读视图切换、保存到库、
拦截规则生效与「清理浏览数据」、无 WebView / 跨平台的降级说明。

## 9 边界、风险与不做的事

- **不是完整 Moment**：只支持 §5 的 token 白名单；遇到 `Do`、`X`、`gggg` 等
  一律报错并在设置页说明，而不是悄悄产出 `2026-11-06` 之外的怪名字。
- **不执行模板里的代码**：`{{title}}/{{date}}/{{time}}/{{date:FMT}}/
  {{time:FMT}}` 之外原样保留；不做 Templater、不做 JS、不做网络请求。
- **前后导航只在已有日记间跳**：不自动创建缺失日期的笔记；若与 Obsidian
  实际行为不同，作为人工验收项对照后再决定。
- **不绑定热键（v1）**：`keymap.Bindings` 是包级变量，`Label/Desc` 里是中文
  硬编码，包级初始化时调 `locale.Xxx()` 不会跟随语言切换。v1 命令走命令面板
  + 菜单 + ribbon；要加热键得先把这两列改成 `func() string`，是独立的改动。
- **不写用户库的配置目录**：目前 Xinkgo 的状态全在应用目录，向库里写
  `.xink/*.json` 会牵出文件监视、迁移与「应用能不能改用户库」的新问题。
- **符号链接**：与现有 `vaultHost.within` 一样，路径校验不做
  `EvalSymlinks` 追链；库内软链指向库外属已知残余风险，登记不修。
- **一个选择框**：与「一个插件一次一个可见表面」一致，同时只开一个模板
  选择框，不叠模态框。
- **Web viewer 不是完整浏览器**：不 vendor Mozilla Readability（阅读视图是
  启发式，保真度有限）、不订阅远程规则列表、不做「更新频率」、不允许下载、
  不允许脚本自弹窗、不装扩展、不支持多账户会话隔离（只有一个命名会话）、
  不提供 DevTools、不加载 `file:` 页面。规则正则是 WebKit 的受限子集，
  不承诺兼容 EasyList 等外部列表。
- **私网地址默认封禁是保守口径**：开 `AllowLocal` 后网页可访问本机服务；
  DNS rebinding 与符号链接一样属于已知残余风险，登记不修。三端
  `CompileContentRules` 行为不同（Windows 不拦 WebSocket、Linux 需
  WebKitGTK ≥ 2.24），验收时逐端对照。
- 不引入 cgo / Node / 新 Go 依赖；不改 `mygo.json` 版本；不改 shim 与兼容
  清单；不把这三个能力做成 C3/C4 降级插件。

## 10 待拍板（已按左列假设写，改动成本都不大）

| # | 假设 | 若改 | 影响 |
|---|---|---|---|
| 1 | 三个插件默认关闭 | 改 `Default: true` | 老用户下次启动会多出 ribbon 图标与命令；`OpenOnStartup` 仍默认关，不会自动建笔记；Web viewer 无 WebView 时仍不可用 |
| 2 | 选项存 `plugins.json`（应用级） | 改为每库 `<vault>/.xink/*.json` | 更贴近 Obsidian、切库各管各的；但要新增库内写盘、监视与迁移，工作量明显变大 |
| 3 | 路径是库内相对、目录为空即库根/未设置 | 默认 `Daily notes` / `Templates` 并首次使用时建目录 | 首次体验更整齐，但与 Obsidian 默认不同 |
| 4 | 前后导航只跳已有日记 | 缺日期时自动创建 | 需明确「自动创建」是否算用户预期之外地写盘 |
| 5 | v1 不加热键 | 要热键 | 先做 §9 的 keymap 文案改造 |
| 6 | Web viewer 在**预览面板以标签页**打开（2026-11-06 按需求从「独立窗口」改来，见 §12 第 4 条） | 做成主工作区笔记标签，或改回独立窗口 | 前者要重构 `view.Tabs` 容纳非笔记标签；后者要另开窗口，且 `ui.WebView` 只接受同窗口的页 |
| 7 | 远程页用持久命名会话 `xinkgo-webviewer` | 改 ephemeral / 每次退出清理 | 更像隐私窗口，但登录态不保留；`ClearOnQuit` 已提供折中 |
| 8 | 广告拦截默认开、只随包几十条规则 | 默认关 / 支持远程订阅 | 远程订阅要新增网络拉取、缓存、定时更新与规则格式校验，是独立一期 |
| 9 | 允许本地地址默认关、外链沿用 `preview.link_open` | 在 Web viewer 页另设一个「应用内打开外链」开关 | 会出现两个真值来源，需先决定谁覆盖谁 |

## 11 参考

- [obsidian-core-plugins.md](obsidian-core-plugins.md)：Xinkgo 内置核心插件注册表与对照表
- [obsidian-plugins.md](obsidian-plugins.md)：兼容宿主的边界、贡献面与分阶段路线
- [ARCHITECTURE.md](../ARCHITECTURE.md)：分层、线程模型、状态与扩展点
- [CONTRIBUTING.md](../CONTRIBUTING.md)：分支与提交约定
- Obsidian 帮助：<https://help.obsidian.md/Plugins/Daily+notes>、
  <https://help.obsidian.md/Plugins/Templates>、
  <https://help.obsidian.md/plugins/web-viewer>
- MyGo v0.7.4 `WebView` / `Page` / `PageOptions` / `ContentRules` / `Session`
  API（`webview.go`、`window.go`、`page.go`、`content_rules.go`、`session.go`）

## 12 实现偏差与后续修复

1. **脚本与规则改为 `go:embed`，不放 `resources/`。** 评审稿计划把
   `reader.js` 与 `content-rules.json` 作为随包资源并扩展
   `scripts/check-resources.sh`。实现时改为放在
   `internal/app/webviewer_assets/` 并用 `//go:embed` 编进二进制：这两份内容
   只由 Go 侧 `Page.Eval` / `CompileContentRules` 读取，并不通过自定义协议
   喂给页面，随包反而多出「打包漏文件」一类故障。仓库里已有的
   `build/` 产物是早于本轮构建的，若把新文件加进资源门禁，会在旧产物上
   误报失败。嵌入后 `check-resources.sh` 无需改动，也不引入新依赖或 cgo。
2. **发布与释放加了 `mygo.App.IsReady()` 守卫。** `setWorkspace` 与
   `webDispose` 原先无条件调 `RunOnMain`；在没有事件循环的单测里，
   `onMain` 会把回调排进队列并一直等它执行，导致死锁。守卫让启动前与
   无窗口环境直接跳过界面写入；生产路径在 `WhenReady` 之后仍然照常走主线程。
   同理 `webEnsureWindow` 在应用未就绪时不建窗，避免外链提前触发 panic。
3. **补上 Templates 的 ribbon 按钮（2026-11-06 修复）。** 评审稿把 Templates
   的入口写成「命令面板；菜单」，实现里注册表也没给 `templates` 写
   `Ribbon: true`，`coreRibbonActions` 又是独立的 `switch`，两处真值来源
   不同步，结果 Templates 开启后目录树左侧没有快捷按钮。排查时先用启动日志
   确认 `mergeContributions` 只发布 2 个图标（日历 + 地球）、绘制层也确实拿到
   这 2 个，定位到缺的是 Templates 而不是发布链路。修复：注册表给
   `templates` 加 `Ribbon: true`，`coreRibbonActions` 改为**由注册表驱动**
   （`CoreByID(id).Ribbon` + `coreRibbonIcons` + 该插件第一条命令），并加
   一致性单测遍历注册表，保证声明了 ribbon 的插件一定贡献可路由的按钮。
   现在三个插件开启后启动日志为 `ribbon published count=3`。
4. **Web viewer 从独立窗口改为预览面板标签页，模板列表重做（2026-11-06）。**
   需求反馈指出：网页不该另开窗口，应在预览界面里以 tab 打开；模板配置页在
   选定目录后的模板清单也太粗糙。改动：
   - WebView 改由**主窗口** `s.win.NewWebView(...)` 创建（`ui.WebView` 只接受
     同窗口的原生页），`internal/view/webviewer.go` 从「独立窗口的 chrome」
     重写为预览面板的标签条 + 地址栏工具条；`previewPanel` 在
     `WebTabActive != ""` 时改画 `webPane`；`webPublish` 把 `WebTabs` 推给
     `Workspace`；`webOpen` 用 `SetPreviewShown(true)` 保证面板可见；关掉
     「Page preview」也能因网页标签而保留预览面板。`webDispose` 不再销毁窗口，
     只销毁页面并清空标签。真机启动验证：`web tab made wsTabs=1`，无跨窗口
     警告或 panic。
   - 模板清单改为卡片式：图标、名称 + `.md` 徽标、库内相对路径 / 大小 /
     修改时间、计数徽标、居中空状态与坏文件分区；`readTemplateDir` 现在返回
     带元信息的结构，仍在切页/改目录时读一次，不每帧读盘。
   - 顺带修掉真机日志暴露的**内容拦截规则编译失败**：WebKit 的
     `load-type` 必须是数组 `["third-party"]`，评审稿写成了字符串，导致整份
     规则列表被拒。已加单测校验每条规则的 trigger 字段形态。
5. **网页标签长标题溢出（2026-11-06 修复）。** `webTab` 原先用
   `MaxWidth(tabMaxWidth)`，而 `ButtonBase` 是 `Shrink(0)`，放在可横向滚动的
   `ScrollHorizontal` 里时，网页标题（常比文件名长得多）仍会把标签撑开，挤掉
   自身的关闭按钮和后面的标签。修复三层：标签改为**固定宽度**
   `Width(150)`；标题在进布局前由 `shortWebTitle` 按 **rune** 截到 18 个字符
   并加省略号（中文不会切在半个字上）；文本保留 `NoWrap()` + `Ellipsis()`，
   标签再加 `ClipX()` 兜底，hover 显示完整标题。headless 布局实测：第一个标签
   `x=71.5 w=150`、关闭按钮 `x=203.8`、第二个标签 `x=223.5`，三者不重叠。

其余按评审稿落地：三个默认关闭的内置核心插件、`plugins.json` 的 `options`
字段、动态 Plugin options 侧栏、内置与第三方贡献合并的命令/ribbon 分发、
`onDisable` 释放钩子、`internal/templating` 与 `internal/webviewer` 两个纯函数
包、九个命令与三页设置、catalog 文案与 `go tool mygo generate`、K 组人工验收。
§10 的九条待拍板项均按左列假设实现；§2 的入口表已按修复改为 Templates 也有
ribbon 图标，第 6 条按需求改为预览面板标签页。
