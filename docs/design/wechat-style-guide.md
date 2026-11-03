# 公众号样式开发指引

给要写样式的人（包括以后的自己）：样式是**一个 JSON 文件**，放进一个**自己指定
的目录**就出现在预览右上角的菜单里。不需要改代码、不需要编译。

- 功能设计：[wechat.md](wechat.md)
- 示例样式：[wechat-styles/](wechat-styles/)（可直接复制改用）

## 1 三分钟做出第一套

1. 建一个目录，比如 `~/xinkgo-wechat`。
2. 设置 → 预览 → 公众号 → **样式目录**：填它，或点「浏览…」选它。
3. 把 [wechat-styles/demo-minimal.json](wechat-styles/demo-minimal.json) 复制进去，
   把 `id` 和 `name` 改成自己的。
4. 预览右上角「公众号」菜单里选它 —— 菜单每次打开都会重读目录，所以新文件不用
   重启（换一次样式也就重读一次）。
5. 点旁边的「复制」，粘进公众号后台草稿看真机效果。

## 2 一个样式文件

```json
{
  "id": "mine",
  "name": "我的样式",
  "version": "1.0",
  "author": "谁做的",
  "description": "一句话说明：绿底红字，标题居中",
  "extends": "ink",
  "style": {
    "p": "margin: 0 0 1.4em;",
    "h2": "margin: 1.8em 0 .9em; border-left: 4px solid #c0392b; padding-left: .6em;",
    "pre>code": "font-family: Menlo, Consolas, monospace; color: #d8d8d2;"
  }
}
```

| 键 | 必填 | 说明 |
|---|---|---|
| `id` | ✅ | 配置里存的就是它。两个文件不能同 id（后者会被拒并在设置里说明） |
| `name` | | 菜单里显示的名字；不给就用 `id` |
| `version` / `author` / `description` | | 只是说明：设置里那一行说「版本 · 作者 — 说明」 |
| `extends` | | 以哪套为底：内建之一（`ink` `jade` `lapis` `amber` `rose`），或同目录里排在它前面的那套。它没写的键沿用底子，写了的压在底子之上 |
| `style` | | 「元素 → CSS 声明」，见下节 |

## 3 `style` 里能写哪些键

键就是**标签名**，写 CSS 声明（分号分隔的字符串），会被塞进这个标签的 `style`
属性里。权威清单就是 `internal/wechat/theme.go` 里的 `base`：它给了每个键一套
默认值，你只需要写想改的那几个。

| 键 | 作用的元素 | 备注 |
|---|---|---|
| `body` | 整篇文章的外层 `div` | 字体、字号、行高、字色、字距都在这里 |
| `p` | 段落 | |
| `h1` ~ `h6` | 各级标题 | `h1` 通常当文章标题（居中、更大） |
| `blockquote` | 引用块 | 左边框 + 淡底是常见做法 |
| `blockquote>p` | 引用里的段落 | 见下面的「两级选择器」 |
| `ul` / `ol` / `li` | 列表与列表项 | 原生项目符号，只能调间距与缩进 |
| `pre` | 代码块 | 底色、圆角、内距 |
| `pre>code` | 代码块里的 `code` | 与行内 `code` 分开，可给不同颜色 |
| `code` | 行内代码 | 淡底 + 强调色 |
| `table` / `th` / `td` | 表格 | 用 `border-collapse: collapse` |
| `a` | 链接 | 外链在公众号里可能被去掉 `href` |
| `img` | 图片 | 建议 `max-width: 100%` |
| `hr` | 分隔线 | |
| `strong` / `em` / `del` | 加粗 / 斜体 / 删除线 | |
| `section` | 脚注区 | cmark 把脚注包在 `section` 里 |

规则：

- **两级选择器**：`父>子`（如 `blockquote>p`、`pre>code`）只匹配**直接**父子，
  且只查一层；查不到就退回单键（`p`、`code`）。
- **回退顺序**：自己写的键 → `extends` 那套 → `base`。所以只写几行也能出一套
  完整样式。
- **键名拼错不会报错**，只是不生效（未知的键被忽略）——排错时先怀疑这个。
- 键名是小写标签名；`style` 里没写的元素就没有行内样式，由公众号后台按默认渲染。

## 4 公众号的规矩（CSS 能做与不能做）

编辑器只保留**行内样式**，其余一律丢掉。照这条写就不会白干：

| 做 | 别做 |
|---|---|
| `margin` / `padding` / `border` / `border-radius` | `display: flex` / `grid` |
| `font-*` / `color` / `text-align` / `line-height` / `letter-spacing` | `position` / `float` / `z-index` |
| `background`（纯色） | `background-image`（渐变常丢） |
| 字号用 `px` / `em`（正文 15–16px，行高 1.7–1.85） | CSS 变量、`@media`、`!important` |
| 代码块 `white-space: pre-wrap`（公众号没有横向滚动） | `overflow-x: auto` 当真 |
| 表格 `border-collapse: collapse` | `display: block` 的表格 |
| 中文回退字体：`-apple-system, 'PingFang SC', 'Microsoft YaHei', sans-serif` | 只写一个字体族 |
| 图片 `max-width: 100%; height: auto` | 指望外链图片自动带上（后台要重新上传素材） |

其它两条：**伪元素**（`::before` 那种装饰）和 `<style>` 标签都不会过去；链接
指向不在白名单的域名时，`<a href>` 会被去掉链接只剩文字。

## 5 改 → 看 → 试

1. 改 JSON 文件。
2. 预览右上角菜单里**重选一次**样式（重读目录），或等笔记存盘（预览会自动重排）。
3. 预览里看：它按**手机宽度 414** 显示，内容左右各留 20 —— 断行与图片宽度就是
   手机上的样子。
4. 点「复制」，粘进公众号后台草稿：这是唯一能 100% 确认的地方。
5. 想验证回退：临时把某个键删掉，看它是否落到 `extends` / 基底。

## 6 排错

| 症状 | 原因 | 怎么查 |
|---|---|---|
| 菜单里没有这套样式 | 文件不是 `.json`；JSON 写坏了；没写 `id`；`id` 与别的文件重了 | 设置 → 预览 → 公众号 那一组下面会用红字列出「哪个文件、什么原因」；日志里也有一条 |
| 目录设了却一个样式都没有 | 目录路径不对（留空则用应用自己的目录） | 设置里那行的占位文字写着当前生效的目录 |
| 选了样式没变化 | 键名拼错（未知键被忽略）；改了文件但没重选样式 | 先在菜单里重选一次；再对照第 3 节的表 |
| 只有部分生效 | `父>子` 只匹配直接父子一层 | 比如 `ul>li` 有效，`blockquote>ul>li` 不会命中 |
| `extends` 没起作用 | 指向的 id 不是内建、也不在同目录且在它之前 | 设置里的红字会说「extends a style that is not there」 |
| 图片不显示 | 公众号要求素材走它自己的上传；预览里是 base64 内联 | 后台里重新插图 |
| 中文变成了别的字体 | 字体族只写了一个，且目标机器上没有 | 按第 4 节写完整回退列表 |
| 粘到别处没样式 | 目标不支持 `text/html`（剪贴板另有纯文本兜底） | 换支持富文本的编辑器 |

## 7 示例样式

[wechat-styles/](wechat-styles/) 下三套，复制即用，也是三种写法的样板：

| 文件 | 演示什么 |
|---|---|
| `demo-minimal.json` | 最小可用：只写 `body` / `p` / `h2` 三个键，其余沿用基底 |
| `demo-magazine.json` | 完整一套：两端对齐 + 首行缩进、标题居中带细线、表格斑马底 |
| `demo-over-ink.json` | `extends`：整套沿用内建「墨黑」，只把代码块换成深色 |

改的时候保留 `id` 唯一、`name` 好认，其余随便改。

## 8 想做成内建样式（改代码）

用户样式够用就别进代码。要进的话：

1. `internal/wechat/theme.go`：在 `builtin` 里加一项（`ID` / `Name` / `Version` /
   `Author` / `Description` / `Style`），只用内建常量当 `ID`，只写要改的键。
2. 数组顺序 = 菜单里的顺序（内建排在用户样式之前）。
3. 名字是样式自己的名字（同 `theme.Scheme.Title` 的规矩），**不进 locale catalog**。
4. 加测试：`internal/wechat/wechat_test.go`（照 `TestArticle_*` 的写法，断言落在
   片段上）。
5. 跑 `scripts/check.sh`，再人眼验一遍预览与粘贴效果。

## 9 参考

- 功能与边界：[wechat.md](wechat.md)（第 3 节是"与预览对齐"的取舍）
- 键的默认值：`internal/wechat/theme.go` 的 `base`
- 注入器的行为：`internal/wechat/style.go`（标签扫描、两级选择器、转义）
- 验收项：[../ACCEPTANCE.md](../ACCEPTANCE.md)
