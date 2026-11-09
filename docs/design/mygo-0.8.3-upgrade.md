# MyGo 0.7.4 → 0.8.3 升级评估

日期：2026-11-09。状态：**可以升级，已在 `feature/mygo-0.8.3` 完成升级并通过全部静态门禁，待真机验收与合并**。

## 1 结论先行

**可以升级，且不需要改一行业务代码。** `go.mod` 只从 `v0.7.4` 改到
`v0.8.3`，`go mod tidy` 后 `go.sum` 只换了两行；`CGO_ENABLED=0 go build ./...`、
`go test ./...`、`make check`（五平台编译 / 版本一致 / 资源门禁）、
`make package`（macOS 双架构 + Windows 双架构）全部通过。

**但 0.8.3 没有解决 Markdown 行内样式扩展的问题**，详见 §4。上一轮
Markdown 行内样式扩展调研（在 `feature/markdown-style-research` 分支）的结论仍然成立。

## 2 版本与依赖

| 项 | 0.7.4 | 0.8.3 |
| --- | --- | --- |
| Go 版本 | 1.27.3 | 1.27.3（一致） |
| 直接依赖 | `mygo v0.7.4` | `mygo v0.8.3` |
| 间接依赖 | purego / typesetting / x-image / x-net / x-text | **完全相同**，`go mod tidy` 未新增 |

0.8.3 自身新增了 `go-mp3` / `oggvorbis` / `vorbis`（音频解码），但 Xinkgo 没有
import `mygo/audio`，Go 的模块裁剪没有把它们拉进来，`CGO_ENABLED=0` 下编译通过，
符合 AGENTS.md 的禁 cgo 约束。

改动文件：`go.mod`（1 行）、`go.sum`（2 行）；文档里的版本引用见 §6。

## 3 API 兼容性

对仓库实际用到的 MyGo 面做核对：

- 顶层：`App.Path/SetName`、`RunOnMain`、`Protocol.HandleFunc`、`NewEvent`、
  `WebViewAvailable`、`Window/NewWindow`、`Dialog`、`Shell`、`Clipboard`、
  `ContentRules`、`PageOptions`、`PermissionRequest` 等符号在 0.8.3 全部保留。
- `ui`：`Fill / Grow / Shrink / Height / Padding / FillWidth / Children /
  AlignItems / Background / Border / Radius` 等元素方法签名对比**无差异**
  —— 这也是上一轮状态栏布局修复（`panels.Grow(1)` + `Shrink(0)`）不受影响的原因。
- 包路径 `ui`、`ui/editor`、`ui/markdown`、`ui/markdown/parser`、`transfer`、
  `i18n`、`locales/zh-Hans` 全部存在。
- CLI：`go tool mygo version` 输出 `mygo 0.8.3`；`go tool mygo generate` 与
  `go tool mygo build -platform ...` 在打包脚本中工作正常，四平台产物正常生成。

0.8.3 里有个 `internal/uimigrate` 包，是上游自己的 UI 迁移辅助，不是给下游用的
破坏性变更；我们编译零错误也印证了公开 API 兼容。

产物体积略增（macOS `.app` 约 19.1 → 19.9 MB，Windows exe 约 19.2 → 20.3 MB），
属于框架自身增长，可接受。

## 4 行为变化（需要真机验收）

### 4.1 Markdown 新增扩展

`parser.Default` 从 `GFM | ExtFootnotes | ExtMath` 变成
`GFM | ExtFootnotes | ExtMath | ExtAlerts | ExtMentions | ExtReferences`。
Xinkgo 的 `markdownOptions()` 没有显式设置 `Extensions`，用的是零值即 `Default`，
所以升级后预览会自动获得：

- GitHub 告警块：`> [!NOTE]` 等引用块会渲染成带图标与色条的告警框
  （`Theme.Alerts [5]Color` 新增，上游 `DefaultTheme` 已给 GitHub 默认色，
  `resolve()` 会补齐，我们的 `previewTheme()` 不设也能正常显示）。
- `@octocat`、`#123`、`GH-123` 解析为 `Mention` / `Reference` inline 节点。
  `Options.Resolve` 是新增钩子，我们没有设置，因此它们不会变成可点击链接；
  `parser.HTML()` 也把它们当文本输出，导出与公众号链基本不变。

这是**功能新增而非回归**，但会改变部分笔记的预览外观，需要真机确认观感，
以及确认 `internal/markdown/facts.go` 的标签提取（基于源码文本，不依赖 parser）
不受影响。

### 4.2 Markdown 行内样式扩展：结论不变

重点核对了与上一份调研直接相关的三处：

| 位置 | 0.7.4 | 0.8.3 |
| --- | --- | --- |
| `spans.go` 的 `spanStyle` 字段 | `color / size / font / mark / sub / strike / underline` | **完全一致**，仍未导出 |
| `spans.go` 的 `inlineHTML` 白名单 | 17 个格式标签，无 `span`，忽略所有属性 | **完全一致** |
| `parser.Extensions` | tables / task lists / strikethrough / autolinks / footnotes / math / dollar math | 只多了 alerts / mentions / references，**没有**高亮或属性语法 |

也就是说：背景色、字体颜色、临时字号、`==高亮==` 在 0.8.3 里**依旧不支持**，
`<span style="...">` 在预览里依旧会被当纯文本显示。上一份调研的方案矩阵
（尤其推荐的方案 D：推动上游增加行内样式节点）仍然适用。

一个积极信号：0.8.3 新增了 `Options.Resolve` 这类行内钩子，说明上游愿意扩展
`Options`；向 MyGo 提「可设置 `spanStyle` 的行内节点 + 受限属性白名单」时，
可以沿用这个扩展模式作为提案切入点。

## 5 验收清单

静态门禁已全绿，以下**需要真机**（静态编译不能替代）：

1. `make run` 打开窗口，确认主窗口布局、状态栏（上一轮修的 `Grow(1)`/`Shrink(0)`）、
   左侧 ribbon、菜单、明暗主题都正常。
2. 打开一篇 Markdown 笔记：普通段落、表格、代码块、公式、删除线渲染正常；
   试一个 `> [!NOTE]` 看告警框是否符合预期。
3. 插件体系：启用 demo、命令面板、ribbon、状态栏独立项、插件视图抽屉、
   官方插件库页、WebView 隔离页（含 `xinkgo` / `obsidian` 双入口）。
4. 公众号排版预览与复制到剪贴板、导出 HTML / PDF。
5. Windows / Linux 上的 WebView 差异（P1/P2，按 AGENTS.md 的平台优先级）。

## 6 文档更新范围

已把**当前环境**的版本引用改为 0.8.3：`README.md`、`AGENTS.md`、
`docs/HARNESS.md`、`docs/PROJECT.md`（含 CLI 输出示例）。

历史调研文档（`design/harmonyos.md`、`design/themes.md`、
`design/tree-follows-note.md`、`design/obsidian-plugins.md`、
`design/daily-notes-templates.md`、`requirements/xink-port.md`）里的
`v0.7.4` 是**当时的调研基线**，以及指向 `blob/v0.7.4/...` 的固定版本链接，
保留不改，避免把历史结论误写成新版本的观察。

## 7 回退与风险

- 回退成本极低：`go.mod` 一行、`go.sum` 两行还原即可，无业务代码改动。
- 主要风险是 §4.1 的预览外观变化与真机 GUI/WebView 行为，不是编译或 API。
- 合并前建议先按 §5 走一遍真机验收；本次未合并、未推送。

## 8 参考

- 上一轮 Markdown 行内样式扩展调研（`feature/markdown-style-research` 分支）
- 上游 `mygo v0.8.3`：`ui/markdown/parser/document.go`（`Extensions` / `Default`）、
  `parser/inline.go`（`InlineKind`）、`spans.go`（`spanStyle` / `inlineHTML`）、
  `theme.go`（`Theme.Alerts`、`DefaultTheme`）、`view.go`（`Options.Resolve`）
- `scripts/check.sh`、`scripts/package.sh`：门禁与打包链路
