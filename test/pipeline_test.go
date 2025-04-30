package test

import (
	"context"
	"testing"

	"github.com/chenyingqiao/pipelinex"
)

func TestDGA_BFS(t *testing.T) {
	dgaGraph := pipelinex.NewDGAGraph()
	node1 := pipelinex.NewDGANode("a", "RUNNING")
	node2 := pipelinex.NewDGANode("b", "UNKNOWN")
	node3 := pipelinex.NewDGANode("c", "UNKNOWN	")
	node4 := pipelinex.NewDGANode("e", "UNKNOWN")
	dgaGraph.AddVertex(node1)
	dgaGraph.AddVertex(node2)
	dgaGraph.AddVertex(node3)
	dgaGraph.AddVertex(node4)
	dgaGraph.AddEdge(node1, node2)
	dgaGraph.AddEdge(node1, node4)
	dgaGraph.AddEdge(node2, node3)
	dgaGraph.AddEdge(node4, node3)
	// Collect visited nodes to verify traversal order
	visited := []string{}
	if err := dgaGraph.Traversal(context.Background(), func(node pipelinex.Node) error {
		t.Log("Visiting node:", node.Id())
		visited = append(visited, node.Id())
		return nil
	}); err != nil {
		t.Error(err)
	}

	// Verify traversal includes all nodes
	expectedNodes := []string{"a", "b", "e", "c"}
	if len(visited) != len(expectedNodes) {
		t.Errorf("Expected to visit %d nodes, but visited %d", len(expectedNodes), len(visited))
	}

	// Verify first node is always the starting point
	if visited[0] != "a" {
		t.Errorf("First visited node should be 'a', got %s", visited[0])
	}
}

func TestPipelineImpl_Run(t *testing.T) {
	// 创建一个简单的DAG图
	graph := NewDGAGraph()
	node1 := &NodeImpl{id: "node1"}
	node2 := &NodeImpl{id: "node2"}
	node3 := &NodeImpl{id: "node3"}
	assert.NoError(t, graph.AddVertex(node1))
	assert.NoError(t, graph.AddVertex(node2))
	assert.NoError(t, graph.AddVertex(node3))
	graph.AddEdge(node1, node2)
	graph.AddEdge(node2, node3)

	// 创建PipelineImpl实例
	pipeline := &PipelineImpl{graph: graph}

	// 执行Run方法
	err := pipeline.Run(context.Background())
	assert.NoError(t, err)

	// 验证输出 (这部分取决于Run方法的具体实现，这里只是一个例子)
	//  在实际测试中，需要根据Run方法的输出结果进行相应的断言
}
