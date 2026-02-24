package test

import (
	"context"
	"testing"

	"github.com/chenyingqiao/pipelinex"
)

func TestDGAGraph_AddEdge_Success(t *testing.T) {
	graph := pipelinex.NewDGAGraph()
	node1 := pipelinex.NewDGANode("a", "RUNNING")
	node2 := pipelinex.NewDGANode("b", "UNKNOWN")

	graph.AddVertex(node1)
	graph.AddVertex(node2)

	edge := pipelinex.NewDGAEdge(node1, node2)
	err := graph.AddEdge(edge)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestDGAGraph_AddEdge_SourceNotFound(t *testing.T) {
	graph := pipelinex.NewDGAGraph()
	node1 := pipelinex.NewDGANode("a", "RUNNING")
	node2 := pipelinex.NewDGANode("b", "UNKNOWN")

	// 只添加目标节点，不添加源节点
	graph.AddVertex(node2)

	edge := pipelinex.NewDGAEdge(node1, node2)
	err := graph.AddEdge(edge)

	if err == nil {
		t.Error("Expected error for missing source node")
	}
}

func TestDGAGraph_AddEdge_TargetNotFound(t *testing.T) {
	graph := pipelinex.NewDGAGraph()
	node1 := pipelinex.NewDGANode("a", "RUNNING")
	node2 := pipelinex.NewDGANode("b", "UNKNOWN")

	// 只添加源节点，不添加目标节点
	graph.AddVertex(node1)

	edge := pipelinex.NewDGAEdge(node1, node2)
	err := graph.AddEdge(edge)

	if err == nil {
		t.Error("Expected error for missing target node")
	}
}

func TestDGAGraph_AddEdge_Cycle(t *testing.T) {
	graph := pipelinex.NewDGAGraph()
	node1 := pipelinex.NewDGANode("a", "RUNNING")
	node2 := pipelinex.NewDGANode("b", "UNKNOWN")
	node3 := pipelinex.NewDGANode("c", "UNKNOWN")

	graph.AddVertex(node1)
	graph.AddVertex(node2)
	graph.AddVertex(node3)

	// 添加边 a -> b -> c
	edge1 := pipelinex.NewDGAEdge(node1, node2)
	edge2 := pipelinex.NewDGAEdge(node2, node3)
	graph.AddEdge(edge1)
	graph.AddEdge(edge2)

	// 添加边 c -> a，这会形成循环
	edge3 := pipelinex.NewDGAEdge(node3, node1)
	err := graph.AddEdge(edge3)

	if err == nil {
		t.Error("Expected error for cycle detection")
	}

	if err != pipelinex.ErrHasCycle {
		t.Errorf("Expected ErrHasCycle, got: %v", err)
	}
}

func TestDGAGraph_AddEdge_MultipleEdges(t *testing.T) {
	graph := pipelinex.NewDGAGraph()
	node1 := pipelinex.NewDGANode("a", "RUNNING")
	node2 := pipelinex.NewDGANode("b", "UNKNOWN")
	node3 := pipelinex.NewDGANode("c", "UNKNOWN")

	graph.AddVertex(node1)
	graph.AddVertex(node2)
	graph.AddVertex(node3)

	// a -> b
	edge1 := pipelinex.NewDGAEdge(node1, node2)
	if err := graph.AddEdge(edge1); err != nil {
		t.Errorf("Failed to add edge a->b: %v", err)
	}

	// a -> c
	edge2 := pipelinex.NewDGAEdge(node1, node3)
	if err := graph.AddEdge(edge2); err != nil {
		t.Errorf("Failed to add edge a->c: %v", err)
	}

	// 验证遍历
	evalCtx := pipelinex.NewEvaluationContext()
	visited := []string{}

	if err := graph.Traversal(context.Background(), evalCtx, func(ctx context.Context, node pipelinex.Node) error {
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
	graph := pipelinex.NewDGAGraph()
	node1 := pipelinex.NewDGANode("a", "RUNNING")

	graph.AddVertex(node1)

	// 自环 a -> a
	edge := pipelinex.NewDGAEdge(node1, node1)
	err := graph.AddEdge(edge)

	if err == nil {
		t.Error("Expected error for self-loop")
	}

	if err != pipelinex.ErrHasCycle {
		t.Errorf("Expected ErrHasCycle for self-loop, got: %v", err)
	}
}

func TestDGAGraph_Traversal_WithConditionalEdge(t *testing.T) {
	graph := pipelinex.NewDGAGraph()
	node1 := pipelinex.NewDGANode("a", "RUNNING")
	node2 := pipelinex.NewDGANode("b", "FAILED")
	node3 := pipelinex.NewDGANode("c", "UNKNOWN")

	graph.AddVertex(node1)
	graph.AddVertex(node2)
	graph.AddVertex(node3)

	// a -> b (无条件)
	edge1 := pipelinex.NewDGAEdge(node1, node2)
	graph.AddEdge(edge1)

	// b -> c (条件：nodeStatus == 'SUCCESS')
	edge2 := pipelinex.NewConditionalEdge(node2, node3, "{{ nodeStatus == 'SUCCESS' }}")
	graph.AddEdge(edge2)

	// node2 状态是 FAILED，所以条件边不应该被遍历
	evalCtx := pipelinex.NewEvaluationContext().WithNode(node2)
	visited := []string{}

	if err := graph.Traversal(context.Background(), evalCtx, func(ctx context.Context, node pipelinex.Node) error {
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
	graph := pipelinex.NewDGAGraph()
	node1 := pipelinex.NewDGANode("a", "RUNNING")
	node2 := pipelinex.NewDGANode("b", "UNKNOWN")
	node3 := pipelinex.NewDGANode("c", "UNKNOWN")
	node4 := pipelinex.NewDGANode("d", "UNKNOWN")

	graph.AddVertex(node1)
	graph.AddVertex(node2)
	graph.AddVertex(node3)
	graph.AddVertex(node4)

	// a -> b (无条件)
	graph.AddEdge(pipelinex.NewDGAEdge(node1, node2))

	// b -> c (条件：env == 'prod')
	graph.AddEdge(pipelinex.NewConditionalEdge(node2, node3, "{{ env == 'prod' }}"))

	// b -> d (条件：env == 'dev')
	graph.AddEdge(pipelinex.NewConditionalEdge(node2, node4, "{{ env == 'dev' }}"))

	// 测试 env=prod 场景
	evalCtx := pipelinex.NewEvaluationContext().WithParams(map[string]any{
		"env": "prod",
	})
	visited := []string{}

	if err := graph.Traversal(context.Background(), evalCtx, func(ctx context.Context, node pipelinex.Node) error {
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
	graph := pipelinex.NewDGAGraph()
	node1 := pipelinex.NewDGANode("a", "RUNNING")
	node2 := pipelinex.NewDGANode("b", "UNKNOWN")

	graph.AddVertex(node1)
	graph.AddVertex(node2)

	// 使用无效的表达式语法
	edge := pipelinex.NewConditionalEdge(node1, node2, "{{ unclosed")
	graph.AddEdge(edge)

	evalCtx := pipelinex.NewEvaluationContext()
	err := graph.Traversal(context.Background(), evalCtx, func(ctx context.Context, node pipelinex.Node) error {
		return nil
	})

	if err == nil {
		t.Error("Expected error for invalid expression in conditional edge")
	}
}

func TestDGAGraph_Traversal_EmptyGraph(t *testing.T) {
	graph := pipelinex.NewDGAGraph()

	evalCtx := pipelinex.NewEvaluationContext()
	err := graph.Traversal(context.Background(), evalCtx, func(ctx context.Context, node pipelinex.Node) error {
		t.Error("Should not visit any node in empty graph")
		return nil
	})

	if err != nil {
		t.Errorf("Unexpected error for empty graph: %v", err)
	}
}

func TestDGAGraph_Traversal_SingleNode(t *testing.T) {
	graph := pipelinex.NewDGAGraph()
	node1 := pipelinex.NewDGANode("solo", "RUNNING")

	graph.AddVertex(node1)

	evalCtx := pipelinex.NewEvaluationContext()
	visited := []string{}

	if err := graph.Traversal(context.Background(), evalCtx, func(ctx context.Context, node pipelinex.Node) error {
		visited = append(visited, node.Id())
		return nil
	}); err != nil {
		t.Errorf("Traversal failed: %v", err)
	}

	if len(visited) != 1 || visited[0] != "solo" {
		t.Errorf("Expected [solo], got %v", visited)
	}
}
