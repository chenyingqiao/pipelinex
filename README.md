# PipelineX

A flexible and extensible CI/CD pipeline execution library for Go, supporting multiple execution backends and DAG-based workflow orchestration.

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.23-blue)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

## Features

- **DAG-Based Workflow**: Define complex pipelines using Directed Acyclic Graph (DAG) structure with Mermaid syntax
- **Multi-Backend Execution**: Support for Local, Docker, and Kubernetes executors
- **Concurrent Execution**: Independent tasks run in parallel for optimal performance
- **Conditional Edges**: Dynamic execution paths with template-based condition expressions
- **Event-Driven Architecture**: Monitor pipeline lifecycle through event listeners
- **Template Engine**: Dynamic configuration rendering with Pongo2 templates
- **Metadata Management**: Process-safe metadata storage and retrieval
- **Log Streaming**: Real-time log output with customizable log pushing

## Installation

```bash
go get github.com/chenyingqiao/pipelinex
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "github.com/chenyingqiao/pipelinex"
)

func main() {
    ctx := context.Background()

    // Create runtime
    runtime := pipelinex.NewRuntime(ctx)

    // Pipeline configuration
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

    // Execute pipeline synchronously
    pipeline, err := runtime.RunSync(ctx, "pipeline-1", config, nil)
    if err != nil {
        fmt.Printf("Pipeline failed: %v\n", err)
        return
    }

    fmt.Println("Pipeline completed successfully!")
}
```

## Configuration

PipelineX uses YAML configuration with the following structure:

```yaml
Version: "1.0"              # Configuration version
Name: my-pipeline           # Pipeline name

Metadate:                   # Metadata configuration
  type: in-config           # Store type: in-config, redis, http
  data:
    key: value

Param:                      # Pipeline parameters
  buildId: "123"
  branch: "main"

Executors:                  # Global executor definitions
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

Graph: |                    # DAG definition (Mermaid stateDiagram-v2)
  stateDiagram-v2
    [*] --> Build
    Build --> Test
    Test --> Deploy
    Deploy --> [*]

Nodes:                      # Node definitions
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

## Executors

### Local Executor
Executes commands on the local machine.

```yaml
Executors:
  local:
    type: local
    config:
      shell: bash          # Shell to use (bash, sh, zsh)
      workdir: /tmp        # Working directory
      env:                 # Environment variables
        KEY: value
      pty: true            # Enable PTY for interactive programs
```

### Docker Executor
Executes commands inside Docker containers.

```yaml
Executors:
  docker:
    type: docker
    config:
      registry: docker.io
      network: host
      workdir: /app
      tty: true
      ttyWidth: 120
      ttyHeight: 40
      volumes:
        - /host/path:/container/path
      env:
        GO_VERSION: "1.21"
```

### Kubernetes Executor
Executes commands inside Kubernetes pods.

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

## Conditional Edges

Define conditional execution paths using template expressions:

```yaml
Graph: |
  stateDiagram-v2
    [*] --> Build
    Build --> Deploy: "{{ eq .Param.branch 'main' }}"
    Build --> Test: "{{ ne .Param.branch 'main' }}"
    Test --> [*]
    Deploy --> [*]
```

## Event Monitoring

Monitor pipeline execution through event listeners:

```go
listener := pipelinex.NewListener()
listener.Handle(func(p pipelinex.Pipeline, event pipelinex.Event) {
    switch event {
    case pipelinex.PipelineStart:
        fmt.Println("Pipeline started")
    case pipelinex.PipelineFinish:
        fmt.Println("Pipeline finished")
    case pipelinex.PipelineNodeStart:
        fmt.Println("Node started")
    }
})

pipeline, err := runtime.RunSync(ctx, "id", config, listener)
```

## Architecture

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

## API Reference

### Runtime

```go
type Runtime interface {
    Get(id string) (Pipeline, error)
    Cancel(ctx context.Context, id string) error
    RunAsync(ctx context.Context, id string, config string, listener Listener) (Pipeline, error)
    RunSync(ctx context.Context, id string, config string, listener Listener) (Pipeline, error)
    Rm(id string)
    Done() chan struct{}
    Notify(data interface{}) error
    Ctx() context.Context
    StopBackground()
    StartBackground()
    SetPusher(pusher Pusher)
    SetTemplateEngine(engine TemplateEngine)
}
```

### Pipeline

```go
type Pipeline interface {
    Run(ctx context.Context) error
    Cancel()
    Done() chan struct{}
    SetGraph(graph Graph)
    GetGraph() Graph
    SetExecutorProvider(provider ExecutorProvider)
    Listening(listener Listener)
    SetMetadata(metadata MetadataStore)
}
```

## Testing

```bash
go test ./...
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see [LICENSE](LICENSE) for details.
