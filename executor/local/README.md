# Local Executor

Local Executor 实现了在本地机器上直接执行 Pipeline 节点的功能，无需额外的容器或远程环境。

## 功能特性

- 在本地机器上直接执行流水线步骤
- 支持自定义工作目录
- 支持环境变量配置
- 支持多步骤顺序执行
- 支持自定义 Shell（bash/sh/powershell/cmd）
- 支持执行超时设置
- 支持伪终端模式（PTY）
- 实时流式输出执行结果
- 支持执行终止功能
- 跨平台支持（Windows/Linux/macOS）

## 依赖

Local Executor 仅依赖 Go 标准库，无需额外安装。

## 配置示例

### 全局执行器配置

```yaml
Executors:
  local:
    type: local
    config:
      workdir: /home/user/project      # 可选：工作目录
      shell: bash                       # 可选：指定 shell (bash/sh/powershell/cmd)
      timeout: 10m                      # 可选：默认超时时间
      pty: true                         # 可选：启用伪终端模式
      ptyWidth: 120                     # 可选：终端宽度（默认 80）
      ptyHeight: 40                     # 可选：终端高度（默认 24）
      env:                              # 可选：环境变量
        GO_VERSION: "1.21"
        NODE_ENV: production
```

### 节点配置

```yaml
Nodes:
  Build:
    executor: local
    steps:
      - name: install-deps
        run: npm install
      - name: build
        run: npm run build

  Test:
    executor: local
    workdir: /home/user/project
    steps:
      - name: unit-test
        run: go test ./...
      - name: integration-test
        run: make test-integration
```

## API 使用

### 基本使用

```go
import (
    "context"
    "log"

    "github.com/chenyingqiao/pipelinex"
    "github.com/chenyingqiao/pipelinex/executor/local"
)

func main() {
    ctx := context.Background()

    // 创建执行器
    executor := local.NewLocalExecutor()

    // 配置执行器
    executor.SetWorkdir("/home/user/project")
    executor.SetEnv("CGO_ENABLED", "0")
    executor.SetShell("bash")

    // 准备环境（验证工作目录和 shell）
    if err := executor.Prepare(ctx); err != nil {
        log.Fatal(err)
    }
    defer executor.Destruction(ctx) // 确保终止正在执行的进程

    // 创建通信通道
    resultChan := make(chan any)   // 用于接收执行结果
    commandChan := make(chan any)  // 用于发送命令

    // 启动 Transfer 处理 goroutine
    go executor.Transfer(ctx, resultChan, commandChan)

    // 发送步骤执行
    steps := []pipelinex.Step{
        {Name: "build", Run: "go build -o app ."},
        {Name: "test", Run: "go test ./..."},
    }
    commandChan <- steps

    // 接收执行结果
    for i := 0; i < len(steps); i++ {
        result := <-resultChan
        if r, ok := result.(*local.StepResult); ok {
            if r.Error != nil {
                log.Printf("Step %s failed: %v\n", r.StepName, r.Error)
            } else {
                log.Printf("Step %s succeeded\n", r.StepName)
            }
        }
    }
}
```

### 使用 Bridge 和 Adapter 模式

```go
// 创建桥接器
bridge := local.NewLocalBridge()

// 创建适配器并配置
adapter := local.NewLocalAdapter()
config := map[string]any{
    "workdir": "/home/user/project",
    "shell": "bash",
    "timeout": "5m",
    "env": map[string]string{
        "GO_VERSION": "1.21",
    },
}

if err := adapter.Config(ctx, config); err != nil {
    log.Fatal(err)
}

// 通过桥接器创建执行器
executor, err := bridge.Conn(ctx, adapter)
if err != nil {
    log.Fatal(err)
}
```

## 接口实现

Local Executor 实现了以下接口：

| 接口 | 实现类型 | 说明 |
|------|---------|------|
| `pipelinex.Executor` | `LocalExecutor` | 执行器主接口 |
| `pipelinex.Adapter` | `LocalAdapter` | 配置适配器接口 |
| `pipelinex.Bridge` | `LocalBridge` | 连接桥接器接口 |

## LocalExecutor 方法

### 构造函数

```go
// 创建新的本地执行器
func NewLocalExecutor() *LocalExecutor
```

### 配置方法

```go
func (l *LocalExecutor) SetWorkdir(workdir string)       // 设置工作目录
func (l *LocalExecutor) SetEnv(key, value string)        // 设置环境变量
func (l *LocalExecutor) SetShell(shell string)           // 设置 shell
func (l *LocalExecutor) SetTimeout(timeout time.Duration) // 设置默认超时
func (l *LocalExecutor) SetPTY(enabled bool)             // 设置是否启用伪终端
func (l *LocalExecutor) SetPTYSize(width, height int)    // 设置终端尺寸
```

### 生命周期方法

```go
func (l *LocalExecutor) Prepare(ctx context.Context) error      // 准备执行环境
func (l *LocalExecutor) Transfer(ctx context.Context, resultChan chan<- any, commandChan <-chan any)  // 执行命令
func (l *LocalExecutor) Destruction(ctx context.Context) error  // 终止当前进程
```

### 获取配置方法

```go
func (l *LocalExecutor) GetWorkdir() string   // 获取工作目录
func (l *LocalExecutor) GetShell() string     // 获取当前 shell
```

## StepResult 结构

```go
type StepResult struct {
    StepName   string    // 步骤名称
    Command    string    // 执行的命令
    Output     string    // 命令输出（预留字段）
    Error      error     // 执行错误（如果 exit code != 0）
    StartTime  time.Time // 开始时间
    FinishTime time.Time // 结束时间
}
```

## 跨平台支持

Local Executor 自动检测操作系统并使用相应的 shell：

| 操作系统 | 优先 Shell | 回退 Shell |
|---------|-----------|-----------|
| Windows | PowerShell (pwsh/powershell) | cmd |
| Linux/macOS | bash | sh |

## 注意事项

1. **工作目录**：
   - 如果指定了工作目录，`Prepare()` 会验证目录是否存在
   - 所有命令都会在指定的工作目录下执行

2. **Shell 选择**：
   - Windows 优先使用 PowerShell，回退到 cmd
   - Unix-like 系统优先使用 bash，回退到 sh
   - 可以手动指定 shell（如 `SetShell("zsh")`）

3. **执行终止**：
   - `Destruction()` 会终止当前正在执行的进程
   - 当 context 被取消时，也会自动终止进程
   - 先尝试发送中断信号（Unix）或 Ctrl+Break（Windows），如果失败则强制终止

4. **超时设置**：
   - 可以为每个执行器设置默认超时时间
   - 超时后会取消执行并返回错误
   - 设置为 0 表示无超时限制

5. **伪终端模式（PTY）**：
   - 启用 PTY 后，命令会在伪终端中执行
   - 适用于需要终端交互的程序
   - 可以设置终端尺寸（宽度/高度）

6. **环境变量**：
   - 继承当前进程的所有环境变量
   - 通过 `SetEnv` 设置的变量会覆盖现有变量

## 架构说明

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   LocalBridge   │────▶│   LocalAdapter  │────▶│  LocalExecutor  │
│   (连接管理)     │     │   (配置管理)     │     │   (执行管理)     │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                                                        │
                                                        ▼
                                               ┌─────────────────┐
                                               │  os/exec 包      │
                                               │  (Go 标准库)      │
                                               └─────────────────┘
                                                        │
                                                        ▼
                                               ┌─────────────────┐
                                               │   本地 Shell     │
                                               │  (bash/sh/cmd)  │
                                               └─────────────────┘
```

## 测试

```bash
# 运行 Local Executor 测试
go test ./executor/local/ -v

# 运行所有测试
go test ./... -v
```

注意：测试会在本地机器上实际执行命令，请确保测试环境中的命令安全。