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
	for _, e := range [][2]pipelinex.Node{
		{node1, node2},
		{node1, node4},
		{node2, node3},
		{node4, node3},
	} {
		edge := pipelinex.NewDGAEdge(e[0], e[1])
		if err := dgaGraph.AddEdge(edge); err != nil {
			t.Fatalf("AddEdge(%s→%s): %v", e[0].Id(), e[1].Id(), err)
		}
	}
	// Collect visited nodes to verify traversal order
	visited := []string{}
	evalCtx := pipelinex.NewEvaluationContext()
	if err := dgaGraph.Traversal(context.Background(), evalCtx, func(ctx context.Context, node pipelinex.Node) error {
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

func TestDGA_MultipleStartNodes(t *testing.T) {
	dgaGraph := pipelinex.NewDGAGraph()
	nodeA := pipelinex.NewDGANode("a", "UNKNOWN")
	nodeB := pipelinex.NewDGANode("b", "UNKNOWN")
	nodeC := pipelinex.NewDGANode("c", "UNKNOWN")

	dgaGraph.AddVertex(nodeA)
	dgaGraph.AddVertex(nodeB)
	dgaGraph.AddVertex(nodeC)

	// 添加边：A -> C, B -> C
	// 这样 A 和 B 都是入度为0的起始节点
	for _, e := range [][2]pipelinex.Node{
		{nodeA, nodeC},
		{nodeB, nodeC},
	} {
		edge := pipelinex.NewDGAEdge(e[0], e[1])
		if err := dgaGraph.AddEdge(edge); err != nil {
			t.Fatalf("AddEdge(%s→%s): %v", e[0].Id(), e[1].Id(), err)
		}
	}

	// 收集访问的节点
	visited := make(map[string]bool)
	evalCtx := pipelinex.NewEvaluationContext()
	if err := dgaGraph.Traversal(context.Background(), evalCtx, func(ctx context.Context, node pipelinex.Node) error {
		t.Log("Visiting node:", node.Id())
		visited[node.Id()] = true
		return nil
	}); err != nil {
		t.Error(err)
	}

	// 验证所有节点都被访问
	expectedNodes := []string{"a", "b", "c"}
	for _, nodeID := range expectedNodes {
		if !visited[nodeID] {
			t.Errorf("Node %s was not visited", nodeID)
		}
	}

	if len(visited) != len(expectedNodes) {
		t.Errorf("Expected to visit %d nodes, but visited %d", len(expectedNodes), len(visited))
	}
}

func TestPipeline_Run(t *testing.T) {
	dgaGraph := pipelinex.NewDGAGraph()
	node1 := pipelinex.NewDGANode("a", "RUNNING")
	node2 := pipelinex.NewDGANode("b", "UNKNOWN")
	node3 := pipelinex.NewDGANode("c", "UNKNOWN	")
	node4 := pipelinex.NewDGANode("e", "UNKNOWN")
	dgaGraph.AddVertex(node1)
	dgaGraph.AddVertex(node2)
	dgaGraph.AddVertex(node3)
	dgaGraph.AddVertex(node4)
	for _, e := range [][2]pipelinex.Node{
		{node1, node2},
		{node1, node4},
		{node2, node3},
		{node4, node3},
	} {
		edge := pipelinex.NewDGAEdge(e[0], e[1])
		if err := dgaGraph.AddEdge(edge); err != nil {
			t.Fatalf("AddEdge(%s→%s): %v", e[0].Id(), e[1].Id(), err)
		}
	}
	ctx := context.Background()
	pipeline := pipelinex.NewPipeline(ctx)
	pipeline.SetGraph(dgaGraph)
	// 收集遍历过程中访问的节点
	err := pipeline.Run(ctx)
	if err != nil {
		t.Error(err)
	}
}
