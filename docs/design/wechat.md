# 公众号排版（预览右上角）

日期：2026-11-02。范围：预览面板右上角加一个入口，把当前笔记按公众号文章的
样式排版：默认正常 Markdown，另外可以选一个公众号主题；选了主题，预览里看到
的就是那篇文章，"复制"把它放到剪贴板上，粘贴进公众号后台即可发布。样式要能
不断增加。

**要写样式看这篇**：[wechat-style-guide.md](wechat-style-guide.md)（键位表、
CSS 能做与不能做、改 → 看 → 试、排错清单），示例在 [wechat-styles/](wechat-styles/)。

## 1 两个前提

1. **公众号编辑器不吃样式表**：它保留行内 `style`，丢掉 `<style>`、`<script>`、
   class、大部分 `display`/`position`、`<input>`。所以排版结果必须是**每个标签
   自带样式**的 HTML 片段，不能靠 CSS 文件。
2. **行内样式可以数据化**：一个"主题"就是"元素 → CSS 声明"的一张表
   （`p`、`h2`、`blockquote p`、`pre>code`……）。渲染器把 MyGo 解析器写出的
   HTML（与预览同一棵树）过一遍，按这张表给每个标签加 `style`。于是**加一个样式
   只是加一个 JSON 文件**，不用改代码——这就是"样式可以不断增加"的落点。

备选（**不做**）：把 Markdown 渲染成图片再粘。保真但不可编辑、不可选字，且要
新依赖。

## 2 入口与交互

- 预览面板右上角（`previewHead`，整窗预览时是 `previewBar`）：
  - 「公众号」菜单按钮：第一项「正常 Markdown」（默认），下面依次是各个主题，
    当前生效的打勾；
  - 选中某个主题后，旁边多出一个「复制」按钮：按当前主题复制，可直接粘贴。
- 选了主题，预览面板里画的就是那篇文章（**所见即所贴**）：`ui.WebView` 里显示
  生成的页面，而这块页面是**手机屏幕那么宽**（414 DIP，圆角 + 1px 边框，居中，
  四周留 12）：公众号是在手机上看的，一行在哪儿断、一张图占多宽，都是这个宽度
  下的样子，预览就该是这个宽度下的样子。屏幕比文章宽：页面左右各留 20 CSS px
  （`wechat.pagePad`，只在预览的页面上，粘贴的那份不留——公众号后台自己会给），
  剩下的 370 上下就是文章在手机上的宽度。窗格比屏幕窄时先让出留白，再让出
  宽度——被切掉一边的文章不算文章。
- 正常 Markdown 预览照旧自动适配：吃满窗格、`MaxWidth` 封顶，与手机宽度无关。
- `mygo.WebViewAvailable()` 为假时，仍可复制，预览退回普通 Markdown 并说明原因。
- 选择写进配置（`preview.wechat`，空串 = 正常 Markdown），下次打开还是它。

## 3 排版的边界（与预览对齐 + 公众号的规矩）

源文本与预览一致：`w.previewText(n)`（wiki 链接已解析、数据文件渲染成代码块、
图片文件渲染成 `![](...)`）。

| 点 | 处理 |
|---|---|
| 样式 | 全部行内。主题缺的键落回内建基底，所以一个主题只要写它想改的那几项 |
| 标题 | 各主题不同：居中/左侧色条/下边框；`h1` 当文章标题 |
| 代码块 | `pre` 带底色与圆角，`white-space: pre-wrap`（公众号不给横向滚动）。仓库里没有公开的分词器 → v1 是单色代码，配色随主题 |
| 行内代码 | 底色 + 主题色 + 等宽字体 |
| 表格 | `border-collapse: collapse`、表头底色、单元格描边；不用 `display:block` |
| 列表 | 保留原生 `ul/ol/li`，只给间距 |
| 任务列表 | cmark 写的是 `<input type=checkbox>`，公众号会丢 → 换成 ☐ / ☑ |
| 引用 | 左侧色条 + 淡底 |
| 图片 | 写成 base64 内联（公众号后台没有笔记的文件夹可读）。图片要在公众号里"重新上传"是它自己的规矩，见 §7 |
| 链接 | 保留 `<a href>`；公众号对外链有限制，外链的成败由它决定 |
| 数学 | 预览未接渲染器，导出同样保留 TeX 文本（等宽字体） |
| 脚注 | 保留 cmark 的 `<section class="footnotes">`，样式走 `section` 键 |

## 4 主题从哪来：内建 + 一个可指定的目录

- **内建**：`internal/wechat` 里几套（`ink` / `jade` / `lapis` / `amber` /
  `rose`），名字是样式自己的名字（同 `theme.Scheme.Title` 的规矩，不进 catalog），
  另带 `version` / `author` / `description`。
- **目录可指定**：设置 → 预览 → 公众号 →「样式目录」，手填或用「浏览…」选一个
  文件夹；写进配置 `preview.wechat_theme_dir`，**留空**用应用自己的目录
  （`<用户数据目录>/wechat`）。改了目录立刻重读，样式菜单与预览随之更新。
- **一个目录多个文件**：目录里每个 `*.json` 是一套样式（不递归子目录），按文件名
  顺序读入。文件里写什么：

  ```json
  {
    "id": "my-style",
    "name": "我的样式",
    "version": "2.1.0",
    "author": "谁做的",
    "description": "绿底红字，标题居中",
    "extends": "ink",
    "style": {
      "h2": "margin: 1.6em 0 .8em; border-left: 4px solid #0f9d58; padding-left: .6em;",
      "code": "background: #f2faf5; color: #0f7a45;"
    }
  }
  ```

  | 键 | 必填 | 说明 |
  |---|---|---|
  | `id` | ✅ | 配置里存的就是它；两个文件不能同 id（同 id 后者被拒并说明） |
  | `name` | | 菜单里显示的名字；不给就用 id |
  | `version` / `author` / `description` | | 只作说明：设置里那一行说「版本 · 作者 — 说明」 |
  | `extends` | | 以哪套为底：可以是内建之一，也可以是同目录里排在它前面的那套。它的键压在底子之上，没写的键沿用底子（没有底子就沿用基底）。指向不存在的样式会记一条警告并按基底处理 |
  | `style` | | 「元素 → CSS 声明」，`p` / `h2` / `blockquote p` / `pre>code` 等；键位表与 CSS 规矩见 [wechat-style-guide.md](wechat-style-guide.md) |

- **读不出来的文件**会记日志并在设置里列出来（读不了 / 没写 id / id 被占 /
  `extends` 指向不存在），其余样式照常可用：一个坏文件不妨碍整个目录。
- 设置里那一行下面是目录里读到的样式：名字 + `版本 · 作者 — 说明`，末尾一句
  "此外还有应用自带的 N 套"。

## 5 代码落点

| 包 | 内容 |
|---|---|
| `internal/wechat`（新，叶子） | `Theme`（含 version / author / description / extends）、内建主题、读样式目录（重复 id、extends、坏文件）、渲染 `HTML` / `Article` / `Page`、行内样式注入器 |
| `internal/config` | `Preview.Wechat`（样式 id）、`Preview.WechatThemeDir`（样式目录）+ 默认值 + 校验（不认识的 id 落回正常 Markdown；目录只去空白，能否读由 app 说） |
| `internal/view` | 预览右上角的「公众号」菜单与「复制」按钮；`Workspace.WechatPage`（app 造的 web view）、`WechatThemes`、`OnWechatStyle`、`OnWechatCopy`；预览面板在公众号模式下画 web view |
| `internal/app/wechat.go` | 造/填 web view（改样式、换笔记、笔记存盘后刷新）、写剪贴板（`transfer.HTML` + `transfer.Text` 兜底）、状态提示、样式目录的解析与重读 |
| `internal/app/settings.go` | 预览页「公众号」组：样式目录一行（手填 + 浏览…按钮）与目录里读到的样式 |
| `internal/locale` | 菜单、按钮、提示、状态的新键（en + zh-Hans），`go tool mygo generate` |

视图不做 I/O：页面由 app 生成后交给 `WechatPage`，视图只画。

## 6 测试

`internal/wechat` 表驱动：标题/列表/任务框/引用/表格/代码块/图片（内联与隐藏）/
脚注/嵌套，断言落在片段上；主题扩展（缺键落回基底、`pre>code`）；注入器对属性里
带 `>`、自闭合标签、`<style>`/`<script>` 的处理。`internal/config`：新字段的默认值
与校验。预览与剪贴板要真 webview，不进单测（手工验收，macOS 先）。

## 7 风险与没做的

1. 图片：公众号要求素材走它自己的上传，粘进去的内联图多数情况要重新插图（业界
   工具同样如此）。v2 可以考虑调公众号素材接口。
2. 三端 webview（WebKit / WebView2 / GTK）渲染同一份 HTML 会有细微差别；复制的
   是同一份 HTML，与本机预览无关。
3. 外链：公众号只放行白名单域名（含公众号文章），其余 `<a>` 会被去掉链接。
4. 单篇 Markdown 里的巨大图片内联后会让剪贴板很大（同导出内嵌的规矩）。
5. **未做**：代码块语法着色（无分词器）、整篇多笔记合并、公众号草稿箱直发。

## 8 实现状态（v1）

- `internal/wechat`：主题（内建 5 套 + 可指定的样式目录里任意多个 JSON）、渲染、
  行内样式注入、任务框换字符、图片内联。
- 预览右上角：样式菜单 + 复制按钮；整窗预览的顶栏同样有。
- 公众号模式下预览按手机宽度（414，内容左右各留 20）居中显示，窄窗格自动让位；
  `wechatScreen` 是唯一的宽度决策处，有单测钉住。要换手机宽度改一个常量即可
  （将来若要可配置，就是 `preview.wechat_width` 一个字段的事）。
- 右侧竖排按钮（大纲 / 反向链接 / 标签）整体下移到窗格顶行之下
  （`drawerTop` 6 → 40），不再压住预览右上角那两个按钮。
- `preview.wechat` 与 `preview.wechat_theme_dir` 进配置，目录在设置里手填或用
  「浏览…」选；样式文件带 `version` / `author` / `description` / `extends`；
  没有 webview 时预览退回普通 Markdown 并说明。
- **要人工验收**：复制后粘贴进公众号后台的样子（macOS 先）、中文与 emoji、
  表格与代码块、图片、各主题。
