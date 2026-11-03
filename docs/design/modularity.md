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

| 现在 | 将来 |
|---|---|
| `settings.go` + `settings_ai.go` + `settings_vault.go` + `settings_quick.go` | `internal/app/settings/`：`window.go`（骨架与各页注册）、`editor.go`、`preview.go`、`ai.go`、`vault.go`、`basic.go` |

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

阶段 1 之后 `internal/view` 的窗口部分：

| 文件 | 行数 | 装什么 |
|---|---|---|
| `workspace.go` | 456 | `Workspace` 类型、`View` 骨架、面板显隐与拖拽 |
| `preview.go` | 327 | 预览区、导出、链接 |
| `bars.go` | 316 | 标签栏、工具栏、状态行 |
| `metrics.go` | 287 | 几何：宽度、留白、换行列数、侧栏带 |
| `panes.go` | 225 | 树 / 编辑区 / 预览区的组装 |
| `modes.go` | 51 | 三种形态与切换 |
| **2** | 把 `metrics.go` 提成子包 `internal/view/layout`：导出一个 `layout.Metrics`，`Workspace` 持有一份 | 中 | 同上 + `pad_test.go` 迁过去 |
| **3** | `Workspace` 的状态分组：`mode`（三模式）+ `panels`（宽度）+ 其余交互状态各自成组，字段带前缀、不再是一张平表 | 中 | 同上 |
| **4** | `internal/app/settings/` 独立成包，按页分文件 | 中 | 设置窗口人工验收（ACCEPTANCE 里那几条） |
| **5** | `ai_bar.go`（756）按“消息列表 / 输入区 / 快捷词”拆；`file_tree.go` 拆出“行绘制 / 图标 / 选中与滚动” | 中 | 对话条与目录树人工验收 |

阶段 1 先行：它把后面每一步的“落点”先定下来，且不碰任何行为。

## 4 新约定（之后加代码按这个来）

1. **几何只写在 `metrics.go`（阶段 2 后是 `internal/view/layout`）**：任何
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
