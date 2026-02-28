package pipelinex

type Node interface {
	//ID 获取节点唯一id
	Id() string
	//PipelineId 获取节点所属的流水线id
	PipelineId() string
	//Status 获取节点状态
	Status() string
	//Get 获取节点属性数据
	Get(key string) string
	// Set 设置节点属性数据
	Set(key string, value any)
	// GetExecutor 获取节点执行器名称
	GetExecutor() string
	// GetSteps 获取节点执行步骤
	GetSteps() []Step
	// GetImage 获取节点镜像
	GetImage() string
	// GetConfig 获取节点配置
	GetConfig() map[string]any
}
