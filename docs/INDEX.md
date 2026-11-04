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
| [design/themes.md](design/themes.md) | 系统多主题：明暗 × 配色方案的分工、五套方案、配置迁移与生效路径 | 加/改配色方案前 |
| [design/tree-follows-note.md](design/tree-follows-note.md) | 目录树跟随当前笔记：谁触发、展开+选中+滚动怎么落地、边界与取舍 | 动目录树定位前 |
| [design/wechat.md](design/wechat.md) | 公众号排版：样式菜单、手机宽度预览、行内样式的由来、样式目录与文件格式 | 动公众号排版前 |
| [design/obsidian-plugins.md](design/obsidian-plugins.md) | **Obsidian 插件兼容（P0 spike 已实现）**：机制调研、兼容边界、非信任 WebView 宿主、API 映射与分阶段路线 | 做插件兼容前 |
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
- **P2（R-10，AI）**：主体完成。Provider 抽象（Ollama native + OpenAI 兼容）、
  流式事件解析、对话条、会话持久化、AI 修订逐块接受、只读工具集与 agent loop、
  Skills、模型能力清单与探针、批量标签任务、改前快照与取回、模型列表克隆与排序、
  AI 旋钮（超时 / 重试 / 步数 / 上下文 / 输出语言）、AI 动作（命令面板 · 整篇与选区 ·
  一律走评审）。验收项 D1 ~ D16。
- 门禁：`scripts/check.sh` 全绿 —— 27 个包的单测通过，darwin/arm64、
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
  安装/下载次数与 release 详情。P1/P2/P3 仍待做，全部插件直接兼容不是目标，见
  [design/obsidian-plugins.md](design/obsidian-plugins.md)。

## 还剩什么

计划里的任务（T-1 ~ T-46）已全部完成。剩下的是**人眼验收**：见
[ACCEPTANCE.md](ACCEPTANCE.md) 的 E1 ~ E6（面板显隐、快照取回、模型克隆与排序、
模型的 Key 真的发出去、AI 旋钮、AI 动作）。

已知缺口与人工验收项一并登记在 [ACCEPTANCE.md](ACCEPTANCE.md)（末尾「已知缺口」）。
