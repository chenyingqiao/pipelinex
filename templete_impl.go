package pipelinex

import (
	"fmt"
	"strings"

	"github.com/flosch/pongo2/v6"
)

// 预检查Pongo2TemplateEngine是否实现了TemplateEngine接口
var _ TemplateEngine = (*Pongo2TemplateEngine)(nil)

// Pongo2TemplateEngine 使用pongo2作为模板引擎的实现
type Pongo2TemplateEngine struct{}

// NewPongo2TemplateEngine 创建一个新的Pongo2模板引擎实例
func NewPongo2TemplateEngine() TemplateEngine {
	return &Pongo2TemplateEngine{}
}

// EvaluateBool 评估模板表达式，返回布尔值
func (e *Pongo2TemplateEngine) EvaluateBool(expression string, ctx map[string]any) (bool, error) {
	// 确保表达式使用pongo2模板语法
	template, err := pongo2.FromString(expression)
	if err != nil {
		return false, fmt.Errorf("failed to parse expression '%s': %w", expression, err)
	}

	// 执行模板
	result, err := template.Execute(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to execute expression '%s': %w", expression, err)
	}

	// 将结果转换为布尔值
	result = strings.TrimSpace(result)

	// 处理空值
	if result == "" {
		return false, nil
	}

	// 转换为小写进行比较
	lowerResult := strings.ToLower(result)

	switch lowerResult {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	default:
		// 对于其他值，非空字符串视为true
		return result != "", nil
	}
}

// EvaluateString 评估模板表达式，返回字符串
func (e *Pongo2TemplateEngine) EvaluateString(expression string, ctx map[string]any) (string, error) {
	template, err := pongo2.FromString(expression)
	if err != nil {
		return "", fmt.Errorf("failed to parse expression '%s': %w", expression, err)
	}

	result, err := template.Execute(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to execute expression '%s': %w", expression, err)
	}

	return strings.TrimSpace(result), nil
}

// Validate 验证表达式语法是否正确
func (e *Pongo2TemplateEngine) Validate(expression string) error {
	_, err := pongo2.FromString(expression)
	if err != nil {
		return fmt.Errorf("invalid expression syntax: %w", err)
	}
	return nil
}
