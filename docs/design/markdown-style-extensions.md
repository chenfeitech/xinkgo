# Markdown 行内样式扩展调研（MyGo 0.8.3 基线）

日期：2026-11-10。状态：**调研结论，未实现**。基线：Xinkgo 0.1.2 + MyGo v0.8.3
（[升级评估](mygo-0.8.3-upgrade.md)）。问题：现有 Markdown 能否支持背景色、
字体颜色、临时字体大小等扩展语法。

## 1 结论先行

**目前不能完好支持，升级到 0.8.3 也没有改变这一点。** 渲染器的底层样式字段
是齐的，但「语法 → 样式节点」这一段缺失；导出链会透传 raw HTML，预览链不会，
因此用 HTML 属性写出来的样式只会造成预览与导出不一致。

| 需求 | 预览（原生 UI） | 导出 HTML / 公众号 | 一致性 |
| --- | --- | --- | --- |
| 背景色 | 仅 `<mark>` 一种固定黄色，不能指定颜色 | `<span style="background:...">` 生效 | `<mark>` 一致，`<span>` 不一致 |
| 字体颜色 | 不支持；`style` 属性被忽略，`<span>` 标签被当纯文本显示 | `<span style="color:...">` 生效 | 不一致 |
| 临时字号 | 不支持；`<small>` 被渲染成下标而非缩小字号 | `<span style="font-size:...">` 生效 | 不一致 |
| `==高亮==`（Obsidian 风格） | 原样显示文字，无此扩展 | 原样输出文字 | 一致但无效 |

**不要**把 `<span style="...">` 当成已支持——预览里会把标签原文
`<span style="color:#e11">` 显示给用户。

## 2 现状：三条渲染链路

| 层 | 位置 | 说明 |
| --- | --- | --- |
| 解析 | 上游 `mygo/ui/markdown/parser` | 自研 CommonMark + GFM 解析器，`Parse(src)` 产出 Block/Inline AST，`HTML()` 产出 HTML |
| 预览 | 上游 `mygo/ui/markdown` 的 `View(c, src, opts)` | **原生 UI 绘制**，不是 DOM；内部 `spanStyle` 有 `color / size / font / mark / sub / strike / underline` 字段 |
| 导出 | [internal/export/export.go](../../internal/export/export.go) | `parser.Parse(src).HTML()` 塞进自带 CSS 的独立页面（HTML / PDF） |
| 公众号 | [internal/wechat/html.go](../../internal/wechat/html.go) | 同一份 `HTML()`，经 `styled()` 给每个标签注入主题的内联样式 |
| 编辑器 | 上游 `mygo/ui/editor` + `internal/highlight` | 源码视图只给代码 token 上色，与富文本样式无关 |
| 索引 | [internal/markdown/facts.go](../../internal/markdown/facts.go) | 只提取标题/标签/链接等事实，不参与渲染 |

Xinkgo 的预览选项在 [internal/view/apply.go](../../internal/view/apply.go) 的
`previewTheme()`：只设置字体、字号、行高、疏密和代码配色，没有设置
`Theme.Mark`（由上游 `resolve()` 补成默认黄色 `RGBA(250,204,21,0.35)`），
也没有任何自定义 inline 钩子。工具栏在
[internal/markdown/format.go](../../internal/markdown/format.go)：粗体、斜体、
删除线、行内代码、标题、列表、引用、代码块、链接、图片、表格、分隔线，共 13 种。

## 3 实测证据（MyGo 0.8.3）

用 `parser.Parse` 与 `ui.NewTester(markdown.View)` 跑同一段文本：

```text
A <span style="color:#e11;background:#ff0;font-size:20px">styled</span>
B <mark>marked</mark> C <small>small</small>
D <b style="color:blue">bold</b> E ==highlight== F ~~strike~~
```

`parser.HTML()` 输出（raw HTML 完全透传、不转义）：

```html
<p>A <span style="color:#e11;background:#ff0;font-size:20px">styled</span>
B <mark>marked</mark> C <small>small</small>
D <b style="color:blue">bold</b> E ==highlight== F <del>strike</del></p>
```

Inline 节点：每个标签是独立的 `HTML` 节点；`==highlight==` 是普通 `text`；
`~~strike~~` 是 `strikethrough`。

预览 `markdown.View` 的 `HasText` 结果：

```text
HasText("<span")=true        HasText("style=")=true     ← 标签被当文本显示
HasText("styled")=true       HasText("marked")=true
HasText("==highlight==")=true                          ← 无高亮扩展
```

原因在上游 `spans.go`（0.8.3）：

- `inlineHTML()` 只认白名单标签
  `br, b, strong, i, em, u, ins, s, del, strike, code, kbd, samp, mark, sub, sup, small`；
  `span` 不在其中，于是 `sb.add(n.Text, ...)` 把标签原文画出来。
- 白名单标签的 `style` 属性被**完全忽略**（`t = t[:i]` 在第一个空格处截断，
  只取标签名），样式由 `htmlStyle()` 固定映射：`mark` → 背景，`small` → `sub`（下标），
  `b` → 粗体。
- `parser.Extensions` 是固定位掩码（tables / task lists / strikethrough /
  autolinks / footnotes / math / dollar math / **alerts / mentions / references**），
  **没有自定义扩展注册点，也没有高亮或属性语法**。
- `markdown.Options` 有 `Block`、`HTML`、`InlineMath`、`CodeBlock`、
  `Resolve`（0.8.3 新增，给 @mention / #reference 解析链接）等钩子，
  但**没有通用行内 span 钩子**；`Block` 是块级，只能整块替换。
- `spanStyle` 字段全部未导出，Xinkgo 无法从包外设置颜色或字号。

`export.go` 的 CSS 与公众号主题 `Style` 都没有 `span` / `mark` 键，
`wechat.withStyle()` 会保留原标签的 `attrs`——所以 HTML 样式在导出里生效，
这恰恰放大了与预览的不一致。

## 4 可行方案

| 方案 | 做法 | 覆盖链路 | 代价 | 一致性 | 评价 |
| --- | --- | --- | --- | --- | --- |
| A | 只用 `<mark>` 做单一高亮 | 预览 + 导出 | 零改动 | 一致 | 只有一种黄色，语法是 HTML 不是扩展语法 |
| B | Xinkgo 层预处理，把自定义语法转成 `<span style>` 后再交给 `HTML()` | 仅导出/公众号 | 小 | 预览会显示标签原文 | **不推荐单独使用** |
| C | 预处理 + `Options.Block` 自己画整段（用 `ui.Span` 设色/字号/背景） | 仅预览 | 大：要复刻段落排版、选区、复制为 Markdown/HTML、链接与图片 | 导出仍需另做 | 工作量与回归风险最高 |
| D | 推动上游 MyGo 增加扩展：新 Inline 节点 + 把样式落到 `spanStyle`（或让 `inlineHTML` 解析有限的 `style` 属性） | 预览 + 导出 + 公众号 | 中（依赖升级 + Xinkgo 三处适配） | 一致 | **推荐的中长期路径** |
| E | 把 `ui/markdown` fork 进仓库改造 | 全链路 | 高：长期与上游同步 | 一致 | 维护负担重，不推荐 |

0.8.3 新增的 `Options.Resolve` 行内钩子是个积极信号：上游愿意扩展 `Options`，
向 MyGo 提「可设置 `spanStyle` 的行内节点 + 受限属性白名单」时可以沿用这个模式。

## 5 建议的语法与落地顺序

语法候选（都需要 D 才有意义）：

1. `==text==` 高亮：复用 Obsidian 习惯，映射到单一 `mark` 背景；多色用
   `=={red}text==` 或 `==text=={color:red}` 之类带属性的形式。
2. Pandoc 风格 `[text]{.cls}` 或 `[text]{color=#e11;font-size=1.2em}`：
   表达力完整，但与 wikilink `[[...]]`、数字/标签解析容易冲突。
3. HTML `<span data-...>`：最省解析改动，但正文里写 HTML 体验差。

落地顺序：

1. 先定「预览优先」口径：任何样式必须先在原生预览可见，再谈导出。
2. 上游加可扩展的行内样式节点（颜色/背景/字号三个属性即可），并让
   `HTML()` 为它输出 `<span style="...">`；同时定义**白名单属性值**（只接受
   颜色、长度，无 `url()` / 表达式），避免导出链注入面扩大。
3. Xinkgo 升级依赖，在 `previewTheme()` 补主题色（明暗两套），在
   `export.go` CSS 与 `wechat/theme.go` 的 `Style` 里补 `span` / `mark` 规则。
4. 工具栏加「字体颜色 / 背景色 / 字号」按钮，往选区里写新语法
   （`Format` 枚举 + `format.go` 的插入规则）。
5. `facts.go` 的标签/链接提取要屏蔽新标记（类似现有 `maskInlineCode`），
   避免 `=={red}#tag==` 之类的文本污染索引。
6. 明暗主题、PDF 打印、复制到富文本（预览的 Copy as HTML）都要验收。

## 6 风险与明确不做

- 不用 raw HTML 冒充「已支持」：预览会显示标签文本，正文看起来是破损的。
- 不引入新的 Go 依赖（遵循 AGENTS.md）；D 是升级现有 `mygo` 版本。
- 不放开任意 `style`：`HTML()` 目前对 raw HTML 不转义，公众号/导出链已有
  透传面，扩展语法必须用受限属性白名单。
- 不做「编辑器里实时着色」：源码视图由 `ui/editor` 的语法高亮负责，富文本
  样式只在预览出现，这是 Xinkgo 的双视图模型决定的。
- 0.8.3 默认新增了 alerts / mentions / references，预览外观已有变化；
  行内样式扩展要避开 `#123` 式的属性写法，别和 reference 语法打架。

## 7 参考

- 上游 `mygo v0.8.3`：`ui/markdown/parser/document.go`（`Extensions` / `Default`）、
  `parser/inline.go`（`InlineKind`）、`parser/html.go`（`HTML()`）、
  `spans.go`（`spanStyle` / `inlineHTML` / `htmlStyle`）、`theme.go`（`Theme.Mark`）、
  `view.go`（`Options`，含 `Resolve`）
- [mygo-0.8.3-upgrade.md](mygo-0.8.3-upgrade.md)：升级评估与行为变化
- [../../internal/view/preview.go](../../internal/view/preview.go)：预览调用点
- [../../internal/export/export.go](../../internal/export/export.go)：HTML / PDF 页面
- [../../internal/wechat/style.go](../../internal/wechat/style.go)：`styled()` / `withStyle()`
- [../../internal/markdown/format.go](../../internal/markdown/format.go)：工具栏格式
