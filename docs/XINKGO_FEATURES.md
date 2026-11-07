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
| 打开文件夹作为库 | ✅ | `internal/app/app_vault.go:63` `openVaultDialog` | 库根就是普通目录 |
| 多库注册表（新增/切换/改名/重新定位/删除） | ✅ | `internal/vault/vault.go` `Registry`、`internal/app/vaults_window.go:14` | 五项齐全；删除只移除记录 |
| 库状态徽标（6 态） | ✅ | `internal/vault/vault.go` `StatusOf` | 菜单与库窗口都显示不可用原因 |
| 目录树扫描（全量、非懒加载） | ✅ | `internal/vault/vault.go` `Scan` | 目录在前，同层按名排序 |
| 忽略规则 | ✅ | `internal/vault/vault.go` `ignored` | 另恒跳 `*.md.tmp` 与 `.xink`（`internal/media/media.go`） |
| 文件类型图标（10 类） | ✅ | `internal/media/media.go` `KindOf`、`internal/view/tree_icons.go` | |
| 新建笔记 / 目录 | ✅ | `internal/app/files.go` / `:49` | 自动补 `.md`，重名递增（`internal/fs/fs.go`） |
| 重命名 | ✅ | `internal/app/files.go`、`internal/fs/fs.go` | |
| 删除（含非空目录） | ✅ | `internal/app/files.go` / `:157` | 拒绝库根与库外路径（`fs.RemoveAll:168`） |
| 移动到…（对话框） | ✅ | `internal/app/files.go:173` `moveItem` | |
| **树内拖拽移动** | ❌ | — | 只有「从外部拖入」`internal/view/tree_rows.go:181` `takeDrop` |
| 外部导入（访达拖入） | ✅ | `internal/app/files.go:213` `importFiles` | 逐个返回成败，单文件失败不影响其余 |
| 粘贴（访达文件 / 截图） | ✅ | `internal/app/files.go:250` `pasteIntoVault`、`pastePicture:271` | 菜单 `Paste into Vault`（`Cmd+Shift+V`） |
| 右键菜单 | ⚠️ | `internal/view/tree_rows.go:117` `menu` | 5 项：新建笔记/新建文件夹/重命名/移动到…/删除。缺「复制相对路径」「复制为 Markdown 链接」「插入到当前笔记」「粘贴到此目录」 |
| 目录树自动刷新 | ✅ | `internal/app/watch.go:70` `handleEvents` | 仅结构变化才重扫 + 重建索引 |
| 保状态合并刷新 | ⚠️ | `internal/view/file_tree.go:124` `SetRoot` | 展开态与选中保留；滚动位置未显式保存/恢复（依赖 `ui.ListState`） |
| 切换笔记后定位 | ✅ | `internal/view/tree_paths.go:67` `Reveal`、`internal/view/tree_paths.go:67` `followActiveNote` | 每帧认「当前笔记」：点标签页 / 跟双链 / 命令面板 / 重启恢复都展开祖先、选中并滚入视口 |
| 库选择器（树底部） | ❌ | — | 切换库只走菜单 `Vault` 与设置 ▸ Vault |

## 2. 笔记编辑

| 功能点 | 状态 | 实现位置 | 说明 |
|---|---|---|---|
| 源码模式编辑 | ✅ | `internal/view/tabs.go:139` `newEditor` | MyGo `editor.Buffer` |
| 自动保存（防抖，默认 300ms） | ✅ | `internal/view/tabs.go:294` `scheduleSave`、`SaveDue:286` | 写盘只在画帧/主线程做，不用定时器 goroutine |
| 立即保存 `Cmd+S` | ✅ | `internal/app/menu.go` `Save` | 评审未收尾时拒绝保存并提示 |
| 原子写盘 | ✅ | `internal/view/tabs.go:333` `Save` | 同目录 tmp + rename |
| 脏标记 | ✅ | `internal/view/tabs.go:512` `Title`、`internal/view/tabs.go:512` 状态栏 | |
| 外部修改检测 | ✅ | `internal/view/tabs.go:432` `TakeExternalText`、`internal/view/tabs.go:432` `Reload` | 一致则不回填 |
| **放弃改动重新加载** | ❌ | — | 只提示并保留用户内容，无「丢弃并重载」入口 |
| 标签唯一 | ✅ | `internal/view/tabs.go:88` `Open`、`internal/view/tabs.go:88` `TabKey` | |
| 标签持久化与恢复 | ✅ | `internal/app/app.go:311`、`internal/app/app.go:311` `restore` | 只恢复属于当前库的标签，恢复后写回 |
| 空白占位页 | ✅ | `internal/view/panes.go:111` `editor` | |
| 缩进 / 软换行 / 行号 | ✅ | `internal/view/apply.go:29` `applyEditor` | |
| 编辑区左右留白（可配置） | ✅ | `internal/view/geom/geom.go` `EditorPadLeft`、`EditorPadRight` | 默认 26，范围 4–64；正文到左右边缘等距 |
| 编辑区右边距（可配置） | ✅ | `internal/config/config.go` `MarginRight`、`internal/view/geom/geom.go` `EditorMarginRight` | 默认 0，范围 0–120；在左右留白之外只给右边再加一段；换行按整字算，故实际比设定值少半个字符上下 |
| 编辑区滚动条 | ✅ | `internal/view/panes.go` `editorTheme` | 常态不画：笔记用一份自己的主题，把滚动条设为透明；指针移上去才显形，不再遮住正文 |
| 格式工具栏 | ✅ | `internal/view/bars.go:203` `toolbar` | 14 个按钮（xink 13 个 + 行内代码） |
| 格式快捷键 | ✅ | `internal/app/menu.go` `Format`、`internal/view/panes.go:217` | |
| 编辑区语法高亮 | ✅ | `internal/view/tabs.go` | md / json / yaml / toml 按文件名 |
| **编辑区 Markdown 装饰**（行首标记淡化、链接文字着色） | ❌ | — | xink 有，xinkgo 未做 |
| 数据文件校验标记 | ✅ | `internal/view/panes.go:187` `dataDiagnostics` | 波浪线；JSON 有行列 |

## 3. 预览

| 功能点 | 状态 | 实现位置 | 说明 |
|---|---|---|---|
| Markdown 渲染 | ✅ | `internal/view/preview.go:22` `previewPanel` | 用 MyGo `ui/markdown`，能力由框架决定 |
| 本地图片内联 | ✅ | `internal/view/preview.go:231` `markdownOptions` | `FileImages` 按目录解析，非 data URL |
| 图片缓存 | ➖ | 框架内 | 无自研指纹缓存，无 5MB 上限 |
| 图片显示方式（适应宽度 / 隐藏） | ✅ | 同上（`ImageDisplay`） | |
| **图片宽度百分比 20–100** | ❌ | `internal/config/config.go` 有字段 | 设置里有滑块，但 preview 无读取点 → 不生效 |
| **图片点击放大** | ❌ | — | 未做 |
| 预览向左伸展（盖住编辑区） | ✅ | `internal/view/preview.go` `spreadButton` | 预览面板左上角按钮；不落盘，编辑区宽度保留，再点还原；伸展后正文与侧栏按钮之间另留一段 |
| 代码块语法高亮（编辑区与预览同源） | ✅ | `internal/highlight/highlight.go` `EditorTheme`、`internal/view/apply.go` | |
| 高亮主题 3 套 | ✅ | `internal/highlight/highlight.go` `Presets` | |
| 滚动同步（双向，三档） | ✅ | `internal/view/scroll.go:56` `followScroll`、`internal/view/scroll.go:56` `BlockSpans` | 方向锁 + 回显吸收 |
| 预览区留白（左右 / 上下，可配置） | ✅ | `internal/view/geom/geom.go` `PreviewPadX`、`PreviewPadY` | 默认 26 / 12，范围 4–64 |
| **滚动同步调试数字** | ❌ | `config.go:104` `DebugScrollSync` 有字段 | 无读取点 |
| 版面密度 / 最大宽度 | ✅ | `internal/view/apply.go:83` `spacing`、`previewWidth:91` | |
| 链接点击行为 | ✅ | `internal/view/preview.go:274` `openLink`、`internal/view/preview.go:274` | 库内打开笔记；外部链接按设置走系统浏览器或应用内窗口 |
| **畸形 Markdown 防御性预处理** | ➖ | — | 由框架解析器承担，未自研 |
| 公众号样式目录可配置（多文件 + 元数据） | ✅ | `internal/wechat/theme.go` `OfDir`、`internal/app/settings.go` `wechatDirField` | 设置里指定任意目录，一个 `*.json` 一套样式；文件带 id/name/version/author/description/extends；坏文件、重复 id、extends 落空都会在设置里说明 |
| 公众号排版（样式可选 + 一键复制） | ✅ | `internal/wechat/`、`internal/view/wechat.go`、`internal/app/wechat.go` | 预览右上角；默认正常 Markdown，另有内建 5 套 + `<用户数据>/wechat/*.json` 自加样式；行内样式，复制后可直接粘贴到公众号后台 |
| 公众号模式下预览按手机宽度显示 | ✅ | `internal/view/wechat.go` `wechatScreen` / `wechatArticle`、`internal/wechat/html.go` `pagePad` | 414 DIP 居中（圆角 + 边框），内容左右各留 20；窗格窄时先让留白再让宽度；正常预览照旧自动适配 |

## 4. 数据文件（JSON / YAML / TOML）

| 功能点 | 状态 | 实现位置 | 说明 |
|---|---|---|---|
| 类型判定 | ✅ | `internal/datafile/datafile.go` `KindOf` | |
| 编辑区高亮 | ✅ | `internal/view/tabs.go` | |
| 预览格式化 | ⚠️ | `internal/view/preview.go:249` `previewText` | JSON 完整；YAML/TOML 只判定不格式化（缺 yaml 库，已决策） |
| 校验（行/列 + 原因） | ⚠️ | `internal/datafile/datafile.go` `Check` | 仅 JSON 报行列；YAML/TOML 不报 |
| 大文件保护 | ✅ | `internal/datafile/datafile.go` `TooLarge` | 默认 1MB |
| **设置里的数据文件旋钮**（缩进 / 上限） | ❌ | `config.go` `Editor.DataFile` 有字段 | 无 UI |

## 5. 导航、索引与结构

| 功能点 | 状态 | 实现位置 | 说明 |
|---|---|---|---|
| 全库索引（标题/标签/双链/反链） | ✅ | `internal/index/index.go` `Build` | 后台构建，不落盘；`Reindex:171` 增量 |
| 双链解析（路径 → 文件名 → 标题，歧义可辨） | ✅ | `internal/index/index.go` `resolveIn` | 含 `Ambiguous` |
| 预览里双链可点击 | ✅ | `internal/markdown/links.go` `Links` | 支持 `别名`、`#锚点` |
| `[[` 补全 | ✅ | `internal/view/completion.go` | 写库内相对路径 |
| 反向链接面板 | ✅ | `internal/view/panels.go` | |
| 标签面板（聚合 + 展开） | ✅ | `internal/view/panels.go` | 标签来源含 frontmatter（`internal/markdown/facts.go`） |
| 大纲面板（实时解析，点击跳行） | ✅ | `internal/view/outline.go` | |
| 右侧按钮条 + 抽屉 | ✅ | `internal/view/outline.go:63` `rail`、`:73` `drawerPanel` | |
| 命令面板 `Cmd+P` | ✅ | `internal/view/palette.go` | 前 50 条、命中高亮；另列 Skill 与 AI 动作 |
| 全文搜索 `Cmd+Shift+F` | ✅ | `internal/view/search_panel.go` | 防抖、后台、点击跳行 |
| 模糊匹配与打分 | ✅ | `internal/fuzzy/fuzzy.go`、`internal/search/search.go` | |
| 明暗主题切换 | ✅ | `internal/app/app.go` `toggleTheme` | |

## 6. AI 协助（默认关闭）

| 功能点 | 状态 | 实现位置 | 说明 |
|---|---|---|---|
| 对话条 `Cmd+Shift+A` | ✅ | `internal/view/ai_bar.go:134` `aiBar` | 按笔记归属 |
| 会话持久化（30 会话 / 200 块） | ✅ | `internal/state/ai_sessions.go`、`internal/view/ai_bar.go` | |
| 快捷动作（内置 6 项 + 命令面板入口） | ✅ | `internal/view/ai_ask.go` `runAction`、`internal/app/settings_quick.go` `aiQuickWords` | 整篇 + 选区；设置 ▸ AI ▸ 动作与提示词里可增删改、恢复内置项 |
| **提示词变量** | ✅ | `internal/view/ai_vars.go` `expandActionPrompt`、`internal/templating/expand.go` | `{{title}}` / `{{note}}` / `{{selection}}` / `{{known_tags}}` / `{{date}}` / `{{time}}` / `{{date:格式}}`；未知变量原样保留 |
| 结果落法 | ⚠️ | `internal/view/review.go` / `:60` | 一律走评审 diff；`Apply=insert/replace` 自动落盘未实现（已决策） |
| 修订评审（逐块接受/拒绝） | ✅ | `internal/view/review.go:101` `reviewPane` | 整篇与选区两种 |
| 改前快照 + 取回窗口 | ✅ | `internal/snapshot/snapshot.go`、`internal/app/history_window.go` | 上限 200，取回前先存当前 |
| Provider：Ollama / OpenAI 兼容 | ✅ | `internal/ai/ollama.go`、`internal/ai/openai.go` | |
| 流式解析（分帧 + 语义） | ✅ | `internal/ai/framing.go`、`internal/ai/parse.go` | NDJSON / SSE |
| 重试与备用模型 / 两类超时 | ✅ | `internal/ai/ai.go` `Ask` | |
| **场景路由**（对话/动作/Skill/Agent 各指定模型） | ✅ | `internal/config/ai.go` `ResolveModel`、`internal/view/ai_vars.go` `askModel` | 动作指定 > 场景路由 > 栏内手选 > 默认 > 首个启用；带工具循环的对话走 Agent 路由，带 Skill 的走 Skill 路由 |
| 上下文裁剪 | ✅ | `internal/view/ai_vars.go` `contextMaxChars` | 全局 `MaxContextChars` 与模型 `ContextLimit` 取较小值（token 按 2 字符保守折算） |
| 密钥管理 | ✅ | `internal/ai/ai.go` `keyOf`、`internal/secret/` | 明文 / 环境变量 / Keychain 三来源，`KeyRef` 稳定命名 |
| 生成用量显示 | ✅ | `internal/view/ai_talk.go` `aiSaid`、`internal/ai/event.go` `Usage` | 回答尾部显示输入 / 输出 token 与耗时，模型未上报则显示 `-` |
| AI 设置分组 | ✅ | `internal/app/settings_ai.go` `ai` | 模型 / 提问 / 动作与提示词 / 工具与技能四组，稳定 ID 切语言不丢 |
| 连接测试（探针） | ✅ | `internal/ai/probe.go` `Try`、`internal/app/settings_ai.go` | |
| 从 Ollama 拉模型列表 | ✅ | `internal/ai/discover.go` `OllamaModels`、`internal/app/settings_ai.go` | 设置 · AI 里点 Refresh：`/api/tags` 取名、`/api/show` 取能力；勾选用即入库 |
| 其他模型：地址 / 密钥 / 列出端点模型 | ✅ | `internal/app/settings_ai.go:267` `aiOtherModel`、`internal/app/settings_ai.go:267` `OpenAIModels` | 密钥可明文或环境变量；OpenAI 兼容端点可 `GET /models` 选取模型名 |
| 模型能力字典与推断 + 手动 pin | ✅ | `internal/ai/capability.go` / `:79`、`internal/app/settings_ai.go` | |
| **编辑界面选模型**（AI 栏下拉） | ✅ | `internal/view/ai_bar.go:235` `aiModelPick`、`internal/view/ai_bar.go:235` `modelFor` | 列出设置里已启用模型；默认跟随设置默认项，手动选后不再跟随；模型被删/停用则回落 |
| 逐模型参数档位 | ✅ | `internal/app/settings_ai.go` `aiModel` | num_ctx / num_predict / temperature / think / keep_alive |
| 模型克隆与排序 | ✅ | `internal/app/settings_ai.go` / `:224` | |
| Agent 工具（7 个，只读、钉在库内） | ✅ | `internal/ai/tool.go`、`internal/app/host.go` `within` | |
| Agent 循环 + 结果校验 | ✅ | `internal/ai/agent.go` `Loop`、`internal/ai/verify.go` | |
| Skills（`SKILL.md` 两级发现、启停、命令面板） | ✅ | `internal/skill/skill.go` `Discover` | |
| 库级批量任务（加/改/删标签） | ✅ | `internal/task/task.go`、`internal/app/tasks_window.go` | 只读计划 + 分批执行可续跑 |
| 截断可见 | ✅ | `internal/view/ai_bar.go` | 区分「预算耗尽」与「连接中断」 |
| 服务商分层（预设 / 拉取模型列表 / 批量加入） | ✅ | `internal/config/ai_providers.go`、`internal/app/settings_ai_provider.go` | `Models` 仍是扁平可调用列表，`ProviderID` 归组；老 `config.json` 按地址自动迁移；Ollama 保持独立入口 |
| 模型选择器 popover | ✅ | `internal/view/ai_p1.go` `aiModelPick` | 搜索、最近使用、按服务商 / Ollama 分组，替代原生下拉 |
| 最近使用模型 | ✅ | `internal/state/recent_models.go` | 运行期状态，最多 8 个，不写 `config.json` |
| 上下文 chip（可移除） | ✅ | `internal/view/ai_p1.go` `syncContexts` / `contextStrip` | 当前笔记与选区各自成一个 chip；移除后不再发给模型，直到笔记或选区变化 |
| 自由指令改写选区 | ✅ | `internal/ai/inline.go` `Replacement`、`internal/view/ai_ask.go` `aiReviewDone` | 回答含 `<replacement>` 时只对选中片段做 diff 评审，笔记变过则拒绝写回 |
| 回复落地 | ✅ | `internal/app/ai_reply.go` | 复制 / 追加到今日日记 / 追加到指定笔记，只追加不覆盖 |
| Prompt 库（固定 / 分类 / 场景 / blanks） | ✅ | `internal/config/ai.go` `AiAction`、`internal/view/ai_p1.go` `blankForm` | `{{blank}}` 运行前弹表单；固定的排快捷条最前；文件化未做（可选项） |
| 首屏场景化 | ✅ | `internal/view/ai_p1.go` `starterStrip` | 按有选区 / 有笔记 / 无笔记给标题与提示，建议复用 quickStrip |
| 问题卡（模型向用户提问） | ✅ | `internal/ai/{event.go,question.go,agent.go}`、`internal/view/ai_p2.go` `questionCard` | `ask_user` 工具 + `EventQuestion`；一次一题，可选项 / 自填 / 跳过；私密回答不写进会话 |
| 工作时间线与折叠 | ✅ | `internal/view/ai_p2.go` `timelineLine` | 「已探索 N 个文件，M 次搜索」折叠，可展开逐步与用时 |
| 上下文用量条 | ✅ | `internal/view/ai_p2.go` `usageBar` | 模型声明 `ContextLimit` 时显示已用百分比；未声明则不画 |
| 编辑重发 / 会话列表 | ✅ | `internal/view/ai_p2.go` `editUserBlock` / `sessionList` | 编辑用户消息截断后续分支；头部 popover 列出有对话的笔记并切换 |
| `@笔记` 引用 | ✅ | `internal/view/ai_p2.go` `atCompletions` | `@` 弹库内笔记补全，附加为可移除上下文；未做 `@文件夹` |
| 按轮撤销 | ✅ | `internal/view/review.go`、`internal/app/ai_reply.go` `aiUndo` | 撤销最近一次评审写回，撤销前检查笔记未被改动 |
| 文件权限两档 | ✅ | `internal/config/ai.go` `Permission` | 只读（默认）/ 允许追加；工具集始终只读，评审写回始终过 diff |
| **生成预算用量显示** | ❌ | `internal/ai/event.go` `Used` 已发出 | UI 未显示 `EventUsage` |

## 7. 界面与外观

| 功能点 | 状态 | 实现位置 | 说明 |
|---|---|---|---|
| 三栏布局 + 自研分隔条 | ✅ | `internal/view/workspace.go:399` `divider` | |
| 布局持久化（版本不符重建） | ✅ | `internal/state/state.go` `SaveLayout` | |
| 主题（跟随系统 / 浅 / 深） | ✅ | `internal/app/app.go` `WhenReady` → `mygo.Theme.SetSource` | |
| 目录树 / 编辑区 / 预览区字体 | ⚠️ | `internal/app/settings_editor.go:12` `editorSettings` 等 | 只可设**字号与行高**；字族、字重无 UI（配置有字段） |
| **目录树图标尺寸（12–48px）** | ❌ | `config.go:68` `IconSize` 有字段 | 无读取点，视图用固定 14px |
| 面板显隐（树 / 预览） | ✅ | `internal/app/settings.go:363` `window` | xink 没有的额外项 |
| 编辑区向右伸展（盖住预览） | ✅ | `internal/view/bars.go` `widenButton` | 标签栏末端按钮；不落盘，预览宽度保留，再点还原；伸展动画 180ms（系统「减弱动态效果」时为 0）|
| 预览向左伸展（盖住编辑区） | ✅ | `internal/view/preview.go` `spreadButton` | 预览面板左上角按钮；不落盘，编辑区宽度保留，再点还原；伸展后正文与侧栏按钮之间另留一段 |

## 8. 设置

单例窗口 ✅（`internal/app/settings.go`），改动即时生效并落盘 ✅。

| 分组 | 状态 | 实现位置 |
|---|---|---|
| Appearance（主题、显示全部文件、目录树字号） | ✅ | `settings.go:118` |
| Editor（自动保存、软换行、行号、缩进、字号行高、代码主题） | ✅ | `settings.go:128` |
| Preview（同步开关与模式、图片、链接、最大宽度、密度） | ⚠️ | `settings.go:147`；「图片宽度」不生效 |
| Vault（当前库、库管理、打开另一个） | ✅ | `settings.go:162` |
| AI（开关、Skills、服务端、Asking、模型与能力） | ✅ | `internal/app/settings_ai.go` 起 |
| **快捷键页**（逐动作改键） | ❌ | — |
| 设置文件（导出 / 导入 / 恢复默认） | ✅ | `settings.go:179` |
| **设置项搜索、单项重置、⚠ 暂未生效标注** | ❌ | — |

## 9. 持久化与容错

应用数据目录由 MyGo `PathUserData` 决定（macOS：`~/Library/Application Support/Xinkgo/`）。

| 文件 | 状态 | 实现位置 |
|---|---|---|
| `config.json` | ✅ | `internal/config/config.go` |
| `layout.json` | ✅ | `internal/state/state.go` |
| `open_notes.json` | ✅ | `internal/state/state.go` |
| `vaults.json`（含旧 `vault.json` 迁移） | ✅ | `internal/state/state.go` / `:124` |
| `ai_sessions.json` | ✅ | `internal/state/ai_sessions.go` |
| `ai-history/`（上限 200） | ✅ | `internal/snapshot/snapshot.go` |
| 单实例 | ✅（改法不同） | `internal/app/app.go` `RequestSingleInstanceLock`，非 pid 文件 |

容错：笔记正文始终在库目录；配置/会话文件损坏回落默认值，不阻塞启动。

## 10. 快捷键（默认）

默认表 ✅（`internal/keymap/keymap.go` `Bindings`，17 个动作）；覆盖值非法回落默认、冲突保留靠前者 ✅；
菜单加速键跟随覆盖 ✅（`internal/app/app_apply.go:117`，改动后 `refreshMenu` 重建）。
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
10. 图片点击放大。
11. 编辑区 Markdown 装饰（行首标记淡化、链接文字着色）。
12. ~~AI：补环境变量 / Keychain 两条密钥来源~~ 已补。
13. ~~AI：场景路由 `Ai.Route` 接线~~ 已补（`ResolveModel` / `askModel`，见 L1）。
14. ~~AI：快捷动作增删改界面 + 提示词变量~~ 已补（见 L4）。
15. ~~AI：从 Ollama 拉模型列表；生成预算用量显示~~ 已补（见 L2）。
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
| 文件监听 | `notify` crate | MyGo `WatchFiles`（`internal/app/watch.go`） |
| 布局 | `DockArea` | 自研三栏 + 分隔条，版本号不符重建 |
| 本地图片 | 转 data URL | `FileImages` 按相对路径加载 |
| 数据目录 | `~/Library/Application Support/xink/` | `.../Xinkgo/`（文件名与 JSON 结构仍兼容） |
| AI 结果落法 | 插入 / 替换 / 不采用 | 一律先过评审 diff（2026-10-31 用户拍板） |
| 外部链接「应用内打开」 | 系统浏览器或内嵌 | 自建 Web 窗口（`internal/app/web.go`） |
