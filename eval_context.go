package pipelinex

// DGAEvaluationContext 是EvaluationContext接口的实现
type DGAEvaluationContext struct {
	data     map[string]any
	node     Node
	pipeline Pipeline
}

// NewEvaluationContext 创建一个新的求值上下文
func NewEvaluationContext() EvaluationContext {
	return &DGAEvaluationContext{
		data: make(map[string]any),
	}
}

// Get 从上下文中获取值
func (c *DGAEvaluationContext) Get(key string) (any, bool) {
	val, ok := c.data[key]
	return val, ok
}

// All 返回上下文中所有数据的副本
// 合并了：基础数据、节点数据、流水线数据
func (c *DGAEvaluationContext) All() map[string]any {
	result := make(map[string]any)

	// 复制基础数据
	for k, v := range c.data {
		result[k] = v
	}

	// 添加节点相关数据
	if c.node != nil {
		result["nodeId"] = c.node.Id()
		result["nodeStatus"] = c.node.Status()
	}

	// 添加流水线相关数据
	if c.pipeline != nil {
		result["pipelineId"] = c.pipeline.Id()
		result["pipelineStatus"] = c.pipeline.Status()
		if metadata := c.pipeline.Metadata(); metadata != nil {
			for k, v := range metadata {
				result[k] = v
			}
		}
	}

	return result
}

// WithNode 设置当前节点并返回新的上下文（链式调用）
func (c *DGAEvaluationContext) WithNode(node Node) EvaluationContext {
	newCtx := &DGAEvaluationContext{
		data:     make(map[string]any),
		node:     node,
		pipeline: c.pipeline,
	}
	for k, v := range c.data {
		newCtx.data[k] = v
	}
	return newCtx
}

// WithPipeline 设置流水线并返回新的上下文（链式调用）
func (c *DGAEvaluationContext) WithPipeline(pipeline Pipeline) EvaluationContext {
	newCtx := &DGAEvaluationContext{
		data:     make(map[string]any),
		node:     c.node,
		pipeline: pipeline,
	}
	for k, v := range c.data {
		newCtx.data[k] = v
	}
	return newCtx
}

// WithParams 添加参数到上下文并返回新的上下文（链式调用）
func (c *DGAEvaluationContext) WithParams(params map[string]any) EvaluationContext {
	newCtx := &DGAEvaluationContext{
		data:     make(map[string]any),
		node:     c.node,
		pipeline: c.pipeline,
	}
	for k, v := range c.data {
		newCtx.data[k] = v
	}
	for k, v := range params {
		newCtx.data[k] = v
	}
	return newCtx
}
