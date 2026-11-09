# 文档索引

| 文档 | 内容 | 什么时候看 |
|---|---|---|
| [PROJECT.md](PROJECT.md) | **入口说明**：项目结构、启动步骤、规范要点 | 第一次接触项目 |
| [CONTRIBUTING.md](CONTRIBUTING.md) | 代码风格、目录约定、提交信息、分支策略、评审关注点 | 写代码、提交、开分支前 |
| [HARNESS.md](HARNESS.md) | 构建 / 运行 / 测试 / 调试流程、故障排查、CI | 日常开发与排障 |
| [ARCHITECTURE.md](ARCHITECTURE.md) | 分层、依赖方向、启动时序、线程模型、扩展点 | 加功能、决定代码放哪 |
| [ACCEPTANCE.md](ACCEPTANCE.md) | **人工验收项**：只能靠人眼/手感验的都在这里 | 交付前后逐条走 |
| [XINKGO_FEATURES.md](XINKGO_FEATURES.md) | xinkgo 已实现功能清单：逐项对照 xink，标出已实现 / 部分 / 未实现 | 看进度、找缺口 |
| [XINK_FEATURES.md](XINK_FEATURES.md) | Rust 版 xink 的已实现清单，上面那份的对照骨架 | 核对 xinkgo 与 xink 的差异 |
| [requirements/xink-port.md](requirements/xink-port.md) | 复刻 xink 的需求口径（R-1 ~ R-10，附 xink 证据） | 开工、验收 |
| [design/xink-port.md](design/xink-port.md) | 复刻的技术设计（底座核查、数据模型、流程、风险） | 动手前 |
| [design/ai-assist.md](design/ai-assist.md) | AI 编辑辅助：菜单/工具条入口 + 编辑区下方对话条（草案，待拍板） | 加 AI 入口前 |
| [design/ai-upgrade-qiaomu.md](design/ai-upgrade-qiaomu.md) | **AI 能力升级规划（v1 已拍板）**：对照 Qiaomu Agent 分析配置分层、Agent 辨析、选区与上下文；已定不做外部 CLI Agent / 联网搜索，按 P0 → P1 → P2 落地 | 改 AI 能力前 |
| [design/themes.md](design/themes.md) | 系统多主题：明暗 × 配色方案的分工、五套方案、配置迁移与生效路径 | 加/改配色方案前 |
| [design/tree-follows-note.md](design/tree-follows-note.md) | 目录树跟随当前笔记：谁触发、展开+选中+滚动怎么落地、边界与取舍 | 动目录树定位前 |
| [design/wechat.md](design/wechat.md) | 公众号排版：样式菜单、手机宽度预览、行内样式的由来、样式目录与文件格式 | 动公众号排版前 |
| [design/xinkgo-plugin-library.md](design/xinkgo-plugin-library.md) | **以 Xinkgo 架构为目标的插件体系与官方插件库（P0/P1 已落地）**：体系排查、三层模型、能力契约、库清单、运行时接入与官方库页、迁移路线 | 规划插件体系、收录插件前 |
| [design/obsidian-plugins.md](design/obsidian-plugins.md) | **Obsidian 插件兼容（P0 spike 已实现）**：机制调研、兼容边界、非信任 WebView 宿主、API 映射与分阶段路线 | 做插件兼容前 |
| [design/obsidian-core-plugins.md](design/obsidian-core-plugins.md) | **Obsidian 核心插件对照**：内置能力映射、下载榜前 20、Core 兼容插件与后续路线 | 规划核心插件前 |
| [design/daily-notes-templates.md](design/daily-notes-templates.md) | **Daily notes / Templates / Web viewer 内置核心插件（v1 已实现）**：三者功能与设置页、插件机制改造、WebView 安全边界、数据模型与实现偏差 | 改这三个核心插件前 |
| [design/text-files.md](design/text-files.md) | **全文本文件编辑与源码预览（v1 已实现）**：`internal/textfile` 判类、内容嗅探、BOM/编码、源码围栏、工具栏/预览/导出边界 | 加文本类型或改源码预览前 |
| [../examples/plugins/README.md](../examples/plugins/README.md) | **两个插件示例**：基础能力（C2）与全力支持能力全集（C3）的安装、授权、命令清单、视图入口，以及「不写进插件的能力」准入清单 | 写插件、验证插件机制前 |
| [plugin-demos.md](plugin-demos.md) | **插件示例使用手册**：要不要编译、两种安装路径的授权差异、逐条命令预期、日志位置与排错清单 | 安装或试用两个 demo 前 |
| [plugin-development.md](plugin-development.md) | **插件开发指南**：能力准入与黑名单、包结构与 manifest、CommonJS 加载链路、真实能力的 API 映射、CSS 与资源、调试发布；以两个 demo 逐段作案例 | 写插件前 |
| [design/modularity.md](design/modularity.md) | 模块化梳理：现状量出来的问题、`view`/`app` 的目标文件划分、分阶段计划与新约定 | 加功能前、觉得文件太大时 |
| [design/wechat-style-guide.md](design/wechat-style-guide.md) | **写公众号样式的指引**：键位表、CSS 能做与不能做、改 → 看 → 试、排错清单 | 写/改一套公众号样式时 |
| [exec-plans/xinkgo-port.md](exec-plans/xinkgo-port.md) | 研发计划与状态（T-1 ~ T-46） | 接着做、看进度 |
| [../README.md](../README.md) | 仓库首页：三步跑起来 + 文档入口 | — |
| [../AGENTS.md](../AGENTS.md) | AI / 新人的操作地图与技术铁律 | 交给 AI 任务时 |

外部参考：

- MyGo 仓库与文档：<https://github.com/egoist/mygo>（`docs/` 下有 getting-started、
  architecture、configuration、distribution、cli 等）
- MyGo 原生 UI：`ui` 包文档在上游仓库的 `docs/native.md` 与 `docs/ui/`

## 状态

- 骨架：完成（构建、运行、测试、Windows 打包均已验证）。
- **P0（R-1 ~ R-7）**：完成。人工验收项 [ACCEPTANCE.md](ACCEPTANCE.md) A1 ~ A18。
- **P1（R-8、R-9）**：完成。索引 / 双链补全 / 反链 / 标签 / 全文搜索 / 大纲 /
  命令面板 / 设置窗口 / 库管理 / 数据文件（JSON 校验与格式化）。验收项 B1 ~ B8、
  C1 ~ C9。
- **目录树跟随当前笔记**：完成（2026-11-02）。编辑区显示哪一篇，目录树就展开
  祖先目录、选中它并滚到它——点标签页、跟双链、命令面板、重启恢复都算。设计见
  [design/tree-follows-note.md](design/tree-follows-note.md)，验收项 G1。
- **多主题**：完成（2026-11-02）。`internal/theme` 五套方案（xink / ocean /
  moss / plum / sepia），明暗与方案两个设置项，设置里色卡即点即换，代码预设跟随。
  设计见 [design/themes.md](design/themes.md)，验收项 F1 ~ F4。
- **全文本文件编辑与源码预览**：完成（2026-11-09）。HTML/JS/CSS 等沿用 MyGo
  语言表高亮，Makefile 等无后缀文件按内容识别；非 Markdown 文本按源码围栏预览，
  BOM/UTF-16 与大小上限有保护。设计见
  [design/text-files.md](design/text-files.md)，验收项 C10 ~ C13。
- **P2（R-10，AI）**：主体完成。Provider 抽象（Ollama native + OpenAI 兼容）、
  流式事件解析、对话条、会话持久化、AI 修订逐块接受、只读工具集与 agent loop、
  Skills、模型能力清单与探针、批量标签任务、改前快照与取回、模型列表克隆与排序、
  AI 旋钮（超时 / 重试 / 步数 / 上下文 / 输出语言）、AI 动作（命令面板 · 整篇与选区 ·
  一律走评审）。验收项 D1 ~ D16。
- 门禁：`scripts/check.sh` 全绿 —— 全部包的单测通过，darwin/arm64、
  darwin/amd64、windows/amd64、windows/arm64、linux/amd64 全部编译通过。

- **编辑区右边距（可配置）**：完成。`config.editor.margin_right`，默认 0、范围
  0–120；在左右留白之外只给右边再加一段，换行按整字算。设置 ▸ 编辑区 ▸「右边距」。
  验收项 I1。
- **编辑区滚动条不再压正文**：完成。笔记用一份自己的主题，把滚动条的 thumb 设为
  透明，指针移上去才显形。验收项 I2。
- **面板之间的分割线**：完成。平时不画（两边透明），指针移到线上才显出来。验收项
  I3。
- **预览向左伸展（盖住编辑区）**：完成。预览面板左上角按钮；不落盘，编辑区宽度
  保留，再点还原。验收项 I5。
- **代码结构梳理**：完成。两轮、八个提交，全部行为不变：`workspace.go`（1601 行）
  按职责拆开、几何提成 `internal/view/geom` 并自带单测、`Workspace` 状态分组、
  设置窗口一页一文件、对话条与目录树各拆四份、`app.go` 按事情拆开；持久化职责写进
  ARCHITECTURE.md 第 7 节。设计、取舍与「不做什么」见
  [design/modularity.md](design/modularity.md)。
- **Obsidian 插件兼容**：P0 spike 已实现：插件发现/静态能力扫描、隔离宿主、
  命令与 Editor/Vault 子集、设置窗口里的插件分组、默认只看兼容插件的浏览器、
  安装/下载次数与 release 详情；兼容清单扩到 128 个，按下载量排序，其中
  C3/C4 明确标为降级、10 个常用插件标为 Core；Core 与 Other 分开管理，Core
  可启用/停用，并预留左侧 plugin ribbon；目录视图缓存、分页绘制和详情并发限制
  已做性能收敛。
  P1/P2/P3 仍待做，全部插件直接兼容不是目标，见
  [design/obsidian-plugins.md](design/obsidian-plugins.md)。
- **Daily notes / Templates / Web viewer 内置核心插件**：v1 已实现。内置核心
  注册表扩到 13 项，新增设置页/命令/ribbon/WebView 可用性与释放钩子声明；命令
  与 ribbon 由内置和第三方贡献**合并**发布、`appState` 统一分发；插件选项写进
  `plugins.json` 的 `options`，设置侧栏新增动态的「插件选项」分组（三页）。
  Web viewer 在**预览面板以标签页**打开（WebView 挂在主窗口），远程页非信任、
  独立会话、只放行 http(s)，含阅读视图、保存到库与内容拦截；模板清单为卡片式
  （图标、名称、路径/大小/时间、计数与空状态）；脚本与规则 `go:embed` 进二进制。
  设计、取舍与实现偏差见
  [design/daily-notes-templates.md](design/daily-notes-templates.md)，验收项
  K1 ~ K17。
- **AI 升级 P0 / P1**：已实现。P0 清账（四条场景路由、用量显示、快照按库归属、
  动作提示词变量、AI 设置分组、修过时文档），P1 主体（服务商分层与迁移、模型
  选择器 popover、最近模型、上下文 chip、自由指令改写选区、回复落地、Prompt 库
  的固定 / 分类 / 场景 / blanks、首屏引导）。外部 CLI Agent 与联网搜索按拍板
  不做。设计、取舍与实现偏差见
  [design/ai-upgrade-qiaomu.md](design/ai-upgrade-qiaomu.md)，验收项 L1 ~ L6、
  M1 ~ M8。
- **AI 升级 P2**：已实现。问题卡（`ask_user` 工具 + `EventQuestion`）、工作时间线
  折叠与用量条、编辑重发与会话列表、`@笔记` 引用、按轮撤销（撤销前检查文件未变）、
  文件权限两档（只读 / 允许追加）。工具集仍全只读，评审写回仍一律过 diff。验收项
  N1 ~ N8；偏差与「不做计划清单 / `@文件夹` / 多文件回合追踪」的理由见
  [design/ai-upgrade-qiaomu.md](design/ai-upgrade-qiaomu.md) §7 P2。

## 还剩什么

计划里的任务（T-1 ~ T-46）已全部完成。剩下的是**人眼验收**：见
[ACCEPTANCE.md](ACCEPTANCE.md) 的 E1 ~ E6（面板显隐、快照取回、模型克隆与排序、
模型的 Key 真的发出去、AI 旋钮、AI 动作）。

已知缺口与人工验收项一并登记在 [ACCEPTANCE.md](ACCEPTANCE.md)（末尾「已知缺口」）。
