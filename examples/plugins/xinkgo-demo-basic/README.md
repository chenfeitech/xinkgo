# Xinkgo Demo: Basic

最小可用的插件示例：命令、Notice、`loadData/saveData`、ribbon、状态栏、
`file-open` 事件、读当前笔记与 Editor 缓冲区。只走 C1/C2 的原生适配器，
兼容等级 **C2 / LevelNative**，非降级，扫描出 7 项能力。

## 安装

把**整个目录**拷到用户级插件目录，目录名必须保持 `xinkgo-demo-basic`
（等于 `manifest.json` 的 `id`）：

```sh
PLUGINS="$HOME/Library/Application Support/Xinkgo/plugins"
mkdir -p "$PLUGINS"
cp -R xinkgo-demo-basic "$PLUGINS/"
```

Windows 用 `%AppData%\Xinkgo\plugins\`。也可以放进库内的
`.obsidian/plugins/` 或 `.xink/plugins/`。拷完在设置窗口 ▸
**Community plugins** ▸ **Installed** 点 **Refresh**，再打开开关。

## 授权

`data`、`commands`、`notice` 启用即有。展开 Grants 勾上：
`ui`、`vault.read`、`editor.read`、`editor.write`。
**先勾授权再启用**，或勾完点一次 **Reload**，否则 ribbon / 状态栏不会补注册。

## 命令

命令面板（`Cmd/Ctrl+P`）里都带 `Xinkgo Demo: Basic:` 前缀：

| 命令 | 作用 |
| --- | --- |
| Say hello | 弹一条 Notice，状态栏计数 +1 |
| Read the active note | 用 `vault.read` 读当前笔记 |
| Count words in the active note | 用 `editor.getValue` 统计词数 |
| Insert today's date at the caret | 用 `replaceSelection` 插入日期 |
| Describe the selection | 显示选区；无选区时命令自动隐藏 |

左侧插件 ribbon 的图标等价于第一条命令。每条命令都会弹 Notice，并写一行日志
（`Xinkgo Demo: Basic | notice | ...`）；计数写进用户数据目录的
`plugins/xinkgo-demo-basic/data.json`。启用时另有一行 `onload` 日志，切换笔记时
`file-open` 事件也记一行。`Describe the selection` 需要选区，无选区时命令隐藏，
自然也不会产生日志。

## 完整文档

安装、授权、日志位置与排错见
[使用手册](../../../docs/plugin-demos.md)，能力对照与开发约束见
[插件示例总览](../README.md)；以本插件为案例的逐段讲解见
[插件开发指南](../../../docs/plugin-development.md)。
