package pipelinex

import "fmt"

// 预检查DGAEdge是否实现了Edge接口
var _ Edge = (*DGAEdge)(nil)

// DGAEdge 是Edge接口的实现
type DGAEdge struct {
	source     Node
	target     Node
	expression string
	engine     TemplateEngine
}

// NewDGAEdge 创建一条无条件边
func NewDGAEdge(source, target Node) Edge {
	return &DGAEdge{
		source:     source,
		target:     target,
		expression: "",
		engine:     nil,
	}
}

// NewConditionalEdge 创建一条条件边
func NewConditionalEdge(source, target Node, expression string) Edge {
	return &DGAEdge{
		source:     source,
		target:     target,
		expression: expression,
		engine:     nil,
	}
}

// NewConditionalEdgeWithEngine 创建一条带有模板引擎的条件边
func NewConditionalEdgeWithEngine(source, target Node, expression string, engine TemplateEngine) Edge {
	return &DGAEdge{
		source:     source,
		target:     target,
		expression: expression,
		engine:     engine,
	}
}

// Source 返回边的源节点
func (e *DGAEdge) Source() Node {
	return e.source
}

// Target 返回边的目标节点
func (e *DGAEdge) Target() Node {
	return e.target
}

// Expression 返回边的条件表达式
func (e *DGAEdge) Expression() string {
	return e.expression
}

// ID 返回边的唯一标识符
func (e *DGAEdge) ID() string {
	return fmt.Sprintf("%s->%s", e.source.Id(), e.target.Id())
}

// Evaluate 评估条件表达式
// 如果表达式为空，返回true（无条件边总是可以通过）
// 如果有表达式但没有设置模板引擎，使用默认的Pongo2模板引擎
func (e *DGAEdge) Evaluate(ctx EvaluationContext) (bool, error) {
	// 无条件边总是返回true
	if e.expression == "" {
		return true, nil
	}

	// 获取模板引擎
	engine := e.engine
	if engine == nil {
		engine = NewPongo2TemplateEngine()
	}

	// 评估表达式
	return engine.EvaluateBool(e.expression, ctx.All())
}

// SetEngine 设置模板引擎（用于延迟初始化）
func (e *DGAEdge) SetEngine(engine TemplateEngine) {
	e.engine = engine
}
