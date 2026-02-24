package test

import (
	"context"
	"testing"

	"github.com/chenyingqiao/pipelinex"
)

func TestConditionalEdge_Basic(t *testing.T) {
	dgaGraph := pipelinex.NewDGAGraph()
	node1 := pipelinex.NewDGANode("a", "RUNNING")
	node2 := pipelinex.NewDGANode("b", "UNKNOWN")
	node3 := pipelinex.NewDGANode("c", "UNKNOWN")

	dgaGraph.AddVertex(node1)
	dgaGraph.AddVertex(node2)
	dgaGraph.AddVertex(node3)

	// 无条件边 a -> b
	edge1 := pipelinex.NewDGAEdge(node1, node2)
	if err := dgaGraph.AddEdge(edge1); err != nil {
		t.Fatalf("AddEdge(a->b): %v", err)
	}

	// 条件边 b -> c, 条件是 nodeStatus == 'SUCCESS'
	edge2 := pipelinex.NewConditionalEdge(node2, node3, "{{ nodeStatus == 'SUCCESS' }}")
	if err := dgaGraph.AddEdge(edge2); err != nil {
		t.Fatalf("AddEdge(b->c): %v", err)
	}

	// 测试场景1: node2状态不是SUCCESS，条件边不应该被遍历
	evalCtx := pipelinex.NewEvaluationContext().WithNode(node2)
	visited := []string{}

	if err := dgaGraph.Traversal(context.Background(), evalCtx, func(ctx context.Context, node pipelinex.Node) error {
		t.Log("Visiting node:", node.Id())
		visited = append(visited, node.Id())
		return nil
	}); err != nil {
		t.Error(err)
	}

	// 应该只访问 a 和 b，因为 b->c 的条件不满足
	if len(visited) != 2 {
		t.Errorf("Expected to visit 2 nodes, but visited %d: %v", len(visited), visited)
	}
}

func TestConditionalEdge_WithTrueCondition(t *testing.T) {
	dgaGraph := pipelinex.NewDGAGraph()
	node1 := pipelinex.NewDGANode("a", "RUNNING")
	node2 := pipelinex.NewDGANode("b", "SUCCESS") // 状态为 SUCCESS
	node3 := pipelinex.NewDGANode("c", "UNKNOWN")

	dgaGraph.AddVertex(node1)
	dgaGraph.AddVertex(node2)
	dgaGraph.AddVertex(node3)

	// 无条件边 a -> b
	edge1 := pipelinex.NewDGAEdge(node1, node2)
	if err := dgaGraph.AddEdge(edge1); err != nil {
		t.Fatalf("AddEdge(a->b): %v", err)
	}

	// 条件边 b -> c, 条件是 nodeStatus == 'SUCCESS'
	edge2 := pipelinex.NewConditionalEdge(node2, node3, "{{ nodeStatus == 'SUCCESS' }}")
	if err := dgaGraph.AddEdge(edge2); err != nil {
		t.Fatalf("AddEdge(b->c): %v", err)
	}

	// 测试: node2状态是SUCCESS，条件边应该被遍历
	evalCtx := pipelinex.NewEvaluationContext().WithNode(node2)
	visited := []string{}

	if err := dgaGraph.Traversal(context.Background(), evalCtx, func(ctx context.Context, node pipelinex.Node) error {
		t.Log("Visiting node:", node.Id())
		visited = append(visited, node.Id())
		return nil
	}); err != nil {
		t.Error(err)
	}

	// 应该访问所有三个节点
	if len(visited) != 3 {
		t.Errorf("Expected to visit 3 nodes, but visited %d: %v", len(visited), visited)
	}
}

func TestConditionalEdge_WithParams(t *testing.T) {
	dgaGraph := pipelinex.NewDGAGraph()
	node1 := pipelinex.NewDGANode("a", "RUNNING")
	node2 := pipelinex.NewDGANode("b", "UNKNOWN")
	node3 := pipelinex.NewDGANode("c", "UNKNOWN")
	node4 := pipelinex.NewDGANode("d", "UNKNOWN")

	dgaGraph.AddVertex(node1)
	dgaGraph.AddVertex(node2)
	dgaGraph.AddVertex(node3)
	dgaGraph.AddVertex(node4)

	// 无条件边 a -> b
	edge1 := pipelinex.NewDGAEdge(node1, node2)
	if err := dgaGraph.AddEdge(edge1); err != nil {
		t.Fatalf("AddEdge(a->b): %v", err)
	}

	// 条件边 b -> c, 条件是 branch == 'main'
	edge2 := pipelinex.NewConditionalEdge(node2, node3, "{{ branch == 'main' }}")
	if err := dgaGraph.AddEdge(edge2); err != nil {
		t.Fatalf("AddEdge(b->c): %v", err)
	}

	// 条件边 b -> d, 条件是 branch == 'develop'
	edge3 := pipelinex.NewConditionalEdge(node2, node4, "{{ branch == 'develop' }}")
	if err := dgaGraph.AddEdge(edge3); err != nil {
		t.Fatalf("AddEdge(b->d): %v", err)
	}

	// 测试场景1: branch = main，应该走 b -> c
	evalCtx := pipelinex.NewEvaluationContext().WithParams(map[string]any{
		"branch": "main",
	})
	visited := []string{}

	if err := dgaGraph.Traversal(context.Background(), evalCtx, func(ctx context.Context, node pipelinex.Node) error {
		t.Log("Visiting node:", node.Id())
		visited = append(visited, node.Id())
		return nil
	}); err != nil {
		t.Error(err)
	}

	// 应该访问 a, b, c
	if len(visited) != 3 {
		t.Errorf("Expected to visit 3 nodes, but visited %d: %v", len(visited), visited)
	}

	// 测试场景2: branch = develop，应该走 b -> d
	evalCtx2 := pipelinex.NewEvaluationContext().WithParams(map[string]any{
		"branch": "develop",
	})
	visited2 := []string{}

	if err := dgaGraph.Traversal(context.Background(), evalCtx2, func(ctx context.Context, node pipelinex.Node) error {
		t.Log("Visiting node:", node.Id())
		visited2 = append(visited2, node.Id())
		return nil
	}); err != nil {
		t.Error(err)
	}

	// 应该访问 a, b, d
	if len(visited2) != 3 {
		t.Errorf("Expected to visit 3 nodes, but visited %d: %v", len(visited2), visited2)
	}
}

func TestEdge_Evaluate(t *testing.T) {
	node1 := pipelinex.NewDGANode("a", "RUNNING")
	node2 := pipelinex.NewDGANode("b", "SUCCESS")

	// 测试无条件边
	uncondEdge := pipelinex.NewDGAEdge(node1, node2)
	if uncondEdge.Expression() != "" {
		t.Errorf("Expected empty expression for unconditional edge, got %s", uncondEdge.Expression())
	}

	evalCtx := pipelinex.NewEvaluationContext()
	result, err := uncondEdge.Evaluate(evalCtx)
	if err != nil {
		t.Errorf("Unexpected error evaluating unconditional edge: %v", err)
	}
	if !result {
		t.Error("Unconditional edge should always evaluate to true")
	}

	// 测试条件边
	condEdge := pipelinex.NewConditionalEdge(node1, node2, "{{ nodeStatus == 'SUCCESS' }}")
	if condEdge.Expression() == "" {
		t.Error("Expected non-empty expression for conditional edge")
	}

	// 条件不满足的情况
	evalCtx2 := pipelinex.NewEvaluationContext().WithNode(node1) // node1 状态是 RUNNING
	result2, err := condEdge.Evaluate(evalCtx2)
	if err != nil {
		t.Errorf("Unexpected error evaluating conditional edge: %v", err)
	}
	if result2 {
		t.Error("Conditional edge should evaluate to false when condition is not met")
	}

	// 条件满足的情况
	evalCtx3 := pipelinex.NewEvaluationContext().WithNode(node2) // node2 状态是 SUCCESS
	result3, err := condEdge.Evaluate(evalCtx3)
	if err != nil {
		t.Errorf("Unexpected error evaluating conditional edge: %v", err)
	}
	if !result3 {
		t.Error("Conditional edge should evaluate to true when condition is met")
	}
}

func TestEdge_ID(t *testing.T) {
	node1 := pipelinex.NewDGANode("a", "RUNNING")
	node2 := pipelinex.NewDGANode("b", "UNKNOWN")

	edge := pipelinex.NewDGAEdge(node1, node2)
	expectedID := "a->b"
	if edge.ID() != expectedID {
		t.Errorf("Expected edge ID %s, got %s", expectedID, edge.ID())
	}
}
