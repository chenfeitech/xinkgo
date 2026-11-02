# xinkgo 功能清单与实现对照

> **性质**：以 [XINK_FEATURES.md](XINK_FEATURES.md)（Rust 版 xink 的已实现清单）
> 为骨架，逐项核对 xinkgo（Go + MyGo）当前代码，标出**已实现 / 部分 / 未实现**。
> **基准**：提交 `1c2e53a`（分支 `feature/preview-links`，工作区有未提交改动），
> 核对日期 2026-10-31。`internal` + `cmd` 共 108 个 `.go`、非测试约 14.9k 行，
> 42 个测试文件、458 个 `Test*` 函数。
> **状态标记**：✅ 已实现｜⚠️ 部分实现｜❌ 未实现｜➖ 有意不做（沿用 xink 的决定）
>
> 列「实现位置」写 `文件:行` 的入口函数；同一功能只标最关键的那一处。

---

## 1. 库与文件树

| 功能点 | 状态 | 实现位置 | 说明 |
|---|---|---|---|
| 打开文件夹作为库 | ✅ | `internal/app/app.go:356` `openVaultDialog` | 库根就是普通目录 |
| 多库注册表（新增/切换/改名/重新定位/删除） | ✅ | `internal/vault/vault.go:198` `Registry`、`internal/app/vaults_window.go:14` | 五项齐全；删除只移除记录 |
| 库状态徽标（6 态） | ✅ | `internal/vault/vault.go:264` `StatusOf` | 菜单与库窗口都显示不可用原因 |
| 目录树扫描（全量、非懒加载） | ✅ | `internal/vault/vault.go:42` `Scan` | 目录在前，同层按名排序 |
| 忽略规则 | ✅ | `internal/vault/vault.go:22` `ignored` | 另恒跳 `*.md.tmp` 与 `.xink`（`internal/media/media.go:60`） |
| 文件类型图标（10 类） | ✅ | `internal/media/media.go:104` `KindOf`、`internal/view/tree_icons.go:48` | |
| 新建笔记 / 目录 | ✅ | `internal/app/files.go:28` / `:49` | 自动补 `.md`，重名递增（`internal/fs/fs.go:21`） |
| 重命名 | ✅ | `internal/app/files.go:69`、`internal/fs/fs.go:36` | |
| 删除（含非空目录） | ✅ | `internal/app/files.go:94` / `:157` | 拒绝库根与库外路径（`fs.RemoveAll:168`） |
| 移动到…（对话框） | ✅ | `internal/app/files.go:173` `moveItem` | |
| **树内拖拽移动** | ❌ | — | 只有「从外部拖入」`internal/view/file_tree.go:411` `takeDrop` |
| 外部导入（访达拖入） | ✅ | `internal/app/files.go:213` `importFiles` | 逐个返回成败，单文件失败不影响其余 |
| 粘贴（访达文件 / 截图） | ✅ | `internal/app/files.go:250` `pasteIntoVault`、`pastePicture:271` | 菜单 `Paste into Vault`（`Cmd+Shift+V`） |
| 右键菜单 | ⚠️ | `internal/view/file_tree.go:347` `menu` | 5 项：新建笔记/新建文件夹/重命名/移动到…/删除。缺「复制相对路径」「复制为 Markdown 链接」「插入到当前笔记」「粘贴到此目录」 |
| 目录树自动刷新 | ✅ | `internal/app/watch.go:67` `handleEvents` | 仅结构变化才重扫 + 重建索引 |
| 保状态合并刷新 | ⚠️ | `internal/view/file_tree.go:112` `SetRoot` | 展开态与选中保留；滚动位置未显式保存/恢复（依赖 `ui.ListState`） |
| 切换笔记后定位 | ✅ | `internal/view/file_tree.go:592` `Reveal`、`internal/view/workspace.go:400` `followActiveNote` | 每帧认「当前笔记」：点标签页 / 跟双链 / 命令面板 / 重启恢复都展开祖先、选中并滚入视口 |
| 库选择器（树底部） | ❌ | — | 切换库只走菜单 `Vault` 与设置 ▸ Vault |

## 2. 笔记编辑

| 功能点 | 状态 | 实现位置 | 说明 |
|---|---|---|---|
| 源码模式编辑 | ✅ | `internal/view/tabs.go:121` `newEditor` | MyGo `editor.Buffer` |
| 自动保存（防抖，默认 300ms） | ✅ | `internal/view/tabs.go:270` `scheduleSave`、`SaveDue:286` | 写盘只在画帧/主线程做，不用定时器 goroutine |
| 立即保存 `Cmd+S` | ✅ | `internal/app/menu.go` `Save` | 评审未收尾时拒绝保存并提示 |
| 原子写盘 | ✅ | `internal/note/note.go:63` `Save` | 同目录 tmp + rename |
| 脏标记 | ✅ | `internal/view/tabs.go:471` `Title`、`internal/view/workspace.go:655` 状态栏 | |
| 外部修改检测 | ✅ | `internal/note/note.go:119` `TakeExternalText`、`internal/view/tabs.go:400` `Reload` | 一致则不回填 |
| **放弃改动重新加载** | ❌ | — | 只提示并保留用户内容，无「丢弃并重载」入口 |
| 标签唯一 | ✅ | `internal/view/tabs.go:84` `Open`、`internal/session/session.go:18` `TabKey` | |
| 标签持久化与恢复 | ✅ | `internal/state/state.go:70`、`internal/app/app.go:260` `restore` | 只恢复属于当前库的标签，恢复后写回 |
| 空白占位页 | ✅ | `internal/view/workspace.go:541` `editor` | |
| 缩进 / 软换行 / 行号 | ✅ | `internal/view/apply.go:29` `applyEditor` | |
| 格式工具栏 | ✅ | `internal/view/workspace.go:404` `toolbar` | 14 个按钮（xink 13 个 + 行内代码） |
| 格式快捷键 | ✅ | `internal/app/menu.go` `Format`、`internal/keymap/keymap.go:34` | |
| 编辑区语法高亮 | ✅ | `internal/view/tabs.go:121` | md / json / yaml / toml 按文件名 |
| **编辑区 Markdown 装饰**（行首标记淡化、链接文字着色） | ❌ | — | xink 有，xinkgo 未做 |
| 数据文件校验标记 | ✅ | `internal/view/workspace.go:496` `dataDiagnostics` | 波浪线；JSON 有行列 |

## 3. 预览

| 功能点 | 状态 | 实现位置 | 说明 |
|---|---|---|---|
| Markdown 渲染 | ✅ | `internal/view/workspace.go:571` `previewPanel` | 用 MyGo `ui/markdown`，能力由框架决定 |
| 本地图片内联 | ✅ | `internal/view/workspace.go:589` `markdownOptions` | `FileImages` 按目录解析，非 data URL |
| 图片缓存 | ➖ | 框架内 | 无自研指纹缓存，无 5MB 上限 |
| 图片显示方式（适应宽度 / 隐藏） | ✅ | 同上（`ImageDisplay`） | |
| **图片宽度百分比 20–100** | ❌ | `internal/config/config.go:98` 有字段 | 设置里有滑块，但 preview 无读取点 → 不生效 |
| **图片点击放大 / 向左伸展** | ❌ | — | 两项都未做 |
| 代码块语法高亮（编辑区与预览同源） | ✅ | `internal/highlight/highlight.go:95` `EditorTheme`、`internal/view/apply.go:63` | |
| 高亮主题 3 套 | ✅ | `internal/highlight/highlight.go:22` `Presets` | |
| 滚动同步（双向，三档） | ✅ | `internal/view/scroll.go:19` `followScroll`、`internal/sync/sync.go:94` `BlockSpans` | 方向锁 + 回显吸收 |
| **滚动同步调试数字** | ❌ | `config.go:104` `DebugScrollSync` 有字段 | 无读取点 |
| 版面密度 / 最大宽度 | ✅ | `internal/view/apply.go:78` `spacing`、`previewWidth:91` | |
| 链接点击行为 | ✅ | `internal/view/workspace.go:607` `openLink`、`internal/app/web.go:101` | 库内打开笔记；外部链接按设置走系统浏览器或应用内窗口 |
| **畸形 Markdown 防御性预处理** | ➖ | — | 由框架解析器承担，未自研 |

## 4. 数据文件（JSON / YAML / TOML）

| 功能点 | 状态 | 实现位置 | 说明 |
|---|---|---|---|
| 类型判定 | ✅ | `internal/datafile/datafile.go:45` `KindOf` | |
| 编辑区高亮 | ✅ | `internal/view/tabs.go:121` | |
| 预览格式化 | ⚠️ | `internal/view/workspace.go:467` `previewText` | JSON 完整；YAML/TOML 只判定不格式化（缺 yaml 库，已决策） |
| 校验（行/列 + 原因） | ⚠️ | `internal/datafile/datafile.go:70` `Check` | 仅 JSON 报行列；YAML/TOML 不报 |
| 大文件保护 | ✅ | `internal/datafile/datafile.go:112` `TooLarge` | 默认 1MB |
| **设置里的数据文件旋钮**（缩进 / 上限） | ❌ | `config.go` `Editor.DataFile` 有字段 | 无 UI |

## 5. 导航、索引与结构

| 功能点 | 状态 | 实现位置 | 说明 |
|---|---|---|---|
| 全库索引（标题/标签/双链/反链） | ✅ | `internal/index/index.go:152` `Build` | 后台构建，不落盘；`Reindex:171` 增量 |
| 双链解析（路径 → 文件名 → 标题，歧义可辨） | ✅ | `internal/index/index.go:377` `resolveIn` | 含 `Ambiguous` |
| 预览里双链可点击 | ✅ | `internal/markdown/links.go:47` `Links` | 支持 `别名`、`#锚点` |
| `[[` 补全 | ✅ | `internal/view/completion.go:20` | 写库内相对路径 |
| 反向链接面板 | ✅ | `internal/view/panels.go:12` | |
| 标签面板（聚合 + 展开） | ✅ | `internal/view/panels.go:50` | 标签来源含 frontmatter（`internal/markdown/facts.go:342`） |
| 大纲面板（实时解析，点击跳行） | ✅ | `internal/view/outline.go:89` | |
| 右侧按钮条 + 抽屉 | ✅ | `internal/view/outline.go:32` `rail`、`:73` `drawerPanel` | |
| 命令面板 `Cmd+P` | ✅ | `internal/view/palette.go:44` | 前 50 条、命中高亮；另列 Skill 与 AI 动作 |
| 全文搜索 `Cmd+Shift+F` | ✅ | `internal/view/search_panel.go:51` | 防抖、后台、点击跳行 |
| 模糊匹配与打分 | ✅ | `internal/fuzzy/fuzzy.go:100`、`internal/search/search.go:43` | |
| 明暗主题切换 | ✅ | `internal/app/app.go:415` `toggleTheme` | |

## 6. AI 协助（默认关闭）

| 功能点 | 状态 | 实现位置 | 说明 |
|---|---|---|---|
| 对话条 `Cmd+Shift+A` | ✅ | `internal/view/ai_bar.go:132` `aiBar` | 按笔记归属 |
| 会话持久化（30 会话 / 200 块） | ✅ | `internal/state/ai_sessions.go:16`、`internal/view/ai_bar.go:513` | |
| 快捷动作（内置 6 项 + 命令面板入口） | ⚠️ | `internal/config/ai.go:182` `DefaultActions`、`internal/view/ai_bar.go:287` `runAction` | 整篇 + 选区；**增删改无 UI**，只能改 `config.json` |
| **提示词变量**（`{{known_tags}}`） | ❌ | — | |
| 结果落法 | ⚠️ | `internal/view/review.go:36` / `:60` | 一律走评审 diff；`Apply=insert/replace` 自动落盘未实现（已决策） |
| 修订评审（逐块接受/拒绝） | ✅ | `internal/view/review.go:100` `reviewPane` | 整篇与选区两种 |
| 改前快照 + 取回窗口 | ✅ | `internal/snapshot/snapshot.go:53`、`internal/app/history_window.go:27` | 上限 200，取回前先存当前 |
| Provider：Ollama / OpenAI 兼容 | ✅ | `internal/ai/ollama.go:14`、`internal/ai/openai.go:16` | |
| 流式解析（分帧 + 语义） | ✅ | `internal/ai/framing.go`、`internal/ai/parse.go:41` | NDJSON / SSE |
| 重试与备用模型 / 两类超时 | ✅ | `internal/ai/ai.go:146` `Ask` | |
| **场景路由**（对话/动作/Skill/Agent 各指定模型） | ❌ | `internal/config/ai.go:73` 有字段 | 仍无读取点；但对话可在 AI 栏选模型（`internal/view/ai_bar.go:283` `aiModelPick`），动作可用 `AiAction.Model` |
| 上下文裁剪 | ⚠️ | `internal/view/ai_bar.go:620` | 只用全局 `MaxContextChars`；模型侧 `ContextLimit` 未参与 |
| 密钥管理 | ⚠️ | `internal/ai/ai.go:95` `keyOf` | 明文 + 环境变量已实现；`keychain` 仍未实现（Sanitize 一律归为明文） |
| 连接测试（探针） | ✅ | `internal/ai/probe.go:34` `Try`、`internal/app/settings_ai.go:439` | |
| 从 Ollama 拉模型列表 | ✅ | `internal/ai/discover.go:88` `OllamaModels`、`internal/app/settings_ai.go:104` | 设置 · AI 里点 Refresh：`/api/tags` 取名、`/api/show` 取能力；勾选用即入库 |
| 其他模型：地址 / 密钥 / 列出端点模型 | ✅ | `internal/app/settings_ai.go:240` `aiOtherModel`、`internal/ai/discover.go:170` `OpenAIModels` | 密钥可明文或环境变量；OpenAI 兼容端点可 `GET /models` 选取模型名 |
| 模型能力字典与推断 + 手动 pin | ✅ | `internal/ai/capability.go:35` / `:79`、`internal/app/settings_ai.go:361` | |
| **编辑界面选模型**（AI 栏下拉） | ✅ | `internal/view/ai_bar.go:283` `aiModelPick`、`internal/app/ask.go:157` `modelFor` | 列出设置里已启用模型；默认跟随设置默认项，手动选后不再跟随；模型被删/停用则回落 |
| 逐模型参数档位 | ✅ | `internal/app/settings_ai.go:284` `aiModel` | num_ctx / num_predict / temperature / think / keep_alive |
| 模型克隆与排序 | ✅ | `internal/app/settings_ai.go:196` / `:224` | |
| Agent 工具（7 个，只读、钉在库内） | ✅ | `internal/ai/tool.go:37`、`internal/app/host.go:57` `within` | |
| Agent 循环 + 结果校验 | ✅ | `internal/ai/agent.go:20` `Loop`、`internal/ai/verify.go:75` | |
| Skills（`SKILL.md` 两级发现、启停、命令面板） | ✅ | `internal/skill/skill.go:87` `Discover` | |
| 库级批量任务（加/改/删标签） | ✅ | `internal/task/task.go:172`、`internal/app/tasks_window.go:27` | 只读计划 + 分批执行可续跑 |
| 截断可见 | ✅ | `internal/view/ai_bar.go:224` | 区分「预算耗尽」与「连接中断」 |
| **生成预算用量显示** | ❌ | `internal/ai/event.go:91` `Used` 已发出 | UI 未显示 `EventUsage` |

## 7. 界面与外观

| 功能点 | 状态 | 实现位置 | 说明 |
|---|---|---|---|
| 三栏布局 + 自研分隔条 | ✅ | `internal/view/workspace.go:273` `divider` | |
| 布局持久化（版本不符重建） | ✅ | `internal/state/state.go:217` `SaveLayout` | |
| 主题（跟随系统 / 浅 / 深） | ✅ | `internal/app/app.go` `WhenReady` → `mygo.Theme.SetSource` | |
| 目录树 / 编辑区 / 预览区字体 | ⚠️ | `internal/app/settings.go:128` `editorSettings` 等 | 只可设**字号与行高**；字族、字重无 UI（配置有字段） |
| **目录树图标尺寸（12–48px）** | ❌ | `config.go:68` `IconSize` 有字段 | 无读取点，视图用固定 14px |
| 面板显隐（树 / 预览） | ✅ | `internal/app/settings.go:301` `window` | xink 没有的额外项 |

## 8. 设置

单例窗口 ✅（`internal/app/settings.go:73`），改动即时生效并落盘 ✅。

| 分组 | 状态 | 实现位置 |
|---|---|---|
| Appearance（主题、显示全部文件、目录树字号） | ✅ | `settings.go:118` |
| Editor（自动保存、软换行、行号、缩进、字号行高、代码主题） | ✅ | `settings.go:128` |
| Preview（同步开关与模式、图片、链接、最大宽度、密度） | ⚠️ | `settings.go:147`；「图片宽度」不生效 |
| Vault（当前库、库管理、打开另一个） | ✅ | `settings.go:162` |
| AI（开关、Skills、服务端、Asking、模型与能力） | ✅ | `internal/app/settings_ai.go:36` 起 |
| **快捷键页**（逐动作改键） | ❌ | — |
| 设置文件（导出 / 导入 / 恢复默认） | ✅ | `settings.go:179` |
| **设置项搜索、单项重置、⚠ 暂未生效标注** | ❌ | — |

## 9. 持久化与容错

应用数据目录由 MyGo `PathUserData` 决定（macOS：`~/Library/Application Support/Xinkgo/`）。

| 文件 | 状态 | 实现位置 |
|---|---|---|
| `config.json` | ✅ | `internal/config/config.go` |
| `layout.json` | ✅ | `internal/state/state.go:217` |
| `open_notes.json` | ✅ | `internal/state/state.go:70` |
| `vaults.json`（含旧 `vault.json` 迁移） | ✅ | `internal/state/state.go:153` / `:124` |
| `ai_sessions.json` | ✅ | `internal/state/ai_sessions.go:66` |
| `ai-history/`（上限 200） | ✅ | `internal/snapshot/snapshot.go:53` |
| 单实例 | ✅（改法不同） | `internal/app/app.go` `RequestSingleInstanceLock`，非 pid 文件 |

容错：笔记正文始终在库目录；配置/会话文件损坏回落默认值，不阻塞启动。

## 10. 快捷键（默认）

默认表 ✅（`internal/keymap/keymap.go:34` `Bindings`，17 个动作）；覆盖值非法回落默认、冲突保留靠前者 ✅；
菜单加速键跟随覆盖 ✅（`internal/app/menu.go:15`，改动后 `refreshMenu` 重建）。
**❌ 无改键界面**：`Editor.Keymap` 只能手改 `config.json`。

## 11. 工程与质量

- 分层 ✅：`cmd/xinkgo → internal/app → internal/{view,config,logging,sysinfo}`，`view`/`config` 不反向依赖 `app`。
- `CGO_ENABLED=0` ✅，全平台编译（darwin 双架构 / windows 双架构 / linux-amd64）。
- 测试：42 个测试文件、458 个 `Test*` 函数；`go test ./...` 除 `internal/ai` 外全绿
  （`internal/ai` 的失败是沙箱禁止监听端口导致 `httptest` panic，非代码缺陷）。
- 构建与打包脚本 ✅（`Makefile` + `scripts/*.sh`，macOS → `.app` + `.dmg`）。

## 12. 未实现清单（按建议优先级）

**A. 有配置/有入口但没接上（改一处即见效）**

1. 图片宽度百分比不生效（`Preview.ImageWidthPercent` 无读取点）。
2. 目录树图标尺寸不生效（`Basic.IconSize` 无读取点）。
3. 滚动同步调试数字不生效（`Preview.DebugScrollSync` 无读取点）。
4. 设置 ▸ 快捷键页缺失 —— 键位覆盖只能手改 `config.json`。
5. 数据文件旋钮（缩进 / 上限）无 UI。

**B. 功能缺口（需新写代码）**

6. 树内拖拽移动（当前只有外部拖入导入）。
7. 右键菜单补齐：复制相对路径、复制为 Markdown 链接、插入到当前笔记、粘贴到此目录。
8. 库选择器（树底部）或等价入口。
9. 「放弃改动并重新加载」入口（外部改动冲突时的第三个选项）。
10. 图片点击放大 / 预览向左伸展。
11. 编辑区 Markdown 装饰（行首标记淡化、链接文字着色）。
12. AI：密钥只支持明文 → 补环境变量 / Keychain 两条来源。
13. AI：场景路由 `Ai.Route` 接线（现在只按默认模型）。
14. AI：快捷动作增删改界面 + 提示词变量（`{{known_tags}}`）。
15. AI：从 Ollama 拉模型列表；生成预算用量显示。
16. 设置：搜索、单项重置、「暂未生效」标注。
17. 目录树滚动位置的显式保存/恢复。
18. YAML / TOML 校验与格式化（卡在 `gopkg.in/yaml.v3` 取不到，需联网或换方案）。

**C. 与 xink 一致、有意不做**

跨库检索与跨库双链、拼写检查、界面语言切换、编辑区独立配色、视频播放、PDF 预览、
目录树懒加载、最近文件列表、关系图谱、插件系统、WYSIWYG、云同步。

## 13. 与 xink 的有意差异（不是缺口）

| 项 | xink | xinkgo |
|---|---|---|
| 单实例 | pid 文件 + 存活检查 | MyGo 官方 `RequestSingleInstanceLock` |
| 文件监听 | `notify` crate | MyGo `WatchFiles`（`internal/app/watch.go:14`） |
| 布局 | `DockArea` | 自研三栏 + 分隔条，版本号不符重建 |
| 本地图片 | 转 data URL | `FileImages` 按相对路径加载 |
| 数据目录 | `~/Library/Application Support/xink/` | `.../Xinkgo/`（文件名与 JSON 结构仍兼容） |
| AI 结果落法 | 插入 / 替换 / 不采用 | 一律先过评审 diff（2026-10-31 用户拍板） |
| 外部链接「应用内打开」 | 系统浏览器或内嵌 | 自建 Web 窗口（`internal/app/web.go`） |
