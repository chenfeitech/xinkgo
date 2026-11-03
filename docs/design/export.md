# 预览导出（PDF / HTML）设计

日期：2026-11-02。范围：在预览面板加一个"导出"入口，把当前笔记导出成 PDF 或 HTML。

## 1 两个决定性发现

1. **MyGo 的 Markdown 解析器自带 HTML 输出**：`mygo/ui/markdown/parser` 的
   `Document.HTML()` "renders the document as HTML, as cmark-gfm renders it"。预览
   走的是同一棵树（`markdown.View(c, w.previewText(n), opts)`）。所以 HTML 导出
   **不需要自己写渲染器**，也不必担心与预览不一致。
2. **MyGo 有 `Page.PrintToPDF`**：`PDFOptions{PageSize, Landscape, Margins,
   Background}`，三个桌面端都实现了（macOS：WebKit print op；Windows：DevTools
   协议；Linux：GTK "Print to File"；wasm：unsupported）。它还专门处理了 WebKit
   把中文打成 radical（⼈ U+2F08）的问题。

结论：**一个 HTML 导出器 + 一条打印通路**，PDF 就是同一份 HTML 交给系统 WebView
打印。中文字体、分页、断行全部由浏览器引擎负责，我们只写样式表。

备选（**不推荐，只作兜底**）：纯 Go 生成 PDF（gopdf/fpdf 之类 + 嵌入中文字体子集）。
代价是新依赖（必须无 cgo、许可宽松）、自己分页排版、字体子集化，2～4k 行 + 字体
资源，且保真度反而更差。只有打印通路在三端都不可用时才考虑。

## 2 入口与流程

- 预览面板加"导出"按钮（图标 + 菜单：PDF / HTML）；File 菜单加"导出为 → PDF /
  HTML…"（菜单归 `internal/app`，视图不碰）。
- 视图只回调：`Workspace` 增加 `OnExport func(format string)`（沿用现有的
  `OnSnapshot` / `OnAiLoad` / `OnOpenOutside` 回调模式），由 app 接线；视图不做
  任何 I/O（AGENTS 第 6 条）。
- app 侧流程（照 `settings.go` 里 `exportConfig()` 的写法）：
  1. `mygo.Dialog.Save`（Filters 按格式，默认文件名 = 笔记名，默认目录 = 笔记所在
     目录或上次导出目录）；
  2. goroutine 里生成 HTML（可能慢，不能卡视图）；
  3. HTML：直接写文件；PDF：回到主线程，建 `WindowOptions{Hidden: true}` 的窗口 →
     `Page.LoadFile` 临时文件 → 等加载完成 → `PrintToPDF` → 写文件 → 关窗口。临时
     页面写在**目标 PDF 的同一目录**，且只在打印结束（或失败）后才删：`RunOnMain`
     同步返回时窗口还没读到它，在 goroutine 里 `defer os.Remove` 会先把它删掉，
     于是既没有 PDF 也没有报错；放在机器临时目录里还会让"图片放在旁边"的相对
     路径落空（见 §8）；
  4. 成败用 `ws.Say(...)` 报一行；失败另外弹提示。
- `mygo.WebViewAvailable()` 为假时，PDF 一项禁用并说明原因。

## 3 导出内容：与预览对齐的边界

源文本必须与预览一致：`w.previewText(n)`（wiki 链接已解析、数据文件渲染成代码块、
图片文件渲染成 `![](...)`）。解析扩展也照预览：`parser.Default`（GFM + 脚注 + 数学），
**不开** `ExtDollarMath`（预览没开）。

导出必须自己补的几处：

| 点 | 处理 |
|---|---|
| 图片 | 预览用 `markdown.FileImages(dir)` 按相对路径读。导出默认复制到 `<name>.assets/` 并改相对路径；另给"单文件（base64 内联）"选项。`ImageDisplay == ImageHidden` 时按预览一样不带图 |
| wiki / 内部链接 | 单篇导出时目标笔记不在产物里 → v1 保留文字、不可点；整库导出（后续）再改成相对链接 |
| 数学 | 预览未接渲染器，显示 TeX 源码；导出同样保留 TeX 文本 |
| 代码块高亮 | parser 的 HTML 只给 `<pre><code class="language-x">`，**没有 token 高亮**。要用 `internal/highlight` 的主题再跑一次分词 + 生成 CSS；若其 API 取不到 token→颜色（待读源码确认），v1 降级为单色代码块 + 边框 |
| 主题配色 | 用 `internal/theme` 当前方案生成 CSS 变量；`@media print` 里按"打印背景"选项决定要不要底色 |
| 分页质量 | `break-inside: avoid`（代码块/表格/图片）、标题 `break-after: avoid`、`orphans/widows`；纸张/横竖/边距走 `PDFOptions` |
| 页眉页脚页码 | Chromium 对 `@page` 边距盒支持有限 → v1 不做，HTML 顶部放标题/日期元信息；后续再说 |
| 封面 / front matter | v1 不导出 front matter（它不是预览的一部分） |

## 4 代码落点（按仓库约定）

- **新包 `internal/export`**（叶子）：`HTML(src string, o Options) ([]byte, error)`，
  依赖 mygo 的 parser、`internal/theme`、`internal/highlight`、`internal/media`、
  `internal/config`。按 CONTRIBUTING 要求同步更新依赖方向表与 PROJECT.md 的目录树。
- **`internal/app/export.go`**：菜单项、按钮回调接线、Save 对话框、goroutine、
  PDF 的隐藏窗口流程、状态与错误提示。
- **`internal/view/preview.go`**：预览面板的按钮与弹出菜单、新增 `OnExport`
  回调；对"由系统程序预览的文件"（图片、PDF）禁用。
- **`internal/config`**：新增 `ExportConfig`（纸张、横竖、边距、是否嵌入图片、
  导出配色）＋默认值＋ Sanitize ＋测试——按 config 包的三件套约定。
- **`internal/locale`**：新键（菜单、对话框标题、按钮、状态、错误、选项名），
  en 为源、加 zh-Hans，`go tool mygo generate` 重新生成。
- **设置窗口**：放一组"导出"选项（或并入预览分组）。

## 5 测试

- `internal/export`：表驱动覆盖标题/列表/任务/脚注/表格/代码块/图片/引用/嵌套/wiki
  链接；断言落在片段上（整文件快照太脆）；图片内联与外链两种模式。
- `internal/app`：菜单项与禁用条件（照 `menu_test.go` 的写法）；导出写文件与错误
  分支注入临时目录。
- PDF 不进单测（需要真实 WebView）：接口隔离 + 手工验收清单。
- 手工验收（**macOS 先**，P0）：中文/emoji、表格跨页、代码高亮、大图、脚注；
  再 Windows（P1）、Linux（P2）。

## 6 风险与待确认

1. `parser.HTML()` 与预览 `View` 的细节差异（代码块 class、脚注、任务框）→ 先写
   一组对照测试钉住，别靠肉眼。
2. `internal/highlight` 能否给出 token→颜色映射（待读源码）→ 决定代码块高亮是否
   进 v1。
3. 打印通路依赖系统 WebView：隐藏窗口是否需要先显示才能打印（macOS 的 WebKit
   print op 可能要求窗口存在）→ 已在 macOS 验过：隐藏窗口不必显示，
   `OnReadyToShow`（= 页面 load finished）之后 `PrintToPDF` 正常出 PDF。
4. 三套引擎（WebKit / WebView2 / GTK）分页结果不同 → 逐端验收，样式保守一点。
5. 单文件 HTML 内嵌大图会到几十 MB → 给上限或提示。
6. 必须保持 `CGO_ENABLED=0`：本方案不引入新依赖（除后续真要做纯 Go PDF 时）。

## 7 里程碑

| 阶段 | 内容 | 估时 |
|---|---|---|
| M1 spike | `parser.HTML()` + 打印样式 + 隐藏窗口 `PrintToPDF` 出一份中文 PDF | 1～2 天 |
| M2 | HTML 导出（图片、主题、设置、catalog、测试） | 3～5 天 |
| M3 | PDF 接入（纸张/边距/背景、状态提示、禁用条件） | 2～3 天 |
| M4 | 三端验收与修 bug | 2～3 天 |

合计约 1～1.5k 行（导出器 400～700、编排 250～400、配置 80、测试 200～400），
一人 1～2 周。整库导出（index.html + 每篇一页 / 合并 PDF）不在本次范围，但链接
与 CSS 要为它留口。

## 8 实现状态（v1）

- **`internal/export`**：`HTML(src, Options)` 用 MyGo 的 parser（与预览同一棵
  树）写出 HTML，套上配色（当前方案的明暗色）、排版与打印规则。图片三种处理：
  写进页面（默认，base64）、放在旁边（`assets/`，同名自动加序号）、省掉（留下
  它的描述）。
- **`internal/app/export.go`**：File 菜单"导出为…"（HTML / PDF），预览右下角
  一个浮动的"导出"按钮（绝对定位，不占预览的版面）；对话框由 app 问，生成与写
  文件在 goroutine；PDF 走隐藏窗口 → `LoadFile` → `Page.PrintToPDF`（A4 或
  Letter，来自配置）→ 写文件。机器上没有 webview 时，PDF 一项直接说明原因。
  打印用的临时页面（`.xinkgo-*.html`）与目标 PDF 同目录，打印结束或页面加载失败
  时才删除；另接 `Page.OnDidFailLoad`，页面读不出来时报错而不是一直等。
- **`internal/config`**：`export`（纸张、图片）+ 默认值 + 校验 + 测试；设置窗口
  的预览页多一组"导出"。
- **PDF 用浅色**：纸是白的，而深色方案的正文是浅色的，印上去什么也看不清。
- **未做**：代码块语法着色（parser 的 HTML 不带 token，仓库里也没有公开的分词
  器）、页眉页脚页码、整库导出。
- **要人工验收**：PDF 打印通路（隐藏窗口在各端是否真能打印）、三端分页效果、
  中文与 emoji、大图与脚注。
- **已验（macOS，darwin/arm64，macOS 26）**：隐藏窗口打印通、中文与 emoji、
  代码块与表格、图片（内联 base64 与 `assets/` 相对路径两种）都进了 PDF；
  `internal/app` 的 `TestWritePage_*` 钉住临时页面的位置与内容，PDF 打印本身
  要真 webview，不进单测。
