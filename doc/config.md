 以下是配置文件各字段的功能说明：

---

# 流水线配置文件字段说明

## 1. 元信息

| 字段 | 类型 | 功能 |
|------|------|------|
| `Version` | string | 配置文件格式版本，用于引擎兼容性判断 |
| `Name` | string | 流水线唯一标识，用于日志、监控、管理 |

---

## 2. AI 智能字段

| 字段 | 类型 | 功能 |
|------|------|------|
| `AI.intent` | string | **核心**：一句话描述流水线业务意图，供 AI 理解上下文，实现智能修改 |
| `AI.constraints` | []string | **核心**：关键约束条件列表，用于配置验证和异常诊断时给出针对性建议 |
| `AI.template` | string | **核心**：模板标识符，用于相似流水线推荐和最佳实践复用 |
| `AI.generatedAt` | string | **核心**：生成时间戳，用于版本追踪和审计 |
| `AI.version` | int | 意图版本号，记录 AI 生成/修改次数 |

**支持功能**：智能修改、异常诊断、模板推荐、配置验证

---

## 3. 参数定义

| 字段 | 类型 | 功能 |
|------|------|------|
| `Param` | map | 全局变量池，支持在配置中通过 `${Param.xxx}` 引用 |

---

## 4. 执行器定义

| 字段 | 类型 | 功能 |
|------|------|------|
| `Executors` | map | 全局执行器注册表，供 Nodes 引用 |
| `Executors.{name}.type` | string | 执行器类型：`local` \| `docker` \| `k8s` |
| `Executors.{name}.config` | object | 执行器全局配置，被 Nodes 继承 |

### 4.1 local 执行器

| 子字段 | 功能 |
|--------|------|
| `config.shell` | 指定 shell 类型（bash/sh/zsh） |
| `config.workdir` | 默认工作目录 |

### 4.2 docker 执行器

| 子字段 | 功能 |
|--------|------|
| `config.registry` | 默认镜像仓库 |
| `config.network` | 容器网络模式 |
| `config.volumes` | 挂载卷列表（支持 Docker Socket 挂载实现 DinD） |

### 4.3 k8s 执行器

| 子字段 | 功能 |
|--------|------|
| `config.namespace` | 默认 K8s 命名空间 |
| `config.resources.cpu` | Pod CPU 限制 |
| `config.resources.memory` | Pod 内存限制 |

---

## 5. 日志配置

| 字段 | 类型 | 功能 |
|------|------|------|
| `Logging.endpoint` | string | 日志接收服务 HTTP 接口地址 |
| `Logging.headers` | map | 请求头（用于认证、租户标识等） |
| `Logging.timeout` | duration | 单次推送超时时间 |
| `Logging.retry` | int | 推送失败重试次数 |

---

## 6. 流程定义

| 字段 | 类型 | 功能 |
|------|------|------|
| `Graph` | string | Mermaid 状态图语法，定义节点执行顺序和依赖关系 |
| `Status` | map | 运行时状态（引擎写入），键为节点名，值为状态枚举 |

### 状态枚举

| 值 | 含义 |
|-----|------|
| `Pending` | 等待执行 |
| `Running` | 执行中 |
| `Finished` | 执行成功 |
| `Failed` | 执行失败 |
| `Cancelled` | 已取消 |

---

## 7. 节点配置

| 字段 | 类型 | 功能 |
|------|------|------|
| `Nodes.{name}` | object | 单个节点完整配置 |
| `Nodes.{name}.executor` | string | 引用 `Executors` 中的执行器名称 |
| `Nodes.{name}.image` | string | Docker/K8s 执行时使用的容器镜像 |
| `Nodes.{name}.steps` | []object | 执行步骤列表 |

### 步骤字段

| 字段 | 类型 | 功能 |
|------|------|------|
| `steps[].name` | string | 步骤标识，用于日志和状态展示 |
| `steps[].run` | string | 实际执行的 shell 命令 |

---

## 8. 字段引用关系图

```
Param ──┬──► Executors.config (全局默认值)
        ├──► Nodes.steps[].run (命令参数)
        └──► Logging.headers (动态认证)

Executors ──► Nodes.executor (执行器选择)

AI.template ──► 推荐系统匹配相似配置
AI.intent ────► 智能修改上下文理解
AI.constraints ──► 配置验证规则库
```

---

## 9. 完整功能映射

| 需求功能 | 依赖字段 |
|---------|---------|
| 智能修改 | `AI.intent` |
| 异常诊断 | `AI.constraints` |
| 模板推荐 | `AI.template` |
| 文档生成 | `AI.intent` + `AI.constraints` |
| 配置验证 | `AI.constraints` |

需要补充其他说明吗？
