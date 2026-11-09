# 文本文件编辑与源码预览（v1）

日期：2026-11-09。范围：让库里所有非二进制文本文件都能打开、编辑、自动保存，
并得到一份安全的源码预览；HTML / JS / CSS 等有语言着色，Makefile 等无后缀文件
按内容识别。

## 1 现状与问题

改动前有两条互不一致的判定：

- `internal/media` 手写扩展名白名单，`.html/.js/.css` 能编辑，但 `.scss/.vue/.mjs`
  等 MyGo 已支持的扩展名漏掉，无后缀 `Makefile` 落到 `KindOther`。
- `internal/view/tabs.go` 的 `newEditor` 已经调用 `editor.LanguageFor`，它认识
  `Makefile`、`Dockerfile`、`.scss`、`.vue`、`.mjs` 等；但 `Tabs.Open` 用
  `!media.IsEditableText` 决定 `OpenPreview`，于是无后缀文本文件只显示「不可用」卡片。

预览也有问题：`previewText` 只对 JSON/YAML/TOML 包 fenced code，其它文本一律走
`xmd.Links` 后交给 `markdown.View`，所以 JS/HTML/CSS 源码被当 Markdown 渲染；
`ExportText()` 返回同一个结果，导出与公众号排版跟着错。

## 2 目标与非目标

目标：所有非二进制文本文件可编辑；HTML/JS/CSS 等沿用 MyGo 的词法高亮、括号匹配、
注释切换与缩进规则；`Makefile`、`.env`、`LICENSE`、`.gitignore` 等无后缀文件按内容
识别；非 Markdown 文本在预览里显示为带语言着色的源码块。

非目标：不引入语言服务器、语法诊断、跳转定义；不渲染或执行 HTML/JS；不默认支持
无 BOM 的非 UTF 编码；不导出源码原文；不把非 Markdown 文件纳入双链索引、命令面板、
大纲、反向链接或全局搜索；不编辑 SVG 源码；不做 HTML 实时渲染预览。

## 3 判类：一个入口

新增叶子包 `internal/textfile`，`media`、`vault`、`note`、`view` 都委托它，避免扩展名
表继续漂移。`textfile` 只依赖标准库与 `mygo/ui/editor`，不反向依赖 app/view，无 cgo，
无新增直接依赖。

- `ByName(path)`：已知二进制扩展名直接判为不可编辑；否则用 `editor.LanguageFor`
  取语言名，小写后经 `editor.LanguageNamed` 复核为 fence info；`.txt/.csv/.env` 等
  纯文本扩展名与 `LICENSE/Procfile/Justfile` 等无后缀名判为纯文本；其余返回 Unknown。
- `Sniff(head)`：纯函数，只读前 `SniffBytes = 4096` 字节。空文件视为文本；NUL、
  非法 UTF-8、控制字符比例过高判为二进制或编码不支持；UTF-8 BOM、UTF-16LE/BE BOM
  分别记录编码。读取头部的尾部可能截断多字节字符，先裁掉不完整的尾部再判合法性。
- `Classify(path, maxTextBytes)`：按名字能定的不读内容或只读头部确认编码；名字不定
  的文件只读头部嗅探；超过 `maxTextBytes` 直接返回 `ReasonTooLarge`。
- `Fence(text, lang)`：扫描源码中最长连续反引号，围栏取 `max(3, n+1)` 个反引号，
  防止 JS 模板字符串或含 ``` 的文本截断代码块。

语言名不再手写：`textfile.Language(path)` 取 `editor.LanguageFor(base).Name` 的小写，
所以 `Makefile → makefile`、`.scss → css`、`.vue → html`、`.mjs → javascript`
与编辑器同源。`media.Language` 与 `media.IsEditableText` 改为委托 `textfile`。

## 4 编辑链路

`Tabs.Open` 调 `textfile.Classify`，把 `textfile.Info` 存进 `OpenNote.Info`：

- `Info.Text` 为真：`note.OpenEncoded(path, Info.Encoding)` 读全文件并解码，编辑器
  仍由 `editor.LanguageFor(filepath.Base(path))` 设置语言。
- `Info.Text` 为假：`note.OpenPreview` 只 stat，不把二进制/过大/编码不支持的文件读进
  编辑器；`fileCard` 按 `Reason` 显示「文件太大」「编码不支持」或「既不能编辑也不能显示」。

工具栏分三路：`.md/.markdown` 保留 Markdown 按钮；JSON/YAML/TOML 保留 `dataBar`；
其它文本用 `textBar` 显示文件类型与「按源码显示，不渲染」，不再给 Markdown 格式按钮。

`note.Note` 增加 `Encoding` 字段，`Save` 用 `textfile.Encode` 写回，保留 BOM 与
CRLF；`TakeExternalText` 用 `textfile.Detect/Decode` 重新载入外部改动并更新编码。

## 5 预览与导出

`previewText(n)` 按四类分派：

| 文件 | 预览 |
|---|---|
| `n.Preview` | 图片本身或原因卡片，不变 |
| `.md/.markdown` | `xmd.Links` + Markdown 渲染，不变 |
| JSON/YAML/TOML | 原有格式化 + `textfile.Fence(text, 语言)` |
| 其它文本 | `textfile.Fence(原文, textfile.Language(path))` |

非 Markdown 源码预览不再调用 `xmd.Links`，避免把 `[[...]]` 或 HTML 当 Markdown 解释。
`markdown.Options.WrapCode` 跟随 `editor.soft_wrap`。`markdown.Theme.CodeColors` 已由
`highlight.EditorTheme` 注入，fenced code block 会用 `editor.LanguageNamed(info)` 按
同一套 token 主题着色，不需要新着色器。

`ExportText()` 仍返回 `previewText(n)`，因此导出/公众号得到的是「带代码块的页面」，
不是源码原文；源码另存留作后续功能。HTML 只作为源码显示，不执行脚本、不加载外部
资源。

## 6 文件树、索引与搜索

`vault.Entry` 增加 `Kind media.Kind` 与 `Text bool`，扫描时写一次，树图标和后续过滤
复用。`show_all_files=true` 时保持全显示，只按名字分类；`show_all_files=false` 时，
已知图片保留，已知二进制跳过，名字不定的文件用 `Classify` 嗅探后决定是否显示。

`media.Kind` 增加 `KindText`，`.csv/.tsv` 从表格类移到文本类，`.xls/.xlsx/.numbers`
仍是二进制表格；`.svg` 保持图片预览。`README` 等 MyGo 识别为 Markdown 的无后缀文件
仍按源码预览，避免把任意无后缀文件当 Markdown 渲染。

`index.IsNote` 仍只认 `.md`，命令面板、大纲、反向链接、全局搜索保持 Markdown 语义；
扩展文本文件类型不改变双链索引或 AI 工具的白名单。

## 7 编码、大小与配置

v1 支持 UTF-8、UTF-8 BOM、UTF-16LE/BE BOM；不带 BOM 的 GBK/Big5 等不静默猜测，
显示「编码不支持」并交给外部程序。新增 `config.editor.max_text_file_bytes`，默认
4 MiB，范围 1–64 MiB，超过上限不进编辑器。所有新文案进 `internal/locale/en.json`
与 `zh-Hans.json`，执行 `go tool mygo generate` 生成访问器。

## 8 测试与验收

- `internal/textfile`：名称表、嗅探、编码往返、围栏转义、Classify 单测。
- `internal/media` / `internal/vault` / `internal/config` / `internal/note`：新类型、
  扫描过滤、默认值与校验、编码保存。
- `internal/view`：打开 HTML/JS/CSS/SCSS/Vue/MJS、Makefile、LICENSE、.env、.txt、
  .csv；过大/编码不支持/二进制的原因卡片；源码预览、工具栏、导出语义。
- 门禁：`make check`、`make test-race`。人工验收 macOS 优先，样例库含
  `index.html`、`app.js`、`styles.css`、`Makefile`、`LICENSE`、`.env`、`data.json`、
  `pic.png`、`a.zip`，并切换 `show_all_files=false`。

## 9 已知限制与后续

v1 不支持 HTML 渲染预览、无 BOM 非 UTF 编码、SVG 源码编辑、源码原文导出、非
Markdown 全文搜索/索引、语言服务器。后续可按需增加：编码设置与 `x/text`、HTML
沙箱渲染切换、SVG「编辑源码」动作、导出源码、全库文本搜索开关。
