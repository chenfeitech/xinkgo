# 设计：目录树跟随当前笔记（`Reveal` / `followActiveNote`）

> 状态：**已实现**（2026-11-02），分支 `feature/tree-follows-note`。
> 一句话：**编辑区显示哪篇，目录树就展开到哪篇**——不管这篇是怎么被显示的。

## 1 背景与目标

用户口径：编辑区点击激活 note 的时候，需要展开目录树，定位到相应的目录树中文件。

改动前的缺口：

- `Tree.Reveal` 只在少数入口被显式调用（新建 / 重命名 / 移动 / 粘贴 / 拖入 /
  命令面板），**点标签页、跟双链、重启恢复**都不经过它；
- 即便是显式调用的那些入口，`Reveal` 也只做两件事：把祖先目录标开、把
  `Selected` 指过去。**不滚动**：库一大，那一行在折叠线以下，用户看不到；
- 定位的结果是"树里那一行被点亮"，但"是不是在视口里"没人管。

目标：

1. 任何方式让一篇笔记成为当前笔记，树都展开祖先目录、选中它、并滚到它；
2. 只有"当前笔记变了"才动树，不打扰用户自己展开/折叠的状态；
3. 库外笔记（不在树里）不动树里已有的选择；
4. 面板关着时不被强行打开——用户自己收起来的，不该被点一下标签页弹回来。

非目标（本轮不做）：

- 反向同步（在树里选中目录 -> 编辑区做什么）：`chosen()` 的语义不动；
- 记住/恢复树的滚动位置（见 `XINKGO_FEATURES.md` 的已知缺口）；
- 配置项开关：先给一个确定行为，用户真要关再说。

## 2 现状盘点（改动前）

| 能力 | 原来在哪 | 问题 |
|---|---|---|
| 展开祖先 + 选中 | `Tree.Reveal`（`internal/view/file_tree.go`） | 调用点散在 `internal/app/files.go` 与命令面板，漏了标签页/双链/恢复 |
| 滚动到某一行 | 没有 | `ui.ListState.ScrollIntoView` 一直没用上 |
| 当前笔记在哪 | `Tabs.Active` / `Tabs.ActiveNote()` | 视图里没人把它和 `Tree.Selected` 连起来 |

关键机制（MyGo v0.7.4，`ui/list.go`）：

- `ListState.ScrollIntoView(row)` / `ScrollTo(row, align)` 是**给视图调的**：
  写在 `s.req` 里，布局时才兑现；
- `plan()` 在建列表时就会读 `s.req`，把要显示的那一行周围的行优先建出来，
  所以**在建列表之前请求滚动**，这一帧就能画出很靠下的那一行；
- `s.req` 在每次布局结束时清空，所以"请求"要发生在"画"之前。

## 3 设计

### 3.1 谁来触发：只认一个事实

不逐个入口去补 `Reveal`（补一个漏一个），而是每帧问一句：**当前笔记还是不是
上次让树显示的那篇**。不是，就让树重新定位。

```go
func (w *Workspace) followActiveNote() {
	n := w.Tabs.ActiveNote()
	if n == nil || n.Path == w.treeFollowed || w.Tree.Root == nil {
		return
	}
	w.Tree.Reveal(n.Path)
	w.treeFollowed = n.Path
}
```

- 在 `Workspace.View` 的开头调用，早于 `sidebar` 画树，所以标签页点击引起的
  变化在**下一次画树**时就生效（MyGo 处理完点击会再要一帧，不用额外等）；
- 所有入口自动覆盖：点标签页、点树里的文件、跟 `[[双链]]`、命令面板、搜索面板、
  AI 评审打开笔记、重启恢复上次的标签；
- `treeFollowed` 记的是"上次让树显示的那篇"，不是"当前笔记"：没记就说明还没
  定位过（如刚启动、库还没扫完），下一帧会再问一次。

### 3.2 树这一侧：定位 = 展开 + 选中 + 滚动

`Reveal` 现在返回"库里到底有没有这个文件"，并把要显示的文件记下来：

```go
func (t *Tree) Reveal(path string) bool   // 记 t.revealPath，回 true 表示库里有
func (t *Tree) bringIntoView(rows []row)  // 画之前调：找到行号 -> ScrollIntoView
```

`revealPath` 是**待办**，不是立即滚动：

- `Reveal` 可能在树没被画的时候被调（面板关着、库还没扫完）。留着，等树下一次
  被画时再滚；面板一打开，那篇就正好在那儿；
- 滚动请求必须在 `ui.List` 建之前发出（见 2 的 `plan()`），所以放在 `View` 里
  算完 `rows()` 之后、建列表之前；
- 一帧只兑现一次，兑现完就清空。

### 3.3 边界

| 情况 | 行为 | 理由 |
|---|---|---|
| 库还没扫完（`Root == nil`） | 不记 `treeFollowed`，下一帧再问 | 启动恢复标签早于重扫时也能定位 |
| 笔记不在库里（库外文件） | 记 `treeFollowed`，树保持原样 | 没有行可滚，也不该把已选中的清空；每帧走一遍树去问一个不会有答案的问题不值得 |
| 用户自己展开/折叠 | 不动 | `Reveal` 只把祖先标开，从不折叠 |
| 目录树面板关着 | 只更新展开态与选中，不打开面板 | 面板是用户收起来的；打开时选择已经是对的 |
| 关闭当前笔记后自动切到另一篇 | 跟过去 | `Tabs` 改了 `Active`，对这里就是"当前笔记变了" |

### 3.4 分层

改动全在 `internal/view`：`Tree` 与 `Tabs` 都是窗口自己的状态，`Workspace` 把
两者连起来。`internal/app` 一行未改，也不必改——生命周期层不需要知道"树跟着
编辑区走"这件事。

## 4 实现清单

| 文件 | 改动 |
|---|---|
| `internal/view/file_tree.go` | `Reveal` 返回 `bool` 并记 `revealPath`；新增 `bringIntoView`，在 `View` 里建列表前调 |
| `internal/view/workspace.go` | 新增 `treeFollowed` 与 `followActiveNote()`，`View` 开头调用 |
| `internal/view/file_tree_test.go` | 4 个用例（见下） |

## 5 测试

- `TestTreeReveal_saysWhetherTheVaultHoldsIt`：`Reveal` 的返回值；
- `TestTreeReveal_bringsTheNoteIntoView`：40 篇笔记 + 末尾一个文件夹里的笔记，
  定位后 `ui.Tester` 能找到那一行且它落在窗口高度内（**这条在加滚动前是失败的**）；
- `TestWorkspace_theTreeFollowsTheNoteOnShow`：树上没有的行（文件夹收着）只能
  点到标签页；点完文件夹开、文件被选中；
- `TestWorkspace_aNoteTheVaultDoesNotHoldIsNotShown`：库外笔记不改树的选择。

## 6 验收

见 [ACCEPTANCE.md](../ACCEPTANCE.md) G1。
