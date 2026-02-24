package pipelinex

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/thoas/go-funk"
	"golang.org/x/sync/errgroup"
)

// 预检查PipelineImpl是否实现了Pipeline接口
var _ Pipeline = (*PipelineImpl)(nil)
var _ Graph = (*DGAGraph)(nil)

// 保存了流水线的图结构
type DGAGraph struct {
	mu       sync.RWMutex
	nodes    map[string]Node
	edges    map[string]Edge            // edgeID -> Edge
	graph    map[string][]string        // src -> [dest1, dest2, ...] (保持兼容性)
	edgeMap  map[string]map[string]Edge // src -> dest -> Edge (快速查找)
	sequence []string
	hasCycle bool
}

func NewDGAGraph() *DGAGraph {
	return &DGAGraph{
		nodes:    map[string]Node{},
		edges:    map[string]Edge{},
		graph:    map[string][]string{},
		edgeMap:  map[string]map[string]Edge{},
		sequence: []string{},
	}
}

// Nodes 返回所有的节点map
func (dga *DGAGraph) Nodes() map[string]Node {
	dga.mu.RLock()
	defer dga.mu.RUnlock()
	return funk.Map(dga.nodes, func(k string, v Node) (string, Node) {
		return k, v
	}).(map[string]Node)
}

// AddVertex 向图中添加顶点（节点）
// 检查是否存在循环；如果存在循环，则返回 ErrHasCycle
// 否则返回 nil
func (dga *DGAGraph) AddVertex(node Node) {
	dga.mu.Lock()
	defer dga.mu.Unlock()
	dga.nodes[node.Id()] = node
	dga.graph[node.Id()] = []string{}
}

// AddEdge 向图中添加边
func (dga *DGAGraph) AddEdge(edge Edge) error {
	dga.mu.Lock()
	defer dga.mu.Unlock()

	src := edge.Source()
	dest := edge.Target()

	if _, ok := dga.nodes[src.Id()]; !ok {
		return fmt.Errorf("source vertex %s not found", src.Id())
	}
	if _, ok := dga.nodes[dest.Id()]; !ok {
		return fmt.Errorf("dest vertex %s not found", dest.Id())
	}

	// 添加到edges映射
	dga.edges[edge.ID()] = edge

	// 添加到graph映射（保持兼容性）
	dga.graph[src.Id()] = append(dga.graph[src.Id()], dest.Id())

	// 添加到edgeMap（快速查找）
	if dga.edgeMap[src.Id()] == nil {
		dga.edgeMap[src.Id()] = make(map[string]Edge)
	}
	dga.edgeMap[src.Id()][dest.Id()] = edge

	dga.hasCycle = dga.cycleCheck()
	if dga.hasCycle {
		return ErrHasCycle
	}
	return nil
}

// Traversal 对DAG执行广度优先遍历
// 为图中的每个节点执行提供的 TraversalFn 函数
// 支持多个起始节点并发执行
// 支持条件边：如果边有表达式，会评估表达式决定是否遍历该边
func (dga *DGAGraph) Traversal(ctx context.Context, evalCtx EvaluationContext, fn TraversalFn) error {
	dga.mu.RLock()
	defer dga.mu.RUnlock()

	// 如果没有节点，直接返回
	if len(dga.nodes) == 0 {
		return nil
	}

	// 计算所有节点的入度（基于原始图结构）
	indeg := dga.getIndegrees()

	// 收集所有入度为0的起始节点
	startNodes := make([]string, 0)
	for v, d := range indeg {
		if d == 0 {
			startNodes = append(startNodes, v)
		}
	}

	if len(startNodes) == 0 {
		return nil // 没有起始节点
	}

	visited := make(map[string]bool)
	queue := make([]string, 0)

	// 创建 errgroup 用于并发控制第一层
	g, ctx := errgroup.WithContext(ctx)

	// 并发执行所有起始节点
	for _, startNodeID := range startNodes {
		nodeID := startNodeID // 避免闭包捕获问题
		visited[nodeID] = true
		queue = append(queue, nodeID)

		g.Go(func() error {
			return fn(ctx, dga.nodes[nodeID])
		})
	}

	// 等待所有起始节点完成
	if err := g.Wait(); err != nil {
		return err
	}

	// BFS 遍历剩余节点
	for len(queue) > 0 {
		vertexFocus := queue[0]
		queue = queue[1:]

		// 为当前节点的所有邻居创建 errgroup
		g, ctx := errgroup.WithContext(ctx)

		for _, neighbor := range dga.graph[vertexFocus] {
			if visited[neighbor] {
				continue
			}

			// 获取边并评估条件
			shouldTraverse := true
			if edge, ok := dga.edgeMap[vertexFocus][neighbor]; ok && edge.Expression() != "" {
				result, err := edge.Evaluate(evalCtx)
				if err != nil {
					return fmt.Errorf("failed to evaluate edge condition %s->%s: %w",
						vertexFocus, neighbor, err)
				}
				shouldTraverse = result
			}

			// 条件不满足，跳过此边（不减少入度）
			if !shouldTraverse {
				continue
			}

			// 减少邻居的入度
			indeg[neighbor]--

			// 只有当入度减为0时才访问节点
			if indeg[neighbor] == 0 {
				visited[neighbor] = true
				queue = append(queue, neighbor)

				// 并发执行邻居节点
				g.Go(func() error {
					return fn(ctx, dga.nodes[neighbor])
				})
			}
		}

		// 等待当前层的所有 goroutine 完成
		if err := g.Wait(); err != nil {
			return err
		}
	}

	return nil
}

// cycleCheck 检查有向无环图（DAG）中是否存在循环
// 如果找到循环则返回 true，否则返回 false
func (dga *DGAGraph) cycleCheck() bool {
	indeg := dga.getIndegrees()
	q := make([]string, 0)
	for v, d := range indeg {
		if d == 0 {
			q = append(q, v)
		}
	}
	visited := 0
	for len(q) > 0 {
		v := q[0]
		q = q[1:]
		visited++
		for _, n := range dga.graph[v] {
			indeg[n]--
			if indeg[n] == 0 {
				q = append(q, n)
			}
		}
	}
	return visited != len(dga.nodes)
}

// getIndegrees 计算所有节点的入度
// 返回一个 map，key 是节点ID，value 是入度值
func (dga *DGAGraph) getIndegrees() map[string]int {
	indeg := make(map[string]int)
	for v := range dga.nodes {
		indeg[v] = 0
	}
	for _, adj := range dga.graph {
		for _, n := range adj {
			indeg[n]++
		}
	}
	return indeg
}

// HasCycle 检查图中是否存在循环
func (dga *DGAGraph) HasCycle() bool {
	dga.mu.RLock()
	defer dga.mu.RUnlock()
	return dga.hasCycle
}

type PipelineImpl struct {
	id         string
	graph      Graph
	status     string
	metadata   Metadata
	listening  ListeningFn
	listener   Listener
	doneChan   <-chan struct{}
	cancelFunc context.CancelFunc
	mu         sync.RWMutex
}

func NewPipeline(ctx context.Context) Pipeline {
	return &PipelineImpl{
		id: uuid.NewString(),
	}
}

// Id 返回流水线的ID
func (p *PipelineImpl) Id() string {
	return p.id
}

// GetGraph 返回流水线的图结构
func (p *PipelineImpl) GetGraph() Graph {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.graph
}

// SetGraph 设置流水线的图结构
func (p *PipelineImpl) SetGraph(graph Graph) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.graph = graph
}

// Status 返回流水线的整体状态
func (p *PipelineImpl) Status() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

// Metadata 返回流水线执行的源数据
func (p *PipelineImpl) Metadata() Metadata {
	return Metadata{}
}

// Listening 设置流水线执行事件监听器
func (p *PipelineImpl) Listening(fn Listener) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.listener = fn
}

// Done 返回一个通道，用于通知流水线何时完成
func (p *PipelineImpl) Done() <-chan struct{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.doneChan
}

// Run 执行流水线
func (p *PipelineImpl) Run(ctx context.Context) error {
	p.mu.Lock()
	ctx, cancel := context.WithCancel(ctx)
	p.cancelFunc = cancel
	p.mu.Unlock()

	done := make(chan struct{})
	p.doneChan = done

	defer func() {
		close(done)
		p.mu.Lock()
		p.cancelFunc = nil
		p.mu.Unlock()
	}()

	// 通知流水线开始
	p.notifyEvent(PipelineStart)

	// 创建求值上下文
	evalCtx := NewEvaluationContext().WithPipeline(p)

	err := p.graph.Traversal(ctx, evalCtx, func(ctx context.Context, node Node) error {
		// 检查context是否已取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 通知节点开始
		p.notifyEvent(PipelineNodeStart)
		fmt.Println(node.Id())

		// 模拟工作执行，检查context取消
		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				fmt.Printf("Pipeline cancelled for node %s\n", node.Id())
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
				// 继续执行
			}
		}

		// 通知节点完成
		p.notifyEvent(PipelineNodeFinish)
		return nil
	})

	// 通知流水线完成
	p.notifyEvent(PipelineFinish)
	return err
}

// 这个主要是在运行过程中节点状态或者流水线状态变化，就会触发这个函数
// 节点
// 我们就可以在这里做一些处理
// 执行ListeningFn函数
func (p *PipelineImpl) Notify() {
	p.mu.RLock()
	listening := p.listening
	listener := p.listener
	p.mu.RUnlock()

	// 如果设置了ListeningFn则调用它
	if listening != nil {
		listening(p)
	}

	// 如果设置了事件监听器则处理它
	if listener != nil {
		// 通知当前状态
		p.notifyCurrentStatus(listener)
	}
}

// 终止流水线
func (p *PipelineImpl) Cancel() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cancelFunc != nil {
		p.cancelFunc()
		p.status = StatusCancelled

		// 通知监听器关于取消事件
		if p.listener != nil {
			p.listener.Handle(p, EventPipelineCancelled)
		}

		if p.listening != nil {
			p.listening(p)
		}
	}
}

// notifyEvent 通知监听器特定事件
func (p *PipelineImpl) notifyEvent(event Event) {
	p.mu.RLock()
	listener := p.listener
	listening := p.listening
	p.mu.RUnlock()

	if listener != nil {
		listener.Handle(p, event)
	}

	if listening != nil {
		listening(p)
	}
}

// notifyCurrentStatus 通知监听器当前流水线状态
func (p *PipelineImpl) notifyCurrentStatus(listener Listener) {
	// 此方法可用于通知详细的状态变化
	// 目前，它仅用当前流水线调用监听器
	listener.Handle(p, EventPipelineStatusUpdate)
}
