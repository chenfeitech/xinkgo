# AI 能力升级规划：对照 Qiaomu Agent（v1 · 已拍板）

日期：2026-11-06 调研，2026-11-07 成稿并拍板。状态：**规划已定，按 §10 执行；
P0 起开始落地**。分支 `feature/ai-upgrade-plan`。

参照对象：<https://github.com/joeseesun/qiaomu-agent>，Obsidian 第三方插件
（要求 Obsidian ≥ 1.11.4，v0.6.x）。调研方式是浅克隆后读源码与文档，而不是只看
README：`DESIGN.md`、`types.ts`、`settings-tab.ts`、`ui/provider-settings.ts`、
`ui/model-picker.tsx`、`main.ts`、`chat-view.ts`、
`integrations/reading-context.ts`、`services/{api-providers,provider-models,
change-tracker,user-questions,context-usage,prompt-library,starter-prompts,
inline-edit,cli-backend,native-agent-backend}.ts` 以及 `docs/research/` 下的模型管理、
选区工作流、Skills/MCP 等调研稿。

> 与 [ai-assist.md](ai-assist.md) 的关系：那份是 2026-11-02 对 AI 编辑辅助入口的
> 拍板并已落地。本文是**下一轮**升级盘点，不推翻其中「菜单不做 / 按钮放工具条 /
> AI 结果一律先过评审 diff」等结论。
> 与 [XINKGO_FEATURES.md](../XINKGO_FEATURES.md) 的关系：该文 §6 与
> [ACCEPTANCE.md](../ACCEPTANCE.md)「已知缺口」已登记若干 AI 待办，本文把它们并进
> 同一条路线，不另起炉灶。

## 1 结论先行

1. 它设计得好，好在**信息架构**而不是某个炫技功能：任务导向的渐进披露、
   「一个选择器管所有来源」、上下文和写入安全被当成一等公民、把 Agent 的工作
   过程如实呈现。这些是可以借鉴的**结构**，不是可以照搬的 UI（我们是 Go 原生
   UI，没有 DOM / React / Obsidian 渲染器）。
2. **配置管理的真实差距是「服务商」这一层**，不是「能不能加模型」。我们是
   `ModelEntry` 扁平列表：一个条目 = 一个可调用模型，自带地址/密钥/参数；同一个
   `baseURL` + key 要在每个模型里重复填，拉到的 `/models` 列表只用来重命名当前
   条目。它是 `ProviderConfig`（服务商）→ `models[]`（该服务商下多个模型，逐模型
   启用 / 设为默认 / 参数覆盖）→ 一个全局模型选择器。这一层值得补（P1）。
3. **Ollama 是我们的优势，不要为了对齐而降级**。它有独立的 `OllamaConfig`、
   `/api/tags` 探测 + `/api/show` 取能力与上下文长度、模型能力推断、num_ctx /
   keep_alive 参数；qiaomu 把 Ollama 放在「本地与自定义」预设末尾当一个**无 key
   的普通 OpenAI 兼容端点**。补服务商分层时必须保留 Ollama 的独立入口与探测，
   而不是把它塞进通用 provider 网格。
4. **「Agent」是同名不同物**，这是本轮最需要澄清的一点（§4）。qiaomu 的 Agent
   是本机 CLI 进程（Codex / Claude Code / ACP / App Server）；我们的 `Agentic` 是
   内置只读工具循环。**2026-11-07 拍板：外部 CLI Agent 不做**；只按解读 A 把
   内置工具循环做成可命名的配置档，落在 P0/P1（§4、§10）。
5. **选区我们不是没有，而且比它更严谨**：`pieceOf` 取选区、`askingOf` 明确要求
   「只答这一段、别带别的」、`StartPieceReview` 只改选中的片段、`writeReview`
   放回原位置，且笔记中途变过会拒绝写回。差距在**可见性与自由度**：选区没有
   作为对话里可移除的上下文 chip，也不能在自由对话里说「把我选中的改正式些，
   直接改文件」让模型返回 `<replacement>` 再做词级 diff 预览（P1）。
6. 有一批**已登记但未接线**的低成本项（场景路由、用量显示、快照按库归属、
   动作提示词变量），应先清理再谈大改（P0）。
7. 不做：外部 CLI Agent、**联网搜索与公共网页读取**、React/TS、账号订阅与
   OAuth、图片生成、会话配色、完全访问权限档。理由见 §9。

## 2 它为什么设计得好

按 `DESIGN.md` 的 `Functional contract` 与 `Do / do not` 归纳，可迁移的是这些原则：

| 原则 | 具体做法 | 对我们的意义 |
|---|---|---|
| 任务导向的渐进披露 | 设置页 5 个 tab（模型 / 对话 / Prompt / 工具 / 关于）；默认模型页先回答「这次对话用什么」，协议、Base URL、MCP、系统 Prompt 藏在下级展开 | 我们 `settings_ai.go` 是一页长滚动，八组控件全平铺；可做**页内分组** |
| 一个选择器管所有来源 | 模型 chip 打开非模态 popover：顶部搜索、左侧来源 rail（全部 / 最近 / 各 Agent / 各服务商）、列表按来源分组；选模型即选来源，没有「先连哪个」这一步 | 我们现在是 `ui.Select` 原生下拉，只有已启用模型的平铺 ID |
| 配置一次，多处复用 | 服务商预设网格（海外 / 国内 / 聚合 / 本地与自定义）；添加时只问一个字段（key），随后拉模型列表；逐模型开关 / 设为默认 / 手动 ID / 参数覆盖 | 正是我们缺的 `Provider` 层 |
| 上下文是一等公民 | `<active_note path>` / `<editor_selection path lines>` 统一格式；选区是可移除的 composer chip，焦点在侧栏时用 CSS Custom Highlight 保持选区可见；`@` 引用库内文件/文件夹，可附网页与图片 | 我们 `aiBrief` 已把当前笔记塞进 system prompt，但没有 chip、没有可移除、没有引用 |
| 安全写入 | 权限三档（只读默认 / 当前库可写 / 桌面完全访问）；按轮变更追踪（before-content 四级回退）；撤销前预检文件是否还是 Agent 离开时的样子；新建文件进回收站 | 我们是只读工具 + `AutoApply=false` + 按文件名快照；更保守，但缺「按轮」与「预检撤销」 |
| 如实呈现工作过程 | 时间线按发生顺序穿插文字 / 已运行 / 已编辑 / 已搜索 / 计划清单；完成后折叠成「已处理 · 用时」，只留最终回答；失败行标红不强制展开 | 我们 `ai.Event` 已有 `Called`/`Found`/`Thinking`/`Usage`，但 `aiTalk` 只显示小字工具行，没有时间线与折叠 |
| 让用户中途做决定 | 问题卡：Codex `requestUserInput` / ACP form elicitation / API 的 `ask_user` 工具，统一成带选项的卡片，字母键选择、可自填、可跳过 | 我们没有「模型向用户提问」这条回路 |
| Prompt 是资产不是字符串 | 场景化内置 Prompt、blanks（`{{名称}}` / `{argument}`）、用户 Prompt 以 Markdown 文件存在库里、可固定任意多条、每条也是命令面板命令 | 我们有 `AiAction`（触发词 + prompt + scope + apply）与 `quickStrip`，但无 blanks / 场景 / 文件化 / 分类搜索 |
| 首屏不随机 | 空白对话按屏幕上内容（选区 → 阅读上下文 → 今日日记 → 当前笔记 → 库）决定标题、提示与四条建议；无模型源时只给一个「连接模型」按钮 | 我们空白对话条没有场景化引导 |
| 克制的视觉语言 | 语义 CSS 变量、无渐变/玻璃/斜体/巨型标题、6–14px 圆角、容器查询而非视口查询、强调色只表示焦点或进行中的工作 | 与我们 `theme` 的克制取向一致，可直接沿用，不需要抄它的 CSS |
| 隐私边界写进设计 | SecretStorage 存密钥、私网地址拒绝、每次重定向重新校验并 pin 公网 IP、无遥测、无自建服务器 | 我们 `vaultHost.within` 已拒绝库外；补联网时必须带上同等边界 |

## 3 配置管理：逐项对照

| 能力 | Qiaomu Agent | Xinkgo 现状 | 判断 |
|---|---|---|---|
| 设置页结构 | 插件自有设置页 + 顶部 5 tab，模型页两张卡片（本机 Agent / 服务商），行内展开详情 | 设置窗口里一个 AI 分组页，`ai()` 顺序：开关 → Ollama → 手写模型 → 选哪个 → 提问旋钮 → 快捷词 → Skills，全平铺 | **可做**：页内分组（模型 / 对话 / 动作 / 工具），不嵌套卡片，沿用现有 `group()`/`field()`/`settingRow()` |
| 添加模型 | 「添加服务商」→ 可搜索的品牌瓦片网格（约 30 个预设 + 自定义）→ 填 key → 拉模型列表 → 逐模型启用 | 「Add a model」→ 手填一条 `ModelEntry`；`listWhatItHolds(id)` 能 `GET /models`，但结果只作为**重命名当前条目**的按钮 | **结构性差距**：补 `AiProvider` 层（P1） |
| 添加 Agent | 检测本机 CLI（Codex / Claude Code / …），逐 Agent 显隐、模型缓存、可见性覆盖 | 无外部 Agent；`Agentic` 是内置工具循环开关 | **辨析后再定**（§4）；低代价版本是把「Agent 档」做成内置工具循环的配置档 |
| 服务商 → 多模型 | `ProviderConfig.models` / `enabledModels` / `removedModels` / `manualModels` / `showInPicker` / `modelOptions` | 扁平 `ModelEntry[]`，`Enabled` 是逐模型的，`Capabilities` / `Params` / `ContextLimit` 是逐模型的 | 我们**已有逐模型粒度**，缺的是分组与「一次拉取、批量启用」 |
| Ollama | 「本地与自定义」组里一个无 key 的 OpenAI 兼容预设 | `OllamaConfig` 独立配置 + `/api/tags` + `/api/show` + 能力推断 + num_ctx/keep_alive；设置页独立一组、勾选用即入库 | **我们更强**，保留并强化 |
| 密钥 | Obsidian SecretStorage，不写数据文件；有「获取密钥」链接 | 三来源 `plain` / `env` / `keychain`，`KeyRef` 稳定命名，keyring 读写 | **我们更完整**，不动 |
| 能力 | `ModelOptions`（temperature / maxOutputTokens / contextWindow / reasoning / vision），「未设置 = 用厂商上报」 | `ai.Try` 实测 + `ai.Infer` 推断 + `CapabilitiesPinned` 手动 pin + `ContextLimit` + 参数档位 | **我们更严谨**（实测胜过只信上报），保留 |
| 模型选择器 | 非模态 popover：搜索、来源 rail、分组表头、最近使用、推理强度 pill、键盘操作 | `aiModelPick` 的 `ui.Select`，平铺已启用 ID；`aiPicked` 记住手动选择 | **可做**：popover 化（P1） |
| 最近模型 | `recentModels` 跨 Agent/服务商 | 无 | 低成本，随 popover 一起做 |
| 路由 | 每个对话可选模型与 effort；`modelSelections` | `AiRouteConfig{Chat,Action,Skill,Agent}` 与 `AiAction.Model` **有字段但未接线** | **P0 清账**（已登记待办） |
| 用量 | `ContextUsage`（used/size）+ 上下文环 | `ai.Usage{PromptTokens,CompletionTokens,DurationMs}` 已在事件里，UI 未显示 | **P0 清账**（已登记待办） |

## 4 「Agent」同名不同物

这是最容易误判的一处，直接决定工作量级。

| | Qiaomu 的 Agent | Xinkgo 的 Agentic |
|---|---|---|
| 是什么 | 本机已安装并登录的 **CLI 进程**：Codex App Server、ACP（Claude Code）、Pi、Zcode 等 | `internal/ai/agent.go` 的 **Loop**：模型问、Go 侧执行工具、结果回灌，循环到答完或步数用尽 |
| 怎么接 | spawn 子进程、写临时 MCP 配置文件（仅当前用户可读写）、读技能目录、协议事件转时间线、OAuth loopback 收回调 | 纯 Go，`internal/ai/tool.go` 的 `Host` 接口；工具只读，`vaultHost.within` 拒绝库外 |
| 配置 | `BackendKind`、`preferredCli`、`hiddenAgents`、`agentVisibility`、`agentModelCache`、`agentEnabledModels`、权限三档 | `AiConfig.Agentic`、`MaxSteps`、`MaxToolChars`、`SkillsEnabled` |
| 代价 | 依赖 Node/CLI 生态、进程生命周期与超时、跨平台差异、临时文件、审批协议、密钥回调 | 已在仓库里，零外部依赖 |

Xinkgo 的铁律是 `CGO_ENABLED=0`、不引入 Node / 新运行期依赖、原生 UI、主线程、
macOS P0 → Windows P1 → Linux P2。外部 CLI Agent 会同时踩中「外部进程 +
平台差异 + 权限 + 凭据」四条，且它的卖点（能写文件 / 跑命令）与我们
`ai-assist.md` 拍板的「一律先过评审 diff」直接冲突。

两种解读与 2026-11-07 的拍板结论：

- **解读 A（配置档，采纳）**：补服务商分层，并把内置工具循环做成可命名的
  「Agent 档」（步数 / 工具集 / 技能 / 允许哪类动作），代价小，落在 P0/P1。
- **解读 B（外部 CLI，本次排除）**：按 qiaomu 语义新增外部进程宿主，需要打破
  「无 Node / 无外部进程」、重新定权限与跨平台检测。**不做**，也不留作本轮 P3；
  若将来要做，另起一份独立设计，不并入本路线。

## 5 AI 功能：逐项对照

| 功能 | Qiaomu Agent | Xinkgo 现状 | 判断 |
|---|---|---|---|
| 选区改写 | 自由对话里说「把我选中的改正式些，直接改文件」；模型返回 `<replacement>`，解析后做 `Intl.Segmenter` 词级 LCS diff，预览后替换；`inlineTargetIsCurrent` 校验编辑器快照未变 | **已有且更严谨**：`pieceOf` 取选区 → `askingOf` 要求只答该片段 → `StartPieceReview` 只改片段 → `writeReview` 放回；笔记变过就拒绝写回。但只在**预设动作**里触发，且没有词级 diff 预览 | **P1**：把这条链路接到自由对话，补 `<replacement>` 解析与词级 diff；保留我们的边界校验 |
| 选区作为上下文 | composer 里可移除的 chip「选中 N 行 · 笔记」；焦点离开编辑器时用 CSS Custom Highlight 保持选区可见；`pin`/`dismiss`/`consumed` 生命周期 | 每帧由 `pieceOf` 隐式读取；`aiBrief` 把整篇笔记（按 `MaxContextChars` 截断）塞进 system prompt，**不可见、不可移除、不区分选区** | **P1**：chip + 可移除 + 选区高亮；native UI 用 editor 的 selection 绘制，不用 CSS Highlight |
| 阅读上下文 | `ReadingContextService` 跟踪最近阅读的 leaf，适配 PDF / Web viewer / 外部插件视图，来源插件可 `pin` 交接 | 只有「当前编辑的笔记」；我们的 Web viewer 在预览面板有网页标签，但 AI 完全不知道 | **P2**：把当前显示的笔记 / 网页 / PDF 作为可切换上下文源 |
| `@` 引用与附件 | `@` 引用库内文件/文件夹，附网页、图片、RSS；回复中的库内链接可点开 | 无；模型只能通过只读工具自己找（`search_notes` / `read_note`） | **P2**：先做 `@文件`/`@文件夹` 复用 `internal/index`，图片/网页后置 |
| 问题卡 | `ask_user` 工具 / Codex requestUserInput / ACP elicitation → 选项卡片，字母键、自填、跳过、可脱敏 | 无「模型向用户提问」回路 | **P2**：新增工具 + `ai.Event` 类型 + `ui.Modal` 卡片 |
| 工作过程时间线 | 文字 / 已运行 / 已编辑 / 已搜索 / 计划清单按序穿插，完成时折叠「已处理 · 用时」 | `aiTalk` 有 thinking 弱化小字与工具调用小字（`Called`/`Found`），无计划、无折叠、无耗时汇总 | **P2**：复用现有事件，只改呈现 |
| 按轮变更与撤销 | `TurnChangeTracker` 每轮追踪，四级回退取 before-content；`planRollback` 把文件分成 safe / conflict / untracked，撤销前预检 | `internal/snapshot` 按**文件名**存旧版本（已知缺口：同名两篇不猜、换库找不到），`ai.Verify` 改后只提示不阻断 | **P2**：改成按路径 + 按轮，撤销预检；这是已登记缺口的根治 |
| 权限档 | 只读（默认）/ 当前库可写 / 桌面完全访问；写入需显式选择，删除进回收站 | 工具集**全只读**，`AutoApply=false`，写回必须经 `review` + `snapshot`；没有显式档位 | **P2**：只读 / 当前库可写两档即可；**不做完全访问**（§9） |
| Prompt 库 | 场景 / 精选 / 用户三组，`{{blank}}` 表单、pinned strip、每条是 Markdown 文件、每条也是命令面板命令、分类搜索 | `AiAction`（6 内置 + 增删改 UI 已在 `settings_quick.go`）、`quickStrip` 上限 8、slash 后能加额外指令（`askingMore`） | **P1**：把 `AiAction` 升级为 Prompt 条目（blanks / 场景 / pinned / 文件化 / 分类），不另起数据模型 |
| 首屏场景化 | 按选区 → 阅读上下文 → 今日日记 → 当前笔记 → 库，决定标题与四条建议；无模型源只给「连接模型」 | 空白对话条只有输入框与快捷词 | **P1**：低成本，且能与我们已实现的 Daily notes 联动 |
| 回复落地 | 复制、追加到今日日记、追加到指定笔记 | 只有评审 → 写回当前笔记 | **P1**：复用 `ensureNote` 与 Daily notes 的今日笔记路径 |
| 会话管理 | 多会话、标题、最近对话、编辑用户消息就地重发（截断后续分支） | 每笔记一个 `aiSession`，持久化到 `state.AiSessions`；Forget 清空；无会话列表、无编辑重发 | **P2**：先做「按笔记」之外的会话切换，编辑重发后置 |
| 联网搜索 / 网页读取 | 厂商原生搜索 / OpenRouter server tools / Brave Key；公共页读取器接受 HTML/文本/Markdown/JSON/RSS，重定向逐次校验、DNS 解析后 pin 公网 IP、拒绝私网 | 无；`internal/ai` 只有本地工具 | **P2，需拍板**：网络边界与隐私代价（§9） |
| 上下文用量 | `ContextUsage`（used/size）+ 环，只显示真实上报数字 | `ai.Usage` 已解析，UI 未用；全局 `MaxContextChars` 裁剪，模型侧 `ContextLimit` 未参与 | **P0 清账**：先显示，再让 `ContextLimit` 参与裁剪 |
| 场景路由 | 每对话选模型 + effort | `AiRouteConfig` / `AiAction.Model` 有字段未接线 | **P0 清账** |

## 6 我们已做得更好，不要自废武功

逐条对照后，以下能力**优于或强于**参照对象，升级时保留：

- **Ollama 专门处理**（§3 第 5 行）：独立配置、探测、能力与上下文长度、keep_alive。
  补服务商分层时把它作为**并列的第一类来源**，不是塞进通用网格。
- **密钥三来源 + Keychain**：明文 / 环境变量 / 系统钥匙串，`KeyRef` 稳定命名，
  失败不阻断启动。参照对象只有 SecretStorage 一种。
- **能力实测 + 手动 pin**：`ai.Try` 真发一次带工具的请求，避免「声明支持 tools
  但一发就 500」的模型被误标；`CapabilitiesPinned` 让用户覆盖推断。
- **只读工具边界**：`vaultHost.within` 明确拒绝库外路径，ACCEPTANCE D16 有验收；
  参照对象要到「完全访问」才允许库外。
- **改前快照 + 改后校验 + 一律先过评审 diff**（D9 / D18 / D19，ai-assist.md 拍板）：
  写回路径有两道闸，比「Agent 直接改文件再撤销」更保守。
- **CJK 与离线**：原生 Go UI、无 webview 依赖（`mygo.WebViewAvailable()` 有回落）、
  字数按字计、无遥测、无自建服务器。
- **与其它能力的组合**：Daily notes / Templates / Web viewer / 批量标签任务 /
  Skills 两级发现，这些是宿主能力，参照对象要靠别的插件或 CLI。

## 7 分级路线

每期单独提交，`make check` 全绿后再动下一期；macOS（P0 平台）先验，再
Windows/amd64、Windows/arm64 与 Linux/amd64 编译。

### P0 清账：已登记待办 + 设置页内分组（2026-11-07 已落地）

1. `Ai.Route` 与 `AiAction.Model` 接线：`appState.ask` / `modelFor` 与
   `view.aiSend` 统一走一个 `resolveModel(kind, action)`，优先级
   动作指定 → 场景路由 → 栏内手选 → 默认 → 首个启用 → 备用。
2. 用量显示：`ai.Usage` 已在 `EventUsage`，在 `aiTalk` 回答尾部显示
   `prompt / completion / 用时`；让 `ContextLimit` 参与 `MaxContextChars` 裁剪。
3. 快照按库归属：把 `internal/snapshot` 的键从文件名换成「库 ID + 库内相对路径」，
   兼容旧文件名的读取与迁移；消除「同名两篇不猜」。
4. 动作提示词变量：`{{known_tags}}`、`{{title}}`、`{{selection}}`、`{{date}}`
   等，复用 `internal/templating.Expand`（注意它目前把未知变量原样保留）。
5. AI 设置页**内部分组**：模型 / 对话 / 动作 / 工具四个子页，用
   `ui.SidebarSection` 或页内 tab；`ai()` 按组调用，不改 `config.json` 结构。
6. 顺手修过时的文档描述：[XINKGO_FEATURES.md](../XINKGO_FEATURES.md) §6 与
   [ACCEPTANCE.md](../ACCEPTANCE.md)「已知缺口」仍写「动作没有增删界面」，而
   `settings_quick.go` 的 `aiQuickWords` 已经能增删改与恢复内置项。规划前先把
   清单对齐，否则会把已完成的事又排进路线。

落点：`internal/app/{ask.go,settings_ai.go,settings_quick.go}`、
`internal/view/{ai_bar.go,ai_talk.go,ai_ask.go}`、`internal/config/ai.go`、
`internal/snapshot/`、`internal/locale/`。测试：路由优先级、用量渲染、
快照跨库同名、变量展开、分组渲染（`ui.Tester`）。验收项拟为 L1 ~ L6。

实际落点比上面更集中：`config.AiConfig.ResolveModel` +
`view.askModel`（路由）、`view.aiBlock.Usage` 与 `aiSaid`（用量）、
`view.contextMaxChars`（裁剪）、`view.expandActionPrompt` +
`templating.Vars.Extras`（变量）、`snapshot.NameFor/WriteFor/Scope` +
`appState.snapshotNote` / `whereBack`（按库快照）、
`settings.aiSection`（分组）。六项全部有单测，验收项
[ACCEPTANCE.md](../ACCEPTANCE.md) L1 ~ L6；`XINKGO_FEATURES.md` §6 与
「已知缺口」里过时的描述已同步修正。

### P1 配置结构 + 上下文 + 改写（2026-11-07 已落地）

1. **服务商分层**：`config.AiConfig` 增 `Providers []AiProvider`
   （id / name / baseURL / protocol / apiKeySource / keyRef / models / enabledModels /
   showInPicker / fetchedAt），`Models` 保留为向后兼容的扁平视图；
   `fromFile` / `fillMissing` / `Sanitize` 加迁移：老文件缺 `providers` 时从
   `models` 按 `(baseURL, key)` 反推服务商。**Ollama 保持独立入口**。
2. **模型选择器 popover**：替代 `ui.Select`，搜索 + 来源分组（Ollama / 各服务商 /
   最近）+ 键盘；`recentModels` 落在 `state`（不写 `config.json`，属运行期状态）。
3. **上下文 chip**：`view` 增 `AiContext{Kind, Label, Removable}`，由 app 在提问
   时灌入当前笔记 / 选区；可移除、移除后保持到下次选区变化；选区在对话条获得
   焦点时用编辑器自身的 selection 绘制保持可见。
4. **自由指令改写选区**：新增 `internal/ai/inline.go` 解析 `<replacement>`，
   `internal/view` 做词级 diff（Go 侧分词，中文按字/词，不引 `Intl.Segmenter`）；
   复用 `StartPieceReview` / `writeReview` 与 `inlineTargetIsCurrent` 等价的
   编辑器快照校验。**仍一律先过评审 diff**。
5. **回复落地**：复制 / 追加到今日日记 / 追加到指定笔记，复用
   `appState.ensureNote` 与 `dailyRelPath`。
6. **Prompt 库**：把 `config.AiAction` 扩展为 Prompt 条目（blanks / 场景 /
   pinned / 分类 / 可选文件化到 `<vault>/.xink/prompts/`），`quickStrip` 保留，
   设置页从「快捷词列表」升级为「库」；每条仍是命令面板命令。
7. **首屏场景化**：空白对话按选区 / 今日日记 / 当前笔记 / 库给标题与建议。

落点：`internal/config/ai.go`、`internal/app/{settings_ai.go,settings_prompts.go,
ask.go,files.go}`、`internal/view/{ai_bar.go,ai_talk.go,ai_context.go,
picker.go,quick.go,review.go}`、`internal/ai/inline.go`、`internal/locale/`。
风险：`config.json` 结构演进必须兼容 xink 旧文件；视图每帧不做 I/O，拉模型列表 /
解析网页一律 goroutine + `mygo.RunOnMain`。验收项 L7 ~ L16。

实际落点：`config/{ai.go,ai_providers.go}`（服务商、迁移、预设、blanks 解析）、
`app/{settings_ai_provider.go,settings_quick.go,ask.go,ai_reply.go}`、
`view/{ai_p1.go,ai_bar.go,ai_talk.go,ai_ask.go,quick.go}`、`ai/inline.go`、
`state/recent_models.go`、`internal/locale/`。验收项
[ACCEPTANCE.md](../ACCEPTANCE.md) M1 ~ M8。

与上面清单有意不同的几处，都是为了让改动更贴合既有代码：

- 服务商只存**元数据**（id / name / baseURL / 密钥来源 / showInPicker /
  fetchedAt），可调用模型仍是 `Models` 扁平列表，每条加 `ProviderID`。这样
  所有现有读取点不用改，迁移只是把老模型按地址归组；规划里的 `models[]` /
  `enabledModels[]` 由 `Models` 本身承担。
- 上下文 chip 由 `view.syncContexts` 从当前笔记与选区自己生成（不经过 app），
  因为编辑器状态本来就在 view；移除状态留在 `aiSession`，`aiBrief` 发消息时
  跳过已移除的上下文。
- 自由指令改写复用 `StartPieceReview` + MyGo 的 `editor.DiffState`，没有另写
  一套词级 LCS：`pieceDiff` 已经是逐块 diff，且我们的边界校验
  （笔记中途变过就拒绝写回）比参照对象更严。
- 回复落地用 `appendToFile`（只追加、不覆盖）而非 `ensureNote`，因为「追加」
  和「新建」语义不同；今日日记路径仍由 `dailyRelPath` 决定。
- Prompt 库的「文件化」是评审稿里的可选项，本轮未做：固定 / 分类 / 场景 /
  blanks 表单都在，库仍是 `Ai.Actions` 里的条目，避免再引入一套文件发现与
  迁移。
- 首屏按「有选区 / 有笔记 / 无笔记」给三种标题与提示，没有单独做「今日日记」
  场景；建议沿用 quickStrip 的动作 chip，不再画第二排按钮——这样不会与
  slash 输入的焦点竞争（有过一次真实回归，已修并加了测试）。

### P2 agentic 体验（2026-11-07 已落地）

问题卡（`ask_user` 工具 + 新 `ai.EventKind` + 卡片）、时间线/计划/折叠已处理、
上下文用量环、按轮变更追踪与预检撤销（替换按文件名快照）、会话列表与编辑重发、
`@文件`/`@文件夹` 引用、显式权限两档（只读 / 当前库可写）。**联网搜索与公共
网页读取本次不做**，所以 `@` 引用只覆盖库内文件与文件夹。

实际落点：`ai/{event.go,question.go,tool.go,agent.go}`（`ask_user` 工具与
`EventQuestion`，Loop 遇到提问即结束本轮）、`view/{ai_p2.go,ai_bar.go,
ai_ask.go,ai_talk.go,review.go}`、`app/{ai_reply.go,settings_ai.go,app.go}`、
`config/ai.go`、`internal/locale/`。验收项
[ACCEPTANCE.md](../ACCEPTANCE.md) N1 ~ N8。

有意与清单不同的几处：

- **问题卡**按规划走 `ask_user` 工具：模型一调用它，Loop 发出
  `EventQuestion` 并结束本轮，用户回答后以该回答作为新一轮提问。没有做
  Codex / ACP 那种「同一轮里继续跑」的协议续接，因为我们的 Loop 是一次
  性的；这样每一轮仍是完整的一次请求，状态机更简单。
- **时间线**做折叠的「已探索 N 个文件，M 次搜索」+ 展开逐步 + 用时，
  **没有做计划清单**：我们没有模型上报 plan 的通道，硬造一个只会显示假数据。
- **按轮撤销**是「最近一次评审写回」的单步撤销，撤销前检查文件是否仍是
  写入后的样子；不是参照对象那种一次 Agent 回合涉及多文件的追踪器。快照
  仍留在磁盘，作为更长的历史。
- **`@` 引用**只支持库内单个笔记（正文一次性读入并作为可移除 chip），
  未做 `@文件夹`：文件夹展开需要新的索引遍历与字数上限策略。
- **权限两档**是「只读 / 允许追加到笔记」，不是「只读 / 当前库可写」：
  工具集本来全只读，评审写回无论哪一档都要过 diff；append 档只放开回复
  落地的追加动作。这样与 ai-assist.md「一律先过评审 diff」的拍板不冲突。
- **会话列表**是 AI 栏头部的一个 popover，列出有对话的笔记并切换过去；
  编辑重发是就地截断后续分支再重新提问。没有另开一个会话管理窗口。

### P3 独立立项 / 明确不做

- **外部 CLI Agent（§4 解读 B）与联网搜索 / 公共网页读取**：2026-11-07 明确
  本次不做，也不作为「待办」留在路线里；需要时另立设计。
- React / TypeScript / Obsidian DOM 渲染 / Mermaid / 六套会话配色：我们是原生
  UI，已有全局 `theme`；照搬只会破坏一致性与零依赖。
- 账号订阅、OAuth loopback、Magpie 网关、图片生成/编辑、「完全访问」权限档。

## 8 数据模型与兼容

- 所有 AI 选项继续放 `config.json` 的 `ai` 段，**不**挪进 `plugins.json`
  （后者是插件选项，见 [daily-notes-templates.md](daily-notes-templates.md)）。
  `config.json` 是与 xink 兼容的六段结构，新增字段必须走
  `fillMissing`（缺字段取默认）与 `Sanitize`（钳制 + 迁移），保证手改的旧文件可用。
- `AiProvider.ID` 稳定：服务商改名不应让已启用模型失效；从 `Models` 反推时用
  `(BaseURL, ApiKeySource, KeyRef)` 作归并键，冲突时保留最早的一条并记日志。
- `recentModels`、会话列表、上下文 chip 的「已移除」状态属运行期或 `state`，
  不写 `config.json`；与 `open_notes.json` / `ai_sessions.json` 同一套原子写。
- 新增界面文案全部进 `internal/locale/en.json` + `zh-Hans.json`，跑
  `go tool mygo generate`；**不要**像参照对象那样把中文写死在代码里。

## 9 约束、风险与不做的事

- 不引入 cgo / Node / JS 引擎 / 新 Go 依赖；`go.mod` 不变。模型列表拉取、能力
  探测、网页读取都在 goroutine，结果 `mygo.RunOnMain` 回主线程；视图每帧不做
  阻塞 I/O。依赖方向仍 `app → view/config/...`，`view` 不 import `app`。
- **一律先过评审 diff**（ai-assist.md 拍板）不因「直接改文件」而放宽：任何写回
  都经 `review` + `snapshot`；权限档最多到「当前库可写」，不做完全访问。
- **联网搜索与公共网页读取本轮不做**，因此不引入任何外部网络读取路径；
  `@` 引用只解析库内路径。将来若单独立项，必须带参照对象同级别的边界：只收
  公共 HTTP/HTTPS、拒绝私网与 loopback、重定向逐次校验、外部文本当不可信材料、
  工具循环有步数与字数上限、无遥测、密钥不进日志。
- macOS P0 先验；服务商预设里的国内端点、keychain、临时文件等差异要在
  Windows（WebView2）/ Linux（WebKitGTK）上复核。
- 不照搬它的 CSS / React 组件 / 品牌图标；只借鉴信息架构与交互原则，视觉沿用
  `internal/theme` 与现有 `group()`/`settingRow()`/`cardBox()` 组件。

## 10 拍板结论（2026-11-07）

| # | 问题 | 结论 |
|---|---|---|
| 1 | 「支持添加 Agent」指外部 CLI 还是内置工具循环配置档？ | **不做外部 CLI**。按解读 A 把内置工具循环做成配置档，落在 P0/P1；解读 B 从本路线移除，不在 P3 挂待办 |
| 2 | 是否接受 `config.json` 增加 `Providers` 层与迁移？ | **接受**（按推荐），`Models` 保留兼容视图 |
| 3 | Ollama 是否保持独立入口？ | **保持**（按推荐），作为并列第一类来源 |
| 4 | 是否做联网搜索 / 公共网页读取？ | **不做**。`@` 引用只覆盖库内文件与文件夹 |
| 5 | 权限档是否需要「完全访问」？ | **不需要**（按推荐），最多当前库可写 |
| 6 | Prompt 库沿用 `AiAction` 还是另立数据？ | **沿用并扩展**（按推荐），不另起一套触发词 |
| 7 | 先做 P0 清账还是先做 P1 结构？ | **先 P0**（按推荐），再 P1；P2 逐项再评 |

其余推荐（P0 的六项、P1 的服务商分层 / 模型选择器 / 上下文 chip / 选区改写 /
回复落地 / Prompt 库 / 首屏引导，以及 §9 的约束）全部按本稿执行。

## 11 参考

- <https://github.com/joeseesun/qiaomu-agent>（`DESIGN.md`、`docs/research/`、
  `README.md`；本轮浅克隆版本 v0.6.x）
- [ai-assist.md](ai-assist.md)：2026-11-02 AI 编辑辅助拍板
- [XINKGO_FEATURES.md](../XINKGO_FEATURES.md) §6 与「已知缺口」：AI 待办登记
- [ACCEPTANCE.md](../ACCEPTANCE.md) D 组：AI 现有人工验收
- [ARCHITECTURE.md](../ARCHITECTURE.md)：分层、线程模型、持久化与扩展点
- [daily-notes-templates.md](daily-notes-templates.md)：插件选项与预览标签页，
  P1 的 `ensureNote` / 今日日记路径来自这里
