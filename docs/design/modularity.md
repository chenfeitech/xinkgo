# 模块化梳理与设计

目标：让 `internal/view` 与 `internal/app` 从“几个大文件”变成“按职责分块、
边界清楚、加功能知道往哪放”。本文档只讲**怎么分**与**按什么顺序做**，不动
外部行为；每一步都可单独提交、单独验证。

## 1 现状（量出来的，不是感觉）

按包统计（不含测试与 `catalog_gen.go` 之外的生成代码）：

| 包 | 行数 | 说明 |
|---|---|---|
| `internal/view` | 11.0k | 界面：窗口、面板、对话条、树、标签页、大纲、命令面板 |
| `internal/app` | 7.8k | 生命周期、菜单、设置窗口、AI 编排、导出、公众号 |
| `internal/locale` | 5.7k | 其中 5.6k 是生成物 |
| `internal/ai` | 4.9k | 模型与工具 |

最大的几个文件：

| 文件 | 行数 | 性质 |
|---|---|---|
| `internal/view/workspace.go` | 1601 | **窗口上帝文件**：类型字段 185 行，几何、面板、工具栏、预览、链接全在里面 |
| `internal/app/settings_ai.go` | 1069 | 设置窗口的 AI 一页 |
| `internal/app/settings.go` | 914 | 设置窗口的骨架与其余各页 |
| `internal/view/ai_bar.go` | 756 | 编辑区下方的对话条 |
| `internal/app/app.go` | 661 | 启动装配 |
| `internal/view/file_tree.go` | 641 | 目录树 |

依赖方向是干净的（`view` 不依赖 `app`，已用 `go list` 复核），所以问题不在
分层，而在**包内的组织**：

1. **一个文件装一个窗口的全部**：`workspace.go` 里同时有纯几何（padding /
   margin / 栏宽 / 换行列数 / 侧栏留白）、状态变更（面板显隐、拖拽、三种
   “占满/伸展”模式）、绘制（面板、工具栏、标签栏、按钮）和副作用（开链接）。
   改一个边距要在 1600 行里找位置。
2. **几何散在绘制代码里**：`editorWidth`、`railClear`、`wrapColumn` 这些纯
   计算被 `toolbar`、`aiButton` 夹在中间，无法单独看、也无法整体测。
3. **`Workspace` 一个结构扛所有状态**：面板宽度、三种模式、滚动同步、命令
   面板、搜索、补全、对话、评审……185 行字段，谁都能改谁。
4. **设置窗口按“文件”而不是按“页”组织**：`settings.go`（骨架+多页）与
   `settings_ai.go`（一页就 1069 行）。

## 2 目标结构

`internal/view` 按“**一块界面一个文件，几何单独一处**”来放：

| 文件 | 装什么 |
|---|---|
| `workspace.go` | `Workspace` 类型与构造、`View` 的骨架、面板显隐与拖拽（窗口级） |
| `metrics.go` | **纯几何**：面板/编辑区/预览区宽度、padding、margin、换行列数、留白、侧栏带。只依赖 `config` 与 `state`，不画东西 |
| `panes.go` | 三块面板的组装：目录树、编辑区（含数据文件栏）、预览区 |
| `preview.go` | 预览区自己：正文、页眉页脚按钮、导出、跳转链接 |
| `bars.go` | 标签栏与工具栏：标签、Markdown 按钮、三个“占满/伸展”按钮、AI 入口 |
| `modes.go` | `zoom`（整窗）/ `stretched`（向右伸展）/ `spread`（预览铺开）三个模式的类型与切换 |

`internal/app` 按“**一个窗口一个目录**”：

| 现在（阶段 4 之后） | 行数 |
|---|---|
| `settings.go` | 602 — 窗口、导航、页签、状态读写、共用控件（toggle / slider / choice / field / group） |
| `settings_ai.go` | 1069 — AI 一页（模型、技能、Ollama） |
| `settings_vault.go` | 226 — 库管理一页 |
| `settings_quick.go` | 265 — 快捷动作一页 |
| `settings_preview.go` | 130 — 预览一页 |
| `settings_appearance.go` | 97 — 外观与配色一页 |
| `settings_config.go` | 93 — 配置文件的导入导出 |
| `settings_editor.go` | 39 — 编辑区一页 |

**为什么不做成独立包**（原计划的 `internal/app/settings/`，已放弃）：设置窗口的
每个方法都挂在 `*appState` 上，用到库、索引、模型目录、重启等十余个应用级动作。
要独立成包就得先抽出一层 `Host` 接口（设置窗口能做的十几件事），那是另一次成
规模的设计与搬运，收益是“边界写死”，代价是一次动 2500 行用户可见的界面代码。
按页分文件已经解决了“找不到改哪”的问题，包边界留到真有第二个窗口需要它时再
做，并把 `Host` 的形状先写在这里：

```go
// 将来若要独立成包，设置窗口只需要应用给这些：
type Host interface {
    Config() *config.AppConfig     // 读
    Apply(config.AppConfig)        // 写并生效
    Restart(note string)           // 有些改动要重启
    Vaults() []vault.VaultEntry    // 库管理
    KeepVault(path string) error
    LookForOllamaModels()          // 拉模型列表
    ExportConfig() (path string, err error)
    ImportConfig(path string) error
}
```

不变的东西（写死，别动）：

- 依赖方向仍是 `cmd → app → {view, config, …}`，`view` 不许 import `app`。
- 界面文案仍进 `internal/locale`，键名不变则行为不变。
- 几何与绘制分离：几何函数不接 `*ui.Context` 做绘制，只算数；绘制函数不自己
  算几何，只调 `metrics.go`。
- 一个文件的职责说不完（超过约 400 行）就是拆的信号。

## 3 分阶段

每一步都可独立提交、独立回退，都以 `scripts/check.sh` 全绿为验收。

| 阶段 | 内容 | 风险 | 验证 |
|---|---|---|---|
| **1** | ✅ 已完成：`workspace.go` 拆成 `metrics.go` / `panes.go` / `preview.go` / `bars.go` / `modes.go`（纯搬运，零行为变更） | 低 | 构建 + 全量测试 + 声明清单比对（搬运前后 64 个 func、`const`/`var`/`type` 一个不差） |

阶段 3 之后 `Workspace` 的字段不再是 185 行平表：

| 组 | 装的什么 |
|---|---|
| `w.mode` | `Zoom` / `Stretched` / `Stretch` / `Drawer`：窗口的形状 |
| `w.scroll` | `scrollFollow`：两侧滚动互跟的全部记账（锚点表、手势、上次位置） |
| 其余 | 面板宽度 `Layout`、命令面板、搜索、补全、对话、评审，各自紧挨注释 |

还留着没分的：`Layout`（面板宽度，单独一个 `state.Layout` 值，分组收益小）、命令面板 / 补全 / 大纲 / 快捷词的散字段（各 2–4 个，已有注释）—— 真要动，等它们再长。

阶段 2 之后：

| 文件 | 行数 | 装什么 |
|---|---|---|
| `internal/view/geom/geom.go` | 335 | **几何**：面板/编辑区/预览区宽度、留白、边距、换行列数、侧栏带、三种形态的枚举。不 import `view`，只依赖 `config` / `state` / `ui`，可脱离窗口单测 |
| `internal/view/geom/geom_test.go` | 145 | 几何单测：不渲染、不建窗口 |
| `internal/view/workspace.go` | 472 | `Workspace` 类型、`View` 骨架、面板显隐与拖拽 |

阶段 1 之后 `internal/view` 的窗口部分：

| 文件 | 行数 | 装什么 |
|---|---|---|
| `workspace.go` | 456 | `Workspace` 类型、`View` 骨架、面板显隐与拖拽 |
| `preview.go` | 327 | 预览区、导出、链接 |
| `bars.go` | 316 | 标签栏、工具栏、状态行 |
| `metrics.go` | 287 | 几何：宽度、留白、换行列数、侧栏带 |
| `panes.go` | 225 | 树 / 编辑区 / 预览区的组装 |
| `modes.go` | 51 | 三种形态与切换 |
| **2** | ✅ 已完成：几何提成子包 `internal/view/geom`，导出 `geom.Geometry` 与 `geom.Zoom`；`Workspace` 每次用时现取 `w.geometry()`，不做缓存（避免过期） | 中 | 同上 + `geom` 自有单测（不渲染窗口即可验几何） |
| **3** | ✅ 已完成（一半）：`Workspace` 的状态分组 —— `mode`（三形态 + 抽屉）与 `scroll`（滚动同步的 12 个记账字段）各自成组，类型 `scrollFollow` 跟 `scroll.go` 走 | 中 | 同上 |
| **4** | ✅ 已完成（改了做法）：设置窗口按页分文件（`settings_appearance` / `_editor` / `_preview` / `_config`），**不再独立成包** —— 理由见下 | 低 | 构建 + 全量测试 + 声明清单比对（43 个 func 一个不差）+ 设置窗口人工验收 |
| **5** | ✅ 已完成：`ai_bar.go`（756）拆成 bar / session / talk / ask 四个文件；`file_tree.go`（641）拆成 tree / rows / vault / paths 四个文件 | 低 | 构建 + 全量测试 + 声明清单比对（32 / 31 个 func 一个不差）+ 对话条与目录树人工验收 |

阶段 5 之后，两个大控件按“由哪几部分组成”拆开：

| 文件 | 行数 | 装什么 |
|---|---|---|
| `ai_bar.go` | 307 | 对话条本身：开合、页头、输入区、模型选择 |
| `ai_ask.go` | 220 | 问：发出什么、回来的片段、结束后怎么处理 |
| `ai_talk.go` | 154 | 说了什么：一轮一轮的呈现 |
| `ai_session.go` | 110 | 一场对话的数据与落盘 |
| `file_tree.go` | 289 | 树的数据与面板 |
| `tree_rows.go` | 197 | 行：怎么画、怎么拖、右键菜单 |
| `tree_paths.go` | 116 | 路径：定位、选中、滚到 |
| `tree_vault.go` | 89 | 库切换 |

五个阶段做完，`internal/view` 与 `internal/app` 里已没有超过 1100 行的文件
（`settings_ai.go` 1069 是最大的一个，且它是单独一页）。

阶段 1 先行：它把后面每一步的“落点”先定下来，且不碰任何行为。

## 4 新约定（之后加代码按这个来）

1. **几何只写在 `metrics.go`（阶段 2 后是 `internal/view/geom`）**：任何
   “多少 DIPs”的常量与计算都去那里，绘制代码只叫名字。
2. **一块界面一个文件**：新增面板/对话条/抽屉，新建文件，不往 `workspace.go`
   里塞。
3. **模式只有一个入口**：新增“占满/伸展”这类形态，在 `modes.go` 加值与切换，
   不另起字段。
4. **状态分组**：新状态进对应的组（`mode` / `panels` / 交互），不直接拍在
   `Workspace` 上。
5. **超过 400 行就拆**，拆不动就写注释说明为什么。

## 5 不做的事

- 不改依赖方向，不把 `view` 拆成多个顶层包（现阶段子包够用，多顶层包只会
  让 import 表变长）。
- 不改任何用户可见行为：边距数值、文案、快捷键一律不动。
- 不顺手“重构”看不顺眼的地方：每阶段只做表上那一件事。
