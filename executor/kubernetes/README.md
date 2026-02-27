# Kubernetes Executor

Kubernetes Executor 使用 [Kubernetes Go SDK (client-go)](https://github.com/kubernetes/client-go) 实现了在 Kubernetes Pod 中执行 Pipeline 节点的功能。

## 功能特性

- 在 Kubernetes Pod 中执行流水线步骤
- 支持自定义镜像（自动拉取）
- 支持 ConfigMap 挂载
- 支持 Secret 挂载
- 支持 PVC 挂载
- 支持 EmptyDir 挂载
- 支持环境变量配置
- 支持自定义命名空间
- 支持自定义 ServiceAccount
- 支持 TTY 模式（模拟真实终端，支持颜色输出）
- 实时流式输出执行结果
- 支持优雅的 Pod 生命周期管理

## 依赖

```bash
go get k8s.io/client-go@v0.29.0
go get k8s.io/api@v0.29.0
go get k8s.io/apimachinery@v0.29.0
```

## 配置示例

### 全局执行器配置

```yaml
Executors:
  k8s:
    type: kubernetes
    config:
      namespace: pipeline               # 可选：命名空间（默认 default）
      image: alpine:latest              # 可选：默认镜像
      workdir: /workspace               # 可选：工作目录
      serviceAccount: pipeline-sa       # 可选：ServiceAccount
      podReadyTimeout: 120              # 可选：Pod 就绪等待超时时间（秒，默认 60）
      tty: true                         # 可选：启用 TTY 模式
      ttyWidth: 120                     # 可选：TTY 终端宽度（默认 80）
      ttyHeight: 40                     # 可选：TTY 终端高度（默认 24）
      env:                              # 可选：环境变量
        GO_VERSION: "1.21"
        NODE_ENV: production
      configMaps:                       # 可选：ConfigMap 挂载列表
        - name: my-config
          mountPath: /etc/config
          configMapName: my-config      # ConfigMap 名称（默认为 name）
          optional: false               # 是否可选
      secrets:                          # 可选：Secret 挂载列表
        - name: my-secret
          mountPath: /etc/secrets
          secretName: my-secret         # Secret 名称（默认为 name）
          optional: false
      volumes:                          # 可选：PVC 挂载列表
        - name: my-pvc
          mountPath: /data
          claimName: my-pvc             # PVC 名称（默认为 name）
          readOnly: false
      emptyDirs:                        # 可选：EmptyDir 挂载列表
        - name: tmp
          mountPath: /tmp
          medium: Memory                # 存储介质（Memory 或空）
          sizeLimit: "1Gi"              # 大小限制
```

### 节点配置

```yaml
Nodes:
  Build:
    executor: k8s
    image: golang:1.21-alpine
    steps:
      - name: build
        run: go build -o app .
      - name: test
        run: go test ./...

  Deploy:
    executor: k8s
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
    "github.com/chenyingqiao/pipelinex/executor/kubernetes"
)

func main() {
    ctx := context.Background()

    // 创建执行器（使用当前环境的 kubeconfig）
    executor, err := kubernetes.NewKubernetesExecutor()
    if err != nil {
        log.Fatal(err)
    }

    // 配置执行器
    executor.SetImage("golang:1.21-alpine")
    executor.SetWorkdir("/workspace")
    executor.SetEnv("CGO_ENABLED", "0")
    executor.SetNamespace("pipeline")

    // 准备环境（创建 Pod）
    if err := executor.Prepare(ctx); err != nil {
        log.Fatal(err)
    }
    defer executor.Destruction(ctx) // 确保清理 Pod

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
        if r, ok := result.(*kubernetes.StepResult); ok {
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
bridge := kubernetes.NewKubernetesBridge()

// 创建适配器并配置
adapter := kubernetes.NewKubernetesAdapter()
config := map[string]any{
    "namespace": "pipeline",
    "image": "golang:1.21-alpine",
    "workdir": "/app",
    "serviceAccount": "pipeline-sa",
    "configMaps": []map[string]any{
        {
            "name":      "my-config",
            "mountPath": "/etc/config",
        },
    },
    "secrets": []map[string]any{
        {
            "name":      "my-secret",
            "mountPath": "/etc/secrets",
        },
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
if k8sExec, ok := executor.(*kubernetes.KubernetesExecutor); ok {
    k8sExec.SetImage("golang:1.21-alpine")
}
```

### 使用自定义 Kubernetes 配置

```go
import "k8s.io/client-go/tools/clientcmd"

// 从特定 kubeconfig 文件加载配置
config, err := clientcmd.BuildConfigFromFlags("", "/path/to/kubeconfig")
if err != nil {
    log.Fatal(err)
}

// 创建执行器
executor, err := kubernetes.NewKubernetesExecutorWithConfig(config)
if err != nil {
    log.Fatal(err)
}
```

## 接口实现

Kubernetes Executor 实现了以下接口：

| 接口 | 实现类型 | 说明 |
|------|---------|------|
| `pipelinex.Executor` | `KubernetesExecutor` | 执行器主接口 |
| `pipelinex.Adapter` | `KubernetesAdapter` | 配置适配器接口 |
| `pipelinex.Bridge` | `KubernetesBridge` | 连接桥接器接口 |

## KubernetesExecutor 方法

### 构造函数

```go
// 使用当前环境的 kubeconfig 创建执行器
func NewKubernetesExecutor() (*KubernetesExecutor, error)

// 使用指定的 rest.Config 创建执行器
func NewKubernetesExecutorWithConfig(config *rest.Config) (*KubernetesExecutor, error)

// 使用指定的 Kubernetes 客户端创建执行器
func NewKubernetesExecutorWithClient(client kubernetes.Interface, config *rest.Config, namespace string) *KubernetesExecutor
```

### 配置方法

```go
func (k *KubernetesExecutor) SetImage(image string)                    // 设置镜像
func (k *KubernetesExecutor) SetWorkdir(workdir string)                // 设置工作目录
func (k *KubernetesExecutor) SetEnv(key, value string)                 // 设置环境变量
func (k *KubernetesExecutor) SetNamespace(namespace string)            // 设置命名空间
func (k *KubernetesExecutor) SetServiceAccount(sa string)              // 设置 ServiceAccount
func (k *KubernetesExecutor) SetVolume(volume corev1.Volume, mount corev1.VolumeMount)  // 添加卷挂载
func (k *KubernetesExecutor) SetTTY(enabled bool)                      // 设置是否启用 TTY 模式
func (k *KubernetesExecutor) SetTTYSize(width, height uint)            // 设置 TTY 终端尺寸
func (k *KubernetesExecutor) SetPodReadyTimeout(timeout time.Duration) // 设置 Pod 就绪等待超时时间
```

### 生命周期方法

```go
func (k *KubernetesExecutor) Prepare(ctx context.Context) error      // 创建并等待 Pod 运行
func (k *KubernetesExecutor) Transfer(ctx context.Context, resultChan chan<- any, commandChan <-chan any)  // 执行命令
func (k *KubernetesExecutor) Destruction(ctx context.Context) error  // 删除 Pod
```

### 查询方法

```go
func (k *KubernetesExecutor) GetPodName() string      // 获取 Pod 名称
func (k *KubernetesExecutor) GetNamespace() string    // 获取命名空间
```

## TTY 使用示例

```go
// 创建执行器
executor, err := kubernetes.NewKubernetesExecutor()
if err != nil {
    log.Fatal(err)
}

// 启用 TTY 模式（支持颜色输出和交互式程序）
executor.SetTTY(true)

// 设置终端尺寸（可选，默认 80x24）
executor.SetTTYSize(120, 40)

// 设置镜像并执行
executor.SetImage("ubuntu:22.04")

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

## 卷挂载配置

### ConfigMap 挂载

```yaml
configMaps:
  - name: app-config
    mountPath: /etc/app
    configMapName: my-config    # 可选，默认为 name
    optional: false             # 可选
    subPath: config.yaml        # 可选，挂载特定文件
```

### Secret 挂载

```yaml
secrets:
  - name: app-secret
    mountPath: /etc/secrets
    secretName: my-secret       # 可选，默认为 name
    optional: false
    subPath: credentials.json   # 可选
```

### PVC 挂载

```yaml
volumes:
  - name: data-volume
    mountPath: /data
    claimName: my-pvc           # 可选，默认为 name
    readOnly: false
    subPath: subdir             # 可选
```

### EmptyDir 挂载

```yaml
emptyDirs:
  - name: tmp
    mountPath: /tmp
    medium: Memory              # 使用内存（默认空）
    sizeLimit: "1Gi"            # 大小限制
```

## 注意事项

1. **Kubernetes 环境**：需要配置好的 kubeconfig 文件或运行在 Pod 中的服务账号

2. **权限要求**：
   - 需要创建/删除 Pod 的权限
   - 需要 exec 到 Pod 的权限
   - 需要获取/列出 Pod 的权限

3. **Pod 生命周期**：
   - `Prepare()` 创建并等待 Pod 进入 Running 状态
   - `Transfer()` 在 Pod 中执行命令
   - `Destruction()` 删除 Pod
   - 建议使用 `defer executor.Destruction(ctx)` 确保清理

4. **镜像拉取**：
   - 如果镜像不存在，Kubernetes 会自动拉取
   - 支持私有镜像仓库（需要配置 ImagePullSecret）

5. **Shell 选择**：
   - 默认使用 `/bin/bash`
   - Alpine/BusyBox 镜像自动使用 `/bin/sh`

6. **错误处理**：
   - 命令返回非零退出码时，`StepResult.Error` 会包含错误信息
   - Pod 启动失败会在 `Prepare()` 阶段返回错误

7. **上下文取消**：
   - 当 ctx 被取消时，会立即停止执行新命令
   - 当前正在执行的命令会收到 **Ctrl+C 信号** (`\x03`)，触发优雅终止
   - 如果进程不响应 Ctrl+C，可能需要等待 Pod 删除才能强制终止

## 架构说明

```
┌────────────────────┐     ┌────────────────────┐     ┌────────────────────┐
│   KubernetesBridge │────▶│  KubernetesAdapter │────▶│ KubernetesExecutor │
│   (连接管理)        │     │   (配置管理)        │     │   (执行管理)        │
└────────────────────┘     └────────────────────┘     └────────────────────┘
                                                              │
                                                              ▼
                                                     ┌────────────────────┐
                                                     │  Kubernetes Client │
                                                     │    (client-go)     │
                                                     └────────────────────┘
                                                              │
                                                              ▼
                                                     ┌────────────────────┐
                                                     │  Kubernetes API    │
                                                     │   (Pod/Exec)       │
                                                     └────────────────────┘
```

## 测试

```bash
# 运行 Kubernetes Executor 测试
go test ./executor/kubernetes/ -v

# 运行所有测试
go test ./... -v
```

注意：部分测试需要 Kubernetes 环境，如果没有配置 kubeconfig 会跳过相关测试。

## RBAC 配置示例

如果需要在 Kubernetes 集群中使用，建议配置以下 RBAC：

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: pipeline-executor
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: pipeline-executor
  namespace: default
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["create", "delete", "get", "list", "watch"]
- apiGroups: [""]
  resources: ["pods/exec"]
  verbs: ["create"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: pipeline-executor
  namespace: default
subjects:
- kind: ServiceAccount
  name: pipeline-executor
  namespace: default
roleRef:
  kind: Role
  name: pipeline-executor
  apiGroup: rbac.authorization.k8s.io
```
