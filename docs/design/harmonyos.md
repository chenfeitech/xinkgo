# 鸿蒙（HarmonyOS NEXT / OpenHarmony）适配调研

调研日期：2026-11-02。调研范围：桌面版 Xinkgo（Go 1.27.3 + MyGo v0.7.4）能否、
以及要付出多大代价才能跑到鸿蒙上。

结论先说：**能跑，但不是 Xinkgo 自己能完成的事**。窗口、菜单、对话框、剪贴板、
文件监听、通知、托盘、密钥这些平台能力全部由 MyGo 的平台后端提供，而后端代码在
MyGo 的 `internal/` 里。适配鸿蒙 = 给 MyGo 新增一个后端（量级 1.5 万行）+ 让 Go
工具链支持 `openharmony`。本仓库要改的反而不多。

本文中的事实分两类：本仓库与依赖源码里读出来的（一手的，可靠）；公开搜索得到的
（二手的，标了"待验证"的必须上真机/DevEco 复核）。

---

## 1 我们手里有什么：依赖边界

从 `go.mod` 与模块缓存里读出的：

- `github.com/egoist/mygo v0.7.4`，其平台后端按 GOOS 切分：
  `backend_darwin.go`（AppKit/objc）、`backend_windows.go`（win32）、
  `backend_linux.go`（GTK3 + Cairo + GLib）、`backend_js.go`（wasm + canvas），
  其余平台落到 `backend_other.go` → `internal/unsupported`（`Init` 直接报
  `mygo: <goos>/<goarch> is not supported yet`）。
- 后端要实现的接口是 `internal/platform.Backend`：窗口、应用菜单、右键菜单、对话框、
  剪贴板、Shell、屏幕、主题、电源、网络、Secrets、已安装应用、文件监听、
  原子替换文件、托盘、全局热键、通知、WebView 可用性。
- 各后端规模（不含测试）：linux 15.6k 行、windows 20.8k、darwin 14.9k、web 2.5k
  （web 是不完整的：菜单空实现、打开/保存对话框直接返回 unsupported、文件监听空实现）。
- FFI 走 `github.com/ebitengine/purego`（无 cgo 调原生库）。它只覆盖
  darwin/freebsd/linux/netbsd：`dlfcn_nocgo_linux.go` 从 **`libdl.so.2`** 取
  `dlopen`，`syscall_unix.go`/`sys_unix_arm64.s` 的 build tag 也不含 openharmony。
- 渲染：`internal/gpu`，macOS Metal、Windows D3D11、Linux/Web 用 GL（Web 是 WebGL2，
  Linux 是 GtkGLArea + OpenGL 3.3 / ES 3.0，无 GPU 时退回内存绘制）。

本仓库自身的平台相关代码很少：`internal/app/platform_darwin.go`、
`platform_other.go`、`internal/secret/secret_{darwin,other}.go`（密钥走 `security`
命令）。也就是说：**Xinkgo 侧的工作量小，工作量在依赖里**。

## 2 鸿蒙侧的事实

**应用模型**：一个鸿蒙应用是 HAP，入口是 ArkTS/ArkUI 的 Ability；原生代码只能是
被 NAPI 加载的 `.so`（`libxxx.so`）。没有"Go 应用"这种一等公民。来源：公开文档与
多个移植案例，二手。

**Go 目标**：官方 Go 没有 `openharmony`。OpenHarmony-SIG 维护
`openharmony-sig/ohos_golang_go`（GitCode），分支 `release-branch.go1.24`
（go1.24.5），只支持 arm64，用 TLSDESC 修掉了 musl 的 TLS 问题。华为官方"鸿蒙
开发语言支持情况"（2026-04）把 Go 标为"支持"，但注明是 SIG 开源版本、"Go 官网未
声明支持"。**本项目 `go.mod` 要求 go 1.27.3，与 SIG fork 有 3 个版本代差**——要么
等 fork 跟进，要么自己把补丁搬到新版本上。（二手，待验证）

**`.so` 加载的坑（关键）**：标准 Go 的 `-buildmode=c-shared` 用 initial-exec TLS，
musl 会拒绝 dlopen 这类库（`golang/go` 的 #54805、#13492 就是这个问题）。社区实测
在 openharmony 上直接编 c-shared 会报
`-buildmode=c-shared not supported on openharmony/arm64`（华为论坛，2025-07）。
用 SIG fork 编出的产物带 `PT_TLS` + `R_AARCH64_TLSDESC`，有团队（2048 小游戏、
Hey 项目文档）用它跑通了 CGO_ENABLED=1 的 `.so`。**结论：走 `.so` 必须 CGO_ENABLED=1，
与本项目"CGO_ENABLED=0"的铁律冲突，需为这个平台开例外。**（二手，待验证）

**纯静态 ELF 能跑**：有团队用 openharmony/arm64 工具链产出静态自签名 ELF（CGO=0
也可以），在**鸿蒙 PC（AArch64，HongMeng Kernel）**上跑通了命令行程序，需要
`binary-sign-tool` 自签名。这说明 Go 运行时本身在鸿蒙上是能站住的。（二手，待验证）

**渲染通道**：ArkUI 的 `XComponent`（type=surface/texture）把 `OH_NativeWindow`
交给原生层，并提供触摸、鼠标、按键、焦点/失焦回调；原生侧用 EGL/OpenGL ES（也可
Vulkan 的 `VK_OHOS_surface`）绘制并提交 buffer。Flutter 的 OHOS 嵌入层正是这个
架构（XComponent + FlutterEngine + EGL/Vulkan），可以作为工程量参照。（二手，
架构描述一致，可信度较高）

**文件访问**：鸿蒙是沙箱 + 授权模型。用户目录要用 `DocumentViewPicker` 之类的
选择器授予，默认是临时授权；长期读写需要 `fileShare.persistPermission` +
`activatePermission`（需 `ohos.permission.FILE_ACCESS_PERSIST` 及目录授权能力）。
**"打开一个文件夹当 vault"这件事要改成 URI/授权模型**，而不是现在的一个绝对路径。
（二手，待验证；这条直接冲击 `internal/vault`、`internal/fs`、文件监听）

**打包分发**：HAP 由 DevEco/hvigor 打包 + 证书签名；鸿蒙 PC 上的独立 ELF 需自签名。
`mygo` CLI 目前只会打 `.app`/`.dmg`/`.exe`，没有鸿蒙目标，`scripts/package.sh`
也要新增一段。

**ArkWeb + wasm**：ArkWeb 基于 Chromium M114（二手）；官方"坚盾守护模式"文档把
WebAssembly/WebGL 列为该模式下被禁用的能力，反推正常模式可用；JSVM-API 也能跑
wasm（需 JIT 权限）。但 MyGo 的 web 后端是残缺的（见 §1）。（二手，待验证）

## 3 三条路径

| 路径 | 做法 | 体验 | 工作量（粗估） | 主要风险 |
|---|---|---|---|---|
| A 原生 `.so` | ArkTS 外壳（UIAbility + XComponent）→ NAPI → Go `.so`（CGO=1，OHOS clang 或 SIG fork）→ EGL/GLES 画到 NativeWindow | 最好，接近桌面 | MyGo 新后端 15k+ 行 + NAPI 桥 + Xinkgo 适配，**3～6 人月** | 依赖 MyGo 上游（或长期 fork）；purego 要支持 openharmony；CGO 例外；IME |
| B ArkWeb + wasm | `GOOS=js GOARCH=wasm` 编译，跑在 ArkTS 外壳的 Web 组件里，文件由 ArkTS 桥接 | 打折：窗口/菜单/对话框要自己补，性能靠 WebView | 补齐 MyGo web 后端 + 文件桥 + 调优，**1～2 人月出原型** | 软件渲染在 WebView 里的帧率；wasm 无真并行，索引/搜索慢；wasm 支持随模式变化 |
| C 鸿蒙 PC 跑 ELF | 静态自签名 ELF，纯 Go（CGO=0） | 只能命令行/服务，**没有 GUI** | **0.5～1 人月** | 没有找到"独立进程创建窗口"的公开接口（图形栈是 Rosen/ArkUI，无 X11/Wayland）；上架无从谈起 |

另有传闻中的"鸿蒙 PC 兼容层（HCL）"可跑静态 x86_64 Linux ELF 并桥接 OpenGL，
来源是一篇风格偏推测的技术博客，**不可作为决策依据**，但值得留意。

## 4 阻塞点（按严重度）

1. **MyGo 后端在 `internal/` 下**：Xinkgo 无法自行实现，必须 MyGo 上游支持或长期
   fork。这是唯一的结构性门槛。
2. **Go 工具链**：官方无 openharmony；SIG fork 停在 go1.24，本项目要 1.27.3。
3. **CGO_ENABLED=0 铁律**：`.so` 路线必须 CGO=1；且鸿蒙的 musl 要求 TLSDESC，
   普通 Go 编出的 c-shared 会被 dlopen 拒绝。
4. **purego 不支持 openharmony**（build tag 不含，且鸿蒙的 `dlopen` 在 `libc.so`
   里而非 `libdl.so.2`）→ MyGo 现有 FFI 机制要么改 cgo，要么补 purego 上游。
5. **文件模型**：vault 从"绝对路径"改成"URI + 持久化授权"，并要处理授权失效、
   跨重启恢复。
6. **IME/中文输入**：Markdown 编辑器是核心交互。原生自绘表面上做输入要靠输入法
   框架或"隐藏输入框"技巧（MyGo 的 web 后端就用隐藏 textarea）。这条在鸿蒙上
   没有现成答案，风险高。
7. **多窗口**：Xinkgo 会开多个窗口（设置、任务、历史、快照），鸿蒙侧要映射成
   UIAbility 多实例或 `windowStage` 子窗口，生命周期与状态恢复都要重新设计。
8. **密钥**：`internal/secret` 现在是调 `security` 命令，鸿蒙上要换 HUKS（ArkTS/NAPI）。
9. **打包/CI/签名**：DevEco、证书、鸿蒙 SDK 都要进流水线。

## 5 建议

1. **现在不要启动 A**。先花 0.5～1 人月做两个 spike：
   - 用 SIG fork（或 OHOS clang + cgo）编一个 hello-world `.so`，在 XComponent 上
     画一帧，验证 TLS/加载/EGL 这条链；
   - 用纯 Go 静态 ELF 在鸿蒙 PC 上跑 Xinkgo 的命令行能力（`internal/index`、
     `internal/search`、`internal/task`、`internal/ai`），验证 musl、网络、文件、
     mmap 等行为。
2. **同时向 MyGo 上游确认**：是否愿意支持 harmony 后端（接受 PR 或出资）。这一步
   决定 A 是否可行——不解决它，后面都是空谈。
3. **若要快速验证产品形态**：做 B 的原型的成本最低，但要先补 MyGo 的 web 后端
   （菜单、对话框、文件监听）。
4. **目标若是鸿蒙 PC 桌面**：重点跟进"第三方原生窗口/桌面应用"的官方支持信号，
   以及 SIG fork 对新版 Go 的跟进。

## 6 待验证（本次调研的边界）

本次没有真机、没有 DevEco、没有鸿蒙 PC，以下只能标为未知：

- 鸿蒙 PC 是否存在第三方可用的窗口协议（Wayland/X11）——未找到任何公开接口，
  倾向没有；
- SIG fork 是否会跟进 go 1.27，或官方 Go 是否计划支持 openharmony；
- ArkWeb 的 wasm 内存上限与 WebGL2 性能；
- XComponent 多窗口/自由窗口（可拖动、可缩放）的支持度；
- 鸿蒙 PC 兼容层（HCL）能否承载 GTK 应用。
