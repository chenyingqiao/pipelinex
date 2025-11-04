package pipelinex

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

// 预检查RuntimeImpl是否实现了Runtime接口
var _ Runtime = (*RuntimeImpl)(nil)

// PipelineConfig 流水线配置结构
type PipelineConfig struct {
	Param map[string]interface{} `yaml:"Param"`
	Graph string                 `yaml:"Graph"`
	Nodes map[string]NodeConfig  `yaml:"Nodes"`
}

// NodeConfig 节点配置结构
type NodeConfig struct {
	Image  string                 `yaml:"Image"`
	Config map[string]interface{} `yaml:"Config"`
	Cmd    string                 `yaml:"Cmd"`
}

// RuntimeImpl Runtime接口的实现
type RuntimeImpl struct {
	pipelines  map[string]Pipeline // 存储所有流水线
	mu         sync.RWMutex        // 读写锁
	ctx        context.Context     // 上下文
	cancel     context.CancelFunc  // 取消函数
	doneChan   chan struct{}       // 完成通道
	background chan struct{}       // 后台处理完成通道
}

// NewRuntime 创建新的Runtime实例
func NewRuntime(ctx context.Context) Runtime {
	ctx, cancel := context.WithCancel(ctx)
	return &RuntimeImpl{
		pipelines:  make(map[string]Pipeline),
		ctx:        ctx,
		cancel:     cancel,
		doneChan:   make(chan struct{}),
		background: make(chan struct{}),
	}
}

// Get 获取流水线状态
func (r *RuntimeImpl) Get(id string) (Pipeline, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pipeline, exists := r.pipelines[id]
	if !exists {
		return nil, fmt.Errorf("pipeline with id %s not found", id)
	}
	return pipeline, nil
}

// Cancel 取消运行中的流水线
func (r *RuntimeImpl) Cancel(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	pipeline, exists := r.pipelines[id]
	if !exists {
		return fmt.Errorf("pipeline with id %s not found", id)
	}

	// 调用流水线的Cancel方法
	if p, ok := pipeline.(*PipelineImpl); ok {
		p.Cancel()
	}

	return nil
}

// RunAsync 执行异步流水线
func (r *RuntimeImpl) RunAsync(ctx context.Context, id string, config string, listener Listener) (Pipeline, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查是否已存在相同ID的流水线
	if _, exists := r.pipelines[id]; exists {
		return nil, fmt.Errorf("pipeline with id %s already exists", id)
	}

	// 解析配置
	pipelineConfig, err := r.parseConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// 创建流水线
	pipeline := NewPipeline(ctx)

	// 设置监听器
	if listener != nil {
		pipeline.Listening(listener)
	}

	// 构建图结构
	graph := r.buildGraph(pipelineConfig)
	pipeline.SetGraph(graph)

	// 存储流水线
	r.pipelines[id] = pipeline

	// 异步执行流水线
	go func() {
		defer func() {
			r.mu.Lock()
			delete(r.pipelines, id)
			r.mu.Unlock()
		}()

		if err := pipeline.Run(ctx); err != nil {
			fmt.Printf("Pipeline %s execution failed: %v\n", id, err)
		}
	}()

	return pipeline, nil
}

// RunSync 执行同步流水线
func (r *RuntimeImpl) RunSync(ctx context.Context, id string, config string, listener Listener) (Pipeline, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查是否已存在相同ID的流水线
	if _, exists := r.pipelines[id]; exists {
		return nil, fmt.Errorf("pipeline with id %s already exists", id)
	}

	// 解析配置
	pipelineConfig, err := r.parseConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// 创建流水线
	pipeline := NewPipeline(ctx)

	// 设置监听器
	if listener != nil {
		pipeline.Listening(listener)
	}

	// 构建图结构
	graph := r.buildGraph(pipelineConfig)
	pipeline.SetGraph(graph)

	// 存储流水线
	r.pipelines[id] = pipeline

	// 同步执行流水线
	defer func() {
		r.mu.Lock()
		delete(r.pipelines, id)
		r.mu.Unlock()
	}()

	err = pipeline.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("pipeline execution failed: %w", err)
	}

	return pipeline, nil
}

// Rm 移除流水线记录
func (r *RuntimeImpl) Rm(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.pipelines, id)
}

// Done runtime已经执行完成
func (r *RuntimeImpl) Done() chan struct{} {
	return r.doneChan
}

// Notify 通知runtime
func (r *RuntimeImpl) Notify(data interface{}) error {
	// 这里可以根据data的内容进行不同的处理
	// 例如：更新流水线状态、触发事件等
	switch v := data.(type) {
	case string:
		fmt.Printf("Runtime notification: %s\n", v)
	case map[string]interface{}:
		if msg, ok := v["message"].(string); ok {
			fmt.Printf("Runtime notification: %s\n", msg)
		}
	default:
		fmt.Printf("Runtime notification: %+v\n", v)
	}
	return nil
}

// Ctx 返回runtime公共上下文
func (r *RuntimeImpl) Ctx() context.Context {
	return r.ctx
}

// StopBackground 停止后台处理
func (r *RuntimeImpl) StopBackground() {
	r.cancel()
	close(r.doneChan)
	close(r.background)
}

// parseConfig 解析流水线配置
func (r *RuntimeImpl) parseConfig(config string) (*PipelineConfig, error) {
	var pipelineConfig PipelineConfig

	err := yaml.Unmarshal([]byte(config), &pipelineConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml config: %w", err)
	}

	return &pipelineConfig, nil
}

// buildGraph 构建图结构
func (r *RuntimeImpl) buildGraph(config *PipelineConfig) Graph {
	graph := NewDGAGraph()

	// 创建节点
	nodeMap := make(map[string]Node)
	for nodeName := range config.Nodes {
		node := NewDGANode(nodeName, StatusUnknown)
		nodeMap[nodeName] = node
		graph.AddVertex(node)
	}

	// 解析图关系并添加边
	if config.Graph != "" {
		r.parseGraphEdges(graph, nodeMap, config.Graph)
	}

	return graph
}

// parseGraphEdges 解析图边关系
func (r *RuntimeImpl) parseGraphEdges(graph Graph, nodeMap map[string]Node, graphStr string) {
	// 简单的图解析，格式如 "A->B\nB->C"
	// 这里可以实现更复杂的解析逻辑
	// 暂时留空，可以根据实际需要实现
	_ = graphStr // 避免未使用变量警告
}

// StartBackground 启动后台处理
func (r *RuntimeImpl) StartBackground() {
	go func() {
		defer close(r.background)

		ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
		defer ticker.Stop()

		for {
			select {
			case <-r.ctx.Done():
				return
			case <-ticker.C:
				// 定期清理已完成的流水线
				r.cleanupCompletedPipelines()
			}
		}
	}()
}

// cleanupCompletedPipelines 清理已完成的流水线
func (r *RuntimeImpl) cleanupCompletedPipelines() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, pipeline := range r.pipelines {
		select {
		case <-pipeline.Done():
			// 流水线已完成，可以清理
			delete(r.pipelines, id)
		default:
			// 流水线仍在运行
		}
	}
}
