package pipelinex

import "context"

type Graph interface {
	//Nodes
	Nodes() map[string]Node
	//AddVertex 添加顶点
	AddVertex(node Node)
	//AddEdge 添加边
	AddEdge(src, dest string)
	//CycelCheck
	CycelCheck() bool
	//BFS 广度有限搜索
	BFS() []string
}

// 流水线事件
type Event string

type Listener interface {
	// 处理对应的事件将事件发生的对应的流水线和对应的事件作为参数传入
	Handle(p Pipeline, event Event)
	// 获取当前注册的Event
	Events() []Event
}

// PipelineListeningFn 流水线监听函数
type ListeningFn func(p Pipeline)
type Metadata map[string]interface{}

type Pipeline interface {
	//ID 流水线的id
	ID() string
	//GetGraph 返回图结构
	GetGraph() Graph
	//SetGraph 设置图结构
	SetGraph(graph Graph)
	//Status 返回流水线的整体状态
	Status() string
	//Metadata 返回流水线执行的源数据
	Metadata() Metadata
	//Listening 流水线执行事件监听设置
	Listening(listener Listener)
	//Done流水线是否执行完成
	Done() <-chan struct{}
	//Run执行流水线
	Run(ctx context.Context) error
	//Notify 执行的步骤通知流水线
	Notify()
	//Cancel 取消流水线
	Cancel()
}
