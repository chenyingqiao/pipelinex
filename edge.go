package pipelinex

// Edge 表示DAG中的边，支持条件表达式
type Edge interface {
	// Source 返回边的源节点
	Source() Node
	// Target 返回边的目标节点
	Target() Node
	// Expression 返回边的条件表达式（pongo2模板语法）
	// 如果返回空字符串，表示无条件边，总是可以遍历
	Expression() string
	// Evaluate 评估条件表达式，返回bool表示是否通过
	Evaluate(ctx EvaluationContext) (bool, error)
	// ID 返回边的唯一标识符（格式：source->target）
	ID() string
}

// EvaluationContext 表达式求值上下文
type EvaluationContext interface {
	Get(key string) (any, bool)
	All() map[string]any
	WithNode(node Node) EvaluationContext
	WithPipeline(pipeline Pipeline) EvaluationContext
	WithParams(params map[string]any) EvaluationContext
}
