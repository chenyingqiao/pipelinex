# Docker Executor

Docker Executor 使用 [Docker Go SDK](https://github.com/docker/docker/client) 实现了在 Docker 容器中执行 Pipeline 节点的功能。

## 功能特性

- 在 Docker 容器中执行流水线步骤
- 支持自定义镜像（自动拉取）
- 支持卷挂载（bind mount）
- 支持环境变量配置
- 支持自定义网络模式
- 支持私有镜像仓库
- 支持多步骤顺序执行
- 支持 TTY 模式（模拟真实终端，支持颜色输出和交互式程序）
- 实时流式输出执行结果

## 依赖

```bash
go get github.com/docker/docker/client
go get github.com/docker/docker/api/types/container
go get github.com/docker/docker/api/types/image
go get github.com/docker/docker/api/types/mount
```

## 配置示例

### 全局执行器配置

```yaml
Executors:
  docker:
    type: docker
    config:
      registry: myregistry.com      # 可选：镜像仓库地址
      network: host                  # 可选：网络模式 (bridge/host/none/自定义)
      workdir: /app                  # 可选：工作目录
      tty: true                      # 可选：启用 TTY 模式（支持颜色输出和交互式程序）
      ttyWidth: 120                  # 可选：TTY 终端宽度（默认 80）
      ttyHeight: 40                  # 可选：TTY 终端高度（默认 24）
      volumes:                       # 可选：卷挂载列表
        - /var/run/docker.sock:/var/run/docker.sock
        - /host/data:/container/data
      env:                           # 可选：环境变量
        GO_VERSION: "1.21"
        NODE_ENV: production
```

### 节点配置

```yaml
Nodes:
  Build:
    executor: docker
    image: golang:1.21-alpine
    steps:
      - name: build
        run: go build -o app .
      - name: test
        run: go test ./...

  Deploy:
    executor: docker
    image: bitnami/kubectl:latest
    steps:
      - name: deploy
        run: kubectl apply -f k8s/
```

## API 使用

### 基本使用

```go
import (
    "context"
    "log"

    "github.com/chenyingqiao/pipelinex"
    "github.com/chenyingqiao/pipelinex/executor/docker"
)

func main() {
    ctx := context.Background()

    // 创建执行器
    executor, err := docker.NewDockerExecutor()
    if err != nil {
        log.Fatal(err)
    }

    // 配置执行器
    executor.SetImage("golang:1.21-alpine")
    executor.SetWorkdir("/workspace")
    executor.SetEnv("CGO_ENABLED", "0")
    executor.SetVolume("/host/go", "/go")

    // 准备环境（创建并启动容器）
    if err := executor.Prepare(ctx); err != nil {
        log.Fatal(err)
    }
    defer executor.Destruction(ctx) // 确保清理容器

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
        if r, ok := result.(*docker.StepResult); ok {
            if r.Error != nil {
                log.Printf("Step %s failed: %v\n", r.StepName, r.Error)
            } else {
                log.Printf("Step %s succeeded:\n%s\n", r.StepName, r.Output)
            }
        }
    }
}
```

### 使用 Bridge 和 Adapter 模式

```go
// 创建桥接器
bridge := docker.NewDockerBridge()

// 创建适配器并配置
adapter := docker.NewDockerAdapter()
config := map[string]any{
    "registry": "myregistry.com",
    "network": "host",
    "workdir": "/app",
    "volumes": []string{
        "/var/run/docker.sock:/var/run/docker.sock",
        "/host/data:/container/data",
    },
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

// 设置镜像并执行
if dockerExec, ok := executor.(*docker.DockerExecutor); ok {
    dockerExec.SetImage("golang:1.21-alpine")
}
```

## 接口实现

Docker Executor 实现了以下接口：

| 接口 | 实现类型 | 说明 |
|------|---------|------|
| `pipelinex.Executor` | `DockerExecutor` | 执行器主接口 |
| `pipelinex.Adapter` | `DockerAdapter` | 配置适配器接口 |
| `pipelinex.Bridge` | `DockerBridge` | 连接桥接器接口 |

## DockerExecutor 方法

### 构造函数

```go
// 使用默认 Docker 客户端创建执行器
func NewDockerExecutor() (*DockerExecutor, error)

// 使用指定的 Docker 客户端创建执行器
func NewDockerExecutorWithClient(cli *client.Client) *DockerExecutor
```

### 配置方法

```go
func (d *DockerExecutor) SetImage(image string)           // 设置镜像
func (d *DockerExecutor) SetWorkdir(workdir string)       // 设置工作目录
func (d *DockerExecutor) SetEnv(key, value string)        // 设置环境变量
func (d *DockerExecutor) SetVolume(hostPath, containerPath string)  // 设置卷挂载
func (d *DockerExecutor) SetNetwork(network string)       // 设置网络模式
func (d *DockerExecutor) SetRegistry(registry string)     // 设置镜像仓库
func (d *DockerExecutor) SetTTY(enabled bool)             // 设置是否启用 TTY 模式（默认 false）
func (d *DockerExecutor) SetTTYSize(width, height uint)   // 设置 TTY 终端尺寸（默认 80x24）
```

### 生命周期方法

```go
func (d *DockerExecutor) Prepare(ctx context.Context) error      // 创建并启动容器
func (d *DockerExecutor) Transfer(ctx context.Context, resultChan chan<- any, commandChan <-chan any)  // 执行命令
func (d *DockerExecutor) Destruction(ctx context.Context) error  // 停止并删除容器
```

### TTY 使用示例

```go
// 创建执行器
executor, err := docker.NewDockerExecutor()
if err != nil {
    log.Fatal(err)
}

// 启用 TTY 模式（支持颜色输出和交互式程序）
executor.SetTTY(true)

// 设置终端尺寸（可选，默认 80x24）
executor.SetTTYSize(120, 40)

// 设置镜像并执行
executor.SetImage("golang:1.21-alpine")

// 准备并执行
if err := executor.Prepare(ctx); err != nil {
    log.Fatal(err)
}
defer executor.Destruction(ctx)

// 现在执行的命令会像在真实终端中一样运行
// 支持：
// - 带颜色的输出（ls --color=auto）
// - 需要终端的程序（top、htop 等）
// - 正确的终端格式和换行
```

## StepResult 结构

```go
type StepResult struct {
    StepName   string    // 步骤名称
    Command    string    // 执行的命令
    Output     string    // 命令输出（stdout + stderr）
    Error      error     // 执行错误（如果 exit code != 0）
    StartTime  time.Time // 开始时间
    FinishTime time.Time // 结束时间
}
```

## 注意事项

1. **Docker 环境**：需要主机上安装并运行 Docker，执行器通过 Docker API 与守护进程通信

2. **权限要求**：
   - 需要访问 Docker socket 的权限（通常是 `/var/run/docker.sock`）
   - 或者配置 Docker HTTP API 访问

3. **容器生命周期**：
   - `Prepare()` 创建并启动容器
   - `Transfer()` 在容器中执行命令
   - `Destruction()` 停止并删除容器
   - 建议使用 `defer executor.Destruction(ctx)` 确保清理

4. **镜像拉取**：
   - 如果镜像不存在，会自动拉取
   - 支持配置私有仓库（通过 `SetRegistry` 或配置中的 `registry`）

5. **Shell 选择**：
   - 默认使用 `/bin/bash`
   - Alpine/BusyBox 镜像自动使用 `/bin/sh`

6. **错误处理**：
   - 命令返回非零退出码时，`StepResult.Error` 会包含错误信息
   - 容器启动失败会在 `Prepare()` 阶段返回错误

## 架构说明

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   DockerBridge  │────▶│  DockerAdapter  │────▶│ DockerExecutor  │
│   (连接管理)     │     │   (配置管理)     │     │   (执行管理)     │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                                                        │
                                                        ▼
                                               ┌─────────────────┐
                                               │  Docker Client  │
                                               │  (Go SDK)       │
                                               └─────────────────┘
                                                        │
                                                        ▼
                                               ┌─────────────────┐
                                               │  Docker Daemon  │
                                               │  (容器管理)      │
                                               └─────────────────┘
```

## 测试

```bash
# 运行 Docker Executor 测试
go test ./test/docker_executor_test.go -v

# 运行所有测试
go test ./test/ -v
```

注意：部分测试需要 Docker 环境，如果没有 Docker 会跳过相关测试。