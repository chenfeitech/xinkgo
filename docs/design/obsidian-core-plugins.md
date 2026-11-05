# Obsidian 核心插件对照与 Xinkgo 核心插件

日期：2026-11-05。状态：**设计梳理、浏览器 Core 标记、内置核心注册表与开关
已落地**。本文回答两件事：Obsidian 的「核心插件」在 Xinkgo 里对应什么；哪些最
常用的社区插件值得当作 Xinkgo 的核心兼容插件优先维护。

> 2026-11-05：此前「核心插件」只指社区热门插件的 Core 标记。本轮把它分成三层：
> **内置核心能力**（Xinkgo 自己实现、统一开关、写 `plugins.json`，见 §4）、
> **核心兼容插件**（社区热门、打 Core 标记、仍按 C3/C4 降级，§2）、**其他插件**。
> 下表的「已内置」项现在都是可开关的内置核心插件。

这里把「核心插件」分成两层，避免把内置功能和外部插件混为一谈：

1. **内置核心能力**：Xinkgo 自己实现，不依赖插件宿主，也不进 `plugins.json`。
2. **核心兼容插件**：来自社区目录，经过静态扫描后放进兼容清单，浏览器里打
   `Core` 标记；用户仍需安装，宿主仍按能力授权，不自动启用。

## 1 Obsidian 核心插件对照

| Obsidian 核心插件 | Xinkgo 对应 | 状态 | 说明 |
|---|---|---|---|
| File explorer | 文件树 | 已内置 | 全量扫描、目录树跟随当前笔记、右键菜单、外部变更刷新 |
| Quick switcher | 命令面板 | 已内置 | `Cmd+P`，模糊匹配笔记、Skills、AI 动作和插件命令 |
| Global search | 全文搜索 | 已内置 | 防抖、后台读盘、点击跳行 |
| Backlinks | 反链面板 | 已内置 | 索引驱动，右侧抽屉 |
| Outgoing links | 双链解析与补全 | 部分内置 | 能解析、补全、点击，但没有独立「出链」面板 |
| Tag pane | 标签面板 | 已内置 | 聚合、展开、frontmatter 标签 |
| Outline | 大纲面板 | 已内置 | 实时解析，点击跳行 |
| Command palette | 命令面板 | 已内置 | 与 Quick switcher 共用入口 |
| Editor status bar | 底部状态行 | 部分内置 | 有状态提示，但没有字数、行数、光标位置等常驻统计 |
| Format converter | Markdown 工具栏 | 部分内置 | 14 个格式按钮；没有 Obsidian 的完整转换命令集 |
| Page preview | 无 | 未内置 | 悬停预览双链尚未实现 |
| Daily notes | 无 | 未内置 | 日期笔记、模板、导航均未实现 |
| Templates | 无 | 未内置 | 可考虑复用 AI 动作与 Skills，但不是模板系统 |
| Note composer | 无 | 未内置 | 合并/拆分笔记尚未实现 |
| Slash commands | 无 | 未内置 | 编辑器不是 CodeMirror 6 |
| Bookmarks | 无 | 未内置 | 无书签数据模型与面板 |
| Canvas | 无 | 未内置 | 原生 UI 没有画布场景 |
| Graph view | 无 | 未内置 | 索引有双链数据，但没有图视图 |
| Word count | 无 | 未内置 | 状态栏未做统计 |
| Sync | 无 | 未内置 | 无官方同步；外部云同步由社区插件承担 |
| Bases / Properties | 无 | 未内置 | 无数据库式属性查询 |

建议按收益排序补内置能力：字数/光标统计、最近文件、书签、Daily notes、
Templates、Page preview、Graph view。前四个是轻量原生 UI；后三个需要新的数据模型
或 DOM 表面，放到 P2/P3。

## 2 最常用的社区插件

数据来自官方 `community-plugin-stats.json` 的 `downloads` 总量。浏览器已按该值
降序排列；下表是前 20 个，其中 10 个已在兼容清单里并标记为 Core，另外 10 个
因 ESM、缺少默认导出或静态扫描未通过暂列「待补」，不会伪装成可安装。

| 插件 ID | 下载量 | 名称 | Xinkgo 处理 |
|---|---:|---|---|
| `obsidian-excalidraw-plugin` | 8,893,111 | Excalidraw | Core，C4 降级兼容 |
| `templater-obsidian` | 6,095,437 | Templater | 待补；优先补模板类内置能力 |
| `dataview` | 5,256,385 | Dataview | 待补；需要查询语言和结果视图 |
| `obsidian-tasks-plugin` | 4,563,879 | Tasks | Core，C4 降级兼容 |
| `obsidian-git` | 3,398,865 | Git | Core，C4 降级兼容 |
| `table-editor-obsidian` | 3,338,242 | Advanced Tables | Core，C4 降级兼容 |
| `calendar` | 3,245,094 | Calendar | 待补；与 Daily notes 一起规划 |
| `copilot` | 3,212,173 | Copilot | 待补；与 Xinkgo 内置 AI 能力重叠，优先复用 |
| `obsidian-style-settings` | 2,823,021 | Style Settings | 待补；需要主题变量模型 |
| `obsidian-kanban` | 2,811,211 | Kanban | 待补；需要 DOM 看板表面 |
| `realclaudian` | 2,613,196 | Claudian | 待补；与内置 AI 能力重叠 |
| `tasknotes` | 2,566,270 | TaskNotes | 待补；与 Tasks/Daily notes 一起规划 |
| `remotely-save` | 2,355,287 | Remotely Save | 待补；需要网络与凭据授权 |
| `quickadd` | 2,302,138 | QuickAdd | Core，C4 降级兼容 |
| `obsidian-icon-folder` | 2,293,044 | Iconize | Core，C3 降级兼容 |
| `editing-toolbar` | 2,127,523 | Editing Toolbar | Core，C4 降级兼容 |
| `omnisearch` | 2,099,360 | Omnisearch | Core，C4 降级兼容 |
| `obsidian-importer` | 1,880,688 | Importer | 待补；需要各来源解析器 |
| `obsidian-minimal-settings` | 1,876,624 | Minimal Theme Settings | Core，C3 降级兼容 |
| `obsidian-outliner` | 1,478,525 | Outliner | Core，C4 降级兼容 |

Core 不是「保证全部功能可用」：它表示这个插件在维护优先级、兼容清单和浏览器展示上
被当作一等公民，仍按 C3/C4 规则显示降级标记和能力授权。待补项需要继续扩展 shim、
补内置能力或等待 DOM 表面，不能只靠放宽扫描器塞进清单。

## 3 已落地的界面约定

- 浏览器和已安装列表都按官方 `downloads` 降序；没有统计数据的排在后面，按名称排序。
- 插件管理分成「内置核心」「核心插件」「已安装」「浏览」四页。「内置核心」是
  Xinkgo 自带的原生能力（§4）；「核心插件」仍是 10 个社区热门兼容插件的 Core
  标记页；已安装页与浏览页排除 Core 插件，避免同一插件重复出现。
- 核心兼容插件安装后用 `Enabled` 复选框控制启用/停用，状态写进 `plugins.json`；
  未安装时先显示 Install 和「安装后可启用」。
- 内置核心能力也写进 `plugins.json` 的 `core` 字段，在「内置核心」页用复选框开关
  （§4）；这是 100% 可用、有界面、零兼容风险的真核心插件。
- 性能：近万条社区目录只做一次注解和下载量排序并缓存；Core、Other、搜索和详情
  候选都从这份缓存视图过滤。浏览页默认画 40 行，用「显示更多」每次加 40，
  上限 200；release 详情最多拉取 20 个插件，并用信号量把并发压到 4。
- 浏览页默认隐藏 C3/C4 降级插件，勾「显示降级插件」才列出并附说明；Core 页同样
  注明降级，不再把「能加载」冒充「能用」。
- 目录树左侧预留固定 34 DIP 的 plugin ribbon。`addRibbonIcon` 注册到这里，不再混入
  命令面板；点击图标通过 `action` 事件回到插件回调。「插件 ribbon」本身也是可开关
  的内置核心能力。
- Core 插件在核心插件页显示 `Core` 徽标；C3/C4 仍显示「受限/降级」徽标。

## 4 内置核心插件注册表（2026-11-05 落地）

`internal/plugin/core.go` 列出 10 项内置核心能力，ID 稳定（会写进
`plugins.json`），每项有默认开启值：

| ID | 对应界面 | 关闭后 |
|---|---|---|
| `file-explorer` | 左侧文件树 | 不画侧栏（保留宽度，重开恢复） |
| `command-palette` | 命令面板 | 不画面板 |
| `global-search` | 全局搜索面板 | 不画搜索面板 |
| `page-preview` | 右侧预览 | 不画预览（保留宽度） |
| `outline` / `backlinks` / `tag-pane` | 右侧 rail 的三个抽屉按钮 | 按钮消失，已开的抽屉收起 |
| `status-bar` | 底部状态行 | 整行隐藏 |
| `word-count` | 状态行右侧的字数 | 隐藏字数（本轮补的轻量原生能力，CJK 按字计） |
| `plugin-ribbon` | 文件树左侧插件窄条 | 窄条隐藏 |

开关链路：设置页 `builtinView` → `pluginHost.setCore`（写 `plugins.json`）→
`applyCoreFeatures`（`plugins.json` 的 `core` 转成 `map[string]bool`）→
`view.Workspace.CoreFeatures` → 各绘制点的 `w.coreOn(id)` 门控。缺字段按默认，
显式 `false` 保留，与兼容插件的 `enabled`/`grants` 同一文件、同一套读写。
不加载任何 WebView，不受降级兼容影响。

## 5 参考

- Obsidian 核心插件清单：<https://help.obsidian.md/Plugins>
- 官方社区目录：<https://github.com/obsidianmd/obsidian-releases/blob/master/community-plugins.json>
- 官方下载统计：<https://github.com/obsidianmd/obsidian-releases/blob/master/community-plugin-stats.json>
- Xinkgo 功能清单：[../XINKGO_FEATURES.md](../XINKGO_FEATURES.md)
- 插件宿主设计：[obsidian-plugins.md](obsidian-plugins.md)
