# PipelineX

一个灵活且可扩展的 Go 语言 CI/CD 流水线执行库，支持多种执行后端和基于 DAG 的工作流编排。

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.23-blue)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

## 特性

- **DAG 工作流**：使用 Mermaid 语法定义复杂的流水线有向无环图结构
- **多后端执行**：支持本地、Docker 和 Kubernetes 执行器
- **并发执行**：独立任务并行运行以获得最佳性能
- **条件边**：使用模板表达式实现动态执行路径
- **事件驱动架构**：通过事件监听器监控流水线生命周期
- **模板引擎**：使用 Pongo2 模板进行动态配置渲染
- **元数据管理**：进程安全的元数据存储和检索
- **日志流式传输**：实时日志输出，支持自定义日志推送

## 安装

```bash
go get github.com/chenyingqiao/pipelinex
```

## 快速开始

```go
package main

import (
    "context"
    "fmt"
    "github.com/chenyingqiao/pipelinex"
)

func main() {
    ctx := context.Background()

    // 创建运行时
    runtime := pipelinex.NewRuntime(ctx)

    // 流水线配置
    config := `
Version: "1.0"
Name: example-pipeline

Executors:
  local:
    type: local
    config:
      shell: bash

Graph: |
  stateDiagram-v2
    [*] --> Build
    Build --> Test
    Test --> [*]

Nodes:
  Build:
    executor: local
    steps:
      - name: build
        run: echo "Building..."
  Test:
    executor: local
    steps:
      - name: test
        run: echo "Testing..."
`

    // 同步执行流水线
    pipeline, err := runtime.RunSync(ctx, "pipeline-1", config, nil)
    if err != nil {
        fmt.Printf("Pipeline failed: %v\n", err)
        return
    }

    fmt.Println("Pipeline completed successfully!")
}
```

## 配置说明

PipelineX 使用 YAML 配置，结构如下：

```yaml
Version: "1.0"              # 配置版本
Name: my-pipeline           # 流水线名称

Metadate:                   # 元数据配置
  type: in-config           # 存储类型：in-config, redis, http
  data:
    key: value

Param:                      # 流水线参数
  buildId: "123"
  branch: "main"

Executors:                  # 全局执行器定义
  local:
    type: local
    config:
      shell: bash
      workdir: /tmp

  docker:
    type: docker
    config:
      registry: docker.io
      network: host
      volumes:
        - /var/run/docker.sock:/var/run/docker.sock

Graph: |                    # DAG 定义（Mermaid stateDiagram-v2 语法）
  stateDiagram-v2
    [*] --> Build
    Build --> Test
    Test --> Deploy
    Deploy --> [*]

Nodes:                      # 节点定义
  Build:
    executor: local
    steps:
      - name: build
        run: go build .

  Test:
    executor: docker
    image: golang:1.21
    steps:
      - name: test
        run: go test ./...
```

## 执行器

### 本地执行器
在本地机器上执行命令。

```yaml
Executors:
  local:
    type: local
    config:
      shell: bash          # 使用的 Shell（bash, sh, zsh）
      workdir: /tmp        # 工作目录
      env:                 # 环境变量
        KEY: value
      pty: true            # 启用 PTY 支持交互式程序
```

### Docker 执行器
在 Docker 容器内执行命令。

```yaml
Executors:
  docker:
    type: docker
    config:
      registry: docker.io   # 镜像仓库
      network: host         # 网络模式
      workdir: /app         # 容器内工作目录
      tty: true             # 启用 TTY
      ttyWidth: 120         # TTY 宽度
      ttyHeight: 40         # TTY 高度
      volumes:              # 卷挂载
        - /host/path:/container/path
      env:                  # 环境变量
        GO_VERSION: "1.21"
```

### Kubernetes 执行器
在 Kubernetes Pod 内执行命令。

```yaml
Executors:
  k8s:
    type: k8s
    config:
      namespace: default
      serviceAccount: pipeline-sa
      resources:
        cpu: "1000m"
        memory: "2Gi"
```

## 条件边

使用模板表达式定义条件执行路径：

```yaml
Graph: |
  stateDiagram-v2
    [*] --> Build
    Build --> Deploy: "{{ eq .Param.branch 'main' }}"
    Build --> Test: "{{ ne .Param.branch 'main' }}"
    Test --> [*]
    Deploy --> [*]
```

## 事件监控

通过事件监听器监控流水线执行：

```go
listener := pipelinex.NewListener()
listener.Handle(func(p pipelinex.Pipeline, event pipelinex.Event) {
    switch event {
    case pipelinex.PipelineStart:
        fmt.Println("流水线开始")
    case pipelinex.PipelineFinish:
        fmt.Println("流水线完成")
    case pipelinex.PipelineNodeStart:
        fmt.Println("节点开始执行")
    }
})

pipeline, err := runtime.RunSync(ctx, "id", config, listener)
```

## 架构图

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Runtime   │────▶│   Pipeline   │────▶│     DAG     │
└─────────────┘     └──────────────┘     └─────────────┘
                              │                    │
                              ▼                    ▼
                       ┌──────────────┐     ┌─────────────┐
                       │   Executor   │     │    Node     │
                       │   Provider   │     └─────────────┘
                       └──────────────┘            │
                              │                    │
           ┌──────────────────┼────────────────────┤
           ▼                  ▼                    ▼
    ┌──────────┐      ┌──────────┐         ┌──────────┐
    │  Local   │      │  Docker  │         │   K8s    │
    └──────────┘      └──────────┘         └──────────┘
```

## API 参考

### Runtime

```go
type Runtime interface {
    Get(id string) (Pipeline, error)                          // 获取流水线状态
    Cancel(ctx context.Context, id string) error              // 取消运行中的流水线
    RunAsync(ctx context.Context, id string, config string, listener Listener) (Pipeline, error)  // 异步执行
    RunSync(ctx context.Context, id string, config string, listener Listener) (Pipeline, error)   // 同步执行
    Rm(id string)                                             // 移除流水线记录
    Done() chan struct{}                                      // 运行时完成信号
    Notify(data interface{}) error                            // 通知运行时
    Ctx() context.Context                                     // 获取运行时上下文
    StopBackground()                                          // 停止后台处理
    StartBackground()                                         // 启动后台处理
    SetPusher(pusher Pusher)                                  // 设置日志推送器
    SetTemplateEngine(engine TemplateEngine)                  // 设置模板引擎
}
```

### Pipeline

```go
type Pipeline interface {
    Run(ctx context.Context) error                            // 运行流水线
    Cancel()                                                  // 取消流水线
    Done() chan struct{}                                      // 流水线完成信号
    SetGraph(graph Graph)                                     // 设置 DAG 图
    GetGraph() Graph                                          // 获取 DAG 图
    SetExecutorProvider(provider ExecutorProvider)            // 设置执行器提供者
    Listening(listener Listener)                              // 设置事件监听器
    SetMetadata(metadata MetadataStore)                       // 设置元数据存储
}
```

## 测试

```bash
go test ./...
```

## 贡献

欢迎贡献！请随时提交 Pull Request。

## 许可证

MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。
