# 设计：把 xink 复刻到 xinkgo（原生 UI）

> 需求：[requirements/xink-port.md](../requirements/xink-port.md)
> 计划：[exec-plans/xinkgo-port.md](../exec-plans/xinkgo-port.md)
> 已确认选型：原生 UI（MyGo `ui` 包）· AI 全量（P2）· 一次性交付 P0 ·
> 持久化与 xink 兼容。

## 1 背景与目标

把 xink（Rust + GPUI，117 项功能）的行为对齐复刻到 xinkgo（Go + MyGo v0.7.3），
全部界面用 `github.com/egoist/mygo/ui` 以 Go 绘制，不引入 webview 与前端工具链。

约束（按优先级）：

1. **行为等价**：以 xink 代码为准，包括其边界规则。
2. **数据兼容**：`config.json` / `open_notes.json` / `vaults.json` 与 xink 同构，
   xink 用户可直接切换。
3. **平台优先级**：P0 macOS → P1 Windows/amd64 → P2 其余
   （[HARNESS.md](../HARNESS.md)）。
4. **不引入 cgo**：保持 `CGO_ENABLED=0` 与跨平台交叉编译。

## 2 技术底座核查（MyGo v0.7.3 提供了什么）

| xink 需求 | MyGo 底座 | 结论 |
|---|---|---|
| Markdown 预览 R-5 | `ui/markdown`：`View(c, src, *Options)`、`Options.OnLink`、`Options.LoadImage` + `markdown.FileImages(dir)`、`Options.CodeBlock`、`Extensions`（CommonMark+） | 直接用；图片与链接点击走回调 |
| 流式 AI 输出 R-10.2 | `markdown.Stream(c, src, o)` + `Reveal` | 直接用 |
| 编辑器 R-4 | `ui/editor`：`State{Buffer, SoftWrap, LineNumbers, TabSize, Indent, Font}`、`Buffer.SetLanguage`、`View(c, st)`；多光标、查找替换、补全、hover、诊断、折叠 | 直接用 |
| Markdown 编辑区高亮/装饰 R-4.10、R-4.11 | `editor` 的 `Language`/`Tokenizer`/`Highlighter`/`Completion`/`Diagnostic` 接口 | 自研 Markdown Language + 校验诊断 |
| AI 修订逐块接受 R-10.5 | `editor.DiffView` + `DiffState`（侧栏/统一、逐 hunk 动作） | 直接用 |
| 双链补全浮层 R-8.9 | `ui.Autocomplete` + `editor.CompletionMenu` | 直接用 |
| 命令面板 / 抽屉 R-8.11、R-8.15 | `ui.Overlay`、`ui.Modal`、`ui.Popover`、`ui.PopoverBase` | 自研浮层内容 |
| 列表与树 R-1 | `ui.List` + `ListState`、`ui.Collapsible`/`Accordion`、`ScrollState` | **自研文件树**（缩进 + 展开态） |
| 多标签 R-3 | `ui.TabsParts` | 自研标签条（需与会话落盘联动） |
| 三栏布局 R-8.22 | `ui.Row`/`ui.Column`/`ui.Box` + `Fixed` | **自研**分栏与可拖分隔条 |
| 滚动同步 R-5.9/5.10 | `ScrollState` | 自研锚点换算（xink `core/sync.rs` 是纯数据，可直译） |
| 外部文件监听 R-7 | `mygo.WatchFiles(ctx, path, FileWatchOptions)` + `FileWatcher` | 官方 API，不用第三方库 |
| 打开库对话框 R-1.1 | `mygo.Dialog`（`OpenDialogOptions`） | 直接用 |
| 明暗主题 R-5.8 | `mygo.Theme`（`Theme.SetSource`）+ `ui.Theme` | 直接用 |
| 剪贴板/拖放 R-1.15/1.16 | `mygo.Clipboard`、`ui.Drop`/`DropData`/`DataDragOver` | 直接用 |
| 单实例 R-8.21 | 无官方 API | 自研（pid 文件 + 存活检查，与 xink 同策略） |
| 快捷键 R-8.19 | `ui.Context.Shortcut`、`ui.Key` | 直接用（键位表自研） |
| 通知 / 菜单 / 托盘 | `mygo.NewNotification`、`NewMenu`、`NewTray` | 直接用 |

**结论**：原生 UI 路线的两个最大风险（Markdown 渲染、代码编辑器）都有官方底座；
需自研的是文件树、标签条、分栏布局、命令面板内容、滚动同步算法、索引与搜索
（领域层）。整体可行。

## 3 被否决的方案

| 方案 | 否决理由 |
|---|---|
| **Web 前端（webview）** | 需引入 Node/Bun 前端工具链与构建步骤，违背"单二进制、无前端工具链"的现有骨架定位；内存占用更高；应用运行依赖系统 webview。UI 复刻更快，但代价在长期形态上不可接受（用户拍板选择原生 UI）。 |
| **混合（编辑/预览 Web + 其余原生）** | 两套 UI 的主题、字体、滚动同步与状态同步要跨 IPC 维护，复杂度高于任一单一方案；且预览与编辑是核心高频路径，跨边界同步的抖动风险最大。 |
| **索引落盘（SQLite 等）** | xink 明确索引是磁盘的派生数据、不落盘（`xink/src/app.rs:184-212`）；落盘会引入额外依赖与一致性问题。Go 侧同样内存重建。 |
| **自己写 Markdown 解析器** | `ui/markdown` 已提供渲染与扩展；自研解析器要同时保证渲染与 facts 提取一致，成本无收益。facts（headings/links/wikilinks/tags）仍需自研扫描器，但那是**提取**不是渲染。 |

## 4 架构与分层

```
cmd/xinkgo            入口：装配
internal/app          生命周期、启动恢复编排、菜单、窗口、单实例
internal/view         UI 层（workspace / file_tree / tabs / editor / preview /
                      command / search / outline / backlink / tags / rail /
                      drawer / settings / vault_manager / completion）
internal/config       AppConfig（basic/editor/preview/highlight/ai）+ sanitize + 迁移
internal/state        open_notes / vaults / layout / ai_sessions 读写与原子写
internal/vault        扫描、忽略规则、增删改移、注册表、状态校验
internal/note         Note（内容/dirty/mtime/version）、原子写、外部变更取回
internal/fs           unique_name / rename / 跨设备移动 / 导入
internal/session      tab_key / 保序去重 / active 回落（纯规则）
internal/markdown     facts 提取、wikilink 改写、sanitize、装饰 span
internal/index        NoteIndex + 派生表、双链解析、反链、标签
internal/search       全文检索（逐行、跳 >4MB、截断 160）
internal/fuzzy        模糊匹配与打分
internal/sync         锚点表、方向锁、手势窗（纯数据）
internal/watcher      mygo.WatchFiles 封装与事件分类
internal/datafile     JSON/YAML/TOML 判定、格式化、校验
internal/media        文件类型、图标、可编辑文本、编辑器语言
internal/highlight    3 套高亮主题与旧值迁移
internal/keymap       键位表与覆盖解析
internal/ai/...       Provider / agent loop / tools / skills / tasks（P2）
```

依赖方向（单向，不得反向或循环）：

```
cmd → app → view → {vault, note, session, index, search, fuzzy, markdown,
                     config, state, sync, watcher, datafile, media,
                     highlight, keymap, fs}
                    config ← state（读写）
                    view 依赖 domain；domain 不依赖 view/app
```

- `internal/view` 之下的一切必须能在无运行应用的测试里跑（xink 的经验：
  `tests/core/` 约 380 个用例，UI 只测纯函数）。
- `internal/app` 是唯一编排启动恢复与生命周期的地方。

## 5 数据模型（与 xink 同构）

数据目录：`mygo.App.Path(mygo.PathUserData)`（macOS
`~/Library/Application Support/Xinkgo/`）。

| 文件 | 结构 | 写盘 |
|---|---|---|
| `config.json` | `{basic, editor, preview, highlight, ai}`，字段照抄 xink（`src/core/config.rs:344-353`），未知字段忽略、缺字段补默认，读后 `sanitize()` 钳制 | 普通写 |
| `open_notes.json` | `{paths: [string], active: string?}`，只存路径不存正文 | 普通写 |
| `layout.json` | 自研布局结构 `{version, left, center, right, …}` | 普通写 |
| `vaults.json` | `{vaults:[{id,name,path}], active}`，`id = "v<%x>-<%x>"`(path hash, unix nano) | **原子写** |
| `ai_sessions.json` | `map[笔记路径][]StoredBlock`，外部标签枚举 | **原子写** |
| `xinkgo.pid` | 单实例 pid | — |

关键兼容决策：

- **配置字段逐字照抄** xink（含 `preview.legacy_code_theme` 迁移一次 →
  `highlight.theme`；`sync_mode` 的 `"realtime"` 别名；`image_display` 未知值兜底；
  `num_ctx=8192`、`keep_alive="5m"` 视为未单独设置）。
- **`layout.json` 不复用 xink 的 `DockAreaState`**：我们的布局系统不同（自研三栏
  而非 gpui DockArea）。存自己的结构并带 `version`；读到 xink 的 `version: 6`
  或任何不匹配的版本 → **丢弃重建默认布局**，这与 xink"版本不符即重建"的行为
  等价（`xink/src/app.rs:550-554`）。
- **读失败一律降级为默认值，绝不阻塞启动**（xink `workspace.rs:188-206`）。
- 键位映射 `editor.keymap`：动作名照抄 xink 的 17 个动作。

## 6 关键流程

### 6.1 启动恢复（顺序不可颠倒，照抄 xink `app.rs:569-572`）

```
1. 加载 config（失败 → 默认 + sanitize）
2. 加载 vaults（含 vault.json 迁移），解析 active（失效 → 第一个可用库）
3. 加载 layout（版本不符 → 默认布局）
4. restore_open_notes：过滤失效路径、保序去重、只恢复属于当前库的
5. activate：走**同一个** openNote 入口恢复每个标签
6. 全部恢复完再激活 active（避免预览反复重建）
7. 恢复后写回一次（自愈，剔除失效条目）
8. sync_placeholder：无笔记恰好一个空白页、有笔记零个
```

### 6.2 打开笔记（唯一入口）

```
openNote(path):
  key = canonicalize(path)
  if 已开: 只选中 + 聚焦 + 更新 active（不新建/不重读盘/不重置光标滚动/不改顺序）
  else:    优先认领空白页，无空白页才新建
  立即落盘 open_notes.json
```

### 6.3 编辑与保存

- 防抖 300ms（配置 200–60000）；`Cmd+S` 立即保存；调度新保存时取消旧任务。
- 原子写：同目录 `.md.tmp` + rename，失败清理临时文件。
- 保存后刷新索引（内容变更不产生结构事件）。
- 自动保存关闭时仍刷新预览。

### 6.4 外部变更（三态）

```
磁盘内容与内存一致 → 一个字节都不回填（否则光标被重置）
有未保存改动       → 保留并提示
否则               → 回填，保住光标与滚动位置；打字期间延后 800ms，只排一次
忽略应用自身 *.md.tmp
```

### 6.5 滚动同步

直译 xink `core/sync.rs` 的纯数据结构：`AnchorTable`（块锚点）、`ScrollAnchor`、
`SyncGuard`（方向锁 + 回显吸收）、`GestureGuard`（手势静默窗口）。三档模式：
ratio / anchor / on_stop。

## 7 线程与并发

- MyGo 的回调与窗口方法在**主线程**；视图函数每帧执行，不做阻塞 I/O。
- 索引重建、全文搜索、AI 请求放 goroutine，结果回到主线程更新 UI。
- 文件监听由 `mygo.WatchFiles` 驱动（事件驱动，不轮询），事件经 channel 交给
  主线程处理。
- 索引需要并发保护（`sync.RWMutex`）：后台重建与增量更新、UI 读取并存。

## 8 测试计划

**自动化（Go 单元测试，参照 xink `tests/core/` 约 380 个用例）**

| 包 | 重点 |
|---|---|
| `vault` | 扫描、忽略规则、增删改移、注册表状态与回落（xink 30 例） |
| `fs` | unique_name、rename_name、跨设备移动（23 例） |
| `note` | 原子写、增量同步、外部变更取回（21 例） |
| `session` | tab_key、保序去重、失效判定、active 回落（13 例） |
| `markdown` | facts 解析、wikilink 改写、图片 URL、畸形输入（63 例） |
| `index` | 增量 == 全量、反链、标签、双链解析（24 例） |
| `sync` | 锚点换算、方向锁、手势窗（22 例） |
| `config` | sanitize 钳制、默认值、旧值兼容（55 例） |
| `state` | normalize_open_notes、active_within、迁移（12 例） |
| 其余 | fuzzy、media、keymap、highlight、datafile、search、watcher |
| `view` | `ui.NewTester` 点击/输入（xink 的 12 个 UI 测试同思路） |

**人工验收**（登记到 `docs/ACCEPTANCE.md`）：视觉密度与主题、滚动同步手感、
拖放导入、补全浮层、抽屉动画、命令面板、外部改文件的提示、重启恢复。

**不测**：写真实用户数据目录的函数（用 `mygo.App.SetPath` 或临时目录重定向）、
真实网络请求（用 httptest 或桩）。

## 9 风险

| 风险 | 影响 | 缓解 |
|---|---|---|
| `ui/markdown` 渲染细节（表格、任务列表、脚注）与 xink 的 TextView 不一致 | 预览观感差异 | 先跑 example 验证能力边界，差异登记为人工验收项 |
| 编辑区 Markdown 装饰与高亮需自研 Language | 工作量 | 先接默认高亮，装饰（行首标记淡化）后置 |
| 文件树/标签/分栏自研 | 工作量大 | 用 `ui.List` + `Collapsible` 最小实现，先跑通再美化 |
| 布局不复用 xink `DockAreaState` | xink 用户首次启动布局重置 | 已决策（§5），行为等价 |
| AI 全量（P2）依赖外部服务 | 无法自动化端到端 | Provider 走接口，用 httptest 桩测流式解析 |

## 10 里程碑

| 阶段 | 内容 | 完成判据 |
|---|---|---|
| **M0** | 骨架 + 领域层底座（config/state/vault/fs/session/note/media） | 单测绿，`make check` 绿 |
| **M1 = P0** | R-1 库与文件树、R-2 增删改移、R-3 多标签、R-4 编辑保存、R-5 预览、R-6 会话恢复、R-7 外部监听 | 主链路人工跑通（一次性交付验收） |
| **M2 = P1** | R-8 索引/双链/反链/标签/搜索/大纲/命令面板/主题/设置 | 同上 |
| **M3 = P2** | R-10 AI 全量 | 同上 |

## 11 开放问题

1. `ui/markdown` 是否支持任务列表/表格/脚注 —— 影响 R-5 的验收标准细节。
2. 编辑区 Markdown 装饰是否值得自研 tokenizer —— 影响 R-4.10 的取舍。
3. 图片显示：用 `markdown.FileImages`（文件系统直读）还是先转 data URL ——
   xink 因框架限制转 data URL，MyGo 有 `FileImages`，倾向直读。

以上在实施中遇到即记录到 [exec-plans/xinkgo-port.md](../exec-plans/xinkgo-port.md)
的"决策记录"。
