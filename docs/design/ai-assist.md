# 设计：AI 编辑辅助（菜单入口 + 编辑区下方对话条）

> 状态：**已拍板**（2026-11-02），按 §10 的结论实现：只做窗口工具条末尾的
> 图标按钮，不做菜单，动作仍在命令面板，对话条「开着」不持久化。
> 参考：xink `src/ui/editor.rs:1667-1690`（编辑器下方对话条）、需求 R-10.3、
> 验收 D1（对话条开关）与 D9（AI 修订评审）。

## 1 背景与目标

用户口径：打开一个 md 文件之后，菜单栏有「AI 辅助」按钮，点了在**编辑区下方**
弹出 AI 对话框。

- **目标**：给 AI 编辑辅助一个**看得见**的入口，并把「改笔记」这条链路走通——
  提问 → 流式回答 → **逐块评审** → 写入 → 改前快照与改后校验。
- **非目标**（本轮不做）：
  - 不动 Provider / agent loop / 工具 / Skills / 模型能力（`internal/ai` 现状保留）。
  - 不改任何落盘格式（`ai_sessions.json` 等仍与 xink 兼容）。
  - 不做独立侧边 AI 面板（xink 也是编辑区下方，R-10.3）。
  - 不新增设置项（除非 §10 拍板要）。

## 2 现状盘点

能力大部分已在仓库里：

| 能力 | 现在在哪 | 证据 |
|---|---|---|
| 菜单入口 | File ▸ **Ask AI**，快捷键 `cmd-shift-a` | `internal/app/menu.go`（File 子菜单）、`internal/keymap/keymap.go` `ToggleAiChat`（标签「AI 对话条」） |
| 开关 | `Workspace.ToggleAI()`：翻 `aiSession.Open`，打开时聚焦输入框；没有笔记打开时直接返回 | `internal/view/ai_bar.go` |
| 面板位置 | `center()` 一列里：`tabBar → toolbar → centerBody（编辑器/评审）→ aiBar` | `internal/view/workspace.go` |
| 面板内容 | 头（AI / using skill / answering… / Forget / ×）、对话区（`Scroll`，上限 260，`markdown.Stream` 逐字）、输入区（输入框 + Ask / Stop） | `internal/view/ai_bar.go` |
| 会话归属与持久化 | 按**笔记路径**归属，关闭只隐藏、历史保留；`ai_sessions.json` 原子写 | `ai_bar.go`、`internal/state` |
| 未配置 AI | 发送后回答块直接落一句提示，不转圈等待（D3） | `aiSend` 里 `OnAsk == nil` / `ErrNotConfigured` 分支 |
| 编辑辅助 | 六个内置动作（Polish / Condense / Translate / Retitle / Summarize / Tag），`pieceOf` 决定选区还是整篇，`runAction` 发送并进入评审 | `internal/config/ai.go` `AiAction`、`ai_bar.go` |
| 评审 | `reviewState` + `Take all / Send all back / Write it in / Away`，悬停逐块 Take / Revert | `internal/view/review.go` |
| 写入后 | 改前快照轮转 + 双链等校验提示（只提示不阻断） | `OnSnapshot` / `OnVerify`，`internal/app/app.go` |

**缺口**（本次要补的）：

1. 入口只有一个藏在 File 菜单里的英文项，**窗口内没有任何按钮**。
2. 六个编辑辅助动作只在命令面板（Cmd+P，前缀 `Action ·`）里，用户不知道有
   「润色 / 精简 / 翻译」。
3. 术语不一致：菜单 `Ask AI`、keymap 标签「AI 对话条」、面板头 `AI`。
4. 没有笔记打开时点菜单项**毫无反馈**（`ToggleAI` 静默返回）。
5. 对话条「开着」这件事不跨笔记、也不跨重启（`aiOf` 每次新建，Open 不落盘）。

## 3 交互设计

### 3.1 入口 A：窗口工具条「AI 辅助」按钮（推荐为主入口）

- **位置**：Markdown 工具条末尾、一条分隔线之后，图标 + 文字，Tooltip
  「打开 AI 对话条（Cmd-Shift-A）」。
- **出现条件**：有笔记打开**且**是 Markdown。数据文件走 `dataBar` 分支时不出现，
  没有笔记时整个工具条本来就不画。
- **行为**：点击 = `ToggleAI`；对话条已开时按钮呈选中态（与大纲等按钮一致），
  再点收起。
- **理由**：用户口径是「打开 md 之后有个按钮」。工具条正贴着编辑区，与编辑动作
  同处一处；macOS 的菜单栏在屏幕顶部，离编辑区太远，只放菜单等于把功能藏起来。

### 3.2 菜单：**不做**

File ▸ Ask AI（Cmd-Shift-A）保持原样，不新增顶层菜单，也不往菜单里加编辑辅助
动作——动作继续只在命令面板（Cmd+P，前缀 `Action ·`）里。理由：入口有一个就够，
工具条上的按钮与编辑动作同处一处，比菜单近得多。

### 3.3 面板（编辑区下方）

位置与结构**完全不动**：对话区上限 260、打开即聚焦输入框、未配置时回答块直接给
提示。不加 Esc 收起，不加整条对话条的高度上限——现状的 260 已经够了。

### 3.4 编辑辅助的口径（本轮明确）

- **范围**：有选区 → 改选区；无选区 → 光标所在段落；动作自身声明
  `Scope = whole` → 整篇（`pieceOf` 已是这个顺序）。
- **结果一律进评审**，不直接写进笔记；`config.AiConfig.AutoApply` 为 true 时例外
  （默认 false，保持现状）。
- **写入后**：改前快照 + 校验提示（现状，D18 / D19）。

## 4 分层与改动点

| 层 | 文件 | 改动 |
|---|---|---|
| view | `internal/view/workspace.go` | `toolbar()` 末尾加分隔线 + 「AI 辅助」按钮；非 md / 无笔记时不画 |
| view | `internal/view/ai_bar.go` | `ToggleAI` 已有；补 `aiOpen(path)` 给按钮取选中态（每帧读，不新建会话）、`aiBand` |
| view | `internal/view/icons.go` | 新增 `iconAi`（Lucide sparkles），与工具条其它按钮同一套画法 |
| app / config / keymap | — | **不动**：菜单、设置、快捷键、落盘格式都不改 |

铁律：依赖方向单向，`view` 不得 import `app`；入口只是 `view` 内部的一个按钮，
调 `ws` 自己的 `ToggleAI`，与工具条上其它按钮一样。

## 5 状态机

```
hidden ──点击按钮/菜单/Cmd-Shift-A──▶ open（聚焦输入框）
open ──发送──▶ busy（按钮变 Stop）──答完──▶ over（块下方 Look it over）
over ──Look it over──▶ review（编辑区换成 diff：Take / Revert / Take all / Write it in / Away）
review ──Write it in──▶ 写笔记 + 快照 + 校验提示 ──▶ open
open/hidden：关闭只隐藏，历史按笔记保留
未配置 AI：面板照开，发送后回答块直接给提示，不转圈
无笔记：按钮不出现，菜单项置灰
```

## 6 数据模型

- **不改任何落盘格式**。
- 「对话条开着」**不持久化**：与 xink 一致，关闭只隐藏、历史按笔记保留，开着与
  否是这一次的会话的事。落盘的 `ai_sessions.json` 会话历史不受影响。

## 7 关键流程

1. 打开 md → 工具条末尾出现「AI 辅助」。
2. 点击 → `ToggleAI` → `Open = true`、`focus = true` → `center` 列在 `centerBody`
   之后画出对话条。
3. 输入回车 → `aiSend`：用户的话立刻上屏 → goroutine 收事件 → 回主线程写块 →
   `markdown.Stream` 逐字浮现。
4. 答完 → 块下方出现 `Look it over` → `StartReview` → 编辑区换 diff。
5. `Write it in` → 写笔记 → 快照 + 校验提示 → 回到编辑区。

## 8 风险与对策

| 风险 | 对策 |
|---|---|
| 工具条溢出：`ui.Toolbar` 有 `More` 折叠，`MaxWidth` 是 `editorWidth`，窄窗口下 AI 按钮会被折进 More | 按钮放在工具条行内最右并给固定宽度；实测窄窗口（<900）确认没被折叠，必要时单独一行右对齐 |
| 菜单在启动时构建，动作列表改了菜单不更新 | 先确认 mygo 菜单是否支持每次打开时求值；不支持则退化为「AI 动作」子项打开命令面板并预置前缀 |
| AI 未配置 | 不置灰按钮，点了给一句能行动的话并指向设置（与 D3 一致） |
| 主线程 | AI 请求继续放 goroutine，结果回主线程更新（现状，不动） |
| 语言不一致（面板英文 / 工具条中文） | 见 §10-3 |
| Windows / Linux 也要有这个菜单，不能与 macOS 位置重复 | 顶层菜单在各平台都显示；不要同时在 File 下再留一份 |

## 9 测试计划

- **view**（`toolbar_test.go`）：按钮站在 Markdown 按钮行之后、且在窗口内——
  220 / 320 / 520 / 1100 四种宽度都验（窄窗口下它不能折进行尾的 More）；
  按钮是图标不是文字（宽度不超过三个图标）；点击 → 会话打开，再点 → 收起；
  数据文件 → 不出现。
- **回归**：`toolbar_test.go` 原有四条（行宽、图标、图画各异、Tooltip）；
  `ai_bar_test.go` / `action_test.go` / `review_test.go` 全绿；
  `scripts/check.sh` 通过（含 darwin/arm64、darwin/amd64、windows/amd64 编译）。
- **人工验收**（新增 F 系列，与 D 系列并列）：
  - F1 打开 md，工具条末尾出现 AI 按钮（图标），悬停出文字提示；窗口拖窄时它
    不消失，也不跑出窗口。
  - F2 点击 → 编辑区下方出现对话条并自动聚焦；再点收起，历史还在（D1）。
  - F3 没打开笔记 / 打开的是 JSON 之类的数据文件时，工具条上没有这个按钮。
  - F4 未配置 AI 时点它 → 对话条照开，第一次提问给出「还没配置」的提示，不转圈
    （D3）。
  - F5 选中一段 → 命令面板跑「Polish」→ 进评审 → `Write it in` 后只有选区被改
    （D9，动作入口未变，顺带回归）。

## 10 拍板结论（2026-11-02）

| # | 问题 | 结论 |
|---|---|---|
| 1 | 按钮位置 | **工具条末尾**，只做按钮 |
| 2 | 菜单形态 | **不做**（保留 File ▸ Ask AI 原样，不新增菜单项） |
| 3 | 按钮长相 | **图标 + 文字 Tooltip**，与 Markdown 编辑按钮一致，不显示文字标签 |
| 4 | 六个动作是否进菜单 | **不进**，继续只在命令面板 |
| 5 | 对话条开着是否持久化 | **不持久化** |
| 6 | Esc 收起 | **不加** |
| 7 | 未配置 AI 时置灰 | **不置灰**，点了照常打开，提示由对话条给（D3） |

## 11 实现落点

- `internal/view/icons.go`：新增 `iconAi`（Lucide sparkles）。
- `internal/view/workspace.go`：`toolbar()` 外包一层 `ui.Row`，Markdown 按钮行的
  宽度减去 `aiBand`，行末加分隔线与 `aiButton`。
- `internal/view/ai_bar.go`：`aiBand`、`aiOpen(path)`（每帧读，不新建会话），
  `ToggleAI` 保持不变。
- 按钮放在 `ui.Toolbar` **外面**：工具条放不下的按钮会折进末尾的 More 菜单，
  默认布局下它就是折叠的，AI 入口不能被折进去。
