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
	first    Node
	nodes    map[string]Node
	graph    map[string][]string
	sequence []string
	hasCycle bool
}

func NewDGAGraph() *DGAGraph {
	return &DGAGraph{
		nodes:    map[string]Node{},
		graph:    map[string][]string{},
		sequence: []string{},
	}
}

// Nodes 返回所有的节点map
func (dga *DGAGraph) Nodes() map[string]Node {
	dga.hasCycle = false
	return funk.Map(dga.nodes, func(k string, v Node) (string, Node) {
		return k, v
	}).(map[string]Node)
}

// AddVertex 向图中添加顶点（节点）
// 检查是否存在循环；如果存在循环，则返回 ErrHasCycle
// 否则返回 nil
func (dga *DGAGraph) AddVertex(node Node) {
	if dga.first == nil {
		dga.first = node
	}
	dga.nodes[node.Id()] = node
	dga.graph[node.Id()] = []string{}
}

// AddEdge 向图中添加边
func (dga *DGAGraph) AddEdge(src, dest Node) error {
	if _, ok := dga.nodes[src.Id()]; !ok {
		return fmt.Errorf("source vertex %s not found", src.Id())
	}
	if _, ok := dga.nodes[dest.Id()]; !ok {
		return fmt.Errorf("dest vertex %s not found", dest.Id())
	}
	dga.graph[src.Id()] = append(dga.graph[src.Id()], dest.Id())
	dga.hasCycle = dga.cycleCheck()
	if dga.hasCycle {
		return ErrHasCycle
	}
	return nil
}

// Traversal 对DAG执行广度优先遍历
// 为图中的每个节点执行提供的 TraversalFn 函数
func (dga *DGAGraph) Traversal(ctx context.Context, fn TraversalFn) error {
	visited := make(map[string]bool)                        // 跟踪已访问的节点
	firstNodeID := dga.first.Id()                           // 获取第一个节点的ID
	queue := []string{firstNodeID}                          // 用第一个节点初始化队列
	visited[firstNodeID] = true                             // 标记第一个节点为已访问
	if err := fn(ctx, dga.nodes[firstNodeID]); err != nil { // 为第一个节点执行函数
		return err
	}
	for len(queue) > 0 { // 继续直到队列为空
		vertexFocuse := queue[0]                           // 从队列中获取下一个节点
		queue = queue[1:]                                  // 从队列中移除节点
		g, ctx := errgroup.WithContext(ctx)                // 为并发goroutine创建新上下文
		for _, neighbor := range dga.graph[vertexFocuse] { // 遍历当前节点的邻居
			if visited[neighbor] { // 检查邻居是否已被访问
				continue
			}
			visited[neighbor] = true        // 标记邻居为已访问
			queue = append(queue, neighbor) // 将邻居添加到队列
			g.Go(func() error {             // 并发地为邻居执行函数
				return fn(ctx, dga.nodes[neighbor])
			})
		}
		if err := g.Wait(); err != nil { // 等待所有goroutine完成
			return err
		}
	}
	return nil
}

// cycleCheck 检查有向无环图（DAG）中是否存在循环
// 如果找到循环则返回 true，否则返回 false
func (dga *DGAGraph) cycleCheck() bool {
	indeg := make(map[string]int)
	for v := range dga.nodes {
		indeg[v] = 0
	}
	for _, adj := range dga.graph {
		for _, n := range adj {
			indeg[n]++
		}
	}
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
	return p.graph
}

// SetGraph 设置流水线的图结构
func (p *PipelineImpl) SetGraph(graph Graph) {
	p.graph = graph
}

// Status 返回流水线的整体状态
func (p *PipelineImpl) Status() string {
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

	err := p.graph.Traversal(ctx, func(ctx context.Context, node Node) error {
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
