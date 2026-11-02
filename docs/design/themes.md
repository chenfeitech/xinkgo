# 设计：系统多主题（`internal/theme`）

> 状态：**已实现**（2026-11-02），分支 `feature/multi-theme`。
> 一件事两句话说明白：**明暗**是机器（或用户）说了算的，**配色方案**是用户
> 选的；窗口每帧先按明暗取方案的那一半颜色，再画。

## 1 背景与目标

用户口径：系统要支持多主题，准备至少 3 套方案，可以配置切换。

- **目标**：
  - 一套可增删的配色方案（本轮 5 套），每套自带明暗两种取色；
  - 设置里能选、能即时看到（色卡即用该方案自己的颜色画出来）；
  - 改完落 `config.json`，重启保持；
  - 代码高亮预设跟方案走，用户自己挑过的不动。
- **非目标**（本轮不做）：
  - 不做自定义取色器（用户自己配 accent）：先有方案，再有调色；
  - 不做按库记忆主题（一个应用一套）；
  - 不动代码高亮的三套预设本身（`internal/highlight` 保持原样）。

## 2 现状盘点（改动前）

| 能力 | 原来在哪 | 问题 |
|---|---|---|
| 明暗 | `config.Basic.Theme` = `system/light/dark` → `mygo.Theme.SetSource` | 字段名叫 Theme，其实是 appearance |
| 界面颜色 | 完全没有：一律用 `c.Theme()`（MyGo 的 `ui.LightTheme` / `ui.DarkTheme`） | 加一套配色要改遍所有视图 |
| 代码高亮 | `config.Highlight.Theme` = `xink/ocean/solarized` | 与界面配色无关，切主题不动它 |

关键机制（MyGo v0.7.4，`ui/context.go`）：

- 每帧开始 `c.theme = rt.defaultTheme()`（按系统/覆盖给出明暗），所以**每帧都
  能从 `c.Theme().Dark` 读到当前明暗**；
- `c.SetTheme(t)` 在视图函数开头调用即可整窗生效（含菜单、弹层、tooltip）；
- `ui.Themed(c, t)` 只作用于子树——正好用来画"色卡预览"。

## 3 设计

### 3.1 分层与依赖

```
internal/app ──→ internal/theme ──→ github.com/egoist/mygo/ui
      │               ↑
      └──→ internal/config
```

`internal/theme` 是叶子包（只依赖 `ui`），`config` 依赖它取方案名与默认值，
`view` 依赖它上色，`app` 依赖它出下拉与色卡。无环，且 `theme` 不认识 `config`。

### 3.2 数据模型

```go
type Palette struct { Background, Surface, SurfaceHover, SurfacePressed,
	Border, Text, TextMuted, Accent, AccentHover, AccentPressed, AccentText,
	Danger, Warning, Success, Scrollbar ui.Color }

type Scheme struct { ID, Title, Note string; Light, Dark Palette; Code string }
```

- `Light` / `Dark` 是同一方案的两半，`Scheme.Of(dark)` 取其一；
- `Selection` / `Focus` **不在** `Palette` 里：它们是 accent 压透明度得来的
  （明 .25/.55、暗 .4/.6），少 10 个手写字面量，也不会某套方案忘了给；
- `Code` 是配套的代码高亮预设名（`xink/ocean/solarized`），与
  `internal/highlight` 的取值一致；
- 主题在包初始化时一次建好并缓存：`Of()` 每帧返回**同一个指针**，视图每帧
  调用没有分配，MyGo 也能认出"还是那个主题"。

### 3.3 五套方案

| ID | 标题 | 取向 | accent（明 / 暗） | 配套代码预设 |
|---|---|---|---|---|
| `xink` | Xink | 中性石墨（默认，与改动前一致） | `#2563eb` / `#2f6ae0` | xink |
| `ocean` | Ocean | 冷石板 | `#0f766e` / `#2dd4bf` | ocean |
| `moss` | Moss | 灰绿 | `#4d7c0f` / `#84cc16` | solarized |
| `plum` | Plum | 藕紫 | `#7c3aed` / `#a78bfa` | ocean |
| `sepia` | Sepia | 暖纸，长文阅读 | `#b45309` / `#f59e0b` | solarized |

暗色 accent 比明色亮一档（白字压得住），`xink` 的暗色蓝比 MyGo 默认的
`#3b82f6` 深一点：白字在 `#3b82f6` 上只有 3.68:1，达不到 4.5:1。

### 3.4 配置：两个字段，一次迁移

```jsonc
"basic": { "appearance": "system", "theme": "xink" }
```

- `appearance` = `system/light/dark`，接管原 `theme` 的含义；
- `theme` 改为方案 ID；
- `migrate()`：旧文件里 `theme` 是 `system/light/dark` 之一 → 搬到
  `appearance`，`theme` 取默认 `xink`。老配置不失效，也不会互相误读
  （`ocean` 不是明暗值，不会被搬走）；
- `Sanitize()`：`appearance` 只认三个值，`theme` 走 `theme.Normalize`（不认识
  的名字回落默认，而不是画不出颜色）。

### 3.5 生效路径

```
每帧：c.Theme() 是系统明暗  →  theme.Apply(c, cfg.Basic.Theme)
                                  └─ c.SetTheme(scheme.Of(dark))
```

四个窗口的视图函数开头各调一次：主窗口 `view.Workspace.View`、设置
`settingsView`、任务 `tasksView`、历史 `historyView`。

切方案后要重画：主题是视图画上去的，不画帧的窗口不会自己变。`app` 用
`mygo.Theme.SetSource(mygo.Theme.Source())`（告诉它已经知道的事）触发全窗
重画，写进 `redrawWindows()`，`applyConfig()` 与 `toggleAppearance()` 都调。

### 3.6 交互

设置 ▸ Basic ▸ Appearance：

- `Look`：system / light / dark（下拉）；
- `Theme`：下拉（方案 ID）+ **色卡区**：每张卡用 `ui.Themed` 以该方案自己的
  颜色画（四个圆点：ground / surface / text / accent + 标题按钮），点即换，
  选中的那张带 accent 边框且按钮是 PrimaryButton（顺带预览了 accent 上的字）。
- 色卡排成 3 列 `ui.Grid`，宽度自适应：窗口 640 宽（最小尺寸）也不溢出。
- 代码高亮：换方案时跟着换到该方案的预设，**除非**用户自己挑过
  （`followCodeTheme`：旧值不等于旧方案预设 → 视为用户的选择，不动）。

菜单 View ▸ "Turn the light the other way"（`ToggleTheme` 键位）仍然只管明暗。

## 4 测试

| 层 | 覆盖 |
|---|---|
| `internal/theme` | 每套方案两套取色齐备、`Dark` 标志对、代码预设合法；**对比度门禁**（正文 4.5:1、次要字 4:1、accent 上的字 4.5:1）；`Normalize` 未知名回落默认；`Of()` 每帧同指针；`Apply` 明暗都对 |
| `internal/config` | 旧文件 `theme:"dark"` 迁到 `appearance`；新文件 `appearance+theme` 原样带回；unknown / 非法值钳制 |
| `internal/view` | 主窗口按所选方案（且按明暗）上色；未知名不画空；**像素级**：窗口底色就是方案的 `Background` |
| `internal/app` | 点色卡即换方案并落盘；换方案后代码预设跟随；自选预设不被覆盖 |

## 5 风险与取舍

| 取舍 | 理由 |
|---|---|
| 明暗与方案分成两个字段，且改了 `theme` 的含义 | "theme" 在多数应用里指配色方案；迁移 6 行，老文件不失效 |
| 主题在视图里上，而不是 `WindowOptions` | MyGo 没有窗口级主题选项；每帧一次的 `SetTheme` 是文档明确支持的用法，且能跟随系统明暗切换 |
| 加一套默认蓝的改动（`xink` 暗色 `#3b82f6`→`#2f6ae0`） | 对比度，见 §3.3；明色完全不动，白天观感无变化 |
| 代码预设跟方案走 | 不然"换主题"只有边框变色，代码块还是老样子；但用户挑过的不动，尊重显式选择 |
| 不做自定义取色 | 先有可维护的方案表；自定义要 UI、要校验对比度，另开一轮 |

## 6 加一套方案怎么做

1. `internal/theme/theme.go` 的 `schemes` 里加一个 `Scheme`（ID + Title +
   Note + Light/Dark 两套 `Palette` + `Code`）；
2. `go test ./internal/theme/` —— 对比度与完整性门禁会告诉你哪条没过。

设置的下拉与色卡、`config` 的校验与迁移都从这张表读，不用改。
