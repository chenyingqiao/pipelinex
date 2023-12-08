package pipelinex

import (
	"context"

	"github.com/thoas/go-funk"
)

type DGAGraph struct {
	first    Node
	nodes    map[string]Node
	graph    map[string][]string
	sequence []string
	hasCycle bool
}

func (dga *DGAGraph) Nodes() []Node {
	dga.hasCycle = false
	return funk.Values(dga.nodes).([]Node)
}

// AddVertex 添加顶点
func (dga *DGAGraph) AddVertex(node Node) {
	dga.nodes[node.ID()] = node
	dga.graph[node.ID()] = []string{}
	dga.hasCycle = false
}

// AddEdge 添加边
func (dga *DGAGraph) AddEdge(src, dest string) {
	dga.graph[src] = append(dga.graph[src], dest)
}

// BFS 广度有限搜索返回搜索序列
func (dga *DGAGraph) BFS() []string {
	if len(dga.sequence) == len(dga.nodes) {
		return dga.sequence
	}
	visited := make(map[string]bool)
	firstNodeID := dga.first.ID()
	queue := []string{firstNodeID}
	sequence := []string{firstNodeID}
	visited[firstNodeID] = true
	for len(sequence) > 0 {
		vertexFocuse := queue[0]
		queue = queue[1:]
		for _, neighbor := range dga.graph[vertexFocuse] {
			if visited[neighbor] {
				dga.hasCycle = true
				continue
			}
			queue = append(queue, neighbor)
			sequence = append(sequence, neighbor)
		}
	}
	dga.sequence = sequence
	return dga.sequence
}

// CycelCheck 循环检查序列
func (dga *DGAGraph) CycelCheck() bool {
	dga.BFS()
	return dga.hasCycle
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
func (p *PipelineImpl) ID() string {
	return ""
}

// GetGraph 返回图结构
func (p *PipelineImpl) GetGraph() Graph {
	return p.graph
}

// SetGraph 设置图结构
func (p *PipelineImpl) SetGraph(graph Graph) {}

// Status 返回流水线的整体状态
func (p *PipelineImpl) Status() string {
	return ""
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
	return nil
}

func (p *PipelineImpl) Notify() {

}

func (p *PipelineImpl) Cancel() {

}
