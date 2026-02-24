package pipelinex

// TemplateEngine 模板引擎接口，用于表达式求值
type TemplateEngine interface {
	// EvaluateBool 评估模板表达式，返回布尔值
	EvaluateBool(expression string, ctx map[string]any) (bool, error)
	// EvaluateString 评估模板表达式，返回字符串
	EvaluateString(expression string, ctx map[string]any) (string, error)
	// Validate 验证表达式语法是否正确
	Validate(expression string) error
}
