package pipelinex

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tetrafolium/mermaid-check/ast"
	"github.com/tetrafolium/mermaid-check/parser"
	"gopkg.in/yaml.v2"
)

// 预检查RuntimeImpl是否实现了Runtime接口
var _ Runtime = (*RuntimeImpl)(nil)

// RuntimeImpl Runtime接口的实现
type RuntimeImpl struct {
	pipelines   map[string]Pipeline // 存储所有流水线
	pipelineIds map[string]bool     // 跟踪所有使用过的流水线ID
	mu          sync.RWMutex        // 读写锁
	ctx         context.Context     // 上下文
	cancel      context.CancelFunc  // 取消函数
	doneChan    chan struct{}       // 完成通道
	background  chan struct{}       // 后台处理完成通道
	pusher      Pusher              // 日志推送器
}

// NewRuntime 创建新的Runtime实例
func NewRuntime(ctx context.Context) Runtime {
	ctx, cancel := context.WithCancel(ctx)
	return &RuntimeImpl{
		pipelines:   make(map[string]Pipeline),
		pipelineIds: make(map[string]bool),
		ctx:         ctx,
		cancel:      cancel,
		doneChan:    make(chan struct{}),
		background:  make(chan struct{}),
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
	if _, exists := r.pipelineIds[id]; exists {
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
	graph := r.BuildGraph(pipelineConfig)
	pipeline.SetGraph(graph)

	// 设置metadata
	if err := r.setupMetadata(ctx, pipeline, pipelineConfig); err != nil {
		return nil, fmt.Errorf("failed to setup metadata: %w", err)
	}

	// 存储流水线并标记ID为已使用
	r.pipelines[id] = pipeline
	r.pipelineIds[id] = true

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
	if _, exists := r.pipelineIds[id]; exists {
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
	graph := r.BuildGraph(pipelineConfig)
	pipeline.SetGraph(graph)

	// 设置metadata
	if err := r.setupMetadata(ctx, pipeline, pipelineConfig); err != nil {
		return nil, fmt.Errorf("failed to setup metadata: %w", err)
	}

	// 存储流水线并标记ID为已使用
	r.pipelines[id] = pipeline
	r.pipelineIds[id] = true

	err = pipeline.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("pipeline execution failed: %w", err)
	}

	// 清理已完成的流水线，但保留ID记录
	delete(r.pipelines, id)

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
	select {
	case <-r.doneChan:
		// Channel already closed
	default:
		close(r.doneChan)
	}
}

// setupMetadata 设置流水线的metadata
func (r *RuntimeImpl) setupMetadata(ctx context.Context, pipeline Pipeline, config *PipelineConfig) error {
	// 检查是否有metadata配置（注意配置中是Metadate）
	if config.Metadate.Type == "" {
		return nil
	}

	// 创建metadata store
	factory := NewMetadataStoreFactory()
	store, err := factory.Create(config.Metadate)
	if err != nil {
		return fmt.Errorf("failed to create metadata store: %w", err)
	}

	// 设置到pipeline
	pipeline.SetMetadata(store)
	return nil
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

// BuildGraph 构建图结构
func (r *RuntimeImpl) BuildGraph(config *PipelineConfig) Graph {
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
// 使用 mermaid-check 库解析 stateDiagram-v2 语法
// 支持从边标签中解析条件表达式，例如：A --> B: label[{eq .Param}]
func (r *RuntimeImpl) parseGraphEdges(graph Graph, nodeMap map[string]Node, graphStr string) {
	stateParser := parser.NewStateParser()
	diagram, err := stateParser.Parse(graphStr)
	if err != nil {
		// 解析失败时静默返回，不建立边关系
		return
	}

	// 转换为状态图
	stateDiagram, ok := diagram.(*ast.StateDiagram)
	if !ok {
		return
	}

	// 遍历所有语句，提取转换关系
	for _, stmt := range stateDiagram.Statements {
		// 尝试转换为 Transition
		if transition, ok := stmt.(*ast.Transition); ok {
			// 跳过 [*] 开始/结束节点
			if transition.From == "[*]" || transition.To == "[*]" {
				continue
			}

			srcNode, srcExists := nodeMap[transition.From]
			destNode, destExists := nodeMap[transition.To]

			if !srcExists || !destExists {
				continue
			}

			// 从 Label 中提取条件表达式
			expression := r.extractExpression(transition.Label)

			// 添加边关系（有条件表达式则创建条件边）
			var edge Edge
			if expression != "" {
				edge = NewConditionalEdge(srcNode, destNode, expression)
			} else {
				edge = NewDGAEdge(srcNode, destNode)
			}
			_ = graph.AddEdge(edge)
		}
	}
}

// ExtractExpression 从边标签中提取条件表达式（公共函数供测试使用）
// 格式示例："label[{eq .Param}]" -> "{eq .Param}"
// 提取 [] 中的所有内容
func ExtractExpression(label string) string {
	if label == "" {
		return ""
	}

	// 查找左括号 [
	startIdx := strings.Index(label, "[")
	if startIdx == -1 {
		return ""
	}

	// 查找右括号 ]
	endIdx := strings.Index(label[startIdx:], "]")
	if endIdx == -1 {
		return ""
	}
	endIdx += startIdx

	// 提取 [] 中的内容
	if endIdx <= startIdx+1 {
		return ""
	}

	return label[startIdx+1 : endIdx]
}

// extractExpression 从边标签中提取条件表达式（内部使用）
func (r *RuntimeImpl) extractExpression(label string) string {
	return ExtractExpression(label)
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

// SetPusher 设置日志推送器
func (r *RuntimeImpl) SetPusher(pusher Pusher) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pusher = pusher
}
