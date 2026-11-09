# Xinkgo 插件体系与官方插件库

日期：2026-11-09。状态：**设计 v1，P0（契约与离线门禁）与 P1（运行时接入）已落地**。

本文把目标从「Obsidian 兼容宿主」调整为「以 Xinkgo 自身架构为目标的插件体系」。
参考 Obsidian 的插件形态可以，但**我们做的插件必须能在 Xinkgo 里完好地运行**；
任何只是兼容占位、加载得到但功能残缺的能力，都不写进我们的插件。

相关文档：[obsidian-plugins.md](obsidian-plugins.md)（兼容宿主的调研与 P0 落点）、
[obsidian-core-plugins.md](obsidian-core-plugins.md)（内置核心对照）、
[../plugin-development.md](../plugin-development.md)（作者指南）、
[../plugin-demos.md](../plugin-demos.md)（使用手册）。

## 1 排查：现有插件体系

### 1.1 机制清单

| 组件 | 位置 | 职责 | 是否以 Xinkgo 为目标 |
| --- | --- | --- | --- |
| 内置核心注册表 | `internal/plugin/core.go` | 13 项 Go 原生能力，开关写进 `plugins.json` | 是 |
| 包发现 | `internal/app/plugin_discovery.go` | 三个目录（用户级 / `.obsidian` / `.xink`）扫 `manifest.json` + `main.js` | 兼容导向 |
| 静态扫描 | `internal/plugin/scan.go`、`plugin.go` | 文本匹配 14 项能力、C0~C4 等级、Imports/Issues | 兼容导向 |
| 兼容运行时 | `resources/obsidian-shim/` | `require("obsidian")` / `require("xinkgo")` 双入口别名、DOM helpers、RPC 桥 | 兼容导向 |
| 运行时桥 | `internal/app/plugin_rpc.go`、`plugin_host.go` | `/rpc` 方法表、能力授权、隔离 WebView 页 | 混合 |
| 插件管理器 | `internal/app/plugin_settings.go` | Installed / Core / Browse 三页、Grants 勾选 | 兼容导向 |
| 社区目录 | `internal/app/plugin_catalog.go` | 拉取 Obsidian 社区 catalog + stats | 兼容导向 |
| 兼容清单 | `resources/obsidian-plugins/compatible.json` | 128 个「已检查」条目 | 兼容导向 |
| 官方插件 | `examples/plugins/` | 2 个我们写的 demo | 是（已按真实能力裁剪） |

内置核心注册表是唯一从一开始就按 Xinkgo 架构设计的部分；其余组件都以
「让 Obsidian 插件尽量加载」为目标，能力面围绕 `obsidian.d.ts` 而不是
Xinkgo 的真实功能来划。

### 1.2 所有可运行的插件

用宿主自己的 `ParseManifest` + `ScanPackage` 扫了一遍（2026-11-09）：

| 插件 | 来源 | 等级 | 降级 | 能力 | 完好运行 |
| --- | --- | --- | --- | --- | --- |
| 13 项内置核心 | Go 原生，编译进 app | — | — | 原生 UI，不需要授权 | 是 |
| `xinkgo-demo-basic` | 我们写的 demo | C2 | 否 | 7 项，全为真实能力 | 是 |
| `xinkgo-demo-full` | 我们写的 demo | C3 | 是 | 11 项，全为真实能力 | 是（C3 只是视图在插件页） |
| `highlightr-plugin` | 本机已装的社区插件 | C3 | 是 | `commands data dom editor.read editor.write notice ui` | **否**：6 处 `Menu(`、`addSettingTab` / `PluginSettingTab` 都是空壳 |
| `obsidian-markdown-formatting-assistant-plugin` | 本机已装的社区插件 | C3 | 是 | 同上 | **否**：11 处 `Modal`、`SuggestModal`、`registerView`、`addSettingTab` 混用，弹层是 no-op |
| `obsidian-sort-and-permute-lines` | 本机已装的社区插件 | **C2** | **否** | `commands editor.read metadata` | **否**：3 处 `getFileCache`，宿主返回 `{}`，标题层级排序会算错 |
| 兼容清单 128 项 | 随包分发 | C4 77 / C3 43 / C2 6 / C1 2 | 120 项降级 | 由静态扫描推断 | 不承诺 |

本机 `plugins.json` 还残留 `obsidian-git`、`ollama-chat` 两个 enabled 条目，
但目录里没有对应包，属于已卸载后的状态残留，不影响排查结论。

### 1.3 根本问题

1. **等级模型只描述 API 类别，不判断实现。** `obsidian-sort-and-permute-lines`
   报 C2、非降级，却依赖返回空对象的元数据缓存；反过来 `registerView` 明明是
   宿主真实实现的视图表面，却因为属于「DOM 级」被一律标降级。`Level` /
   `Degraded` 回答的是「用了哪类 Obsidian API」，不是「能不能在 Xinkgo 完好运行」。
2. **C3 里混了真实能力和空壳。** 同一个等级里既有 `registerView` / `ItemView`
   （真实），也有 `Modal` / `Menu` / `Setting` / `MarkdownRenderer`（no-op）。
3. **shim 为兼容保留了大量 stub 导出。** `require("obsidian")` /
   `require("xinkgo")` 是同一个模块对象，其 Proxy 对未知属性也返回惰性对象，
   作者很容易把「能调用」误当成「能力可用」。换个名字并不会改变 API 面。
4. **兼容清单把降级插件当作可推荐项，排查时的一键安装还会自动全授权。**
   128 项里 120 项降级；安装后 `vault.write` / `network` / `dom` 曾被一次性
   写入 `plugins.json`，与设计文档 §5.1 的「默认全关」口径不一致。P1 已改为
   安装不写任何授权（本文 §5.2）。
5. **「我们做的插件」和「外部插件」共用一套报告，没有可信边界。**
   管理器里看不出哪些是官方验证过、保证完好的。

结论：需要一层**独立于 C0~C4 的判定**，唯一判据是 Xinkgo 有没有真实实现。
这就是下一节的能力契约。

## 2 目标：以 Xinkgo 架构为准

### 2.1 「完好运行」的定义

一个插件算完好运行，当且仅当：

1. 它调用的每个 API，宿主都有真实实现，不是 no-op、空对象或惰性 Proxy；
2. 它声明的每项能力，都有对应的运行时行为（必要时经用户授权）；
3. 静态扫描无硬错误，`minAppVersion` 不超过宿主 `APIProfile`；
4. 它的每个命令、视图、事件都能给出可观察的结果；
5. 未授权 / 出错时用 `Notice` 和日志明确反馈，不会静默失败或拖垮宿主。

不满足任一条的插件，不进官方插件库，也不宣称「支持」。

### 2.2 三层模型

| 层 | 名称 | 实现形态 | 清单 | 承诺 |
| --- | --- | --- | --- | --- |
| L0 | 内置核心能力 | Go 原生 UI，编译进 app，只有开关 | `internal/plugin/core.go` | 完好运行 |
| L1 | **官方插件库** | 受控 JS 插件，走能力契约，逐个验收 | `resources/xinkgo-plugins/library.json` | **完好运行（唯一对外承诺）** |
| L2 | 外部兼容插件 | 现有 obsidian shim 宿主，隔离 WebView 页 | `resources/obsidian-plugins/compatible.json` | 不承诺；显式标「兼容 / 降级」 |

三层各自有清单，不混在一张表里。用户能一眼分辨「官方保证」与「尽量加载」。

### 2.3 与 Obsidian 的关系

Obsidian 的插件体系建立在 Electron + DOM workspace + CodeMirror 6 之上；
Xinkgo 是 Go 原生 UI + 每插件一个隔离 WebView 页。参考它的包形态
（`manifest.json` / `main.js` / `styles.css`、CommonJS、命令与生命周期）是合理的，
但**不能以它的 API 面为目标**。`require("obsidian")` 保留为 L2 兼容入口，
`require("xinkgo")` 是官方插件的原生入口；两者目前指向同一个模块对象，
是入口别名而非原生 API。L1 的准入以 Xinkgo 的能力契约为准，与「Obsidian
有没有这个 API」无关。

## 3 能力契约

### 3.1 白名单：11 项真实能力

| 能力 | 真实实现 |
| --- | --- |
| `data` | `data.load` / `data.save` RPC，`data.json` 落盘 |
| `commands` | `command.add` / `remove` / `visibility`，进命令面板 |
| `notice` | `notice` RPC，原生提示 |
| `ui` | `status.add` / `status.set` / `action.add`，真实 ribbon 与状态栏 |
| `vault.read` | `vault.read` / `cachedRead` / `adapter.*` / `workspace.open` |
| `vault.write` | `vault.create` / `modify` / `append` / `rename` / `delete` / `createFolder` |
| `editor.read` | `/state` 缓存里的文本、选区、光标、行 |
| `editor.write` | `editor.write` RPC，真实改编辑器并回发 `editor-change` |
| `network` | `network.request` RPC，Go `net/http`，15s / 1 MiB 限制 |
| `events.app` | 具名事件 `file-open` / `editor-change` / `vault-open` / `vault.*`，DOM 事件与 interval |
| `dom` | `registerView` → 插件视图抽屉 → `ItemView.containerEl` 挂载与生命周期 |

`dom` 允许入契约，是因为视图链路是宿主真实实现；但只允许
`registerView` / `ItemView` / `containerEl` / DOM helpers 这组用法，
设置 tab、弹层、菜单等不在此列。

### 3.2 黑名单

禁用能力 3 项：`metadata`（缓存为空、无事件源）、`editor.extension`（无 CM6
可挂载）、`compat.stub`（惰性 Proxy）。

禁用 API（只做 no-op / 空壳）：设置 tab 与 `Setting`、`Modal` / 建议弹层、
`Menu` / `MenuItem`、`MarkdownRenderer` / `HoverPopover`、Markdown 后处理器、
`EditorSuggest`、协议 / hover / bases / CLI 注册、`secretStorage`、
`addIcon` / `setIcon`、非 `obsidian` 的 `require(...)`。

完整名单以代码为准：`internal/plugin/contract.go` 的 `ContractStubAPIs`，
与 `demo_plugins_test.go` 的黑名单断言共用同一份，避免两份清单漂移。

### 3.3 报告与契约分离

| | 报告 `Report` | 契约 `ContractResult` |
| --- | --- | --- |
| 回答 | 这个包用了什么 Obsidian API | 它能不能在 Xinkgo 完好运行 |
| 依据 | `ScanSource` 文本规则 | 能力白名单 + stub 黑名单 + 等级上限 + 硬错误 |
| 用途 | L2 兼容展示、Grants 推断 | L1 官方库准入、离线门禁、未来运行时校验 |
| 等级 | C0~C4 | 上限 C3（更高即违规） |

`ValidateContract(report, mainJS)` 不修改 `Report`，也不改 `scan.go`：
现有兼容报告对 L2 继续有效，只是不再被当作「可完好运行」的依据。
`obsidian-sort-and-permute-lines` 就是一个实例——报告说 C2 非降级，
契约会因为 `metadata` 与 `getFileCache` 直接拒绝。

### 3.4 等级与 Degraded 的口径修正

契约允许的等级上限是 C3。`Degraded=true` 在本体系里只表示
「视图运行在插件自己的 WebView 页，不是原生 UI 面板」，是**实现位置**，
不是功能缺失。C4 意味着引入了禁用能力，应当修掉而不是接受。

## 4 官方插件库设计

### 4.1 清单格式

`resources/xinkgo-plugins/library.json` 随 app 打包（已加入
`scripts/check-resources.sh` 的资源门禁）：

```json
{
  "schema": 1,
  "profile": "1.0.0",
  "plugins": [
    {
      "id": "xinkgo-demo-full",
      "name": "Xinkgo Demo: Full",
      "version": "1.0.0",
      "kind": "reference",
      "path": "examples/plugins/xinkgo-demo-full",
      "capabilities": ["data", "commands", "..."],
      "verification": {
        "static": "pass",
        "runtime": "pass",
        "checkedAt": "2026-11-09"
      }
    }
  ]
}
```

| 字段 | 含义 |
| --- | --- |
| `schema` / `profile` | 清单格式版本、宿主 `APIProfile` |
| `id` / `name` / `version` | 与 `manifest.json` 一致，门禁会核对 |
| `kind` | `reference`（参考实现）或 `plugin`（正式收录） |
| `path` | 仓库内相对路径，供离线门禁读取包；随包分发时宿主只用元数据 |
| `capabilities` | 契约声明，必须与静态扫描结果完全一致 |
| `verification` | 静态扫描、运行时冒烟的结论与日期 |

`ParseLibrary` 会校验 schema、id 合法性、重复 id、能力是否越出白名单、
verification 是否齐全；门禁测试再对每个条目跑一遍真实包扫描与契约判定。

### 4.2 三重门禁

1. **静态**：`ParseManifest` + `ScanPackage`，无硬错误、`Imports` 只有
   `obsidian`、`minAppVersion` 不超 profile。
2. **契约**：`ValidateContract` 无违规，声明能力与扫描一致，等级不高于 C3。
3. **运行时**：Node 沙箱加载 `shim.js` + `main.js`，在授权 / 未授权两种情形下
   逐条执行命令，要求无未捕获异常、无 `error` RPC、所有 RPC 都有兜底。

三重都通过才写 `verification`，才能进入清单。门禁是 Go 测试，离线可跑，
不依赖网络，也不需要本机装过插件。

### 4.3 收录流程

写插件 → 跑三重门禁 → 填清单条目与 `verification` → `make check` →
随版本发布。升级宿主 `APIProfile` 或能力实现时，库内条目要重新跑一遍验证并
更新 `checkedAt`；实现退化的能力要从白名单移除，相关插件同步降级或下架。

### 4.4 首批条目

`xinkgo-demo-basic`（C2 / 7 项）与 `xinkgo-demo-full`（C3 / 11 项），
`kind: reference`。它们是契约的参考实现，也是新插件的模板；
L0 的 13 项内置核心由 `core.go` 管理，不重复入库。

本机那 3 个社区插件、兼容清单 128 项，全部留在 L2，不进官方库。

## 5 落地

### 5.1 P0：契约与离线门禁

| 落地 | 位置 |
| --- | --- |
| 排查结论与目标模型 | 本文 §1 ~ §2 |
| 能力契约（白名单 / 黑名单 / 等级上限 / 违规判定） | `internal/plugin/contract.go` |
| 官方库清单与解析校验 | `resources/xinkgo-plugins/library.json`、`ParseLibrary` |
| 离线门禁（契约单测 + 库内逐包校验） | `internal/plugin/contract_test.go` |
| 资源随包分发 | `scripts/check-resources.sh` |

### 5.2 P1：运行时接入

| 落地 | 位置 |
| --- | --- |
| 启动时加载随包的库清单 | `pluginHost.refreshLibrary`（`newPluginHost`） |
| 刷新时为每个发现的包算契约结论 | `pluginHost.scanContracts`、`refresh` |
| 库内插件未过契约则拒绝启用 | `pluginHost.contractRefused`、`enable` |
| 管理器新增「官方插件库」子页 | `libraryView`，第 4 个 tab |
| 已安装页的「已验证 / 契约未通过」徽标 | `installedRow` |
| 一键安装不再自动授权 | `installGrants` 返回空、`install` 状态提示 |
| 新文案进 catalog | `internal/locale/{en,zh-Hans}.json` |
| 模块入口别名（`xinkgo` / `obsidian` 同一对象） | `shim.js` 的 `require`、`internal/plugin/scan.go` 的 `HostModules`、契约的 Imports 校验 |

运行时契约**只对 L1 生效**：外部 L2 插件的发现、授权、启用与 WebView 生命周期
完全不变，用户已安装的插件不受影响。库清单缺失或不可解析只记一条 warn，不
阻断启动；契约结论还没算出来的库内插件也不会被误拒。

## 6 迁移路线

| 阶段 | 内容 |
| --- | --- |
| P0（已落地） | 设计 + 契约 + 库清单 + 离线门禁；两个 demo 收编为参考实现 |
| P1（已落地） | app 启动时读取库清单、刷新时算契约、库内插件未过契约拒绝启用；管理器新增「官方插件库」页并在已安装页标 Verified；一键安装改为不写任何授权 |
| P1.5（已落地） | shim 同时接受 `require("xinkgo")`，与 `require("obsidian")` 指向同一个模块对象；扫描器与能力契约放行两个名字；两个官方 demo 迁到原生入口，L2 插件零回归 |
| P2 | 把 `xinkgo` 收窄成 Xinkgo 原生 SDK：只导出 11 项真实能力对应的方法，不再模拟 `WorkspaceLeaf` / `Modal` / CM6；收窄后 `xinkgo` 与 `obsidian` **不再是同一个对象** |
| P3 | 官方库扩充实际功能插件；L2 兼容插件默认折叠、显式标注「不保证完好运行」；每个新能力先有真实实现，再进白名单，最后才允许被插件使用 |

P2 的 SDK 形状（暂定，不本期实现）：`xinkgo` 模块只导出白名单能力对应的
方法，不再模拟 `WorkspaceLeaf` / `Modal` / CM6；生命周期沿用 `Plugin` /
`onload` / `onunload`，包形态仍是单文件 CommonJS。这样「能调用的 API」与
「Xinkgo 真实支持的能力」最终合为一套，从根上消除 stub 误用。P1.5 已先把
入口别名落地（零回归），但两个名字仍指向同一个兼容对象；收窄 SDK 才是决定性
的一步，仍未实现。

## 7 明确不做

- 不追求 100% Obsidian 插件兼容，也不把 128 个兼容条目搬进官方库。
- 不把 no-op / 空壳伪装成能力；shim 的惰性 Proxy 只服务 L2 兼容。
- 不因为「Obsidian 有」就给 Xinkgo 插件加 API；能力先有真实实现再进白名单。
- 不改 L2 外部插件的发现、授权与启用行为；契约只用于 L1 准入。
- 不引入 cgo、Node 运行期依赖或新的 Go 依赖（遵循 AGENTS.md）。

## 8 验收清单

1. `go test ./internal/plugin -run 'Contract|Library'` 通过：两个 demo 通过契约，
   禁用能力、stub API、C4 等级、外部导入都被拒绝。
2. 库清单里每个条目都能被门禁离线校验，且声明能力与扫描一致。
3. `make check` 全绿，新清单出现在源码与各平台产物里。
4. 排查表（§1.2）里的 L2 插件都不会被误判为完好运行。
5. 文档索引能找到本文、开发指南与使用手册。
6. `go test ./internal/app -run 'Contract|Library|Enable|InstallGrants|UISnapshot'` 通过：
   库清单加载、契约扫描、启用拒绝、外部插件不受约束、安装不写授权、官方库页渲染。
7. 一键安装后 `plugins.json` 的 `grants` 为空，状态提示授权后 Reload；
   官方库页里库内插件带 Verified 徽标。

## 9 参考

- [obsidian-plugins.md](obsidian-plugins.md)：兼容宿主的调研、等级与 P0 落点
- [../plugin-development.md](../plugin-development.md)：能力准入与避坑清单
- [../plugin-demos.md](../plugin-demos.md)：安装、授权、命令预期
- [../../examples/plugins/README.md](../../examples/plugins/README.md)：两个参考实现
- `internal/plugin/scan.go`、`plugin.go`：报告与等级定义
- `internal/app/plugin_rpc.go`：真实 RPC 方法表
