# 文档索引

| 文档 | 内容 | 什么时候看 |
|---|---|---|
| [PROJECT.md](PROJECT.md) | **入口说明**：项目结构、启动步骤、规范要点 | 第一次接触项目 |
| [CONTRIBUTING.md](CONTRIBUTING.md) | 代码风格、目录约定、提交信息、分支策略、评审关注点 | 写代码、提交、开分支前 |
| [HARNESS.md](HARNESS.md) | 构建 / 运行 / 测试 / 调试流程、故障排查、CI | 日常开发与排障 |
| [ARCHITECTURE.md](ARCHITECTURE.md) | 分层、依赖方向、启动时序、线程模型、扩展点 | 加功能、决定代码放哪 |
| [ACCEPTANCE.md](ACCEPTANCE.md) | **人工验收项**：只能靠人眼/手感验的都在这里 | 交付前后逐条走 |
| [requirements/xink-port.md](requirements/xink-port.md) | 复刻 xink 的需求口径（R-1 ~ R-10，附 xink 证据） | 开工、验收 |
| [design/xink-port.md](design/xink-port.md) | 复刻的技术设计（底座核查、数据模型、流程、风险） | 动手前 |
| [exec-plans/xinkgo-port.md](exec-plans/xinkgo-port.md) | 研发计划与状态（T-1 ~ T-44） | 接着做、看进度 |
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
- **P2（R-10，AI）**：主体完成。Provider 抽象（Ollama native + OpenAI 兼容）、
  流式事件解析、对话条、会话持久化、AI 修订逐块接受、只读工具集与 agent loop、
  Skills、模型能力清单与探针、批量标签任务、改前快照与取回、模型列表克隆与排序。
  验收项 D1 ~ D16。
- 门禁：`scripts/check.sh` 全绿 —— 22 个包单测通过，darwin/arm64、
  darwin/amd64、windows/amd64、windows/arm64、linux/amd64 全部编译通过。

## 还剩什么

计划里的任务（T-1 ~ T-44）已全部完成。剩下的是**人眼验收**：见
[ACCEPTANCE.md](ACCEPTANCE.md) 的 E1 ~ E3（面板显隐、快照取回、模型克隆与排序）。

已知缺口与人工验收项一并登记在 [ACCEPTANCE.md](ACCEPTANCE.md)（末尾「已知缺口」）。
