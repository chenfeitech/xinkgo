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

## M3 进度

| ID | 状态 | 说明 |
|---|---|---|
| T-33 | ✅ | `internal/ai`：Provider 抽象 + 分帧 + 重试与备用模型 |
| T-34 | ✅ | `internal/ai`：流式事件解析（Delta/Thinking/ToolCall/Usage/Truncated/Done/Failed） |
| T-35 | ✅ | AI 对话条 UI（`internal/view/ai_bar.go` + `markdown.Stream`）+ app 侧提问接线 |
| T-36 | ✅ | 会话持久化（`internal/state/ai_sessions.go`，30 会话 / 200 块裁剪） |
| T-37 | ✅ | AI 修订评审：`editor.DiffView` 逐块 Take/Revert + 整篇写入 |
| T-42 | ✅ | `.xink` 恒跳过：随 T-7 一起做完（`vault.Scan` 的 `ignored` 用 `media.AIDirName`，树/索引/命令面板都不出现），`vault_test.go` 有断言 |
| T-41a | ✅ | 能力清单领域层：`internal/ai/capability.go`（字典 + 推断 + 生效优先级）+ `probe.go`（`ai.Try` 探针）。设置窗口里的模型列表与手动勾选是 T-41b |

人工验收登记在 [../ACCEPTANCE.md](../ACCEPTANCE.md)：D1 ~ D7。

冒烟：`vault open` + `index built`，窗口 ALIVE。

最后一次 `scripts/check.sh`：

```
== go test ./... ==       20 个包全 ok（新增 internal/ai）
== 全平台编译 ==          ok darwin/arm64 · darwin/amd64 · windows/amd64
                          ok windows/arm64 · linux/amd64
== versions agree ==      ok 0.1.0
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

## 人工验收

登记在 [../ACCEPTANCE.md](../ACCEPTANCE.md)：A1 ~ A18，含打开库、增删改移、
拖入导入、粘贴、多标签、自动保存、工具栏、预览与同源高亮、滚动同步、
外部变更、重启恢复、三栏与分隔条、单实例、键位覆盖。
