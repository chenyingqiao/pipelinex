package test

import (
	"testing"

	"github.com/chenyingqiao/pipelinex"
)

func TestNewEvaluationContext(t *testing.T) {
	ctx := pipelinex.NewEvaluationContext()
	if ctx == nil {
		t.Error("Expected non-nil evaluation context")
	}
}

func TestDGAEvaluationContext_Get(t *testing.T) {
	ctx := pipelinex.NewEvaluationContext().WithParams(map[string]any{
		"key": "value",
	})

	val, ok := ctx.Get("key")
	if !ok {
		t.Error("Expected to find key 'key'")
	}

	if val != "value" {
		t.Errorf("Expected value 'value', got '%v'", val)
	}
}

func TestDGAEvaluationContext_Get_NotFound(t *testing.T) {
	ctx := pipelinex.NewEvaluationContext()

	_, ok := ctx.Get("nonexistent")
	if ok {
		t.Error("Expected not to find non-existent key")
	}
}

func TestDGAEvaluationContext_All_Empty(t *testing.T) {
	ctx := pipelinex.NewEvaluationContext()

	all := ctx.All()
	if len(all) != 0 {
		t.Errorf("Expected empty map, got %d items", len(all))
	}
}

func TestDGAEvaluationContext_All_WithParams(t *testing.T) {
	ctx := pipelinex.NewEvaluationContext().WithParams(map[string]any{
		"branch": "main",
		"commit": "abc123",
	})

	all := ctx.All()
	if len(all) != 2 {
		t.Errorf("Expected 2 items, got %d", len(all))
	}

	if all["branch"] != "main" {
		t.Errorf("Expected branch='main', got '%v'", all["branch"])
	}

	if all["commit"] != "abc123" {
		t.Errorf("Expected commit='abc123', got '%v'", all["commit"])
	}
}

func TestDGAEvaluationContext_All_WithNode(t *testing.T) {
	node := pipelinex.NewDGANode("test-node", "RUNNING")
	ctx := pipelinex.NewEvaluationContext().WithNode(node)

	all := ctx.All()

	if all["nodeId"] != "test-node" {
		t.Errorf("Expected nodeId='test-node', got '%v'", all["nodeId"])
	}

	if all["nodeStatus"] != "RUNNING" {
		t.Errorf("Expected nodeStatus='RUNNING', got '%v'", all["nodeStatus"])
	}
}

func TestDGAEvaluationContext_WithNode_Chaining(t *testing.T) {
	node := pipelinex.NewDGANode("node-1", "SUCCESS")

	// 测试链式调用
	ctx := pipelinex.NewEvaluationContext().
		WithParams(map[string]any{"key": "value"}).
		WithNode(node)

	all := ctx.All()

	// 应该保留参数
	if all["key"] != "value" {
		t.Errorf("Expected key='value', got '%v'", all["key"])
	}

	// 应该包含节点信息
	if all["nodeId"] != "node-1" {
		t.Errorf("Expected nodeId='node-1', got '%v'", all["nodeId"])
	}
}

func TestDGAEvaluationContext_WithNode_DoesNotModifyOriginal(t *testing.T) {
	ctx1 := pipelinex.NewEvaluationContext().WithParams(map[string]any{
		"key1": "value1",
	})

	node := pipelinex.NewDGANode("node-1", "RUNNING")
	ctx2 := ctx1.WithNode(node)

	// ctx1 不应该包含节点信息
	all1 := ctx1.All()
	if _, ok := all1["nodeId"]; ok {
		t.Error("Original context should not have nodeId")
	}

	// ctx2 应该包含节点信息
	all2 := ctx2.All()
	if all2["nodeId"] != "node-1" {
		t.Errorf("New context should have nodeId='node-1', got '%v'", all2["nodeId"])
	}

	// ctx2 应该保留原有参数
	if all2["key1"] != "value1" {
		t.Errorf("New context should retain key1='value1', got '%v'", all2["key1"])
	}
}

func TestDGAEvaluationContext_WithParams_DoesNotModifyOriginal(t *testing.T) {
	ctx1 := pipelinex.NewEvaluationContext().WithParams(map[string]any{
		"key1": "value1",
	})

	ctx2 := ctx1.WithParams(map[string]any{
		"key2": "value2",
	})

	// ctx1 不应该包含 key2
	all1 := ctx1.All()
	if _, ok := all1["key2"]; ok {
		t.Error("Original context should not have key2")
	}

	// ctx2 应该包含 key1 和 key2
	all2 := ctx2.All()
	if all2["key1"] != "value1" {
		t.Errorf("New context should have key1='value1', got '%v'", all2["key1"])
	}
	if all2["key2"] != "value2" {
		t.Errorf("New context should have key2='value2', got '%v'", all2["key2"])
	}
}

func TestDGAEvaluationContext_WithParams_Overwrite(t *testing.T) {
	ctx := pipelinex.NewEvaluationContext().
		WithParams(map[string]any{
			"key": "value1",
		}).
		WithParams(map[string]any{
			"key": "value2",
		})

	all := ctx.All()
	if all["key"] != "value2" {
		t.Errorf("Expected key='value2' (overwritten), got '%v'", all["key"])
	}
}

func TestDGAEvaluationContext_All_MultipleTypes(t *testing.T) {
	ctx := pipelinex.NewEvaluationContext().WithParams(map[string]any{
		"string":  "text",
		"int":     42,
		"bool":    true,
		"float":   3.14,
		"nil":     nil,
	})

	all := ctx.All()

	if all["string"] != "text" {
		t.Errorf("Expected string='text', got '%v'", all["string"])
	}

	if all["int"] != 42 {
		t.Errorf("Expected int=42, got '%v'", all["int"])
	}

	if all["bool"] != true {
		t.Errorf("Expected bool=true, got '%v'", all["bool"])
	}

	if all["float"] != 3.14 {
		t.Errorf("Expected float=3.14, got '%v'", all["float"])
	}

	if all["nil"] != nil {
		t.Errorf("Expected nil=nil, got '%v'", all["nil"])
	}
}

func TestDGAEvaluationContext_Chaining_Multiple(t *testing.T) {
	node := pipelinex.NewDGANode("node-x", "PENDING")

	ctx := pipelinex.NewEvaluationContext().
		WithParams(map[string]any{"a": "1"}).
		WithParams(map[string]any{"b": "2"}).
		WithNode(node).
		WithParams(map[string]any{"c": "3"})

	all := ctx.All()

	// 所有参数都应该存在
	if all["a"] != "1" {
		t.Errorf("Expected a='1', got '%v'", all["a"])
	}
	if all["b"] != "2" {
		t.Errorf("Expected b='2', got '%v'", all["b"])
	}
	if all["c"] != "3" {
		t.Errorf("Expected c='3', got '%v'", all["c"])
	}
	if all["nodeId"] != "node-x" {
		t.Errorf("Expected nodeId='node-x', got '%v'", all["nodeId"])
	}
	if all["nodeStatus"] != "PENDING" {
		t.Errorf("Expected nodeStatus='PENDING', got '%v'", all["nodeStatus"])
	}
}

func TestDGAEvaluationContext_EmptyKey(t *testing.T) {
	ctx := pipelinex.NewEvaluationContext().WithParams(map[string]any{
		"": "empty-key-value",
	})

	val, ok := ctx.Get("")
	if !ok {
		t.Error("Expected to find empty key")
	}
	if val != "empty-key-value" {
		t.Errorf("Expected empty-key-value, got '%v'", val)
	}
}
