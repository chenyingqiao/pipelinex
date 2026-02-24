package test

import (
	"testing"

	"github.com/chenyingqiao/pipelinex"
)

func TestNewPongo2TemplateEngine(t *testing.T) {
	engine := pipelinex.NewPongo2TemplateEngine()
	if engine == nil {
		t.Error("Expected non-nil template engine")
	}
}

func TestPongo2TemplateEngine_EvaluateBool_True(t *testing.T) {
	engine := pipelinex.NewPongo2TemplateEngine()
	ctx := map[string]any{
		"status": "SUCCESS",
	}

	result, err := engine.EvaluateBool("{{ status == 'SUCCESS' }}", ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !result {
		t.Error("Expected true for matching status")
	}
}

func TestPongo2TemplateEngine_EvaluateBool_False(t *testing.T) {
	engine := pipelinex.NewPongo2TemplateEngine()
	ctx := map[string]any{
		"status": "FAILED",
	}

	result, err := engine.EvaluateBool("{{ status == 'SUCCESS' }}", ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result {
		t.Error("Expected false for non-matching status")
	}
}

func TestPongo2TemplateEngine_EvaluateBool_EmptyResult(t *testing.T) {
	engine := pipelinex.NewPongo2TemplateEngine()
	ctx := map[string]any{}

	// 当模板输出为空时，应该返回 false
	result, err := engine.EvaluateBool("{{ '' }}", ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result {
		t.Error("Expected false for empty result")
	}
}

func TestPongo2TemplateEngine_EvaluateBool_StringTrue(t *testing.T) {
	engine := pipelinex.NewPongo2TemplateEngine()
	ctx := map[string]any{}

	// 当模板输出 "true" 时，应该返回 true
	result, err := engine.EvaluateBool("{{ 'true' }}", ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !result {
		t.Error("Expected true for 'true' string result")
	}
}

func TestPongo2TemplateEngine_EvaluateBool_StringFalse(t *testing.T) {
	engine := pipelinex.NewPongo2TemplateEngine()
	ctx := map[string]any{}

	// 当模板输出 "false" 时，应该返回 false
	result, err := engine.EvaluateBool("{{ 'false' }}", ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result {
		t.Error("Expected false for 'false' string result")
	}

	// 测试不同大小写
	result2, err := engine.EvaluateBool("{{ 'FALSE' }}", ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result2 {
		t.Error("Expected false for 'FALSE' string result")
	}
}

func TestPongo2TemplateEngine_EvaluateBool_StringOne(t *testing.T) {
	engine := pipelinex.NewPongo2TemplateEngine()
	ctx := map[string]any{}

	result, err := engine.EvaluateBool("{{ '1' }}", ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !result {
		t.Error("Expected true for '1' string result")
	}
}

func TestPongo2TemplateEngine_EvaluateBool_StringZero(t *testing.T) {
	engine := pipelinex.NewPongo2TemplateEngine()
	ctx := map[string]any{}

	result, err := engine.EvaluateBool("{{ '0' }}", ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result {
		t.Error("Expected false for '0' string result")
	}
}

func TestPongo2TemplateEngine_EvaluateBool_NonEmptyString(t *testing.T) {
	engine := pipelinex.NewPongo2TemplateEngine()
	ctx := map[string]any{}

	// 非空字符串应该返回 true
	result, err := engine.EvaluateBool("{{ 'hello' }}", ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !result {
		t.Error("Expected true for non-empty string result")
	}
}

func TestPongo2TemplateEngine_EvaluateBool_InvalidSyntax(t *testing.T) {
	engine := pipelinex.NewPongo2TemplateEngine()
	ctx := map[string]any{}

	_, err := engine.EvaluateBool("{{ unclosed tag", ctx)
	if err == nil {
		t.Error("Expected error for invalid template syntax")
	}
}

func TestPongo2TemplateEngine_EvaluateBool_ComplexCondition(t *testing.T) {
	engine := pipelinex.NewPongo2TemplateEngine()
	ctx := map[string]any{
		"branch": "main",
		"status": "SUCCESS",
	}

	// 测试复杂条件
	result, err := engine.EvaluateBool("{{ branch == 'main' and status == 'SUCCESS' }}", ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !result {
		t.Error("Expected true for matching complex condition")
	}
}

func TestPongo2TemplateEngine_EvaluateString(t *testing.T) {
	engine := pipelinex.NewPongo2TemplateEngine()
	ctx := map[string]any{
		"name": "World",
	}

	result, err := engine.EvaluateString("{{ name }}", ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != "World" {
		t.Errorf("Expected 'World', got '%s'", result)
	}
}

func TestPongo2TemplateEngine_EvaluateString_WithSpaces(t *testing.T) {
	engine := pipelinex.NewPongo2TemplateEngine()
	ctx := map[string]any{
		"text": "  hello world  ",
	}

	result, err := engine.EvaluateString("{{ text }}", ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// 结果应该被 trim
	if result != "hello world" {
		t.Errorf("Expected trimmed 'hello world', got '%s'", result)
	}
}

func TestPongo2TemplateEngine_EvaluateString_InvalidSyntax(t *testing.T) {
	engine := pipelinex.NewPongo2TemplateEngine()
	ctx := map[string]any{}

	_, err := engine.EvaluateString("{{ unclosed tag", ctx)
	if err == nil {
		t.Error("Expected error for invalid template syntax")
	}
}

func TestPongo2TemplateEngine_Validate_Valid(t *testing.T) {
	engine := pipelinex.NewPongo2TemplateEngine()

	err := engine.Validate("{{ status == 'SUCCESS' }}")
	if err != nil {
		t.Errorf("Unexpected error for valid expression: %v", err)
	}
}

func TestPongo2TemplateEngine_Validate_Invalid(t *testing.T) {
	engine := pipelinex.NewPongo2TemplateEngine()

	err := engine.Validate("{{ unclosed tag")
	if err == nil {
		t.Error("Expected error for invalid expression syntax")
	}
}

func TestPongo2TemplateEngine_Validate_Empty(t *testing.T) {
	engine := pipelinex.NewPongo2TemplateEngine()

	// 空字符串应该是有效的
	err := engine.Validate("")
	if err != nil {
		t.Errorf("Unexpected error for empty expression: %v", err)
	}
}

func TestPongo2TemplateEngine_EvaluateBool_WithPipelineData(t *testing.T) {
	engine := pipelinex.NewPongo2TemplateEngine()

	// 创建模拟的 pipeline 数据
	ctx := map[string]any{
		"pipelineId":     "pipe-123",
		"pipelineStatus": "RUNNING",
	}

	result, err := engine.EvaluateBool("{{ pipelineStatus == 'RUNNING' }}", ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !result {
		t.Error("Expected true for matching pipeline status")
	}
}

func TestPongo2TemplateEngine_EvaluateBool_WithNodeData(t *testing.T) {
	engine := pipelinex.NewPongo2TemplateEngine()

	ctx := map[string]any{
		"nodeId":     "node-1",
		"nodeStatus": "SUCCESS",
	}

	result, err := engine.EvaluateBool("{{ nodeStatus == 'SUCCESS' }}", ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !result {
		t.Error("Expected true for matching node status")
	}
}
