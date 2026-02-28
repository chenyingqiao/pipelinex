package provider

import (
	"context"
	"fmt"

	"github.com/chenyingqiao/pipelinex/executor"
	"github.com/chenyingqiao/pipelinex/executor/docker"
	"github.com/chenyingqiao/pipelinex/executor/kubernetes"
	"github.com/chenyingqiao/pipelinex/executor/local"
)

// Provider ExecutorProvider 实现
type Provider struct {
	executorConfigs map[string]ExecutorConfig
}

// ExecutorConfig 执行器配置
type ExecutorConfig struct {
	Type   string
	Config map[string]any
}

// NewProvider 创建新的 Provider
func NewProvider() *Provider {
	return &Provider{
		executorConfigs: make(map[string]ExecutorConfig),
	}
}

// RegisterExecutor 注册执行器配置
func (p *Provider) RegisterExecutor(name string, config ExecutorConfig) {
	p.executorConfigs[name] = config
}

// GetExecutor 根据执行器名称创建对应的 Executor 实例
func (p *Provider) GetExecutor(ctx context.Context, name string) (executor.Executor, error) {
	execConfig, exists := p.executorConfigs[name]
	if !exists {
		return nil, fmt.Errorf("executor config not found for: %s", name)
	}

	// 根据类型创建对应的 executor
	switch execConfig.Type {
	case "local":
		return p.createLocalExecutor(ctx, execConfig.Config)
	case "docker":
		return p.createDockerExecutor(ctx, execConfig.Config)
	case "kubernetes", "k8s":
		return p.createKubernetesExecutor(ctx, execConfig.Config)
	default:
		return nil, fmt.Errorf("unsupported executor type: %s", execConfig.Type)
	}
}

// createLocalExecutor 创建本地执行器
func (p *Provider) createLocalExecutor(ctx context.Context, config map[string]any) (executor.Executor, error) {
	adapter := local.NewLocalAdapter()
	if err := adapter.Config(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to config local adapter: %w", err)
	}

	bridge := local.NewLocalBridge()
	exec, err := bridge.Conn(ctx, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create local executor: %w", err)
	}

	if err := exec.Prepare(ctx); err != nil {
		return nil, fmt.Errorf("failed to prepare local executor: %w", err)
	}

	return exec, nil
}

// createDockerExecutor 创建 Docker 执行器
func (p *Provider) createDockerExecutor(ctx context.Context, config map[string]any) (executor.Executor, error) {
	adapter := docker.NewDockerAdapter()
	if err := adapter.Config(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to config docker adapter: %w", err)
	}

	bridge := docker.NewDockerBridge()
	exec, err := bridge.Conn(ctx, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker executor: %w", err)
	}

	if err := exec.Prepare(ctx); err != nil {
		return nil, fmt.Errorf("failed to prepare docker executor: %w", err)
	}

	return exec, nil
}

// createKubernetesExecutor 创建 Kubernetes 执行器
func (p *Provider) createKubernetesExecutor(ctx context.Context, config map[string]any) (executor.Executor, error) {
	adapter := kubernetes.NewKubernetesAdapter()
	if err := adapter.Config(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to config kubernetes adapter: %w", err)
	}

	bridge := kubernetes.NewKubernetesBridge()
	exec, err := bridge.Conn(ctx, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes executor: %w", err)
	}

	if err := exec.Prepare(ctx); err != nil {
		return nil, fmt.Errorf("failed to prepare kubernetes executor: %w", err)
	}

	return exec, nil
}
