package test

import (
	"testing"

	"github.com/chenyingqiao/pipelinex"
)

func TestNewDGAEdge(t *testing.T) {
	node1 := pipelinex.NewDGANode("node1", "RUNNING")
	node2 := pipelinex.NewDGANode("node2", "UNKNOWN")

	edge := pipelinex.NewDGAEdge(node1, node2)

	if edge.Source().Id() != "node1" {
		t.Errorf("Expected source id 'node1', got '%s'", edge.Source().Id())
	}

	if edge.Target().Id() != "node2" {
		t.Errorf("Expected target id 'node2', got '%s'", edge.Target().Id())
	}

	if edge.Expression() != "" {
		t.Errorf("Expected empty expression for unconditional edge, got '%s'", edge.Expression())
	}

	if edge.ID() != "node1->node2" {
		t.Errorf("Expected edge id 'node1->node2', got '%s'", edge.ID())
	}
}

func TestNewConditionalEdge(t *testing.T) {
	node1 := pipelinex.NewDGANode("node1", "RUNNING")
	node2 := pipelinex.NewDGANode("node2", "UNKNOWN")
	expression := "{{ nodeStatus == 'SUCCESS' }}"

	edge := pipelinex.NewConditionalEdge(node1, node2, expression)

	if edge.Source().Id() != "node1" {
		t.Errorf("Expected source id 'node1', got '%s'", edge.Source().Id())
	}

	if edge.Target().Id() != "node2" {
		t.Errorf("Expected target id 'node2', got '%s'", edge.Target().Id())
	}

	if edge.Expression() != expression {
		t.Errorf("Expected expression '%s', got '%s'", expression, edge.Expression())
	}
}

func TestDGAEdge_Evaluate_Unconditional(t *testing.T) {
	node1 := pipelinex.NewDGANode("node1", "RUNNING")
	node2 := pipelinex.NewDGANode("node2", "UNKNOWN")

	edge := pipelinex.NewDGAEdge(node1, node2)
	evalCtx := pipelinex.NewEvaluationContext()

	result, err := edge.Evaluate(evalCtx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !result {
		t.Error("Unconditional edge should always evaluate to true")
	}
}

func TestDGAEdge_Evaluate_ConditionalTrue(t *testing.T) {
	node1 := pipelinex.NewDGANode("node1", "RUNNING")
	node2 := pipelinex.NewDGANode("node2", "SUCCESS")

	edge := pipelinex.NewConditionalEdge(node1, node2, "{{ nodeStatus == 'SUCCESS' }}")
	evalCtx := pipelinex.NewEvaluationContext().WithNode(node2)

	result, err := edge.Evaluate(evalCtx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !result {
		t.Error("Expected condition to evaluate to true")
	}
}

func TestDGAEdge_Evaluate_ConditionalFalse(t *testing.T) {
	node1 := pipelinex.NewDGANode("node1", "RUNNING")
	node2 := pipelinex.NewDGANode("node2", "FAILED")

	edge := pipelinex.NewConditionalEdge(node1, node2, "{{ nodeStatus == 'SUCCESS' }}")
	evalCtx := pipelinex.NewEvaluationContext().WithNode(node2)

	result, err := edge.Evaluate(evalCtx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result {
		t.Error("Expected condition to evaluate to false")
	}
}

func TestDGAEdge_Evaluate_WithParams(t *testing.T) {
	node1 := pipelinex.NewDGANode("node1", "RUNNING")
	node2 := pipelinex.NewDGANode("node2", "UNKNOWN")

	edge := pipelinex.NewConditionalEdge(node1, node2, "{{ branch == 'main' }}")
	evalCtx := pipelinex.NewEvaluationContext().WithParams(map[string]any{
		"branch": "main",
	})

	result, err := edge.Evaluate(evalCtx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !result {
		t.Error("Expected condition to evaluate to true with branch='main'")
	}
}

func TestDGAEdge_Evaluate_InvalidExpression(t *testing.T) {
	node1 := pipelinex.NewDGANode("node1", "RUNNING")
	node2 := pipelinex.NewDGANode("node2", "UNKNOWN")

	// 使用无效的模板语法
	edge := pipelinex.NewConditionalEdge(node1, node2, "{{ unclosed tag")
	evalCtx := pipelinex.NewEvaluationContext()

	_, err := edge.Evaluate(evalCtx)
	if err == nil {
		t.Error("Expected error for invalid expression syntax")
	}
}

func TestDGAEdge_ID(t *testing.T) {
	node1 := pipelinex.NewDGANode("start", "RUNNING")
	node2 := pipelinex.NewDGANode("end", "SUCCESS")

	edge := pipelinex.NewDGAEdge(node1, node2)

	expectedID := "start->end"
	if edge.ID() != expectedID {
		t.Errorf("Expected edge ID '%s', got '%s'", expectedID, edge.ID())
	}
}

func TestDGAEdge_ID_SpecialChars(t *testing.T) {
	node1 := pipelinex.NewDGANode("node-1_test", "RUNNING")
	node2 := pipelinex.NewDGANode("node.2", "SUCCESS")

	edge := pipelinex.NewDGAEdge(node1, node2)

	expectedID := "node-1_test->node.2"
	if edge.ID() != expectedID {
		t.Errorf("Expected edge ID '%s', got '%s'", expectedID, edge.ID())
	}
}
