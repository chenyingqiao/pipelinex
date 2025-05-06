package pipelinex

import (
	"context"
	"fmt"

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
	return funk.Map(dga.nodes, func(x Node) (string, Node) {
		return x.Id(), x
	}).(map[string]Node)
}

// AddVertex adds a vertex (node) to the graph.
// It checks for cycles; if a cycle exists, it returns ErrHasCycle.
// Otherwise, it returns nil.
func (dga *DGAGraph) AddVertex(node Node) {
	if dga.first == nil {
		dga.first = node
	}
	dga.nodes[node.Id()] = node
	dga.graph[node.Id()] = []string{}
}

// AddEdge adds an edge to the graph.
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

// Traversal performs a breadth-first traversal of the DAG.
// It executes the provided TraversalFn function for each node in the graph.
func (dga *DGAGraph) Traversal(ctx context.Context, fn TraversalFn) error {
	visited := make(map[string]bool)                        // Keep track of visited nodes
	firstNodeID := dga.first.Id()                           // Get the ID of the first node
	queue := []string{firstNodeID}                          // Initialize the queue with the first node
	visited[firstNodeID] = true                             // Mark the first node as visited
	if err := fn(ctx, dga.nodes[firstNodeID]); err != nil { // Execute the function for the first node
		return err
	}
	for len(queue) > 0 { // Continue until the queue is empty
		vertexFocuse := queue[0]                           // Get the next node from the queue
		queue = queue[1:]                                  // Remove the node from the queue
		g, ctx := errgroup.WithContext(ctx)                // Create a new context for concurrent goroutines
		for _, neighbor := range dga.graph[vertexFocuse] { // Iterate over the neighbors of the current node
			if visited[neighbor] { // Check if the neighbor has already been visited
				continue
			}
			visited[neighbor] = true        // Mark the neighbor as visited
			queue = append(queue, neighbor) // Add the neighbor to the queue
			g.Go(func() error {             // Execute the function for the neighbor concurrently
				return fn(ctx, dga.nodes[neighbor])
			})
		}
		if err := g.Wait(); err != nil { // Wait for all goroutines to finish
			return err
		}
	}
	return nil
}

// cycleCheck checks if a cycle exists in the directed acyclic graph (DAG).
// It returns true if a cycle is found, and false otherwise.
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
	id        string
	graph     Graph
	status    string
	metadata  Metadata
	listening ListeningFn
	doneChan  <-chan struct{}
}

func NewPipeline(ctx context.Context) Pipeline {
	return &PipelineImpl{
		id: uuid.NewString(),
	}
}

// Id returns the ID of the pipeline.
func (p *PipelineImpl) Id() string {
	return p.id
}

// GetGraph returns the graph structure of the pipeline.
func (p *PipelineImpl) GetGraph() Graph {
	return p.graph
}

// SetGraph sets the graph structure of the pipeline.
func (p *PipelineImpl) SetGraph(graph Graph) {
	p.graph = graph
}

// Status returns the overall status of the pipeline.
func (p *PipelineImpl) Status() string {
	return p.status
}

// Metadata returns the source data for pipeline execution.
func (p *PipelineImpl) Metadata() Metadata {
	return Metadata{}
}

// Listening sets the pipeline execution event listener.
func (p *PipelineImpl) Listening(fn Listener) {

}

// Done returns a channel that signals when the pipeline is finished.
func (p *PipelineImpl) Done() <-chan struct{} {
	return p.doneChan
}

// Run executes the pipeline.
func (p *PipelineImpl) Run(ctx context.Context) error {
	done := make(chan struct{})
	p.doneChan = done
	err := p.graph.Traversal(ctx, func(ctx context.Context, node Node) error {
		fmt.Println(node.Id())
		return nil
	})
	close(done)
	return err
}

func (p *PipelineImpl) Notify() {

}

func (p *PipelineImpl) Cancel() {

}
