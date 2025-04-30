package pipelinex

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/thoas/go-funk"
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

func (dga *DGAGraph) Nodes() map[string]Node {
	dga.hasCycle = false
	return funk.Map(dga.nodes, func(x Node) (string, Node) {
		return x.Id(), x
	}).(map[string]Node)
}

// AddVertex 添加顶点
func (dga *DGAGraph) AddVertex(node Node) {
	if dga.first == nil {
		dga.first = node
	}
	dga.nodes[node.Id()] = node
	dga.graph[node.Id()] = []string{}
	dga.hasCycle = false
}

// AddEdge 添加边
func (dga *DGAGraph) AddEdge(src, dest Node) {
	dga.graph[src.Id()] = append(dga.graph[src.Id()], dest.Id())
}

func (dga *DGAGraph) Traversal(fn TraversalFn) {
	if len(dga.sequence) == len(dga.nodes) {
		return
	}
	visited := make(map[string]bool)
	firstNodeID := dga.first.Id()
	// 初始化队列
	queue := []string{firstNodeID}
	// 初始化序列
	sequence := []string{firstNodeID}
	visited[firstNodeID] = true
	for len(queue) > 0 {
		// 获取队列里面的第一个节点
		vertexFocuse := queue[0]
		// 删除队列里面的第一个节点
		queue = queue[1:]
		// 遍历图结构并且执行回调
		wg := sync.WaitGroup{}
		wg.Add(len(dga.graph[vertexFocuse]))
		for _, neighbor := range dga.graph[vertexFocuse] {
			// 判断是否已经访问过
			if visited[neighbor] {
				dga.hasCycle = true
				wg.Done()
				continue
			}
			visited[neighbor] = true
			// 将邻接节点放入队列
			queue = append(queue, neighbor)
			// 将邻接节点放入序列
			sequence = append(sequence, neighbor)
			// 遍历回调
			go func(node Node) {
				defer wg.Done()
				fn(node)
			}(dga.nodes[neighbor])
		}
		wg.Wait()
	}
}

// CycelCheck 
// 检查是否存在循环
// 如果存在循环，则返回true
// 否则返回false
func (dga *DGAGraph) CycelCheck() bool {
	if len(dga.sequence) == len(dga.nodes) {
		return dga.hasCycle
	}
	visited := make(map[string]bool)
	firstNodeID := dga.first.Id()
	// 初始化队列
	queue := []string{firstNodeID}
	// 初始化序列
	sequence := []string{firstNodeID}
	visited[firstNodeID] = true
	for len(queue) > 0 {
		// 获取队列里面的第一个节点
		vertexFocuse := queue[0]
		// 删除队列里面的第一个节点
		queue = queue[1:]
		// 遍历图结构并且执行回调
		for _, neighbor := range dga.graph[vertexFocuse] {
			// 判断是否已经访问过
			if visited[neighbor] {
				dga.hasCycle = true
				return true
			}
			visited[neighbor] = true
			// 将邻接节点放入队列
			queue = append(queue, neighbor)
			// 将邻接节点放入序列
			sequence = append(sequence, neighbor)
		}
	}
	return false
}

type PipelineImpl struct {
	id        string
	graph     Graph
	status    string
	metadata  Metadata
	listening ListeningFn
	doneChan  <-chan struct{}
}

func NewPipeline(ctx context.Context) Pipeline {
	return &PipelineImpl{}
}

// ID 流水线的id
func (p *PipelineImpl) Id() string {
	return uuid.NewString()
}

// GetGraph 返回图结构
func (p *PipelineImpl) GetGraph() Graph {
	return p.graph
}

// SetGraph 设置图结构
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

// Listening 流水线执行事件监听设置
func (p *PipelineImpl) Listening(fn Listener) {

}

// Done流水线是否执行完成
func (p *PipelineImpl) Done() <-chan struct{} {
	return p.doneChan
}

func (p *PipelineImpl) Run(ctx context.Context) error {
	p.graph.Traversal(func(node Node) {
		fmt.Println(node.Id())
	})
	return nil
}

func (p *PipelineImpl) Notify() {

}

func (p *PipelineImpl) Cancel() {

}
