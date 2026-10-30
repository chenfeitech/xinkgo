# Harness：构建、运行、测试、调试

本文是 Xinkgo 的日常操作手册。命令入口统一是仓库根目录的 `Makefile`；
带逻辑的部分在 `scripts/` 里。编码与提交规范见
[CONTRIBUTING.md](CONTRIBUTING.md)。

## 1 环境准备

| 依赖 | 版本 | 用途 | 检查 |
|---|---|---|---|
| Go | 1.27 或更高 | 编译（`go.mod` 声明 1.27.3，低版本由 Go 自动下载 toolchain） | `go version` |
| MyGo CLI | v0.7.3 | 由 `go.mod` 的 `tool` 指令固定，`go tool mygo` 直接跑，不需要全局安装 | `go tool mygo version` |
| Windows 目标机 | Windows 10/11 | 运行与冒烟；**仅当使用 Web 窗口时**需要 WebView2 Runtime（Win11 自带，Win10 需安装） | — |
| NSIS | 可选 | 生成 Windows 安装器；在 Windows 上 `mygo build` 会自行下载，其它系统需自行安装 `makensis` | `makensis -VERSION` |
| Delve | 可选 | 断点调试 | `dlv version` |
| golangci-lint | 可选 | 额外静态检查，非门禁 | `golangci-lint version` |

不需要的：**不需要 cgo，不需要 C 编译器，不需要 Bun/Node**（骨架是原生 UI，
没有前端构建）。因此 macOS/Linux 可以直接编译出 Windows 可执行文件。

环境自检：

```sh
make doctor        # go tool mygo doctor
```

## 2 命令速查

| 命令 | 作用 |
|---|---|
| `make help` | 列出全部命令 |
| `make run` | 从源码运行应用 |
| `make dev` | 运行并监听改动，自动重建重启（live reload） |
| `make build` | 为当前机器构建，产物 `bin/xinkgo` |
| `make build-windows` | 交叉编译 Windows 可执行文件，产物 `bin/windows-amd64/xinkgo.exe` |
| `make package` | 打包可安装应用，默认 Windows/amd64，产物 `build/windows-amd64/` |
| `make cross` | 为全部目标平台编译（不打包） |
| `make test` / `make test-race` | 跑测试 / 带竞态检测跑测试 |
| `make vet` / `make fmt` / `make fmt-check` | 静态检查 / 格式化 / 格式校验 |
| `make lint` | golangci-lint（未安装则跳过） |
| `make check` | **提交前门禁**：格式 + vet + 测试 + 全平台编译 + 版本一致性 |
| `make debug` | 在 Delve 下运行 |
| `make clean` | 清理 `bin/`、`build/`、`.mygo/` |

Windows 上没有 `make` 时，直接跑 `scripts/check.sh`（Git Bash）或 Makefile
打印出的等价 `go` 命令。

## 3 构建

### 3.1 本机构建与运行

```sh
make build          # bin/xinkgo
./bin/xinkgo        # 或用 make run 一步到位
```

### 3.2 交叉编译（Windows）

```sh
make build-windows                       # bin/windows-amd64/xinkgo.exe
make cross                               # 全部平台只编译不打包
GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build ./cmd/xinkgo   # arm64
```

目标平台矩阵：

| GOOS/GOARCH | 状态 | 说明 |
|---|---|---|
| `windows/amd64` | **主目标** | Windows 10/11 x64 |
| `windows/arm64` | 支持 | Windows on ARM |
| `darwin/arm64`、`darwin/amd64` | 支持 | 开发机（macOS）自测 |
| `linux/amd64` | 支持 | CI 冒烟 |

### 3.3 打包可安装应用

```sh
make package                                     # Windows x64
scripts/package.sh windows/amd64 windows/arm64   # 两个架构
scripts/package.sh darwin/universal              # macOS 通用包（在本机）
```

`mygo build` 做的事：编译（注入 `mygo.json` 的名字、标识符、版本）、把
`resources/` 复制到可执行文件旁、生成 Windows 资源（图标与版本信息，
`mygo_windows_*.syso`，已 gitignore）、可用时生成安装器。

产物（`build/<os>-<arch>/`）：

```
build/windows-amd64/
├── Xinkgo.exe        13 MB 左右
└── icon.png
```

- **安装器**：需要 NSIS。在 macOS 上会打印
  `Skipping the Windows installer: install NSIS (makensis)` 并继续产出
  `.exe`；在 Windows 上 `mygo build` 会自行下载 NSIS 并生成
  `xinkgo-setup-<version>-amd64.exe`。
- **签名**（可选）：在 `mygo.json` 配 `windows.certificate`（`.pfx`），密码由
  环境变量 `MYGO_WINDOWS_CERTIFICATE_PASSWORD` 提供：

  ```jsonc
  { "windows": { "certificate": "certs/code-signing.pfx" } }
  ```

- **保留调试特性**：`go tool mygo build -debug`（保留 native UI 检查器）。
- **版本号**：`scripts/package.sh` 会校验 `mygo.json` 的 `version` 与
  `internal/app/app.go` 的 `const Version` 一致，不一致直接失败。

## 4 运行

```sh
make run     # 从源码运行：go run ./cmd/xinkgo
make dev     # live reload：改 .go / mygo.json / resources 自动重建并重启
```

- `make dev` 构建的是**开发版**：日志输出到终端，保留 native UI 检查器，
  窗口状态（位置尺寸，通过 `StateKey`）独立保存。编译失败时保留上一个可用
  版本继续运行。
- `mygo dev` 在 macOS 上产出独立应用包（标识符带 `.dev`），数据、偏好与权限
  与已安装版本隔离。

运行期目录（`mygo.App.Path`）：

| 平台 | 用户数据（`config.json`） | 日志 |
|---|---|---|
| Windows | `%AppData%\Xinkgo\config.json` | `%AppData%\Xinkgo\logs\xinkgo.log` |
| macOS | `~/Library/Application Support/Xinkgo/config.json` | `~/Library/Logs/Xinkgo/xinkgo.log` |
| Linux | `$XDG_CONFIG_HOME/Xinkgo/config.json` | 用户数据目录下的 `logs/` |

启动是否成功，看日志第一行：

```
level=INFO msg=starting app="Xinkgo 0.1.0 (windows/amd64)" data="..." development=true
```

## 5 测试

```sh
make test                                  # go test ./...
make test-race                             # go test -race ./...
go test ./internal/config/... -v           # 单个包
go test -run TestSaveLoad_roundTrip ./internal/config/... -v   # 单个用例
```

约定与边界用例写法见 [CONTRIBUTING.md](CONTRIBUTING.md) 第 3 节。要点：

- 视图用 `ui.NewTester` 在无窗口环境点击与输入，不需要 GUI 会话。
- 目录一律 `t.TempDir()`；需要应用目录时用
  `mygo.App.SetPath(mygo.PathUserData, dir)` 重定向，不碰真实用户目录。
- 测试不得依赖 `time.Sleep` 与机器速度。

## 6 调试

### 6.1 日志

开发模式（`MYGO_ENV` 不是 `production`，例如 `make run`、`make dev`）日志到
终端；安装版写到日志文件（见第 4 节表格）。

调高日志级别（编辑用户数据目录里的 `config.json`）：

```json
{ "theme": "system", "logLevel": "debug" }
```

生产构建下 `MYGO_ENV=production` 由 `mygo build` 注入，也可手动设置：
`MYGO_ENV=production ./bin/xinkgo`。

### 6.2 断点调试

```sh
make debug                    # dlv debug ./cmd/xinkgo
dlv debug ./cmd/xinkgo -- --flag   # 带参数（应用目前没有命令行参数）
```

注意：应用逻辑跑在**主线程**上，在主线程回调里长时间断住会让窗口无响应；
断在 `internal/view` 的视图函数里时，界面每帧都会进来，配合条件断点使用。

### 6.3 界面检查器

Native UI 带有检查器（元素树、尺寸、命中区域）。开发构建默认保留；生产构建
由 `mygo build` 移除，需要保留时：

```sh
MYGO_INSPECTOR=1 go tool mygo build -platform windows/amd64
```

### 6.4 环境与依赖自检

```sh
make doctor     # Go、webview、打包工具链
go env GOOS GOARCH CGO_ENABLED
```

## 7 故障排查

| 现象 | 原因 | 处理 |
|---|---|---|
| `go: ... requires go >= 1.27.3` | 本机 Go 更低 | 升级 Go，或让它自动下载 toolchain（`GOTOOLCHAIN=auto`，Makefile 已导出） |
| `Skipping the Windows installer` | 缺少 NSIS | 本机安装 `makensis`，或到 Windows 上打包 |
| Windows 上 Web 窗口空白 | 缺少 WebView2 Runtime | Win10 安装 WebView2 Runtime；主窗口是原生 UI，不受影响 |
| 启动后没有窗口、无日志 | 日志在文件里（安装版） | 查第 4 节的日志路径 |
| 配置被重置 | 配置损坏无法解析 | 按设计降级为默认值并 `Warn`；查日志里的 `configuration unusable` |
| `version` 不一致报错 | 只改了一处 | 同时改 `mygo.json` 与 `internal/app/app.go` 的 `Version` |
| `make package` 卡住 | 首次编译依赖较慢 | 等首次完成，后续有缓存（分钟级 → 秒级） |

## 8 持续集成建议

```yaml
# .github/workflows/ci.yml（骨架未内置，接入时添加）
- run: go build ./...
- run: scripts/check.sh          # 格式 + vet + 测试 + 全平台编译 + 版本一致
- run: scripts/package.sh windows/amd64
- uses: actions/upload-artifact
  with: { name: xinkgo-windows, path: build/windows-amd64 }
```

Linux  runner 上打包 Windows 会跳过安装器（无 NSIS）；需要安装器时用
`windows-latest` runner。
