package pipelinex

type Node interface {
	//ID 获取节点唯一id
	ID() string
	//节点组id
	GroupID() string
	//Status 获取节点状态
	Status() string
	//Get 获取节点属性数据
	Get(key string) string
}
