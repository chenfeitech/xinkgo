# 研发计划：复刻 xink 到 xinkgo

> 需求：[../requirements/xink-port.md](../requirements/xink-port.md)
> 设计：[../design/xink-port.md](../design/xink-port.md)
> 选型：原生 UI · AI 全量（P2）· P0 一次性交付 · 数据格式与 xink 兼容。

每个任务的验收命令：

```sh
GOTOOLCHAIN=auto go test ./internal/<包>/... -count=1   # 该包的测试
GOTOOLCHAIN=auto make check                             # 全量门禁（每次提交前）
```

**进度（2026-10-30）**：M0 全部完成；**M1 = P0 全部完成**，在
`feature/port-xink` 分支上，待人工验收（[../ACCEPTANCE.md](../ACCEPTANCE.md)
A1 ~ A18）。未合并到 `main`。

最后一次 `make check`（提交 225ba84）：

```
== go test ./... ==       15 个包全 ok（config fs highlight keymap logging
                          markdown media note session state sync vault view）
== 全平台编译 ==          ok darwin/arm64 · darwin/amd64 · windows/amd64
                          ok windows/arm64 · linux/amd64
== 版本一致 ==            ok 0.1.0
Everything passed.
```

## M0 领域层底座（T-1 ~ T-7 全部完成 ✅）

| ID | 任务 | 依赖 | 关联需求 | 验收 |
|---|---|---|---|---|
| T-1 | `internal/config`：AppConfig 五段结构、默认值、`sanitize()` 钳制、`legacy_code_theme` 迁移、load/save | — | R-9.1、R-9.6、R-9.7 | `go test ./internal/config/...` |
| T-2 | `internal/state`：`open_notes.json`、`vaults.json`、`layout.json` 读写 + 原子写 + `vault.json` 迁移 + 归一化 | T-1 | R-9.2-9.5、R-9.8、R-6.9 | `go test ./internal/state/...` |
| T-3 | `internal/vault`：递归扫描、忽略规则、增删改移、注册表状态与回落 | T-2 | R-1.2-1.10、R-1.18、R-2.1-2.6 | `go test ./internal/vault/...` |
| T-4 | `internal/fs`：`unique_name`、`rename_name`、跨设备移动、导入 | — | R-1.18、R-2.6 | `go test ./internal/fs/...` |
| T-5 | `internal/session`：`tab_key`、保序去重、`active` 回落、失效过滤 | — | R-3.2、R-3.11、R-6.4 | `go test ./internal/session/...` |
| T-6 | `internal/note`：内容/dirty/mtime/version、原子写、外部变更取回（一致则不回填） | T-4 | R-4.3、R-7.5-7.8 | `go test ./internal/note/...` |
| T-7 | `internal/media`：文件类型与图标、可编辑文本、编辑器语言、`.xink` 排除 | — | R-1.9、R-1.11、R-10.10 | `go test ./internal/media/...` |

## M1 = P0（一次性做完再验收：T-8 ~ T-22 全部完成 ✅）

| ID | 任务 | 依赖 | 关联需求 | 验收 |
|---|---|---|---|---|
| T-8 | `internal/view/file_tree.go`：文件树（缩进 + 展开态 + 选中 + 跟随） | T-3、T-7 | R-1.6、R-1.12-1.14 | `go test ./internal/view/...` |
| T-9 | 打开库：对话框 + 注册表 UI + 库管理 | T-3 | R-1.1-1.5 | 人工（需 GUI） |
| T-10 | 笔记增删改移的 UI 接线（新建/重命名/删除/移动/拖放导入/粘贴） | T-8、T-4 | R-1.15-1.17、R-2.1-2.6 | 人工 + 领域层单测 |
| T-11 | 编辑器：`ui/editor` 接入、Buffer、软换行/行号/缩进 | T-6 | R-4.1、R-4.9 | 人工 |
| T-12 | 自动保存：300ms 防抖、`Cmd+S`、原子写、保存后刷索引（索引此时可为空实现） | T-11、T-6 | R-4.1-4.6、R-4.13 | 人工 |
| T-13 | 多标签：唯一入口 `openNote`、判重、空白页不变式、落盘 | T-5、T-2 | R-3.1-3.11 | `go test ./internal/view/...` + 人工 |
| T-14 | Markdown 工具栏（13 个按钮）+ 选中包裹/光标插入 | T-11 | R-4.7、R-4.8 | `go test ./internal/view/...` |
| T-15 | 预览：`ui/markdown.View` + 图片 + 链接点击 + 密度/宽度/字体 | T-6 | R-5.1-5.6、R-5.11、R-5.13 | 人工 |
| T-16 | 代码高亮与主题：3 套预置、明暗跟随 | T-15 | R-5.7、R-5.8、R-5.15 | 人工 |
| T-17 | 滚动同步：锚点表 + 方向锁 + 三档模式 | T-15 | R-5.9、R-5.10、R-5.16 | `go test ./internal/sync/...` + 人工 |
| T-18 | 布局：三栏 + 分隔条 + `layout.json` 持久化（版本不符重建） | T-8 | R-8.22、R-6.1 | 人工 |
| T-19 | 会话恢复：布局 → 标签 → 激活 → 空白页校正，恢复后写回 | T-2、T-13、T-18 | R-6.1-6.9 | 人工 |
| T-20 | 外部监听：`mygo.WatchFiles` + 事件分类 + 三态处理 + 忽略 `*.md.tmp` | T-6、T-3 | R-7.1-7.9 | `go test ./internal/watcher/...` + 人工 |
| T-21 | 单实例（pid 文件） | — | R-8.21 | 人工 |
| T-22 | 快捷键表与键位覆盖 | T-1 | R-8.19 | `go test ./internal/keymap/...` |

## M2 = P1（T-23 ~ T-32 全部完成 ✅，2006-10-30；待人工验收）

人工验收登记在 [../ACCEPTANCE.md](../ACCEPTANCE.md)：B1 ~ B8（面板与检索）
与 C1 ~ C9（设置、库、配置生效、数据文件）。

最后一次 `scripts/check.sh`（提交 92f4ebc）：

```
== go test ./... ==       19 个包全 ok
== 全平台编译 ==          ok darwin/arm64 · darwin/amd64 · windows/amd64
                          ok windows/arm64 · linux/amd64
== versions agree ==	   ok 0.1.0
Everything passed.
```

决策：YAML / TOML 只判定不校验、不格式化——本机 `go get gopkg.in/yaml.v3`
取不到（网络不通），判定集中在 `internal/datafile` 一个包里，日后补库
不必动窗口。

| ID | 任务 | 依赖 | 关联需求 |
|---|---|---|---|
| T-23 | `internal/markdown`：facts 提取 | — | R-8.1、R-8.5、R-8.8 |
| T-24 | `internal/index`：主表 + 派生表、detach/attach、双链解析、反链、标签 | T-23 | R-8.2-8.7 |
| T-25 | `internal/fuzzy` + 命令面板（Cmd+P，前 50 条，高亮）+ 索引接线 | T-24 | R-8.11 |
| T-26 | `internal/search` + 搜索面板（Cmd+Shift+F，防抖 200ms，上限 100，跳行） | T-24 | R-8.12、R-8.13 |
| T-27 | 大纲面板（实时解析，点击跳行）+ 抽屉骨架 | T-23 | R-8.14 |
| T-28 | 右侧按钮条 + 抽屉承载 大纲/反链/标签（同项再点收起） | T-24、T-27 | R-8.15 |
| T-29 | `[[` 补全浮层 + 写入相对路径 | T-24 | R-8.9、R-8.10 |

### P1 任务表

| ID | 任务 | 依赖 | 关联需求 |
|---|---|---|---|
| T-23 | `internal/markdown`：facts 提取（headings/links/wikilinks/tags，带行号，跳过代码块） | — | R-8.1、R-8.5、R-8.8 |
| T-24 | `internal/index`：主表 + 5 张派生表、detach/attach 增量、双链解析、反链、标签 | T-23 | R-8.2-8.7 |
| T-25 | `internal/fuzzy` + 命令面板（Cmd+P，前 50 条，高亮） | T-24 | R-8.11 |
| T-26 | `internal/search` + 搜索面板（Cmd+Shift+F，防抖 200ms，上限 100） | T-24 | R-8.12、R-8.13 |
| T-27 | 大纲面板（实时解析，点击跳行） | T-23 | R-8.14 |
| T-28 | 右侧按钮条 + 抽屉（大纲/反链/标签） | T-24、T-27 | R-8.15 |
| T-29 | `[[` 补全浮层 + 写入相对路径 | T-24 | R-8.9、R-8.10 |
| T-30 | 设置窗口 + 库管理窗口 + 配置导入导出 | T-1 | R-8.20 |
| T-31 | 配置项接线（密度/字体/主题/图片/同步模式等即时生效） | T-1、T-15 | R-8.16-8.18 |
| T-32 | `internal/datafile`：JSON/YAML/TOML 判定、格式化、校验诊断 | — | R-4.11、R-4.12、R-4.15 |

## M3 = P2（AI 全量）

| ID | 任务 | 依赖 | 关联需求 |
|---|---|---|---|
| T-33 | Provider 抽象：Ollama native + OpenAI 兼容、统一请求、重试与备用模型 | T-1 | R-10.1 |
| T-34 | 流式事件解析：Delta/Thinking/ToolCall/Truncated | T-33 | R-10.2 |
| T-35 | AI 对话条 UI + `markdown.Stream` 渲染 | T-34 | R-10.3 |
| T-36 | 会话持久化（`ai_sessions.json`，30 会话/200 块裁剪） | T-2 | R-10.4 |
| T-37 | AI 修订：`editor.DiffView` 逐块接受/拒绝 | T-34 | R-10.5 |
| T-38 | 只读工具集 + agent loop + 结果校验 + 快照轮转 | T-33 | R-10.6 |
| T-39 | Skills：`SKILL.md` 发现（库内 `.xink` + 用户级）、命令面板入口、启停 | T-33 | R-10.7 |
| T-40 | 库级批量任务（加/改/删标签）：只读计划 + 批量原子写 + 中断恢复 | T-24、T-33 | R-10.8 |
| T-41 | 模型能力清单（静态推断 + 手动改 + 探针） | T-33 | R-10.9 |
| T-42 | `.xink` 目录恒跳过（树/索引/命令面板） | T-7 | R-10.10 |
| T-43 | 快照取回：历史窗口 + 取回（取回前先把当前存一份） | T-38 | R-10.6 |
| T-44 | 模型列表的克隆与排序（副本深拷贝、上下移动一位） | T-41 | R-10.9 |
| T-45 | 设置里补上只在 `config.json` 里能改的 AI 旋钮，并给 Ollama 常驻时间一个输入框 | T-41 | R-10.9 |
| T-46 | AI 动作：命令面板入口（Cmd+P）、整篇 + 选区、结果一律走评审 diff | T-37、T-41 | R-10.7 |

## M3 进度

| ID | 状态 | 说明 |
|---|---|---|
| T-33 | ✅ | `internal/ai`：Provider 抽象 + 分帧 + 重试与备用模型 |
| T-34 | ✅ | `internal/ai`：流式事件解析（Delta/Thinking/ToolCall/Usage/Truncated/Done/Failed） |
| T-35 | ✅ | AI 对话条 UI（`internal/view/ai_bar.go` + `markdown.Stream`）+ app 侧提问接线 |
| T-36 | ✅ | 会话持久化（`internal/state/ai_sessions.go`，30 会话 / 200 块裁剪） |
| T-37 | ✅ | AI 修订评审：`editor.DiffView` 逐块 Take/Revert + 整篇写入 |
| T-42 | ✅ | `.xink` 恒跳过：随 T-7 一起做完（`vault.Scan` 的 `ignored` 用 `media.AIDirName`，树/索引/命令面板都不出现），`vault_test.go` 有断言 |
| T-40 | ✅ | 库级批量任务：领域层 `internal/task`（整 token 命中、代码块/front-matter 分别处理、跳过必带原因、`Plan` 只读、`Run` 分批可续跑）+ 窗口 `internal/app/tasks_window.go`（选动作/填标签/看预览/分批执行，写盘走 `note.Save` 原子写，写完回灌索引） |
| T-38 | ✅ | 只读工具集 `internal/ai/tool.go`（7 个工具 + schema + `Execute`）→ `agent.go` 的 `ai.Loop`（要工具→执行→回灌→再问，走满步数报 `ErrSteps`）→ `app/host.go` 的 `ai.Host` 实现（工具只能读**库内**，路径越界一律拒绝）→ 对话条显示 `tool → 结果` 小字 |
| T-39 | ✅ | Skills：`internal/skill` 发现与解析（库内 `.xink/skills` + 用户级、同名库内优先、坏文件进 issues）、命令面板入口、AI 条里「using X」可放下、设置里启停 |
| T-41 | ✅ | 能力清单：领域层 `internal/ai/capability.go`（字典 + 推断 + 生效优先级）+ `probe.go`（`ai.Try` 探针）；设置窗口 `internal/app/settings_ai.go`（模型增删、地址/密钥/协议、能力勾选即 pin、Try it 把实测 `tools_enabled` 写回） |
| T-43 | ✅ | 快照取回：`internal/snapshot` 补 `List`（新的在前）与 `Taken`（从名字读回时间）；`internal/app/history_window.go` 列快照（文件名 + 时间 + 开头一行），Put it back **先把当前内容存成一份新快照**再写回（还是 `note.Save` 原子写），写完回灌索引并让开着的标签页重新读盘（`Tabs.PutText`） |
| T-44 | ✅ | 模型克隆与排序：`config.ModelEntry.Clone` 深拷贝（能力切片与三个指针都不共享）并给副本 `<原名>-copy`（重名加数字）；每行 Clone / ↑ / ↓，三种改动与 Forget 一样**画完这一帧再做**，免得从后面几行的脚下抽走它们要画的东西 |
| T-45 | ✅ | AI 旋钮：设置 ▸ AI 加 Asking 一组（输出语言 / 两个超时 / 重试次数与间隔 / 步数 / 笔记与工具能看见多少字），加 Ollama 常驻时间输入框。两端与 `AiConfig.Sanitize` 的夹取**完全一致**；整数用 `NumberInput`（说得出当前值、一次走一格）而非 `Slider`，写回按四舍五入取整 |

| T-46 | ✅ | AI 动作：命令面板 `Action · <名字>`（`internal/view/palette.go` 与 skill 并列，共用一个面板）→ `runAction`（`internal/view/ai_bar.go`：`pieceOf` 定范围、`askingOf` 组话）→ 答完由**画帧** `aiReviewDone` 开评审（整篇走 `StartReview`，选区走新的 `StartPieceReview`，`writeReview` 只把那一段放回去） |

人工验收登记在 [../ACCEPTANCE.md](../ACCEPTANCE.md)：D1 ~ D19 与 E1 ~ E6。

冒烟：`vault open` + `index built`，窗口 ALIVE。

最后一次 `scripts/check.sh`：

```
== go test ./... ==       27 个包全 ok
== 全平台编译 ==          ok darwin/arm64 · darwin/amd64 · windows/amd64
                          ok windows/arm64 · linux/amd64
== versions agree ==      ok 0.1.1
Everything passed.
```

决策：

- **`Piece.Data` 只承载"一行 NDJSON"或"一个 SSE `data:`"，解析留给 T-34。**
  分帧与语义分开，两端各自可单测。
- **OpenAI 兼容端点没有默认地址**：配置里没写 `base_url` 就直接报错，不猜
  `api.openai.com`。Ollama 才有默认（`ollama.base_url`），与 xink 一致。
- **Ollama 的 `think` 放顶层**，`num_ctx`/`num_predict`/`temperature` 放
  `options`；`keep_alive` 顶层。
- **超时用 `http.Client{Timeout}` 而不是 `context.WithTimeout`**：前者把读
  body 的时间也算进去，正好是"整个回答"的超时；首字节超时另用一层
  `RoundTripper`，两者互不干扰。
- **事件用 `Kind` 标签的结构体，不用接口**：Rust 的 `StreamEvent` 枚举到 Go
  没有对等物；七个 kind 共用一个 `Event`，字段由 `Kind` 说明哪个有效，
  比七种实现同一接口的类型好写也好读。
- **`Truncated`（预算耗尽）与 `ErrTruncated`（连接中断）分开**，前者在
  `Done` 之前发出且不是错误——重试一遍只是再烧一次 token。
- **写盘只在画帧时做，不在听答案的 goroutine 里做**：会话表是 app 自己的，
  另一条 goroutine 碰它就是数据竞争；`aiHear` 只置 `dirty`，由 `aiBar` 写。
- **裁到 30 个会话时按路径排序取稳定子集**（与 xink 一致）：会话没有时间戳，
  "哪 30 个"不重要，"每次都是同一 30 个"才重要。
- **批量标签动作不经模型**：规则可判定（整 token 命中），所以是纯函数 + 一次读盘，
  能单测；`Plan` 只读盘算预览，`Run` 才写，且**执行时重新读盘**（计划可能已经过期）。
- **工具一律只读，且只能读库内**：`vaultHost.within()` 把路径钉在 vault root 下，
  `../` 与库外绝对路径一律拒绝。模型是**不可信输入源**，它给的 path 不能直接喂给
  `os.ReadFile`；skill 资源同理（`skill.Resource` 钉在技能目录内）。
- **工具失败不返回 error，而是返回"说给人/模型听的话"**：工具名不存在、参数缺失、
  笔记读不到，都是要回灌给模型让它改正的信息，不是 app 的失败。
- **取回之前先把当前存成一份新快照**：写进去却取不回来的快照等于没有，而"取回"
  本身就是一次改笔记——它同样要留一条退路，否则取回旧稿就是丢掉新稿。
- **快照按文件名认笔记，不猜**：快照名里只有文件名，没有它属于哪个库。取回时在
  当前库里按名字找：开着的优先，其次库里唯一的那篇；同名两篇报「more than one
  note is called X」，库里没有报「X is not in this vault」——**把话放回错误的笔记
  比不放回更糟**。
- **取回要碰到编辑区，不能只写盘**：`Tabs.PutText` 让开着的标签页重新读盘并更新
  buffer；否则自动保存会把编辑区里的旧文字又盖回去。
- **动作的三个取舍由用户拍板（2026-10-31）**：入口走**命令面板**（与 `Skill · xxx`
  并列，复用已有面板，改动最小）；作用域**整篇 + 选区**（有选区就只做选区，`Scope = whole`
  的动作则永远做整篇）；结果**一律走评审 diff**（`replace` / `insert` 都不自动写——
  写进笔记前的最后一眼是用户自己的）。
- **评审由画帧发起，不听答案的 goroutine 发起**：`StartReview` / `StartPieceReview` 要
  造 `editor.DiffState`、动 `w.review`，那都是视图自己的东西。所以会话上挂一个
  `waiting`（"这个答案完事了要评审、是哪一段"），由 `Workspace.View` 每帧看一眼——
  与"写盘只在画帧做"是同一条规矩。
- **选区评审的快照存整篇**：`reviewState.before` 在选区评审里只是那一段，照它存快照会
  让"取回"把整篇笔记换成那一段。`writeReview` 因此改成存**当时整篇**的文字。
- **面板里两行不能共用一个键**：`paletteList` 原先拿 `item.Path` 当 widget key，而
  skill / action 的 Path 都是空——两行会撞成一个。改成 `item.key()`：笔记用路径，
  skill / action 用名字。
- **只给"真有人读"的字段做界面**：把 AI 配置全查了一遍，`Ollama.NumCtx`（全局那个，
  只有模型自己没设时才该起作用，而它从没被进请求体）、`ApiKeySource`、`ContextLimit`、
  `LastCheckedMs`、`MemoryEnabled`、`AutoApply` 六个**运行期无人读取**或一个也不发，
  所以一个都不补——给一个不生效的值做旋钮，是骗人。补的八个都是
  `internal/ai` 或 `internal/view` 里真有读取点的。
- **窗口的两端就是设置的两端**：每个 `NumberInput` 的 lo/hi 抄 `AiConfig.Sanitize`
  的夹取范围（5–600、1000–200000…），免得窗口给出设置随后又拒绝的数。
- **副本要深拷贝**：`ModelEntry` 里有切片（`Capabilities`）和三个指针
  （`ContextLimit`、`Params.NumCtx`、`Params.KeepAlive`）。复制是为了改它，共享
  这些就是"改副本也改了原件"，所以 `Clone` 一个个拷过去。副本名 `<原名>-copy`，
  重名再加数字——名字是提问时用来说清问哪一个的。
- **列表的增删改一律\"画完这一帧再做\"**：`for i := range` 已经把长度定下了，而每行
  又拿着 `&Models[at]` 的指针；中途插入、删除或换位，后面几行就会画到别的模型上。
  所以 Forget / Clone / ↑↓ 都只记下\"要做哪个\"，画完才动手（与 Forget 已有的做法一致）。
- **自动保存改为"记下时间、由画帧去写"**，不再用 `time.AfterFunc` 直接写。
  `Note` 与 `OpenNote` 都没有锁，定时器那条 goroutine 与主线程（Cmd+S）同时
  写就是数据竞争——`-race` 已实测复现（`n.saving` 与 `Note.Text`）。写盘只
  在主线程做，是这项目从一开始就立的规矩。

## 决策记录（实施中补写）

| 日期 | 决策 | 原因 |
|---|---|---|
| 2026-10-30 | `layout.json` 不复用 xink `DockAreaState`，用自研结构 + 版本不符重建 | 布局系统不同；与 xink"版本不符即重建"行为等价 |
| 2026-10-30 | 文件监听用 `mygo.WatchFiles`，不引第三方库 | 官方 API，跨平台且不破坏 `CGO_ENABLED=0` |
| 2026-10-30 | 索引不落盘 | 与 xink 一致，索引是磁盘的派生数据 |
| 2026-10-30 | 单实例用 MyGo 官方 `App.RequestSingleInstanceLock()` 而非自研 pid 文件 | v0.7.3 有官方 API（跨域 socket + 存活判定），比 pid 文件可靠；`OnSecondInstance` 用来唤起窗口 |
| 2026-10-30 | 三栏分隔条自研（可用宽度由中间栏吃掉），不用 `ui.Split` | `ui.Split` 只能让**最后一个**面板吃剩余宽度，会让预览随窗口变宽、编辑区不动；自研分隔条才能让编辑区吃掉剩余宽度 |
| 2026-10-30 | 代码高亮只提供"主题 → `editor.Theme`"，编辑区与预览区共用同一个 | `markdown.Theme.CodeColors` 接受 `*editor.Theme`，一端生成两端使用，天然同源（R-5.7） |
| 2026-10-30 | `sync.BlockSpans` 用"空行分块 + 围栏整体不分"的扫描，不用 mdast | Go 侧无 mdast；预览块与源码行的对应关系按空行划分足够，且可单测 |
| 2026-10-30 | 库改名 / 重新定位 / 删除（R-1.2 的后半）留到 P1 的库管理窗口 | P0 只做"新增 + 切换 + 健康状态提示"，窗口是 P1 的 T-30 |
| 2026-10-31 | 修：模型的 key 从未被发出（`NewEndpoint` 不填 `APIKey`，`PlainKey` 无读者） | 现象明确、改法唯一（一个来源一个去处）、只动 `internal/ai`，属缺陷修复豁免；测试钉住"写进去的 key 上了线"，人验见 E4 |

## 人工验收

登记在 [../ACCEPTANCE.md](../ACCEPTANCE.md)：A1 ~ A18，含打开库、增删改移、
拖入导入、粘贴、多标签、自动保存、工具栏、预览与同源高亮、滚动同步、
外部变更、重启恢复、三栏与分隔条、单实例、键位覆盖。
