# 插件开发指南 · Plugin Development Guide

本文档说明如何为 newapi-tools 开发 Go 原生插件。

---

## 1. 插件接口 · Plugin Interface

所有插件必须实现 `plugin.Plugin` 接口（定义在 `internal/plugin/plugin.go`）：

```go
type Plugin interface {
    Name() string                          // 插件唯一标识，如 "newapi"
    Version() string                       // 语义化版本号，如 "1.0.0"
    Commands() []Command                   // 插件提供的命令列表
    Init(ctx Context) error                // 初始化，接收运行时上下文
    Execute(cmd string, args []string) error // 执行某个命令
    Shutdown() error                       // 清理资源
}
```

**Command 结构体**：

```go
type Command struct {
    Name        string  // 命令名，如 "install"
    Description string  // 简短描述，如 "安装 NewAPI"
    Usage       string  // 用法提示，如 "newapi install [flags]"
}
```

**Context 结构体**：

```go
type Context struct {
    NewAPIHome       string       // NewAPI 安装目录
    NewAPIPort       int          // 服务端口
    NewAPIDockerImg  string       // Docker 镜像名
    NewAPIBackupDir  string       // 备份目录
    DockerComposeCmd string       // compose 命令路径
    Logger           *slog.Logger // 结构化日志
    HomeDir          string       // 用户主目录
}
```

---

## 2. 编译时注册 · Compile-time Registration

newapi-tools 使用 **编译时注册** 机制，不需要运行时扫描文件系统来发现 Go 插件。

### 注册流程

1. 在插件包中定义 `init()` 函数，调用 `plugin.Register()`
2. 在 `cmd/newapi/main.go` 中添加 blank import

```go
// plugins/mypackage/plugin.go
package mypackage

import "github.com/Bonus520/newapi-tools/internal/plugin"

type MyPlugin struct{}

func (p *MyPlugin) Name() string    { return "mypackage" }
func (p *MyPlugin) Version() string { return "1.0.0" }
func (p *MyPlugin) Commands() []plugin.Command {
    return []plugin.Command{
        {Name: "hello", Description: "打个招呼", Usage: "mypackage hello"},
    }
}
func (p *MyPlugin) Init(ctx plugin.Context) error      { return nil }
func (p *MyPlugin) Execute(cmd string, args []string) error { return nil }
func (p *MyPlugin) Shutdown() error                    { return nil }

func init() {
    plugin.Register(&MyPlugin{})
}
```

```go
// cmd/newapi/main.go
import (
    _ "github.com/Bonus520/newapi-tools/plugins/mypackage"
)
```

### 读取已注册插件

```go
// 获取单个插件
p, ok := plugin.Get("mypackage")

// 获取全部插件
all := plugin.All()

// 在 Registry 中统一管理
reg := registry.NewRegistry()
for _, p := range plugin.All() {
    reg.RegisterPlugin(p, logger)
}
```

### 注意事项

- `Register()` 如果注册同名插件会 **panic**，这是故意的——重复注册是编程错误，应在开发阶段暴露
- blank import（`_ "..."`）是触发 `init()` 的唯一方式，忘记加就不会注册
- 注册是 **goroutine 安全** 的，内部有读写锁保护

---

## 3. 最小示例 · Minimal Example

下面是一个最小可运行插件，只需要三个文件：

```
plugins/hello/
├── hello.go
└── metadata.yml      # 可选，Shell 插件需要；Go 插件不需要
```

**hello.go**：

```go
package hello

import (
    "fmt"

    "github.com/Bonus520/newapi-tools/internal/plugin"
)

type HelloPlugin struct{}

func (p *HelloPlugin) Name() string    { return "hello" }
func (p *HelloPlugin) Version() string { return "0.1.0" }
func (p *HelloPlugin) Commands() []plugin.Command {
    return []plugin.Command{
        {Name: "greet", Description: "打招呼", Usage: "hello greet [name]"},
    }
}
func (p *HelloPlugin) Init(ctx plugin.Context) error { return nil }
func (p *HelloPlugin) Execute(cmd string, args []string) error {
    name := "world"
    if len(args) > 0 {
        name = args[0]
    }
    fmt.Printf("Hello, %s!\n", name)
    return nil
}
func (p *HelloPlugin) Shutdown() error { return nil }

func init() {
    plugin.Register(&HelloPlugin{})
}
```

---

## 4. 目录结构建议 · Recommended Structure

一个完整的 Go 插件包建议按以下方式组织：

```
plugins/myplugin/
├── myplugin.go       # Plugin 接口实现 + init() 注册
├── myplugin_test.go  # 单元测试
├── commands.go       # 命令处理逻辑（可选，拆分用）
└── helpers.go        # 辅助函数（可选）
```

如果插件同时提供 Shell 脚本版本（兼容旧模式）：

```
plugins/myplugin/
├── myplugin.go       # Go 实现（优先）
├── metadata.yml      # Shell 插件元数据（备选）
└── scripts/          # Shell 脚本（备选）
    └── install.sh
```

Loader 加载规则：Go 插件优先，如果 Go 插件不可用则回退到 Shell 插件。

---

## 5. Go 插件 vs Shell 插件 · Go vs Shell Plugins

| 特性 | Go 插件 | Shell 插件 |
|------|---------|-----------|
| 注册方式 | `plugin.Register()` + `init()` | Loader 扫描 `metadata.yml` |
| 文件要求 | `.go` 源文件 | `metadata.yml` + `scripts/` |
| 执行方式 | 直接函数调用 | `bash script.sh` |
| 性能 | 高（零开销调用） | 低（进程启动开销） |
| 错误处理 | Go 原生 error | 解析退出码 + stderr |
| 类型安全 | 编译时检查 | 无 |
| 跨平台 | Go 编译保证 | 依赖 bash |
| 热加载 | 不支持（需重新编译） | 支持（文件系统扫描） |
| 适用场景 | 核心功能、高性能需求 | 快速原型、运维脚本 |

**什么时候用 Shell 插件**：写个一次性脚本、需要直接调用系统命令、不想重新编译。

**什么时候用 Go 插件**：核心业务逻辑、需要类型安全、需要与项目内其他包交互。

---

## 6. 与 Registry 配合 · Working with Registry

`internal/registry` 包提供命令路由功能。Go 插件注册后，可以通过 Registry 统一调度：

```go
reg := registry.NewRegistry()
for _, p := range plugin.All() {
    if err := reg.RegisterPlugin(p, logger); err != nil {
        log.Fatal(err)
    }
}

// 执行命令
if err := reg.ExecuteCommand("install", []string{}); err != nil {
    log.Fatal(err)
}
```

Registry 会自动处理命令名冲突（后注册的命令会被跳过，并打印警告日志）。

---

## 7. 常见问题 · FAQ

**Q: 忘记加 blank import 会怎样？**

A: 插件的 `init()` 不会执行，`plugin.Get("xxx")` 返回 `false`。不会报错，只是静默跳过。排查时检查 `main.go` 的 import 列表。

**Q: 可以在运行时动态加载 Go 插件吗？**

A: 目前不支持。Go 插件必须编译进主二进制文件。如需运行时扩展，使用 Shell 插件。

**Q: 插件之间可以互相调用吗？**

A: 不直接支持。插件间通信应通过 Registry 的命令路由，或通过共享的 Context 数据。
