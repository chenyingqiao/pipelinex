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
	if err := dgaGraph.Traversal(context.Background(), func(ctx context.Context, node pipelinex.Node) error {
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

