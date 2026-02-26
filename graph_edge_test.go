package pipelinex

import (
	"context"
	"testing"
)

func TestDGAGraph_AddEdge_Success(t *testing.T) {
	graph := NewDGAGraph()
	node1 := NewDGANode("a", "RUNNING")
	node2 := NewDGANode("b", "UNKNOWN")

	graph.AddVertex(node1)
	graph.AddVertex(node2)

	edge := NewDGAEdge(node1, node2)
	err := graph.AddEdge(edge)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestDGAGraph_AddEdge_SourceNotFound(t *testing.T) {
	graph := NewDGAGraph()
	node1 := NewDGANode("a", "RUNNING")
	node2 := NewDGANode("b", "UNKNOWN")

	// 只添加目标节点，不添加源节点
	graph.AddVertex(node2)

	edge := NewDGAEdge(node1, node2)
	err := graph.AddEdge(edge)

	if err == nil {
		t.Error("Expected error for missing source node")
	}
}

func TestDGAGraph_AddEdge_TargetNotFound(t *testing.T) {
	graph := NewDGAGraph()
	node1 := NewDGANode("a", "RUNNING")
	node2 := NewDGANode("b", "UNKNOWN")

	// 只添加源节点，不添加目标节点
	graph.AddVertex(node1)

	edge := NewDGAEdge(node1, node2)
	err := graph.AddEdge(edge)

	if err == nil {
		t.Error("Expected error for missing target node")
	}
}

func TestDGAGraph_AddEdge_Cycle(t *testing.T) {
	graph := NewDGAGraph()
	node1 := NewDGANode("a", "RUNNING")
	node2 := NewDGANode("b", "UNKNOWN")
	node3 := NewDGANode("c", "UNKNOWN")

	graph.AddVertex(node1)
	graph.AddVertex(node2)
	graph.AddVertex(node3)

	// 添加边 a -> b -> c
	edge1 := NewDGAEdge(node1, node2)
	edge2 := NewDGAEdge(node2, node3)
	graph.AddEdge(edge1)
	graph.AddEdge(edge2)

	// 添加边 c -> a，这会形成循环
	edge3 := NewDGAEdge(node3, node1)
	err := graph.AddEdge(edge3)

	if err == nil {
		t.Error("Expected error for cycle detection")
	}

	if err != ErrHasCycle {
		t.Errorf("Expected ErrHasCycle, got: %v", err)
	}
}

func TestDGAGraph_AddEdge_MultipleEdges(t *testing.T) {
	graph := NewDGAGraph()
	node1 := NewDGANode("a", "RUNNING")
	node2 := NewDGANode("b", "UNKNOWN")
	node3 := NewDGANode("c", "UNKNOWN")

	graph.AddVertex(node1)
	graph.AddVertex(node2)
	graph.AddVertex(node3)

	// a -> b
	edge1 := NewDGAEdge(node1, node2)
	if err := graph.AddEdge(edge1); err != nil {
		t.Errorf("Failed to add edge a->b: %v", err)
	}

	// a -> c
	edge2 := NewDGAEdge(node1, node3)
	if err := graph.AddEdge(edge2); err != nil {
		t.Errorf("Failed to add edge a->c: %v", err)
	}

	// 验证遍历
	evalCtx := NewEvaluationContext()
	visited := []string{}

	if err := graph.Traversal(context.Background(), evalCtx, func(ctx context.Context, node Node) error {
		visited = append(visited, node.Id())
		return nil
	}); err != nil {
		t.Errorf("Traversal failed: %v", err)
	}

	if len(visited) != 3 {
		t.Errorf("Expected 3 visited nodes, got %d: %v", len(visited), visited)
	}
}

func TestDGAGraph_AddEdge_SelfLoop(t *testing.T) {
	graph := NewDGAGraph()
	node1 := NewDGANode("a", "RUNNING")

	graph.AddVertex(node1)

	// 自环 a -> a
	edge := NewDGAEdge(node1, node1)
	err := graph.AddEdge(edge)

	if err == nil {
		t.Error("Expected error for self-loop")
	}

	if err != ErrHasCycle {
		t.Errorf("Expected ErrHasCycle for self-loop, got: %v", err)
	}
}

func TestDGAGraph_Traversal_WithConditionalEdge(t *testing.T) {
	graph := NewDGAGraph()
	node1 := NewDGANode("a", "RUNNING")
	node2 := NewDGANode("b", "FAILED")
	node3 := NewDGANode("c", "UNKNOWN")

	graph.AddVertex(node1)
	graph.AddVertex(node2)
	graph.AddVertex(node3)

	// a -> b (无条件)
	edge1 := NewDGAEdge(node1, node2)
	graph.AddEdge(edge1)

	// b -> c (条件：nodeStatus == 'SUCCESS')
	edge2 := NewConditionalEdge(node2, node3, "{{ nodeStatus == 'SUCCESS' }}")
	graph.AddEdge(edge2)

	// node2 状态是 FAILED，所以条件边不应该被遍历
	evalCtx := NewEvaluationContext().WithNode(node2)
	visited := []string{}

	if err := graph.Traversal(context.Background(), evalCtx, func(ctx context.Context, node Node) error {
		visited = append(visited, node.Id())
		return nil
	}); err != nil {
		t.Errorf("Traversal failed: %v", err)
	}

	// 应该只访问 a 和 b
	if len(visited) != 2 {
		t.Errorf("Expected 2 visited nodes (a, b), got %d: %v", len(visited), visited)
	}
}

func TestDGAGraph_Traversal_MultipleConditionalEdges(t *testing.T) {
	graph := NewDGAGraph()
	node1 := NewDGANode("a", "RUNNING")
	node2 := NewDGANode("b", "UNKNOWN")
	node3 := NewDGANode("c", "UNKNOWN")
	node4 := NewDGANode("d", "UNKNOWN")

	graph.AddVertex(node1)
	graph.AddVertex(node2)
	graph.AddVertex(node3)
	graph.AddVertex(node4)

	// a -> b (无条件)
	graph.AddEdge(NewDGAEdge(node1, node2))

	// b -> c (条件：env == 'prod')
	graph.AddEdge(NewConditionalEdge(node2, node3, "{{ env == 'prod' }}"))

	// b -> d (条件：env == 'dev')
	graph.AddEdge(NewConditionalEdge(node2, node4, "{{ env == 'dev' }}"))

	// 测试 env=prod 场景
	evalCtx := NewEvaluationContext().WithParams(map[string]any{
		"env": "prod",
	})
	visited := []string{}

	if err := graph.Traversal(context.Background(), evalCtx, func(ctx context.Context, node Node) error {
		visited = append(visited, node.Id())
		return nil
	}); err != nil {
		t.Errorf("Traversal failed: %v", err)
	}

	// 应该访问 a, b, c
	if len(visited) != 3 {
		t.Errorf("Expected 3 visited nodes, got %d: %v", len(visited), visited)
	}
}

func TestDGAGraph_Traversal_ConditionalEdgeError(t *testing.T) {
	graph := NewDGAGraph()
	node1 := NewDGANode("a", "RUNNING")
	node2 := NewDGANode("b", "UNKNOWN")

	graph.AddVertex(node1)
	graph.AddVertex(node2)

	// 使用无效的表达式语法
	edge := NewConditionalEdge(node1, node2, "{{ unclosed")
	graph.AddEdge(edge)

	evalCtx := NewEvaluationContext()
	err := graph.Traversal(context.Background(), evalCtx, func(ctx context.Context, node Node) error {
		return nil
	})

	if err == nil {
		t.Error("Expected error for invalid expression in conditional edge")
	}
}

func TestDGAGraph_Traversal_EmptyGraph(t *testing.T) {
	graph := NewDGAGraph()

	evalCtx := NewEvaluationContext()
	err := graph.Traversal(context.Background(), evalCtx, func(ctx context.Context, node Node) error {
		t.Error("Should not visit any node in empty graph")
		return nil
	})

	if err != nil {
		t.Errorf("Unexpected error for empty graph: %v", err)
	}
}

func TestDGAGraph_Traversal_SingleNode(t *testing.T) {
	graph := NewDGAGraph()
	node1 := NewDGANode("solo", "RUNNING")

	graph.AddVertex(node1)

	evalCtx := NewEvaluationContext()
	visited := []string{}

	if err := graph.Traversal(context.Background(), evalCtx, func(ctx context.Context, node Node) error {
		visited = append(visited, node.Id())
		return nil
	}); err != nil {
		t.Errorf("Traversal failed: %v", err)
	}

	if len(visited) != 1 || visited[0] != "solo" {
		t.Errorf("Expected [solo], got %v", visited)
	}
}
