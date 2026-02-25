# 更新日志

## [Unreleased] - 2026-02-25

本版本是 Pipeline 执行引擎的重大更新，引入了模板引擎、条件边执行、元数据管理等核心功能，同时大幅增强了并发安全性和可观测性。

### 新增功能

#### 1. 模板引擎支持
- 新增模板引擎接口 (`templete.go`, `templete_impl.go`)
- 支持基于文本的模板渲染，可用于动态生成节点配置
- 提供 `TemplateEngine` 接口，便于扩展不同的模板实现

#### 2. 条件边执行（Conditional Edge）
- 新增条件边功能 (`edge.go`, `edge_impl.go`)
- 支持基于表达式评估的边条件判断
- 添加 `EvalContext` 评估上下文 (`eval_context.go`)
- 支持根据节点执行结果动态决定执行路径

#### 3. 元数据管理（Metadata）
- 新增元数据系统 (`metadata.go`, `metadata_impl.go`)
- 支持在 Pipeline 执行过程中存储和传递元数据
- 提供进程安全的元数据读写操作

#### 4. Pipeline Runtime 重构
- 全新实现的 Pipeline Runtime (`runtime.go`, `runtime_impl.go`)
- 支持进程安全的并发执行
- 添加日志推送设置方法
- 支持 Pipeline 执行状态的实时监控

#### 5. 配置系统增强
- 重构配置相关逻辑到独立的 `config.go`
- 更新配置文件结构，支持执行器和多步骤配置
- 添加配置示例文件 (`config.example.yaml`)
- 支持节点状态信息持久化到配置

#### 6. 可视化增强
- 新增 Graph 文本绘图支持
- 支持在控制台输出 Pipeline 执行过程的可视化展示

#### 7. 执行器接口更新
- 重构执行器接口 (`executor.go`)
- 为不同类型的执行器（K8s、Docker、SSH、Local）提供更清晰的接口定义
- 完善 Kubernetes 执行器文档

### 改进与优化

#### 并发安全
- Pipeline Runtime 添加进程安全支持
- 使用 `sync.WaitGroup` 替代 errgroup 进行并发控制
- 修复入度为0的节点有多个时的调度问题

#### 错误处理
- 完善错误类型定义 (`err.go`)
- 添加更详细的错误信息和上下文

#### 日志系统
- 新增日志接口 (`logger.go`)
- 支持自定义日志推送

### Bug 修复

1. **修复死锁问题** - 修复了在特定并发场景下可能发生的死锁
2. **修复多起始节点问题** - 修复入度为0的节点有多个时的执行逻辑
3. **修复并发安全问题** - 修复 Pipeline 执行中的竞态条件

### 文档更新

- 新增 `CLAUDE.md` - 项目开发指南
- 新增 `doc/config.md` - 配置文件详细说明
- 新增 `doc/templete.md` - 模板引擎使用文档
- 更新 `README.md` - 项目介绍和快速开始
- 完善 Kubernetes 执行器文档

### 测试覆盖

新增大量单元测试：
- `test/conditional_edge_test.go` - 条件边功能测试
- `test/edge_test.go` - 边功能基础测试
- `test/eval_context_test.go` - 评估上下文测试
- `test/graph_edge_test.go` - 图边功能测试
- `test/runtime_test.go` - Pipeline Runtime 测试
- `test/templete_test.go` - 模板引擎测试
- 更新 `test/pipeline_test.go` - 基础 Pipeline 测试

### 依赖更新

- 更新 Go 版本要求至 1.23.0
- 更新 Kubernetes 相关依赖
- 移除未使用的 `golang.org/x/sync` 依赖

### 文件变更统计

- 新增文件：18 个
- 修改文件：17 个
- 删除文件：0 个
- 总行数变化：+3,839 / -116

### 迁移指南

从旧版本迁移到新版本时，请注意以下变更：

1. **配置文件格式更新** - 请参考 `config.example.yaml` 更新配置文件
2. **执行器接口变更** - 自定义执行器需要适配新的接口定义
3. **Pipeline 创建方式** - 建议使用新的 Runtime API 创建 Pipeline

---

**完整提交记录**：
```
33e9201 feat: 不使用errgroup直接使用WaitGroup代替
eda1f77 feat: 添加设置模板引擎
97cc674 feat: 添加state图的标签表达式读取功能
4745b3f feat：元数据支持
281c94c feat: 添加metadata支持
ee0e6eb feat: config相关的逻辑移动到config.go中
a54d377 feat: 添加模板引擎支持/条件边支持
fd7cf42 fix: 修复入度为0的节点有多个的情况
7fe6892 feat: pipeline_runtime 添加进程安全支持
726d4b0 feat: 添加配置示例
4dfd4f4 feat: 添加graph中文本绘图支持
dcd64a8 feat: 添加日志推送设置方法到runtime中
173dc13 feat: 修改更新配置文件结构，添加执行器和多步骤的功能
e38d371 feat: 修改执行器interface
ba0f601 feat: 修复死锁问题
dece094 feat: 配置文件添加节点状态信息
57105f4 feat: 添加单元测试
27fbc77 feat: 修改文档添加模板引擎选型和k8s执行器的实现目标
f1094ab feat: 常量设置
b999420 feat: 修改文件中的注释
909aa04 feat: 提阿娘CLAUDE的配置
```
