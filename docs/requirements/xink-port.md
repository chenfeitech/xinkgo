# 需求：把 xink 的全部功能复刻到 xinkgo

> 复刻对象：`/Users/helightxu/chenfei/rustgpui`（Rust + GPUI 的本地优先 Markdown
> 笔记应用 **xink**）。本文件是复刻需求的唯一口径，证据列到 xink 的
> `文件:行号`，冲突时**以 xink 代码为准**（xink 自己的文档有多处过时，见 §6）。
>
> 状态：**G1 已过**（拍板项已确认，见 §7）。

## 0 已确认的选型（用户拍板，2026-10-30）

| 项 | 结论 |
|---|---|
| UI 形态 | **原生 UI**：全部用 MyGo `ui` 包以 Go 绘制，保持单二进制、低内存，与现有骨架一致 |
| AI 功能 | **全量复刻** R-10（10 项：多 Provider、流式、对话条、agent loop、Skills、库级批量任务） |
| 交付节奏 | **一次性做完 P0（R-1~R-7）再验收**，中途不停 |
| 数据格式 | **与 xink 兼容**：`vaults.json` / `open_notes.json` / `layout.json` / `config.json` 及相同结构，xink 用户可直接切换 |

由此产生的技术约束（设计文档必须回应）：

- Markdown 渲染、富文本、滚动同步、补全浮层、抽屉、命令面板**全部自研**（用
  `ui` 包的绘制原语，不引入 webview）。
- 文件名与 JSON 结构照抄 xink（含旧版 `vault.json` 迁移、旧高亮主题名迁移）。
- AI 为独立阶段（P2），在 P0/P1 之后做，但数据格式与目录约定（`.xink`、Skills
  发现路径）从一开始就留好。

## 1 目标与非目标

### 1.1 目标

在 xinkgo（Go + MyGo v0.7.3）上复刻 xink 的全部功能，包括：库与文件树、
笔记增删改移、多标签编辑与自动保存、Markdown 实时预览、会话恢复、外部变更
监听、索引与双链/反链/标签/搜索/大纲/命令面板、主题与设置、AI 协助。

行为对齐以 xink 的**代码**为准，包括其边界规则（同一文件只有一个标签、
有历史打开记录就不显示空白页、磁盘与内存一致时一个字节都不回填等）。

### 1.2 非目标（明确不做，照抄 xink 的非目标）

- 跨库检索、跨库双链（`xink/docs/REQUIREMENTS.md:83`）
- 最近文件列表（R4.3 已移除，`xink/docs/REQUIREMENTS.md:71`）
- 关系图谱、插件系统、日记模板、PDF/HTML 导出、WYSIWYG、云同步、协作
  （`xink/docs/REQUIREMENTS.md:91-98`）
- 选中文字的前景/背景色与字体（R2.7–R2.9 已撤回）
- PDF 预览（xink 未立项）

### 1.3 平台

与 [HARNESS.md](../HARNESS.md) 一致：P0 macOS、P1 Windows/amd64、P2 其余。

---

## 2 功能需求

编号规则：`R-<组>.<序号>`，每条给出验收标准与 xink 证据。

### R-1 库（Vault）与文件树（18 项）

| ID | 需求 | 验收标准 | 证据 |
|---|---|---|---|
| R-1.1 | 系统目录对话框打开文件夹作为库（Cmd-O） | 选中目录后成为当前库并立即扫描 | `src/ui/file_tree.rs:152-177` |
| R-1.2 | 多库注册表：新增 / 切换 / 重新定位 / 改名 / 删除 | 注册表落盘，切换后标签按库清理 | `src/ui/vault_manager.rs:339-468` |
| R-1.3 | 库路径健康校验 | 区分 Ok/Missing/NotDir/NoPermission/Duplicate/Nested 六种状态并提示 | `src/core/vault.rs:198-232,274-308` |
| R-1.4 | 启动解析当前库，失效回落到第一个可用库 | 只提示，不阻塞启动 | `src/core/vault.rs:354-398` |
| R-1.5 | 上次库的启动恢复 | 重启后回到上次的库 | `src/ui/file_tree.rs:418-439` |
| R-1.6 | 递归全量扫描（**非懒加载**） | 一次性建完整树 | `src/core/vault.rs:41-43,134-180` |
| R-1.7 | 固定忽略 `.git`/`node_modules`/`target`/`.DS_Store`/`*.md.tmp`/`.xink` | 这些不进树 | `src/core/vault.rs:112-120` |
| R-1.8 | 隐藏文件开关 `show_all_files`，**默认 true（显示）** | 默认显示隐藏文件 | `src/core/vault.rs:119`；`src/core/config.rs:413` |
| R-1.9 | 默认只显示"可编辑文本 + 图片"，开 show_all 后全显示 | 两种模式切换正确 | `src/core/vault.rs:151-164` |
| R-1.10 | 排序：目录在前，同层按名称小写 | 稳定排序 | `src/core/vault.rs:168-172` |
| R-1.11 | 文件类型图标（10 类） | Dir/Markdown/Image/Code/Data/Archive/Spreadsheet/Audio/Video/Other | `src/core/media.rs:64-124` |
| R-1.12 | 增量刷新保留展开态/选中/滚动 | 刷新后不丢失状态 | `src/ui/file_tree.rs:52,567-576` |
| R-1.13 | 手动刷新按钮，300ms 连点抑制 | 连点只刷一次 | `src/ui/file_tree.rs:606-615` |
| R-1.14 | 目录树跟随活动笔记：展开 + 选中 + 滚动 | 切标签时树跟随（含 pending_reveal 补做） | `src/ui/file_tree.rs:103-133` |
| R-1.15 | 外部文件拖入导入（复制/移动） | 文件进入库目录 | `src/ui/file_tree.rs:621-664` |
| R-1.16 | 粘贴（含剪贴板图片落盘） | 图片写入库并可用 | `src/ui/file_tree.rs:665-733` |
| R-1.17 | 树内拖拽移动 | 移动后树与索引同步 | `src/ui/file_tree.rs:734-778` |
| R-1.18 | 重名自动递增 `-N`，保留扩展名 | `a.md` 已存在则 `a-1.md` | `src/core/fs.rs:28-49` |

### R-2 笔记增删改移（7 项）

| ID | 需求 | 验收标准 | 证据 |
|---|---|---|---|
| R-2.1 | 新建笔记（自动补 `.md`，Cmd-N） | 只给 Markdown 补扩展名 | `src/core/vault.rs:46-51` |
| R-2.2 | 新建文件夹 | | `src/core/vault.rs:53-58` |
| R-2.3 | 删除（含非空目录）+ 二次确认 | 拒绝删根目录与库外路径 | `src/ui/file_tree.rs:235-265`；`src/core/vault.rs:61-71` |
| R-2.4 | 重命名（F2 / 右键），同名（忽略大小写）视为未修改 | 直接忽略，不报错 | `src/ui/file_tree.rs:292-378,352-354` |
| R-2.5 | 移动到目录（对话框选目标） | | `src/ui/file_tree.rs:390-415` |
| R-2.6 | 跨设备移动：复制 + 校验大小 + 删源，不一致保留源 | 数据不丢 | `src/core/fs.rs:115-146` |
| R-2.7 | **关闭未保存标签无确认弹窗**（代码实况） | 直接关闭（300ms 自动保存通常已落盘） | `src/ui/editor.rs:2813-2835` |

### R-3 多标签（11 项）

| ID | 需求 | 验收标准 | 证据 |
|---|---|---|---|
| R-3.1 | 打开笔记**唯一入口** | 所有入口复用同一函数（Go 侧：`internal/session` 的单一入口） | `src/ui/tabs.rs:181-261` |
| R-3.2 | 同一文件只有一个标签，键为 canonicalize 后的绝对路径 | 重复打开只激活 | `src/core/session.rs:18-20`；`src/ui/tabs.rs:205-217` |
| R-3.3 | 命中已开标签时：只选中 + 聚焦 + 更新 active | **不新建、不重读盘、不重置光标滚动、不改顺序** | `src/ui/tabs.rs:203-217` |
| R-3.4 | 新标签优先认领空白页，无空白页才新建 | | `src/ui/tabs.rs:219-230` |
| R-3.5 | 空白页不变式：无笔记恰好一个、有笔记零个 | 单点维护 | `src/app.rs:399-441,447-486` |
| R-3.6 | 打开/关闭/切换/清空 → 立即落盘会话文件 | | `src/ui/tabs.rs:283-293` |
| R-3.7 | 关闭最后一个标签 → 变回空白页 | | `src/ui/editor.rs:2813-2835` |
| R-3.8 | 标签标题 = 文件名 + 未保存 `●` + 关闭按钮 | | `src/ui/editor.rs:2776-2805` |
| R-3.9 | 跳行打开（大纲/搜索/反链共用，不新建标签） | | `src/ui/tabs.rs:189,272-278` |
| R-3.10 | 切换库时关闭不属于该库的标签并写回 | | `src/ui/tabs.rs:124-170` |
| R-3.11 | 顺序与查找分离：`order` 是唯一顺序来源 | 落盘顺序不漂移 | `src/ui/tabs.rs:20-31` |

### R-4 编辑与自动保存（15 项）

| ID | 需求 | 验收标准 | 证据 |
|---|---|---|---|
| R-4.1 | 防抖自动保存，默认 **300ms**（可配 200–60000） | 停止输入 300ms 后落盘 | `src/ui/editor.rs:1398-1418`；`src/core/config.rs:423,1198` |
| R-4.2 | Cmd+S 立即保存 | | `src/ui/editor.rs:2363-2366` |
| R-4.3 | 原子写盘：同目录临时文件 + rename，失败清临时文件 | 断电不产生半截文件 | `src/core/note.rs:294-308` |
| R-4.4 | 保存后刷新索引（内容变更不产生结构事件） | 保存后立即能被搜索到 | `src/ui/editor.rs:2353-2357` |
| R-4.5 | 保存 / 预览 / 装饰共用同一防抖 | | `src/ui/editor.rs:1420,1526` |
| R-4.6 | 内容版本号，用于跳过无意义重算 | | `src/core/note.rs:57-60,113` |
| R-4.7 | Markdown 工具栏 13 个按钮 | 粗/斜/删除线/行内码/标题/无序/有序/任务/引用/代码块/链接/图片/表格/分隔线 | `src/ui/editor.rs:2487-2570` |
| R-4.8 | 选中包裹 or 光标插入，共用同一实现 | | `src/ui/editor.rs:2679,2979` |
| R-4.9 | 软换行（默认开）、行号（默认关）、tab_size/use_tabs | | `src/core/config.rs:425-428` |
| R-4.10 | 编辑区 Markdown 装饰（行首标记淡化、链接文字） | | `src/core/markdown.rs:832-940` |
| R-4.11 | 数据文件（JSON/YAML/TOML）校验诊断 | 错误处有提示 | `src/ui/editor.rs:485-503` |
| R-4.12 | 数据文件工具栏：语言徽标 + 校验状态，隐藏 Markdown 按钮 | | `src/ui/editor.rs:2469-2471` |
| R-4.13 | 自动保存关闭时**仍刷新预览** | | `src/ui/editor.rs:1398-1420` |
| R-4.14 | AI 评审未收尾时拒绝保存并提示（若做 AI） | | `src/ui/editor.rs:2334-2343` |
| R-4.15 | 数据文件预览：格式化围栏代码块，超 `max_format_bytes` 跳过 | | `src/ui/editor.rs:2852-2864` |

### R-5 Markdown 实时预览（16 项）

| ID | 需求 | 验收标准 | 证据 |
|---|---|---|---|
| R-5.1 | 每篇笔记独立预览状态，切标签整体换引用（不重解析） | | `src/ui/editor.rs:64-65,549-561` |
| R-5.2 | 防抖后增量更新（路径+版本+图片选项+数据配置四重判重） | | `src/ui/editor.rs:410-476` |
| R-5.3 | 畸形 Markdown 防御性预处理，防解析器崩溃 | fuzz 输入不崩 | `src/core/markdown.rs:47-87` |
| R-5.4 | `[[双链]]` 改写为可点链接（只改预览副本，磁盘不动） | | `src/ui/editor.rs:583`；`src/core/markdown.rs:702` |
| R-5.5 | 本地相对路径图片可显示 | Go 版可走 `file://` 或 data URL，见 §7 | `src/core/markdown.rs:143-169,209-253` |
| R-5.6 | 图片显示 FitWidth/Hidden + 宽度百分比 20–100 | | `src/core/config.rs:61-74,217` |
| R-5.7 | 代码块语法高亮，编辑区与预览区**同源** | | `src/ui/preview.rs:751-761`；`src/core/theme.rs:22-35` |
| R-5.8 | 明暗主题 System/Light/Dark，高亮主题自动取对应变体 | | `src/core/theme.rs:25-34`；`src/app.rs:323-382` |
| R-5.9 | 滚动同步三档：比例 / 块锚点 / 停止后对齐 | | `src/core/config.rs:89-98`；`src/core/sync.rs:1-161` |
| R-5.10 | 方向锁 + 回显吸收 + 手势静默窗口（防互推） | 拖动不抖动 | `src/core/sync.rs:166-294` |
| R-5.11 | 版面密度三档 + 正文最大宽度 + 预览字体/行高/字重 | | `src/core/config.rs:236-288`；`src/ui/preview.rs:687-735` |
| R-5.12 | 图片点击 → 复用预览面板显示大图（不新开窗口） | | `src/ui/preview.rs:846-860` |
| R-5.13 | 链接点击：库内 `.md` → 打开笔记；否则系统浏览器 | | `src/ui/preview.rs:853-859`；`src/core/config.rs:100-109` |
| R-5.14 | 预览区"向左伸展" zoom 按钮 | | `src/ui/preview.rs:586-627` |
| R-5.15 | 预置高亮主题 3 套（xink/ocean/solarized）+ 旧值迁移 | | `src/core/theme.rs:22-59` |
| R-5.16 | 滚动同步调试数字（可配） | | `src/core/config.rs:228` |

### R-6 会话恢复（9 项）

| ID | 需求 | 验收标准 | 证据 |
|---|---|---|---|
| R-6.1 | 布局持久化文件，schema 版本不符 → 重建默认布局 | | `src/app.rs:486-492,547-567` |
| R-6.2 | 恢复顺序固定：布局 → 标签 → 激活 → 空白页校正 | 顺序不可颠倒 | `src/app.rs:569-572` |
| R-6.3 | 只恢复属于当前库的标签 | | `src/app.rs:645-655` |
| R-6.4 | 失效路径过滤、保序去重、active 回落最后成功项 | | `src/core/session.rs:25-50` |
| R-6.5 | 恢复走与手动打开**完全相同**的单一入口 | 不绕过判重与落盘 | `src/app.rs:663-670` |
| R-6.6 | 全部恢复完再激活 active（避免预览反复重建） | | `src/app.rs:672-679` |
| R-6.7 | 恢复后写回一次（自愈，剔除失效条目） | | `src/app.rs:681-682` |
| R-6.8 | 退出前兜底保存布局 + 标签 | | `src/app.rs:589-602` |
| R-6.9 | 会话文件存**路径**而非下标；损坏视为空记录；恢复失败只记日志不阻塞 | | `src/state/workspace.rs:136-145,188-206` |

### R-7 外部变更监听（9 项）

| ID | 需求 | 验收标准 | 证据 |
|---|---|---|---|
| R-7.1 | 递归监听，事件驱动（不轮询） | Go 侧：`fsnotify` 或 MyGo 的 watch | `src/core/watcher.rs:64-97` |
| R-7.2 | 事件分类：Create/Remove/Modify(Name)=Renamed/Modify=Modified | | `src/core/watcher.rs:35-46` |
| R-7.3 | 只有结构变化才重扫树，内容写入不重扫 | | `src/ui/file_tree.rs:492-526` |
| R-7.4 | 结构变化时同步索引 | | `src/ui/file_tree.rs:517-524` |
| R-7.5 | 已打开笔记：无未保存改动 → 自动重载；有改动 → 保留并提示 | 三态正确 | `src/ui/editor.rs:2374-2410` |
| R-7.6 | 内容与磁盘一致时**一个字节都不回填**（否则光标被重置） | 光标不动 | `src/core/note.rs:144-158` |
| R-7.7 | 打字期间延后回填（800ms 保护窗口），只排一次 | | `src/ui/editor.rs:2390-2396,2445-2459` |
| R-7.8 | 回填保住光标与滚动位置 | | `src/ui/editor.rs:2418-2442` |
| R-7.9 | 忽略应用自身临时文件 `*.md.tmp` | | `src/core/watcher.rs:127-129` |

### R-8 索引 / 双链 / 反链 / 标签 / 搜索 / 大纲 / 命令面板（22 项）

| ID | 需求 | 验收标准 | 证据 |
|---|---|---|---|
| R-8.1 | 一次扫描提取 headings/links/wikilinks/tags（带行号），代码块与行内代码内不解析 | | `src/core/markdown.rs:399-443,447-700` |
| R-8.2 | 全库索引：主表 + 5 张派生表，detach/attach 增量维护 | 增量 == 全量 | `src/core/index.rs:73-96,256-347` |
| R-8.3 | 索引**不落盘**，启动/切库后台重建 | | `src/app.rs:184-212` |
| R-8.4 | 双链解析顺序：相对路径 → 文件名 → 标题，大小写不敏感，歧义标 Ambiguous | | `src/core/index.rs:445-480` |
| R-8.5 | 双链语法 `目标`、`目标\|别名`、`目标#锚点` | | `src/core/markdown.rs:582-626` |
| R-8.6 | 反向链接（排除自引，按来源+行号排序） | | `src/core/index.rs:179-199` |
| R-8.7 | 标签聚合（数量降序），点击展开带该标签的笔记，大小写归一 | | `src/core/index.rs:202-222` |
| R-8.8 | 标签来源：正文 `#tag` + frontmatter `tags: [a,b]`，中文可用，不误判 `#top`/`#fff` | | `src/core/markdown.rs:628-699` |
| R-8.9 | `[[` 输入补全浮层，模糊排序，最多 30 条 | | `src/ui/completion.rs:66-91` |
| R-8.10 | 写入双链时库内一律写**相对路径** | | `src/ui/completion.rs:194-205` |
| R-8.11 | 命令面板（Cmd+P）：文件名模糊搜索，前 50 条，命中位置高亮，后台扫描 | | `src/ui/command.rs:108-135,359-387` |
| R-8.12 | 全文搜索（Cmd+Shift+F）：防抖 200ms，后台检索，上限 100 条，点击跳行 | | `src/ui/search.rs:22-24,66-97,160-163` |
| R-8.13 | 检索：逐行、大小写不敏感、跳 >4MB 文件、单行上下文截断 160 字符 | | `src/core/search.rs:12-14,30-77` |
| R-8.14 | 大纲：实时解析（不读索引），层级缩进 1–6，点击跳行 | | `src/ui/outline.rs:21,62-72,168-193` |
| R-8.15 | 右侧按钮条 + 半透明抽屉承载 大纲/反链/标签（同项再点收起） | | `src/ui/rail.rs:47-57`；`src/ui/drawer.rs:100-127` |
| R-8.16 | 主题切换快捷键，从 System 起手第一下脱离跟随系统 | | `src/core/config.rs:37-44` |
| R-8.17 | 配置修改**唯一入口**：改 → 校正 → 落盘 → 应用主题 → 刷新（即时生效） | | `src/app.rs:245-268` |
| R-8.18 | 配置越界钳制 + 键位垃圾清理 | | `src/core/config.rs:1239-1293` |
| R-8.19 | 键位可覆盖（17 个动作），冲突保留靠前动作，改键位需重启 | | `src/core/keymap.rs:37-174,226-262` |
| R-8.20 | 设置窗口 + 库管理窗口（单例）+ 配置导入导出 | | `src/app.rs:688-690`；`src/core/config.rs:1318-1324` |
| R-8.21 | 单实例保护（pid 文件 + 存活检查） | 二次启动不新开进程 | `src/state/single_instance.rs:12-40` |
| R-8.22 | 三栏默认布局：文件树 / 编辑区 / 预览区；右侧面板走浮层不占布局宽度 | | `src/app.rs:736-778` |

### R-9 持久化与兼容（8 项）

| ID | 需求 | 验收标准 | 证据 |
|---|---|---|---|
| R-9.1 | 配置 `config.json`：`{basic, editor, preview, highlight, ai}`，子结构缺失取默认 | | `src/core/config.rs:344-353` |
| R-9.2 | 布局 `layout.json`（带 version） | | `src/state/workspace.rs:24-26` |
| R-9.3 | 库注册表 `vaults.json`：`{vaults:[{id,name,path}], active}` | | `src/core/vault.rs:189-196,323-329` |
| R-9.4 | 会话 `open_notes.json`：`{paths:[...], active}`，**只存路径不存正文** | | `src/state/workspace.rs:139-145` |
| R-9.5 | 旧版 `vault.json` 自动迁移一次 | | `src/state/workspace.rs:74-101` |
| R-9.6 | 旧高亮主题名迁移：`base16-ocean*`→ocean、`Solarized*`→solarized、其余→xink | | `src/core/theme.rs:48-59` |
| R-9.7 | 未知枚举值不能导致整个配置解析失败（`#[serde(other)]` 兜底） | | `src/core/config.rs:67-73` |
| R-9.8 | 关键 JSON 用原子写（`.json.tmp` + rename） | | `src/state/workspace.rs:103-125` |

### R-10 AI 协助（10 项，**是否复刻待 §7 拍板**）

| ID | 需求 | 证据 |
|---|---|---|
| R-10.1 | 多 Provider（Ollama native + OpenAI 兼容）、统一请求、重试与备用模型 | `src/core/ai/mod.rs:14-31,298-403` |
| R-10.2 | 流式事件 Delta/Thinking/ToolCall/Truncated | `src/core/ai/mod.rs:149-176` |
| R-10.3 | 编辑器下方 AI 对话条（按笔记归属，关闭只隐藏保留历史） | `src/ui/editor.rs:1667-1690` |
| R-10.4 | AI 会话持久化（按笔记路径，30 会话 / 200 块裁剪） | `src/state/ai_sessions.rs:22-24,99-115` |
| R-10.5 | AI 修订在编辑区内逐块接受/拒绝（真插入式 + 行染色 + 卡片） | `src/ui/editor.rs:1746-1812` |
| R-10.6 | 只读工具集 + agent loop + 结果校验 + 快照轮转 | `src/core/ai/{tools,agent_loop,verify,snapshot}.rs` |
| R-10.7 | Skills：`SKILL.md` 发现（库内 `.xink` + 用户级），命令面板入口 | `src/core/ai/skills.rs` |
| R-10.8 | 库级批量任务（加/改/删标签）：只读计划 + 批量原子写 + 中断恢复 | `src/core/ai/tasks.rs` |
| R-10.9 | 模型能力清单（静态推断 + 手动改 + 探针） | `src/core/ai/{capability,probe}.rs` |
| R-10.10 | AI 目录 `.xink` 恒跳过：不进树、不进索引、不进命令面板 | `src/core/media.rs:21-30` |

---

## 3 边界条件（沿用 xink 的规则，Go 版必须同样处理）

| 场景 | 要求 |
|---|---|
| 磁盘与内存内容一致 | 一个字节都不回填（否则光标被重置到 0） |
| 打字期间外部变更 | 延后 800ms，只排一次延后任务 |
| 半截文件 / 崩溃残留 | 原子写 + 临时文件清理 |
| 大文件 | 搜索跳 >4MB；数据文件预览超阈值跳过格式化 |
| 非 UTF-8 | 读盘失败要有明确错误，不 panic |
| 跨设备移动 | 复制 + 校验大小 + 删源，不一致保留源 |
| 库失效 / 缺失 / 无权限 / 嵌套 | 六种状态可辨，启动回落不阻塞 |
| 并发写（外部程序同时改） | 有未保存改动时不覆盖，提示用户 |
| 重复打开同一文件 | 只激活已有标签 |
| 会话文件损坏 | 视为空记录，不弹窗、不阻塞启动 |

## 4 验收标准（全局）

1. 上述每条 R-n 有对应的自动化测试或人工验收项（人工项登记到
   `docs/ACCEPTANCE.md`）。
2. 领域层（库扫描、索引、双链解析、标签、搜索、会话规则、同步锚点、
   原子写）必须有 Go 单元测试，参照 xink 的 `tests/core/`（约 380 个用例）。
3. `make check` 全绿：格式 + vet + 测试 + 全平台编译（macOS 先）。
4. 人工验收：在 macOS 上真的打开一个库，走完"建笔记 → 编辑 → 自动保存 →
   预览 → 切标签 → 重启恢复 → 外部改文件 → 搜索 → 双链跳转"主链路。

## 5 与 xink 的差异（Go/MyGo 带来的必然差异，需记录在档）

- 文件名与目录：xink 数据目录 `~/Library/Application Support/xink/`；
  xinkgo 用 MyGo 的 `PathUserData`（`~/Library/Application Support/Xinkgo/`）。
  是否复用 xink 的文件名见 §7.4。
- 编辑器内核：xink 用 gpui 的 `Editor`（tree-sitter）；Go 侧取决于 §7.1 选型。
- 图片 URL：xink 因框架限制转 data URL；Go 侧可选 `file://` 或 data URL。
- 布局：xink 用 gpui-component 的 `DockArea`；Go 侧用 MyGo 的布局能力或前端实现。

## 6 已知文档与代码冲突（以代码为准，避免照抄过时文档）

1. R1.2"懒加载子目录"：xink 实际是**全量扫描** — `src/core/vault.rs:37-40`
2. R1.7"默认忽略隐藏文件"：实际 `show_all_files: true`（默认显示）— `src/core/config.rs:412-413`
3. R2.3"关闭未保存标签有提示"：实际**无弹窗** — `src/ui/editor.rs:2813-2835`
4. R4.2"记住窗口大小"：只持久化了 DockArea 布局
5. 不存在 `workspace.json`，实际是 `vaults.json` + `open_notes.json` + `layout.json`

## 7 待用户拍板（G1 未过：以下 4 项确认前不动 `internal/`）

### 7.1 UI 形态（决定工作量与最终形态）

- **A. Web 前端（webview）**：三栏 UI、Markdown 渲染、命令面板、抽屉、补全浮层
  用 Web 技术做，Go 只做领域层并通过 typed IPC 提供接口。复用现成生态
  （markdown 渲染、代码高亮、编辑器），117 项功能里 UI 密集部分成本最低。
  代价：引入 Node/Bun 前端工具链、内存占用更高、应用依赖系统 webview。
- **B. 原生 UI（MyGo `ui` 包）**：全部 Go 绘制，保持单二进制与低内存，与我方
  现有骨架一致。代价：Markdown 渲染、富文本、滚动同步、补全浮层、抽屉等
  UI 细节需按 MyGo 范式自研，工作量显著更大，且部分细节（预览密度、
  滚动同步三档）受组件能力约束。
- **C. 混合**：编辑区与预览区走 Web，其余（文件树、设置、命令面板）原生。

### 7.2 AI 功能（R-10，10 项）是否复刻

- 全量复刻 / 只做对话条（R-10.2、R-10.3、R-10.4）/ 暂不做（其余先完成）

### 7.3 交付节奏

- 一次性做完 P0（R-1~R-7）再进入 P1
- 分模块逐个交付，每完成一组即验收
- 先出"能跑通主链路"的纵向切片，再补全细节

### 7.4 数据格式

- 与 xink 完全兼容（同样的文件名与结构，xink 用户可直接切换）
- 重新设计（只用 MyGo 的目录约定，文件名自定）
