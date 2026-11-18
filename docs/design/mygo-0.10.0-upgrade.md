# MyGo 0.8.3 → 0.10.0 升级评估

日期：2026-11-18。状态：**可以升级，已在 `feature/mygo-0.10.0` 完成升级并通过全部静态门禁与打包验证，待真机验收与合并**。

## 1 结论先行

**可以升级，且不需要改一行业务代码。** `go.mod` 只从 `v0.8.3` 改到
`v0.10.0`；`CGO_ENABLED=0 go build ./...`、`go test ./...`、
`go tool mygo generate`（catalog 无变化）、`scripts/check.sh`（五平台编译 /
版本一致 / 资源门禁）、`make package`（macOS .app + .dmg、Windows .exe）
全部通过。

本次跨过了 0.9.x（0.9.0 / 0.9.1 / 0.9.2）直接到 0.10.0，因此比上一次
升级多核对了公开 API、Markdown 默认扩展、拖拽与主题的运行期行为，见 §3、
§4。上游没有 changelog 与 GitHub Release body，兼容性以两版模块源码对比
和实际编译为准。

## 2 版本与依赖

| 项 | 0.8.3 | 0.10.0 |
| --- | --- | --- |
| Go 版本 | 1.27.3 | 1.27.3（一致） |
| 直接依赖 | `mygo v0.8.3` | `mygo v0.10.0` |
| CLI | `go tool mygo version` → `mygo 0.8.3` | → `mygo 0.10.0` |

0.10.0 的上游 `go.mod` 新增了 flac / aac / rpm（音频解码与 Linux 打包）
相关依赖。`go mod tidy` 后本仓库新增 7 个 indirect 依赖：

```
github.com/ProtonMail/go-crypto v1.4.1
github.com/cloudflare/circl v1.6.2
github.com/klauspost/compress v1.18.6
github.com/ulikunitz/xz v0.5.15
go.digitalxero.dev/rpm v0.3.4
golang.org/x/crypto v0.57.0
golang.org/x/sys v0.48.0
```

`go mod why` 显示它们来自 **MyGo CLI 工具**（`cmd/mygo` 的 rpm 打包支持），
不是业务代码：Xinkgo 的 `import` 面没有变化。逐个扫过这些包的源码，
没有 `import "C"`（rpm 的 `header.go` 与 circl 的 `k12.go` 里的 `"C"` 是
字符串字面量），`CGO_ENABLED=0` 下五平台编译全部通过，符合 AGENTS.md 的
禁 cgo 约束。`go.mod` 里原有的 purego / typesetting / x-image / x-net /
x-text 版本均未变。

改动文件：`go.mod`（依赖表）、`go.sum`（+24/-2）；文档见 §6。

## 3 API 兼容性

对仓库实际用到的 MyGo 面做核对：

- 顶层：`App.Path/SetName`、`RunOnMain`、`Dialog`、`Clipboard`、
  `NewWindow/Window`、`NewMenu/Menu/MenuItem`、`NewEvent`、`Protocol`、
  `Page/PageOptions/ContentRules/NewSession`、`WatchFiles`、`Shell` 等符号
  在 0.10.0 全部保留，编译零错误。
- `ui` 公开 API 生成器 `ui/api_gen.go` 从 8783 行增至 9396 行，是**新增**
  能力；`DragOver[T]` / `Drop[T]` / `EditableText` 等我们调用的入口签名一致。
- 包路径 `ui`、`ui/editor`、`ui/markdown`、`ui/markdown/parser`、
  `transfer`、`i18n`、`locales/zh-Hans` 全部存在。
- `go tool mygo generate` 输出 `The accessors of internal/locale are up
  to date`，`catalog_gen.go` 与两个 JSON 均无需改动。
- CLI：`go tool mygo version` 输出 `mygo 0.10.0`；`generate` 与
  `build -platform ...` 在打包脚本中工作正常，四平台产物正常生成。

## 4 行为变化（需要真机验收）

### 4.1 不变的默认行为

- Markdown：`parser.Default` 在两版均为
  `GFM | ExtFootnotes | ExtMath | ExtAlerts | ExtMentions | ExtReferences`，
  `document.go` 的扩展位完全一致。0.8.3 引入的告警块 / @提及 / #引用外观
  不变，`internal/markdown/facts.go` 基于源码文本、不依赖 parser，不受影响。
- 拖拽：`transfer` 的 `data.go` 两版逐字相同，`FileList` / `URIList` /
  `Copy` 等格式与操作不变，外部文件拖入照旧。
- 主题：`Theme` / `Palette` 结构、`DefaultTheme`、`Color.Alpha` 一致。

### 4.2 窗口内拖拽：上游有改动，语义兼容

`ui/dragdrop.go` 在 0.10.0 有实质改动，是本次最值得核对的一处：新增
`valueDrag.ghost`（源元素不再被构建时，用首帧快照继续画拖拽副本）、
`retarget()`（指针不动但界面变化时重算放置目标）、以及 `paintDrag` 里
`p.copying` 的处理（避免副本透出 WebView 页面）。

这些是**渲染与目标命中的增强，不改公开 API**：
`Drag(value)` / `Dragging()` / `DragOver[T]` / `Drop[T]` / `DropData` 的
行为契约不变。本仓库的目录拖拽（`internal/view/tree_dnd.go`）与快捷重命名
在 `go test ./internal/view/` 下全部通过，包括拖到文件夹 / 文件 / 空白处、
自拖与子目录拒绝、双击改名与焦点选中。真机仍需确认拖拽副本的观感与
高亮（见 §5）。

### 4.3 其余新增

0.10.0 上游 `ui/list.go`、`ui/theme.go`、`ui/paint.go` 有改动，均无公开
签名变化；`docs/` 新增 `mobile/` 与 `native-modules.md`（Xinkgo 未用）。
产物体积略增（macOS `.app` 约 20.7 MB，Windows `.exe` 约 21.2 MB），
属于框架自身增长，可接受。

## 5 验收清单

静态门禁与打包已全绿，以下**需要真机**（静态编译不能替代）：

1. `make run` 打开窗口，确认主窗口布局、状态栏、左侧 ribbon、菜单、
   明暗主题都正常（最近一轮 status bar 常驻修复一并确认）。
2. 打开一篇 Markdown 笔记：普通段落、表格、代码块、公式、删除线、
   `> [!NOTE]` 告警块渲染正常。
3. **目录拖拽**：把一个文件拖到文件夹 / 文件 / 空白处，确认副本跟随、
   目标高亮、落点正确；自拖、拖进自身子目录应被拒绝；双击与 F2 改名、
   扩展名保留、点击别处提交。
4. 插件体系：启用 demo、命令面板、ribbon、状态栏独立项、插件视图抽屉、
   官方插件库页、WebView 隔离页（含 `xinkgo` / `obsidian` 双入口）。
5. 公众号排版预览与复制、导出 HTML / PDF、全文本文件编辑与源码预览。
6. Windows / Linux 上的 WebView 差异（P1/P2，按 AGENTS.md 的平台优先级）。

## 6 文档更新范围

已把**当前环境**的版本引用改为 0.10.0：`AGENTS.md`、`README.md`、
`docs/HARNESS.md`、`docs/PROJECT.md`（含 CLI 输出与打包输出示例）。

历史调研文档（`design/mygo-0.8.3-upgrade.md`、`design/harmonyos.md`、
`design/themes.md`、`design/tree-follows-note.md`、
`design/obsidian-plugins.md`、`design/daily-notes-templates.md`、
`requirements/xink-port.md`）里的 `v0.7.4` / `v0.8.3` 是**当时的调研基线**
与指向 `blob/v0.7.4/...` 的固定版本链接，保留不改，避免把历史结论误写成
新版本的观察。

## 7 回退与风险

- 回退成本极低：`go.mod` 的依赖表、`go.sum` 与 4 个当前环境文档还原即可，
  无业务代码改动。
- 主要风险是 §4.2 拖拽副本观感与 §5 的真机 GUI / WebView 行为，不是编译或
  API；跨版本跨度（跳过 0.9.x）也意味着上游可能有未写入文档的运行期变化，
  需靠真机验收覆盖。
- 合并前建议先按 §5 走一遍真机验收；本次未合并、未推送。

## 8 参考

- 上一轮升级评估：`design/mygo-0.8.3-upgrade.md`
- 上游 `mygo v0.8.3` 与 `v0.10.0` 模块源码：`ui/api_gen.go`、
  `ui/dragdrop.go`、`ui/markdown/parser/document.go`、`transfer/data.go`、
  `ui/theme.go`、`go.mod`
- `scripts/check.sh`、`scripts/package.sh`：门禁与打包链路
